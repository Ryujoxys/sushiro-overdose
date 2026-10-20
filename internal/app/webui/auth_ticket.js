let ticketDraft=null,authReturnIntent='settings',authModeRunning='',authPollTimer=null,authSavedAt='',authPollBusy=false,authSessionRevision=0;
let liveCaptureMode='',liveCaptureTimer=null,liveCaptureRevision=0,liveCapturePolling=false,liveCaptureBaseline='';
const mobileGuideAddresses={},liveCapturePollMS=3000;
const captureStages={preparing_cert:'正在准备证书…',installing_cert_currentuser:'请在系统弹窗中允许安装证书。',installing_cert_localmachine_uac:'请在系统弹窗中授权。',starting_proxy:'正在准备连接…',setting_system_proxy:'正在设置临时代理…',waiting_capture:'请重新打开电脑微信，在寿司郎小程序里浏览门店和我的单据，无需提交订单。',probing:'凭证已收到，正在检查…'};

function ticketStoreID(ticket) { return String(ticket?.monitored_store_id||ticket?.store_id||ticket?.storeId||''); }
function clearTicketView() {
  state.currentTicket=null;state.cancelToken='';state.ticketUncertain=false;state.ticketProgressBase=null;state.ticketForecastRevision=(state.ticketForecastRevision||0)+1;
  el('ticket-result').hidden=true;el('ticket-result').replaceChildren();
  renderTakeTicket();
}
async function reviewTicket() {
  if(!state.selectedStore)throw new Error('先选一家门店。');
  if(state.liveAuthRequired)throw new Error('先连接实时数据，或在小程序里取号后在这里填号码看叫到时间。');
  if(!state.live||state.liveStale)throw new Error('排队信息尚未更新，请先刷新后再取号。');
  if(state.ticketUncertain){toast('上次操作结果还未确认，请先查询已有号码。');return;}
  const options=ticketOptions(el('ticket-adult').value,el('ticket-child').value,el('ticket-table').value);
  await loadStatus();
  ticketDraft={store:state.selectedStore,...options};
  const needsAuth=!state.status.has_config||state.status.auth_health?.status==='stale';
  el('ticket-review').innerHTML=[['门店',storeName(ticketDraft.store)],['人数',options.adult+' 成人 · '+options.child+' 儿童'],['座位',options.table_type==='T'?'桌位':'吧台'],['用餐意向',el('meal-mode').value==='target'?'今天 '+el('meal-time').value:'尽快吃']].map(([label,value])=>'<div><dt>'+escapeHTML(label)+'</dt><dd>'+escapeHTML(value)+'</dd></div>').join('');
  el('ticket-review-hint').textContent=needsAuth?'先连接本人账号，回来后再确认。':el('meal-mode').value==='target'?'现在取号，不是预约 '+el('meal-time').value+'。入座以门店叫号为准。':'确认后立即提交一次取号请求，以门店叫号为准。';
  el('ticket-confirm').textContent=needsAuth?'连接账号':'确认取号';
  el('ticket-confirm').disabled=false;showError('ticket-error',null);
  el('ticket-dialog').showModal();
}
async function confirmTicket() {
  if(!ticketDraft||state.ticketUncertain)return;
  await guard('ticket',async()=>{
    el('ticket-confirm').disabled=true;let submitted=false;
    try{
      await loadStatus();
      if(!state.status.has_config||state.status.auth_health?.status==='stale'){el('ticket-dialog').close();await openAuth('review');return;}
      submitted=true;
      const result=await api('/api/queue/ticket',ticketDraft,{timeout:45000});
      el('ticket-dialog').close();showTicket(result);ticketDraft=null;
    }catch(e){
      showError('ticket-error',e);
      if(submitted){state.ticketUncertain=true;el('ticket-dialog').close();showTicketUncertain(e.message);}
    }finally{el('ticket-confirm').disabled=false;}
  });
}
function showTicketUncertain(message) {
  state.cancelToken='';navigatePage('queue');const box=el('ticket-result');box.hidden=false;
  box.innerHTML='<h2>操作结果还未确认</h2><p class="muted">'+escapeHTML(message)+'</p><p class="caption">先查询已有号码，不要重复操作。也可以在官方小程序核对。</p><button class="button primary" data-action="query-ticket">查询已有号码</button>';
  box.focus();renderTakeTicket();
}
function showTicket(result) {
  state.ticketUncertain=false;navigatePage('queue');const ticket=result.ticket||{},number=ticket.number;
  if(!number){state.ticketUncertain=true;showTicketUncertain('官方没有返回明确号码，请在小程序核对。');return;}
  const id=ticketStoreID(ticket);
  state.currentTicket=ticket;state.cancelToken=result.cancel_token||'';const box=el('ticket-result');box.hidden=false;
  box.innerHTML='<div class="section-heading"><h2>'+escapeHTML(id?storeName(id):'我的排队')+'</h2><span class="badge green">'+(result.recovered?'已有号码':'已取到号')+'</span></div><div class="ticket-body"><div class="ticket-main"><div class="ticket-number">'+escapeHTML(number)+'</div><div id="ticket-progress" class="ticket-progress"></div><p class="caption">以官方小程序和门店叫号为准。</p></div><div class="ticket-side"><div id="ticket-forecast" class="plan">正在读取预计叫到时间…</div><label id="ticket-called-field" class="field travel-field" hidden>小程序里现在叫到几号<input id="ticket-called" type="number" min="1" max="99999" step="1" placeholder="例如 128" inputmode="numeric"></label><label class="field travel-field">路程（分钟，选填）<input id="travel-minutes" type="number" min="0" max="240" step="1" value="'+escapeHTML(state.ticketTravelMinutes??'')+'" placeholder="例如 20" inputmode="numeric"></label></div></div><p id="ticket-result-error" class="notice error" role="alert" hidden></p><div class="actions"><button class="button secondary" data-action="query-ticket">刷新号码</button><button id="cancel-current-ticket" class="button quiet" data-action="cancel-ticket" '+(state.cancelToken?'':'disabled')+'>取消这个号</button></div>'+(state.cancelToken?'':'<p class="caption">取消前请刷新号码；若官方未返回完整票据，请在小程序取消。</p>');
  for(const field of ['travel-minutes','ticket-called'])el(field).addEventListener('change',()=>loadTicketForecast(id,number));box.focus();
  if(id){state.selectedStore=id;loadLive();loadTicketForecast(id,number);}else el('ticket-forecast').textContent='门店尚未确认，请到官方小程序核对。';
}
async function queryTicket() {
  await guard('ticket-query',async()=>{
    try{
      await loadStatus();
      if(!state.status.has_config||state.status.auth_health?.status==='stale'){await openAuth('query');return;}
      const result=await api('/api/queue/ticket/status');if(result.ticket){showTicket({...result,recovered:true});return;}
      clearTicketView();el('ticket-result').hidden=false;
      el('ticket-result').innerHTML='<h2>当前没有查到排队号</h2><p class="muted">如小程序仍有号码，以小程序为准。</p>';
    }catch(e){
      await loadStatus().catch(()=>{});state.cancelToken='';state.currentTicket=null;
      const stale=state.status?.auth_health?.status==='stale';
      el('ticket-result').hidden=false;el('ticket-result').innerHTML='<h2>'+(stale?'连接已过期':'暂时无法查询号码')+'</h2><p class="notice error" role="alert">'+escapeHTML(e.message)+'</p><button class="button primary" data-action="query-ticket">'+(stale?'重新连接后查询':'重新查询')+'</button>';
    }
  });
}
function refreshTicketForecast() {
  const ticket=state.currentTicket,id=ticketStoreID(ticket);
  if(ticket&&id&&el('ticket-forecast'))loadTicketForecast(id,ticket.number);
}
// Progress counts from the first number seen for this ticket, so it only moves forward with the queue.
function renderTicketProgress(number,data) {
  const box=el('ticket-progress');if(!box)return;
  const called=Number(data.current?.called_no)||0,target=Number(String(number).match(/\d+/)?.[0])||0;
  if(!called||!target){box.replaceChildren();return;}
  if(target<called){box.innerHTML='<p class="notice error">当前已叫到 '+called+' 号，这个号可能已过号，请在小程序确认。</p>';return;}
  const key=String(number),base=state.ticketProgressBase?.number===key?state.ticketProgressBase:(state.ticketProgressBase={number:key,called});
  const start=Math.min(base.called,called),percent=Math.round((called-start)/Math.max(1,target-start)*100),remaining=target-called;
  box.innerHTML='<p>当前叫到 <b>'+called+'</b> 号 · '+(remaining?'还差 <b>'+remaining+'</b> 号':'<b>正在叫这个号</b>')+'</p><div class="meter" aria-hidden="true"><span style="width:'+Math.min(100,percent)+'%"></span></div>';
}
// Without live data the ticket card asks for the number the mini program is calling now and
// estimates from this store's usual calling speed; times are always the server's HH:MM fields.
// A number typed on the ticket card was seen at one moment. The 30-second refresh keeps that
// moment instead of treating the number as called just now, or the estimate would slide later on every refresh.
function ticketCalledAt(number,called) {
  const key=String(number)+'|'+called;
  if(state.ticketCalled?.key!==key)state.ticketCalled={key,at:new Date()};
  return state.ticketCalled.at.toISOString().replace(/\.\d+Z$/,'Z');
}
async function loadTicketForecast(id,number) {
  const revision=state.ticketForecastRevision=(state.ticketForecastRevision||0)+1;
  try{
    const target=String(number).match(/\d+/)?.[0];if(!target){el('ticket-forecast').textContent='号码格式暂不支持估计。';return;}
    const raw=el('travel-minutes')?.value??'',travel=Number(raw||0);
    if(!Number.isInteger(travel)||travel<0||travel>240){el('ticket-forecast').textContent='路程请填写 0 到 240 的整数分钟。';return;}
    state.ticketTravelMinutes=raw;
    const field=el('ticket-called-field'),liveUsable=String(state.selectedStore)===String(id)&&!!state.live&&!state.liveStale&&!state.liveAuthRequired;
    if(field&&liveUsable)field.hidden=true;
    const called=field&&!field.hidden?String(el('ticket-called').value??'').trim():'';
    if(called&&(!/^\d+$/.test(called)||Number(called)<1||Number(called)>99999)){el('ticket-forecast').textContent='叫到的号码请填 1 到 99999 之间的整数。';return;}
    const q=new URLSearchParams({store:id,target_no:target,travel_minutes:String(travel)});
    if(called){q.set('called_no',called);q.set('called_at',ticketCalledAt(number,called));}
    const data=await api('/api/queue/advisor?'+q);
    const box=el('ticket-forecast');if(!box||revision!==state.ticketForecastRevision)return;
    renderTicketProgress(number,data);
    if(data.needs_called_no){if(field)field.hidden=false;box.textContent='实时叫号暂时拿不到。在下面填上小程序里现在叫到几号，就能推算这个号几点叫到。';return;}
    const eta=etaHTML(data);
    box.innerHTML=eta?eta+(called&&data.called_at_hhmm?'<p class="caption">'+escapeHTML('按 '+data.called_at_hhmm+' 填的叫到 '+called+' 号推算。')+'</p>':''):escapeHTML(data.message||'记录还不足，暂时算不准这个号码几点叫到。');
  }catch(e){const box=el('ticket-forecast');if(box&&revision===state.ticketForecastRevision)box.textContent='预计时间暂未更新，请以官方号码状态为准。';}
}
async function cancelTicket() {
  await guard('ticket-cancel',async()=>{
    const ticket=state.currentTicket,token=state.cancelToken;
    if(!ticket||!token){toast('请先刷新号码，再确认取消。');return;}
    const id=ticketStoreID(ticket);
    if(!await confirmDialog('取消 '+ticket.number+' 号',(id?storeName(id)+'。':'')+'取消后无法恢复。确认前会重新核对当前号码。'))return;
    if(state.currentTicket!==ticket||state.cancelToken!==token){toast('号码已变化，请刷新后重新确认。');return;}
    el('cancel-current-ticket').disabled=true;
    try{await api('/api/queue/ticket/cancel',{cancel_token:token});clearTicketView();el('ticket-result').hidden=false;el('ticket-result').innerHTML='<h2>已取消排队号</h2>';}
    catch(e){state.cancelToken='';if(!e.status||e.status>=500){state.ticketUncertain=true;renderTakeTicket();}showError('ticket-result-error',new Error(e.message+' 请刷新号码核对，不要重复取消。'));}
  });
}
function renderAuthInstructions() {
  const method=el('auth-method').value;
  el('auth-instructions').textContent=method==='android'?'使用你已配置好的抓包工具，导出本人寿司郎请求并粘贴到下方。这里不会启动代理；无法读取请求时可改用电脑微信。':method==='mobile'?'iPhone 和电脑连接同一 Wi-Fi。按手机页面安装证书、设置代理，再打开小程序；收到凭证后关闭手机代理。':'需要安装证书并临时启用系统代理，只读取寿司郎接口。完成或停止后恢复代理；请在系统弹窗中授权。';
  if(!authModeRunning){el('auth-start').hidden=method==='android';el('auth-stop').hidden=true;}
  if(method==='android')el('auth-import').open=true;
}
async function openAuth(intent='settings') {
  // One capture at a time: a live-data connection still running would be replaced by this one.
  if(liveCaptureMode&&!await stopLiveCapture())throw new Error('实时数据连接还在进行，请先在设置里停止连接。');
  await loadStatus();authReturnIntent=intent===true?'review':intent===false?'settings':intent;authSavedAt=state.status.auth_meta?.captured_at||'';
  authSessionRevision++;
  el('auth-method').value=state.status.platform==='darwin'?'desktop':'mobile';
  el('auth-progress').textContent='';el('mobile-guide').hidden=true;el('auth-continue').hidden=true;el('auth-verify').hidden=true;el('auth-stop').hidden=true;
  el('auth-method').disabled=false;el('auth-start').disabled=false;showError('auth-error',null);
  el('auth-continue').textContent=authReturnIntent==='review'?'返回取号确认':authReturnIntent==='query'?'查询已有号码':'完成连接';
  renderAuthInstructions();
  if(state.status.has_config&&state.status.auth_health?.status!=='ok')showAuthVerificationState(state.status);
  el('auth-dialog').showModal();
}
async function startAuth() {
  await guard('auth-start',async()=>{
    const method=el('auth-method').value;if(method==='android'){renderAuthInstructions();return;}
    el('auth-start').disabled=true;el('auth-method').disabled=true;showError('auth-error',null);
    el('auth-continue').hidden=true;el('auth-verify').hidden=true;authModeRunning=method;authSessionRevision++;
    try{
      const result=await api(method==='mobile'?'/api/mobile-auth/start':'/api/engine/capture',{},{timeout:30000});
      el('auth-start').hidden=true;el('auth-stop').hidden=false;if(method==='mobile')renderMobileGuide(result);await pollAuth();
    }catch(e){
      if(e.status){authModeRunning='';el('auth-method').disabled=false;showError('auth-error',e);renderAuthInstructions();}
      else{el('auth-start').hidden=true;el('auth-stop').hidden=false;el('auth-progress').textContent='启动结果还未确认，正在检查。也可以停止连接。';authPollTimer=setTimeout(pollAuth,1800);}
    }finally{el('auth-start').disabled=false;}
  });
}
// The phone guide (QR code and link) is shared by the account dialog ('mobile') and the
// live-data card ('live'); each place has its own elements and error line.
function mobileGuideIDs(target) { return {box:target+'-guide',qr:target+'-qr',link:target+'-guide-link',address:target+'-address',error:target==='live'?'live-error':'auth-error'}; }
function renderMobileGuide(data,target='mobile') {
  const ids=mobileGuideIDs(target),box=el(ids.box);box.hidden=false;box.replaceChildren();
  let addresses=(data.addresses||[]).filter(a=>{try{const u=new URL(a.url);return u.protocol==='http:'&&!['127.0.0.1','localhost','0.0.0.0'].includes(u.hostname);}catch{return false;}});
  if(!addresses.length&&(data.guide_urls||[]).length){try{const u=new URL(data.guide_urls[0]);if(u.protocol==='http:'&&!['127.0.0.1','localhost','0.0.0.0'].includes(u.hostname))addresses=[{url:u.href,qr_svg:data.qr_svg}];}catch{}}
  mobileGuideAddresses[target]=addresses;
  if(!addresses.length){showError(ids.error,new Error('没有可用的局域网地址，请检查电脑网络或换一种连接方式。'));return;}
  const img=document.createElement('img');img.id=ids.qr;img.alt='手机连接引导二维码';box.append(img);
  if(addresses.length>1){
    const label=document.createElement('label');label.className='field';label.textContent='手机打不开？试试其他电脑地址';const select=document.createElement('select');select.id=ids.address;
    addresses.forEach((address,i)=>{const option=document.createElement('option');option.value=String(i);option.textContent=address.host||new URL(address.url).hostname;select.append(option);});
    select.addEventListener('change',()=>selectMobileAddress(Number(select.value),target));label.append(select);box.append(label);
  }
  const link=document.createElement('a');link.id=ids.link;link.textContent='打开手机连接说明';link.target='_blank';link.rel='noopener noreferrer';box.append(link);
  const p=document.createElement('p');p.className='caption';p.textContent='用 iPhone 浏览器扫码。打不开时检查手机和电脑是否连着同一个 Wi-Fi、电脑防火墙是否拦截；不要关闭整个防火墙，可改用其他连接方式。完成后关闭手机 Wi-Fi 代理。';box.append(p);selectMobileAddress(0,target);
}
function selectMobileAddress(index,target='mobile') {
  const ids=mobileGuideIDs(target),address=(mobileGuideAddresses[target]||[])[index],img=el(ids.qr),link=el(ids.link);if(!address||!img||!link)return;
  try{const url=new URL(address.url);if(url.protocol!=='http:'||['127.0.0.1','localhost','0.0.0.0'].includes(url.hostname))throw new Error('没有可用的局域网地址。');img.hidden=!address.qr_svg;if(address.qr_svg)img.src='data:image/svg+xml;charset=utf-8,'+encodeURIComponent(address.qr_svg);link.href=url.href;}
  catch(e){img.hidden=true;link.hidden=true;showError(ids.error,e);}
}
function showAuthVerificationState(status,message='') {
  el('auth-stop').hidden=true;el('auth-method').disabled=false;el('mobile-guide').hidden=true;const health=status.auth_health?.status;
  el('auth-continue').hidden=health!=='ok';el('auth-verify').hidden=health==='ok';
  el('auth-verify').textContent=health==='stale'?'重新验证':'验证连接';showError('auth-error',null);
  if(health==='ok')el('auth-progress').textContent='连接已验证。'+(el('auth-method').value==='mobile'?'请确认手机 Wi-Fi 代理已关闭。':'');
  else if(health==='stale'){el('auth-progress').textContent='凭证已收到，但连接验证失败。请重新连接或换一种方式。';showError('auth-error',new Error(status.auth_health?.reason||message||'官方接口没有接受当前凭证。'));}
  else el('auth-progress').textContent=(message||'凭证已收到，尚未验证。')+(el('auth-method').value==='mobile'?'先关闭手机 Wi-Fi 代理，再验证连接。':'请点“验证连接”，不会取号。');
  el('auth-start').hidden=health==='ok'||el('auth-method').value==='android';
}
async function pollAuth() {
  if(!el('auth-dialog').open||!authModeRunning||authPollBusy)return;authPollBusy=true;
  const revision=authSessionRevision,mode=authModeRunning;
  try{
    const s=await loadStatus(),engine=s.engine||{};let done=false;
    if(revision!==authSessionRevision||!el('auth-dialog').open)return;
    if(mode==='mobile'){
      const mobile=await api('/api/mobile-auth');if(revision!==authSessionRevision||!el('auth-dialog').open)return;
      if(mobile.active&&el('mobile-guide').hidden)renderMobileGuide(mobile);
      el('auth-progress').textContent=mobile.message||'等待手机连接…';done=!!mobile.saved&&!!s.has_config;
      if(!mobile.active&&!done){authModeRunning='';el('auth-method').disabled=false;el('mobile-guide').hidden=true;renderAuthInstructions();}
    }else{
      el('auth-progress').textContent=engine.status==='error'?engine.message:captureStages[engine.stage]||engine.message||'连接中…';
      done=!!s.has_config&&engine.status==='idle'&&(engine.stage==='done'||(s.auth_meta?.captured_at&&s.auth_meta.captured_at!==authSavedAt));
      if(engine.status==='error'||(engine.status==='idle'&&!done)){authModeRunning='';el('auth-method').disabled=false;renderAuthInstructions();}
    }
    if(done){authModeRunning='';clearTicketView();showAuthVerificationState(s,mode==='desktop'&&s.auth_health?.status!=='ok'?engine.message:'');}
  }catch(e){if(revision===authSessionRevision)showError('auth-error',e);}
  finally{authPollBusy=false;if(authModeRunning&&el('auth-dialog').open)authPollTimer=setTimeout(pollAuth,1800);}
}
async function stopAuth() {
  if(state.busy.has('auth-start')){showError('auth-error',new Error('连接正在启动，请稍等再停止。'));return false;}
  return await guard('auth-stop',async()=>{
    clearTimeout(authPollTimer);const mode=authModeRunning;authSessionRevision++;
    try{
      if(mode){const result=await api(mode==='mobile'?'/api/mobile-auth/stop':'/api/engine/stop',{});if(result.engine?.status==='stopping')throw new Error('代理正在恢复，请稍后再停止。');}
      authModeRunning='';el('auth-method').disabled=false;el('mobile-guide').hidden=true;showError('auth-error',null);renderAuthInstructions();
      if(mode)el('auth-progress').textContent=mode==='mobile'?'连接已停止，请关闭手机 Wi-Fi 代理。':'连接已停止，已恢复本应用修改的代理。';return true;
    }catch(e){showError('auth-error',e);if(authModeRunning)authPollTimer=setTimeout(pollAuth,1800);return false;}
  });
}
async function closeAuth() {
  if(state.busy.has('auth-verify')||state.busy.has('auth-import')){showError('auth-error',new Error('正在完成操作，请稍后关闭。'));return;}
  if(!await stopAuth())return;el('auth-text').value='';el('auth-dialog').close();
}
async function finishAuth() {
  const intent=authReturnIntent;await closeAuth();if(el('auth-dialog').open)return;
  if(intent==='review')await reviewTicket();else if(intent==='query')await queryTicket();else await loadStatus();
}
async function verifyAuth() {
  await guard('auth-verify',async()=>{
    if(authModeRunning){showError('auth-error',new Error('请先停止连接，再验证凭证。'));return;}
    el('auth-verify').disabled=true;showError('auth-error',null);el('auth-progress').textContent='正在只读验证，不会取号…';
    try{const result=await api('/api/auth/verify',{},{timeout:60000});const status=await loadStatus();const valid=result.valid&&status.auth_health?.status==='ok';showAuthVerificationState({...status,auth_health:{...status.auth_health,status:valid?'ok':status.auth_health?.status==='stale'?'stale':'unknown'}});if(!valid)showError('auth-error',new Error([result.message,result.detail].filter(Boolean).join(' ')));}
    catch(e){showError('auth-error',e);el('auth-progress').textContent='暂时无法验证，已保存的凭证未删除。';}
    finally{el('auth-verify').disabled=false;}
  });
}
async function importAuth() {
  await guard('auth-import',async()=>{
    showError('auth-error',null);if(authModeRunning){showError('auth-error',new Error('先点“停止连接”，再导入；已粘贴的内容会保留。'));return;}
    const text=el('auth-text').value.trim();if(!text){showError('auth-error',new Error('先粘贴凭证。'));return;}
    try{const result=await api('/api/auth/import',{text});if(!result.saved){showError('auth-error',new Error('还缺少：'+(result.missing||[]).join('、')));return;}el('auth-text').value='';clearTicketView();showAuthVerificationState(await loadStatus());}
    catch(e){showError('auth-error',e);}
  });
}

