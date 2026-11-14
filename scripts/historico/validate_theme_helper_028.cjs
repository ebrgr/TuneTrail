/* Exercise the global theme editor without reading any profile data. */
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const origins = process.argv.slice(2);
if (origins.length !== 3 || origins.some(x => !/^http:\/\/127\.0\.0\.1:\d+$/.test(x))) throw Error('expected three loopback origins');
(async () => {
  const browser = await chromium.launch({headless: true, executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    for (const [i, origin] of origins.entries()) {
      const context = await browser.newContext();
      await context.route('**/*', r => new URL(r.request().url()).hostname === '127.0.0.1' ? r.continue() : r.abort());
      const page = await context.newPage();
      const errors = [], profileRequests = [];
      page.on('pageerror', error => errors.push(error.message));
      page.on('console', message => { if (message.type() === 'error' && !message.text().startsWith('Failed to load resource:')) errors.push(message.text()); });
      page.on('request', request => {
        const path = new URL(request.url()).pathname;
        if (page.url().includes('/theme-helper') && /\/apis\/web\/v1\/(?:profiles\/[^/]+|stats|top\/|listen-activity|listens|first-activity)(?:\/|$)/.test(path)) profileRequests.push(path);
      });
      await page.goto(origin + '/theme-helper');
      const input = page.locator('#custom-theme-input');
      await input.waitFor({state:'visible', timeout:12000});
      await page.getByText('Example artist', {exact:true}).first().waitFor();
      const theme = JSON.parse(await input.inputValue());
      theme.bg = '#123456'; theme.primary = '#fedcba';
      await input.fill(JSON.stringify(theme));
      await page.getByRole('button', {name:'Apply theme'}).click();
      await page.waitForFunction(() => document.documentElement.getAttribute('data-theme') === 'custom' && getComputedStyle(document.documentElement).getPropertyValue('--color-bg').trim() === '#123456');
      await page.reload();
      await input.waitFor({state:'visible'});
      await page.waitForFunction(() => document.documentElement.getAttribute('data-theme') === 'custom' && getComputedStyle(document.documentElement).getPropertyValue('--color-bg').trim() === '#123456');
      for (const invalid of ['{', 'null', '[]', '{}', JSON.stringify({...theme, bg:'not-a-color'})]) {
        await input.fill(invalid);
        await page.getByRole('button', {name:'Apply theme'}).click();
        await page.getByRole('alert').getByText('Enter a complete JSON theme with valid CSS colors.').waitFor();
        if (await page.evaluate(() => localStorage.getItem('theme')) !== 'custom' || await page.evaluate(() => document.documentElement.style.getPropertyValue('--color-bg')) !== '#123456') throw Error('invalid palette changed active theme');
      }
      await page.goto(origin + '/');
      await page.getByRole('button', {name:'Settings'}).click();
      await page.getByText('Yuu', {exact:true}).click();
      await page.waitForFunction(() => document.documentElement.getAttribute('data-theme') === 'yuu' && !document.documentElement.style.getPropertyValue('--color-bg'));
      await page.evaluate(() => { localStorage.setItem('theme', 'custom'); localStorage.setItem('custom-theme', 'null'); });
      await page.goto(origin + '/theme-helper');
      await input.waitFor({state:'visible'});
      await page.waitForFunction(() => document.documentElement.getAttribute('data-theme') !== 'custom');
      if (errors.length || profileRequests.length) throw Error(JSON.stringify({errors, profileRequests}));
      console.log(['zero','one','many'][i] + ': direct editor, fictional preview, apply and reload OK; no profile data requests');
      await context.close();
    }
  } finally { await browser.close(); }
})().catch(e => { console.error(e.stack); process.exitCode = 1; });
