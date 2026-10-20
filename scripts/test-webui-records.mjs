import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const root=new URL('../internal/app/webui/',import.meta.url);
const source=['app.js','record_chart.js','store_select.js','auth_ticket.js'].map(name=>fs.readFileSync(new URL(name,root),'utf8')).join('\n');
const page=fs.readFileSync(new URL('index.html',root),'utf8');
// A mocked response may carry an HTTP status; anything else answers 200.
const HTTP_STATUS=Symbol('status');
const httpError=(status,body)=>({[HTTP_STATUS]:status,...body});
const tick=()=>new Promise(resolve=>setImmediate(resolve));
// Real advisor ETA shape: RFC3339 times for machines, HH:MM fields for the page.
const liveEta={target_no:128,remaining_groups:16,estimated_called_at:'2026-10-20T18:30:00+08:00',estimated_called_at_hhmm:'18:30',estimated_called_at_range:{early:'2026-10-20T18:20:00+08:00',late:'2026-10-20T18:40:00+08:00'},range_hhmm:{early:'18:20',late:'18:40'},wait_minutes_range:{low:30,high:50},source:'realtime',risk:'medium'};
{
  const bindings={app:{DesktopBridge:{}}};
  const browserGlobal=vm.createContext({go:bindings});
  browserGlobal.window=browserGlobal;
  vm.runInContext(source,browserGlobal);
  assert.equal(browserGlobal.go,bindings,'page globals overwrote native Go bindings');
}
function fixture() {
  const nodes=new Map(),calls=[];
  function node(id) {
    if(!nodes.has(id))nodes.set(id,{id,value:'',hidden:false,disabled:false,open:false,textContent:'',innerHTML:'',options:[],dataset:{},style:{setProperty(){}},attributes:{},listeners:{},classList:{toggle(){}},
      showModal(){this.open=true;},close(){this.open=false;},focus(){},select(){},scrollIntoView(){},getBoundingClientRect(){return {top:100,bottom:146};},replaceChildren(){},append(){},add(option){this.options.push(option);},querySelector(){return null;},querySelectorAll(){return[];},contains(){return false;},closest(){return null;},addEventListener(name,fn){this.listeners[name]=fn;},setAttribute(key,value){this.attributes[key]=value;},getAttribute(key){return this.attributes[key];},removeAttribute(key){delete this.attributes[key];}});
    return nodes.get(id);
  }
  let respond=async(path)=> {
    if(path==='/api/status')return {has_config:true,platform:'darwin',version:'test',engine:{status:'idle'},auth_health:{status:'ok'}};
    if(path==='/api/queue/ticket')return {ticket:{number:'A128',store_id:'1012'},cancel_token:'ticket-confirmation'};
    if(path==='/api/queue/ticket/status')return {ticket:null};
    if(path==='/api/queue/service')return {config:{store_ids:['1012'],enabled:false,interval_minutes:5},state:{store_ids:['1012']},autostart:{},model:{}};
    if(path.startsWith('/api/records?'))return {stores:[],available_stores:[{id:1012,name:'测试店'}],selected_store:1012,points:[],called_points:[],export_record_count:0,total_record_count:0};
    if(path.startsWith('/api/queue/live'))return {store_id:'1012',store_name:'测试店',store_status:'OPEN',online_open:true,called_no:112,wait_groups:15,server_wait_minutes:30};
    if(path.startsWith('/api/queue/plan'))return {meal_range:{early:'18:20',late:'18:40'}};
    if(path.startsWith('/api/queue/advisor'))return {store_id:'1012',current:{called_no:112},source:'live',eta:liveEta};
    if(path==='/api/live-session')return {state:'none',connected:false,remaining_seconds:0};
    return {};
  };
  const ctx=vm.createContext({console,URL,URLSearchParams,AbortController,Option:function(text,value){this.text=text;this.value=value;},
    setTimeout(){return 1;},clearTimeout(){},setInterval(){},clearInterval(){},
    location:{hash:'#queue'},history:{replaceState(){}},window:{innerHeight:800,addEventListener(){}},
    document:{getElementById:node,querySelector(selector){return selector.includes('meta')?{content:'test-token'}:null;},querySelectorAll(){return[];},addEventListener(){},activeElement:null,createElement(tag){return node('created-'+tag);}},
    fetch:async(path,options)=>{
      calls.push({path,options});
      let data;try{data=await respond(path,options);}catch(error){return {ok:true,status:200,json:()=>Promise.reject(error)};}
      const status=data?.[HTTP_STATUS]||200;
      return {ok:status>=200&&status<300,status,json:async()=>data};
    }
  });
  vm.runInContext(source,ctx);
  node('ticket-adult').value='2';node('ticket-child').value='0';node('ticket-table').value='T';node('meal-mode').value='now';
  node('analysis-store-popup').hidden=true;node('analysis-days').value='all';node('analysis-date-type').value='all';
  vm.runInContext("state.selectedStore='1012';state.live={online_open:true};",ctx);
  return {ctx,node,calls,run:code=>vm.runInContext(code,ctx),setRespond:fn=>{respond=fn;}};
}