// ---- 实时数据连接（spec 018）：用户自己的小程序登录，约 1 小时有效，只存在这台电脑上 ----
// The page only ever sees the connection's state and remaining time, never its header values.
function liveSessionView(s) {
  const seconds=Number(s?.remaining_seconds)||0,left=seconds>=60?'还剩 '+Math.floor(seconds/60)+' 分钟':'还剩不到 1 分钟';
  const until=s?.expires_at?clockHHMM(s.expires_at):'';
  switch(s?.state){
    case 'active':return {label:'已连接，'+left,tone:'green',detail:(until?'约 '+until+' 到期。':'')+'实时叫号、记录和提醒都可用。'};
    case 'expiring':return {label:'快到期，'+left,tone:'amber',detail:'打开小程序刷新一下，再连接一次。'};
    case 'expired':return {label:'已到期',tone:'amber',detail:'实时叫号和记录已停下。打开小程序刷新一下，再连接一次。'};
    default:return {label:'未连接',tone:'',detail:'没连接时，排队页可以填小程序里的叫号来估算叫到时间。'};
  }
}
function renderLiveSession() {
  const s=state.liveSession,busy=!!liveCaptureMode;
  if(s){
    const view=liveSessionView(s);
    el('live-session-state').textContent=view.label;el('live-session-state').className='status '+view.tone;
    el('live-session-detail').textContent=view.detail;
  }
  el('live-disconnect').hidden=!s?.state||s.state==='none';
  // The computer capture changes the system proxy; it is offered where the account capture already uses it.
  el('live-desktop').hidden=!['darwin','windows'].includes(state.status?.platform);
  for(const id of ['live-mobile','live-desktop','live-import'])el(id).disabled=busy;
  el('live-capture-stop').hidden=!busy;
  el('live-capture').hidden=!busy&&!el('live-capture-progress').textContent;
}
async function refreshLiveSession() {
  try{state.liveSession=await api('/api/live-session');renderLiveSession();}
  catch{
    // A failed first read must not keep saying "读取中"; a state read earlier stays as it was.
    if(!state.liveSession){el('live-session-state').textContent='暂时读不到';el('live-session-state').className='status';}
    el('live-session-detail').textContent='连接状态暂时读取失败，请稍后再试。';
  }
  return state.liveSession;
}
function gotoLiveSettings() {
  navigatePage('settings');
  el('live-card').scrollIntoView({block:'start'});el('live-title').focus({preventScroll:true});
}
function liveCaptureNote(text) { el('live-capture-progress').textContent=text;renderLiveSession(); }
function liveCaptureHint(mode) {
  return mode==='mobile'?'用 iPhone 扫码，按页面装证书、设代理，再打开寿司郎小程序看看门店或排队页。连上后这里会自动显示。':'请重新打开电脑微信，在寿司郎小程序里看看门店或排队页。连上后这里会自动显示。';
}
function scheduleLivePoll() { clearTimeout(liveCaptureTimer);if(liveCaptureMode)liveCaptureTimer=setTimeout(pollLiveCapture,liveCapturePollMS); }
// Starts the existing phone or computer capture, then waits for a new connection to be saved.
async function startLiveCapture(mode) {
  if(liveCaptureMode)return;
  await guard('live-capture',async()=>{
    showError('live-error',null);
    const revision=++liveCaptureRevision;
    liveCaptureMode=mode;liveCaptureNote(mode==='mobile'?'正在准备手机连接…':'正在准备证书和临时代理，请在系统弹窗中授权。');
    // Compare against the connection saved right now, not one read minutes ago, so an older save is never taken for this capture's.
    try{const current=await api('/api/live-session');if(revision===liveCaptureRevision)state.liveSession=current;}catch{}
    liveCaptureBaseline=state.liveSession?.captured_at||'';
    try{
      const result=await api(mode==='mobile'?'/api/mobile-auth/start':'/api/engine/capture',{},{timeout:30000});
      if(revision!==liveCaptureRevision)return;
      if(mode==='mobile'){renderMobileGuide(result,'live');liveCaptureNote(liveCaptureHint('mobile'));}
    }catch(e){
      if(revision!==liveCaptureRevision)return;
      // A refused start is final; an unanswered one may still be running, so keep the handle to stop it.
      if(e.status){liveCaptureMode='';liveCaptureNote('');showError('live-error',e);return;}
      liveCaptureNote('启动结果还未确认，正在检查。也可以停止连接。');
    }
    scheduleLivePoll();
  });
}
function isNewLiveConnection(session) { return !!(session?.connected&&session.captured_at&&session.captured_at!==liveCaptureBaseline); }
async function readCaptureSession(revision) {
  const session=await api('/api/live-session');
  if(revision!==liveCaptureRevision)return null;
  state.liveSession=session;return session;
}
async function pollLiveCapture() {
  if(!liveCaptureMode||liveCapturePolling)return;
  liveCapturePolling=true;
  const revision=liveCaptureRevision,mode=liveCaptureMode;
  try{
    let session=await readCaptureSession(revision);if(!session)return;
    if(isNewLiveConnection(session)){await finishLiveCapture(mode);return;}
    let ended='';
    if(mode==='mobile'){
      const mobile=await api('/api/mobile-auth');
      if(revision!==liveCaptureRevision)return;
      if(!mobile.active)ended='手机连接已结束，还没收到实时数据。可以再试一次，或粘贴抓包。';
      else{if(el('live-guide').hidden)renderMobileGuide(mobile,'live');liveCaptureNote(liveCaptureHint('mobile'));}
    }else{
      const engine=(await loadStatus()).engine||{};
      if(revision!==liveCaptureRevision)return;
      if(engine.status==='error')ended=engine.message||'连接没有完成，请重试或粘贴抓包。';
      else if(engine.status==='idle')ended='电脑微信连接已结束，还没收到实时数据。可以再试一次，或粘贴抓包。';
      else liveCaptureNote(engine.stage==='waiting_capture'?liveCaptureHint('desktop'):captureStages[engine.stage]||engine.message||'连接中…');
    }
    if(!ended)return;
    // A capture can save the connection and stop between the two reads above: look once more
    // before saying nothing arrived.
    session=await readCaptureSession(revision);if(!session)return;
    if(isNewLiveConnection(session)){await finishLiveCapture(mode);return;}
    endLiveCapture(ended);
  }catch(e){if(revision===liveCaptureRevision)showError('live-error',e);}
  finally{liveCapturePolling=false;if(revision===liveCaptureRevision)scheduleLivePoll();renderLiveSession();}
}
function endLiveCapture(text) {
  liveCaptureRevision++;clearTimeout(liveCaptureTimer);liveCaptureMode='';el('live-guide').hidden=true;liveCaptureNote(text);
}
async function finishLiveCapture(mode) {
  endLiveCapture('实时数据已连接，正在停止抓包…');
  // Say the proxy is restored only once the stop has answered that it is.
  let note=mode==='mobile'?'实时数据已连接。记得关闭手机 Wi-Fi 代理。':'实时数据已连接，已恢复本应用修改的代理。';
  try{
    const result=await api(mode==='mobile'?'/api/mobile-auth/stop':'/api/engine/stop',{});
    if(result.engine?.status==='stopping')note='实时数据已连接，代理正在恢复。';
  }catch(e){
    note=mode==='mobile'?'实时数据已连接。记得关闭手机 Wi-Fi 代理。':'实时数据已连接。';
    showError('live-error',new Error('已连接，但停止抓包没有完成：'+e.message+(mode==='mobile'?'':' 可以在“遇到问题”里恢复系统代理。')));
  }
  if(!liveCaptureMode)liveCaptureNote(note);
  liveConnected('实时数据已连接');
}
async function stopLiveCapture() {
  const mode=liveCaptureMode;if(!mode)return true;
  if(state.busy.has('live-capture')){showError('live-error',new Error('连接正在启动，请稍等再停止。'));return false;}
  return await guard('live-stop',async()=>{
    try{
      const result=await api(mode==='mobile'?'/api/mobile-auth/stop':'/api/engine/stop',{});
      if(result.engine?.status==='stopping')throw new Error('代理正在恢复，请稍后再停止。');
      showError('live-error',null);endLiveCapture(mode==='mobile'?'连接已停止，请关闭手机 Wi-Fi 代理。':'连接已停止，已恢复本应用修改的代理。');
      return true;
    }catch(e){showError('live-error',e);return false;}
  })??false;
}
// After a new connection: the queue page reads live data again and recording resumes on its next round.
function liveConnected(message) {
  state.liveAuthRequired=false;state.liveStore='';renderLiveSession();toast(message);
  refreshServiceOnly();
  if(state.page==='queue'&&state.selectedStore)loadLive();
}
async function importLiveSession() {
  await guard('live-import',async()=>{
    showError('live-error',null);
    if(liveCaptureMode){showError('live-error',new Error('先点“停止连接”，再粘贴；已粘贴的内容会保留。'));return;}
    const text=String(el('live-text').value??'').trim();
    if(!text){showError('live-error',new Error('先粘贴一条抓包里的寿司郎请求。'));return;}
    el('live-import').disabled=true;
    try{
      const result=await api('/api/live-session/import',{text},{timeout:30000});
      el('live-text').value='';el('live-paste').open=false;state.liveSession=result;liveCaptureNote('');
      liveConnected(result.message||'实时数据已连接');
    }catch(e){showError('live-error',e);}
    finally{el('live-import').disabled=false;renderLiveSession();}
  });
}
async function disconnectLive() {
  await guard('live-clear',async()=>{
    if(!await confirmDialog('断开实时数据','删除这台电脑上保存的小程序登录。已有记录和历史规律都会保留。'))return;
    state.liveSession=await api('/api/live-session/clear',{});state.liveStore='';liveCaptureNote('');
    toast('已断开实时数据');refreshServiceOnly();
  });
}
