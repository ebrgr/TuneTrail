/* Exercise isolated public flows in a real browser without external requests. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const [go,react]=process.argv.slice(2);
if(![go,react].every(o=>/^http:\/\/127\.0\.0\.1:\d+$/.test(o)))throw Error('two loopback origins required');
(async()=>{const b=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});try{
for(const [mode,origin] of [['go',go],['react',react]]){
 const c=await b.newContext({viewport:{width:1440,height:1000},timezoneId:'America/Sao_Paulo'});await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const p=await c.newPage();p.setDefaultTimeout(12000);
 await p.goto(origin+'/');await p.getByRole('link',{name:/Alpha/}).first().click();await p.waitForURL('**/alpha');await p.getByText('Alpha',{exact:true}).first().waitFor();
 await p.getByRole('button',{name:/Choose listening profile/}).click();await p.locator('a[href="/beta"],a[href^="/beta?"]').first().click();await p.waitForURL('**/beta');
 await p.reload();await p.getByText('Beta',{exact:true}).first().waitFor();
 await p.goBack();await p.waitForURL('**/alpha');await p.goForward();await p.waitForURL('**/beta');
 const second=await c.newPage();await second.goto(origin+'/alpha?period=all_time');await second.getByText('Alpha',{exact:true}).first().waitFor();if(!p.url().endsWith('/beta'))throw Error('other tab changed');
 await second.getByRole('link',{name:/TOP ARTISTS/i}).first().click();await second.waitForURL(/\/alpha\/chart\/top\/artists/);await second.getByText('Artista comum',{exact:false}).first().waitFor();await second.goBack();await second.waitForURL(/\/alpha\?period=all_time/);
 await second.getByRole('link',{name:/LAST PLAYED/i}).first().click();await second.waitForURL(/\/alpha\/listens/);await second.getByText('Faixa comum',{exact:false}).first().waitFor();
 if(mode==='go'){await second.locator('select[name=year]').selectOption('2018');await second.locator('input[name=month]').fill('11');await second.getByRole('button',{name:'Apply',exact:true}).click()}else{await second.locator('select').nth(0).selectOption('2018');await second.waitForURL(/year=2018/);await second.locator('select').nth(1).selectOption('11')};await second.waitForURL(/month=11/);await second.waitForFunction(()=>document.querySelectorAll('tbody tr').length===2);
 await p.getByRole('button',{name:'Search',exact:true}).click();if(mode==='go'){const dlg=p.locator('dialog[open]');await dlg.locator('input[type=search]').fill('Só Beta');await dlg.locator('form').evaluate(el=>el.requestSubmit());await p.waitForURL(/\/beta\/search\?/)}else{await p.getByPlaceholder('Search for an artist, album, or track').fill('Só Beta')};await p.getByText('Só Beta',{exact:true}).first().waitFor();
 await p.goto(origin+'/alpha/track/1');await p.getByText('Faixa comum',{exact:true}).first().waitFor();await p.goto(origin+'/alpha/rewind?year=2018&month=11');await p.getByText('Alpha',{exact:true}).first().waitFor();
 console.log(mode+': selection, switch, reload, history, two tabs, personal search, track and rewind OK');await c.close();
}
const c=await b.newContext({javaScriptEnabled:false});const p=await c.newPage();await p.goto(go+'/alpha');await p.getByRole('link',{name:'Choose listening profile'}).click();await p.waitForURL(go+'/');await p.getByRole('link',{name:/Beta/}).click();await p.waitForURL('**/beta');await p.getByRole('link',{name:'Search',exact:true}).click();await p.waitForURL('**/beta/search');await p.getByRole('searchbox').fill('Só Beta');await p.getByRole('button',{name:'Search',exact:true}).click();await p.waitForURL(/q=/);await p.getByText('Só Beta',{exact:true}).first().waitFor();console.log('go sem JS: troca de perfil e busca pessoal OK');await c.close();
}finally{await b.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
