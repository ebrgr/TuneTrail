/* Verify media dialogs against a disposable fixture; the optional Koito context is read-only. */
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const origin = process.argv[2];
const out = path.resolve(process.argv[3]);
fs.mkdirSync(out, {recursive: true});
const checks = [];
const entries = [];
(async () => {
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    const context = await browser.newContext({viewport: {width: 1440, height: 1000}, locale: 'en-US'});
    const login = await context.request.post(origin + '/apis/web/v1/login', {data: {username: 'validation-admin', password: 'validation-password'}, headers: {Origin: origin}});
    assert.equal(login.status(), 204);
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    const go = async kind => {
      await page.goto(origin + '/alpha/' + kind + '/1');
      await page.locator('[data-media-task]').first().waitFor();
    };
    const open = async task => {
      await page.locator(`[data-media-task][href$="/${task}"]`).click();
      await page.locator('#media-task-dialog #task').waitFor();
      await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    };
    const close = async () => {
      await page.keyboard.press('Escape');
      await page.waitForFunction(() => !document.querySelector('#media-task-dialog').open);
    };
    // Capture the three page types at both reference viewport sizes before mutations.
    for (const [size, viewport] of Object.entries({desktop: {width: 1440, height: 1000}, mobile: {width: 390, height: 844}})) {
      await page.setViewportSize(viewport);
      for (const kind of ['track', 'album', 'artist']) {
        await go(kind);
        const actions = await page.locator('[data-media-task]').evaluateAll(links => links.map(link => ({task: link.pathname.split('/').pop(), width: link.getBoundingClientRect().width, height: link.getBoundingClientRect().height, icon: getComputedStyle(link.querySelector('svg')).width})));
        assert.deepEqual(actions.map(action => action.task), kind === 'track' ? ['add-listen','edit','merge','delete'] : ['edit','image','merge','delete']);
        assert(actions.every(action => action.width >= 44 && action.height >= 44 && action.icon === (size === 'desktop' ? '30px' : '22px')));
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
        const file = `tunetrail-${kind}-${size}.png`;
        await page.screenshot({path: path.join(out, file)});
        entries.push({file, kind, size, actions});
        for (const task of actions.map(action => action.task)) {
          await open(task);
          assert.equal(new URL(page.url()).pathname, '/alpha/' + kind + '/1');
          assert.equal(await page.locator('#media-task-dialog h1').count(), 0);
          for (let i = 0; i < 15; i++) {
            await page.keyboard.press('Tab');
            assert(await page.evaluate(() => document.querySelector('#media-task-dialog').contains(document.activeElement)));
          }
          await page.locator('#media-task-title').focus();
          await page.locator('#media-task-dialog').evaluate(dialog=>{dialog.scrollTop=0;});
          await page.screenshot({path: path.join(out, `dialog-${kind}-${task}-${size}.png`), animations:'disabled'});
          await close();
          assert.equal(await page.evaluate(() => document.activeElement.pathname.split('/').pop()), task);
        }
      }
    }
    checks.push('Three media types, both sizes: icon order, 30/22px glyphs, 44px targets, no overflow, isolated tasks, Tab trap, Escape and restored focus');
    await page.setViewportSize({width: 1440, height: 1000});
    await go('track');
    const publicURL = page.url();
    const [popup] = await Promise.all([
      context.waitForEvent('page'),
      page.locator('[data-media-task][href$="/edit"]').click({modifiers:[process.platform === 'darwin' ? 'Meta' : 'Control']})
    ]);
    await popup.waitForLoadState('domcontentloaded');
    assert.equal(new URL(popup.url()).pathname,'/alpha/manage/track/1/edit');
    assert.equal(await page.locator('#media-task-dialog').evaluate(dialog=>dialog.open),false);
    await popup.close();
    checks.push('Modified click follows the real task link in a separate tab');
    await open('edit');
    const heading = await page.locator('.media-heading h1').innerText();
    const alias = page.locator('#task form[action$="alias-add"]');
    await alias.locator('input[name=alias]').fill('Dialog verified alias');
    await alias.locator('button').click();
    await page.locator('#task li').filter({hasText: 'Dialog verified alias'}).waitFor();
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    await page.locator('#task li').filter({hasText: 'Dialog verified alias'}).locator('form[action$="alias-primary"] button').click();
    await page.locator('.media-heading h1').filter({hasText: 'Dialog verified alias'}).waitFor();
    assert.notEqual(await page.locator('.media-heading h1').innerText(), heading);
    const artistSearch = page.locator('#task form[method=get]');
    await artistSearch.locator('input[name=artist_query]').fill('Artista');
    await artistSearch.locator('button').click();
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    assert.equal(page.url(), publicURL);
    assert((await page.locator('.media-heading h1').innerText()).includes('Dialog verified alias'));
    const metadata = page.locator('#task form[action$="metadata"]');
    await metadata.locator('input[name=mbid]').fill('b24d0e8e-07dd-4028-8e65-7f2168ea9a22');
    await metadata.locator('button').click();
    await page.locator('.media-actions a[href*="musicbrainz.org"]').waitFor();
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    for (const [size, viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:390,height:844}})) {
      await page.setViewportSize(viewport);
      // The dialog obscures the toolbar; close it before capturing the public MBID action.
      if (await page.locator('#media-task-dialog').evaluate(dialog=>dialog.open)) await close();
      await page.screenshot({path:path.join(out,`tunetrail-track-with-mbid-${size}.png`)});
    }
    await page.setViewportSize({width:1440,height:1000});
    await open('add-listen');
    const previousCount = await page.locator('.media-heading').getAttribute('data-listen-count');
    const listen = page.locator('#task form[action$="listen-add"]');
    await listen.locator('input[name=local_time]').fill('2023-02-03T12:34');
    await listen.locator('input[name=timezone]').fill('UTC');
    await listen.locator('button').click();
    await page.waitForFunction(count=>document.querySelector('.media-heading').dataset.listenCount !== count,previousCount);
    assert.equal(page.url(),publicURL);
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    checks.push('Manual listen updates public heading count without navigation');
    checks.push('Alias mutation rotates ticket; primary alias refreshes public title; artist search preserves public URL and heading; metadata refreshes MusicBrainz link');
    await close();
    // Long names retain separate title/type and action geometry at intermediate desktop widths.
    for (const width of [768, 900, 960, 1024, 1440]) {
      await page.setViewportSize({width,height:1000});
      await go('track');
      const geometry = await page.evaluate(() => {
        const heading = document.querySelector('.media-heading h1');
        heading.textContent = 'A very long media title '.repeat(12);
        const range = document.createRange();
        range.selectNodeContents(document.querySelector('.media-type'));
        const type = range.getBoundingClientRect();
        const title = heading.getBoundingClientRect();
        const actions = document.querySelector('.media-actions').getBoundingClientRect();
        const overlap = rect => rect.left < actions.right && rect.right > actions.left && rect.top < actions.bottom && rect.bottom > actions.top;
        return {typeOverlaps:overlap(type),titleOverlaps:overlap(title)};
      });
      assert.deepEqual(geometry,{typeOverlaps:false,titleOverlaps:false},String(width));
    }
    checks.push('Long media titles and type labels do not overlap action targets at 768, 900, 960, 1024 and 1440px');
    await go('album');
    await open('image');
    const original = await page.locator('.media-cover').getAttribute('src');
    await page.evaluate(()=>{window.mediaDocumentIdentity='retained';});
    const artwork = page.locator('#task form[enctype="multipart/form-data"]');
    await artwork.locator('input[type=file]').setInputFiles(path.resolve('assets/preview/avatar.webp'));
    await artwork.locator('button').click();
    await page.waitForFunction(previous => document.querySelector('.media-cover').getAttribute('src') !== previous, original);
    assert.equal(new URL(page.url()).pathname, '/alpha/album/1');
    assert.equal(await page.evaluate(()=>window.mediaDocumentIdentity),'retained');
    assert(await page.locator('#media-task-dialog').evaluate(dialog => dialog.open));
    assert((await page.locator('.media-cover').getAttribute('srcset')).includes(await page.locator('.media-cover').getAttribute('src')));
    await page.waitForFunction(() => !document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));
    checks.push('Multipart image upload refreshes cover and srcset while dialog and public document remain open');
    await close();
    // Delayed responses cannot populate a later task after close and reopen.
    let release;
    const delayed = new Promise(resolve => { release = resolve; });
    await page.route('**/manage/album/1/edit?partial=1', async route => { await delayed; await route.continue().catch(() => {}); });
    await page.locator('[data-media-task][href$="/edit"]').click();
    await close();
    await open('image');
    release();
    await page.waitForTimeout(250);
    assert.equal(await page.locator('#task form[action$="alias-add"]').count(), 0);
    await page.unroute('**/manage/album/1/edit?partial=1');
    checks.push('Close/reopen aborts delayed edit response; new image dialog retains its own task');
    await close();
    await page.route('**/manage/album/1/edit?partial=1', route => route.abort('failed'));
    await page.locator('[data-media-task][href$="/edit"]').click();
    await page.locator('[data-task-content] a').waitFor();
    assert.equal(await page.locator('[data-task-content] form').count(), 0);
    await page.screenshot({path: path.join(out, 'network-error.png')});
    await page.locator('[data-task-content] a').click();
    await page.waitForURL('**/alpha/manage/album/1/edit');
    assert(await page.locator('main form[action$="alias-add"]').count());
    await page.unroute('**/manage/album/1/edit?partial=1');
    await go('album');
    await open('edit');
    await close();
    checks.push('Network failure fallback navigates to the complete task page; returning and reopening recovers without replay');
    // Simulate a suspended document, then verify real history navigation as well.
    await open('edit');
    await page.evaluate(() => dispatchEvent(new PageTransitionEvent('pagehide', {persisted:true})));
    assert.equal(await page.locator('#media-task-dialog form').count(), 0);
    await page.evaluate(() => dispatchEvent(new PageTransitionEvent('pageshow', {persisted:true})));
    await page.locator('[data-media-task]').first().waitFor({state:'visible'});
    await open('edit');
    await page.goto(origin+'/alpha');
    await page.goBack();
    await page.locator('.media-heading').waitFor();
    assert.equal(await page.locator('#media-task-dialog form').count(), 0);
    checks.push('pagehide clears active forms; pageshow revalidates; real Back does not restore an active dialog');
    // An expired session never inserts a login document or token form into a dialog.
    await context.clearCookies();
    await page.locator('[data-media-task][href$="/edit"]').click();
    await page.locator('[data-task-content] a').waitFor();
    assert.equal(await page.locator('#media-task-dialog form').count(), 0);
    await page.screenshot({path: path.join(out, 'expired-session.png')});
    await close();
    await page.evaluate(() => dispatchEvent(new PageTransitionEvent('pageshow', {persisted:true})));
    await page.waitForFunction(() => !document.body.hasAttribute('data-admin'));
    assert.equal(await page.locator('[data-media-task]').count(), 0);
    checks.push('Missing session cookie leaves no form; bfcache revalidation removes all administrator actions');
    const nojs = await browser.newContext({javaScriptEnabled:false, locale:'en-US'});
    await nojs.request.post(origin+'/apis/web/v1/login', {data:{username:'validation-admin',password:'validation-password'},headers:{Origin:origin}});
    const np = await nojs.newPage();
    await np.goto(origin+'/alpha/track/1');
    await np.locator('[data-media-task][href$="/edit"]').click();
    assert.equal(new URL(np.url()).pathname, '/alpha/manage/track/1/edit');
    assert(await np.locator('form[action$="alias-add"]').count());
    checks.push('No-JavaScript action link opens its complete task document');
    await nojs.close();
    assert.deepEqual(errors, []);
    await context.close();
    if (process.env.REFERENCE_USER && process.env.REFERENCE_PASSWORD) {
      const reference = await browser.newContext({locale:'en-US'});
      const result = await reference.request.post('http://127.0.0.1:4111/apis/web/v1/login', {data:{username:process.env.REFERENCE_USER,password:process.env.REFERENCE_PASSWORD},headers:{Origin:'http://127.0.0.1:4111'}});
      assert.equal(result.status(),204);
      const rp = await reference.newPage();
      for (const [size,viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:390,height:844}})) {
        await rp.setViewportSize(viewport);
        for (const kind of ['track','album','artist']) {
          await rp.goto('http://127.0.0.1:4111/'+kind+'/1');
          await rp.getByTitle('Edit Item',{exact:true}).waitFor();
          await rp.screenshot({path:path.join(out,`koito-${kind}-${size}.png`)});
          const actions = await rp.locator('[title]').evaluateAll(es => es.filter(e=>e.querySelector('svg')).map(e=>({title:e.title,rect:e.getBoundingClientRect().toJSON(),svg:e.querySelector('svg').getAttribute('width')})));
          entries.push({file:`koito-${kind}-${size}.png`,kind,size,actions});
        }
      }
      await reference.close();
    }
    fs.writeFileSync(path.join(out,'browser-checks.json'),JSON.stringify({checks,entries,jsErrors:errors},null,2)+'\n');
    console.log(JSON.stringify({checks, screenshots:entries.length,jsErrors:errors},null,2));
  } finally { await browser.close(); }
})().catch(error=>{console.error(error);process.exit(1)});
