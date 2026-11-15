/* Exercise ISSUE-140 refinements on a fresh disposable roadmap06 fixture. Optional Koito session stays in memory. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const origin=process.argv[2],out=path.resolve(process.argv[3]);fs.mkdirSync(out,{recursive:true});
(async()=>{const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const checks=[],captures=[],errors=[];
 try {
  const context=await browser.newContext({locale:'en-US',viewport:{width:1440,height:1000},reducedMotion:'reduce'});
  assert.equal((await context.request.post(origin+'/apis/web/v1/login',{data:{username:'validation-admin',password:'validation-password'},headers:{Origin:origin}})).status(),204);
  const page=await context.newPage();page.setDefaultTimeout(10000);page.on('pageerror',e=>errors.push(e.message));
  const go=async(kind='track')=>{await page.goto(origin+'/alpha/'+kind+'/1');await page.locator('[data-media-task]').first().waitFor();};
  const open=async task=>{await page.locator(`[data-media-task][href$="/${task}"]`).click();await page.locator('#task').waitFor();await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));};
  const close=async()=>{await page.keyboard.press('Escape');await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').open);};
  await go();
  const link=page.locator('[data-media-task][href$="/edit"]');
  await link.hover();await page.waitForTimeout(120);
  let tip=await link.evaluate(e=>({opacity:getComputedStyle(e,'::after').opacity,text:getComputedStyle(e,'::after').content,title:e.getAttribute('title'),rect:e.getBoundingClientRect().toJSON()}));
  assert.equal(tip.opacity,'1');assert.equal(tip.title,null);assert(tip.text.includes('Edit item'));
  await page.screenshot({path:path.join(out,'tooltip-hover.png')});
  await page.mouse.move(5,5);await page.locator('.media-heading h1').click();await link.focus();
  assert.equal(await link.getAttribute('title'),null);assert.equal(await link.evaluate(e=>getComputedStyle(e,'::after').opacity),'1');
  await page.screenshot({path:path.join(out,'tooltip-keyboard.png')});
  await page.locator('.media-heading h1').click();assert.equal(await link.getAttribute('title'),'Edit item');
  checks.push('Toolbar tooltips on hover and keyboard focus, accessible label retained, native title suppressed only while active and restored afterwards');
  await open('merge');const input=page.locator('input[name=merge_query]');const url=page.url();
  await input.fill('Long');await page.locator('#merge-results .manage-result').first().waitFor();
  assert.equal(await input.evaluate(e=>e===document.activeElement),true);assert.equal(page.url(),url);
  assert((await page.locator('#merge-results').innerText()).includes('Long fictional'));
  // An earlier request is held while a newer query finishes; its response must not overwrite the list.
  let release;const held=new Promise(resolve=>{release=resolve;});
  await page.route('**/merge?**',async route=>{if(new URL(route.request().url()).searchParams.get('merge_query')==='hold-old'){await held;await route.continue().catch(()=>{});}else await route.continue();});
  const old=page.waitForRequest(request=>request.url().includes('merge_query=hold-old'));
  await input.fill('hold-old');await old;await input.fill('Long');
  await page.waitForResponse(response=>response.url().includes('merge_query=Long')&&response.headers()['x-tunetrail-fragment']==='merge-results');
  release();await page.waitForTimeout(150);assert((await page.locator('#merge-results').innerText()).includes('Long fictional'));
  await page.unroute('**/merge?**');
  await page.locator('#merge-results .manage-result').first().click();await page.locator('#review input[name=plan]').waitFor({state:'attached'});
  assert.equal(page.url(),url);assert.equal(await page.locator('#review form[action$="confirm"]').count(),1);
  await close();checks.push('Debounced merge search preserves cursor and public URL, ignores old responses and opens existing impact review without confirming');
  await open('merge');
  await page.route('**/merge?**',route=>route.abort('failed'));
  await page.locator('input[name=merge_query]').fill('network failure');
  await page.locator('#merge-results .section-error').waitFor();assert.equal(await page.locator('#merge-results .manage-result').count(),0);
  await page.unroute('**/merge?**');await page.locator('input[name=merge_query]').fill('Long');await page.locator('#merge-results .manage-result').first().waitFor();
  assert.equal(await page.locator('[data-task-status]').innerText(),'');
  await close();checks.push('Failed live search clears stale options and recovers on a new query without closing the task');
  await open('edit');const ticket=await page.locator('#task input[name=submission_token]').first().inputValue();
  await page.locator('input[name=artist_query]').fill('Beta');await page.locator('#artist-results form[action$="artist-add"]').first().waitFor();
  assert.equal(await page.locator('#task input[name=submission_token]').first().inputValue(),ticket);
  assert.equal(await page.locator('input[name=artist_query]').evaluate(e=>e===document.activeElement),true);
  await page.locator('#artists .manage-advanced summary').focus();await page.keyboard.press('Tab');
  assert.equal(await page.evaluate(()=>document.activeElement.name),'mbid');
  await close();checks.push('Artist live search shares the existing editor ticket and retains input focus');
  await open('add-listen');assert.equal(await page.locator('input[name=timezone]').inputValue(),'');assert.equal(await page.locator('input[name=timezone]').getAttribute('required'),null);
  await page.locator('input[name=local_time]').fill('2023-02-05T12:34');await page.locator('input[name=timezone]').fill('No/Such_Zone');
  const before=await page.locator('.media-heading').getAttribute('data-listen-count');
  await page.locator('.listen-submit').click();await page.locator('#task .section-error').filter({hasText:'Invalid time zone'}).waitFor();
  assert.equal(await page.locator('.media-heading').getAttribute('data-listen-count'),before);
  assert.equal(await page.locator('input[name=timezone]').inputValue(),'No/Such_Zone');
  await page.screenshot({path:path.join(out,'invalid-timezone.png'),animations:'disabled'});
  await page.locator('input[name=timezone]').fill('');await page.locator('.listen-submit').click();
  await page.waitForFunction(count=>document.querySelector('.media-heading').dataset.listenCount!==count,before);
  await page.waitForFunction(()=>!document.querySelector('#media-task-dialog').hasAttribute('aria-busy'));await close();
  checks.push('Optional timezone starts empty; invalid zone reports a clear error without inserting; corrected blank zone inserts using server-resolved page context');
  // A nested live-results container must not replace the legacy editor's outer enhancement boundary.
  await page.goto(origin+'/alpha/manage/track/1/edit?artist_query=Beta');
  await page.locator('#artist-results form').first().waitFor();
  await page.evaluate(()=>{window.editorIdentity='preserved';});
  const artistPosted=page.waitForResponse(response=>response.request().method()==='POST'&&response.url().endsWith('/artist-add'));
  await page.locator('#artist-results form button').first().click();
  await artistPosted;
  await page.waitForFunction(()=>!document.querySelector('#artists').hasAttribute('aria-busy'));
  assert.equal(await page.evaluate(()=>window.editorIdentity),'preserved');
  checks.push('Artist submission in the full administrative editor retains the document and replaces the outer artists section');
  await go();await open('edit');
  let releasePost;const pendingPost=new Promise(resolve=>{releasePost=resolve;});
  await page.route('**/metadata',async route=>{await pendingPost;await route.continue();});
  await page.locator('#task input[name=mbid]').fill('invalid-uuid');
  const posting=page.waitForRequest(request=>request.method()==='POST'&&request.url().endsWith('/metadata'));
  await page.locator('#task form[action$="metadata"] button').click();await posting;
  assert.equal(await page.evaluate(()=>document.activeElement.id),'media-task-title');
  assert.equal(await page.locator('[data-task-content]').evaluate(e=>e.inert),true);
  releasePost();await page.locator('#task .section-error').waitFor();await page.unroute('**/metadata');await close();
  checks.push('A pending mutation moves focus to its visible dialog title before the form becomes inert');
  // Every shipped palette must use its own surface and foreground; measure actual control contrast.
  const themes=JSON.parse(fs.readFileSync(path.resolve('assets/themes/palettes.json'),'utf8'));
  const contrasts=[];
  for(const theme of Object.keys(themes)) {
   await go('album');await open('edit');await page.evaluate(t=>{document.documentElement.dataset.theme=t},theme);
   const sample=await page.locator('#task').evaluate(root=>{
    const rgb=value=>value.match(/[\d.]+/g).slice(0,3).map(Number);
    const luminance=rgb=>rgb.map(v=>{v/=255;return v<=.04045?v/12.92:((v+.055)/1.055)**2.4}).reduce((a,v,i)=>a+v*[.2126,.7152,.0722][i],0);
    const contrast=(a,b)=>{const x=luminance(rgb(a)),y=luminance(rgb(b));return (Math.max(x,y)+.05)/(Math.min(x,y)+.05)};
    return [...root.querySelectorAll('button,input:not([type=hidden]),select')].filter(e=>e.getClientRects().length).map(e=>{const c=getComputedStyle(e);return {tag:e.tagName,ratio:contrast(c.color,c.backgroundColor),background:c.backgroundColor,color:c.color,border:c.borderColor};});
   });
   assert(sample.every(control=>control.ratio>=4.5),theme+': '+JSON.stringify(sample));contrasts.push({theme,controls:sample});await close();
  }
  checks.push('Outlined buttons, inputs and selects follow every shipped palette with measured text contrast at least 4.5:1, including Pearl');
  // Pair both viewport sizes and the dark/light themes with the observed reference tasks.
  let reference=null,rp=null;
  if(process.env.REFERENCE_USER&&process.env.REFERENCE_PASSWORD){reference=await browser.newContext({locale:'en-US',reducedMotion:'reduce'});assert.equal((await reference.request.post('http://127.0.0.1:4111/apis/web/v1/login',{data:{username:process.env.REFERENCE_USER,password:process.env.REFERENCE_PASSWORD},headers:{Origin:'http://127.0.0.1:4111'}})).status(),204);rp=await reference.newPage();}
  for(const theme of ['yuu','pearl']) for(const [size,viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:390,height:844}})) {
   await page.setViewportSize(viewport);if(rp)await rp.setViewportSize(viewport);
   for(const kind of ['track','album','artist']) for(const task of (kind==='track'?['edit','merge','add-listen']:['edit','merge','image','delete'])) {
    await go(kind);await open(task);await page.evaluate(t=>document.documentElement.dataset.theme=t,theme);
    if(task==='merge'){await page.locator('input[name=merge_query]').fill('a');await page.locator('#merge-results .manage-result').first().waitFor();}
    const overflow=await page.locator('#media-task-dialog').evaluate(e=>e.scrollWidth>e.clientWidth+1);assert.equal(overflow,false,`${kind}/${task}/${size}`);
    if(task==='add-listen'){
     const fields=await page.evaluate(()=>Object.fromEntries(['local_time','timezone'].map(name=>[name,document.querySelector(`[name=${name}]`).getBoundingClientRect().toJSON()]).concat([['button',document.querySelector('.listen-submit').getBoundingClientRect().toJSON()]])));
     assert(fields.timezone.top>fields.local_time.bottom);
     if(size==='desktop'){assert(Math.abs(fields.local_time.bottom-fields.button.bottom)<2);assert(fields.button.left>=fields.local_time.right);}
    }
    const file=`tunetrail-${kind}-${task}-${theme}-${size}.png`;await page.screenshot({path:path.join(out,file),animations:'disabled'});captures.push({file,kind,task,theme,size});await close();
    if(rp&&['edit','merge','add-listen'].includes(task)){
     await rp.goto('http://127.0.0.1:4111/'+kind+'/1');await rp.getByTitle(task==='edit'?'Edit Item':task==='merge'?'Merge Items':'Add Listen',{exact:true}).click();
     const field=task==='edit'?rp.getByPlaceholder('Add a new alias'):task==='merge'?rp.locator('input[placeholder^="Search for "]'):rp.locator('input[type=datetime-local]');await field.waitFor();
     await rp.evaluate(t=>document.documentElement.dataset.theme=t,theme);await rp.waitForTimeout(200);
     const referenceFile=`koito-${kind}-${task}-${theme}-${size}.png`;await rp.screenshot({path:path.join(out,referenceFile),animations:'disabled'});captures.push({file:referenceFile,kind,task,theme,size});
    }
   }
  }
  if(reference)await reference.close();await context.close();assert.deepEqual(errors,[]);
  fs.writeFileSync(path.join(out,'browser-checks.json'),JSON.stringify({checks,contrasts,captures,jsErrors:errors,referenceMutations:'None in this script: only searches, selection of an impact review, theme changes and screenshots'},null,2)+'\n');
  console.log(JSON.stringify({checks,screenshots:captures.length,jsErrors:errors},null,2));
 } finally {await browser.close();}
})().catch(e=>{console.error(e);process.exit(1)});
