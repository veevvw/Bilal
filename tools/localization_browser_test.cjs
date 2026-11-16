const http=require('http'),fs=require('fs'),path=require('path'),assert=require('assert/strict');
const {chromium}=require('playwright');
const root=path.resolve('internal/webapp/static');
fs.mkdirSync('.tmp',{recursive:true});
const server=http.createServer((req,res)=>{const pathname=new URL(req.url,'http://localhost').pathname;const file=path.join(root,pathname.startsWith('/static/')?pathname.slice(8):pathname==='/room'?'room.html':pathname==='/privacy'?'privacy.html':'home.html');try{const body=fs.readFileSync(file);res.setHeader('Content-Type',file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':file.endsWith('.svg')?'image/svg+xml':'text/html; charset=utf-8');res.end(body)}catch{res.statusCode=404;res.end()}});
(async()=>{await new Promise(r=>server.listen(0,'127.0.0.1',r));const origin=`http://127.0.0.1:${server.address().port}`;const browser=await chromium.launch({headless:true,channel:'chrome'});const page=await browser.newPage({viewport:{width:390,height:844}});const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.route(/https:\/\//,route=>route.abort());
try{
 for(const url of ['/','/privacy','/room']){
  await page.goto(origin+url);await page.waitForLoadState('networkidle');assert.equal(await page.getAttribute('html','lang'),'ar');assert.equal(await page.getAttribute('html','dir'),'rtl');
  await Promise.all([page.waitForNavigation({waitUntil:'networkidle',timeout:10000}),page.selectOption('#ui-language','en')]);assert.equal(await page.getAttribute('html','lang'),'en');assert.equal(await page.getAttribute('html','dir'),'ltr');
  await page.reload();assert.equal(await page.getAttribute('html','lang'),'en');await Promise.all([page.waitForNavigation({waitUntil:'networkidle',timeout:10000}),page.selectOption('#ui-language','ar')]);
  console.log('PASS language persistence and direction',url);
 }
 await page.route('**/api/**',route=>route.fulfill({json:{albums:[],playlists:[],mixes:[],results:[]}}));
 await page.routeWebSocket('**/ws?**',ws=>ws.onMessage(raw=>{
  const message=JSON.parse(raw);if(message.type!=='join')return;
  ws.send(JSON.stringify({event:'user_info',data:{userId:42,isAdmin:true,isAuth:true,canControl:true,canPlay:true,allowsWriteToPM:true}}));
  ws.send(JSON.stringify({event:'room_state',data:{roomId:-42,revision:1,track:null,playback:{status:'stopped',position:0,serverTime:Date.now()},queue:[],listeners:[{userId:42,firstName:'Test User',isAdmin:true}],loop:0,autoplay:false,chatEnabled:true,chatCooldown:0,chatSettings:{language:'ar',adminMode:'everyone',autoDeleteMinutes:5},vc:{participants:[],muteNewParticipants:false}}}));
 }));
 await page.evaluate(()=>sessionStorage.setItem('synctune:room-session',JSON.stringify({initData:'user='+encodeURIComponent(JSON.stringify({id:42,first_name:'Test User'})),roomId:'-42'})));
 await page.goto(origin+'/room');await page.waitForTimeout(800);
 await page.locator('#session-primary').click();await page.waitForTimeout(200);assert.equal(await page.locator('#app').isVisible(),true);await page.screenshot({path:'.tmp/arabic-connected.png',fullPage:true});
 console.log('PASS connected room renders in Arabic');
 assert.equal(await page.locator('.language-switch').isVisible(),false);
 await page.locator('[data-nav="profile"]:visible').first().click();
 await page.locator('[data-action="open-settings"][data-section="account"]:visible').click();
 await Promise.all([page.waitForNavigation({waitUntil:'networkidle'}),page.selectOption('#ui-language-settings','en')]);
 assert.equal(await page.getAttribute('html','lang'),'en');
 await Promise.all([page.waitForNavigation({waitUntil:'networkidle'}),page.selectOption('#ui-language','ar')]);
 console.log('PASS account language selector');
 const test=await page.evaluate(async()=>{const {t,errorText}=await import('/static/js/i18n.js');return {home:t('Home'),dynamic:t('Added by {{0}}','English Song / عنوان عربي'),error:errorText('Sorry, song exceeds max duration of 20 minutes.'),unknown:errorText('Unknown technical error')}});
 assert.equal(test.home,'الرئيسية');assert.ok(test.dynamic.includes('English Song / عنوان عربي'));assert.ok(test.error.includes('20'));assert.ok(!/[A-Za-z]/.test(test.error));assert.ok(!/[A-Za-z]/.test(test.unknown));console.log('PASS translations preserve user values and format errors');
 assert.deepEqual(errors,[]);console.log('PASS no browser runtime errors');
 await page.screenshot({path:'.tmp/arabic-room.png',fullPage:true});
}finally{await browser.close();server.close()}})().catch(e=>{console.error(e);server.close();process.exitCode=1});

