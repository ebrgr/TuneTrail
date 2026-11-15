#!/usr/bin/env python3
"""ISSUE-124 evidence: media management without full reloads in a real browser (section
replaced in place, scroll and focus kept, chained actions with fresh tickets, heading
updated, listen notice announced, artist search without reload) and the no-JavaScript
anchor. Fictional disposable data only."""
import json
import os
import subprocess
import sys
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from validate_multiuser import Server, WEB, LBZ, scrobble  # noqa: E402

NODE = r'''
const {chromium} = require(process.env.PLAYWRIGHT_MODULE);
const origin = process.argv[2], shots = process.argv[3];
const login = page => page.evaluate(() => fetch('/apis/web/v1/login', {method: 'POST', headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({username: 'validation-admin', password: 'validation-password'})}).then(r => r.status));
(async () => {
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH ||
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const context = await browser.newContext({locale: 'en-US', timezoneId: 'UTC', viewport: {width: 1024, height: 700}});
  await context.addCookies([{name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin}]);
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const sizes = [];
  page.on('response', async r => { if (r.request().headers()['x-tunetrail-partial']) sizes.push({fragment: r.headers()['x-tunetrail-fragment'], bytes: (await r.body()).length, ms: Math.round(r.request().timing().responseEnd)}); });
  await page.goto(origin + '/cache-study');
  const out = {login: await login(page)};
  const href = await page.locator('a[href*="/track/"]').first().getAttribute('href');
  const manage = href.replace(/\/track\//, '/manage/track/').split('?')[0];
  await page.goto(origin + manage);
  const full = await page.evaluate(async () => (await (await fetch(location.href, {credentials: 'same-origin'})).text()).length);
  await page.evaluate(() => { window.__sameDocument = true; });
  // 1. Add an alias from the middle of the page: no reload, same scroll, focus back in the field.
  await page.locator('#aliases input[name=alias]:not([type=hidden])').scrollIntoViewIfNeeded();
  await page.evaluate(() => window.scrollBy(0, 120));
  const scrollBefore = await page.evaluate(() => scrollY);
  await page.locator('#aliases input[name=alias]:not([type=hidden])').fill('Alias One');
  await page.locator('#aliases input[name=alias]:not([type=hidden])').press('Enter');
  await page.waitForSelector('#aliases li:has-text("Alias One")');
  out.alias = {sameDocument: await page.evaluate(() => window.__sameDocument === true),
    scrollKept: Math.abs(await page.evaluate(() => scrollY) - scrollBefore) < 4,
    focus: await page.evaluate(() => document.activeElement?.name), url: new URL(page.url()).pathname + new URL(page.url()).hash};
  await page.screenshot({path: `${shots}/apelido-adicionado-1024.png`});
  // 2. Second action in the same section with the fresh ticket: make it primary, heading follows.
  await page.locator('#aliases li:has-text("Alias One") button').first().click();
  await page.waitForFunction(() => document.querySelector('h1').textContent.includes('Alias One'));
  out.primary = {heading: await page.locator('h1').textContent(), back: await page.locator('[data-media-back]').textContent(),
    sameDocument: await page.evaluate(() => window.__sameDocument === true)};
  // 3. Another section: add a listen, the notice is shown in the section and announced.
  await page.locator('#listens input[name=local_time]').fill('2023-11-14T19:13');
  await page.locator('#listens input[name=timezone]').fill('UTC');
  await page.locator('#listens form[action$="listen-add"] button').click();
  await page.waitForSelector('#listens [data-section-message]');
  await page.waitForTimeout(200);
  out.listen = {message: (await page.locator('#listens [data-section-message]').textContent()).trim(),
    status: (await page.locator('[data-enhance-status]').textContent()).trim(), sameDocument: await page.evaluate(() => window.__sameDocument === true)};
  // 4. Artist search without reload; the query stays in the address bar.
  await page.locator('#artists input[name=artist_query]').fill('Validation');
  await page.locator('#artists input[name=artist_query]').press('Enter');
  await page.waitForSelector('#artists button:has-text("Add Validation")');
  out.search = {sameDocument: await page.evaluate(() => window.__sameDocument === true), url: new URL(page.url()).search};
  // 5. Without JavaScript the same action returns to the section anchor.
  const nojs = await browser.newContext({javaScriptEnabled: false, locale: 'en-US', viewport: {width: 1024, height: 700}});
  await nojs.addCookies(await context.cookies());
  const plain = await nojs.newPage();
  await plain.goto(origin + manage);
  await plain.locator('#aliases input[name=alias]:not([type=hidden])').fill('Alias Two');
  await Promise.all([plain.waitForNavigation(), plain.locator('#aliases input[name=alias]:not([type=hidden])').press('Enter')]);
  out.noJs = new URL(plain.url()).hash;
  out.sizes = {fullPage: full, fragments: sizes};
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
'''


def main():
    binary, shots = Path(sys.argv[1]).resolve(strict=True), Path(sys.argv[2]).resolve()
    with tempfile.TemporaryDirectory(prefix='tunetrail-124-', dir='/tmp') as tmp:
        server = Server(binary, Path(tmp) / 'data')
        try:
            server.start()
            server.login()
            server.request('POST', WEB + '/profiles', {'display_name': 'Cache Study', 'slug': 'cache-study',
                                                      'avatar_icon': 'headphones'}, admin=True, expected=201)
            key = server.request('GET', WEB + '/profiles/cache-study/user/apikeys', admin=True)[0]['key']
            server.request('POST', LBZ + '/submit-listens', scrobble(int(time.time()) - 3600), token=key)
            script = Path(tmp) / 'check.cjs'
            script.write_text(NODE)
            result = subprocess.run(['node', str(script), server.origin, str(shots)], capture_output=True, text=True,
                                    timeout=240, env=os.environ)
            if result.returncode:
                print(result.stdout, result.stderr[:600], result.stderr[-1500:])
                return 1
            r = json.loads(result.stdout)
            print(json.dumps(r, indent=1))
            ok = (r['login'] == 204 and r['alias']['sameDocument'] and r['alias']['scrollKept'] and r['alias']['focus'] == 'alias'
                  and r['primary']['sameDocument'] and 'Alias One' in r['primary']['heading'] and 'Alias One' in r['primary']['back']
                  and r['listen']['message'] == 'Listen added.' and r['listen']['status'] == 'Listen added.'
                  and r['listen']['sameDocument'] and r['search']['sameDocument'] and 'artist_query=Validation' in r['search']['url']
                  and 'partial' not in r['search']['url'] and r['noJs'] == '#aliases')
            print('PASS' if ok else 'FAIL')
            return 0 if ok else 1
        finally:
            server.stop()


if __name__ == '__main__':
    sys.exit(main())
