/* Capture paired fictional fixtures; all external requests are blocked. */
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const path = require('node:path');
const [go, react, output] = process.argv.slice(2);
if (![go,react,output].every(Boolean)) throw Error('usage: node capture_public_023.cjs GO_ORIGIN REACT_ORIGIN OUTPUT');
for (const origin of [go,react]) if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(origin)) throw Error('only loopback origins allowed');
const entries=[];
const cases=[
  ['profiles','/','Choose a listening profile',true],
  ['home','/alpha?period=all_time','Alpha',true],
  ['track','/alpha/track/1','Faixa comum',true],
  ['rewind','/alpha/rewind?year=2018&month=11','Alpha',true],
  ['empty','/empty?period=all_time','Empty',true],
  ['artists','/alpha/chart/top/artists?period=all_time','Artista comum',false],
  ['albums','/alpha/chart/top/albums?period=all_time','Álbum comum',false],
  ['history','/alpha/listens?period=all_time','Faixa comum',false],
  ['artist','/alpha/artist/1','Artista comum',false],
  ['album','/alpha/album/1','Álbum comum',false],
];
(async()=>{
 fs.mkdirSync(output,{recursive:true});
 const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 try {
  for (const theme of ['yuu','pearl','black']) for(const [view,viewport] of Object.entries({desktop:{width:1440,height:1000},mobile:{width:390,height:844}})) for(const [mode,origin] of [['go',go],['react',react]]) {
   const context=await browser.newContext({viewport,deviceScaleFactor:1,timezoneId:'America/Sao_Paulo',locale:'en-US',reducedMotion:'reduce'});
   await context.addInitScript(name=>localStorage.setItem('theme',name),theme);
   await context.route('**/*',route=>new URL(route.request().url()).hostname==='127.0.0.1'?route.continue():route.abort());
   const page=await context.newPage();page.setDefaultTimeout(12000);
   for(const [name,url,ready,representative] of cases){
    if(theme!=='yuu'&&!representative)continue;
    await page.goto(origin+url,{waitUntil:'domcontentloaded'});
    await page.getByText(ready,{exact:false}).first().waitFor({state:'visible'});
    await page.evaluate(()=>document.fonts.ready);
    await page.waitForFunction(()=>[...document.images].filter(x=>x.loading!=='lazy' && x.getBoundingClientRect().top<innerHeight).every(x=>x.complete));
    const state=await page.evaluate(()=>({theme:document.documentElement.dataset.theme,fonts:document.fonts.status,fontFaces:[...document.fonts].map(f=>({family:f.family,status:f.status})),overflow:document.documentElement.scrollWidth>innerWidth+1,images:[...document.images].filter(x=>x.getBoundingClientRect().top<innerHeight).map(x=>({complete:x.complete,loaded:x.naturalWidth>0}))}));
    const file=`${mode}-${name}-${theme}-${view}.jpg`;
    await page.screenshot({path:path.join(output,file),type:'jpeg',quality:72,animations:'disabled'});
    entries.push({file,mode,case:name,route:url,theme,viewport,scale:1,timezone:'America/Sao_Paulo',region:'topo',network:'somente loopback; fontes remotas bloqueadas',...state});
   }
   await context.close();
  }
  fs.writeFileSync(path.join(output,'../screenshots-023.json'),JSON.stringify({date:'2026-11-05',fixture:'roadmap06 v1; alpha 7 beta 2 empty 0',browser:browser.version(),notes:'Estado atual da árvore local; relógio do servidor real; imagens locais; fontes externas bloqueadas.',entries},null,2)+'\n');
  console.log(JSON.stringify({captures:entries.length,overflow:entries.filter(x=>x.overflow).map(x=>x.file)}));
 } finally {await browser.close()}
})().catch(e=>{console.error(e.stack);process.exit(1)});
