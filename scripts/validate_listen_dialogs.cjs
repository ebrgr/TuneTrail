/* Exercise listen confirmation with real Chrome and a fresh disposable roadmap fixture. */
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const {execFileSync} = require('node:child_process');
const assert = require('node:assert/strict');
const origin = process.argv[2];
const out = path.resolve(process.argv[3]);
const database = process.argv[4];
fs.mkdirSync(out, {recursive:true});
const checks = [];
(async () => {
  const browser = await chromium.launch({headless:true, executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    const context = await browser.newContext({viewport:{width:1440,height:1000}, locale:'pt-BR'});
    const login = async () => assert.equal((await context.request.post(origin+'/apis/web/v1/login', {data:{username:'validation-admin',password:'validation-password'},headers:{Origin:origin}})).status(),204);
    await login();
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors=[];
    page.on('pageerror', error=>errors.push(error.message));
    const go = async route => {
      await page.goto(origin+'/alpha'+route);
      await page.locator('[data-listen-task]').first().waitFor({state:'attached'});
    };
    const idle = () => page.waitForFunction(()=>!document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    // Native history disclosures open from the keyboard before a dialog action is selected.
    const choose = async () => {
      const menu = page.locator('.listen-actions').first();
      if (!await menu.evaluate(menu=>menu.open)) await menu.locator('summary').press('Enter');
      await page.locator('[data-listen-task]').first().click();
    };
    const open = async () => {
      await choose();
      await page.locator('#selected-listen').waitFor();
      await idle();
    };
    const close = async () => {
      await page.keyboard.press('Escape');
      await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').open);
    };
    // Every history surface uses the same page fragment, keyboard contract and no-script fallback.
    for (const [size,viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:320,height:800}})) {
      await page.setViewportSize(viewport);
      for (const theme of ['yuu','pearl']) {
        await page.addInitScript(theme=>localStorage.setItem('theme',theme),theme);
        for (const route of ['', '/listens?period=all_time','/track/1','/album/1','/artist/1']) {
          await go(route);
          assert.equal(await page.locator('.listen-manage:visible').count(),0);
          const opener=page.locator('[data-listen-task]').first();
          const target=await opener.getAttribute('href');
          const doc=await page.evaluate(()=>window.__listenDocument=crypto.randomUUID());
          await open();
          assert.equal(await page.locator('dialog[open]').count(),1);
          assert((await page.locator('#selected-listen').innerText()).includes('Não existe desfazer'));
          assert.equal(await page.locator('.listen-details a').count()>=3,true);
          assert.equal(await page.locator('.listen-details time').count(),1);
          assert((await page.locator('.listen-details').innerText()).includes('America/Sao_Paulo'));
          for(let i=0;i<12;i++) {
            await page.keyboard.press('Tab');
            assert(await page.evaluate(()=>document.querySelector('#media-task-dialog').contains(document.activeElement)));
          }
          assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
          const name=route.split('?')[0].replaceAll('/','-') || '-home';
          await page.locator('#media-task-title').focus();
          await page.screenshot({animations:'disabled',path:path.join(out,`dialog${name}-${theme}-${size}.png`)});
          await close();
          assert.equal(await page.evaluate(()=>document.activeElement.href),new URL(target,origin).href);
          assert.equal(await page.evaluate(()=>window.__listenDocument),doc);
        }
      }
    }
    checks.push('Home, listens, track, album and artist × desktop/mobile × Yuu/Pearl: details, no separate Manage, native dialog, Tab, Esc, restored focus and no overflow');
    // A read failure and a delayed response never leak stale forms into a reopened task.
    await page.setViewportSize({width:1440,height:1000});
    await go('/listens?period=all_time');
    await page.route('**/manage/track/*/listen?**',route=>route.abort());
    await choose();
    await page.locator('[data-task-page]').waitFor();
    await page.screenshot({animations:'disabled',path:path.join(out,'erro-rede.png')});
    await close();
    await page.unroute('**/manage/track/*/listen?**');
    await open(); await close();
    let release;
    const delayed = new Promise(resolve=>{release=resolve;});
    await page.route('**/manage/track/*/listen?**',async route=>{const response=await route.fetch();await delayed;try { await route.fulfill({response}); } catch { /* Closing aborts the route in Chrome. */ }});
    await choose();
    await page.locator('#media-task-dialog[open]').waitFor();
    await close();
    release();
    await page.unroute('**/manage/track/*/listen?**');
    await open(); await close();
    checks.push('Network failure offers a normal page fallback; reopening recovers; closed requests cannot restore stale content');
    // A one-listen media item remains an empty personal snapshot after its final deletion.
    await go('/track/4');
    assert.equal(await page.locator('tr[data-unix]').count(),1);
    await open();
    await page.locator('#selected-listen input[name=confirm]').check();
    await page.locator('#selected-listen button').click();
    await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').open);
    assert.equal(await page.locator('tr[data-unix]').count(),0);
    assert.equal(await page.locator('.media-heading').getAttribute('data-listen-count'),'0');
    assert.equal(await page.locator('.media-history').getByText('Nada para mostrar nesta página.').count(),1);
    await page.screenshot({animations:'disabled',path:path.join(out,'historico-vazio.png')});
    checks.push('Final media listen yields count zero and empty history without navigation; ordinary public route remains unavailable');
    // Confirmed deletion uses the one-use POST, retains the document and preserves the other profile.
    await go('/listens?period=all_time');
    const beforeBeta=await (await context.request.get(origin+'/apis/web/v1/profiles/beta/stats?period=all_time')).json();
    const before=await page.locator('tr[data-unix]').count();
    const doc=await page.evaluate(()=>window.__listenDocument=crypto.randomUUID());
    await open();
    const selected=await page.locator('#selected-listen input[name=unix]').inputValue();
    const form=page.locator('#selected-listen form');
    const action=await form.getAttribute('action');
    const payload=await form.evaluate(form=>new URLSearchParams(new FormData(form)).toString());
    assert.equal((await context.request.post(origin+action,{data:payload,headers:{Origin:origin,'Content-Type':'application/x-www-form-urlencoded','X-TuneTrail-Partial':'1','X-TuneTrail-Dialog':'1'}})).status(),400);
    // A spent ticket cannot delete; reopen to obtain a fresh one.
    await close(); await open();
    await page.locator('#selected-listen input[name=confirm]').check();
    await page.locator('#selected-listen button').click();
    await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').open);
    assert.equal(await page.locator('tr[data-unix]').count(),before-1);
    assert.equal(await page.locator(`tr[data-unix="${selected}"]`).count(),0);
    assert.equal(await page.evaluate(()=>window.__listenDocument),doc);
    assert.equal(await page.evaluate(()=>document.activeElement.matches('.listen-actions summary')),true);
    const beta=await (await context.request.get(origin+'/apis/web/v1/profiles/beta/stats?period=all_time')).json();
    assert.equal(beta.listen_count,beforeBeta.listen_count);
    checks.push('No confirmation returns 400; reopening rotates authorization; confirmed POST removes one row without reload and leaves beta intact');
    // Delete through the home snapshot to prove all-time statistics update too.
    await go('');
    const stats=Number(await page.locator('.stats-columns').getAttribute('data-listen-count'));
    await open();
    const scroll=await page.evaluate(()=>scrollY);
    await page.locator('#selected-listen input[name=confirm]').check();
    await page.locator('#selected-listen button').click();
    await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').open);
    assert.equal(Number(await page.locator('.stats-columns').getAttribute('data-listen-count')),stats-1);
    assert(Math.abs(await page.evaluate(()=>scrollY)-scroll)<3);
    await page.locator('main img[data-cover]').first().evaluate(image=>{image.classList.remove('cover-failed');image.removeAttribute('srcset');image.src='/fictional-missing-cover-141.png';});
    await page.waitForFunction(()=>{const image=document.querySelector('main img[data-cover]');return image.classList.contains('cover-failed') && image.complete && image.naturalWidth>0;});
    await page.locator('.period-selector a[href*=all_time]').click();
    assert.equal(await page.evaluate(()=>localStorage.getItem('period_selection_1:home')),'all_time');
    checks.push('Home statistics, scroll and focus update after deletion; refreshed period links retain preferences and newly failed covers use the local fallback');
    // JavaScript-disabled navigation uses the exact task and returns to the source media context.
    const noJS=await browser.newContext({javaScriptEnabled:false,storageState:await context.storageState()});
    const plain=await noJS.newPage();
    await plain.goto(origin+'/alpha/track/1');
    assert.equal(await plain.locator('[data-listen-task]').count(),0);
    await plain.locator('.listen-actions summary').first().press('Enter');
    await plain.locator('.listen-manage').first().click();
    assert(new URL(plain.url()).pathname.endsWith('/listen'));
    assert((await plain.locator('input[name=return_to]').last().inputValue()).startsWith('/alpha/track/1'));
    await plain.locator('#selected-listen input[name=confirm]').check();
    await plain.locator('#selected-listen button').click();
    assert.equal(new URL(plain.url()).pathname,'/alpha/track/1');
    await noJS.close();
    checks.push('No JavaScript retains Manage, uses the shared confirmation page and returns to the originating media page');
    // Expire actual fixture sessions without touching or exporting their secrets.
    await go('/listens?period=all_time');
    if (!database || !path.resolve(database).startsWith('/tmp/tunetrail-141-')) throw Error('An explicit disposable fixture database is required');
    execFileSync('python3',['-c',"import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute('UPDATE sessions SET expires_at=1'); c.commit()",database]);
    await choose();
    await page.locator('[data-task-page]').waitFor();
    assert.equal(await page.locator('#media-task-dialog input[name=submission_token]').count(),0);
    await page.screenshot({animations:'disabled',path:path.join(out,'sessao-expirada.png')});
    await page.locator('[data-task-page]').click();
    await page.waitForURL('**/admin?next=**');
    checks.push('Real server-side expiry rejects the dialog read and fallback reaches sign-in without inserting a login document');
    assert.deepEqual(errors,[]);
    await context.close();
    const visitor=await browser.newContext();const guest=await visitor.newPage();
    await guest.goto(origin+'/alpha/listens?period=all_time');
    assert.equal(await guest.locator('.listen-manage,[data-listen-task],.delete-listen').count(),0);
    await visitor.close();
    checks.push('Visitors receive neither Manage nor ×');
    fs.writeFileSync(path.join(out,'browser-checks.json'),JSON.stringify({browser:browser.version(),fixtureOnly:true,checks},null,2)+'\n');
    console.log(checks.join('\n'));
  } finally { await browser.close(); }
})().catch(error=>{console.error(error);process.exitCode=1;});
