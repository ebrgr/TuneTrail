const {chromium}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs=require('fs');
(async()=>{const b=await chromium.launch({executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});const result={externalFonts:'blocked; no screenshots (Orca owns captures)',measurements:[],pages:[]};
try{for(const phase of ['before','after'])for(const width of [390,768,1440]){
const c=await b.newContext({viewport:{width,height:900},locale:'en-US',timezoneId:'UTC'});let external=[];await c.route('**/*',r=>{const u=new URL(r.request().url());if(u.hostname!=='127.0.0.1'){external.push(r.request().url());return r.abort()}return r.continue()});
await c.addInitScript(()=>localStorage.setItem('theme','yuu'));
const p=await c.newPage();await p.goto(`http://127.0.0.1:${phase==='before'?14171:14172}/public/rewind.html`);await p.evaluate(()=>document.fonts.ready);
const d=await c.newCDPSession(p);await d.send('DOM.enable');await d.send('CSS.enable');const {root}=await d.send('DOM.getDocument');
for(const selector of ['.rewind-content h1','.rewind-year-nav h2','.rewind-group h2','.rewind-top h3','.rewind-stats b','.rewind-choice']){
const {nodeId}=await d.send('DOM.querySelector',{nodeId:root.nodeId,selector});if(!nodeId)continue;
result.measurements.push({phase,width,selector,computed:await p.locator(selector).first().evaluate(e=>{const s=getComputedStyle(e),r=e.getBoundingClientRect();return{family:s.fontFamily,size:s.fontSize,weight:s.fontWeight,style:s.fontStyle,transform:s.textTransform,lineHeight:s.lineHeight,width:r.width,height:r.height}}),rendered:(await d.send('CSS.getPlatformFontsForNode',{nodeId})).fonts});}
if(phase==='after'){
for(const route of ['/public/home.html','/preview/home.html','/admin/login.html','/admin/settings.html']){external=[];await p.goto('http://127.0.0.1:14172'+route);await p.evaluate(()=>document.fonts.ready);result.pages.push({route,width,external:[...new Set(external)],fonts:await p.evaluate(()=>performance.getEntriesByType('resource').filter(x=>x.name.includes('.woff2')).map(x=>x.name))});if(external.some(x=>/fonts\.(googleapis|gstatic)\.com/.test(x)))throw Error('external fonts');}
// Probe Portuguese glyphs and both body families explicitly, including real italics.
for(const family of ['Jost','Inter','League Spartan'])for(const style of family==='League Spartan'?['normal']:['normal','italic']){
await p.evaluate(async({family,style})=>{let e=document.getElementById('font-probe');if(!e){e=document.createElement('p');e.id='font-probe';document.body.append(e)}e.textContent='Ação, São João, coração, música, bênção: ÁÀÂÃÉÊÍÓÔÕÚÜÇ áàâãéêíóôõúüç';e.style.cssText=`font-family:"${family}";font-weight:400;font-style:${style};font-size:20px`;await document.fonts.ready},{family,style});
const {root:doc}=await d.send('DOM.getDocument');const {nodeId}=await d.send('DOM.querySelector',{nodeId:doc.nodeId,selector:'#font-probe'});const fonts=(await d.send('CSS.getPlatformFontsForNode',{nodeId})).fonts;result.pages.push({probe:{family,style,width},rendered:fonts});if(!fonts.length||fonts.some(f=>!f.isCustomFont))throw Error('fallback in Portuguese probe');}
}
await c.close();}
fs.writeFileSync('backlog/evidencias/issue-070/medidas-fontes.json',JSON.stringify(result,null,2)+'\n');}finally{await b.close()}})().catch(e=>{console.error(e);process.exit(1)});
