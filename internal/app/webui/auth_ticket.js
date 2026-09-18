let ticketDraft=null,authReturnToTicket=false,authModeRunning='',authPollTimer=null,authSavedAt='',authPollBusy=false;

async function reviewTicket() {
  if(!state.selectedStore||!state.live)throw new Error('先选一家门店。');
  if(state.ticketUncertain){toast('上次取号结果还未确认，请先查询已有号码。');return;}
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
    el('ticket-confirm').disabled=true;
    let submitted=false;
    try{
      await loadStatus();
      if(!state.status.has_config||state.status.auth_health?.status==='stale'){
        el('ticket-dialog').close();await openAuth(true);return;
      }
      submitted=true;
      const result=await api('/api/queue/ticket',ticketDraft,{timeout:45000});
      el('ticket-dialog').close();showTicket(result);ticketDraft=null;
    }catch(e){
      showError('ticket-error',e);
      if(submitted){
        state.ticketUncertain=true;el('ticket-dialog').close();
        showTicketUncertain(e.message);
      }
    }finally{el('ticket-confirm').disabled=false;}
  });
}
function showTicketUncertain(message) {
  navigatePage('queue');const box=el('ticket-result');box.hidden=false;
  box.innerHTML='<h2>取号结果还未确认</h2><p class="muted">'+escapeHTML(message)+'</p><p class="caption">先查询已有号码，不要重复取号。也可以在官方小程序核对。</p><button class="button primary" data-action="query-ticket">查询已有号码</button>';
  box.focus();el('take-ticket').disabled=true;
}
function showTicket(result) {
  state.ticketUncertain=false;
  navigatePage('queue');
  const ticket=result.ticket||{},number=ticket.number;
  if(!number){state.ticketUncertain=true;showTicketUncertain('官方没有返回明确号码，请在小程序核对。');return;}
  const id=String(ticket.monitored_store_id||ticket.store_id||ticket.storeId||'');
  const box=el('ticket-result');box.hidden=false;
  box.innerHTML='<div class="section-heading"><h2>'+escapeHTML(id?storeName(id):'我的排队')+'</h2><span class="badge green">'+(result.recovered?'已有号码':'已取到号')+'</span></div><div class="ticket-number">'+escapeHTML(number)+'</div><p class="muted">以官方小程序和门店叫号为准。</p><div id="ticket-forecast" class="plan">正在读取预计入座时间…</div><div class="actions"><button class="button secondary" data-action="query-ticket">刷新号码</button><button class="button quiet" data-action="cancel-ticket">取消这个号</button></div>';
  box.focus();state.currentTicket=ticket;
  if(id){state.selectedStore=id;loadLive();loadTicketForecast(id,number);}else el('ticket-forecast').textContent='门店尚未确认，请到官方小程序核对。';
}
async function queryTicket() {
  await guard('ticket-query',async()=>{
    await loadStatus();
    if(!state.status.has_config||state.status.auth_health?.status==='stale'){await openAuth(false);return;}
    const result=await api('/api/queue/ticket/status');
    if(result.ticket){showTicket(result);return;}
    state.ticketUncertain=false;state.currentTicket=null;
    el('ticket-result').hidden=false;
    el('ticket-result').innerHTML='<h2>当前没有查到排队号</h2><p class="muted">如小程序仍有号码，以小程序为准。</p>';
    el('take-ticket').disabled=!state.live?.online_open;
  });
}
async function loadTicketForecast(id,number) {
  try{
    const target=String(number).match(/\d+/)?.[0];if(!target){el('ticket-forecast').textContent='号码格式暂不支持估计。';return;}
    const travel=Number(el('travel-minutes').value||0);
    const data=await api('/api/queue/advisor?store='+encodeURIComponent(id)+'&target_no='+target+'&travel_minutes='+Math.max(0,Math.min(240,travel)));
    const box=el('ticket-forecast');if(!box)return;
    const eta=data.eta||{},window=eta.wait_minutes_range,range=eta.estimated_called_at_range;
    box.textContent='记录还不足，暂时算不准这个号码的入座时间。';
    if(range)box.innerHTML='预计入座<strong>'+escapeHTML(range.early+'–'+range.late)+'</strong>'+(window?'还需 '+window.low+'–'+window.high+' 分钟。':'')+'以门店叫号为准。'+(eta.arrival_suggestion?'<p>'+escapeHTML(eta.arrival_suggestion)+'</p>':'');
  }catch(e){const box=el('ticket-forecast');if(box)box.textContent='预计时间暂未更新，号码仍然有效。';}
}
async function cancelTicket() {
  await guard('ticket-cancel',async()=>{
    if(!await confirmDialog('取消当前排队号','这会取消真实号码，取消后无法恢复。'))return;
    await api('/api/queue/ticket/cancel',{});state.currentTicket=null;
    el('ticket-result').hidden=false;el('ticket-result').innerHTML='<h2>已取消排队号</h2>';
  });
}
function renderAuthInstructions() {
  const mobile=el('auth-method').value==='mobile';
  el('auth-instructions').textContent=mobile?'手机和电脑连接同一 Wi-Fi。按手机页面安装证书、设置代理，再打开寿司郎小程序；完成后关闭手机代理。':'需要安装证书并临时启用系统代理，只读取寿司郎接口。完成或停止后恢复代理；请在系统弹窗中授权。';
}
async function openAuth(returnToTicket) {
  await loadStatus();authReturnToTicket=returnToTicket;authSavedAt=state.status.auth_meta?.captured_at||'';
  el('auth-method').value=state.status.platform==='darwin'?'desktop':'mobile';
  el('auth-progress').textContent='';el('mobile-guide').hidden=true;el('auth-continue').hidden=true;el('auth-start').hidden=false;
  el('auth-method').disabled=false;el('auth-start').disabled=false;
  el('auth-continue').textContent=returnToTicket?'返回取号确认':'完成连接';
  renderAuthInstructions();el('auth-dialog').showModal();
}
async function startAuth() {
  await guard('auth-start',async()=>{
    const method=el('auth-method').value;
    el('auth-start').disabled=true;el('auth-method').disabled=true;
    authModeRunning=method;
    try{
      const result=await api(method==='mobile'?'/api/mobile-auth/start':'/api/engine/capture',{},{timeout:30000});
      el('auth-start').hidden=true;
      if(method==='mobile')renderMobileGuide(result);
      await pollAuth();
    }catch(e){
      if(e.status){authModeRunning='';el('auth-method').disabled=false;el('auth-progress').textContent=e.message;}
      else{
        el('auth-start').hidden=true;
        el('auth-progress').textContent='连接结果还未确认，正在检查。关闭窗口会尝试停止连接。';
        authPollTimer=setTimeout(pollAuth,1800);
      }
    }
    finally{el('auth-start').disabled=false;}
  });
}
function renderMobileGuide(data) {
  const box=el('mobile-guide');box.hidden=false;box.replaceChildren();
  if(data.qr_svg){const img=document.createElement('img');img.alt='手机连接引导二维码';img.src='data:image/svg+xml;charset=utf-8,'+encodeURIComponent(data.qr_svg);box.append(img);}
  const link=data.guide_urls?.[0];
  if(link){try{const u=new URL(link);if(u.protocol==='http:'){const a=document.createElement('a');a.href=link;a.textContent='打开手机连接说明';a.target='_blank';a.rel='noopener noreferrer';box.append(a);}}catch{}}
  const p=document.createElement('p');p.className='caption';p.textContent='用手机浏览器扫码，跟随页面操作。结束后请关闭手机 Wi-Fi 代理。';box.append(p);
}
async function pollAuth() {
  if(!el('auth-dialog').open||!authModeRunning||authPollBusy)return;
  authPollBusy=true;
  try{
    const s=await loadStatus(),engine=s.engine||{};
    let done=false;
    if(authModeRunning==='mobile'){
      const mobile=await api('/api/mobile-auth');
      if(mobile.active&&el('mobile-guide').hidden)renderMobileGuide(mobile);
      el('auth-progress').textContent=mobile.message||'等待手机连接…';
      done=!!mobile.saved&&!!s.has_config;
      if(!mobile.active&&!done){authModeRunning='';el('auth-start').hidden=false;el('auth-method').disabled=false;}
    }else{
      const stages={preparing_cert:'正在准备证书…',installing_cert_currentuser:'请在系统弹窗中允许安装证书。',installing_cert_localmachine_uac:'请在系统弹窗中授权。',starting_proxy:'正在准备连接…',setting_system_proxy:'正在设置临时代理…',waiting_capture:'请重新打开电脑微信，在寿司郎小程序里浏览门店和我的单据，无需提交订单。',probing:'凭证已收到，正在检查…'};
      el('auth-progress').textContent=engine.status==='error'?engine.message:stages[engine.stage]||engine.message||'连接中…';
      done=!!s.has_config&&engine.status==='idle'&&(engine.stage==='done'||(s.auth_meta?.captured_at&&s.auth_meta.captured_at!==authSavedAt));
      if(engine.status==='error'||(engine.status==='idle'&&!done)){authModeRunning='';el('auth-start').hidden=false;el('auth-method').disabled=false;}
    }
    if(done){authModeRunning='';el('auth-progress').textContent='凭证已保存在本机。'+(el('auth-method').value==='mobile'?'请关闭手机 Wi-Fi 代理。':'');el('auth-continue').hidden=false;}
  }catch(e){el('auth-progress').textContent=e.message;}
  finally{authPollBusy=false;if(authModeRunning&&el('auth-dialog').open)authPollTimer=setTimeout(pollAuth,1800);}
}
async function closeAuth() {
  if(state.busy.has('auth-start')){toast('连接正在启动，请稍等再关闭。');return;}
  await guard('auth-close',async()=>{
    clearTimeout(authPollTimer);
    const mode=authModeRunning;
    if(mode){
      const result=await api(mode==='mobile'?'/api/mobile-auth/stop':'/api/engine/stop',{});
      if(result.engine?.status==='stopping')throw new Error('代理正在恢复，请稍后再关闭。');
      authModeRunning='';
    }
    el('auth-text').value='';el('auth-dialog').close();
    if(mode==='mobile')toast('请同时关闭手机 Wi-Fi 代理。');
  });
}
async function finishAuth() {
  const resume=authReturnToTicket;
  await closeAuth();if(el('auth-dialog').open)return;
  if(resume)await reviewTicket();
}
async function importAuth() {
  await guard('auth-import',async()=>{
    if(authModeRunning)throw new Error('先停止当前连接，再导入凭证。');
    const text=el('auth-text').value.trim();if(!text)throw new Error('先粘贴凭证。');
    const result=await api('/api/auth/import',{text});
    if(!result.saved){el('auth-progress').textContent='还缺少：'+(result.missing||[]).join('、');return;}
    el('auth-text').value='';await loadStatus();el('auth-progress').textContent='凭证已保存在本机。';el('auth-continue').hidden=false;el('auth-start').hidden=true;
  });
}
