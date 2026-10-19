'use strict';

const el = id => document.getElementById(id);
const state = {page:'records', analysisMetric:'called', analysisTime:'', service:null, records:null, status:null, stores:new Map(), selectedStore:'', live:null, liveStore:'', livePending:'', liveStale:false, picker:new Set(), pickerMode:'records', recordsRevision:0, liveRevision:0, searchRevision:0, planRevision:0, busy:new Set()};
const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
function storeName(id) { return state.stores.get(String(id))?.name || state.records?.stores?.find(s=>String(s.id)===String(id))?.name || '门店 '+id; }
function clockTime(value) { if(!value)return '还没有'; const d=new Date(value); return Number.isNaN(d.getTime())?'时间未知':d.toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false}); }
function freshness(value) {
  const age=Date.now()-new Date(value).getTime();
  if(!value||Number.isNaN(age))return clockTime(value);
  if(age<60000)return '刚刚更新';
  return age<3600000?Math.floor(age/60000)+' 分钟前更新':clockTime(value)+' 更新';
}
function chinaDate(date=new Date()) {
  const part=Object.fromEntries(new Intl.DateTimeFormat('en-US',{timeZone:'Asia/Shanghai',year:'numeric',month:'2-digit',day:'2-digit'}).formatToParts(date).map(p=>[p.type,p.value]));
  return part.year+'-'+part.month+'-'+part.day;
}
function chinaHHMM(date=new Date()) { return date.toLocaleTimeString('en-GB',{timeZone:'Asia/Shanghai',hour:'2-digit',minute:'2-digit',hour12:false}); }
function serviceLabel(data) {
  if(!data?.config?.enabled)return [data?.model?.sample_count?'已暂停':'未记录',''];
  if(data.state?.last_error)return ['记录出错','amber'];
  if(data.state?.paused_reason)return ['稍后继续','amber'];
  if(data.state?.running)return ['记录中','green'];
  return [data.background_running?'等待下一次记录':'正在启动','amber'];
}
// Relative time answers "is it fresh?" without reading the clock.
function ago(value) {
  const age=Date.now()-new Date(value).getTime();
  if(!value||Number.isNaN(age))return clockTime(value);
  if(age<60000)return '刚刚';
  if(age<3600000)return Math.floor(age/60000)+' 分钟前';
  return age<86400000?Math.floor(age/3600000)+' 小时前':clockTime(value);
}
function pausedText(reason) {
  if(/取号/.test(reason))return '正在取号，完成后自动继续记录。';
  if(/主流程|手机/.test(reason))return '正在连接账号，完成后自动继续记录。';
  if(/抢号/.test(reason))return '旧版后台还在运行，记录会等它结束后继续。';
  return reason;
}
function storeFreshness(latest,data) {
  const enabled=!!data?.config?.enabled;
  if(!latest)return ['',enabled?'等待第一条记录':'还没有记录'];
  if(!enabled)return ['',clockTime(latest.collected_at)+' 记录'];
  const limit=Math.max(2*(Number(data.state?.interval_seconds)||Number(data.config.interval_minutes)*60||300),600)*1000;
  return Date.now()-new Date(latest.collected_at).getTime()>limit?['amber',ago(latest.collected_at)+'，最近没有新记录']:['green',ago(latest.collected_at)+'记录'];
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
  const link=event.target.closest('a[href]');
  if(!link)return;
  if(link.getAttribute('aria-disabled')==='true'){event.preventDefault();return;}
  const bridge=window.go?.app?.DesktopBridge;
  if(!bridge)return;
  if(link.id==='update-link'){event.preventDefault();guard('update-open',()=>bridge.OpenReleasePage());return;}
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
function setMarkup(id,html) {const node=el(id);if(node.innerHTML!==html)node.innerHTML=html;}
function navigatePage(page) {
  if(!['records','queue','settings'].includes(page))page='records';
  state.page=page;
  document.querySelectorAll('[id^="page-"]').forEach(n=>{n.hidden=n.id!=='page-'+page;});
  document.querySelectorAll('[data-page]').forEach(n=>{if(n.dataset.page===page)n.setAttribute('aria-current','page');else n.removeAttribute('aria-current');});
  if(location.hash!=='#'+page)history.replaceState(null,'','#'+page);
  closeAnalysisStores();
  if(page==='queue'){if(!state.selectedStore)state.selectedStore=String(recordIDs()[0]||'');if(state.selectedStore)loadLive();}
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
function recordsInteractionActive() {return !el('analysis-store-popup').hidden||!!document.activeElement?.closest('.analysis-card input, .analysis-card select, .analysis-card [data-record-point], .analysis-card summary, #record-interval');}
async function refreshRecords({background=false}={}) {
  const revision=++state.recordsRevision;
  const query=recordQuery().toString();
  const changed=state.recordsQuery!==query;
  if(changed){
    setExportLink('export-records',false,'');state.records=null;recordChartSignature='';
    // Refetch keeps the frame: the previous chart stays, dimmed, instead of flashing a placeholder.
    if(state.recordsQuery===undefined)el('record-chart').innerHTML='<p class="empty">正在读取记录…</p>';
    for(const id of ['record-chart','analysis-inspection','analysis-summary','record-table'])el(id).classList.toggle('stale',true);
  }
  el('record-chart').setAttribute('aria-busy','true');
  try {
    const [service,records]=await Promise.all([api('/api/queue/service'),api('/api/records?'+query)]);
    const today=await loadTodayRecords(records);
    if(revision!==state.recordsRevision||(background&&recordsInteractionActive()))return;
    state.service=service;state.records=records;state.recordsToday=today;state.analysisStores=records.available_stores||records.stores||[];
    for(const s of [...(records.available_stores||[]),...(records.stores||[])])state.stores.set(String(s.id),{...state.stores.get(String(s.id)),...s});
    showError('records-error',null);renderRecords();renderService();renderAnalysis();state.recordsQuery=recordQuery().toString();
  }catch(e){if(revision===state.recordsRevision){showError('records-error',new Error('未更新：'+e.message));if(!state.records){el('record-chart').innerHTML='<p class="empty">记录暂时读取失败，请刷新重试。</p>';el('record-table').innerHTML='';el('analysis-inspection').hidden=true;el('analysis-summary').hidden=true;el('analysis-source').textContent='';}}}
  finally{if(revision===state.recordsRevision){el('record-chart').removeAttribute('aria-busy');for(const id of ['record-chart','analysis-inspection','analysis-summary','record-table'])el(id).classList.toggle('stale',false);}}
}
function renderSteps() {
  const ids=recordIDs(),samples=Number(state.service.model?.sample_count)||0,steps=el('record-steps');
  steps.hidden=samples>0||(ids.length>0&&!!state.service.config?.enabled);
  el('record-overview').hidden=samples===0;
  el('pick-records').className='button small '+(ids.length?'secondary':'primary');
  if(steps.hidden)return;
  const done={stores:ids.length>0,start:!!state.service.config?.enabled,data:false};
  let current='';
  steps.querySelectorAll('li').forEach(item=>{
    const step=item.dataset.step,finished=done[step];
    if(!finished&&!current)current=step;
    item.classList.toggle('done',finished);
    if(step===current)item.setAttribute('aria-current','step');else item.removeAttribute('aria-current');
  });
}
// Today's own snapshots are drawn over the multi-day pattern so people can compare with a usual day.
async function loadTodayRecords(records) {
  const today=chinaDate(),store=el('analysis-store').value||String(records.selected_store||'');
  if(!store||el('analysis-date').value||el('analysis-date-type').value!=='all'||!(records.dates||[]).includes(today))return null;
  const q=recordQuery();q.set('store',store);q.set('date',today);
  try{const data=await api('/api/records?'+q);return {...data,date:today};}catch{return null;}
}
function renderRecords() {
  const model=state.service.model||{};
  renderSteps();
  setMarkup('record-overview','已选 '+recordIDs().length+' 家店 · 共 '+(model.sample_count||0)+' 条 · 记了 '+(model.day_count||0)+' 天');
  syncAnalysisStore();
  const charted=el('analysis-store').value;
  const rows=recordIDs().map(id=>{
    const s=state.records.stores.find(row=>String(row.id)===id),latest=s?.latest,[tone,fresh]=storeFreshness(latest,state.service),current=id===charted;
    return '<div class="store-row"'+(current?' aria-current="true"':'')+'><div><button class="store-name" data-action="analyze-store" data-id="'+escapeHTML(id)+'">'+escapeHTML(storeName(id))+'<span aria-hidden="true"> ›</span></button><p class="store-fresh"><span class="dot '+tone+'" aria-hidden="true"></span>'+escapeHTML(fresh+(s?' · 共 '+s.samples+' 条':''))+(current?'<span class="tag">图表中</span>':'')+'</p></div>'+(latest?'<p class="store-last">'+escapeHTML(storeLatestText(latest))+'</p>':'')+'</div>';
  });
  setMarkup('record-stores',rows.join('')||'<div class="empty"><strong>先选常去的几家店</strong>不用连接账号，也能开始记录。</div>');
  el('data-path').textContent=state.service.data_path||'~/.sushiro/';
  setExportLink('backup-records',Number(state.records.total_record_count)>0,'/api/records/export?days=all');
  el('backup-hint').textContent=state.records.total_record_count?'全部 '+state.records.total_record_count+' 条个人记录，不含内置历史和账号凭证。':'还没有个人记录，开始记录后即可备份。';
}
// The last snapshot is history, not live; it reads as one small line.
function storeLatestText(latest) {
  if(latest.store_status!=='OPEN')return '上次：未营业';
  const called=Number(latest.display_called_no)||0,wait='等约 '+(Number(latest.wait_minutes)||0)+' 分钟';
  return '上次：'+(called?'叫到 '+called+' 号 · ':'')+wait;
}
function renderService() {
  const data=state.service,cfg=data.config,auto=data.autostart||{},ids=recordIDs(),[label,tone]=serviceLabel(data);
  el('service-status').textContent=label;el('service-status').className='status '+tone;
  el('service-summary').textContent=!ids.length?'还没选门店。':cfg.enabled?'每 '+(cfg.interval_minutes||5)+' 分钟记一次 · '+(data.state.last_at?'上次记录：'+ago(data.state.last_at):'等待第一条记录'):data.model?.sample_count?'已暂停，已有的记录都会保留。':'已选 '+ids.length+' 家店，尚未开始。';
  const interval=el('record-interval');
  if(![...interval.options].some(o=>Number(o.value)===cfg.interval_minutes))interval.add(new Option('每 '+cfg.interval_minutes+' 分钟',cfg.interval_minutes));
  if(document.activeElement!==interval)interval.value=cfg.interval_minutes||5;
  el('record-toggle').textContent=cfg.enabled?'暂停记录':'开始记录';el('record-toggle').className='button '+(cfg.enabled?'secondary':'primary');el('record-toggle').hidden=!ids.length;el('record-toggle').disabled=!ids.length||state.busy.has('service');
  el('autostart').checked=!!auto.enabled;el('autostart').disabled=auto.supported===false||(!ids.length&&!auto.enabled)||state.busy.has('service');
  el('autostart').title=auto.error||'单独开启，不会随开始记录自动启用';
  el('repair-autostart').hidden=!auto.needs_update;el('repair-autostart').disabled=state.busy.has('service');
  el('record-settings').hidden=!ids.length&&!auto.enabled&&!auto.needs_update;
  // Only exceptions get a line here; the normal state is already in the status line.
  const detail=data.state.last_error?'上次没记上：'+data.state.last_error+'。会自动重试。':(cfg.enabled&&data.state.paused_reason?pausedText(data.state.paused_reason):'')||(auto.needs_update?'程序位置变了，请点“更新自启动位置”。':'')||auto.error||(cfg.enabled&&!data.background_running?'后台没有运行，请先保持这个窗口打开。':'');
  el('service-detail').textContent=detail;el('service-detail').hidden=!detail;el('service-detail').className='notice'+(data.state.last_error?' warn':'');
}
function setExportLink(id,enabled,path) {
  const link=el(id);link.setAttribute('aria-disabled',String(!enabled));
  if(enabled){link.href=path;link.removeAttribute('tabindex');}else{link.removeAttribute('href');link.setAttribute('tabindex','-1');}
}
function renderAnalysis() {
  const data=state.records,id=el('analysis-store').value,called=state.analysisMetric!=='wait';
  if(!data)return;
  setExportLink('export-records',Number(data.export_record_count)>0,'/api/records/export?'+recordQuery());
  el('export-hint').textContent=data.export_record_count?data.export_record_count+' 条自己的记录，不含内置历史':'当前筛选下没有自己的记录';
  el('include-history').checked=!!data.history?.included;
  el('include-history').disabled=state.busy.has('history');
  const span=data.history?.cutoff_at?data.history.first_at.slice(0,10)+' 至 '+data.history.cutoff_at.slice(0,10):'';
  el('history-meta').textContent=data.history?.error||(!span?'未附带内置历史':data.history.included?'内置历史：'+span+' 的公开数据，不再更新，不会导出。':'已关闭内置历史（'+span+'），只看自己的记录。');
  const dateSelect=el('analysis-date'),selectedDate=dateSelect.value;
  const dates=[...new Set([...(data.dates||[]),...(selectedDate?[selectedDate]:[])])].sort().reverse();
  setMarkup('analysis-date','<option value="">不选</option>'+dates.map(date=>'<option value="'+escapeHTML(date)+'">'+escapeHTML(date)+'</option>').join(''));
  dateSelect.value=selectedDate;
  el('analysis-source').textContent=analysisSourceText(data,id,called,selectedDate);
  const summary=id&&!selectedDate?recordSummary(data):null;
  el('analysis-summary').hidden=!summary;
  setMarkup('analysis-summary',summary?'<p>'+summary.text+'</p>'+(summary.note?'<p class="caption">'+escapeHTML(summary.note)+'</p>':''):'');
  renderRecordChart(data);
}
// Says how much data the chart rests on, and how much of it is the user's own.
function analysisSourceText(data,id,called,date) {
  if(!id)return '选择门店查看。';
  const own=Number(called?data.called_local_samples:data.local_samples)||0,history=Number(called?data.called_history_samples:data.history_samples)||0,days=Number(called?data.called_days:data.days)||0;
  if(!own&&!history)return '这个筛选下还没有数据';
  const pct=Math.round(own/(own+history)*100);
  return (date?date+' 一天':days+' 天')+'的数据 · '+(!history?'全是自己的记录':!own?'全部来自内置历史':pct<1?'自己的记录不到 1%':'自己的记录占 '+pct+'%')+(!date&&days<3?' · 数据较少':'');
}
async function changeHistory() {
  const included=el('include-history').checked;
  await guard('history',async()=>{
    state.recordsRevision++;
    el('include-history').disabled=true;
    try { await api('/api/records/settings',{include_history:included}); }
    finally { await refreshRecords(); }
  });
  el('include-history').disabled=false;
}
async function saveRecordConfig(ids=recordIDs()) {
  if(!state.service)throw new Error('请先刷新记录状态。');
  await api('/api/queue/baseline',{...state.service.config,enabled:ids.length?state.service.config.enabled:false,store_ids:ids,use_preference_stores:false,interval_minutes:Number(el('record-interval').value)});
}
async function toggleRecording() {
  await guard('service',async()=>{
    el('record-toggle').disabled=true;
    const pause=state.service.config.enabled;
    try{await saveRecordConfig();await api('/api/queue/service',{action:pause?'pause':'start'},{timeout:20000});toast(pause?'已暂停记录':'记录已开启');}
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
async function repairAutostart() {
  if(!await confirmDialog('更新自启动位置','将使用当前程序启动后台记录，不会立即开始记录。'))return;
  await guard('service',async()=>{try{await api('/api/queue/service',{action:'repair_autostart'},{timeout:20000});toast('自启动位置已更新');}finally{await refreshRecords();}});
  if(state.service)renderService();
}
async function quitApp() {
  if(!await confirmDialog('退出应用及后台记录','记录会停止，数据和自启动设置保留。下次登录电脑仍按原设置启动。'))return;
  await guard('quit',async()=>{
    const bridge=window.go?.app?.DesktopBridge;
    if(bridge){await bridge.Quit();return;}
    await api('/api/app/quit',{}, {timeout:20000});
    state.quitting=true;toast('已退出，可以关闭这个页面。');
    document.querySelectorAll('button,input,select,textarea').forEach(node=>{node.disabled=true;});
  });
}
// Restore is additive: the server skips rows already recorded and never rewrites the file.
async function restoreRecords() {
  const bridge=window.go?.app?.DesktopBridge;
  if(!bridge){el('restore-file').value='';el('restore-file').click();return;}
  await guard('restore',async()=>{
    const response=await bridge.ImportRecords();
    if(response?.status)finishRestore(response.status,JSON.parse(response.body||'{}'));
  });
}
async function uploadRestoreFile() {
  const file=el('restore-file').files?.[0];
  if(!file)return;
  await guard('restore',async()=>{
    el('restore-records').disabled=true;
    try{
      const response=await fetch('/api/records/import',{method:'POST',headers:{Accept:'application/json','Content-Type':'application/x-ndjson','X-Sushiro-CSRF':document.querySelector('meta[name="sushiro-csrf"]').content},body:file,cache:'no-store'});
      finishRestore(response.status,await response.json().catch(()=>({})));
    }finally{el('restore-records').disabled=false;el('restore-file').value='';}
  });
}
function finishRestore(status,data) {
  if(status<200||status>=300||data.error)throw new Error(data.error||'恢复失败，请重试。');
  toast(data.imported?'已恢复 '+data.imported+' 条记录'+(data.duplicates?'，跳过 '+data.duplicates+' 条已有记录':''):'备份里的记录都已存在，没有新增。');
  refreshRecords();
}
let searchTimer;
function openStorePicker(mode) {
  state.pickerMode=mode;state.picker=new Set(mode==='records'?recordIDs():state.selectedStore?[state.selectedStore]:[]);
  el('store-dialog-title').textContent=mode==='records'?'选择记录门店':'今天去哪家店';
  el('store-save').textContent=mode==='records'?'保存门店':'选这家店';
  showError('store-error',null);
  el('store-search').value='';el('store-dialog').showModal();updatePicked();searchStores();
}
function updatePicked(){
  const records=state.pickerMode==='records',saving=state.busy.has('stores');
  el('store-picked').textContent='已选 '+state.picker.size+' 家';
  el('store-save').disabled=saving||(!records&&!state.picker.size);
  el('store-search').disabled=saving;
  el('store-selection').hidden=!records;
  setMarkup('store-selected',[...state.picker].map(id=>'<button class="selected-store" data-action="remove-picked-store" data-id="'+escapeHTML(id)+'" aria-label="移除 '+escapeHTML(storeName(id))+'"'+(saving?' disabled':'')+'><span>'+escapeHTML(storeName(id))+'</span><span aria-hidden="true">×</span></button>').join(''));
  el('store-selection-hint').textContent=state.picker.size?'移除门店不会删除记录。':recordIDs().length?'保存后暂停记录，已有数据保留。':'先搜索常去的门店。';
  el('store-results').querySelectorAll('input[name="picked-store"]').forEach(input=>{input.checked=state.picker.has(input.value);input.disabled=saving;});
}
function removePickedStore(id){
  if(state.pickerMode!=='records'||state.busy.has('stores'))return;
  const index=[...state.picker].indexOf(id);
  state.picker.delete(id);updatePicked();
  const remaining=el('store-selected').querySelectorAll('button');
  (remaining[Math.max(0,Math.min(index,remaining.length-1))]||el('store-search')).focus();
}
function closeStorePicker(){
  if(state.busy.has('stores'))return;
  state.searchRevision++;clearTimeout(searchTimer);el('store-dialog').close();
}
async function searchStores() {
  const revision=++state.searchRevision,q=el('store-search').value.trim();
  el('store-results').innerHTML='<p class="empty">正在找门店…</p>';
  try{
    // Without a keyword, chosen and usual stores come first so they need no search.
    const mine=q?[]:[...new Set([...state.picker,...recordIDs()])];
    const [data,usual]=await Promise.all([api('/api/queue/stores?limit=100&q='+encodeURIComponent(q)),mine.length?api('/api/queue/stores?limit=50&stores='+encodeURIComponent(mine.join(','))).catch(()=>({stores:[]})):{stores:[]}]);
    if(revision!==state.searchRevision)return;
    const seen=new Set(),list=[...(usual.stores||[]),...(data.stores||[])].filter(s=>!seen.has(String(s.id))&&seen.add(String(s.id)));
    for(const store of list)state.stores.set(String(store.id),store);
    el('store-results').innerHTML=list.map(s=>'<label class="picker-row"><input type="'+(state.pickerMode==='records'?'checkbox':'radio')+'" name="picked-store" value="'+escapeHTML(s.id)+'" '+(state.picker.has(String(s.id))?'checked':'')+'><span><b>'+escapeHTML(s.name)+'</b><small>'+escapeHTML([s.name_kana||s.city,s.address].filter(Boolean).join(' · '))+'</small></span></label>').join('')||'<p class="empty">没有找到，试试城市或其他店名。</p>';
    updatePicked();
  }catch(e){if(revision===state.searchRevision)el('store-results').innerHTML='<p class="empty">'+escapeHTML(e.message)+'</p><button class="button secondary" data-action="retry-stores">重试</button>';}
}
async function savePickedStores() {
  await guard('stores',async()=>{
    const ids=[...state.picker];if(state.pickerMode!=='records'&&!ids.length)return;
    showError('store-error',null);updatePicked();
    try{
      if(state.pickerMode==='records'){await saveRecordConfig(ids);if(ids.length)syncAnalysisStore(ids[0]);el('analysis-date').value='';await refreshRecords();toast(!ids.length?'已暂停记录，历史数据保留':state.service?.config.enabled?'门店已保存':'门店已保存，点“开始记录”即可');}
      else{state.selectedStore=ids[0];loadLive();}
      state.searchRevision++;clearTimeout(searchTimer);el('store-dialog').close();
    }catch(error){showError('store-error',error);}
  });
  updatePicked();
}
async function loadLive() {
  const id=state.selectedStore;
  if(!id||state.livePending===id)return;
  const revision=++state.liveRevision;state.livePending=id;
  state.planRevision++;
  const keep=state.liveStore===id&&state.live;
  el('take-ticket').disabled=true;
  if(!keep){state.live=null;el('live-store').innerHTML='<p class="empty">正在获取排队…</p>';el('meal-plan').textContent='正在读取预计时间…';}
  el('live-store').setAttribute('aria-busy','true');
  try{
    const panel=await api('/api/queue/live?store='+encodeURIComponent(id));
    if(revision!==state.liveRevision)return;
    state.live=panel;state.liveStore=id;state.liveStale=false;state.stores.set(id,{...state.stores.get(id),name:panel.store_name});
    renderLive();
    el('take-ticket').disabled=!panel.online_open||!!state.ticketUncertain;await loadMealPlan();
  }catch(e){if(revision===state.liveRevision){state.liveStale=true;if(keep)renderLive(e.message);else el('live-store').innerHTML='<p class="empty">'+escapeHTML(e.message)+'</p><button class="button secondary" data-action="refresh-live">重试</button>';el('meal-plan').textContent='实时数据未更新，请重试。';}}
  finally{if(revision===state.liveRevision){state.livePending='';el('live-store').removeAttribute('aria-busy');}}
}
function renderLive(error='') {
  const panel=state.live,open=panel.store_status==='OPEN',stale=state.liveStale;
  const restoreFocus=!!document.activeElement?.closest('#live-store [data-action="refresh-live"]');
  setMarkup('live-store','<div class="live-title"><h3>'+escapeHTML(panel.store_name)+'</h3><p>'+escapeHTML(stale?'上次读取的排队状态':(open?'营业中':'当前未营业')+' · '+(panel.online_open?'可线上取号':'线上取号暂未开放'))+'</p></div><div class="live-numbers"><div><strong>'+escapeHTML(panel.called_no>0?panel.called_no:'—')+'</strong><span>'+(stale?'上次叫到':'当前叫到')+'</span></div><div><strong>'+escapeHTML(open?panel.wait_groups:'—')+'</strong><span>在等桌数</span></div><div><strong>'+escapeHTML(open?panel.server_wait_minutes:'—')+'</strong><span>官方等待 / 分钟</span></div></div>'+(stale?'<p class="notice error" role="status">未更新：'+escapeHTML(error)+'</p>':'')+'<div class="live-foot"><p class="caption">官方公开数据 · '+escapeHTML(freshness(panel.observed_at))+'</p><div class="actions"><button class="button quiet small" data-action="live-history">看历史规律</button><button class="button secondary small" data-action="refresh-live">刷新排队</button></div></div>');
  if(restoreFocus)el('live-store').querySelector('[data-action="refresh-live"]')?.focus();
}
async function loadMealPlan() {
  const revision=++state.planRevision;
  if(!state.selectedStore||!state.live||state.liveStale)return;
  const target=el('meal-mode').value==='target',time=target?el('meal-time').value:chinaHHMM();
  el('meal-fields').hidden=!target;
  if(target&&(!time||time<=chinaHHMM())){el('meal-plan').textContent='请选择今天稍后的时间。';return;}
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
  el('account-state').textContent=!state.status.has_config?'未连接':state.status.auth_health?.status==='stale'?'连接需重新验证':state.status.auth_health?.status==='ok'?'已连接':'凭证已保存，尚未验证';
  return state.status;
}
async function checkUpdate() {
  await guard('update',async()=>{
    const button=el('check-update'),box=el('update-state');
    button.disabled=true;box.textContent='正在检查…';el('update-link').hidden=true;
    try{
      const info=await api('/api/update',undefined,{timeout:15000});
      if(info.update_available){box.textContent='有新版本 '+info.latest_version+'，当前 v'+info.current_version+'。';el('update-link').hidden=false;}
      else if(info.error)box.textContent='暂时无法连接 GitHub，请稍后重试。';
      else if(!info.latest_version)box.textContent='开发版本，不检查更新。';
      else box.textContent='已是最新版本 v'+info.current_version+'。';
    }catch(e){box.textContent='暂时无法检查：'+e.message;}
    finally{button.disabled=false;}
  });
}
function init() {
  initRecordChart();
  initAnalysisStorePicker();
  document.addEventListener('click',handleDesktopDownload);
  window.runtime?.EventsOn?.('desktop:busy',()=>toast('正在完成操作，请稍候再关闭。'));
  const actions={
    refresh:refreshRecords,'pick-records':()=>openStorePicker('records'),'pick-queue':()=>openStorePicker('queue'),
    'retry-stores':searchStores,'save-stores':savePickedStores,'close-store':closeStorePicker,'remove-picked-store':button=>removePickedStore(button.dataset.id),
    'toggle-recording':toggleRecording,'check-update':checkUpdate,'restore-records':restoreRecords,'repair-autostart':repairAutostart,'quit-app':quitApp,'refresh-live':loadLive,
    'analyze-store':button=>{selectAnalysisStore(button.dataset.id);el('analysis-title').scrollIntoView({block:'start'});},
    'live-history':()=>{const id=state.selectedStore;location.hash='#records';selectAnalysisStore(id).then(()=>el('analysis-title').scrollIntoView({block:'start'}));},
    'review-ticket':reviewTicket,'confirm-ticket':confirmTicket,'close-ticket':()=>{if(!state.busy.has('ticket'))el('ticket-dialog').close();},
    'query-ticket':queryTicket,'cancel-ticket':cancelTicket,'connect-account':()=>openAuth('settings'),'start-auth':startAuth,'stop-auth':stopAuth,'verify-auth':verifyAuth,'close-auth':closeAuth,'import-auth':importAuth,'auth-continue':finishAuth,
    'reset-account':()=>guard('account',async()=>{if(await confirmDialog('断开账号','只删除本机凭证，不删除排队记录，也不取消已取号码。')){await api('/api/auth/reset',{});clearTicketView();await loadStatus();toast('已断开');}}),
    'repair-proxy':()=>guard('repair',async()=>{if(await confirmDialog('恢复系统代理','会停止当前认证并恢复本应用修改过的代理。')){await api('/api/repair-proxy',{});toast('代理已恢复');}})
  };
  document.addEventListener('click',event=>{const button=event.target.closest('[data-action]');if(button&&!button.disabled&&actions[button.dataset.action])Promise.resolve(actions[button.dataset.action](button)).catch(e=>toast(e.message));const answer=event.target.closest('[data-answer]');if(answer)finishConfirm(answer.dataset.answer==='yes');});
  window.addEventListener('hashchange',()=>navigatePage(location.hash.slice(1)));
  let resizeTimer;
  window.addEventListener('resize',()=>{clearTimeout(resizeTimer);resizeTimer=setTimeout(()=>{if(state.page==='records'&&state.records)renderAnalysis();},150);});
  el('store-search').addEventListener('input',()=>{state.searchRevision++;clearTimeout(searchTimer);searchTimer=setTimeout(searchStores,250);});
  el('store-dialog').addEventListener('cancel',event=>{event.preventDefault();closeStorePicker();});
  el('store-results').addEventListener('change',event=>{const input=event.target;if(input.name!=='picked-store')return;if(state.pickerMode==='queue')state.picker.clear();if(input.checked)state.picker.add(input.value);else state.picker.delete(input.value);updatePicked();});
  el('record-interval').addEventListener('change',()=>guard('config',async()=>{await saveRecordConfig();await refreshRecords();toast('记录间隔已保存');}));
  el('autostart').addEventListener('change',changeAutostart);
  el('restore-file').addEventListener('change',uploadRestoreFile);
  for(const id of ['analysis-store','analysis-days','analysis-date-type'])el(id).addEventListener('change',()=>{el('analysis-date').value='';refreshRecords();});
  el('analysis-date').addEventListener('change',refreshRecords);
  el('include-history').addEventListener('change',changeHistory);
  for(const id of ['meal-mode','meal-time'])el(id).addEventListener('change',()=>{el('meal-fields').hidden=el('meal-mode').value!=='target';loadMealPlan();});
  el('auth-method').addEventListener('change',renderAuthInstructions);
  el('auth-dialog').addEventListener('cancel',event=>{event.preventDefault();closeAuth();});
  el('ticket-dialog').addEventListener('cancel',event=>{if(state.busy.has('ticket'))event.preventDefault();});
  el('confirm-dialog').addEventListener('cancel',event=>{event.preventDefault();finishConfirm(false);});
  refreshRecords();navigatePage(location.hash.slice(1));loadStatus().catch(()=>{el('account-state').textContent='状态暂时读取失败，连接时会重试。';});
  document.addEventListener('visibilitychange',()=>{if(!document.hidden)refreshVisiblePage();});
  setInterval(refreshVisiblePage,30000);
}
function refreshVisiblePage() {
  if(document.hidden||state.quitting||state.busy.size||document.querySelector('dialog[open]'))return;
  if(state.page==='queue'){loadLive();refreshTicketForecast();}
  if(state.page==='records'&&!recordsInteractionActive())refreshRecords({background:true});
}
