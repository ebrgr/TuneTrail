/* Compare computed styles of two exported UI references (ISSUE-068): proves a CSS reorganisation kept the cascade.
   Usage: PLAYWRIGHT_MODULE=/path/node_modules/playwright node scripts/compare_computed_styles.cjs BASE NEW [report.json]
   Exit 0 when every element (and its ::before/::after) has identical computed styles in every page, theme and width. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http');
const pages=['preview/home','public/profiles','public/home','public/artist','public/rewind','admin/login','admin/profiles','admin/account','admin/settings','admin/track','admin/merge'];
const types={'.html':'text/html','.css':'text/css','.js':'application/javascript','.json':'application/json','.svg':'image/svg+xml','.webp':'image/webp','.png':'image/png'};

// serve exposes only the exported fixture bytes on an ephemeral loopback port.
function serve(root){
  const server=http.createServer((req,res)=>{const p=path.resolve(root,'.'+new URL(req.url,'http://localhost').pathname);
    if(!p.startsWith(root+path.sep)){res.writeHead(403).end();return}
    fs.readFile(p,(err,data)=>{if(err){res.writeHead(404).end();return}res.setHeader('Content-Type',types[path.extname(p)]||'application/octet-stream');res.end(data)})});
  return new Promise(r=>server.listen(0,'127.0.0.1',()=>r(server)));
}

// snapshot records every computed property per element path; external requests (fonts) are blocked for determinism.
async function snapshot(browser,root){
  const server=await serve(root),origin=`http://127.0.0.1:${server.address().port}`,out={};
  try{for(const theme of ['yuu','pearl'])for(const width of [360,768,1024,1440,1600]){
    const context=await browser.newContext({viewport:{width,height:900},deviceScaleFactor:1,locale:'en-US',timezoneId:'UTC',reducedMotion:'reduce'});
    await context.addInitScript(t=>localStorage.setItem('theme',t),theme);
    await context.route('**/*',r=>new URL(r.request().url()).origin===origin?r.continue():r.abort());
    const page=await context.newPage();await page.clock.setFixedTime(new Date('2024-06-01T12:00:00Z'));
    for(const name of pages){
      await page.goto(`${origin}/${name}.html`,{waitUntil:'networkidle',timeout:30000});
      out[`${name} ${theme} ${width}`]=await page.evaluate(()=>{
        const rows={},seen={};
        for(const el of document.querySelectorAll('*')){
          let key=el.tagName.toLowerCase()+(el.id?'#'+el.id:'')+(el.classList.length?'.'+[...el.classList].join('.'):'');
          for(let p=el.parentElement;p&&p!==document.documentElement;p=p.parentElement)key=p.tagName.toLowerCase()+(p.classList.length?'.'+p.classList[0]:'')+'>'+key;
          seen[key]=(seen[key]||0)+1;key+='#'+seen[key];
          for(const pseudo of [null,'::before','::after']){
            const cs=getComputedStyle(el,pseudo),style={};
            if(pseudo&&(cs.content==='none'||cs.content==='normal'))continue;
            for(let i=0;i<cs.length;i++)style[cs[i]]=cs.getPropertyValue(cs[i]);
            rows[key+(pseudo||'')]=style;
          }
        }
        return rows;
      });
    }
    await context.close();
  }}finally{server.close()}
  return out;
}

(async()=>{
  const [base,candidate,report]=process.argv.slice(2).map(p=>p&&path.resolve(p));
  if(!base||!candidate)throw Error('usage: compare_computed_styles.cjs BASE NEW [report.json]');
  const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  try{
    const a=await snapshot(browser,base),b=await snapshot(browser,candidate),differences=[];let elements=0;
    for(const view of Object.keys(a)){
      const left=a[view],right=b[view]||{};
      for(const el of new Set([...Object.keys(left),...Object.keys(right)])){
        elements++;
        if(!left[el]||!right[el]){differences.push({view,element:el,missing:left[el]?'new':'base'});continue}
        for(const prop of new Set([...Object.keys(left[el]),...Object.keys(right[el])]))
          if(left[el][prop]!==right[el][prop])differences.push({view,element:el,property:prop,base:left[el][prop],new:right[el][prop]});
      }
    }
    const result={views:Object.keys(a).length,elements,differences};
    if(report)fs.writeFileSync(report,JSON.stringify(result,null,1));
    console.log(JSON.stringify({views:result.views,elements,differences:differences.length,sample:differences.slice(0,20)},null,1));
    process.exitCode=differences.length?2:0;
  }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
