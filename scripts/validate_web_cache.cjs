/* Real Chrome/Firefox history checks. Only the Python runner's disposable SQLite may be changed. */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const net = require('node:net');
const http = require('node:http');
const https = require('node:https');
const {spawn, execFileSync} = require('node:child_process');
const {X509Certificate, createHash} = require('node:crypto');
const [backend, database, output, mode, profileID, key, cert] = process.argv.slice(2);
let phase = 'setup';
const networkAssets = [];
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

// This boundary rejects database paths outside a fixture explicitly created by the runner.
function expireFixtureSessions() {
  assert(fs.existsSync(path.join(path.dirname(database), 'CACHE_TEST_FIXTURE')));
  assert(path.basename(path.dirname(path.dirname(database))).startsWith('tunetrail-cache106-'));
  execFileSync('python3', ['-c',
    'import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute("UPDATE sessions SET expires_at=1"); c.commit(); c.close()', database]);
}

// Observe real pageshow events without modifying cache or injecting synthetic restoration.
const observe = () => {
  window.__cacheDoc = crypto.randomUUID();
  addEventListener('pageshow', event => {
    window.__cachePersisted = event.persisted;
    const events = JSON.parse(sessionStorage.getItem('cache-events') || '[]');
    events.push({path: location.pathname, persisted: event.persisted});
    sessionStorage.setItem('cache-events', JSON.stringify(events));
  });
};

// Use an isolated Chrome profile with normal HTTP and back/forward caches enabled.
async function chrome(origin) {
  const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
  const tlsArgs = [];
  if (mode === 'https') {
    // Trust only this disposable certificate in this browser; never alter system trust.
    const publicKey = new X509Certificate(fs.readFileSync(cert)).publicKey.export({type:'spki',format:'der'});
    tlsArgs.push('--ignore-certificate-errors-spki-list='+createHash('sha256').update(publicKey).digest('base64'));
  }
  const browser = await chromium.launch({
    headless: true,
    executablePath: process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    ignoreDefaultArgs: ['--disable-back-forward-cache'],
    args: ['--host-resolver-rules=MAP fonts.googleapis.com ~NOTFOUND, MAP fonts.gstatic.com ~NOTFOUND', ...tlsArgs],
  });
  const context = await browser.newContext({locale: 'en-US', timezoneId: 'UTC'});
  await context.addCookies([
    {name: 'tz', value: 'UTC', url: origin}, {name: 'tunetrail-weekstart', value: '0', url: origin},
  ]);
  await context.addInitScript(observe);
  const page = await context.newPage();
  page.on('dialog', dialog => dialog.accept());
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.setBlockedURLs', {urls: ['https://fonts.googleapis.com/*', 'https://fonts.gstatic.com/*']});
  return {
    version: browser.version(),
    evaluate: expression => page.evaluate(expression),
    goto: url => page.goto(url, {waitUntil: 'load'}),
    back: () => page.goBack({waitUntil: 'commit'}),
    offline: value => context.setOffline(value),
    close: () => browser.close(),
  };
}

