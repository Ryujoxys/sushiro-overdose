import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const root=new URL('../internal/app/webui/',import.meta.url);
const source=['app.js','record_chart.js','store_select.js','auth_ticket.js'].map(name=>fs.readFileSync(new URL(name,root),'utf8')).join('\n');
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
    if(!nodes.has(id))nodes.set(id,{id,value:'',hidden:false,disabled:false,open:false,textContent:'',innerHTML:'',options:[],dataset:{},style:{},attributes:{},listeners:{},classList:{toggle(){}},
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
    if(path.startsWith('/api/queue/advisor'))return {eta:{estimated_called_at_range:{early:'18:20',late:'18:40'},wait_minutes_range:{low:30,high:50}}};
    return {};
  };
  const ctx=vm.createContext({console,URL,URLSearchParams,AbortController,Option:function(text,value){this.text=text;this.value=value;},
    setTimeout(){return 1;},clearTimeout(){},setInterval(){},clearInterval(){},
    location:{hash:'#queue'},history:{replaceState(){}},window:{innerHeight:800,addEventListener(){}},
    document:{getElementById:node,querySelector(selector){return selector.includes('meta')?{content:'test-token'}:null;},querySelectorAll(){return[];},addEventListener(){},activeElement:null,createElement(tag){return node('created-'+tag);}},
    fetch:async(path,options)=>{calls.push({path,options});return {ok:true,json:()=>respond(path,options)};}
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
 f.run('state.service.autostart.enabled=false;renderService();');
 assert.equal(f.node('autostart').disabled,true,'empty selection allowed enabling autostart');
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
console.log('Web UI behavior: passed (search, selected stores, clear/pause, save retry, keyboard focus, stale filter/catalog, focus-safe refresh, export scopes, auth recovery, bound cancellation, one-shot ticket, native bridge).');
