/* Render the exported fictional reference in an isolated headless browser, never an operational server. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http');
const root=path.resolve(process.argv[2]),out=path.resolve(process.argv[3]);
const allPages=['preview/home','public/profiles','public/home','public/artist','public/rewind','admin/login','admin/profiles','admin/account','admin/settings','admin/track','admin/merge'];
const pages=process.env.CAPTURE_PAGES?process.env.CAPTURE_PAGES.split(','):allPages;if(pages.some(p=>!allPages.includes(p)))throw Error('Unknown page');
const previous=process.env.CAPTURE_PAGES?JSON.parse(fs.readFileSync(path.join(out,'manifest.json'),'utf8')):null;
const entries=previous?previous.entries.filter(e=>!pages.includes(e.page)):[],typography=previous&&!pages.includes('public/rewind')?previous.typography:[];
const types={'.html':'text/html','.css':'text/css','.js':'application/javascript','.json':'application/json','.svg':'image/svg+xml','.webp':'image/webp','.png':'image/png'};
// Serve only exported fixture bytes on an ephemeral loopback port.
const server=http.createServer((req,res)=>{const p=path.resolve(root,'.'+new URL(req.url,'http://localhost').pathname);if(!p.startsWith(root+path.sep)){res.writeHead(403).end();return}fs.readFile(p,(err,data)=>{if(err){res.writeHead(404).end();return}res.setHeader('Content-Type',types[path.extname(p)]||'application/octet-stream');res.end(data)})});
(async()=>{fs.mkdirSync(out,{recursive:true});await new Promise(r=>server.listen(0,'127.0.0.1',r));const origin=`http://127.0.0.1:${server.address().port}`;
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
try{for(const theme of ['yuu','pearl'])for(const width of [320,360,390,768,1024,1440]){
 const context=await browser.newContext({viewport:{width,height:900},deviceScaleFactor:1,locale:'en-US',timezoneId:'UTC',reducedMotion:'reduce'});
 await context.addInitScript(t=>localStorage.setItem('theme',t),theme);
 // Permit only the fixture and the current font providers. Record failed fonts rather than silently substituting.
 const failed=[],missing=[];await context.route('**/*',r=>{const u=new URL(r.request().url());return u.origin===origin||['fonts.googleapis.com','fonts.gstatic.com'].includes(u.hostname)?r.continue():r.abort()});
 const page=await context.newPage();page.on('response',r=>{if(r.status()===404&&r.url().startsWith(origin)&&!new URL(r.url()).pathname.startsWith('/image/')&&['image','stylesheet','script'].includes(r.request().resourceType()))missing.push(r.url())});page.on('requestfailed',r=>{if(/fonts\./.test(r.url()))failed.push(r.url())});
 await page.clock.setFixedTime(new Date('2024-06-01T12:00:00Z'));
 for(const name of pages){
  failed.length=0;missing.length=0;const response=await page.goto(`${origin}/${name}.html`,{waitUntil:'networkidle',timeout:30000});if(response.status()!==200)throw Error(name+' '+response.status());
  await page.evaluate(()=>document.fonts.ready);if(missing.length)throw Error('Missing fixture assets '+missing.join(','));
  const actualTheme=await page.locator('html').getAttribute('data-theme');if(actualTheme!==theme)throw Error('theme mismatch '+actualTheme);
  const file=`${name.replace('/','-')}-${theme}-${width}.png`;await page.screenshot({path:path.join(out,file),animations:'disabled',fullPage:name==='public/rewind'});
  entries.push({file,page:name,theme,width,height:900,overflow:await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),failedFonts:[...new Set(failed)]});
  if(name==='public/rewind'&&[390,768,1440].includes(width)){
   const cdp=await context.newCDPSession(page);await cdp.send('DOM.enable');await cdp.send('CSS.enable');const {root:doc}=await cdp.send('DOM.getDocument');
   for(const selector of ['.rewind-content h1','.rewind-year-nav h2','.rewind-group h2','.rewind-top h3','.rewind-stats b','.rewind-choice']){
    const {nodeId}=await cdp.send('DOM.querySelector',{nodeId:doc.nodeId,selector});if(!nodeId)continue;
    const actual=await cdp.send('CSS.getPlatformFontsForNode',{nodeId});
    const computed=await page.locator(selector).first().evaluate(e=>{const s=getComputedStyle(e);return{font:s.fontFamily,size:s.fontSize,weight:s.fontWeight,transform:s.textTransform,lineHeight:s.lineHeight}});
    typography.push({theme,width,selector,computed,renderedFonts:actual.fonts});
   }await cdp.detach();
  }
 }
 await context.close();console.log(theme,width,'done');
}
fs.writeFileSync(path.join(out,'manifest.json'),JSON.stringify({browser:browser.version(),fixtureOnly:true,externalFonts:'allowed; failures recorded',entries,typography},null,2)+'\n');
}finally{await browser.close();server.close()}
})().catch(e=>{console.error(e);server.close();process.exitCode=1});
