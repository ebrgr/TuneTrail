#!/usr/bin/env python3
"""ISSUE-109 evidence: with the back/forward cache disabled, a history navigation keeps the
administrator badge without reloading while the session lives, and never shows the badge or
deletion controls after logout. Fictional data only."""
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
const origin = process.argv[2];
(async () => {
  // Playwright's default launch keeps --disable-back-forward-cache: no bfcache here.
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH ||
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page = await (await browser.newContext({locale: 'en-US', timezoneId: 'UTC'})).newPage();
  await page.context().addCookies([{name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin}]);
  const state = async () => page.evaluate(() => ({
    nav: performance.getEntriesByType('navigation')[0].type,
    persisted: window.__persisted ?? null,
    badge: document.querySelectorAll('.admin-badge').length,
    dataAdmin: 'admin' in document.body.dataset,
    controls: document.querySelectorAll('[data-listen-controls]').length,
  }));
  await page.addInitScript(() => addEventListener('pageshow', e => { window.__persisted = e.persisted; }));
  const post = path => page.evaluate(p => fetch(p, {method: 'POST', headers: {'Content-Type': 'application/json'},
    body: p.endsWith('login') ? JSON.stringify({username: 'validation-admin', password: 'validation-password'}) : undefined}).then(r => r.status), path);
  const out = {};
  await page.goto(origin + '/cache-study/listens?period=all_time');
  out.visitor = await state();
  out.login = await post('/apis/web/v1/login');
  await page.goto(origin + '/cache-study/listens?period=all_time&page=1');
  await page.waitForTimeout(300);
  out.admin = await state();
  await page.goto(origin + '/cache-study/search?q=x');
  await page.goBack();
  await page.waitForLoadState('load');
  await page.waitForTimeout(1500);
  out.adminBack = await state();
  await page.goto(origin + '/cache-study/search?q=x');
  out.logout = await post('/apis/web/v1/logout');
  await page.goBack();
  await page.waitForLoadState('load');
  await page.waitForTimeout(1500);
  out.afterLogoutBack = await state();
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
'''


def main():
    binary = Path(sys.argv[1]).resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix='tunetrail-109-', dir='/tmp') as tmp:
        server = Server(binary, Path(tmp) / 'data')
        try:
            server.start()
            server.login()
            server.request('POST', WEB + '/profiles', {'display_name': 'Cache Study', 'slug': 'cache-study',
                                                      'avatar_icon': 'headphones'}, admin=True, expected=201)
            key = server.request('GET', WEB + '/profiles/cache-study/user/apikeys', admin=True)[0]['key']
            for offset in range(3):
                server.request('POST', LBZ + '/submit-listens', scrobble(int(time.time()) - 3600 - offset), token=key)
            script = Path(tmp) / 'check.cjs'
            script.write_text(NODE)
            result = subprocess.run(['node', str(script), server.origin], check=True, capture_output=True, text=True,
                                    timeout=120, env=os.environ)
            report = json.loads(result.stdout)
            print(json.dumps(report, indent=1))
            after = report['afterLogoutBack']
            back = report['adminBack']
            ok = (report['admin']['badge'] == 1 and report['admin']['controls'] > 0 and
                  back['nav'] == 'back_forward' and back['badge'] == 1 and back['controls'] > 0 and
                  after['badge'] == 0 and not after['dataAdmin'] and after['controls'] == 0)
            print('PASS' if ok else 'FAIL')
            return 0 if ok else 1
        finally:
            server.stop()


if __name__ == '__main__':
    sys.exit(main())