// The installed, unpatched Firefox speaks standard WebDriver BiDi, not Playwright/Juggler.
async function firefox(origin) {
  const listener = net.createServer();
  await new Promise(resolve => listener.listen(0, '127.0.0.1', resolve));
  const port = listener.address().port;
  await new Promise(resolve => listener.close(resolve));
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'tunetrail-firefox106-'));
  fs.writeFileSync(path.join(profile, 'user.js'), [
    'user_pref("network.dns.localDomains", "fonts.googleapis.com,fonts.gstatic.com");',
    'user_pref("network.proxy.type", 0);',
    'user_pref("intl.accept_languages", "en-US,en");',
    'user_pref("browser.shell.checkDefaultBrowser", false);',
  ].join('\n'));
  const child = spawn(process.env.TUNETRAIL_TEST_FIREFOX_PATH || '/Applications/Firefox.app/Contents/MacOS/firefox',
    ['--headless', '--no-remote', '--profile', profile, '--remote-debugging-port', String(port)], {stdio: 'ignore'});
  let socket;
  try {
    for (let i=0; i<100; i++) {
      try {
        socket = await new Promise((resolve, reject) => {
          const ws = new WebSocket(`ws://127.0.0.1:${port}/session`);
          ws.addEventListener('open', () => resolve(ws), {once: true});
          ws.addEventListener('error', reject, {once: true});
        });
        break;
      } catch { await sleep(100); }
    }
    assert(socket, 'Firefox BiDi readiness');
    let sequence = 0;
    const pending = new Map();
    const send = (method, params = {}) => new Promise((resolve, reject) => {
      const id = ++sequence;
      const timer = setTimeout(() => { pending.delete(id); reject(Error('BiDi timeout '+method)); }, 20000);
      pending.set(id, {resolve, reject, timer});
      socket.send(JSON.stringify({id, method, params}));
    });
    socket.addEventListener('message', event => {
      const message = JSON.parse(event.data);
      if (message.method === 'browsingContext.userPromptOpened') {
        send('browsingContext.handleUserPrompt', {context: message.params.context, accept: true}).catch(() => {});
      }
      const request = pending.get(message.id);
      if (!request) return;
      clearTimeout(request.timer);
      pending.delete(message.id);
      if (message.type === 'error') request.reject(Error(JSON.stringify(message)));
      else request.resolve(message.result);
    });
    const session = await send('session.new', {capabilities: {alwaysMatch: {acceptInsecureCerts: true}}});
    await send('session.subscribe', {events: ['browsingContext.userPromptOpened']});
    await send('script.addPreloadScript', {functionDeclaration: observe.toString()});
    const {context} = await send('browsingContext.create', {type: 'tab'});
    for (const [name,value] of [['tz','UTC'],['tunetrail-weekstart','0']]) {
      await send('storage.setCookie', {cookie: {name, value: {type: 'string', value}, domain: '127.0.0.1', path: '/', sameSite: 'lax', secure: origin.startsWith('https:')}, partition: {type: 'context', context}});
    }
    return {
      version: session.capabilities.browserVersion,
      evaluate: async expression => {
        const result = await send('script.evaluate', {
          expression: `(async()=>JSON.stringify(await (${expression})))()`,
          target: {context}, awaitPromise: true,
        });
        if (result.type === 'exception') throw Error(result.exceptionDetails.text);
        return result.result.value === undefined ? undefined : JSON.parse(result.result.value);
      },
      goto: url => send('browsingContext.navigate', {context, url, wait: 'complete'}),
      back: () => send('browsingContext.traverseHistory', {context, delta: -1}),
      close: async () => {
        socket.close();
        const exited = new Promise(resolve => child.once('exit', resolve));
        child.kill();
        await Promise.race([exited, sleep(3000)]);
        fs.rmSync(profile, {recursive: true, force: true});
      },
    };
  } catch (error) {
    socket?.close(); child.kill();
    fs.rmSync(profile, {recursive: true, force: true});
    throw error;
  }
}

