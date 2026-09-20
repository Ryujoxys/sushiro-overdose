// Optional browser QA. Run against an isolated local preview; APIs are mocked.
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
const {chromium}=createRequire(import.meta.url)('playwright');
const origin=process.env.SUSHIRO_QA_URL||'http://127.0.0.1:39871';
assert.ok(['127.0.0.1','localhost'].includes(new URL(origin).hostname),'local isolated preview required');
const output=process.env.SUSHIRO_QA_OUTPUT||'/tmp/sushiro-browser-qa';
await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.SUSHIRO_QA_BROWSER?{executablePath:process.env.SUSHIRO_QA_BROWSER}:{})});
const context=await browser.newContext({viewport:{width:1280,height:960},locale:'zh-CN'});
const page=await context.newPage(),errors=[],requests=[];
page.on('pageerror',e=>errors.push(e.message));
let selected=[],enabled=false,autostart=false,authenticated=false,hasRecords=false,uncertain=false,ticket=null,includeHistory=true,calledAvailable=true,liveFailure=false;
const stamp=new Date().toISOString();
const stores=[{id:1012,name:'深圳海岸城店',city:'深圳',address:'深圳市南山区海德三道',wait:35},{id:3006,name:'深圳万象天地店',city:'深圳',address:'深圳市南山区深南大道',wait:50}];
await page.route('**/api/**',async route=>{
  const request=route.request(),url=new URL(request.url()),body=request.postDataJSON(),key=url.pathname;
  requests.push({key,method:request.method(),body});
  let data={},status=200;
  if(key==='/api/queue/service'){
    if(body?.action==='start')enabled=true;
    if(body?.action==='pause')enabled=false;
    if(body?.action==='enable_autostart')autostart=true;
    if(body?.action==='disable_autostart')autostart=false;
    data={config:{enabled,interval_minutes:5,store_ids:selected,use_preference_stores:false},state:{running:enabled,store_ids:selected,last_at:hasRecords?stamp:''},background_running:enabled,autostart:{enabled:autostart,supported:true},model:{sample_count:hasRecords?460:0,day_count:hasRecords?8:0},data_path:'/Users/example/.sushiro'};
  }else if(key==='/api/queue/baseline'){selected=body.store_ids;data=body;}
  else if(key==='/api/records'){
    const id=Number(url.searchParams.get('store'))||1012;
    const date=url.searchParams.get('date'),recent=url.searchParams.get('days')==='7';
    const hasCurve=(hasRecords||includeHistory)&&!recent;
    data={source:includeHistory?'mixed':'local',selected_store:id,local_samples:hasRecords?230:0,history_samples:includeHistory?200:0,history:{included:includeHistory,first_at:'2026-06-17T10:20:07+08:00',cutoff_at:'2026-09-06T21:30:17+08:00'},available_stores:hasRecords||includeHistory?stores:[],dates:hasCurve?['2026-09-06','2026-09-05']:[],samples:hasRecords?230:0,days:hasRecords?8:0,latest_at:hasRecords?stamp:'',stores:hasRecords?stores.map(s=>({...s,samples:230,days:8,latest:{collected_at:stamp,store_status:'OPEN',wait_minutes:s.wait}})):[],called_local_samples:hasRecords&&calledAvailable?230:0,called_history_samples:includeHistory&&calledAvailable?200:0,called_days:hasCurve&&calledAvailable?8:0,called_points:hasCurve&&id&&calledAvailable?Array.from({length:12},(_,i)=>({time:`${String(16+Math.floor(i/2)).padStart(2,'0')}:${i%2?'30':'00'}`,median:200+i*100,lower:100+i*100,upper:400+i*100,samples:date?1:20,days:date?1:8})):[],points:hasCurve&&id?[20,25,35,48,58,62,75,70,60,42,30,25].map((median,i)=>({time:`${String(16+Math.floor(i/2)).padStart(2,'0')}:${i%2?'30':'00'}`,median,upper:median+15,samples:date?1:20,days:date?1:8})):[]};
    data.export_record_count=hasRecords?230:0;data.total_record_count=hasRecords?460:0;
  }else if(key==='/api/records/settings'){includeHistory=body.include_history;data={include_history:includeHistory};}
  else if(key==='/api/status')data={version:'preview',has_config:authenticated,platform:'windows',engine:{status:'idle'},auth_health:{status:authenticated?'ok':'unknown'}};
  else if(key==='/api/queue/stores')data={stores};
  else if(key==='/api/queue/live'){data={store_id:'1012',store_name:stores[0].name,store_status:'OPEN',online_open:true,called_no:123,wait_groups:18,server_wait_minutes:35,observed_at:stamp};if(liveFailure){status=502;data={error:'网络未连接'};}}
  else if(key==='/api/queue/plan')data={meal_range:{early:'18:00',late:'18:30'}};
  else if(key==='/api/auth/import'){authenticated=true;data={saved:true};}
  else if(key==='/api/auth/verify')data={valid:true,message:'验证通过'};
  else if(key==='/api/queue/ticket'){
    if(uncertain){status=502;data={error:'请求结果未确认'};}
    else{ticket={number:'A148',store_id:'1012'};data={ticket,cancel_token:'mock-confirmation'};}
  }else if(key==='/api/queue/ticket/status')data={ticket,cancel_token:'mock-confirmation'};
  else if(key==='/api/queue/ticket/cancel'){ticket=null;data={ok:true};}
  else if(key==='/api/queue/advisor')data={eta:{estimated_called_at_range:{early:'18:00',late:'18:30'},wait_minutes_range:{low:35,high:65}}};
  else if(key==='/api/records/export')return route.fulfill({contentType:'application/x-ndjson',body:'{"store_id":1012}\n'});
  else throw new Error('Unexpected API call: '+key);
  await route.fulfill({status,contentType:'application/json',body:JSON.stringify(data)});
});
const shot=async name=>page.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'});
const click=action=>page.locator(`[data-action="${action}"]`).first().click();
const noOverflow=async()=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'horizontal overflow');
try{
  await page.goto(origin);await page.getByText('先选常去的几家店').waitFor();
  assert.equal(requests.some(r=>r.method==='POST'),false);
  await page.locator('#record-chart svg').waitFor();
  assert.equal(await page.locator('#export-records').getAttribute('aria-disabled'),'true');
  const storeSearch=page.getByRole('combobox',{name:'门店',exact:true});
  await storeSearch.fill('万象');await page.getByRole('option',{name:/万象天地/}).waitFor();
  await storeSearch.press('ArrowDown');await storeSearch.press('Enter');
  await page.waitForFunction(()=>document.getElementById('analysis-store').value==='3006'&&document.getElementById('record-chart').getAttribute('aria-busy')!=='true');
  await storeSearch.fill('不存在');await page.getByText('没有找到，换个关键词试试。',{exact:true}).waitFor();
  await storeSearch.press('Escape');assert.match(await storeSearch.inputValue(),/万象天地/);
  await storeSearch.fill('深圳');assert.equal(await page.locator('#analysis-store-options [role="option"]').count(),2);await shot('store-search');await storeSearch.press('Escape');
  assert.equal(requests.some(r=>r.method==='POST'),false,'chart store search wrote data');
  assert.match(await page.locator('#history-meta').innerText(),/2026-09-06/);
  await page.getByRole('radio',{name:'叫到几号',exact:true}).check();
  await page.locator('[data-record-point="18:00"]').hover();
  assert.match(await page.locator('#analysis-readout').innerText(),/600 号/);
  await page.getByRole('radio',{name:'等多久',exact:true}).check();
  assert.match(await page.locator('#analysis-readout').innerText(),/600 号/);
  await page.getByRole('radio',{name:'叫到几号',exact:true}).check();
  await shot('history-first-open');
  calledAvailable=false;await click('refresh');await page.getByText('这些时段还没有叫号记录',{exact:true}).waitFor();
  await page.getByRole('radio',{name:'等多久',exact:true}).check();await page.locator('#record-chart svg').waitFor();
  assert.match(await page.locator('#analysis-readout').innerText(),/暂无记录/);
  calledAvailable=true;await click('refresh');await page.getByRole('radio',{name:'叫到几号',exact:true}).check();await page.locator('#record-chart svg').waitFor();
  await page.locator('#analysis-date').selectOption('2026-09-05');await page.getByText('2026-09-05 · 每半小时一条记录',{exact:true}).waitFor();
  await page.locator('#analysis-days').selectOption('7');await page.getByText('这个范围没有记录',{exact:true}).waitFor();
  await page.locator('#analysis-days').selectOption('all');await page.locator('#record-chart svg').waitFor();
  await page.locator('#include-history').uncheck();await page.getByText('还没有符合条件的本机记录',{exact:true}).waitFor();
  await page.reload();await page.getByText('还没有符合条件的本机记录',{exact:true}).waitFor();
  assert.equal(await page.locator('#include-history').isChecked(),false,'opt-out lost on reload');
  assert.equal(requests.filter(r=>r.method==='POST').length,1,'view issued unrelated writes');
  await shot('history-off-empty');
  await click('pick-records');await page.locator('input[value="1012"]').check();await page.locator('input[value="3006"]').check();await click('save-stores');
  await page.getByText('门店已保存').waitFor();assert.equal(enabled,false,'selecting stores started collection');
  await click('toggle-recording');await page.getByRole('button',{name:'暂停记录'}).waitFor();assert.equal(autostart,false,'start enabled autostart implicitly');
  assert.equal(await page.locator('#analysis-store').inputValue(),'1012','saved store did not become chart selection');
  hasRecords=true;await click('refresh');await page.locator('#record-chart svg').waitFor();
  assert.equal(await page.locator('#export-records').getAttribute('aria-disabled'),'false');
  assert.equal(await page.locator('#backup-records').getAttribute('href'),'/api/records/export?days=all');
  assert.equal(await page.locator('#include-history').isChecked(),false,'local samples reenabled history');
  await page.locator('#include-history').check();await page.locator('#record-chart svg').waitFor();
  await noOverflow();await shot('records-desktop');
  await page.setViewportSize({width:390,height:844});await page.waitForTimeout(200);await noOverflow();await shot('records-mobile');
  await page.setViewportSize({width:320,height:740});await noOverflow();
  await page.setViewportSize({width:1280,height:960});
  await page.getByRole('link',{name:'手动取号',exact:true}).click();await page.getByText('官方等待 / 分钟').waitFor();await shot('manual-ticket');
  liveFailure=true;await click('refresh-live');await page.getByText('未更新：网络未连接',{exact:true}).waitFor();assert.equal(await page.locator('#take-ticket').isDisabled(),true);await page.getByText('上次叫到',{exact:true}).waitFor();await shot('live-offline');
  liveFailure=false;await page.getByRole('link',{name:'我的记录',exact:true}).click();await page.getByRole('link',{name:'手动取号',exact:true}).click();await page.getByText('当前叫到',{exact:true}).waitFor();
  await click('review-ticket');await page.locator('#ticket-dialog').waitFor();await click('confirm-ticket');await page.locator('#auth-dialog').waitFor();
  assert.equal(await page.locator('#auth-method').inputValue(),'mobile');
  assert.equal(requests.filter(r=>r.key==='/api/queue/ticket').length,0);
  assert.equal(requests.filter(r=>r.key==='/api/mobile-auth/start'||r.key==='/api/engine/capture').length,0);
  await shot('auth-needed');await page.setViewportSize({width:390,height:844});await noOverflow();await shot('auth-mobile');await page.setViewportSize({width:1280,height:960});await page.locator('#auth-import summary').click();await page.locator('#auth-text').fill('mock credentials');await click('import-auth');
  await page.locator('#auth-continue').waitFor();await click('auth-continue');await page.locator('#ticket-dialog').waitFor();
  assert.equal(requests.filter(r=>r.key==='/api/queue/ticket').length,0,'auth auto-submitted ticket');
  await click('confirm-ticket');await page.getByText('A148',{exact:true}).waitFor();assert.equal(requests.filter(r=>r.key==='/api/queue/ticket').length,1);await shot('ticket-result');
  await click('cancel-ticket');await page.locator('#confirm-dialog [data-answer="yes"]').click();await page.getByText('已取消排队号',{exact:true}).waitFor();
  assert.equal(requests.find(r=>r.key==='/api/queue/ticket/cancel').body.cancel_token,'mock-confirmation');
  uncertain=true;await click('review-ticket');await click('confirm-ticket');await page.getByText('操作结果还未确认',{exact:true}).waitFor();
  await click('refresh-live');await page.getByText('官方等待 / 分钟').waitFor();assert.equal(await page.locator('#take-ticket').isDisabled(),true);
  await shot('ticket-uncertain');await click('query-ticket');await page.getByText('当前没有查到排队号',{exact:true}).waitFor();
  await page.getByRole('link',{name:'设置',exact:true}).click();await noOverflow();await shot('settings');
  assert.deepEqual(errors,[]);
  console.log('Browser QA passed: desktop/mobile, recording start, no implicit autostart, offline history, date filters, persisted opt-out, chart, auth confirmation, manual ticket, cancel, unknown result. Screenshots: '+output);
}finally{await browser.close();}
