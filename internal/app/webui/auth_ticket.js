let ticketDraft=null,authReturnIntent='settings',authModeRunning='',authPollTimer=null,authSavedAt='',authPollBusy=false,authMobileAddresses=[],authSessionRevision=0;

function clearTicketView() {
  state.currentTicket=null;state.cancelToken='';state.ticketUncertain=false;state.ticketForecastRevision=(state.ticketForecastRevision||0)+1;
  el('ticket-result').hidden=true;el('ticket-result').replaceChildren();
  el('take-ticket').disabled=!state.live?.online_open||!!state.liveStale;
}
async function reviewTicket() {
  if(!state.selectedStore||!state.live)throw new Error('先选一家门店。');
  if(state.liveStale)throw new Error('排队信息尚未更新，请先刷新后再取号。');
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
  box.focus();el('take-ticket').disabled=true;
}
function showTicket(result) {
  state.ticketUncertain=false;navigatePage('queue');const ticket=result.ticket||{},number=ticket.number;
  if(!number){state.ticketUncertain=true;showTicketUncertain('官方没有返回明确号码，请在小程序核对。');return;}
  const id=String(ticket.monitored_store_id||ticket.store_id||ticket.storeId||'');
  state.currentTicket=ticket;state.cancelToken=result.cancel_token||'';const box=el('ticket-result');box.hidden=false;
  box.innerHTML='<div class="section-heading"><h2>'+escapeHTML(id?storeName(id):'我的排队')+'</h2><span class="badge green">'+(result.recovered?'已有号码':'已取到号')+'</span></div><div class="ticket-number">'+escapeHTML(number)+'</div><p class="muted">以官方小程序和门店叫号为准。</p><label class="field">路程（分钟，选填）<input id="travel-minutes" type="number" min="0" max="240" step="1" value="'+escapeHTML(state.ticketTravelMinutes??'')+'" placeholder="例如 20" inputmode="numeric"></label><div id="ticket-forecast" class="plan">正在读取预计入座时间…</div><p id="ticket-result-error" class="notice error" role="alert" hidden></p><div class="actions"><button class="button secondary" data-action="query-ticket">刷新号码</button><button id="cancel-current-ticket" class="button quiet" data-action="cancel-ticket" '+(state.cancelToken?'':'disabled')+'>取消这个号</button></div>'+(state.cancelToken?'':'<p class="caption">取消前请刷新号码；若官方未返回完整票据，请在小程序取消。</p>');
  el('travel-minutes').addEventListener('change',()=>loadTicketForecast(id,number));box.focus();
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
async function loadTicketForecast(id,number) {
  const revision=state.ticketForecastRevision=(state.ticketForecastRevision||0)+1;
  try{
    const target=String(number).match(/\d+/)?.[0];if(!target){el('ticket-forecast').textContent='号码格式暂不支持估计。';return;}
    const raw=el('travel-minutes')?.value??'',travel=Number(raw||0);
    if(!Number.isInteger(travel)||travel<0||travel>240){el('ticket-forecast').textContent='路程请填写 0 到 240 的整数分钟。';return;}
    state.ticketTravelMinutes=raw;
    const data=await api('/api/queue/advisor?store='+encodeURIComponent(id)+'&target_no='+target+'&travel_minutes='+travel);
    const box=el('ticket-forecast');if(!box||revision!==state.ticketForecastRevision)return;
    const eta=data.eta||{},window=eta.wait_minutes_range,range=eta.estimated_called_at_range;
    box.textContent='记录还不足，暂时算不准这个号码的入座时间。';
    if(range)box.innerHTML='预计入座<strong>'+escapeHTML(range.early+'–'+range.late)+'</strong>'+(window?'还需 '+window.low+'–'+window.high+' 分钟。':'')+'以门店叫号为准。'+(eta.arrival_suggestion?'<p>'+escapeHTML(eta.arrival_suggestion)+'</p>':'');
  }catch(e){const box=el('ticket-forecast');if(box&&revision===state.ticketForecastRevision)box.textContent='预计时间暂未更新，请以官方号码状态为准。';}
}
async function cancelTicket() {
  await guard('ticket-cancel',async()=>{
    const ticket=state.currentTicket,token=state.cancelToken;
    if(!ticket||!token){toast('请先刷新号码，再确认取消。');return;}
    const id=String(ticket.monitored_store_id||ticket.store_id||ticket.storeId||'');
    if(!await confirmDialog('取消 '+ticket.number+' 号',(id?storeName(id)+'。':'')+'取消后无法恢复。确认前会重新核对当前号码。'))return;
    if(state.currentTicket!==ticket||state.cancelToken!==token){toast('号码已变化，请刷新后重新确认。');return;}
    el('cancel-current-ticket').disabled=true;
    try{await api('/api/queue/ticket/cancel',{cancel_token:token});clearTicketView();el('ticket-result').hidden=false;el('ticket-result').innerHTML='<h2>已取消排队号</h2>';}
    catch(e){state.cancelToken='';if(!e.status||e.status>=500){state.ticketUncertain=true;el('take-ticket').disabled=true;}showError('ticket-result-error',new Error(e.message+' 请刷新号码核对，不要重复取消。'));}
  });
}
function renderAuthInstructions() {
  const method=el('auth-method').value;
  el('auth-instructions').textContent=method==='android'?'使用你已配置好的抓包工具，导出本人寿司郎请求并粘贴到下方。这里不会启动代理；无法读取请求时可改用电脑微信。':method==='mobile'?'iPhone 和电脑连接同一 Wi-Fi。按手机页面安装证书、设置代理，再打开小程序；收到凭证后关闭手机代理。':'需要安装证书并临时启用系统代理，只读取寿司郎接口。完成或停止后恢复代理；请在系统弹窗中授权。';
  if(!authModeRunning){el('auth-start').hidden=method==='android';el('auth-stop').hidden=true;}
  if(method==='android')el('auth-import').open=true;
}
async function openAuth(intent='settings') {
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
function renderMobileGuide(data) {
  const box=el('mobile-guide');box.hidden=false;box.replaceChildren();
  authMobileAddresses=(data.addresses||[]).filter(a=>{try{const u=new URL(a.url);return u.protocol==='http:'&&!['127.0.0.1','localhost','0.0.0.0'].includes(u.hostname);}catch{return false;}});
  if(!authMobileAddresses.length&&(data.guide_urls||[]).length){try{const u=new URL(data.guide_urls[0]);if(u.protocol==='http:'&&!['127.0.0.1','localhost','0.0.0.0'].includes(u.hostname))authMobileAddresses=[{url:u.href,qr_svg:data.qr_svg}];}catch{}}
  if(!authMobileAddresses.length){showError('auth-error',new Error('没有可用的局域网地址，请检查电脑网络或换一种连接方式。'));return;}
  const img=document.createElement('img');img.id='mobile-qr';img.alt='手机连接引导二维码';box.append(img);
  if(authMobileAddresses.length>1){
    const label=document.createElement('label');label.className='field';label.textContent='手机打不开？试试其他电脑地址';const select=document.createElement('select');select.id='mobile-address';
    authMobileAddresses.forEach((address,i)=>{const option=document.createElement('option');option.value=String(i);option.textContent=address.host||new URL(address.url).hostname;select.append(option);});
    select.addEventListener('change',()=>selectMobileAddress(Number(select.value)));label.append(select);box.append(label);
  }
  const link=document.createElement('a');link.id='mobile-guide-link';link.textContent='打开手机连接说明';link.target='_blank';link.rel='noopener noreferrer';box.append(link);
  const p=document.createElement('p');p.className='caption';p.textContent='用 iPhone 浏览器扫码。打不开时检查同一 Wi-Fi、电脑防火墙和 VPN；不要关闭整个防火墙，可改用其他连接方式。完成后关闭手机 Wi-Fi 代理。';box.append(p);selectMobileAddress(0);
}
function selectMobileAddress(index) {
  const address=authMobileAddresses[index],img=el('mobile-qr'),link=el('mobile-guide-link');if(!address||!img||!link)return;
  try{const url=new URL(address.url);if(url.protocol!=='http:'||['127.0.0.1','localhost','0.0.0.0'].includes(url.hostname))throw new Error('没有可用的局域网地址。');img.hidden=!address.qr_svg;if(address.qr_svg)img.src='data:image/svg+xml;charset=utf-8,'+encodeURIComponent(address.qr_svg);link.href=url.href;}
  catch(e){img.hidden=true;link.hidden=true;showError('auth-error',e);}
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
      const stages={preparing_cert:'正在准备证书…',installing_cert_currentuser:'请在系统弹窗中允许安装证书。',installing_cert_localmachine_uac:'请在系统弹窗中授权。',starting_proxy:'正在准备连接…',setting_system_proxy:'正在设置临时代理…',waiting_capture:'请重新打开电脑微信，在寿司郎小程序里浏览门店和我的单据，无需提交订单。',probing:'凭证已收到，正在检查…'};
      el('auth-progress').textContent=engine.status==='error'?engine.message:stages[engine.stage]||engine.message||'连接中…';
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