// Run the same cache, session and history invariants in each real browser.
async function run(driver, origin) {
  const wait = async (expression, label) => {
    for (let i=0; i<100; i++) {
      try { if (await driver.evaluate(expression)) return; } catch {}
      await sleep(100);
    }
    throw Error('Timeout: '+label);
  };
  const goto = async route => { await driver.goto(origin+route); await wait('document.readyState === "complete"', route); await sleep(250); };
  const home = '/cache-study?period=all_time&tz=UTC';
  const search = '/cache-study/search?q=Validation&period=all_time&tz=UTC';
  const resourceExpression = `performance.getEntriesByType('resource').filter(r=>new URL(r.name).pathname.startsWith('/web-assets/') && /\.(css|js|json)$/.test(new URL(r.name).pathname)).map(r=>({url:new URL(r.name).pathname+new URL(r.name).search,transfer:r.transferSize,bytes:r.encodedBodySize}))`;
  const controls = 'document.querySelectorAll("[data-listen-controls]").length';
  const rows = 'document.querySelectorAll("tr[data-track-id][data-unix]").length';
  const login = () => driver.evaluate(`fetch('/apis/web/v1/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:'validation-admin',password:'validation-password'})}).then(r=>r.status)`);
  const logout = () => driver.evaluate(`fetch('/apis/web/v1/logout',{method:'POST'}).then(r=>r.status)`);
  const results = {mode, browser: driver.version, fixtureOnly: true, checks: []};
  const check = name => results.checks.push(name);

  phase = 'first';
  await goto(home);
  await wait(`performance.getEntriesByType('resource').some(r=>new URL(r.name).pathname==='/web-assets/themes.json') && document.getElementById('custom-theme-input').value.length>0`, 'initial palette request completed');
  const first = await driver.evaluate(resourceExpression);
  phase = 'second';
  await goto(search);
  await wait(`performance.getEntriesByType('resource').some(r=>new URL(r.name).pathname==='/web-assets/themes.json')`, 'second palette request completed');
  const second = await driver.evaluate(resourceExpression);
  assert(first.length >= 6 && second.length >= 6, 'resource timing coverage');
  assert(networkAssets.some(r=>r.phase==='first' && r.bytes>0), 'origin saw initial asset bodies');
  assert(!networkAssets.some(r=>r.phase==='second'), 'second navigation contacted the asset origin');
  assert(second.every(r=>r.transfer===0), 'second navigation transferred assets: '+JSON.stringify({first,second}));
  assert(second.every(r=>r.url.includes('?v=')), 'unversioned asset');
  results.cache = {first, second, originRequests: networkAssets.filter(r=>r.phase==='first'||r.phase==='second')};
  phase = 'history';
  check('second navigation reuses CSS/JS/JSON without body transfer');
  await goto(home);
  await driver.evaluate(`(()=>{const image=document.querySelector('img[data-cover]');image.classList.remove('cover-failed');image.removeAttribute('srcset');image.src='/fictional-missing-cover.png'})()`);
  await wait(`(()=>{const image=document.querySelector('img[data-cover]');return image.classList.contains('cover-failed') && image.complete && image.naturalWidth>0 && new URL(image.src).pathname==='/web-assets/icons/Disc3.svg' && new URL(image.src).searchParams.has('v')})()`, 'versioned cover fallback');
  check('failed artwork uses a functioning versioned local SVG fallback');


  await goto(home);
  const id = await driver.evaluate('window.__cacheDoc');
  await driver.evaluate('window.scrollTo(0,400)');
  const scroll = await driver.evaluate('scrollY');
  await goto(search);
  await driver.back();
  await wait(`window.__cachePersisted === true && window.__cacheDoc === ${JSON.stringify(id)}`, 'actual bfcache restoration');
  assert.equal(await driver.evaluate('scrollY'), scroll);
  assert.equal(await driver.evaluate('location.search'), '?period=all_time&tz=UTC');
  check('actual bfcache document, query and scroll restored');

  await goto('/');
  const selectionID = await driver.evaluate('window.__cacheDoc');
  await goto(search);
  await driver.evaluate(`document.cookie='tunetrail-weekstart=1; Path=/'`);
  await driver.back();
  await wait(`window.__cachePersisted === true && window.__cacheDoc === ${JSON.stringify(selectionID)}`, 'selection restore without false week reload');
  await sleep(300);
  assert.equal(await driver.evaluate('window.__cacheDoc'), selectionID);
  await driver.evaluate(`document.cookie='tunetrail-weekstart=0; Path=/'`);
  check('profile selection ignores unresolved week preference and preserves bfcache');

  await driver.evaluate(`document.cookie='tz=UnknownZone; Path=/'`);
  await goto('/cache-study?period=all_time');
  const fallbackZoneID = await driver.evaluate('window.__cacheDoc');
  await goto(search);
  await driver.back();
  await wait(`window.__cachePersisted === true && window.__cacheDoc === ${JSON.stringify(fallbackZoneID)}`, 'unknown timezone without repeated reload');
  await sleep(300);
  assert.equal(await driver.evaluate('window.__cacheDoc'), fallbackZoneID);
  await driver.evaluate(`document.cookie='tz=UTC; Path=/'`);
  check('unknown timezone preference uses the server fallback without restore loops');

  assert.equal(await login(), 204);
  await goto(home);
  await wait(`${controls} === ${rows} && ${rows}>0`, 'authenticated controls');
  for (let cycle=0; cycle<2; cycle++) {
    await goto(search);
    await driver.back();
    await wait(`window.__cachePersisted === true && ${controls} === ${rows} && ${rows}>0`, 'reactivated controls');
  }
  check('two bfcache cycles restore one administrative control per row');

  await goto(search);
  assert.equal(await logout(), 204);
  await driver.back();
  await wait('window.__cachePersisted === true', 'logout restore');
  await sleep(300);
  assert.equal(await driver.evaluate(controls), 0);
  check('logout while away removes restored mutation controls');

  assert.equal(await login(), 204);
  await goto(home);
  await wait(`${controls}>0`, 'controls before expiry');
  await goto(search);
  expireFixtureSessions();
  await driver.back();
  await wait('window.__cachePersisted === true', 'expired restore');
  await sleep(300);
  assert.equal(await driver.evaluate(controls), 0);
  check('server-side expiry without client cookie change removes controls');

  assert.equal(await login(), 204);
  await goto(home);
  const count = await driver.evaluate(rows);
  await wait(`${controls}>0`, 'delete controls');
  await goto(search); await driver.back();
  await wait(`${controls}>0 && window.__cachePersisted === true`, 'delete after restoration');
  await driver.evaluate('document.querySelector(".delete-listen").click()');
  await wait('!!document.querySelector("#selected-listen")', 'listen confirmation after restoration');
  await driver.evaluate('document.querySelector("#selected-listen input[name=confirm]").checked=true; document.querySelector("#selected-listen form").requestSubmit()');
  await wait(`${rows} < ${count}`, 'confirmed deletion after restoration');
  check('Confirmed listen deletion works after restoration with fresh authorization');

  if (driver.offline) {
    assert.equal(await login(), 204);
    await goto(home);
    await wait(`${controls}>0`, 'controls before offline restore');
    await goto(search);
    await driver.offline(true);
    try {
      await driver.back();
      await wait('window.__cachePersisted === true', 'offline bfcache restore');
      await sleep(300);
      assert.equal(await driver.evaluate(controls), 0);
    } finally { await driver.offline(false); }
    check('offline restoration keeps public snapshot without mutation controls');
  }

  await goto(home);
  const languageID = await driver.evaluate('window.__cacheDoc');
  await goto(search);
  const languageStatus = await driver.evaluate(`fetch('/language',{method:'POST',body:new URLSearchParams({language:'pt-BR',return_to:${JSON.stringify(home)}})}).then(async r=>{await r.text();return r.status})`);
  assert.equal(languageStatus, 200);
  await driver.back();
  await wait(`document.documentElement.lang === 'pt-BR' && window.__cacheDoc !== ${JSON.stringify(languageID)}`, 'locale revalidation on restore');
  const resetLanguage = await driver.evaluate(`fetch('/language',{method:'POST',body:new URLSearchParams({language:'en',return_to:${JSON.stringify(home)}})}).then(async r=>{await r.text();return r.status})`);
  assert.equal(resetLanguage, 200);
  check('restored locale mismatch reloads with the current HttpOnly language preference');

  assert.equal(await login(), 204);
  await goto('/admin?profile='+profileID);
  const adminID = await driver.evaluate('window.__cacheDoc');
  await goto(search);
  expireFixtureSessions();
  await driver.back();
  await wait(`document.querySelector('input[name="username"]') && !document.body.hidden`, 'administrative reload with expired session');
  assert.notEqual(await driver.evaluate('window.__cacheDoc'), adminID);
  check('administrative history revalidates expired session and does not reuse forms');
  results.history = await driver.evaluate('JSON.parse(sessionStorage.getItem("cache-events"))');
  fs.writeFileSync(path.join(output,'browser.json'), JSON.stringify(results,null,2)+'\n');
}

