#!/usr/bin/env python3
"""ISSUE-110 evidence: the theme panel opens by click, keyboard and "\\", closes with Esc
returning focus, applies a theme on public pages and in administration, and saves the
language. Fictional disposable data only."""
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
  const context = await browser.newContext({locale: 'pt-BR', timezoneId: 'UTC', viewport: {width: 1024, height: 900}});
  await context.addCookies([{name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin}]);
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const open = () => page.evaluate(() => document.getElementById('theme-panel').matches(':popover-open'));
  const out = {};
  const check = async (label, path) => {
    await page.goto(origin + path);
    const trigger = page.locator('button[popovertarget="theme-panel"]').first();
    out[label] = {closedAtLoad: !(await open())};
    await trigger.click();
    out[label].clickOpens = await open();
    await page.locator('[data-theme-choice="pearl"]').click();
    out[label].applied = await page.evaluate(() => document.documentElement.dataset.theme);
    out[label].pressed = await page.locator('[data-theme-choice="pearl"]').getAttribute('aria-pressed');
    await page.keyboard.press('Escape');
    out[label].escCloses = !(await open());
    out[label].focusBack = await page.evaluate(() => document.activeElement?.getAttribute('popovertarget') === 'theme-panel');
    await page.keyboard.press('Enter');
    out[label].enterOpens = await open();
    await page.keyboard.press('Escape');
    // Shortcut path: focus enters the panel and returns to the previously focused link.
    await page.locator('a.nav-button[href="/admin"]').focus();
    await page.keyboard.press('Backslash');
    out[label].backslashOpens = await open();
    out[label].backslashFocusInside = await page.evaluate(() => document.getElementById('theme-panel').contains(document.activeElement));
    await page.screenshot({path: `${shots}/painel-${label}-1024.png`});
    await page.keyboard.press('Escape');
    out[label].backslashFocusBack = await page.evaluate(() => document.activeElement?.matches('a.nav-button[href="/admin"]'));
    await trigger.click();
    await page.locator('[data-theme-reset]').click();
    await page.keyboard.press('Escape');
  };
  await check('publico', '/cache-study');
  out.login = await page.evaluate(() => fetch('/apis/web/v1/login', {method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({username: 'validation-admin', password: 'validation-password'})}).then(r => r.status));
  await check('admin', '/admin');
  out.adminBadge = await page.locator('a[href="/admin"].nav-button .admin-badge').count();
  // Language form inside the panel works through the normal POST.
  await page.goto(origin + '/cache-study');
  await page.locator('button[popovertarget="theme-panel"]').first().click();
  await page.selectOption('#theme-panel select[name="language"]', 'en');
  await Promise.all([page.waitForNavigation(), page.locator('#theme-panel .language-picker button').click()]);
  out.languageAfter = await page.evaluate(() => document.documentElement.lang);
  // Mobile layout screenshot with the panel open.
  await page.setViewportSize({width: 390, height: 844});
  await page.goto(origin + '/cache-study');
  await page.locator('button[popovertarget="theme-panel"]').first().click();
  await page.screenshot({path: `${shots}/painel-publico-390.png`});
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
'''


def main():
    binary, shots = Path(sys.argv[1]).resolve(strict=True), Path(sys.argv[2]).resolve()
    shots.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='tunetrail-110-', dir='/tmp') as tmp:
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
            result = subprocess.run(['node', str(script), server.origin, str(shots)], check=True, capture_output=True,
                                    text=True, timeout=180, env=os.environ)
            report = json.loads(result.stdout)
            print(json.dumps(report, indent=1))
            ok = all(r['closedAtLoad'] and r['clickOpens'] and r['applied'] == 'pearl' and r['pressed'] == 'true'
                     and r['escCloses'] and r['focusBack'] and r['enterOpens'] and r['backslashOpens']
                     and r['backslashFocusInside'] and r['backslashFocusBack']
                     for r in (report['publico'], report['admin']))
            ok = ok and report['adminBadge'] == 1 and report['languageAfter'] == 'en'
            print('PASS' if ok else 'FAIL')
            return 0 if ok else 1
        finally:
            server.stop()


if __name__ == '__main__':
    sys.exit(main())
