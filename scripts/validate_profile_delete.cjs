/* Exercise personal deletion in real Chrome against a new serve_roadmap06 fixture only. */
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const {execFileSync} = require('node:child_process');
const origin = process.argv[2];
const out = path.resolve(process.argv[3]);
const fixtureDB = path.resolve(process.argv[4]);
assert.equal(path.basename(fixtureDB), 'tunetrail.db');
// Refuse any database that is not inside a generated disposable fixture.
assert.equal(JSON.parse(fs.readFileSync(path.join(path.dirname(fixtureDB), '..', 'fixture-report.json'))).fixture_version, 1);
const checks = [];
fs.mkdirSync(out, {recursive: true});
(async () => {
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    const context = await browser.newContext({locale: 'pt-BR', viewport: {width: 1440, height: 1000}});
    const login = async () => {
      const result = await context.request.post(origin + '/apis/web/v1/login', {data: {username: 'validation-admin', password: 'validation-password'}, headers: {Origin: origin}});
      assert.equal(result.status(), 204);
    };
    await login();
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    const open = async (kind, profile = 'alpha', id = 1) => {
      await page.goto(`${origin}/${profile}/${kind}/${id}`);
      await page.locator('[data-media-task][href$="/delete"]').click();
      await page.locator('#media-task-dialog [data-delete-count]').waitFor();
      await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    };
    const count = async () => Number(await page.locator('[data-delete-count]').getAttribute('data-delete-count'));
    const close = async () => {
      await page.keyboard.press('Escape');
      await page.waitForFunction(() => !document.querySelector('#media-task-dialog').open);
      assert(await page.locator('[data-media-task][href$="/delete"]').evaluate(link => document.activeElement === link));
    };
    // These snapshots precede all mutations and cover both palettes and viewport sizes.
    for (const [size, viewport] of Object.entries({desktop: {width: 1440, height: 1000}, mobile: {width: 390, height: 844}})) {
      await page.setViewportSize(viewport);
      for (const theme of ['yuu', 'pearl']) {
        for (const kind of ['track', 'album', 'artist']) {
          await open(kind);
          await page.evaluate(theme => window.TuneTrailTheme.apply(theme), theme);
          assert.equal(await count(), kind === 'artist' ? 1 : kind === 'album' ? 7 : 6);
          assert.match(await page.locator('.profile-delete').innerText(), /Não existe desfazer/);
          if (kind === 'artist') assert.match(await page.locator('.profile-delete').innerText(), /colaboração/);
          for (let i = 0; i < 8; i++) {
            await page.keyboard.press('Tab');
            assert(await page.locator('#media-task-dialog').evaluate(dialog => dialog.contains(document.activeElement)));
          }
          assert(await page.locator('#media-task-dialog').evaluate(dialog => dialog.getBoundingClientRect().width <= 460));
          assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
          await page.locator('#media-task-title').focus();
          await page.screenshot({path: path.join(out, `${kind}-${theme}-${size}.png`)});
          await close();
        }
      }
    }
    checks.push('12 capturas: três tipos × Yuu/Pearl × desktop/celular; contagens, foco preso, Esc e foco devolvido');
    await open('track');
    await page.locator('[data-task-cancel]').click();
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').open);
    assert.equal(new URL(page.url()).pathname, '/alpha/track/1');
    checks.push('Cancelar fecha o diálogo sem navegar nem enviar mutação');
    await open('track');
    // A failed network send must never cause an automatic second POST.
    let posts = 0;
    await page.route('**/manage/track/1/confirm', async route => { posts++; await route.abort('failed'); });
    await page.locator('.profile-delete button[type=submit]').click();
    await page.locator('[data-task-page]').waitFor();
    assert.equal(posts, 1);
    await page.screenshot({path: path.join(out, 'falha-rede.png')});
    await page.unroute('**/manage/track/1/confirm');
    await close();
    await open('track');
    assert.equal(await count(), 6);
    checks.push('Falha de rede mostra recuperação e não repete o POST; reabrir consulta o estado e emite novos tokens');
    // An expired/removed session cannot confirm a form that was already visible.
    execFileSync('python3', ['-c', 'import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute("UPDATE sessions SET expires_at=1"); c.commit(); c.close()', fixtureDB]);
    await page.locator('.profile-delete button[type=submit]').click();
    await page.locator('[data-task-page]').waitFor();
    await page.screenshot({path: path.join(out, 'sessao-expirada.png')});
    await login();
    await close();
    await open('track');
    assert.equal(await count(), 6);
    checks.push('Sessão expirada no banco entre abrir e confirmar não altera escutas e mostra erro');
    // Artist deletion preserves the six collaboration listens on track 1.
    for (const [kind, profile, expected] of [['artist', 'alpha', 1], ['track', 'alpha', 6], ['album', 'beta', 1]]) {
      await open(kind, profile);
      assert.equal(await count(), expected);
      const responsePromise = page.waitForResponse(response => response.url().endsWith('/confirm') && response.request().method() === 'POST');
      await page.locator('.profile-delete button[type=submit]').click();
      const response = await responsePromise;
      assert.equal(response.status(), 204);
      assert.equal(response.headers()['x-tunetrail-deleted-listens'], String(expected));
      await page.waitForURL(origin + '/' + profile);
      const stats = await context.request.get(`${origin}/apis/web/v1/profiles/${profile}/stats?period=all_time`);
      assert.equal((await stats.json()).listen_count, kind === 'artist' ? 6 : profile === 'alpha' ? 0 : 1);
      checks.push(`${kind}: clique real remove ${expected} escutas, resposta concorda com contagem e navega para /${profile}`);
      if (kind === 'artist') {
        await open('artist');
        assert.equal(await count(), 0);
        let uncertainPosts = 0;
        await page.route('**/manage/artist/1/confirm', async route => {
          uncertainPosts++;
          const committed = await route.fetch();
          assert.equal(committed.status(), 204);
          await route.abort('failed');
        });
        await page.locator('.profile-delete button[type=submit]').click();
        await page.locator('[data-task-page]').waitFor();
        assert.equal(uncertainPosts, 1);
        await page.unroute('**/manage/artist/1/confirm');
        checks.push('Resposta perdida após confirmação efetivamente processada mostra erro sem reenviar automaticamente');
        await close();
      }
    }
    // The shared catalog still serves the task even with no listens left in this profile.
    const empty = await context.request.get(origin + '/alpha/manage/track/1/delete');
    assert.equal(empty.status(), 200);
    assert.match(await empty.text(), /data-delete-count="0"/);
    const beta = await context.request.get(origin + '/apis/web/v1/profiles/beta/stats?period=all_time');
    assert.equal((await beta.json()).listen_count, 1);
    assert.deepEqual(errors, []);
    await context.close();
    const plain = await browser.newContext({javaScriptEnabled: false, locale: 'en-US'});
    const auth = await plain.request.post(origin + '/apis/web/v1/login', {data: {username: 'validation-admin', password: 'validation-password'}, headers: {Origin: origin}});
    assert.equal(auth.status(), 204);
    const fallback = await plain.newPage();
    await fallback.goto(origin + '/alpha/manage/track/1/delete');
    await fallback.locator('.profile-delete button[type=submit]').click();
    await fallback.waitForURL(origin + '/alpha');
    checks.push('Sem JavaScript: confirmação com zero escutas funciona e retorna ao perfil; catálogo e beta permanecem');
    await plain.close();
    fs.writeFileSync(path.join(out, 'browser-checks.json'), JSON.stringify({checks, errors}, null, 2) + '\n');
    console.log(JSON.stringify({checks, errors}, null, 2));
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
