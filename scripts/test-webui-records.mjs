import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const root=new URL('../internal/app/webui/',import.meta.url);
const source=fs.readFileSync(new URL('app.js',root),'utf8')+'\n'+fs.readFileSync(new URL('record_chart.js',root),'utf8')+'\n'+fs.readFileSync(new URL('auth_ticket.js',root),'utf8');
function fixture() {
  const nodes=new Map(),calls=[];
  function node(id) {
    if(!nodes.has(id))nodes.set(id,{id,value:'',hidden:false,disabled:false,open:false,textContent:'',innerHTML:'',options:[],dataset:{},style:{},
      showModal(){this.open=true;},close(){this.open=false;},focus(){},scrollIntoView(){},replaceChildren(){},append(){},setAttribute(){},removeAttribute(){}});
    return nodes.get(id);
  }
  let respond=async(path)=> {
    if(path==='/api/status')return {has_config:true,platform:'darwin',version:'test',engine:{status:'idle'},auth_health:{status:'unknown'}};
    if(path==='/api/queue/ticket')return {ticket:{number:'A128',store_id:'1012'}};
    if(path==='/api/queue/ticket/status')return {ticket:null};
    if(path.startsWith('/api/queue/live'))return {store_id:'1012',store_name:'测试店',store_status:'OPEN',online_open:true,called_no:112,wait_groups:15,server_wait_minutes:30};
    if(path.startsWith('/api/queue/plan'))return {meal_range:{early:'18:20',late:'18:40'}};
    if(path.startsWith('/api/queue/advisor'))return {eta:{estimated_called_at_range:{early:'18:20',late:'18:40'},wait_minutes_range:{low:30,high:50}}};
    return {};
  };
  const ctx=vm.createContext({console,URL,URLSearchParams,AbortController,
    setTimeout(){return 1;},clearTimeout(){},setInterval(){},clearInterval(){},
    location:{hash:'#queue'},history:{replaceState(){}},window:{addEventListener(){}},
    document:{getElementById:node,querySelector(){return {content:'test-token'};},querySelectorAll(){return[];},addEventListener(){},activeElement:null},
    fetch:async(path,options)=>{calls.push({path,options});return {ok:true,json:()=>respond(path,options)};}
  });
  vm.runInContext(source,ctx);
  node('ticket-adult').value='2';node('ticket-child').value='0';node('ticket-table').value='T';node('meal-mode').value='now';
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
 f.setRespond(async()=>({has_config:true,engine:{status:'idle'},auth_health:{status:'unknown'}}));
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
 f.setRespond(async path=>{
   if(path==='/api/queue/ticket')throw new Error('network timeout');
   if(path==='/api/status')return {has_config:true,engine:{status:'idle'}};
   if(path.startsWith('/api/queue/live'))return {store_id:'1012',store_name:'测试店',online_open:true};
   return {};
 });
 await f.run('reviewTicket()');await f.run('confirmTicket()');await f.run('loadLive()');
 assert.equal(f.node('take-ticket').disabled,true,'live refresh allowed resubmission of uncertain ticket');
}
console.log('Web UI behavior: passed (read-only entry, explicit auth, one-shot ticket, unknown result, stale search, auth cleanup).');
