/* Exercise browser admin actions against separate fictional fixture databases. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const [go,react]=process.argv.slice(2);if(![go,react].every(x=>/^http:\/\/127\.0\.0\.1:\d+$/.test(x)))throw Error('loopback only');
(async()=>{const b=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});try{
for(const [mode,origin] of [['go',go],['react',react]]){
 const profileName='Alpha UI '+Date.now();
 const c=await b.newContext({javaScriptEnabled:mode==='react',viewport:{width:1440,height:1000},timezoneId:'America/Sao_Paulo'});await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const p=await c.newPage();p.setDefaultTimeout(12000);
 await p.goto(origin+'/admin?profile=1');
 if(mode==='go'){await p.locator('input[name=username]').fill('validation-admin');await p.locator('input[name=password]').fill('validation-password');await p.getByRole('button',{name:'Sign in'}).click();await p.getByRole('heading',{name:'Profiles'}).waitFor();const form=p.locator('form[action="/admin/profile/1/save"]');await form.locator('[name=display_name]').fill(profileName);await form.getByRole('button',{name:'Save profile'}).click();await p.getByText(profileName,{exact:true}).first().waitFor()}
 else{await p.getByPlaceholder('Username').fill('validation-admin');await p.getByPlaceholder('Password').fill('validation-password');await p.getByRole('button',{name:'Login',exact:true}).click();await p.getByLabel('Display name').fill(profileName);const saved=p.waitForResponse(r=>r.request().method()==='PATCH'&&r.url().includes('/apis/web/v1/profiles/alpha'));await p.getByRole('button',{name:'Save profile',exact:true}).click();if(!(await saved).ok())throw Error('profile save failed')}
 const profile=await (await c.request.get(origin+'/apis/web/v1/profiles/alpha')).json();if(profile.display_name!==profileName)throw Error(mode+' wrong profile name');
 const before=await (await c.request.get(origin+'/apis/web/v1/profiles/beta/stats?period=all_time')).json();if(before.listen_count!==2)throw Error('Beta altered');
 await p.goto(origin+'/alpha/manage/track/1');await p.getByText('Faixa comum',{exact:false}).first().waitFor();
 if(mode==='go'){
  const alias='Fixture alias '+Date.now();const form=p.locator('form[action$="/alias-add"]');await form.locator('input[name=alias]').fill(alias);await form.getByRole('button',{name:'Add alias'}).click();await p.getByText(alias,{exact:false}).first().waitFor();
  const aliases=await (await c.request.get(origin+'/apis/web/v1/profiles/beta/track/1/aliases')).json();if(!JSON.stringify(aliases).includes(alias))throw Error('shared catalog alias absent in beta');
  await p.getByRole('button',{name:'Review deletion'}).click();await p.getByRole('heading',{name:/Review delete/}).waitFor();if(!(await p.locator('body').innerText()).includes('all profiles'))throw Error('missing global impact');await p.getByRole('link',{name:'Cancel'}).click();await p.getByText(alias,{exact:false}).first().waitFor();
 }else{
  const button=p.getByRole('button',{name:'Edit Item'});await button.focus();await button.click();const dialog=p.getByRole('dialog');await dialog.waitFor();await p.keyboard.press('Escape');await dialog.waitFor({state:'hidden'});await p.waitForFunction(()=>document.activeElement?.getAttribute('title')==='Edit Item');
 }
 console.log(mode+': login, profile update, other profile retained; '+(mode==='go'?'no-JS alias and cancel global delete':'modal Escape and focus restoration'));await c.close();
}
// Go public native dialog should restore focus after Escape.
const c=await b.newContext();await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());const p=await c.newPage();await p.goto(go+'/alpha');const opener=p.getByRole('button',{name:'Search',exact:true});await opener.focus();await opener.click();await p.locator('dialog[open]').waitFor();await p.keyboard.press('Escape');await p.locator('dialog[open]').waitFor({state:'hidden'});await p.waitForFunction(()=>document.activeElement?.getAttribute('aria-label')==='Search');console.log('go public: dialog Escape and focus restoration');await c.close();
}finally{await b.close()}})().catch(e=>{console.error(e.stack);process.exit(1)});
