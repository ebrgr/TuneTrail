/* Validate every exported page against the ISSUE-069 layout contract using fictional fixtures. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http');
const root=path.resolve(process.argv[2]),report=path.resolve(process.argv[3]);
const widths=[320,360,390,639,640,641,768,1023,1024,1025,1439,1440,1441];
const types={'.css':'text/css','.js':'text/javascript','.html':'text/html','.svg':'image/svg+xml',
  '.json':'application/json','.png':'image/png','.webp':'image/webp'};
const pages=fs.readdirSync(root,{recursive:true}).filter(p=>p.endsWith('.html'));
// Expose only the generated fixture, never the working tree or operational server.
const server=http.createServer((req,res)=>{
  const file=path.resolve(root,'.'+new URL(req.url,'http://localhost').pathname);
  if(!file.startsWith(root+path.sep)){res.writeHead(403).end();return;}
  fs.readFile(file,(err,data)=>{
    if(err){res.writeHead(404).end();return;}
    res.setHeader('Content-Type',types[path.extname(file)]||'application/octet-stream');res.end(data);
  });
});
// Check rendered geometry and the actual stylesheets the browser accepted.
async function inspect(page){
  return page.evaluate(()=>{
    const allowed=new Set(['(min-width: 40rem)','(min-width: 64rem)','(min-width: 90rem)']);
    const media=[];
    function walk(rules){
      for(const rule of rules){
        if(rule instanceof CSSMediaRule&&/width|height|orientation|aspect-ratio/.test(rule.conditionText)
          &&!allowed.has(rule.conditionText))media.push(rule.conditionText);
        if(rule.cssRules)walk(rule.cssRules);
      }
    }
    for(const sheet of document.styleSheets){try{walk(sheet.cssRules);}catch{/* External font sheets have no layout rules. */}}
    const overflow=document.documentElement.scrollWidth>document.documentElement.clientWidth;
    const outside=[...document.querySelectorAll('body *')].filter(el=>{
      const r=el.getBoundingClientRect(),s=getComputedStyle(el);
      if(s.position==='absolute'||s.position==='fixed'||!r.width||!r.height)return false;
      return r.right>document.documentElement.clientWidth+1;
    }).slice(0,12).map(el=>el.tagName+'.'+el.className);
    return {overflow,media,outside};
  });
}
(async()=>{
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  const origin=`http://127.0.0.1:${server.address().port}`;
  const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const entries=[];
  try{
    for(const theme of ['yuu','pearl'])for(const width of widths){
      const context=await browser.newContext({viewport:{width,height:900},deviceScaleFactor:1,
        locale:'en-US',timezoneId:'UTC',reducedMotion:'reduce'});
      await context.addInitScript(t=>localStorage.setItem('theme',t),theme);
      // Geometry is also checked offline with the existing fallback fonts.
      await context.route('**/*',r=>new URL(r.request().url()).origin===origin?r.continue():r.abort());
      const page=await context.newPage();await page.clock.setFixedTime(new Date('2024-06-01T12:00:00Z'));
      for(const name of pages){
        await page.goto(`${origin}/${name}`,{waitUntil:'load'});await page.evaluate(()=>document.fonts.ready);
        entries.push({page:name,theme,width,...await inspect(page)});
      }
      await context.close();
    }
    const failures=entries.filter(e=>e.overflow||e.media.length);
    fs.writeFileSync(report,JSON.stringify({browser:browser.version(),fonts:'offline fallback',entries,failures},null,2)+'\n');
    console.log(JSON.stringify({views:entries.length,failures},null,2));
    process.exitCode=failures.length?1:0;
  }finally{await browser.close();server.close();}
})().catch(e=>{console.error(e);server.close();process.exitCode=1;});
