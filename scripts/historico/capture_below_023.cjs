/* Capture matching scrolled regions once content and images are ready. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const fs=require('node:fs');
const [go,react]=process.argv.slice(2);
if(![go,react].every(o=>/^http:\/\/127\.0\.0\.1:\d+$/.test(o)))throw Error('loopback only');
const dir='backlog/evidencias/roadmap-06/screenshots-023';
const manifest='backlog/evidencias/roadmap-06/screenshots-023.json';
(async()=>{
 const b=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const data=JSON.parse(fs.readFileSync(manifest));data.entries=data.entries.filter(x=>x.region!=='meio');
 try {
  for(const [mode,origin] of [['go',go],['react',react]])for(const [name,url,ready] of [['track','/alpha/track/1','Faixa comum'],['artist','/alpha/artist/1','Artista comum'],['rewind','/alpha/rewind?year=2018&month=11','Alpha']])for(const [view,width,height] of [['desktop',1440,1000],['mobile',390,844]]){
   const c=await b.newContext({viewport:{width,height},deviceScaleFactor:1,timezoneId:'America/Sao_Paulo',locale:'en-US',reducedMotion:'reduce'});
   await c.addInitScript(()=>localStorage.setItem('theme','yuu'));
   await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());
   const p=await c.newPage();await p.goto(origin+url);await p.getByText(ready,{exact:false}).first().waitFor();await p.evaluate(()=>document.fonts.ready);
   await p.evaluate(()=>window.scrollTo(0,document.documentElement.scrollHeight/2));
   await p.waitForFunction(()=>[...document.images].filter(x=>{const r=x.getBoundingClientRect();return r.bottom>0&&r.top<innerHeight}).every(x=>x.complete));
   const state=await p.evaluate(()=>({scrollY,fonts:document.fonts.status,fontFaces:[...document.fonts].map(f=>({family:f.family,status:f.status})),images:[...document.images].filter(x=>{const r=x.getBoundingClientRect();return r.bottom>0&&r.top<innerHeight}).map(x=>({complete:x.complete,loaded:x.naturalWidth>0}))}));
   const file=`${mode}-${name}-below-yuu-${view}.jpg`;await p.screenshot({path:dir+'/'+file,type:'jpeg',quality:72,animations:'disabled'});
   data.entries.push({file,mode,case:name,route:url,theme:'yuu',viewport:{width,height},scale:1,timezone:'America/Sao_Paulo',region:'meio',network:'somente loopback; fontes remotas bloqueadas',...state});await c.close();
  }
 } finally {await b.close()}
 fs.writeFileSync(manifest,JSON.stringify(data,null,2)+'\n');console.log('12 capturas abaixo da dobra');
})().catch(e=>{console.error(e.stack);process.exit(1)});
