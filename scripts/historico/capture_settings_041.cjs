// Capture and exercise the settings page on a disposable TuneTrail fixture only.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const origin = process.env.TUNETRAIL_PREVIEW_ORIGIN;
const username = process.env.TUNETRAIL_PREVIEW_USER;
const password = process.env.TUNETRAIL_PREVIEW_PASSWORD;
const output = process.argv[2];
if (!origin || !username || !password || !output) throw new Error('A disposable preview origin, user, password and output directory are required.');

(async () => {
  fs.mkdirSync(output, { recursive: true });
  const browser = await chromium.launch({ headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' });
  const results = [];
  try {
    for (const [name, viewport] of Object.entries({ desktop: { width: 1440, height: 1000 }, mobile: { width: 390, height: 844 }, nojs: { width: 390, height: 844 } })) {
      const context = await browser.newContext({ viewport, javaScriptEnabled: name !== 'nojs' });
      await context.route('**/*', route => {
        const target = new URL(route.request().url());
        return target.origin === origin ? route.continue() : route.abort();
      });
      const page = await context.newPage();
      await page.goto(origin + '/admin/settings', { waitUntil: 'domcontentloaded' });
      await page.locator('input[name="username"]').fill(username);
      await page.locator('input[name="password"]').fill(password);
      await Promise.all([page.waitForURL('**/admin/settings'), page.getByRole('button', { name: 'Sign in' }).click()]);
      await page.getByRole('heading', { name: 'Server configuration' }).waitFor();
      const geometry = await page.evaluate(() => ({ document: document.documentElement.scrollWidth, viewport: document.documentElement.clientWidth }));
      if (geometry.document > geometry.viewport) throw new Error(`${name}: horizontal overflow ${JSON.stringify(geometry)}`);
      if (await page.locator('form.server-settings-form [name^="KOITO_"]').count() !== 10) throw new Error(`${name}: expected ten fields`);
      if (name !== 'nojs') {
        await page.screenshot({ path: path.join(output, `settings-${name}.png`), fullPage: true });
        await page.goto(origin + '/admin', { waitUntil: 'domcontentloaded' });
        await page.screenshot({ path: path.join(output, `admin-${name}.png`), fullPage: true });
        await page.goto(origin + '/admin/settings', { waitUntil: 'domcontentloaded' });
      }
      await page.keyboard.press('Tab');
      const focus = await page.evaluate(() => document.activeElement?.className || '');
      if (name === 'nojs') {
        await page.locator('[name="KOITO_LASTFM_API_KEY"]').fill('preview-value');
        await Promise.all([page.waitForURL('**/admin/settings'), page.getByRole('button', { name: 'Save configuration' }).click()]);
        await page.getByRole('status').filter({ hasText: 'Configuration saved' }).waitFor();
        if (await page.locator('[name="KOITO_LASTFM_API_KEY"]').inputValue() !== 'preview-value') throw new Error('No-JavaScript save did not persist');
      }
      results.push({ name, viewport, geometry, firstFocus: focus, noJavaScriptSave: name === 'nojs' ? 'passed' : undefined });
      await context.close();
    }
  } finally { await browser.close(); }
  fs.writeFileSync(path.join(output, 'results.json'), JSON.stringify(results, null, 2) + '\n');
  process.stdout.write(JSON.stringify(results, null, 2) + '\n');
})().catch(error => { console.error(error.message); process.exitCode = 1; });
