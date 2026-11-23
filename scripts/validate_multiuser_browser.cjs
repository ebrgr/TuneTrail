/* Run against the disposable server created by validate_multiuser_browser.py. */
const assert = require('node:assert/strict');
const {execFileSync} = require('node:child_process');
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const [origin, database, password, mode] = process.argv.slice(2);
assert.ok(['on', 'off'].includes(mode));
assert.match(origin, /^http:\/\/127\.0\.0\.1:\d+$/);

async function visit(page, path, status) {
  assert.equal((await page.goto(origin + path)).status(), status, path);
}

async function signIn(page, username) {
  await visit(page, '/login', 200);
  const form = page.locator('form[action="/admin/login"]');
  await form.locator('[name=username]').fill(username);
  await form.locator('[name=password]').fill(password);
  const destination = username === 'validation-admin' ? '/admin' : '/account';
  await Promise.all([page.waitForURL(origin + destination), form.locator('button').click()]);
}

async function exercise(browser, javaScriptEnabled) {
  const context = await browser.newContext({javaScriptEnabled, locale: 'en-US'});
  const visitorContext = await browser.newContext({javaScriptEnabled, locale: 'en-US'});
  const rootContext = await browser.newContext({javaScriptEnabled, locale: 'en-US'});
  try {
    for (const ctx of [context, visitorContext, rootContext]) {
      await ctx.route('**/*', route => new URL(route.request().url()).origin === origin
        ? route.continue() : route.abort());
    }
    const owner = await context.newPage();
    const visitor = await visitorContext.newPage();
    const root = await rootContext.newPage();
    await visit(visitor, '/', 200);
    await visit(visitor, '/listener', 200);
    await signIn(owner, 'listener');
    await signIn(root, 'validation-admin');
    await visit(owner, '/default', 200);
    await visit(owner, '/admin/settings', 403);
    await visit(root, '/admin/settings', 200);
    await visit(owner, '/account/profile', 200);
    const form = owner.locator('#profile-identity form');
    await form.locator('[name=require_login]').check();
    const saved = owner.waitForResponse(r => r.request().method() === 'POST'
      && /\/admin\/profile\/\d+\/save$/.test(r.url()));
    await form.locator('button').click();
    assert.ok([200, 303].includes((await saved).status()));
    await visit(visitor, '/listener', 404);
    const protectedMessage = await visitor.locator('main h1').innerText();
    await visit(visitor, '/missing-profile', 404);
    assert.equal(await visitor.locator('main h1').innerText(), protectedMessage);
    await visit(owner, '/listener', 200);
    await visit(root, '/listener', 200);
    await visit(root, '/other', 200);
    await visit(owner, '/other', 404);
    assert.equal((await context.request.get(origin + '/apis/web/v1/profiles/other/stats')).status(), 404);
    assert.equal((await context.request.get(origin + '/apis/web/v1/profiles/listener/management')).status(), 200);

    // Use the live cookies on either side of a server-side role/session change.
    execFileSync('python3', ['-c', `
import sqlite3, sys
with sqlite3.connect(sys.argv[1]) as conn:
    conn.execute("UPDATE users SET role='admin' WHERE username='other'")
    conn.execute("UPDATE users SET role='user' WHERE username='validation-admin'")
    conn.execute("UPDATE sessions SET expires_at=1 WHERE user_id=(SELECT id FROM users WHERE username='listener')")
`, database]);
    await visit(root, '/admin/settings', 403);
    await visit(root, '/other', 404);
    await visit(owner, '/account', 200);
    assert.equal(await owner.locator('form[action="/admin/login"]').count(), 1);
    assert.equal((await context.request.get(origin + '/apis/web/v1/profiles/listener/management')).status(), 401);
    await visit(owner, '/listener', 404);

    await visit(root, '/admin/sign-out', 200);
    assert.equal((await rootContext.request.get(origin + '/apis/web/v1/user')).status(), 200);
    await Promise.all([
      root.waitForURL(origin + '/'),
      root.locator('form[action="/admin/logout"] button[type=submit]').click(),
    ]);
    assert.equal((await rootContext.request.get(origin + '/apis/web/v1/user')).status(), 401);
    console.log(`PASS browser JavaScript ${javaScriptEnabled ? 'on' : 'off'}`);
  } finally {
    await Promise.all([context.close(), visitorContext.close(), rootContext.close()]);
  }
}

(async () => {
  const browser = await chromium.launch({executablePath: process.env.CHROME_PATH});
  try {
    await exercise(browser, mode === 'on');
  } finally {
    await browser.close();
  }
})().catch(error => {
  console.error(error.message);
  process.exitCode = 1;
});
