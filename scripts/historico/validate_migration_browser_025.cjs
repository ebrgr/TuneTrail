/* Check migrated profile navigation and admin interaction in both modes. */
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const [go,react]=process.argv.slice(2);
if(![go,react].every(x=>/^http:\/\/127\.0\.0\.1:\d+$/.test(x)))throw Error('loopback only');
(async()=>{
 const b=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 try {
  for(const [mode,origin] of [['go',go],['react',react]]){
   const c=await b.newContext({timezoneId:'UTC'});
   await c.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());
   const p=await c.newPage();p.setDefaultTimeout(12000);
   await p.goto(origin+'/');await p.getByRole('link',{name:/validation-admin/i}).first().click();
   await p.waitForURL('**/default');await p.getByText('validation-admin',{exact:true}).first().waitFor();
   await p.goto(origin+'/listens');await p.waitForURL('**/default/listens');await p.getByText('Legacy track',{exact:false}).first().waitFor();
   if(mode==='react'){
    // Observe either a working editor or the known error in a separate tab.
    const helper=await c.newPage();
    const observed=Promise.race([
     helper.waitForEvent('console',{predicate:m=>m.type()==='error'&&m.text().includes('This screen requires a listening profile'),timeout:12000}).then(()=> 'context-error'),
     helper.locator('#custom-theme-input').waitFor({state:'visible',timeout:12000}).then(()=> 'editor-visible')
    ]);
    await helper.goto(origin+'/theme-helper',{waitUntil:'domcontentloaded'});const helperStatus=await observed;await helper.close();
    console.log('THEME_HELPER='+helperStatus);
    await p.goto(origin+'/missing-profile-025');await p.getByText('Profile or page not found',{exact:false}).first().waitFor();
   }
   await p.goto(origin+'/admin?profile=1');
   if(mode==='go'){
    await p.locator('[name=username]').fill('validation-admin');await p.locator('[name=password]').fill('validation-password');
    await p.getByRole('button',{name:'Sign in'}).click();await p.getByRole('heading',{name:'Profiles'}).waitFor();
    const f=p.locator('form[action="/admin/profile/1/save"]');await f.locator('[name=display_name]').fill('Legacy UI Go');
    await f.getByRole('button',{name:'Save profile'}).click();await p.getByText('Legacy UI Go',{exact:true}).first().waitFor();
   }else{
    await p.getByPlaceholder('Username').fill('validation-admin');await p.getByPlaceholder('Password').fill('validation-password');
    await p.getByRole('button',{name:'Login',exact:true}).click();await p.getByText('Profile details',{exact:false}).first().waitFor();
    await p.getByLabel('Display name').fill('Legacy UI React');
    const saved=p.waitForResponse(r=>r.request().method()==='PATCH'&&r.url().endsWith('/profiles/default'));
    await p.getByRole('button',{name:'Save profile',exact:true}).click();if(!(await saved).ok())throw Error('profile update failed');
   }
   const profile=await(await c.request.get(origin+'/apis/web/v1/profiles/default')).json();
   if(profile.display_name!==(mode==='go'?'Legacy UI Go':'Legacy UI React'))throw Error('profile edit was not persisted');
   const stats=await(await c.request.get(origin+'/apis/web/v1/profiles/default/stats?period=all_time')).json();
   if(stats.listen_count!==1)throw Error('legacy listen changed');
   console.log(mode+': migrated navigation, legacy redirect, UI login and persisted profile edit OK');await c.close();
  }
 } finally {await b.close()}
})().catch(e=>{console.error(e.stack);process.exit(1)});
