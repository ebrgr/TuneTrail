/* ISSUE-071: exercise native disclosure, accessible names, focus and geometry on isolated fixtures. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http'),assert=require('node:assert/strict');
const root=path.resolve(process.argv[2]),out=path.resolve(process.argv[3]);
const types={'.html':'text/html','.css':'text/css','.js':'text/javascript','.json':'application/json','.svg':'image/svg+xml','.webp':'image/webp','.png':'image/png'};
const server=http.createServer((req,res)=>{
  const file=path.resolve(root,'.'+new URL(req.url,'http://localhost').pathname);
  if(!file.startsWith(root+path.sep)){res.writeHead(403).end();return;}
  fs.readFile(file,(err,data)=>{if(err){res.writeHead(404).end();return;}res.setHeader('Content-Type',types[path.extname(file)]||'application/octet-stream');res.end(data);});
});
(async()=>{
 fs.mkdirSync(out,{recursive:true});await new Promise(r=>server.listen(0,'127.0.0.1',r));
 const origin=`http://127.0.0.1:${server.address().port}`;
 const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const entries=[];
 try{
  for(const javaScriptEnabled of [false,true])for(const theme of ['yuu','pearl'])for(const width of [320,360,390,768,1024,1440]){
   const context=await browser.newContext({javaScriptEnabled,viewport:{width,height:900},reducedMotion:'reduce'});
   await context.route('**/*',r=>new URL(r.request().url()).origin===origin?r.continue():r.abort());
   const page=await context.newPage();
   for(const name of ['public/home','admin/account','preview/home']){
    await page.goto(`${origin}/${name}.html`,{waitUntil:'load'});
    // No-script fixtures use the server's built-in theme; set its equivalent CSS state for both palettes.
    await page.evaluate(t=>document.documentElement.setAttribute('data-theme',t),theme);
    const bar=page.locator('.sidebar').first(),summary=bar.locator('summary');
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>document.documentElement.clientWidth),false,`${name} ${width} overflow`);
    const geometry=await bar.boundingBox();
    if(width<640){assert.equal(Math.round(geometry.y+geometry.height),900);assert.equal(Math.round(geometry.width),width);}
    else assert.equal(Math.round(geometry.width),width<1024?72:224);
    if(width>=1024)assert.equal(await summary.evaluate(e=>getComputedStyle(e,'::after').content),'none','visible labels suppress tooltips');
    // Native summary opens with Enter and exposes ordinary controls in reading order.
    await summary.focus();await page.keyboard.press('Enter');
    const palette=bar.locator('[popovertarget="theme-panel"]');
    assert.equal(await palette.isVisible(),true,'More must open without scripts');
    await page.keyboard.press('Tab');
    assert.equal(await bar.locator('.nav-options>a').first().evaluate(e=>e===document.activeElement),true);
    await page.keyboard.press('Tab');assert.equal(await palette.evaluate(e=>e===document.activeElement),true);
    const targets=await bar.locator('a,button,summary,input').evaluateAll(els=>els.filter(e=>e.checkVisibility()).map(e=>{
     const r=e.getBoundingClientRect();return {name:e.getAttribute('aria-label')||e.textContent.trim(),width:r.width,height:r.height};
    }));
    for(const t of targets){assert.ok(t.name,JSON.stringify(t));assert.ok(t.width>=44&&t.height>=44,`${width} ${name}: ${JSON.stringify(t)}`);}
    for(const target of await page.locator('.period-selector a,.profile-switch,.admin-avatar-link').all()){
     if(await target.isVisible()){const box=await target.boundingBox();assert.ok(box.width>=44&&box.height>=44,'period/profile target');}
    }
    const tree=await bar.ariaSnapshot();assert.match(tree,/Main navigation/);assert.match(tree,/Theme and language/);
    const cdp=await context.newCDPSession(page);
    const {nodes}=await cdp.send('Accessibility.getFullAXTree');
    const moreNode=nodes.find(n=>n.name?.value==='More'&&n.properties?.some(p=>p.name==='expanded'));
    assert.ok(moreNode,'native summary has an accessible expansion state');
    assert.equal(moreNode.properties.find(p=>p.name==='expanded').value.value,true);
    await cdp.detach();
    await page.keyboard.press('Enter');
    assert.equal(await page.locator('#theme-panel').evaluate(e=>e.matches(':popover-open')),true);
    await page.keyboard.press('Escape');
    if(width===390&&name==='public/home'){
     await page.screenshot({path:path.join(out,`more-${theme}-${javaScriptEnabled?'js':'nojs'}.png`)});
     fs.writeFileSync(path.join(out,`accessibility-${theme}-${javaScriptEnabled?'js':'nojs'}.txt`),tree);
    }
    if(javaScriptEnabled){
     const toggle=bar.locator('[data-nav-toggle]');
     if(await page.locator('#nav-admin').isVisible())await toggle.click();
     if(!(await bar.locator('details').evaluate(e=>e.open))){await summary.focus();await page.keyboard.press('Enter');}
     await toggle.click();
     assert.equal(await page.locator('#nav-admin').isVisible(),true);
     if(width>=640)assert.equal(Math.round((await page.locator('#nav-admin').boundingBox()).height),900);
     for(const target of await page.locator('#nav-admin a,#nav-admin button').all()){
      if(await target.isVisible()){const box=await target.boundingBox();assert.ok(box.width>=44&&box.height>=44,'secondary target');}
     }
     assert.equal(await bar.locator('details').getAttribute('open'),null,'More closes before secondary opens');
     if(name==='public/home'&&width===1440){
      const margin=()=>page.locator('.app-content').evaluate(e=>parseFloat(getComputedStyle(e).marginLeft));
      assert.equal(await margin(),296,'push reserves both sidebars');
      await page.screenshot({path:path.join(out,`admin-push-${theme}.png`)});
      await page.evaluate(()=>document.documentElement.dataset.navLayout='overlay');
      assert.equal(await margin(),224,'overlay preserves the primary sidebar offset');
      await page.screenshot({path:path.join(out,`admin-overlay-${theme}.png`)});
      await page.evaluate(()=>delete document.documentElement.dataset.navLayout);
     }
     await page.keyboard.press('Escape');assert.equal(await page.locator('#nav-admin').isVisible(),false);
     assert.equal(await toggle.evaluate(e=>e===document.activeElement),true);
    }
    // Close disclosure before inspecting footer clearance and the collapsed desktop state.
    await summary.focus();await page.keyboard.press('Enter');
    if(width>=1024){
     await bar.locator('input[type=checkbox]').uncheck();
     assert.equal(Math.round((await bar.boundingBox()).width),72);
     if(width===1440&&name==='public/home')await page.screenshot({path:path.join(out,`collapsed-${theme}-${javaScriptEnabled?'js':'nojs'}.png`)});
    }
    await page.evaluate(()=>window.scrollTo(0,document.documentElement.scrollHeight));
    const footer=await page.locator('footer').boundingBox();
    if(width<640){const dock=await (await page.locator('#nav-admin').isVisible()?page.locator('#nav-admin'):bar).boundingBox();assert.ok(footer.y+footer.height<=dock.y+1,`${name} ${width}: footer under dock ${JSON.stringify({footer,dock})}`);}
    entries.push({name,width,theme,javaScriptEnabled,geometry,targets,accessibleTree:true,keyboard:true,footerClear:true});
   }
   await context.close();
  }
  fs.writeFileSync(path.join(out,'navigation.json'),JSON.stringify({browser:browser.version(),entries},null,2)+'\n');
  console.log(`PASS ${entries.length} navigation views, native disclosure, keyboard, accessible tree, targets and footer`);
 }finally{await browser.close();server.close();}
})().catch(e=>{console.error(e);server.close();process.exitCode=1;});
