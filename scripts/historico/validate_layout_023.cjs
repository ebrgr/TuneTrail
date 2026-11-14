/* Sample remaining palettes and detect document overflow at planned breakpoints. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const [go,react]=process.argv.slice(2);if(![go,react].every(o=>/^http:\/\/127\.0\.0\.1:\d+$/.test(o)))throw Error('loopback only');
(async()=>{const b=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});const report=[];try{
for(const [mode,origin] of [['go',go],['react',react]])for(const width of [320,360,390,639,640,641,767,768,959,960,1023,1024,1440]){
 const c=await b.newContext({viewport:{width,height:844},reducedMotion:'reduce'});await c.addInitScript(()=>localStorage.setItem('theme','yuu'));await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const p=await c.newPage();await p.goto(origin+'/alpha?period=all_time');await p.getByText('Alpha',{exact:true}).first().waitFor();for(const route of ['/alpha?period=all_time','/alpha/track/1']){if(!p.url().endsWith(route)){await p.goto(origin+route);await p.getByText('Faixa comum',{exact:false}).first().waitFor()};const v=await p.evaluate(()=>({scroll:document.documentElement.scrollWidth,inner:innerWidth}));report.push({mode,width,route,...v,overflow:v.scroll>v.inner+1})}await c.close();
}
for(const [mode,origin] of [['go',go],['react',react]])for(const theme of ['varia','midnight','catppuccin','autumn','wine','rosebud','urim','match','lemon']){
 const c=await b.newContext();await c.addInitScript(t=>localStorage.setItem('theme',t),theme);await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const p=await c.newPage();await p.goto(origin+'/alpha?period=all_time');await p.getByText('Alpha',{exact:true}).first().waitFor();const applied=await p.evaluate(()=>document.documentElement.dataset.theme);if(applied!==theme)throw Error(`${mode} theme ${theme} applied ${applied}`);await c.close();
}
console.log(JSON.stringify({breakpointChecks:report.length,overflow:report.filter(x=>x.overflow),additionalThemesPerMode:9}));
}finally{await b.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
