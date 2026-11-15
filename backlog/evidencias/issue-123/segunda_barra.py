#!/usr/bin/env python3
"""ISSUE-123 evidence: the second sidebar opens and closes by click and space, Esc closes it
and returns focus, "push" moves the content while "overlay" does not, it opens on
administration pages, and it sits below the top bar on phones. Fictional data only."""
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
(async () => {
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH ||
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const context = await browser.newContext({locale: 'pt-BR', timezoneId: 'UTC', viewport: {width: 1024, height: 700}});
  await context.addCookies([{name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin}]);
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const bar = () => page.evaluate(() => !document.getElementById('nav-admin').hidden);
  const contentLeft = sel => page.evaluate(s => Math.round(document.querySelector(s).getBoundingClientRect().left), sel);
  const out = {};
  await page.goto(origin + '/cache-study');
  const toggle = page.locator('[data-nav-toggle]');
  out.visitor = {closed: !(await bar()), role: await toggle.getAttribute('role'), expanded: await toggle.getAttribute('aria-expanded'),
    leftClosed: await contentLeft('.app-content')};
  await toggle.click();
  out.visitor.clickOpens = await bar();
  out.visitor.expandedOpen = await toggle.getAttribute('aria-expanded');
  out.visitor.leftPush = await contentLeft('.app-content');
  out.visitor.items = await page.locator('#nav-admin a').evaluateAll(as => as.map(a => a.getAttribute('href')));
  await page.screenshot({path: `${shots}/barra-visitante-empurrar-1024.png`});
  await page.locator('#nav-admin a').first().focus();
  await page.keyboard.press('Escape');
  out.visitor.escCloses = !(await bar());
  out.visitor.focusBack = await page.evaluate(() => document.activeElement?.hasAttribute('data-nav-toggle'));
  await page.keyboard.press(' ');
  out.visitor.spaceOpens = await bar();
  // Overlay preference from the theme panel.
  await page.locator('.sidebar button[popovertarget="theme-panel"]').click();
  await page.screenshot({path: `${shots}/painel-tema-secoes-1024.png`});
  await page.locator('input[name="nav-layout"][value="overlay"]').check();
  await page.keyboard.press('Escape');
  out.visitor.leftOverlay = await contentLeft('.app-content');
  await page.screenshot({path: `${shots}/barra-visitante-sobrepor-1024.png`});
  await page.reload();
  out.visitor.overlayRemembered = await page.evaluate(() => document.documentElement.dataset.navLayout === 'overlay');
  await page.evaluate(() => { localStorage.setItem('nav-layout', 'push'); delete document.documentElement.dataset.navLayout; });
  // Administrator: administration pages open the second sidebar.
  out.login = await page.evaluate(() => fetch('/apis/web/v1/login', {method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({username: 'validation-admin', password: 'validation-password'})}).then(r => r.status));
  // Overlay in administration: starts closed, opens on demand and closes after an item click.
  await page.evaluate(() => localStorage.setItem('nav-layout', 'overlay'));
  await page.goto(origin + '/admin/settings');
  out.overlayAdmin = {closedAtLoad: !(await bar()), left: await contentLeft('.admin-shell')};
  await page.locator('[data-nav-toggle]').click();
  out.overlayAdmin.opens = await bar();
  await page.screenshot({path: `${shots}/barra-admin-sobrepor-1024.png`});
  const [popup] = await Promise.all([page.waitForEvent('popup'), page.locator('#nav-admin a[target="_blank"]').click()]);
  await popup.close();
  out.overlayAdmin.closesAfterClick = !(await bar());
  await page.evaluate(() => localStorage.setItem('nav-layout', 'push'));
  await page.goto(origin + '/admin/settings');
  out.admin = {open: await bar(), expanded: await toggle.getAttribute('aria-expanded'),
    current: await page.locator('#nav-admin [aria-current="page"]').getAttribute('href'), leftPush: await contentLeft('.admin-shell')};
  out.admin.items = await page.locator('#nav-admin a').evaluateAll(as => as.map(a => a.getAttribute('href')));
  await page.screenshot({path: `${shots}/barra-admin-1024.png`});
  // Phone: below the top bar, in the page flow.
  await page.setViewportSize({width: 390, height: 844});
  await page.goto(origin + '/cache-study');
  await page.locator('[data-nav-toggle]').click();
  out.phone = await page.evaluate(() => {
    const first = document.querySelector('.sidebar:not(.sidebar-secondary)').getBoundingClientRect();
    const second = document.getElementById('nav-admin').getBoundingClientRect();
    return {below: Math.round(second.top) >= Math.round(first.bottom) - 1, fullWidth: Math.round(second.width) >= 380};
  });
  await page.screenshot({path: `${shots}/barra-admin-390.png`});
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
'''


def main():
    binary, shots = Path(sys.argv[1]).resolve(strict=True), Path(sys.argv[2]).resolve()
    shots.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='tunetrail-123-', dir='/tmp') as tmp:
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
            result = subprocess.run(['node', str(script), server.origin, str(shots)], capture_output=True,
                                    text=True, timeout=240, env=os.environ)
            if result.returncode:
                print(result.stdout, result.stderr[-3000:])
                return 1
            r = json.loads(result.stdout)
            print(json.dumps(r, indent=1))
            v, a = r['visitor'], r['admin']
            ok = (v['closed'] and v['role'] == 'button' and v['expanded'] == 'false' and v['clickOpens']
                  and v['expandedOpen'] == 'true' and v['leftPush'] - v['leftClosed'] == 58
                  and v['items'] == ['/admin', 'https://github.com/ebrgr/TuneTrail']
                  and v['escCloses'] and v['focusBack'] and v['spaceOpens']
                  and v['leftOverlay'] == v['leftClosed'] and v['overlayRemembered']
                  and a['open'] and a['expanded'] == 'true' and a['current'] == '/admin/settings'
                  and a['items'][:3] == ['/admin?section=account', '/admin?section=profiles', '/admin/settings']
                  and r['phone']['below'] and r['phone']['fullWidth']
                  and r['overlayAdmin']['closedAtLoad'] and r['overlayAdmin']['opens']
                  and r['overlayAdmin']['closesAfterClick'] and r['overlayAdmin']['left'] == 58)
            print('PASS' if ok else 'FAIL')
            return 0 if ok else 1
        finally:
            server.stop()


if __name__ == '__main__':
    sys.exit(main())
