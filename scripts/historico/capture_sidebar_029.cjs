/* Compare public and administrative navigation with fictional data only. */
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const [origin, output, stage] = process.argv.slice(2);
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(origin) || !['before','after'].includes(stage)) throw Error('invalid arguments');
(async () => {
  const browser = await chromium.launch({headless:true, executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try {
    for (const [viewport, size] of [['desktop',{width:1440,height:1000}], ['mobile',{width:390,height:844}]]) {
      const context = await browser.newContext({viewport:size});
      await context.route('**/*', r => new URL(r.request().url()).hostname === '127.0.0.1' ? r.continue() : r.abort());
      const page = await context.newPage();
      await page.goto(origin + '/admin');
      await page.locator('[name=username]').fill('validation-admin');
      await page.locator('[name=password]').fill('validation-password');
      await page.getByRole('button',{name:'Sign in'}).click();
      for (const [name,path] of [['public','/alpha'],['account','/admin?section=account'],['profile','/admin?profile=1']]) {
        await page.goto(origin + path);
        await page.locator('nav[aria-label="Main navigation"]').waitFor();
        await page.screenshot({path:`${output}/${stage}-${name}-${viewport}.png`});
      }
      if (stage === 'after') {
        const nav = page.getByRole('navigation',{name:'Main navigation'});
        for (const icon of ['Music2','Headphones','Disc3','Radio','UserRound']) if (!(await page.request.get(origin+'/web-assets/icons/'+icon+'.svg')).ok()) throw Error('avatar icon asset missing: '+icon);
        if (await nav.getByRole('link',{name:'Home'}).getAttribute('href') !== '/alpha') throw Error('home lost profile');
        if (await nav.getByRole('link',{name:'Search'}).getAttribute('href') !== '/alpha/search') throw Error('search lost profile');
        if (await nav.getByRole('link',{name:'Rewind'}).getAttribute('href') !== '/alpha/rewind') throw Error('rewind lost profile');
        const box = await nav.boundingBox();
        if (viewport === 'desktop' ? Math.abs(box.width-58)>1 : Math.abs(box.width-size.width)>1) throw Error('navigation breakpoint mismatch');
        const settings=nav.getByRole('button',{name:'Settings'});
        await settings.click();
        const dialog=page.getByRole('dialog',{name:'Settings'});
        await dialog.getByLabel('Theme').selectOption('pearl');
        if (await page.locator('html').getAttribute('data-theme') !== 'pearl') throw Error('theme change failed');
        await page.keyboard.press('Escape');
        if (!(await settings.evaluate(el=>el===document.activeElement))) throw Error('focus not restored');
        await page.reload();
        if (await page.locator('html').getAttribute('data-theme') !== 'pearl') throw Error('theme not persisted');
        await page.goto(origin+'/admin?section=account');
        if (await nav.getByRole('link',{name:'Search'}).count()) throw Error('search without explicit profile');
        const storage=await context.storageState();
        const nojs=await browser.newContext({viewport:size,javaScriptEnabled:false,storageState:storage});
        const plain=await nojs.newPage();
        await plain.goto(origin+'/admin?profile=1');
        const plainNav=plain.getByRole('navigation',{name:'Main navigation'});
        if (await plainNav.getByRole('link',{name:'Home'}).getAttribute('href') !== '/alpha' || await plainNav.getByRole('link',{name:'Administrator account'}).getAttribute('href') !== '/admin?section=account') throw Error('no-JS navigation failed');
        await nojs.close();
        console.log(viewport+': navigation, profile paths, theme, focus and no-JS OK');
      }
      await context.close();
    }
  } finally { await browser.close(); }
})().catch(e=>{console.error(e.stack); process.exitCode=1});
