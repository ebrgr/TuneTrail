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
  const viewports=[...[320,360,390,639,640,641,768,1023,1024,1025,1440,1920].map(width=>[width,900]),[320,480],[640,320],[844,390]];
  for(const javaScriptEnabled of [false,true])for(const theme of ['yuu','pearl'])for(const [width,height] of viewports)
   for(const navLabels of javaScriptEnabled?['icons','labels']:['icons']){
   const context=await browser.newContext({javaScriptEnabled,viewport:{width,height},reducedMotion:'reduce'});
   await context.route('**/*',r=>new URL(r.request().url()).origin===origin?r.continue():r.abort());
   const page=await context.newPage();
   for(const name of ['public/home','admin/account','preview/home']){
    await page.goto(`${origin}/${name}.html`,{waitUntil:'load'});
    // No-script fixtures use the server's built-in theme; set its equivalent CSS state for both palettes.
    await page.evaluate(t=>document.documentElement.setAttribute('data-theme',t),theme);
    // ISSUE-154: icons-only is the default state; exercise the panel without pretending the pending server preference is persistent.
    if(javaScriptEnabled&&navLabels==='labels'){
     await page.keyboard.press('\\');
     await page.locator('[data-nav-labels-choice] input[value="labels"]').check();
     await page.keyboard.press('Escape');
    }
    const bar=page.locator('.sidebar').first(),summary=bar.locator('.nav-more>summary');
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>document.documentElement.clientWidth),false,`${name} ${width} overflow`);
    const geometry=await bar.boundingBox();
    if(width<640){assert.equal(Math.round(geometry.y+geometry.height),height);assert.equal(Math.round(geometry.width),width);}
    else assert.equal(Math.round(geometry.width),width<1024||navLabels==='icons'?58:224);
    if(width>=1024&&navLabels==='labels')assert.equal(await summary.evaluate(e=>getComputedStyle(e,'::after').content),'none','visible labels suppress tooltips');
    assert.equal(await bar.locator('.nav-collapse').count(),0,'no label checkbox remains in the sidebar');
    if(width===1440){
     for(const calendar of await page.locator('.activity-calendar').all()){
      const fits=await calendar.evaluate(e=>{
       const right=e.getBoundingClientRect().right;
       return e.scrollWidth<=e.clientWidth+1&&[...e.querySelectorAll('.activity-slot')].every(c=>c.getBoundingClientRect().right<=right+1);
      });
      assert.ok(fits,'wide heatmap shows every week without right-edge clipping');
     }
     const identity=page.locator('.profile-identity strong');
     if(await identity.count())assert.notEqual(await bar.locator('.nav-profile .nav-label').textContent(),await identity.first().textContent());
    }
    // Native summary opens with Enter and exposes ordinary controls in reading order.
    await summary.focus();await page.keyboard.press('Enter');
    const palette=bar.locator('[popovertarget="theme-panel"]');
    assert.equal(await palette.isVisible(),true,'More must open without scripts');
    const options=bar.locator('.nav-options');
    if(width<1024||navLabels==='icons'){
     const box=await options.boundingBox();
     assert.ok(box.x>=0&&box.y>=0&&box.x+box.width<=width+1&&box.y+box.height<=height+1,'More fits the viewport');
    }
    await page.keyboard.press('Tab');
    assert.equal(await bar.locator('.nav-options>a').first().evaluate(e=>e===document.activeElement),true,`${name} ${width}x${height} ${theme} ${navLabels} js=${javaScriptEnabled}: ${await page.evaluate(()=>document.activeElement.outerHTML.slice(0,400))}`);
    await page.keyboard.press('Tab');assert.equal(await palette.evaluate(e=>e===document.activeElement),true,`${name} ${width}x${height} ${theme} ${navLabels} js=${javaScriptEnabled}: ${await page.evaluate(()=>document.activeElement.outerHTML.slice(0,400))}`);
    const targets=await bar.locator('a,button,summary,input').evaluateAll(els=>els.filter(e=>e.checkVisibility()).map(e=>{
     const r=e.getBoundingClientRect();return {name:e.getAttribute('aria-label')||e.textContent.trim(),width:r.width,height:r.height};
    }));
    for(const t of targets){assert.ok(t.name,JSON.stringify(t));assert.ok(t.width>=44&&t.height>=44,`${width} ${name}: ${JSON.stringify(t)}`);}
    for(const target of await page.locator('.period-selector a,.profile-switch,.admin-avatar-link').all()){
     if(await target.isVisible()){const box=await target.boundingBox();assert.ok(box.width>=44&&box.height>=44,'period/profile target');}
    }
    for(const label of await options.locator('.nav-button span').all())assert.equal(await label.isVisible(),true,'complete action labels stay visible');
    const tree=await bar.ariaSnapshot();assert.match(tree,/Main navigation/);assert.match(tree,/Theme and language/);
    const cdp=await context.newCDPSession(page);
    const {nodes}=await cdp.send('Accessibility.getFullAXTree');
    const moreNode=nodes.find(n=>n.name?.value==='More'&&n.properties?.some(p=>p.name==='expanded'));
    assert.ok(moreNode,'native summary has an accessible expansion state');
    assert.equal(moreNode.properties.find(p=>p.name==='expanded').value.value,true);
    await cdp.detach();
    if(width>=640&&(width<1024||navLabels==='icons')&&await bar.locator('.nav-periods-menu').count()){
     const periods=bar.locator('.nav-periods-menu');
     await periods.locator('summary').focus();await page.keyboard.press('Enter');
     assert.equal(await periods.locator('a').first().isVisible(),true,'icon rail retains a native period selector');
     await page.keyboard.press('Enter');await palette.focus();
    }
    await page.keyboard.press('Enter');
    assert.equal(await page.locator('#theme-panel').evaluate(e=>e.matches(':popover-open')),true);
    await page.keyboard.press('Escape');
    // ISSUE-156: role-aware actions live in More; opening it never changes the content offset.
    assert.equal(await page.locator('#nav-admin,.sidebar-secondary,[data-nav-toggle],[name="nav-layout"]').count(),0);
    const actions=bar.locator('.nav-admin-actions');
    if(name==='preview/home'){
     assert.equal(await actions.locator('a,button').count(),0);
     assert.match(await actions.textContent(),/Preview: administration unavailable/);
    }else if(name==='admin/account'){
     assert.equal(await actions.locator('a').count(),3);
     assert.equal(await actions.locator('[aria-current="page"]').getAttribute('href'),'/admin?section=account');
    }else{
     assert.equal(await actions.locator('a').count(),1);
     assert.equal(await actions.locator('a').getAttribute('href'),'/admin');
    }
    const content=page.locator(name==='admin/account'?'.admin-shell':'.app-content');
    const margin=()=>content.evaluate(e=>parseFloat(getComputedStyle(e).marginLeft));
    if(width>=640)assert.equal(await margin(),width>=1024&&navLabels==='labels'?224:58);
    if(javaScriptEnabled){
     // Close More while the theme is open, then Escape must restore a visible theme invoker.
     await palette.click();
     await page.evaluate(()=>document.querySelector('.nav-more').open=false);
     await page.keyboard.press('Escape');
     await page.waitForFunction(()=>document.activeElement?.matches('[popovertarget="theme-panel"]')&&document.activeElement.checkVisibility());
     assert.equal(await bar.locator('.nav-more').evaluate(e=>e.open),true);
     await page.keyboard.press('Escape');
     assert.equal(await summary.evaluate(e=>e===document.activeElement),true);
     await summary.press('Enter');
    }
    // Close disclosure before inspecting footer clearance and the collapsed desktop state.
    await summary.focus();if(await bar.locator('.nav-more').evaluate(e=>e.open))await page.keyboard.press('Enter');
    if(width>=1024&&javaScriptEnabled){
     const other=navLabels==='icons'?'labels':'icons';
     await page.keyboard.press('\\');
     await page.locator(`[data-nav-labels-choice] input[value="${other}"]`).check();
     assert.equal(Math.round((await bar.boundingBox()).width),other==='icons'?58:224);
     assert.equal(await page.evaluate(()=>localStorage.getItem('nav-labels')),null,'no browser-only persistence is introduced');
     await page.locator(`[data-nav-labels-choice] input[value="${navLabels}"]`).check();
     await page.keyboard.press('Escape');
    }
    if(width>=640)assert.equal(await margin(),width>=1024&&navLabels==='labels'?224:58,'closing More preserves the offset');
    await page.evaluate(()=>window.scrollTo(0,document.documentElement.scrollHeight));
    const footer=await page.locator('footer').boundingBox();
    if(width<640){const dock=await bar.boundingBox();assert.ok(footer.y+footer.height<=dock.y+1,`${name} ${width}: footer under dock ${JSON.stringify({footer,dock})}`);}
    entries.push({name,width,height,theme,javaScriptEnabled,navLabels,geometry,targets,accessibleTree:true,keyboard:true,footerClear:true});
   }
   await context.close();
  }
  fs.writeFileSync(path.join(out,'navigation.json'),JSON.stringify({browser:browser.version(),serverPersistence:'pending configuration contract; labels preview is document-only',entries},null,2)+'\n');
  console.log(`PASS ${entries.length} navigation views, native disclosure, keyboard, accessible tree, targets and footer`);
 }finally{await browser.close();server.close();}
})().catch(e=>{console.error(e);server.close();process.exitCode=1;});
