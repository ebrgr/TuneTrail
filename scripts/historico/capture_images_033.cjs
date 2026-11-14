/* Capture fictional profiles and covers from isolated legacy/new cache fixtures. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs'),path=require('node:path');
const [baseline,candidate,out]=process.argv.slice(2);
if(![baseline,candidate].every(x=>/^http:\/\/127\.0\.0\.1:\d+$/.test(x))||!out)throw Error('usage: BASELINE CANDIDATE OUTPUT');
(async()=>{fs.mkdirSync(out,{recursive:true});const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});let records=[];try{
for(const [label,base] of [['legacy',baseline],['go',candidate]])for(const [view,viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:390,height:844}})){
const context=await browser.newContext({viewport,locale:'en-US',timezoneId:'America/Sao_Paulo',reducedMotion:'reduce'});await context.addInitScript(()=>localStorage.setItem('theme','yuu'));await context.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const page=await context.newPage();
for(const [name,url] of [['profiles','/'],['home','/alpha']]){await page.goto(base+url,{waitUntil:'domcontentloaded'});await page.evaluate(()=>document.fonts.ready);await page.waitForFunction(()=>[...document.images].filter(i=>i.currentSrc.includes('11111111-1111-4111-8111-111111111111')).every(i=>i.complete),null,{timeout:15000});const images=await page.evaluate(()=>[...document.images].filter(i=>i.currentSrc.includes('11111111-1111-4111-8111-111111111111')).map(i=>({url:new URL(i.currentSrc).pathname,loaded:i.naturalWidth>0})));if(!images.length||images.some(i=>!i.loaded))throw Error(`image failed ${label} ${name}: ${JSON.stringify(images)}`);const filename=`033-${label}-${name}-${view}.png`;await page.screenshot({path:path.join(out,filename),animations:'disabled'});records.push({file:filename,view,route:url,mode:label,images});}
await context.close();}
fs.writeFileSync(path.join(out,'screenshots-033.json'),JSON.stringify({fixture:'fictional SQLite v3, same legacy source',browser:browser.version(),records},null,2)+'\n');console.log(JSON.stringify({captures:records.length}));
}finally{await browser.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
