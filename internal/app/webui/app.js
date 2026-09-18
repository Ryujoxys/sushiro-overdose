'use strict';

const el = id => document.getElementById(id);
const state = {page:'records', analysisMetric:'called', analysisTime:'', service:null, records:null, status:null, stores:new Map(), selectedStore:'', live:null, picker:new Set(), pickerMode:'records', recordsRevision:0, liveRevision:0, searchRevision:0, planRevision:0, busy:new Set()};
const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
function storeName(id) { return state.stores.get(String(id))?.name || state.records?.stores?.find(s=>String(s.id)===String(id))?.name || '门店 '+id; }
function clockTime(value) { if(!value)return '还没有'; const d=new Date(value); return Number.isNaN(d.getTime())?'时间未知':d.toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}); }
function chinaHHMM(date=new Date()) { return date.toLocaleTimeString('en-GB',{timeZone:'Asia/Shanghai',hour:'2-digit',minute:'2-digit',hour12:false}); }
function serviceLabel(data) {
  if(!data?.config?.enabled)return ['未记录',''];
  if(data.state?.last_error)return ['记录遇到问题','amber'];
  if(data.state?.paused_reason)return ['暂时避让认证','amber'];
  if(data.state?.running)return [data.background_running?'后台记录中':'应用内记录中','green'];
  return [data.background_running?'等待下一次记录':'等待启动','amber'];
}
function ticketOptions(adult,child,table) {
  if(String(adult).trim()===''||String(child).trim()==='')throw new Error('请填写人数，没有儿童请填 0。');
  const a=Number(adult),c=Number(child);
  if(!Number.isInteger(a)||!Number.isInteger(c)||a<0||c<0||a>10||c>10||a+c<1)throw new Error('人数为 0 到 10 的整数，至少一人。');
  if(!['T','C'].includes(table))throw new Error('请选择座位。');
  return {adult:a,child:c,table_type:table};
}
async function api(path,body,options={}) {
  const bridge=window.go?.app?.DesktopBridge;
  if(bridge){
    try {
      const response=await bridge.Request(body===undefined?'GET':'POST',path,body===undefined?'':JSON.stringify(body),options.timeout||15000);
      const data=JSON.parse(response.body);
      if(response.status<200||response.status>=300||data.error){const error=new Error(data.error||'请求失败，请重试。');error.status=response.status;throw error;}
      return data;
    }catch(error){throw typeof error==='string'?new Error(error):error;}
  }
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),options.timeout||15000);
  try {
    const headers={Accept:'application/json'};
    if(body!==undefined){headers['Content-Type']='application/json';headers['X-Sushiro-CSRF']=document.querySelector('meta[name="sushiro-csrf"]').content;}
    const response=await fetch(path,{method:body===undefined?'GET':'POST',headers,body:body===undefined?undefined:JSON.stringify(body),signal:controller.signal,cache:'no-store'});
    const data=await response.json();
    if(!response.ok||data.error){const error=new Error(data.error||'请求失败，请重试。');error.status=response.status;throw error;}
    return data;
  } catch(error) {
    if(error.name==='AbortError')throw new Error('请求超时，请稍后重试。');
    throw error;
  } finally { clearTimeout(timer); }
}
function handleDesktopDownload(event) {
  const bridge=window.go?.app?.DesktopBridge;
  if(!bridge)return;
  const link=event.target.closest('a[href]');
  if(!link)return;
  const resource=link.getAttribute('href');
  if(!/^\/api\/(records\/export|diagnostics\/bundle)(\?|$)/.test(resource))return;
  event.preventDefault();
  guard('export',async()=>{if(await bridge.Export(resource))toast('已保存');});
}
let toastTimer;
function toast(message) { el('toast').textContent=message;el('toast').hidden=false;clearTimeout(toastTimer);toastTimer=setTimeout(()=>{el('toast').hidden=true;},5000); }
async function guard(key,run) {
  if(state.busy.has(key))return;
  state.busy.add(key);
  try {return await run();} catch(e){toast(typeof e==='string'?e:e.message||'操作失败，请重试。');} finally{state.busy.delete(key);}
}
function showError(id,error) { const box=el(id);box.hidden=!error;box.textContent=error?.message||String(error||''); }
function navigatePage(page) {
  if(!['records','queue','settings'].includes(page))page='records';
  state.page=page;
  document.querySelectorAll('[id^="page-"]').forEach(n=>{n.hidden=n.id!=='page-'+page;});
  document.querySelectorAll('[data-page]').forEach(n=>{if(n.dataset.page===page)n.setAttribute('aria-current','page');else n.removeAttribute('aria-current');});
  if(location.hash!=='#'+page)history.replaceState(null,'','#'+page);
  if(page==='queue'&&!state.selectedStore){state.selectedStore=String(recordIDs()[0]||'');if(state.selectedStore)loadLive();}
  if(page==='records'&&state.records)renderAnalysis();
  if(page==='settings')loadStatus().catch(e=>toast(e.message));
}
function recordIDs(){return (state.service?.state?.store_ids||state.service?.config?.store_ids||[]).map(String);}
function recordQuery() {
  const q=new URLSearchParams({days:el('analysis-days').value,date_type:el('analysis-date-type').value});
  if(el('analysis-store').value)q.set('store',el('analysis-store').value);
  if(el('analysis-date').value)q.set('date',el('analysis-date').value);
  return q;
}
async function refreshRecords() {
  const revision=++state.recordsRevision;
  el('record-chart').setAttribute('aria-busy','true');
  try {
    const [service,records]=await Promise.all([api('/api/queue/service'),api('/api/records?'+recordQuery())]);
    if(revision!==state.recordsRevision)return;
    state.service=service;state.records=records;
    for(const s of [...(records.available_stores||[]),...(records.stores||[])])state.stores.set(String(s.id),{...state.stores.get(String(s.id)),...s});
    showError('records-error',null);renderRecords();renderService();renderAnalysis();
  }catch(e){if(revision===state.recordsRevision){state.records=null;showError('records-error',e);el('record-chart').innerHTML='<p class="empty">记录暂时读取失败，请刷新重试。</p>';el('record-table').innerHTML='';el('analysis-inspection').hidden=true;el('analysis-source').textContent='';}}
  finally{if(revision===state.recordsRevision)el('record-chart').removeAttribute('aria-busy');}
}
function renderRecords() {
  const model=state.service.model||{};
  el('record-overview').innerHTML=[['已保存',model.sample_count||0,'条'],['覆盖',model.day_count||0,'天'],['记录',recordIDs().length,'家店']].map(([name,n,unit])=>'<div class="metric"><span>'+name+' </span><strong>'+n+'</strong><span>'+unit+'</span></div>').join('');
  const rows=recordIDs().map(id=>{
    const s=state.records.stores.find(row=>String(row.id)===id),latest=s?.latest;
    return '<div class="store-row"><div><button class="store-name" data-action="analyze-store" data-id="'+escapeHTML(id)+'">'+escapeHTML(storeName(id))+'</button><p>'+escapeHTML(latest?clockTime(latest.collected_at)+' 记录 · '+s.samples+' 条':'等待第一条记录')+'</p></div><div class="store-values">'+(latest?'<b>'+escapeHTML(latest.store_status==='OPEN'?'约 '+latest.wait_minutes+' 分钟':'非营业时段')+'</b><small>最近记录，不是实时数据</small>':'<span class="badge">尚无记录</span>')+'</div></div>';
  });
  el('record-stores').innerHTML=rows.join('')||'<div class="empty"><strong>先选常去的几家店</strong>开始记录后，关掉界面也能继续。</div>';
  const select=el('analysis-store'),selected=select.value||String(state.records.selected_store||'');
  const ids=[...new Set([...recordIDs(),...(state.records.available_stores||state.records.stores||[]).map(s=>String(s.id)),...(selected?[selected]:[])])];
  select.innerHTML='<option value="">选择门店</option>'+ids.map(id=>'<option value="'+escapeHTML(id)+'">'+escapeHTML(storeName(id))+'</option>').join('');
  if(ids.includes(selected))select.value=selected;
  el('data-path').textContent=state.service.data_path||'~/.sushiro/';
}
function renderService() {
  const data=state.service,cfg=data.config,auto=data.autostart||{},ids=recordIDs(),[label,tone]=serviceLabel(data);
  el('service-status').textContent=label;el('service-status').className='badge '+tone;
  el('service-summary').textContent=ids.length?'已选 '+ids.length+' 家门店。':'先选择至少一家门店。';
  const interval=el('record-interval');
  if(![...interval.options].some(o=>Number(o.value)===cfg.interval_minutes))interval.add(new Option('每 '+cfg.interval_minutes+' 分钟',cfg.interval_minutes));
  if(document.activeElement!==interval)interval.value=cfg.interval_minutes||5;
  el('record-toggle').textContent=cfg.enabled?'暂停记录':'开始记录';el('record-toggle').disabled=!ids.length||state.busy.has('service');
  el('autostart').checked=!!auto.enabled;el('autostart').disabled=auto.supported===false||!ids.length;
  el('autostart').title=auto.error||'单独开启，不会随开始记录自动启用';
  el('service-detail').textContent=data.state.last_error||data.state.paused_reason||auto.error||('最近记录：'+clockTime(data.state.last_at)+(data.background_running?'。可以关闭界面。':cfg.enabled?'。后台未就绪时，请保持界面运行。':''));
}
function renderAnalysis() {
  const data=state.records,id=el('analysis-store').value,called=state.analysisMetric!=='wait';
  el('export-records').href='/api/records/export?'+recordQuery();
  el('include-history').checked=!!data.history?.included;
  el('include-history').disabled=state.busy.has('history');
  el('history-meta').textContent=data.history?.error||(data.history?.cutoff_at?'内置历史 '+data.history.first_at.slice(0,10)+' 至 '+data.history.cutoff_at.slice(0,10)+' · 不会自动更新':'未附带历史数据');
  const dateSelect=el('analysis-date'),selectedDate=dateSelect.value;
  const dates=[...new Set([...(data.dates||[]),...(selectedDate?[selectedDate]:[])])].sort().reverse();
  dateSelect.innerHTML='<option value="">全部日期</option>'+dates.map(date=>'<option value="'+escapeHTML(date)+'">'+escapeHTML(date)+'</option>').join('');
  dateSelect.value=selectedDate;
  const own=called?data.called_local_samples:data.local_samples,history=called?data.called_history_samples:data.history_samples,days=called?data.called_days:data.days;
  el('analysis-source').textContent=id?'本机 '+(own||0)+' · 历史 '+(history||0)+' 个时段样本 · '+(days||0)+' 天':'选择门店查看。';
  renderRecordChart(data);
}
async function changeHistory() {
  const included=el('include-history').checked;
  await guard('history',async()=>{
    state.recordsRevision++;
    el('include-history').disabled=true;
    el('record-chart').innerHTML='<p class="empty">正在更新…</p>';el('record-table').innerHTML='';el('analysis-inspection').hidden=true;
    try { await api('/api/records/settings',{include_history:included}); }
    finally { await refreshRecords(); }
  });
  el('include-history').disabled=false;
}
async function saveRecordConfig(ids=recordIDs()) {
  if(!state.service)throw new Error('请先刷新记录状态。');
  await api('/api/queue/baseline',{...state.service.config,store_ids:ids,use_preference_stores:false,interval_minutes:Number(el('record-interval').value)});
}
async function toggleRecording() {
  await guard('service',async()=>{
    el('record-toggle').disabled=true;
    try{await saveRecordConfig();await api('/api/queue/service',{action:state.service.config.enabled?'pause':'start'},{timeout:20000});}
    finally{await refreshRecords();}
  });
  if(state.service)renderService();
}
async function changeAutostart() {
  const enabled=el('autostart').checked;el('autostart').disabled=true;
  await guard('service',async()=>{
    if(enabled&&!await confirmDialog('登录后自动记录','请把程序放在固定位置。登录电脑后会启动公开数据记录，不会取号。')){renderService();return;}
    try{await saveRecordConfig();await api('/api/queue/service',{action:enabled?'enable_autostart':'disable_autostart'},{timeout:20000});}
    finally{await refreshRecords();}
  });
  if(state.service)renderService();
}
let searchTimer;
function openStorePicker(mode) {
  state.pickerMode=mode;state.picker=new Set(mode==='records'?recordIDs():state.selectedStore?[state.selectedStore]:[]);
  el('store-dialog-title').textContent=mode==='records'?'选择记录门店':'今天去哪家店';
  el('store-save').textContent=mode==='records'?'保存门店':'选这家店';
  el('store-search').value='';el('store-dialog').showModal();updatePicked();searchStores();
}
function updatePicked(){el('store-picked').textContent='已选 '+state.picker.size+' 家';el('store-save').disabled=!state.picker.size;}
async function searchStores() {
  const revision=++state.searchRevision,q=el('store-search').value.trim();
  el('store-results').innerHTML='<p class="empty">正在找门店…</p>';
  try{
    const data=await api('/api/queue/stores?limit=100&q='+encodeURIComponent(q));
    if(revision!==state.searchRevision)return;
    for(const store of data.stores||[])state.stores.set(String(store.id),store);
    el('store-results').innerHTML=(data.stores||[]).map(s=>'<label class="picker-row"><input type="'+(state.pickerMode==='records'?'checkbox':'radio')+'" name="picked-store" value="'+escapeHTML(s.id)+'" '+(state.picker.has(String(s.id))?'checked':'')+'><span><b>'+escapeHTML(s.name)+'</b><small>'+escapeHTML([s.name_kana||s.city,s.address].filter(Boolean).join(' · '))+'</small></span></label>').join('')||'<p class="empty">没有找到，试试城市或其他店名。</p>';
  }catch(e){if(revision===state.searchRevision)el('store-results').innerHTML='<p class="empty">'+escapeHTML(e.message)+'</p><button class="button secondary" data-action="retry-stores">重试</button>';}
}
async function savePickedStores() {
  await guard('stores',async()=>{
    const ids=[...state.picker];if(!ids.length)return;
    el('store-save').disabled=true;
    try{
      if(state.pickerMode==='records'){await saveRecordConfig(ids);await refreshRecords();toast('门店已保存');}
      else{state.selectedStore=ids[0];loadLive();}
      el('store-dialog').close();
    }finally{updatePicked();}
  });
}
async function loadLive() {
  const id=state.selectedStore,revision=++state.liveRevision;
  state.planRevision++;
  state.live=null;el('take-ticket').disabled=true;el('live-store').innerHTML='<p class="empty">正在获取排队…</p>';
  el('meal-plan').textContent='正在读取预计时间…';
  try{
    const panel=await api('/api/queue/live?store='+encodeURIComponent(id));
    if(revision!==state.liveRevision)return;
    state.live=panel;state.stores.set(id,{...state.stores.get(id),name:panel.store_name});
    const open=panel.store_status==='OPEN';
    el('live-store').innerHTML='<div class="live-title"><h3>'+escapeHTML(panel.store_name)+'</h3><p>'+escapeHTML(open?'营业中':'当前未营业')+' · '+escapeHTML(panel.online_open?'可线上取号':'线上取号暂未开放')+'</p></div><div class="live-numbers"><div><strong>'+escapeHTML(panel.called_no>0?panel.called_no:'—')+'</strong><span>当前叫到</span></div><div><strong>'+escapeHTML(open?panel.wait_groups:'—')+'</strong><span>在等桌数</span></div><div><strong>'+escapeHTML(open?panel.server_wait_minutes:'—')+'</strong><span>官方等待 / 分钟</span></div></div><p class="caption">官方公开数据 · '+escapeHTML(clockTime(panel.observed_at))+'</p><button class="button quiet small" data-action="refresh-live">刷新排队</button>';
    el('take-ticket').disabled=!panel.online_open||!!state.ticketUncertain;await loadMealPlan();
  }catch(e){if(revision===state.liveRevision){el('live-store').innerHTML='<p class="empty">'+escapeHTML(e.message)+'</p><button class="button secondary" data-action="refresh-live">重试</button>';el('meal-plan').textContent='实时数据未更新，请重试。';}}
}
async function loadMealPlan() {
  const revision=++state.planRevision;
  if(!state.selectedStore||!state.live)return;
  const target=el('meal-mode').value==='target',time=target?el('meal-time').value:chinaHHMM();
  el('meal-fields').hidden=!target;
  if(target&&(!time||time<=chinaHHMM())){el('meal-plan').textContent='请选择今天稍后的时间。';return;}
  const travel=Number(el('travel-minutes').value||0);
  if(target&&(!Number.isInteger(travel)||travel<0||travel>240)){el('meal-plan').textContent='路程请填写 0 到 240 的整数分钟。';return;}
  el('meal-plan').textContent='正在估算…';
  try{
    const data=await api('/api/queue/plan?store='+encodeURIComponent(state.selectedStore)+'&'+(target?'target_meal':'pickup')+'='+encodeURIComponent(time.replace(':','')));
    if(revision!==state.planRevision)return;
    const range=target?data.recommend_pickup_range:data.meal_range;
    if(!range){el('meal-plan').textContent=target?'这个时段的本机记录还不够，暂时算不准。':data.message||'当前暂时无法估计。';return;}
    if(target&&range.late<chinaHHMM()){el('meal-plan').textContent='可能赶不上 '+time+'，可以换个时间或门店。';return;}
    el('meal-plan').innerHTML=(target?'建议取号时间':'现在取号，预计入座')+'<strong>'+escapeHTML(range.early+'–'+range.late)+'</strong>'+(target?'本机历史参考':'当前等待参考')+' · 以门店叫号为准。';
  }catch(e){if(revision===state.planRevision)el('meal-plan').textContent=e.message;}
}
let confirmResolve;
function confirmDialog(title,message) {
  if(confirmResolve)confirmResolve(false);
  el('confirm-title').textContent=title;el('confirm-message').textContent=message;
  if(!el('confirm-dialog').open)el('confirm-dialog').showModal();
  return new Promise(resolve=>{confirmResolve=resolve;});
}
function finishConfirm(value){el('confirm-dialog').close();if(confirmResolve){confirmResolve(value);confirmResolve=null;}}
async function loadStatus(){
  state.status=await api('/api/status');
  el('version').textContent='v'+state.status.version+(state.status.edition?' · '+state.status.edition:'');
  el('account-state').textContent=!state.status.has_config?'未连接':state.status.auth_health?.status==='stale'?'连接已过期':'凭证已保存在本机';
  return state.status;
}
function init() {
  initRecordChart();
  document.addEventListener('click',handleDesktopDownload);
  window.runtime?.EventsOn?.('desktop:busy',()=>toast('正在完成操作，请稍候再关闭。'));
  const actions={
    refresh:refreshRecords,'pick-records':()=>openStorePicker('records'),'pick-queue':()=>openStorePicker('queue'),
    'retry-stores':searchStores,'save-stores':savePickedStores,'close-store':()=>{state.searchRevision++;el('store-dialog').close();},
    'toggle-recording':toggleRecording,'refresh-live':loadLive,
    'analyze-store':button=>{el('analysis-store').value=button.dataset.id;el('analysis-date').value='';refreshRecords();el('analysis-title').scrollIntoView({block:'start'});},
    'review-ticket':reviewTicket,'confirm-ticket':confirmTicket,'close-ticket':()=>{if(!state.busy.has('ticket'))el('ticket-dialog').close();},
    'query-ticket':queryTicket,'cancel-ticket':cancelTicket,'connect-account':()=>openAuth(false),'start-auth':startAuth,'close-auth':closeAuth,'import-auth':importAuth,'auth-continue':finishAuth,
    'reset-account':()=>guard('account',async()=>{if(await confirmDialog('断开账号','只删除本机凭证，不删除排队记录，也不取消已取号码。')){await api('/api/auth/reset',{});await loadStatus();toast('已断开');}}),
    'repair-proxy':()=>guard('repair',async()=>{if(await confirmDialog('恢复系统代理','会停止当前认证并恢复本应用修改过的代理。')){await api('/api/repair-proxy',{});toast('代理已恢复');}})
  };
  document.addEventListener('click',event=>{const button=event.target.closest('[data-action]');if(button&&!button.disabled&&actions[button.dataset.action])Promise.resolve(actions[button.dataset.action](button)).catch(e=>toast(e.message));const answer=event.target.closest('[data-answer]');if(answer)finishConfirm(answer.dataset.answer==='yes');});
  window.addEventListener('hashchange',()=>navigatePage(location.hash.slice(1)));
  let resizeTimer;
  window.addEventListener('resize',()=>{clearTimeout(resizeTimer);resizeTimer=setTimeout(()=>{if(state.page==='records'&&state.records)renderAnalysis();},150);});
  el('store-search').addEventListener('input',()=>{state.searchRevision++;clearTimeout(searchTimer);searchTimer=setTimeout(searchStores,250);});
  el('store-results').addEventListener('change',event=>{const input=event.target;if(input.name!=='picked-store')return;if(state.pickerMode==='queue')state.picker.clear();if(input.checked)state.picker.add(input.value);else state.picker.delete(input.value);updatePicked();});
  el('record-interval').addEventListener('change',()=>guard('config',async()=>{await saveRecordConfig();await refreshRecords();toast('记录间隔已保存');}));
  el('autostart').addEventListener('change',changeAutostart);
  for(const id of ['analysis-store','analysis-days','analysis-date-type'])el(id).addEventListener('change',()=>{el('analysis-date').value='';refreshRecords();});
  el('analysis-date').addEventListener('change',refreshRecords);
  el('include-history').addEventListener('change',changeHistory);
  for(const id of ['meal-mode','meal-time','travel-minutes'])el(id).addEventListener('change',()=>{el('meal-fields').hidden=el('meal-mode').value!=='target';loadMealPlan();});
  el('auth-method').addEventListener('change',renderAuthInstructions);
  el('auth-dialog').addEventListener('cancel',event=>{event.preventDefault();closeAuth();});
  el('ticket-dialog').addEventListener('cancel',event=>{if(state.busy.has('ticket'))event.preventDefault();});
  el('confirm-dialog').addEventListener('cancel',event=>{event.preventDefault();finishConfirm(false);});
  refreshRecords();navigatePage(location.hash.slice(1));loadStatus().catch(()=>{el('account-state').textContent='状态暂时读取失败，连接时会重试。';});
  setInterval(()=>{if(!document.hidden&&state.page==='records'&&!state.busy.size&&!el('store-dialog').open)refreshRecords();},30000);
}