(async () => {
  let proxy;
  let origin = backend;
  let driver;
  try {
    // Record actual origin traffic as well as Resource Timing: a browser may reload its first page.
    const forward = (req,res) => {
      const route = new URL(req.url, backend);
      const tracked = route.pathname.startsWith('/web-assets/') && /\.(css|js|json)$/.test(route.pathname);
      const entry = {phase, url: route.pathname+route.search, bytes: 0, status: 0};
      if (tracked) networkAssets.push(entry);
      const upstream = http.request(backend+req.url, {
        method:req.method, headers:{...req.headers,'x-forwarded-proto':mode==='https'?'https':'http'},
      }, response => {
        entry.status = response.statusCode;
        response.on('data', chunk => { entry.bytes += chunk.length; });
        res.writeHead(response.statusCode,response.headers);
        response.pipe(res);
      });
      upstream.on('error', () => {res.writeHead(502);res.end();});
      req.pipe(upstream);
    };
    proxy = mode === 'https'
      ? https.createServer({key:fs.readFileSync(key),cert:fs.readFileSync(cert)},forward)
      : http.createServer(forward);
    await new Promise(resolve => proxy.listen(0,'127.0.0.1',resolve));
    origin=(mode==='https'?'https':'http')+'://127.0.0.1:'+proxy.address().port;
    driver = mode === 'firefox' ? await firefox(origin) : await chrome(origin);
    await run(driver,origin);
    console.log('PASS cache/history '+mode+' '+driver.version);
  } finally {
    if (driver) await driver.close();
    if (proxy) await new Promise(resolve => proxy.close(resolve));
  }
})().catch(error => { console.error(error);process.exitCode=1; });
