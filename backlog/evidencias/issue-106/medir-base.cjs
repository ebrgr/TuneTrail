/* Measure only disposable loopback fixtures; HTTP cache remains enabled. */
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');

(async () => {
  const origin = process.argv[2];
  const browser = await chromium.launch({
    headless: true,
    executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    ignoreDefaultArgs: ['--disable-back-forward-cache'],
    args: ['--host-resolver-rules=MAP fonts.googleapis.com ~NOTFOUND, MAP fonts.gstatic.com ~NOTFOUND'],
  });
  try {
    const context = await browser.newContext({ locale: 'en-US', timezoneId: 'UTC' });
    await context.addCookies([
      { name: 'tz', value: 'UTC', url: origin },
      { name: 'tunetrail-weekstart', value: '0', url: origin },
    ]);
    await context.addInitScript(() => {
      window.__studyDoc = crypto.randomUUID();
      addEventListener('pageshow', event => {
        window.__studyPersisted = event.persisted;
        window.__studyShown = true;
      });
    });
    const page = await context.newPage();
    let phase = 'first';
    const network = [];
    const cdp = await context.newCDPSession(page);
    await cdp.send('Network.enable');
    await cdp.send('Network.setBlockedURLs', {
      urls: ['https://fonts.googleapis.com/*', 'https://fonts.gstatic.com/*'],
    });
    cdp.on('Network.responseReceived', event => {
      const url = new URL(event.response.url);
      if (!url.pathname.startsWith('/web-assets/')) return;
      network.push({
        phase,
        path: url.pathname,
        status: event.response.status,
        fromDiskCache: !!event.response.fromDiskCache,
        fromPrefetchCache: !!event.response.fromPrefetchCache,
        encodedDataLength: event.response.encodedDataLength,
      });
    });
    const readResources = () => performance.getEntriesByType('resource')
      .filter(resource => resource.name.includes('/web-assets/'))
      .map(resource => ({
        path: new URL(resource.name).pathname,
        transferSize: resource.transferSize,
        encodedBodySize: resource.encodedBodySize,
      }));
    const home = '/study-alpha?period=all_time&tz=UTC';
    await page.goto(origin + home, { waitUntil: 'load' });
    await page.waitForFunction(() => window.__studyShown);
    await page.waitForTimeout(200);
    const firstID = await page.evaluate(() => window.__studyDoc);
    const first = { resources: await page.evaluate(readResources) };

    phase = 'second';
    await page.goto(origin + '/study-alpha/search?q=fictional&period=all_time&tz=UTC', { waitUntil: 'load' });
    await page.waitForFunction(() => window.__studyShown);
    await page.waitForTimeout(200);
    const second = { resources: await page.evaluate(readResources) };

    phase = 'back';
    await page.goBack({ waitUntil: 'commit', timeout: 10000 });
    await page.waitForFunction(() => window.__studyShown);
    await page.waitForTimeout(200);
    const restored = await page.evaluate(() => ({
      id: window.__studyDoc,
      persisted: window.__studyPersisted,
      reasons: performance.getEntriesByType('navigation')[0]?.notRestoredReasons?.toJSON?.() || null,
    }));
    console.log(JSON.stringify({
      browser: browser.version(),
      headless: true,
      fixtureOnly: true,
      routeInterception: false,
      disableBackForwardCacheDefaultRemoved: true,
      externalFontResolutionBlocked: true,
      first,
      second,
      back: {
        persisted: restored.persisted,
        sameDocument: firstID === restored.id,
        notRestoredReasons: restored.reasons,
      },
      network,
    }, null, 2));
  } finally {
    await browser.close();
  }
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
