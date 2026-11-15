#!/usr/bin/env python3
"""ISSUE-107 evidence: instant search in a real browser (results while typing, stale
responses discarded, keyboard combobox, Escape, status announcement) and timings of the
full search page against the fragment. Fictional disposable data only."""
import json
import os
import subprocess
import sys
import tempfile
import time
import urllib.request
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
  const context = await browser.newContext({locale: 'pt-BR', timezoneId: 'UTC', viewport: {width: 1024, height: 800}});
  await context.addCookies([{name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin}]);
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  // Delay the first query on purpose: its late answer must not replace the newer list.
  let delayed = false;
  await page.route('**/search?*partial=1*', async route => {
    if (!delayed && route.request().url().includes('q=Ca&')) { delayed = true; await new Promise(r => setTimeout(r, 900)); }
    await route.continue();
  });
  const out = {};
  await page.goto(origin + '/cache-study');
  out.focusOnLoad = await page.evaluate(() => document.activeElement.tagName);
  await page.keyboard.press('/');
  await page.waitForTimeout(200);
  out.focusOnOpen = await page.evaluate(() => document.activeElement.id);
  const input = page.locator('#search-input');
  out.dialogOpen = await page.locator('#search-dialog').evaluate(d => d.open);
  const y = async () => Math.round((await input.boundingBox()).y);
  await page.waitForTimeout(300); // let the dialog opening animation finish
  out.inputY = [await y()];
  out.submitHidden = await page.locator('[data-search-submit]').isHidden();
  await input.pressSequentially('Ca', {delay: 20});
  await page.waitForTimeout(250);
  await input.pressSequentially('che', {delay: 20});
  await page.waitForSelector('#search-results [role=option]');
  await page.waitForTimeout(1200);
  out.inputY.push(await y());
  out.options = await page.locator('#search-results [role=option] .search-artist-name, #search-results [role=option] .search-title').allTextContents();
  out.expanded = await input.getAttribute('aria-expanded');
  out.status = await page.locator('[data-search-status]').textContent();
  await page.keyboard.press('ArrowDown');
  out.activedescendant = await input.getAttribute('aria-activedescendant');
  out.selected = await page.locator('#search-results [aria-selected=true]').count();
  await page.screenshot({path: `${shots}/busca-instantanea-1024.png`});
  await page.locator('#search-dialog').evaluate(d => { d.scrollTop = 400; });
  await page.waitForTimeout(200);
  out.inputY.push(await y());
  // ArrowUp from the first option wraps to the last one; it must not hide under the sticky head.
  await page.locator('#search-dialog').evaluate(d => { d.scrollTop = 0; });
  // Wrap to the last option (list scrolled to the bottom), then go up to an artist card
  // above the visible area: scrolling it into view must not leave it under the sticky head.
  for (let i = 0; i < 8; i++) await page.keyboard.press('ArrowUp');
  await page.waitForTimeout(200);
  out.activeVisible = await page.evaluate(() => {
    const head = document.querySelector('.search-head').getBoundingClientRect();
    const option = document.querySelector('#search-results [aria-selected=true]').getBoundingClientRect();
    return option.top >= head.bottom - 1;
  });
  await page.locator('#search-dialog').evaluate(d => { d.scrollTop = 0; });
  await page.keyboard.press('Escape');
  out.listClosed = await input.getAttribute('aria-expanded') === 'false';
  out.dialogStillOpen = await page.locator('#search-dialog').evaluate(d => d.open);
  await page.keyboard.press('Escape');
  out.dialogClosedSecondEsc = !(await page.locator('#search-dialog').evaluate(d => d.open));
  out.focusAfterClose = await page.evaluate(() => document.activeElement?.tagName + '#' + (document.activeElement?.id || ''));
  await page.keyboard.press('/');
  await input.fill('Cache');
  await page.waitForSelector('#search-results [role=option]');
  await page.keyboard.press('ArrowDown');
  const target = await page.locator('#search-results [aria-selected=true] a').getAttribute('href');
  await Promise.all([page.waitForNavigation(), page.keyboard.press('Enter')]);
  out.enterNavigates = new URL(page.url()).pathname === target;
  // Full search page (Enter without a selected option): the field fills the row beside the button.
  const pageField = async () => page.evaluate(() => {
    const form = document.querySelector('.search-page-form'), field = form.querySelector('input'), button = form.querySelector('button');
    const f = field.getBoundingClientRect(), b = button.getBoundingClientRect(), r = form.getBoundingClientRect();
    return {field: Math.round(f.width), form: Math.round(r.width), beside: Math.abs(f.bottom - b.bottom) < 2 && b.left > f.right};
  });
  await page.goto(origin + '/cache-study/search?q=Cache');
  out.pageField1024 = await pageField();
  await page.screenshot({path: `${shots}/busca-pagina-1024.png`});
  await page.setViewportSize({width: 390, height: 844});
  await page.goto(origin + '/cache-study/search?q=Cache');
  out.pageField390 = await pageField();
  await page.screenshot({path: `${shots}/busca-pagina-390.png`});
  await page.goto(origin + '/cache-study');
  await page.keyboard.press('/');
  await page.locator('#search-input').fill('Cache');
  await page.waitForSelector('#search-results [role=option]');
  await page.screenshot({path: `${shots}/busca-instantanea-390.png`});
  console.log(JSON.stringify(out, null, 1));
  await browser.close();
})().catch(e => { console.error(e); process.exit(1); });
'''


def timing(origin, path, headers, runs=30):
    """Median time and size of a GET, measured from this process (no browser)."""
    samples, size = [], 0
    for _ in range(runs):
        start = time.perf_counter()
        with urllib.request.urlopen(urllib.request.Request(origin + path, headers=headers)) as response:
            size = len(response.read())
        samples.append((time.perf_counter() - start) * 1000)
    samples.sort()
    return round(samples[len(samples) // 2], 2), size


def main():
    binary, shots = Path(sys.argv[1]).resolve(strict=True), Path(sys.argv[2]).resolve()
    with tempfile.TemporaryDirectory(prefix='tunetrail-107-', dir='/tmp') as tmp:
        server = Server(binary, Path(tmp) / 'data')
        try:
            server.start()
            server.login()
            server.request('POST', WEB + '/profiles', {'display_name': 'Cache Study', 'slug': 'cache-study',
                                                      'avatar_icon': 'headphones'}, admin=True, expected=201)
            key = server.request('GET', WEB + '/profiles/cache-study/user/apikeys', admin=True)[0]['key']
            pairs = [('Cache Band', 'Cache Song'), ('Cadence', 'Cascade'), ('Other', 'Cachet')]
            pairs += [(f'Cache Artist {n}', f'Cache Track {n}') for n in range(1, 6)]
            for offset, (artist, title) in enumerate(pairs):
                body = scrobble(int(time.time()) - 3600 - offset)
                body['payload'][0]['track_metadata'].update({'artist_name': artist, 'track_name': title})
                server.request('POST', LBZ + '/submit-listens', body, token=key)
            script = Path(tmp) / 'check.cjs'
            script.write_text(NODE)
            result = subprocess.run(['node', str(script), server.origin, str(shots)], capture_output=True, text=True,
                                    timeout=240, env=os.environ)
            if result.returncode:
                print(result.stdout, result.stderr[:400], result.stderr[-1500:])
                return 1
            r = json.loads(result.stdout)
            r['timings_ms_bytes'] = {
                'full_search_page': timing(server.origin, '/cache-study/search?q=Cache', {}),
                'fragment': timing(server.origin, '/cache-study/search?q=Cache&partial=1', {'X-TuneTrail-Partial': '1'}),
            }
            print(json.dumps(r, indent=1))
            ok = (r['dialogOpen'] and r['options'] and all('Cach' in o or 'Cache' in o for o in r['options'])
                  and r['expanded'] == 'true' and r['status'] and r['activedescendant'] and r['selected'] == 1
                  and r['listClosed'] and r['dialogStillOpen'] and r['dialogClosedSecondEsc'] and r['enterNavigates']
                  and r['focusAfterClose'] != 'INPUT#search-input'
                  and len(set(r['inputY'])) == 1 and r['submitHidden'] and r['activeVisible']
                  and r['focusOnLoad'] == 'BODY' and r['focusOnOpen'] == 'search-input'
                  and r['pageField1024']['beside'] and r['pageField390']['beside']
                  and r['pageField1024']['field'] > r['pageField390']['field'] > 150)
            print('PASS' if ok else 'FAIL')
            return 0 if ok else 1
        finally:
            server.stop()


if __name__ == '__main__':
    sys.exit(main())