{
 const f=fixture();
 assert.equal(f.run("escapeHTML('<img onerror=\"x\">')"),'&lt;img onerror=&quot;x&quot;&gt;');
 assert.throws(()=>f.run("ticketOptions('',0,'T')"));
 assert.throws(()=>f.run("ticketOptions(0,0,'T')"));
 assert.throws(()=>f.run("ticketOptions(2.5,0,'T')"));
 assert.equal(f.run("ticketOptions(2,0,'T').adult"),2);
 assert.equal(f.run("serviceLabel({config:{enabled:false}})[0]"),'未记录');
 assert.equal(f.run("recordTimeRange('18:00')"),'18:00–18:29');
 assert.equal(f.run("recordTimeRange('23:30')"),'23:30–23:59');
 assert.equal(f.run("calledPointLabel({median:123.5,days:5})"),'通常叫到约 124 号');
 assert.equal(f.run("calledPointLabel({median:123,days:1})"),'记录叫到约 123 号');
 assert.equal(f.run("calledPointLabel({median:123,days:1},true)"),'当日叫到 123 号');
 assert.equal(f.run("calledPointLabel(null)"),'暂无叫号记录');
 assert.equal(f.run("chartSegments([{time:'18:00'},{time:'18:30'},{time:'19:30'}]).length"),2,'missing called half-hour was interpolated');
 assert.equal(f.run("waitWords(165)"),'约 2 小时 45 分');
 assert.equal(f.run("waitWords(2)"),'基本不用等');
 assert.equal(f.run("waitWords(0,false)"),'不用等');
 assert.equal(f.run("waitWords(59)"),'约 1 小时');
 assert.equal(f.run("recordAxisTicks(2017,true).join('/')"),'0/500/1000/1500/2000/2500');
 assert.equal(f.run("recordAxisTicks(180,false).join('/')"),'0/60/120/180');
 assert.equal(f.run("recordAxisTicks(30,false).join('/')"),'0/10/20/30');
 assert.equal(f.run("recordHourTicks(630,1290,856).map(recordClock).join(' ')"),'12:00 14:00 16:00 18:00 20:00');
 assert.equal(f.run("recordHourTicks(630,1290,258).map(recordClock).join(' ')"),'12:00 16:00 20:00');
 assert.equal(f.run("recordThinDays([{days:76},{days:20}],false)"),38);
 assert.equal(f.run("recordThinDays([{days:76}],true)"),0);
 assert.equal(f.run("recordSummary({days:8,points:[20,25,35,48,58,62,75,70,60,42,30,25].map((median,i)=>({time:String(16+Math.floor(i/2)).padStart(2,'0')+(i%2?':30':':00'),median,upper:median+15,days:8}))}).text"),'<b>18:30–20:30</b> 最挤，通常要等 1 小时以上。');
 assert.equal(f.run("recordSummary({days:9,points:[{time:'12:00',median:60,days:9},{time:'12:30',median:70,days:9},{time:'13:30',median:75,days:9},{time:'14:00',median:20,days:9}]}).text"),'<b>13:30–14:00</b> 最挤，通常要等约 1 小时 15 分。','summary bridged a missing half-hour');
 assert.match(f.run("recordSummary({days:72,points:[{time:'12:00',median:0,days:72},{time:'12:30',median:0,days:72},{time:'13:00',median:0,days:72},{time:'18:00',median:165,days:11},{time:'18:30',median:160,days:12}]}).note"),/只有 11 天/);
 assert.equal(f.run("recordSummary({days:2,points:[{time:'18:00',median:60,days:2}]}).text"),'等待记录还少（2 天），还看不出哪段时间挤。');
 assert.equal(f.run("recordSummary({points:[],called_points:[]})"),null);
 // The dinner peak has calls on 60 days but almost no published wait: the sentence must not crown lunch without saying so.
 const blind="{days:60,points:['11:00','11:30','12:00','12:30','13:00','13:30','17:00','17:30','18:00','18:30','19:00','19:30'].map((time,i)=>({time,median:[10,40,60,70,40,10,30,0,0,0,30,10][i],days:[60,60,60,60,60,60,60,1,1,2,60,60][i]})),called_points:['11:00','11:30','12:00','12:30','13:00','13:30','17:00','17:30','18:00','18:30','19:00','19:30'].map((time,i)=>({time,median:100+i*50,days:60}))}";
 assert.match(f.run("recordSummary("+blind+").text"),/^有等待记录的时段里，<b>12:00–13:00<\/b> 最挤/);
 assert.match(f.run("recordSummary("+blind+").note"),/17:30–19:00 多数日子没有等待记录/);
 assert.doesNotMatch(f.run("recordSummary({days:8,points:[20,25,35,48,58,62,75,70,60,42,30,25].map((median,i)=>({time:String(16+Math.floor(i/2)).padStart(2,'0')+(i%2?':30':':00'),median,upper:median+15,days:8})),called_points:[]}).text"),/有等待记录/);
 assert.match(f.run("recordReadoutItem(false,{median:37.5,upper:50,days:10},{median:30},false,3)"),/约 40 分钟.*今天 30 分钟（比平时少 10 分钟）/,'today delta ignored the rounded usual value');
 assert.equal(f.run("storeLatestText({store_status:'OPEN',display_called_no:356,wait_minutes:35})"),'上次：叫到 356 号 · 等约 35 分钟');
 assert.equal(f.run("storeLatestText({store_status:'CLOSED'})"),'上次：未营业');
 assert.equal(f.run("serviceLabel({config:{enabled:true},state:{running:true},background_running:true}).join('|')"),'记录中|green');
 assert.equal(f.run("analysisSourceText({called_local_samples:0,called_history_samples:1155,called_days:76},'1',true,'')"),'76 天的数据 · 全部来自内置历史');
 assert.equal(f.run("analysisSourceText({local_samples:230,history_samples:200,days:8},'1',false,'')"),'8 天的数据 · 自己的记录占 53%');
}
{
 const f=fixture();
 f.setRespond(async()=>({version:'4.0',edition:'lite正式版'}));
 await f.run('loadStatus()');
 assert.equal(f.node('version').textContent,'v4.0 · lite正式版');
}
{
 const f=fixture();
 await f.run('reviewTicket()');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'review must be read-only');
 assert.equal(f.node('ticket-dialog').open,true);
 await Promise.all([f.run('confirmTicket()'),f.run('confirmTicket()')]);
 const writes=f.calls.filter(c=>c.path==='/api/queue/ticket');
 assert.equal(writes.length,1,'double click created two requests');
 assert.equal(writes[0].options.headers['X-Sushiro-CSRF'],'test-token');
 assert.equal(JSON.parse(writes[0].options.body).adult,2);
}
{
 const f=fixture();
 await f.run('queryTicket()');
 assert.equal(f.calls.find(c=>c.path==='/api/queue/ticket/status').options.method,'GET');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0);
}
{
 const f=fixture();
 f.setRespond(async path=>{
   if(path==='/api/status')return {has_config:false,engine:{status:'idle'},platform:'windows'};
   return {};
 });
 await f.run('reviewTicket()');await f.run('confirmTicket()');
 assert.equal(f.node('auth-dialog').open,true);
 assert.equal(f.node('auth-method').value,'mobile');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'opening auth started a proxy or created a ticket');
 f.setRespond(async()=>({has_config:true,engine:{status:'idle'},auth_health:{status:'ok'}}));
 await f.run('finishAuth()');
 assert.equal(f.node('ticket-dialog').open,true);
 assert.equal(f.calls.filter(c=>c.path==='/api/queue/ticket').length,0,'auth completion submitted without confirmation');
}
{
 const f=fixture();
 f.setRespond(async path=>{
   if(path==='/api/status')return {has_config:true,engine:{status:'idle'}};
   if(path==='/api/queue/ticket')throw new Error('network timeout');
   return {};
 });
 await f.run('reviewTicket()');await f.run('confirmTicket()');await f.run('confirmTicket()');
 assert.equal(f.run('state.ticketUncertain'),true);
 assert.equal(f.calls.filter(c=>c.path==='/api/queue/ticket').length,1,'uncertain result retried a write');
}
{
 const f=fixture(),pending=[];
 f.setRespond(path=>new Promise(resolve=>pending.push({path,resolve})));
 f.node('store-search').value='old';const first=f.run('searchStores()');
 f.node('store-search').value='new';const second=f.run('searchStores()');
 await new Promise(resolve=>setImmediate(resolve));
 pending[1].resolve({stores:[{id:2,name:'新门店'}]});await second;
 pending[0].resolve({stores:[{id:1,name:'旧门店'}]});await first;
 assert.match(f.node('store-results').innerHTML,/新门店/);
 assert.doesNotMatch(f.node('store-results').innerHTML,/旧门店/);
}
{
 const f=fixture();
 await f.run('openAuth(false)');
 f.setRespond(async path=>{
   if(path==='/api/engine/capture')throw new Error('network timeout');
   return {engine:{status:'idle'}};
 });
 await f.run('startAuth()');
 assert.equal(f.run('authModeRunning'),'desktop','uncertain startup lost cleanup handle');
 await f.run('closeAuth()');
 assert.equal(f.calls.filter(c=>c.path==='/api/engine/stop').length,1);
 assert.equal(f.node('auth-dialog').open,false);
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("state.stores.set('1012',{name:'深圳 <店>'});openStorePicker('records');");
 await new Promise(resolve=>setImmediate(resolve));
 f.setRespond(async()=>({stores:[{id:3006,name:'广州店',city:'广州'}]}));
 f.node('store-search').value='广州';await f.run('searchStores()');
 f.run("state.picker.add('3006');updatePicked();");
 assert.match(f.node('store-selected').innerHTML,/深圳 &lt;店&gt;/,'searching another city hid an existing selection');
 assert.match(f.node('store-selected').innerHTML,/广州店/);
 const result={value:'3006',checked:true,disabled:false};
 f.node('store-results').querySelectorAll=()=>[result];
 let focused='';f.node('store-selected').querySelectorAll=()=>[{focus(){focused='remaining';}}];
 f.run("removePickedStore('3006')");
 assert.equal(result.checked,false,'removing a chip left its result checked');
 assert.equal(focused,'remaining','removing a chip lost keyboard focus');
 f.setRespond(async()=>{throw new Error('网络不可用');});
 await f.run('searchStores()');
 assert.match(f.node('store-results').innerHTML,/网络不可用/);
 assert.match(f.node('store-selected').innerHTML,/深圳 &lt;店&gt;/,'failed search discarded selected stores');
 f.node('store-selected').querySelectorAll=()=>[];
 f.node('store-search').focus=()=>{focused='search';};
 f.run("removePickedStore('1012')");
 assert.equal(focused,'search','removing the last chip did not focus search');
 assert.equal(f.node('store-save').disabled,false,'records must allow saving an empty selection');
 assert.match(f.node('store-selection-hint').textContent,/暂停记录/);
 f.run('closeStorePicker()');
 assert.equal(f.node('store-dialog').open,false);
 assert.equal(f.run('recordIDs().join(",")'),'1012','cancelling saved a draft selection');
 assert.equal(f.calls.filter(call=>call.options.method==='POST').length,0,'search/remove/cancel wrote configuration');
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("state.service.config.enabled=true;state.service.autostart={enabled:true,supported:true};openStorePicker('records');removePickedStore('1012');");
 await new Promise(resolve=>setImmediate(resolve));
 f.run('refreshRecords=async()=>{}');
 f.setRespond(async()=>{throw new Error('保存失败，请重试');});
 await f.run('savePickedStores()');
 assert.equal(f.node('store-dialog').open,true,'failed save closed the picker');
 assert.equal(f.run('state.picker.size'),0,'failed save discarded the draft');
 assert.equal(f.node('store-save').disabled,false,'failed save cannot be retried');
 assert.equal(f.node('store-search').disabled,false);
 assert.match(f.node('store-error').textContent,/保存失败/);
 f.setRespond(async()=>({}));
 await f.run('savePickedStores()');
 const writes=f.calls.filter(call=>call.options.method==='POST');
 assert.equal(writes.length,2,'save retry performed extra configuration operations');
 for(const write of writes){
   assert.equal(write.path,'/api/queue/baseline','clearing stores changed autostart or history');
   const body=JSON.parse(write.options.body);
   assert.deepEqual(body.store_ids,[]);assert.equal(body.enabled,false);assert.equal(body.use_preference_stores,false);
 }
 assert.equal(f.node('store-dialog').open,false);
 assert.match(f.node('toast').textContent,/已暂停记录/);
 assert.equal(f.run('state.service.autostart.enabled'),true,'clearing stores disabled autostart');
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("openStorePicker('records')");await new Promise(resolve=>setImmediate(resolve));
 let resolveSave;
 f.setRespond(()=>new Promise(resolve=>{resolveSave=resolve;}));
 f.run('refreshRecords=async()=>{}');
 const save=f.run('savePickedStores()');await new Promise(resolve=>setImmediate(resolve));
 assert.equal(f.node('store-save').disabled,true);assert.equal(f.node('store-search').disabled,true);
 f.run("closeStorePicker();removePickedStore('1012')");
 assert.equal(f.node('store-dialog').open,true,'closing during a write hid its outcome');
 assert.equal(f.run('state.picker.size'),1,'selection changed during a write');
 resolveSave({});await save;
 assert.equal(f.node('store-dialog').open,false);
}
{
 const f=fixture();
 f.run("state.selectedStore='';openStorePicker('queue');");await new Promise(resolve=>setImmediate(resolve));
 assert.equal(f.node('store-selection').hidden,true,'recording chips appeared in manual ticket selection');
 assert.equal(f.node('store-save').disabled,true,'manual ticket selection accepted zero stores');
 await f.run('savePickedStores()');
 assert.equal(f.node('store-dialog').open,true);
 assert.equal(f.calls.filter(call=>call.options.method==='POST').length,0);
 f.run("state.picker.add('3006');updatePicked();closeStorePicker();");
 assert.equal(f.run('state.selectedStore'),'','cancelling changed manual ticket store');
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("state.service.config.store_ids=[];state.service.state.store_ids=[];state.service.autostart={enabled:true,supported:true};renderService();");
 assert.equal(f.node('autostart').disabled,false,'empty selection prevented disabling existing autostart');
 assert.equal(f.node('record-toggle').disabled,true,'empty selection allowed starting collection');
 assert.equal(f.node('record-toggle').hidden,true,'start button shown before any store was chosen');
 assert.equal(f.node('service-detail').hidden,true,'normal state showed an exception line');
 assert.equal(f.node('record-settings').hidden,false,'existing autostart could not be turned off without stores');
 f.run('state.service.autostart.enabled=false;renderService();');
 assert.equal(f.node('autostart').disabled,true,'empty selection allowed enabling autostart');
 assert.equal(f.node('record-settings').hidden,true,'settings row shown before any store was chosen');
 f.run("openStorePicker('records')");
 assert.equal(f.node('store-selection-hint').textContent,'先搜索常去的门店。','first use described pausing a nonexistent collection');
}
{
 const f=fixture();
 f.setRespond(async path=>{
   if(path==='/api/queue/ticket')throw new Error('network timeout');
   if(path==='/api/status')return {has_config:true,engine:{status:'idle'}};
   if(path.startsWith('/api/queue/live'))return {store_id:'1012',store_name:'测试店',online_open:true};
   return {};
 });
 await f.run('reviewTicket()');await f.run('confirmTicket()');await f.run('loadLive()');
 assert.equal(f.node('take-ticket').disabled,true,'live refresh allowed resubmission of uncertain ticket');
}
{
 const f=fixture(),calls=[];
 f.ctx.window.go={app:{DesktopBridge:{Request:async(...args)=>{calls.push(args);return {status:200,body:'{"ok":true}'};}}}};
 assert.equal((await f.run("api('/api/records/settings',{include_history:false},{timeout:20000})")).ok,true);
 assert.deepEqual(calls,[['POST','/api/records/settings','{"include_history":false}',20000]]);
 assert.equal(f.calls.length,0,'native request fell through to browser fetch');
 f.ctx.window.go.app.DesktopBridge.Request=async()=>({status:409,body:'{"error":"请先查询已有号码"}'});
 await assert.rejects(()=>f.run("api('/api/queue/ticket',{})"),e=>e.status===409&&e.message==='请先查询已有号码');
 f.ctx.window.go.app.DesktopBridge.Request=async()=>{throw '本地服务未连接';};
 await assert.rejects(()=>f.run("api('/api/status')"),/本地服务未连接/);
}
{
 const f=fixture();let prevented=false,saved=0;
 f.ctx.window.go={app:{DesktopBridge:{Export:async resource=>{assert.equal(resource,'/api/records/export?days=all');saved++;return false;}}}};
 f.ctx.downloadEvent={target:{closest:()=>({getAttribute:()=>'/api/records/export?days=all'})},preventDefault(){prevented=true;}};
 f.run('handleDesktopDownload(downloadEvent)');
 await new Promise(resolve=>setImmediate(resolve));
 assert.equal(prevented,true);assert.equal(saved,1);assert.equal(f.node('toast').textContent,'','cancelled export reported success');
}
{
 const f=fixture();
 f.run("state.records={available_stores:[{id:1012,name:'海岸城店',city:'深圳'},{id:3006,name:'测试 <店>',city:'广州'}]};state.stores=new Map(state.records.available_stores.map(s=>[String(s.id),s]));syncAnalysisStore('1012');");
 assert.equal(f.node('analysis-store-search').value,'海岸城店');
 const positions=[];f.node('analysis-store-popup').classList.toggle=(name,value)=>positions.push({name,value});
 f.node('analysis-store-search').getBoundingClientRect=()=>({top:650,bottom:700});
 f.run('openAnalysisStores()');
 assert.ok(positions.some(p=>p.name==='above'&&p.value),'dropdown near window bottom did not open upward');
 f.run('closeAnalysisStores()');
 assert.equal(f.run("filterAnalysisStores('深圳').join(',')"),'1012');
 assert.equal(f.run("filterAnalysisStores('广州 测试').join(',')"),'3006');
 assert.equal(f.run("filterAnalysisStores('不存在').length"),0);
 f.run("openAnalysisStores();renderAnalysisStoreOptions('测试');");
 assert.match(f.node('analysis-store-options').innerHTML,/测试 &lt;店&gt;/);
 assert.equal(f.node('analysis-store').value,'1012','typing changed the selected store');
 f.run("renderAnalysisStoreOptions('不存在')");
 assert.equal(f.node('analysis-store-empty').hidden,false);
 f.run("renderAnalysisStoreOptions('测试');analysisStoreKeydown({key:'ArrowDown',preventDefault(){}})");
 assert.equal(f.node('analysis-store-search').attributes['aria-activedescendant'],'analysis-store-option-0');
 f.run("analysisStoreKeydown({key:'Escape',preventDefault(){},stopPropagation(){}})");
 assert.equal(f.node('analysis-store-popup').hidden,true);
 assert.equal(f.node('analysis-store-search').value,'海岸城店');
 await f.run("selectAnalysisStore('3006')");
 assert.equal(f.node('analysis-store').value,'3006');
 assert.ok(f.calls.some(c=>c.path.includes('/api/records?')&&c.path.includes('store=3006')));
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'selecting chart store wrote settings');
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 assert.equal(f.node('export-records').attributes['aria-disabled'],'true');
 assert.equal(f.node('backup-records').attributes['aria-disabled'],'true');
 f.run('state.records.export_record_count=2;state.records.total_record_count=7;renderRecords();renderAnalysis();');
 assert.equal(f.node('export-records').attributes['aria-disabled'],'false');
 assert.ok(f.node('export-records').href.includes('store=1012'));
 assert.equal(f.node('backup-records').href,'/api/records/export?days=all');
 assert.match(f.node('record-overview').innerHTML,/已选/);
 assert.match(f.node('service-summary').textContent,/尚未开始/);
}
{
 const f=fixture();
 await f.run('loadLive()');
 const before=f.calls.filter(c=>c.path.startsWith('/api/queue/live')).length;
 f.run("navigatePage('records');navigatePage('queue');");
 await new Promise(resolve=>setImmediate(resolve));
 assert.equal(f.calls.filter(c=>c.path.startsWith('/api/queue/live')).length,before+1,'reentering queue did not refresh');
 f.setRespond(async()=>{throw new Error('断网');});
 await f.run('loadLive()');
 assert.equal(f.run('state.liveStale'),true);
 assert.match(f.node('live-store').innerHTML,/未更新：断网/);
 assert.match(f.node('live-store').innerHTML,/112/,'failed refresh discarded last reading');
 assert.equal(f.node('take-ticket').disabled,true);
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'refresh made a write');
}
{
 const f=fixture();
 f.run("state.page='queue'");
 let resolve;
 f.setRespond(path=>path.startsWith('/api/queue/live')?new Promise(done=>{resolve=done;}):Promise.resolve({}));
 const first=f.run('loadLive()');await new Promise(done=>setImmediate(done));
 await f.run('loadLive()');
 assert.equal(f.calls.filter(c=>c.path.startsWith('/api/queue/live')).length,1,'overlapping poll requests');
 resolve({store_id:'1012',store_name:'测试店',online_open:true});await first;
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("state.analysisStores.push({id:3006,name:'另一家店',city:'广州'});state.stores.set('3006',{name:'另一家店',city:'广州'});syncAnalysisStore('3006');");
 f.setRespond(async()=>{throw new Error('离线');});
 await f.run('refreshRecords()');
 assert.equal(f.run('state.records'),null,'failed changed filter kept old chart data');
 assert.equal(f.node('export-records').attributes['aria-disabled'],'true');
 assert.equal(f.run("filterAnalysisStores('').length"),2,'failed switch lost store catalog');
 f.run('renderAnalysis()');
 assert.match(f.node('record-chart').innerHTML,/读取失败/);
}
{
 const f=fixture();
 await f.run('refreshRecords()');
 f.run('state.records.export_record_count=2;renderAnalysis();');
 const before=f.run('state.records');
 const pending=[];f.setRespond(path=>new Promise(resolve=>pending.push({path,resolve})));
 const refresh=f.run('refreshRecords({background:true})');
 await new Promise(resolve=>setImmediate(resolve));
 f.ctx.document.activeElement={closest:()=>({})};
 pending.forEach(p=>p.resolve({}));await refresh;
 assert.equal(f.run('state.records'),before,'background response interrupted an active chart interaction');
 assert.equal(f.node('export-records').attributes['aria-disabled'],'false','deferred refresh disabled a usable export');
 f.ctx.document.activeElement={closest:selector=>selector.includes('button')?{}:null};
 assert.equal(f.run('recordsInteractionActive()'),false,'a focused start button stopped periodic refresh');
}
{
 const f=fixture();
 f.run("showAuthVerificationState({auth_health:{status:'stale',reason:'凭证已过期'}})");
 assert.equal(f.node('auth-continue').hidden,true);
 assert.equal(f.node('auth-verify').hidden,false);
 assert.match(f.node('auth-progress').textContent,/验证失败/);
 f.node('auth-text').value='已粘贴的本人凭证';
 f.run("authModeRunning='mobile';el('auth-dialog').showModal()");
 await f.run('stopAuth()');
 assert.equal(f.node('auth-text').value,'已粘贴的本人凭证');
 assert.equal(f.node('auth-dialog').open,true);
 f.node('auth-method').value='android';
 const writes=f.calls.length;await f.run('startAuth()');
 assert.equal(f.calls.length,writes,'Android import started an unnecessary proxy');
 assert.equal(f.node('auth-import').open,true);
}
{
 const f=fixture();let connected=false;
 f.setRespond(async path=>{
   if(path==='/api/status')return {has_config:connected,platform:'windows',auth_health:{status:connected?'ok':'unknown'}};
   if(path==='/api/queue/ticket/status')return {ticket:null};
   return {};
 });
 await f.run('queryTicket()');
 assert.equal(f.run('authReturnIntent'),'query');
 connected=true;await f.run('finishAuth()');
 assert.equal(f.calls.filter(c=>c.path==='/api/queue/ticket/status').length,1,'auth did not return to the original query');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'resuming query wrote a ticket');
}
{
 const f=fixture();
 f.run("showTicket({ticket:{number:'A128',store_id:'1012'},cancel_token:'expected-ticket'});confirmDialog=async()=>true;");
 await f.run('cancelTicket()');
 const cancel=f.calls.find(c=>c.path==='/api/queue/ticket/cancel');
 assert.deepEqual(JSON.parse(cancel.options.body),{cancel_token:'expected-ticket'});
 await f.run('cancelTicket()');
 assert.equal(f.calls.filter(c=>c.path==='/api/queue/ticket/cancel').length,1,'cleared ticket can still cancel');
}
{
 // api() hands the server's machine-readable code to callers, in the browser and in the native window.
 const f=fixture();
 f.setRespond(async()=>httpError(502,{error:'实时叫号需要连接',code:'live_auth_required'}));
 await assert.rejects(()=>f.run("api('/api/queue/live?store=1012')"),e=>e.code==='live_auth_required'&&e.status===502&&e.message==='实时叫号需要连接');
 f.ctx.window.go={app:{DesktopBridge:{Request:async()=>({status:502,body:JSON.stringify({error:'实时叫号需要连接',code:'live_auth_required'})})}}};
 await assert.rejects(()=>f.run("api('/api/queue/live?store=1012')"),e=>e.code==='live_auth_required'&&e.status===502);
 f.ctx.window.go.app.DesktopBridge.Request=async()=>({status:400,body:'{"error":"缺少门店 ID"}'});
 await assert.rejects(()=>f.run("api('/api/queue/live')"),e=>e.code===undefined&&e.status===400);
}
{
 // No connection: a calm connect panel, the called-number estimate, a disabled ticket button with its reason, and a meal plan anyway.
 const f=fixture();
 f.run("chinaHHMM=()=>'12:00'");
 f.setRespond(async path=>{
   if(path.startsWith('/api/queue/live'))return httpError(502,{error:'实时叫号需要连接：打开寿司郎小程序刷新一下，再在应用里连接（约 1 小时有效）',code:'live_auth_required'});
   if(path.startsWith('/api/queue/plan'))return {store_id:'1012',pickup:'12:00',meal_range:{early:'12:20',late:'12:45'},wait_minutes_range:{low:20,high:45},basis:'按这家店以往工作日的等待规律（历史截至 2026-09-06）',source:'history',history:'历史截至 2026-09-06',message:'暂时拿不到实时等待，先按以往同一时段的等待估计。'};
   return {};
 });
 await f.run('loadLive()');
 const panel=f.node('live-store').innerHTML;
 assert.match(panel,/实时叫号需要连接/);
 assert.match(panel,/data-action="goto-live"/);
 assert.doesNotMatch(panel,/HTTP|403|VPN|未更新/,'connect panel shows a raw error');
 assert.equal(f.run('state.liveAuthRequired'),true);
 assert.equal(f.node('offline-eta').hidden,false,'called-number estimate hidden without a connection');
 assert.equal(f.node('take-ticket').disabled,true);
 assert.equal(f.node('take-ticket-reason').hidden,false);
 assert.equal(f.node('take-ticket-reason').textContent,'先连接实时数据，或在小程序里取号后在这里填号码看叫到时间。');
 assert.ok(f.calls.some(c=>c.path.startsWith('/api/queue/plan?')),'meal plan waited for live data');
 assert.match(f.node('meal-plan').innerHTML,/现在取号，预计入座<strong>12:20–12:45<\/strong>/);
 assert.equal((f.node('meal-plan').innerHTML.match(/历史截至 2026-09-06/g)||[]).length,1,'history cutoff missing or repeated');
 await assert.rejects(()=>f.run('reviewTicket()'),/先连接实时数据/);
 // The periodic refresh keeps the connect panel instead of flashing a loading line.
 let resolve;f.setRespond(path=>path.startsWith('/api/queue/live')?new Promise(done=>{resolve=done;}):Promise.resolve({}));
 const refresh=f.run('loadLive()');await tick();
 assert.match(f.node('live-store').innerHTML,/实时叫号需要连接/,'refresh flashed the loading state');
 resolve({store_id:'1012',store_name:'测试店',store_status:'OPEN',online_open:true,called_no:120,wait_groups:10,server_wait_minutes:20,observed_at:new Date().toISOString()});await refresh;
 assert.equal(f.run('state.liveAuthRequired'),false);
 assert.equal(f.node('offline-eta').hidden,true,'estimate inputs stayed after live data came back');
 assert.equal(f.node('take-ticket').disabled,false);
 assert.equal(f.node('take-ticket-reason').hidden,true);
 assert.match(f.node('live-store').innerHTML,/实时数据 · /);
 // Other failures keep the plain error and retry path.
 f.run("state.liveStore='';state.live=null;");
 f.setRespond(async path=>path.startsWith('/api/queue/live')?httpError(502,{error:'寿司郎服务暂时不可用',code:'live_unavailable'}):{});
 await f.run('loadLive()');
 assert.match(f.node('live-store').innerHTML,/寿司郎服务暂时不可用/);
 assert.match(f.node('live-store').innerHTML,/refresh-live/);
 assert.equal(f.node('take-ticket-reason').textContent,'排队信息还没更新，刷新后再取号。');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'reading live data wrote something');
}
{
 // "想几点吃" works with no live data and passes the travel minutes.
 const f=fixture();
 f.run("chinaHHMM=()=>'12:00';state.live=null;state.liveAuthRequired=true;");
 f.node('meal-mode').value='target';f.node('meal-time').value='18:30';f.node('meal-travel').value='20';
 f.setRespond(async path=>path.startsWith('/api/queue/plan')?{store_id:'1012',target_meal:'18:30',travel_minutes:20,recommend_pickup_range:{early:'17:20',late:'17:40'},basis:'按这家店以往工作日的等待规律（历史截至 2026-09-06）',source:'history',history:'历史截至 2026-09-06'}:{});
 await f.run('loadMealPlan()');
 const q=new URL(f.calls.find(c=>c.path.startsWith('/api/queue/plan?')).path,'http://local').searchParams;
 assert.equal(q.get('store'),'1012');assert.equal(q.get('target_meal'),'1830');assert.equal(q.get('travel_minutes'),'20');
 assert.match(f.node('meal-plan').innerHTML,/建议取号时间<strong>17:20–17:40<\/strong>按这家店以往工作日的等待规律（历史截至 2026-09-06），以门店叫号为准。/);
 f.node('meal-travel').value='300';const before=f.calls.length;
 await f.run('loadMealPlan()');
 assert.equal(f.calls.length,before,'invalid travel minutes reached the server');
 assert.match(f.node('meal-plan').textContent,/0 到 240/);
 f.node('meal-travel').value='';
 await f.run('loadMealPlan()');
 assert.equal(new URL(f.calls.at(-1).path,'http://local').searchParams.has('travel_minutes'),false);
 assert.match(f.run("mealPlanHTML({recommend_pickup_range:{early:'17:20',late:'17:40'},source:'live',basis:'按当前实时等待粗估（适合现在取号）',message:'先按此刻的实时等待粗估'},true,'18:30')"),/^建议取号时间<strong>17:20–17:40<\/strong>按当前实时等待粗估（适合现在取号），以门店叫号为准。$/,'live hint leaked into the plan');
 assert.equal(f.run("mealPlanHTML({message:'这家店这段时间的等待规律还不够，暂时估不出来。'},true,'18:30')"),'这家店这段时间的等待规律还不够，暂时估不出来。');
}
{
 // Called-number estimate: typed numbers go to the advisor and every time shown is HH:MM.
 const f=fixture();
 const offline={store_id:'1012',offline:true,source:'input',history:'历史截至 2026-09-06',called_at:'2026-10-20T17:55:00+08:00',called_at_hhmm:'17:55',current:{called_no:128},
   eta:{target_no:156,remaining_groups:28,estimated_called_at:'2026-10-20T18:30:00+08:00',estimated_called_at_hhmm:'18:30',estimated_called_at_range:{early:'2026-10-20T18:20:00+08:00',late:'2026-10-20T18:45:00+08:00'},range_hhmm:{early:'18:20',late:'18:45'},wait_minutes_range:{low:25,high:50},source:'history_speed',basis:'按这家店以往的叫号速度推算（历史截至 2026-09-06）',arrival_suggestion:'建议 18:00 前后出发。'}};
 f.setRespond(async path=>path.startsWith('/api/queue/advisor')?offline:{});
 f.node('offline-called').value='128';f.node('offline-ticket').value='A156';
 await f.run('offlineEta()');
 const q=new URL(f.calls.find(c=>c.path.startsWith('/api/queue/advisor')).path,'http://local').searchParams;
 assert.equal(q.get('store'),'1012');assert.equal(q.get('called_no'),'128');assert.equal(q.get('target_no'),'156');
 const html=f.node('offline-result').innerHTML;
 assert.equal(f.node('offline-result').hidden,false);
 assert.match(html,/预计叫到<strong>18:20–18:45<\/strong>还要等 25–50 分钟。/);
 assert.match(html,/建议 18:00 前后出发/);
 assert.doesNotMatch(html,/2026-10-20T|\+08:00/,'printed an RFC3339 time');
 assert.equal((html.match(/历史截至 2026-09-06/g)||[]).length,1,'history cutoff missing or repeated');
 assert.equal(f.run("etaClock({estimated_called_at:'2026-10-20T18:30:00+08:00',estimated_called_at_hhmm:'18:30'})"),'约 18:30');
 assert.equal(f.run("etaClock({estimated_called_at_range:{early:'2026-10-20T18:20:00+08:00',late:'2026-10-20T18:40:00+08:00'}})"),'','fell back to an RFC3339 range');
 assert.equal(f.run("etaClock({range_hhmm:{early:'18:30',late:'18:30'}})"),'18:30');
 // Without a typed number the server may use a recent reading, or ask for the current number.
 f.setRespond(async()=>({store_id:'1012',offline:true,needs_called_no:true,code:'live_auth_required',message:'暂时拿不到实时叫号。填上小程序里现在叫到几号，就能按这家店以往的叫号速度推算你的号几点叫到（历史截至 2026-09-06）'}));
 f.node('offline-called').value='';
 await f.run('offlineEta()');
 assert.equal(new URL(f.calls.at(-1).path,'http://local').searchParams.has('called_no'),false);
 assert.match(f.node('offline-result').innerHTML,/填上小程序里现在叫到几号/);
 f.node('offline-called').value='0';const before=f.calls.length;
 await f.run('offlineEta()');
 assert.equal(f.calls.length,before,'invalid called number reached the server');
 assert.match(f.node('offline-result').innerHTML,/1 到 99999/);
 f.node('offline-called').value='';f.node('offline-ticket').value='';
 await f.run('offlineEta()');
 assert.equal(f.calls.length,before,'empty inputs reached the server');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'estimating wrote something');
}
{
 // "记一笔" writes one manual reading and only then estimates when a ticket number is filled in.
 const f=fixture();
 f.setRespond(async path=>path==='/api/queue/reading'?{ok:true,store_id:'1012',called_no:128,observed_at:'2026-10-20T17:55:00+08:00',observed_at_hhmm:'17:55'}:path.startsWith('/api/queue/advisor')?{offline:true,eta:liveEta}:{});
 f.node('offline-called').value='128';f.node('offline-ticket').value='';
 await f.run('recordReading()');
 const write=f.calls.find(c=>c.path==='/api/queue/reading');
 assert.equal(write.options.method,'POST');
 assert.equal(write.options.headers['X-Sushiro-CSRF'],'test-token');
 assert.deepEqual(JSON.parse(write.options.body),{store_id:'1012',called_no:128});
 assert.equal(f.node('toast').textContent,'已记下：17:55 叫到 128 号');
 assert.equal(f.calls.filter(c=>c.path.startsWith('/api/queue/advisor')).length,0,'estimated without a ticket number');
 f.node('offline-ticket').value='140';
 await f.run('recordReading()');
 assert.equal(f.calls.filter(c=>c.path.startsWith('/api/queue/advisor')).length,1);
 assert.match(f.node('offline-result').innerHTML,/18:20–18:40/);
 f.node('offline-called').value='';const writes=f.calls.filter(c=>c.path==='/api/queue/reading').length;
 await f.run('recordReading()');
 assert.equal(f.calls.filter(c=>c.path==='/api/queue/reading').length,writes,'empty reading was written');
 assert.match(f.node('offline-result').innerHTML,/先填小程序里现在叫到几号/);
}
{
 // Ticket card: HH:MM from the live advisor; without live data it asks for the current number.
 const f=fixture();
 f.run("showTicket({ticket:{number:'A128',store_id:'1012'},cancel_token:'t'})");
 for(let i=0;i<5;i++)await tick();
 const forecast=f.node('ticket-forecast').innerHTML;
 assert.match(forecast,/预计叫到<strong>18:20–18:40<\/strong>还要等 30–50 分钟。/);
 assert.doesNotMatch(forecast,/2026-10-20T|\+08:00/,'ticket card printed an RFC3339 time');
 f.run("state.live=null;state.liveAuthRequired=true;");
 f.node('ticket-called-field').hidden=true;
 f.setRespond(async path=>path.startsWith('/api/queue/advisor')?{store_id:'1012',offline:true,needs_called_no:true,message:'暂时拿不到实时叫号。'}:{});
 await f.run("loadTicketForecast('1012','A128')");
 assert.equal(f.node('ticket-called-field').hidden,false,'no way to enter the current number');
 assert.match(f.node('ticket-forecast').textContent,/小程序里现在叫到几号/);
 f.node('ticket-called').value='100';
 f.setRespond(async path=>path.startsWith('/api/queue/advisor')?{store_id:'1012',offline:true,history:'历史截至 2026-09-06',current:{called_no:100},eta:{...liveEta,basis:'按这家店以往的叫号速度推算（历史截至 2026-09-06）'}}:{});
 await f.run("loadTicketForecast('1012','A128')");
 const q=new URL(f.calls.at(-1).path,'http://local').searchParams;
 assert.equal(q.get('called_no'),'100');assert.equal(q.get('target_no'),'128');
 assert.match(f.node('ticket-forecast').innerHTML,/18:20–18:40/);
 assert.match(f.node('ticket-forecast').innerHTML,/历史截至 2026-09-06/);
}
{
 // Store picker: offline answers are usable, say where they come from, and show the city label (nameKana).
 const f=fixture();
 f.run("state.selectedStore='';openStorePicker('queue')");await tick();
 f.setRespond(async path=>path.startsWith('/api/queue/stores')?{stores:[{id:3006,name:'天环店',nameKana:'广州',area:'广州天河区'},{id:3014,name:'正佳店',nameKana:'广州'}],count:2,offline:true,source:'history',code:'live_auth_required',message:'暂时连不上实时数据'}:{});
 f.node('store-search').value='广州';await f.run('searchStores()');
 assert.equal(f.node('store-offline-note').hidden,false,'offline list looked live');
 assert.match(page,/id="store-offline-note"[^>]*>实时门店列表暂时取不到，这里是已知门店。</);
 assert.match(f.node('store-results').innerHTML,/<b>天环店<\/b><small>广州天河区<\/small>/);
 assert.match(f.node('store-results').innerHTML,/<b>正佳店<\/b><small>广州<\/small>/,'nameKana city label dropped');
 f.run("state.picker.add('3006');updatePicked();");
 assert.equal(f.node('store-save').disabled,false,'offline store cannot be chosen');
 f.run("state.analysisStores=[{id:3014}]");
 assert.equal(f.run("filterAnalysisStores('广州').join(',')"),'3014','chart search ignored nameKana');
 f.setRespond(async()=>({stores:[{id:1,name:'在线店',nameKana:'上海'}],count:1}));
 await f.run('searchStores()');
 assert.equal(f.node('store-offline-note').hidden,true);
 assert.doesNotMatch(source,/name_kana/,'store city read from a key the API never sends');
}
{
 // Recording status: waiting for a connection, backoff and rate limits are said plainly; nothing promises an automatic retry.
 const f=fixture();
 await f.run('refreshRecords()');
 f.run("state.service.config.enabled=true;state.service.background_running=true;state.service.state={store_ids:['1012'],waiting_for_connection:true,last_error_code:'live_auth_required',last_error:'实时数据连接未建立或已失效',next_attempt_at:new Date(Date.now()+6*3600e3).toISOString()};renderService();");
 assert.equal(f.node('service-status').textContent,'等待连接');
 assert.equal(f.node('service-detail').hidden,false);
 assert.equal(f.node('service-detail-text').textContent,'记录需要连接实时数据（应用开着时记录）。');
 assert.equal(f.node('service-connect').hidden,false,'no way to connect from the recording status');
 f.run("state.service.state={store_ids:['1012'],last_error:'寿司郎服务暂时不可用（HTTP 502），请稍后重试',last_error_code:'live_unavailable',next_attempt_at:'2099-01-01T10:30:00+08:00'};renderService();");
 assert.equal(f.node('service-status').textContent,'记录出错');
 assert.equal(f.node('service-detail-text').textContent,'上次没记上：寿司郎服务暂时不可用，请稍后重试。下一次在 10:30 之后。');
 assert.equal(f.node('service-connect').hidden,true);
 f.run("state.service.state={store_ids:['1012'],last_error:'请求太频繁（HTTP 429）：请稍后再试',last_error_code:'live_rate_limited',next_attempt_at:'2000-01-01T10:30:00+08:00'};renderService();");
 assert.equal(f.node('service-detail-text').textContent,'寿司郎提示请求太频繁，记录先放慢。','past retry time or status code shown');
 f.run("state.service.config.enabled=false;renderService();");
 assert.equal(f.node('service-detail').hidden,true,'paused recording reported an old error');
 for(const text of [source,page])assert.doesNotMatch(text,/会自动重试|不用连接账号|关掉窗口也会继续|不用登录/,'copy promises recording without a connection');
}
{
 // Settings card states for the live-data connection.
 const f=fixture();
 const view=s=>f.run('liveSessionView('+JSON.stringify(s)+')');
 assert.equal(view({state:'active',connected:true,remaining_seconds:2580,expires_at:'2026-10-20T19:00:00+08:00'}).label,'已连接，还剩 43 分钟');
 assert.match(view({state:'active',connected:true,remaining_seconds:2580,expires_at:'2026-10-20T19:00:00+08:00'}).detail,/约 19:00 到期/);
 assert.equal(view({state:'expiring',connected:true,remaining_seconds:400}).label,'快到期，还剩 6 分钟');
 assert.equal(view({state:'expiring',connected:true,remaining_seconds:30}).label,'快到期，还剩不到 1 分钟');
 assert.match(view({state:'expiring',connected:true,remaining_seconds:400}).detail,/打开小程序刷新一下，再连接一次/);
 assert.equal(view({state:'none',connected:false}).label,'未连接');
 assert.equal(view({state:'expired',connected:false}).label,'已到期');
 assert.match(view({state:'expired',connected:false}).detail,/打开小程序刷新一下，再连接一次/);
 let platform='linux',session={state:'active',connected:true,remaining_seconds:1800};
 f.setRespond(async path=>path==='/api/live-session'?session:path==='/api/status'?{platform,version:'t',engine:{status:'idle'}}:{});
 await f.run('loadStatus()');await f.run('refreshLiveSession()');
 assert.equal(f.node('live-session-state').textContent,'已连接，还剩 30 分钟');
 assert.match(f.node('live-session-state').className,/green/);
 assert.equal(f.node('live-disconnect').hidden,false);
 assert.equal(f.node('live-desktop').hidden,true,'computer capture offered where the account capture is not');
 platform='windows';session={state:'none',connected:false,remaining_seconds:0};
 await f.run('loadStatus()');await f.run('refreshLiveSession()');
 assert.equal(f.node('live-session-state').textContent,'未连接');
 assert.equal(f.node('live-disconnect').hidden,true);
 assert.equal(f.node('live-desktop').hidden,false);
 session={state:'expired',connected:false,remaining_seconds:0};await f.run('refreshLiveSession()');
 assert.equal(f.node('live-session-state').textContent,'已到期');assert.match(f.node('live-session-state').className,/amber/);
 assert.equal(f.node('live-disconnect').hidden,false,'expired connection cannot be removed');
 // Refreshes only while the settings page is shown.
 const reads=()=>f.calls.filter(c=>c.path==='/api/live-session').length,count=reads();
 f.run("state.page='queue';refreshLiveSessionIfShown()");
 assert.equal(reads(),count,'refreshed while settings was hidden');
 f.run("state.page='settings';refreshLiveSessionIfShown()");await tick();
 assert.equal(reads(),count+1);
 let focused='';f.node('live-title').focus=()=>{focused='live';};
 f.run("state.page='queue';gotoLiveSettings()");
 assert.equal(f.run('state.page'),'settings');assert.equal(focused,'live','connect button did not land on the card');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'reading the connection state wrote something');
}
{
 // Phone capture: starts once, keeps the guide, stops the capture as soon as a new connection is saved.
 const f=fixture();let connected=false;
 f.setRespond(async path=>{
   if(path==='/api/live-session')return connected?{state:'active',connected:true,remaining_seconds:3500,captured_at:'2026-10-20T18:00:00+08:00'}:{state:'expired',connected:false,remaining_seconds:0,captured_at:'2026-10-20T16:00:00+08:00'};
   if(path==='/api/mobile-auth/start')return {active:true,addresses:[{url:'http://192.168.1.5:50000/mobile-auth/x',qr_svg:'<svg/>'}]};
   if(path==='/api/mobile-auth')return {active:true};
   if(path==='/api/mobile-auth/stop')return {active:false};
   return {};
 });
 await f.run('refreshLiveSession()');
 await Promise.all([f.run("startLiveCapture('mobile')"),f.run("startLiveCapture('mobile')")]);
 assert.equal(f.calls.filter(c=>c.path==='/api/mobile-auth/start').length,1,'double click started two captures');
 assert.equal(f.run('liveCaptureMode'),'mobile');
 assert.equal(f.node('live-guide').hidden,false,'phone guide not shown');
 assert.equal(f.node('live-qr').src.startsWith('data:image/svg+xml'),true);
 assert.equal(f.node('live-capture-stop').hidden,false);
 assert.equal(f.node('live-mobile').disabled,true);
 assert.match(f.node('live-capture-progress').textContent,/iPhone 扫码/);
 await f.run('pollLiveCapture()');
 assert.equal(f.calls.filter(c=>c.path==='/api/mobile-auth/stop').length,0,'stopped before a connection arrived');
 connected=true;await f.run('pollLiveCapture()');
 assert.equal(f.run('liveCaptureMode'),'');
 assert.equal(f.node('live-session-state').textContent,'已连接，还剩 58 分钟');
 assert.match(f.node('live-capture-progress').textContent,/关闭手机 Wi-Fi 代理/);
 assert.equal(f.node('live-capture-stop').hidden,true);
 const posts=f.calls.filter(c=>c.options.method==='POST');
 assert.deepEqual(posts.map(c=>c.path),['/api/mobile-auth/start','/api/mobile-auth/stop']);
 for(const post of posts)assert.equal(post.options.headers['X-Sushiro-CSRF'],'test-token');
 await f.run('pollLiveCapture()');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,2,'finished capture kept polling');
}
{
 // Phone capture that ends without a connection, a refused start, and the stop button.
 const f=fixture();let active=true;
 f.setRespond(async path=>{
   if(path==='/api/live-session')return {state:'none',connected:false,remaining_seconds:0};
   if(path==='/api/mobile-auth/start')return {active:true,addresses:[{url:'http://192.168.1.5:50000/mobile-auth/x',qr_svg:'<svg/>'}]};
   if(path==='/api/mobile-auth')return {active};
   if(path==='/api/mobile-auth/stop')return {active:false};
   return {};
 });
 await f.run("startLiveCapture('mobile')");
 active=false;await f.run('pollLiveCapture()');
 assert.equal(f.run('liveCaptureMode'),'');
 assert.match(f.node('live-capture-progress').textContent,/还没收到实时数据/);
 await f.run("startLiveCapture('mobile')");
 await f.run('stopLiveCapture()');
 assert.equal(f.calls.filter(c=>c.path==='/api/mobile-auth/stop').length,1);
 assert.equal(f.run('liveCaptureMode'),'');
 assert.match(f.node('live-capture-progress').textContent,/连接已停止/);
 f.setRespond(async path=>path==='/api/mobile-auth/start'?httpError(500,{error:'启动手机凭证捕获失败: 没有可供手机连接的局域网地址'}):{});
 await f.run("startLiveCapture('mobile')");
 assert.equal(f.run('liveCaptureMode'),'','refused start kept a capture handle');
 assert.match(f.node('live-error').textContent,/局域网地址/);
 f.setRespond(async path=>{if(path==='/api/mobile-auth/start')throw new Error('network timeout');return {};});
 await f.run("startLiveCapture('mobile')");
 assert.equal(f.run('liveCaptureMode'),'mobile','unanswered start lost its stop handle');
}
{
 // Computer capture (macOS / Windows) uses the existing engine capture and restores the proxy when done.
 const f=fixture();let connected=false,engine={status:'capturing',stage:'waiting_capture'};
 f.setRespond(async path=>{
   if(path==='/api/live-session')return connected?{state:'active',connected:true,remaining_seconds:3500,captured_at:'B'}:{state:'none',connected:false,remaining_seconds:0};
   if(path==='/api/status')return {platform:'darwin',version:'t',engine};
   if(path==='/api/engine/capture')return {ok:true};
   if(path==='/api/engine/stop')return {ok:true,engine:{status:'idle'}};
   return {};
 });
 await f.run('loadStatus()');await f.run('refreshLiveSession()');
 assert.equal(f.node('live-desktop').hidden,false);
 await f.run("startLiveCapture('desktop')");
 await f.run('pollLiveCapture()');
 assert.match(f.node('live-capture-progress').textContent,/电脑微信/);
 connected=true;await f.run('pollLiveCapture()');
 assert.deepEqual(f.calls.filter(c=>c.options.method==='POST').map(c=>c.path),['/api/engine/capture','/api/engine/stop']);
 assert.match(f.node('live-capture-progress').textContent,/已恢复本应用修改的代理/);
 connected=false;engine={status:'idle'};
 await f.run("startLiveCapture('desktop')");await f.run('pollLiveCapture()');
 assert.equal(f.run('liveCaptureMode'),'');
 assert.match(f.node('live-capture-progress').textContent,/电脑微信连接已结束/);
}
{
 // Pasting a capture: errors keep the paste; success says so; disconnect asks first.
 const f=fixture();
 f.setRespond(async(path,options)=>{
   if(path==='/api/live-session/import')return JSON.parse(options.body).text.includes('sapi.sushiro.com.cn')?{state:'active',connected:true,remaining_seconds:3000,saved:true,message:'实时数据已连接'}:httpError(400,{error:'没找到寿司郎新接口的请求：请抓小程序门店或排队页发往 sapi.sushiro.com.cn 的请求'});
   if(path==='/api/live-session/clear')return {state:'none',connected:false,remaining_seconds:0};
   return {};
 });
 f.node('live-text').value='';await f.run('importLiveSession()');
 assert.equal(f.calls.length,0,'empty paste reached the server');
 assert.match(f.node('live-error').textContent,/先粘贴/);
 const old="curl 'https://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/getStoreById?storeId=1012'";
 f.node('live-text').value=old;await f.run('importLiveSession()');
 assert.equal(f.node('live-error').hidden,false);
 assert.match(f.node('live-error').textContent,/没找到寿司郎新接口的请求/);
 assert.equal(f.node('live-text').value,old,'failed import discarded the paste');
 f.node('live-text').value="curl 'https://sapi.sushiro.com.cn/gateway/wechat/api/2.0/getStoreById?storeId=1012' -H 'authorization: Bearer x'";
 await f.run('importLiveSession()');
 assert.equal(f.node('live-error').hidden,true);
 assert.equal(f.node('live-text').value,'','pasted request kept on screen after connecting');
 assert.equal(f.node('live-session-state').textContent,'已连接，还剩 50 分钟');
 assert.equal(f.node('toast').textContent,'实时数据已连接');
 const imports=f.calls.filter(c=>c.path==='/api/live-session/import');
 assert.equal(imports.length,2);
 assert.equal(imports[1].options.headers['X-Sushiro-CSRF'],'test-token');
 f.run('confirmDialog=async()=>false');await f.run('disconnectLive()');
 assert.equal(f.calls.filter(c=>c.path==='/api/live-session/clear').length,0,'disconnected without asking');
 f.run('confirmDialog=async()=>true');await f.run('disconnectLive()');
 assert.equal(f.calls.filter(c=>c.path==='/api/live-session/clear').length,1);
 assert.equal(f.node('live-session-state').textContent,'未连接');
 assert.equal(f.node('live-disconnect').hidden,true);
}
{
 // Opening the account dialog first stops a live-data capture, so two captures never run at once.
 const f=fixture();
 f.setRespond(async path=>path==='/api/mobile-auth/start'?{active:true,addresses:[{url:'http://192.168.1.5:50000/mobile-auth/x',qr_svg:'<svg/>'}]}:path==='/api/status'?{has_config:false,platform:'windows',engine:{status:'idle'}}:{});
 await f.run("startLiveCapture('mobile')");
 await f.run("openAuth('settings')");
 assert.equal(f.calls.filter(c=>c.path==='/api/mobile-auth/stop').length,1);
 assert.equal(f.run('liveCaptureMode'),'');
 assert.equal(f.node('auth-dialog').open,true);
}
{
 // A store first waiting for a connection, then failing for another reason, stops asking to connect.
 const f=fixture();
 f.setRespond(async path=>path.startsWith('/api/queue/live')?httpError(502,{error:'实时叫号需要连接',code:'live_auth_required'}):{});
 await f.run('loadLive()');
 assert.equal(f.run('state.liveAuthRequired'),true);
 f.setRespond(async path=>path.startsWith('/api/queue/live')?httpError(502,{error:'寿司郎服务暂时不可用（HTTP 502），请稍后重试',code:'live_unavailable'}):{});
 await f.run('loadLive()');
 assert.equal(f.run('state.liveAuthRequired'),false,'a network failure still asked to connect');
 assert.equal(f.node('take-ticket-reason').textContent,'排队信息还没更新，刷新后再取号。');
 assert.match(f.node('live-store').innerHTML,/寿司郎服务暂时不可用，请稍后重试。/);
 assert.doesNotMatch(f.node('live-store').innerHTML,/HTTP/,'store card showed a status code');
 f.run("state.selectedStore='';openStorePicker('queue')");await tick();
 f.setRespond(async path=>path.startsWith('/api/queue/stores')?httpError(502,{error:'请求太频繁（HTTP 429）：请稍后再试',code:'live_rate_limited'}):{});
 await f.run('searchStores()');
 assert.match(f.node('store-results').innerHTML,/请求太频繁：请稍后再试。/);
 assert.doesNotMatch(f.node('store-results').innerHTML,/HTTP/,'store search showed a status code');
 f.run("state.selectedStore='1012'");
 assert.equal(f.node('offline-eta').hidden,false,'estimate inputs hidden while live data is missing');
}
{
 // Ticket card: a typed called number keeps the moment it was typed, so the 30-second refresh does not slide the estimate later.
 const f=fixture();
 f.run("showTicket({ticket:{number:'A128',store_id:'1012'},cancel_token:'t'})");
 for(let i=0;i<5;i++)await tick();
 f.run("state.live=null;state.liveAuthRequired=true;");
 f.node('ticket-called-field').hidden=false;f.node('ticket-called').value='100';
 f.setRespond(async path=>{
   if(!path.startsWith('/api/queue/advisor'))return {};
   const q=new URL(path,'http://local').searchParams;
   return {store_id:'1012',offline:true,source:'input',history:'历史截至 2026-09-06',called_at:q.get('called_at'),called_at_hhmm:'17:55',current:{called_no:100},eta:{...liveEta,basis:'按这家店以往的叫号速度推算（历史截至 2026-09-06）'}};
 });
 await f.run("loadTicketForecast('1012','A128')");
 const first=new URL(f.calls.at(-1).path,'http://local').searchParams.get('called_at');
 assert.match(first,/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/,'called_at is not RFC3339 the server accepts');
 assert.match(f.node('ticket-forecast').innerHTML,/按 17:55 填的叫到 100 号推算/);
 await new Promise(resolve=>setTimeout(resolve,1100));
 f.run('refreshTicketForecast()');for(let i=0;i<5;i++)await tick();
 const refreshed=f.calls.filter(c=>c.path.startsWith('/api/queue/advisor')).at(-1);
 assert.equal(new URL(refreshed.path,'http://local').searchParams.get('called_at'),first,'refresh re-stamped an old reading as now');
 f.node('ticket-called').value='104';
 await f.run("loadTicketForecast('1012','A128')");
 assert.notEqual(new URL(f.calls.at(-1).path,'http://local').searchParams.get('called_at'),first,'a new number kept the old time');
 assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0,'estimating wrote something');
 assert.match(f.run("etaHTML({eta:{range_hhmm:{early:'18:20',late:'18:40'},wait_minutes_range:{low:30,high:30}}})"),/还要等 30 分钟。/);
}
{
 // Capture: a connection saved between the two reads still counts; one saved before the capture started does not.
 const f=fixture();let saved=false;
 f.setRespond(async path=>{
   if(path==='/api/live-session')return saved?{state:'active',connected:true,remaining_seconds:3500,captured_at:'B'}:{state:'none',connected:false,remaining_seconds:0};
   if(path==='/api/mobile-auth/start')return {active:true,addresses:[{url:'http://192.168.1.5:50000/mobile-auth/x',qr_svg:'<svg/>'}]};
   // The capture saves the connection and stops itself right as its status is read.
   if(path==='/api/mobile-auth'){saved=true;return {active:false,saved:true};}
   return {};
 });
 await f.run("startLiveCapture('mobile')");
 await f.run('pollLiveCapture()');
 assert.equal(f.run('liveCaptureMode'),'');
 assert.match(f.node('live-capture-progress').textContent,/实时数据已连接/,'saved connection reported as missing');
 const g=fixture();
 g.run("state.liveSession={state:'active',connected:true,remaining_seconds:3000,captured_at:'A'}");
 g.setRespond(async path=>path==='/api/live-session'?{state:'active',connected:true,remaining_seconds:3400,captured_at:'B'}:path==='/api/mobile-auth/start'||path==='/api/mobile-auth'?{active:true,addresses:[{url:'http://192.168.1.5:50000/mobile-auth/x',qr_svg:'<svg/>'}]}:{});
 await g.run("startLiveCapture('mobile')");
 await g.run('pollLiveCapture()');
 assert.equal(g.run('liveCaptureMode'),'mobile','an older connection ended the new capture');
 assert.equal(g.calls.filter(c=>c.path==='/api/mobile-auth/stop').length,0);
}
{
 // Computer capture: the card says the proxy is restored only after the stop answered so.
 const f=fixture();let stop=httpError(500,{error:'停止失败'}),captures=0;
 f.setRespond(async path=>{
   // Each capture saves a newer connection as soon as it starts.
   if(path==='/api/live-session')return captures?{state:'active',connected:true,remaining_seconds:3500,captured_at:'C'+captures}:{state:'none',connected:false,remaining_seconds:0};
   if(path==='/api/status')return {platform:'darwin',version:'t',engine:{status:'capturing',stage:'waiting_capture'}};
   if(path==='/api/engine/capture'){captures++;return {ok:true};}
   if(path==='/api/engine/stop')return stop;
   return {};
 });
 await f.run("startLiveCapture('desktop')");await f.run('pollLiveCapture()');
 assert.equal(f.run('liveCaptureMode'),'');
 assert.doesNotMatch(f.node('live-capture-progress').textContent,/已恢复/,'claimed the proxy was restored after a failed stop');
 assert.match(f.node('live-error').textContent,/恢复系统代理/);
 stop={ok:true,engine:{status:'stopping'}};
 await f.run("startLiveCapture('desktop')");await f.run('pollLiveCapture()');
 assert.match(f.node('live-capture-progress').textContent,/代理正在恢复/);
}
{
 // A first status read that fails does not keep saying "读取中".
 const f=fixture();
 f.setRespond(async path=>path==='/api/live-session'?httpError(500,{error:'x'}):{});
 await f.run('refreshLiveSession()');
 assert.equal(f.node('live-session-state').textContent,'暂时读不到');
 assert.match(f.node('live-session-detail').textContent,/读取失败/);
}
console.log('Web UI behavior: passed (search, selected stores, clear/pause, save retry, keyboard focus, stale filter/catalog, focus-safe refresh, export scopes, auth recovery, bound cancellation, one-shot ticket, native bridge, live code branching, offline estimate and reading, HH:MM times, offline store note, recording status copy, live-data card and capture, typed-number time, capture race).');
