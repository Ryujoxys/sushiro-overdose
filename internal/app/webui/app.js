
let cp='da',es={status:'idle'},hc=0,as=[],sd='',pr={},pf='',cE=null,stores=[],selStores=[],calErrs=[],arTimer=null,lastDiag=null,spCfg={},spState={status:'idle'},spAutoStart={},spQueueState={},qdSelected=[],qdDashboardData={},qtSelected=[],qtTrendStores=[],qaStatus={},ah={},am={},accCalibrated=0,nfc=true,notifyChannels=[],cloudAuth={},cloudVerifyOnLoad=false,cloudRefreshPending=false,_inflight=null,qdAutoTimer=null,qdRefreshToken=0,qdDashToken=0,uiMode='simple',uiModeSeq=0,prefsLoaded=false,prefsLoading=null;
const W=['日','一','二','三','四','五','六'];
const need=['x_app_code','query_auth','reservation_auth','user_agent','referer','wechat_id','phone_number','store_ids'];
const csrfToken=document.querySelector('meta[name="sushiro-csrf"]')?.content||'';
const rawFetch=window.fetch.bind(window);
function sameOriginRequest(input){
  try{
    const target=input instanceof Request?input.url:String(input);
    return new URL(target,location.href).origin===location.origin;
  }catch(e){return true}
}
let staleSessionReloading=false;
window.fetch=async(input,init)=>{
  const opt=init?{...init}:{};
  const method=String(opt.method||(input&&input.method)||'GET').toUpperCase();
  if((method==='POST'||method==='PUT')&&sameOriginRequest(input)){
    const h=new Headers(opt.headers||(input&&input.headers)||{});
    h.set('X-Sushiro-CSRF',csrfToken);
    opt.headers=h;
  }
  const resp=await rawFetch(input,opt);
  // 应用重启后会换 CSRF token：旧页面提交会 403。自动刷新拿新页面，避免用户卡在“CSRF 校验失败”。
  if(resp.status===403&&!staleSessionReloading&&sameOriginRequest(input)){
    try{const d=await resp.clone().json();if(/CSRF/i.test(String(d&&d.error||''))){staleSessionReloading=true;toast('应用已重启，页面已过期，正在自动刷新…');setTimeout(()=>location.reload(),1200)}}catch(e){}
  }
  return resp;
};
function el(id){return document.getElementById(id)}
function esc(s){const d=document.createElement('div');d.textContent=s==null?'':String(s);return d.innerHTML}
function toast(msg,type){if(msg==null||msg==='')return;const s=String(msg);if(!type)type=/失败|错误|不可|无法|未能|超时|缺|invalid|error/i.test(s)?'err':(/请先|请填|请至少|至少填|请选|尚未/.test(s)?'warn':(/已|成功|完成|保存|启用|清理|恢复|启动/.test(s)?'ok':'info'));let w=el('toastWrap');if(!w){w=document.createElement('div');w.id='toastWrap';w.className='toast-wrap';document.body.appendChild(w)}const t=document.createElement('div');t.className='toast '+type;t.textContent=s;w.appendChild(t);requestAnimationFrame(()=>t.classList.add('in'));const long=/失败|错误|不可|无法|未能|超时|invalid|error/i.test(s);const ms=long?6500:2900;let timer=setTimeout(()=>{t.classList.remove('in');setTimeout(()=>t.remove(),280)},ms);t.onclick=()=>{clearTimeout(timer);t.classList.remove('in');setTimeout(()=>t.remove(),280)};t.title='点此关闭'}
function submitting(key){return _inflight&&_inflight.has(key)}
let _qdInputTimer=null;
function qdInputDebounced(){updateQueuePredictionReadiness();clearTimeout(_qdInputTimer);_qdInputTimer=setTimeout(()=>{renderReminderTemplateHint();loadQueueDashboard()},400)}
async function submitGuard(key,fn){if(!_inflight)_inflight=new Set();if(_inflight.has(key)){toast('正在处理，请稍候…','warn');return}const btn=document.activeElement;if(btn&&btn.tagName==='BUTTON'){btn.dataset._oldTxt=btn.textContent;btn.disabled=true;btn.textContent='提交中…'}_inflight.add(key);try{await fn()}finally{_inflight.delete(key);if(btn&&btn.tagName==='BUTTON'&&btn.dataset._oldTxt!=null){btn.disabled=false;btn.textContent=btn.dataset._oldTxt;delete btn.dataset._oldTxt}}}
function confirmDialog(opts){opts=typeof opts==='string'?{body:opts}:(opts||{});const danger=opts.danger!=null?opts.danger:/危险|不可恢复|卸载|清理本地|删除/.test(opts.body||'');return new Promise(res=>{let ov=el('confirmOv');if(!ov){ov=document.createElement('div');ov.id='confirmOv';ov.className='ov';document.body.appendChild(ov)}ov.innerHTML='<div class="ovc confirm-ovc'+(danger?' confirm-danger':'')+'"><div class="confirm-h">'+(danger?'⚠ ':'')+esc(opts.title||(danger?'危险操作':'请确认'))+'</div><div class="confirm-b">'+esc(opts.body||'')+'</div><div class="confirm-acts"><button class="bt bt-w" id="cfNo">'+esc(opts.cancel||'取消')+'</button><button class="bt bt-r" id="cfYes">'+esc(opts.ok||(danger?'确认':'继续'))+'</button></div></div>';ov.classList.remove('hid');ov.style.display='flex';const done=v=>{ov.classList.add('hid');ov.style.display='none';res(v)};el('cfYes').onclick=()=>done(true);el('cfNo').onclick=()=>done(false);el('cfYes').focus();ov.onclick=e=>{if(e.target===ov)done(false)}})}
// ensureNotifyConfigured 在写操作（抢预约/蹲未来/生成提醒）前校验通知渠道是否配置。
// 已配置→返回 true 继续；未配置→弹 confirmDialog 引导去配置：「去配置」返回 false（中断当前操作、跳设置页），
// 「先继续」返回 true（操作照常，但用户已被提醒收不到推送）。actionLabel 用于文案，如"抢到预约"。
async function ensureNotifyConfigured(actionLabel){if(nfc)return true;const go=await confirmDialog({title:'还没配置通知渠道',body:'通知渠道（飞书/Telegram/Bark/Server酱）没配的话，'+(actionLabel||('操作成功')+'后')+'你收不到推送，得一直盯着屏幕。现在去配一个？只需填一次。',ok:'去配置通知',cancel:'先继续'});if(go){focusNotifySettings();return false}return true}
const OV_CLOSERS={confirmOv:()=>{const n=el('cfNo');if(n)n.click()},storePicker:()=>closeStorePicker(),healthPanel:()=>closeHealthPanel(),firstUse:()=>closeFirstUseWizard(),authWiz:()=>closeAuthWizard()};
document.addEventListener('keydown',e=>{if(e.key!=='Escape')return;const hints=document.querySelectorAll('.hint-pop');let hintOpen=false;hints.forEach(h=>{if(h.style.display==='block'){h.style.display='none';hintOpen=true}});if(hintOpen){e.preventDefault();return}const open=Array.from(document.querySelectorAll('.ov')).filter(x=>!x.classList.contains('hid')&&x.style.display!=='none'),ov=open[open.length-1];if(ov&&OV_CLOSERS[ov.id]){e.preventDefault();OV_CLOSERS[ov.id]()}});
async function safeFetch(url,opts,timeoutMs){
  const ms=typeof timeoutMs==='number'?timeoutMs:15000;
  const ctrl=new AbortController();
  const t=setTimeout(()=>ctrl.abort(),ms);
  try{
    const r=await fetch(url,{...(opts||{}),signal:ctrl.signal});
    if(!r.ok){let body='';try{body=(await r.text()).slice(0,500)}catch(e){}
      throw new Error('HTTP '+r.status+' '+r.statusText+(body?' — '+body:''));
    }
    return await r.json();
  }catch(e){
    if(e.name==='AbortError')throw new Error('请求超时（'+ms+'ms）: '+url);
    throw e;
  }finally{clearTimeout(t)}
}
function loadErrBoxHTML(err,retryAttr,label){
  const msg=String((err&&(err.message||err))||'(unknown)');
  const head=label?label+'失败':'加载失败';
  return '<div class="empty"><b>'+esc(head)+'</b><br><code style="word-break:break-all;display:inline-block;margin-top:6px;color:var(--red)">'+esc(msg)+'</code>'+(retryAttr?'<div class="mt8"><button class="bt bt-w bt-s" onclick="'+retryAttr+'">重试</button></div>':'')+'</div>';
}
function escA(s){return esc(s).replaceAll('"','&quot;')}
const NAV_GROUPS=[
  {id:'home',label:'首页',pages:[['da','概览'],['gu','新手入门']]},
  {id:'eat',label:'现在去吃',pages:[['qt','门店排队']]},
  {id:'number',label:'我有号码',pages:[['qd','叫号预测']]},
  {id:'book',label:'约未来',pages:[['ca','可约日历'],['sn','自动抢预约']]},
  {id:'mine',label:'我的单据',pages:[['re','预约 / 排队号']]},
  {id:'settings',label:'设置',pages:[['se','设置']]}
];
const PAGE_GROUP={};NAV_GROUPS.forEach(g=>g.pages.forEach(([p])=>PAGE_GROUP[p]=g.id));
const ADVANCED_GROUPS=new Set(['book','mine']);
const ADVANCED_PAGES=new Set(['ca','sn','re']);
const ADVANCED_FOLDS=new Set(['fold-sm','fold-in','fold-lo','fold-safe','fold-mcp']);
function currentUIMode(){return uiMode==='advanced'?'advanced':'simple'}
function isAdvancedPage(page){return ADVANCED_PAGES.has(page)}
function modeLabel(){return currentUIMode()==='advanced'?'进阶版':'简化版'}
function cachedUIMode(){try{return localStorage.getItem('sushiro_ui_mode')==='advanced'?'advanced':'simple'}catch(e){return 'simple'}}
function cacheUIMode(mode){uiMode=mode==='advanced'?'advanced':'simple';try{localStorage.setItem('sushiro_ui_mode',uiMode)}catch(e){}}
function advancedPageName(page){return({ca:'约未来',sn:'自动抢预约',re:'我的单据'}[page]||'这个功能')}
// 小屏顶部导航横向滚动时，把当前高亮项滚进可视区。
function keepActiveTopNavVisible(){
  requestAnimationFrame(()=>{
    requestAnimationFrame(()=>{
      const nav=document.querySelector('.nav.top');
      const a=nav&&nav.querySelector('a.on:not(.hid)');
      if(!nav||!a)return;
      const nr=nav.getBoundingClientRect();
      const ar=a.getBoundingClientRect();
      if(ar.left<nr.left+2)nav.scrollLeft-=(nr.left-ar.left)+6;
      else if(ar.right>nr.right-2)nav.scrollLeft+=(ar.right-nr.right)+6;
    });
  });
}
function applyUIMode(){
 uiMode=currentUIMode();
 document.body.classList.toggle('advanced-mode',uiMode==='advanced');
 document.body.classList.toggle('simple-mode',uiMode!=='advanced');
 document.querySelectorAll('#uiModeSwitch button,.mode-settings .mode-switch button').forEach(b=>{
  const adv=/进阶|advanced/i.test(b.id||b.textContent||'');
  b.classList.toggle('on',adv?uiMode==='advanced':uiMode!=='advanced');
 });
 document.querySelectorAll('.nav.top a').forEach(a=>{
  const hidden=uiMode!=='advanced'&&ADVANCED_GROUPS.has(a.dataset.group||'');
  a.classList.toggle('hid',hidden);
 });
 keepActiveTopNavVisible();
 renderSettingsStatus();
 renderSetupCard();
 if(uiMode!=='advanced'&&isAdvancedPage(cp))go('da');
}
async function persistUIMode(mode){
 const wanted=mode==='advanced'?'advanced':'simple';
 try{
  const base=await ensurePrefsLoaded();
  if(currentUIMode()!==wanted)return;
  const d=await(await fetch('/api/preferences',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({...base,ui_mode:wanted})})).json();
  if(d.error){toast('模式已切换，但保存失败：'+d.error);return}
  if(currentUIMode()!==wanted)return;
  pr=d.preferences||{...base,ui_mode:wanted};prefsLoaded=true;cacheUIMode(pr.ui_mode==='advanced'?'advanced':'simple');applyUIMode();
 }catch(e){toast('模式已切换，但保存失败，下次打开可能恢复原设置')}
}
async function setUIMode(mode){
 uiModeSeq++;cacheUIMode(mode);applyUIMode();
 toast('已切换到'+modeLabel());
 persistUIMode(uiMode);
}
async function enterAdvanced(target){
 if(currentUIMode()==='advanced'){if(target)go(target);return}
 if(!await confirmDialog({title:'切换到进阶版？',body:advancedPageName(target||'')+'在进阶版中。进阶版会显示完整预约、取号、采集和维护功能；会执行操作仍会单独确认。',ok:'切换到进阶版',cancel:'留在简化版'}))return;
 await setUIMode('advanced');
 if(target)go(target);
}
function renderSubnav(g,active){const sn=el('subnav');if(!sn)return;if(!g||g.pages.length<=1){sn.innerHTML='';sn.classList.add('hid');return}sn.classList.remove('hid');sn.innerHTML=g.pages.map(([p,label])=>'<a href="#" class="'+(p===active?'on':'')+'" onclick="go(\''+p+'\');return false">'+esc(label)+'</a>').join('')}
function goGroup(gid){const g=NAV_GROUPS.find(x=>x.id===gid);if(g)go(g.pages[0][0]);return false}
function stopQDAutoRefresh(){if(qdAutoTimer){clearInterval(qdAutoTimer);qdAutoTimer=null}}
function stopCalendarAutoRefresh(){if(arTimer){clearInterval(arTimer);arTimer=null}}
function go(n,e,noPush){if(!PAGE_GROUP[n])n='da';if(currentUIMode()!=='advanced'&&isAdvancedPage(n)){const target=n;if(location.hash.slice(1)!=='da'){if(noPush)history.replaceState(null,'','#da');else history.pushState(null,'','#da')}n='da';setTimeout(()=>enterAdvanced(target),80)}if(cp==='qd'&&n!=='qd')stopQDAutoRefresh();if(cp==='ca'&&n!=='ca')stopCalendarAutoRefresh();document.querySelectorAll('.wrap>section[id^="p-"]').forEach(p=>p.classList.add('hid'));const sec=el('p-'+n);if(sec)sec.classList.remove('hid');const gid=PAGE_GROUP[n]||'home',g=NAV_GROUPS.find(x=>x.id===gid);document.querySelectorAll('.nav.top a').forEach(a=>a.classList.toggle('on',a.dataset.group===gid));renderSubnav(g,n);const pageChanged=cp!==n;cp=n;if(!noPush&&location.hash.slice(1)!==n)history.pushState(null,'','#'+n);if(pageChanged){try{window.scrollTo(0,0)}catch(err){}}const loader=({da:lDA,ca:lC,qd:lQD,qt:lQT,sn:lSn,re:lR,se:lS})[n];loader?.();if(cloudRefreshPending&&(n==='qd'||n==='qt')){cloudRefreshPending=false;setTimeout(refreshCloudDependentViews,120)}applyUIMode();return false}
window.addEventListener('popstate',()=>{const h=location.hash.slice(1);go(h&&PAGE_GROUP[h]?h:'da',null,true)});
async function loadStatus(){const v=el('ver');try{const r=await(await fetch('/api/status')).json();v.textContent='v'+r.version;v.classList.remove('hid');hc=!!r.has_config;pf=r.platform||'';es=r.engine||{status:'idle'};spState=r.sampling||spState;ah=r.auth_health||{};am=r.auth_meta||{};nfc=r.notify_configured!==false;maybeShowQuarantineCard(r);uE();uD();uAuth();renderSettingsStatus();loadActiveTickets(false);}catch(e){v.textContent='offline';v.classList.remove('hid');heroLoadFailed(e)}}
function heroLoadFailed(err){const badge=el('heroBadge'),t=el('heroTitle'),c=el('heroCopy');if(badge)badge.textContent='连接异常';if(t)t.textContent='读不到运行状态';if(c)c.innerHTML='本机服务没有响应：<code style="word-break:break-all">'+esc(String((err&&err.message)||err||'unknown'))+'</code> <button class="bt bt-w bt-s" onclick="loadStatus()">重试</button>'}
function uAuth(){
 const pill=el('authPill'),banner=el('authBanner'),st=(ah&&ah.status)||'unknown',reason=(ah&&ah.reason)?String(ah.reason):'';
 const softWarn=hc&&st!=='stale'&&am&&am.soft_warn;
 if(pill){let cls='authpill',txt='';
  if(!hc){txt='只读模式'}
  else if(st==='stale'){cls+=' stale';txt='通行证可能失效'}
  else if(softWarn){cls+=' warn';txt='通行证快到期'}
  else if(currentUIMode()==='advanced'&&!nfc){cls+=' warn';txt='通知未配置'}
  else{cls+=' ok';txt='一切就绪'}
  pill.className=cls;pill.textContent=txt;pill.classList.remove('hid')}
 if(banner){
  if(hc&&st==='stale'){
   const lastM=(am&&am.capture_method_label)?am.capture_method_label:'';
   const recapLabel=lastM?('沿用「'+esc(lastM)+'」重新获取'):'重新获取（约 3 分钟）';
   banner.classList.remove('hid');banner.innerHTML='<b>🎫 通行证可能失效了</b>寿司郎会定期回收通行证，也可能被手机端重新登录顶掉。<button class="bt bt-r bt-s" onclick="event.stopPropagation();resetAuthAndStart()">'+recapLabel+'</button>'+(reason?'<details style="flex-basis:100%" onclick="event.stopPropagation()"><summary class="mu" style="cursor:pointer">技术细节</summary><code style="word-break:break-all">'+esc(reason)+'</code></details>':'')}
  else if(softWarn){
   banner.classList.remove('hid');banner.innerHTML='<b>🎫 通行证快到期了</b>当前凭证已用 '+esc(am.age_label||'')+'，接近你以往的平均有效期。挑个空档重新获取一次，免得抢预约/取号时正好失效。<button class="bt bt-o bt-s" onclick="event.stopPropagation();resetAuthAndStart()">提前续期</button>'}
  else{banner.classList.add('hid');banner.innerHTML=''}}}
function healthStripHTML(items){return items.map(x=>'<div class="strip"><span class="st '+x.s+'">'+(x.s==='ok'?'✓':x.s==='bad'?'✕':'!')+'</span><div><b>'+esc(x.t)+'</b><span class="sd">'+esc(x.d)+'</span></div>'+(x.a?'<button class="bt bt-w bt-s" onclick="'+x.a.f+'">'+esc(x.a.l)+'</button>':'')+'</div>').join('')}
function openHealthPanel(){let ov=el('healthPanel');if(!ov){ov=document.createElement('div');ov.id='healthPanel';ov.className='ov';document.body.appendChild(ov)}
 const st=(ah&&ah.status)||'unknown',spOK=!!(spState.running||spState.enabled||spState.sample_runs>0);
 const items=[
  {t:'寿司郎通行证 🎫',d:hc?(st==='stale'?('可能已失效'+((ah&&ah.reason)?('：'+ah.reason):'，建议重新获取')):'已就绪'):'看排队不需要；抢未来预约、远程取号、读单据才需要',s:hc?(st==='stale'?'bad':'ok'):'warn',a:hc?{l:'重新获取',f:st==='stale'?'closeHealthPanel();resetAuthAndStart()':'closeHealthPanel();openAuthWizard()'}:{l:'去获取',f:'closeHealthPanel();openAuthWizard()'}}
 ];
 if(currentUIMode()==='advanced')items.push({t:'通知渠道',d:nfc?('已配置'+(notifyChannels.length?('：'+notifyChannels.join('、')):'')):'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:{l:nfc?'管理':'去配置',f:'closeHealthPanel();focusNotifySettings()'}});
 if(currentUIMode()==='advanced')items.push({t:'预测数据',d:spOK?'采集中，“几点能吃上”会越来越准':'开启后到店预测更准（可选）',s:spOK?'ok':'warn',a:{l:spOK?'查看':'去开启',f:"closeHealthPanel();openSettingsFold('fold-sm')"}});
 ov.innerHTML='<div class="ovc" style="width:min(560px,96vw)"><div class="fl ai jb mb16"><b>运行前置条件</b><button class="bt bt-w bt-s" onclick="closeHealthPanel()">关闭</button></div>'+healthStripHTML(items)+'<p class="mu mt16">红色需要处理，黄色按需配置；任何页面点右上角胶囊都能回到这里。</p></div>';
 ov.onclick=e=>{if(e.target===ov)closeHealthPanel()};
 ov.classList.remove('hid');ov.style.display='flex'}
function closeHealthPanel(){const ov=el('healthPanel');if(ov){ov.classList.add('hid');ov.style.display='none'}}
function authPillClick(){openHealthPanel()}
async function init(){cacheUIMode(cachedUIMode());applyUIMode();consumeCloudAuthResult();fillPageMascots();buildBelt();await loadStatus();await lP();checkUpdate();sse();if(consumeRecapture())return;const h=location.hash.slice(1);if(h&&PAGE_GROUP[h])go(h,null,true);else{go('da',null,true);loadHomeLive(true);maybeShowIntro()}}
/* consumeRecapture：通知里的「一键续期」深链 ?recapture=1 落地后，自动拉起通行证向导（stale 时先重置再抓）。
   返回 true 表示已接管启动流程，跳过常规首页/引导。 */
function consumeRecapture(){try{const p=new URLSearchParams(location.search);if(!p.get('recapture'))return false;history.replaceState(null,'',location.pathname+location.hash);go('se');startAuth();return true}catch(e){return false}}
function consumeCloudAuthResult(){try{const p=new URLSearchParams(location.search);const connected=p.get('cloud_connected');if(connected){cloudRefreshPending=true;cloudVerifyOnLoad=true;toast('云端 GitHub 登录已完成');refreshCloudDependentViews()}if(p.get('cloud_error'))toast(p.get('cloud_error'));if(p.has('cloud_connected')||p.has('cloud_error'))history.replaceState(null,'',location.pathname+location.hash)}catch(e){}}
function refreshCloudDependentViews(){try{if(cp==='qd')loadQueueDashboard();if(cp==='qt')refreshQueueView()}catch(e){}}
function maybeShowIntro(){try{/* sushiro_intro_seen_v2：bump 版本键，重要改版后让所有用户（含已看过 v1 的）重新看到一次引导。
   老用户想随时重看可点 setupCard 的「新手引导」按钮。仍保留 hc 屏蔽：已有通行证的用户不被首启浮层打扰。 */if(hc)return;if(localStorage.getItem('sushiro_intro_seen_v2'))return;localStorage.setItem('sushiro_intro_seen_v2','1');openFirstUseWizard()}catch(e){}}
function maybeShowQuarantineCard(r){try{if(!r||!r.quarantined)return;if(localStorage.getItem('sushiro_q_dismissed'))return;const exe=r.executable_path||'';
/* 取 .app 包路径（quarantine 标记在整个包上），而不是包内的可执行文件；
   路径含空格（"Sushiro Overdose.app"）必须用双引号包住，否则 xattr 把空格当分隔符报 No such file。 */
let target=exe;const ai=target.indexOf('.app/');if(ai>=0)target=target.slice(0,ai+4);if(!target)target='/Applications/Sushiro Overdose.app';
const cmd='xattr -dr com.apple.quarantine "'+target+'"';let ov=el('quarantineOv');if(!ov){ov=document.createElement('div');ov.id='quarantineOv';ov.className='ov';document.body.appendChild(ov)}
ov.innerHTML='<div class="ovc" style="width:min(560px,96vw)"><div class="fl ai jb mb16"><b>⚠ macOS 隔离标记</b><button class="bt bt-w bt-s" onclick="dismissQuarantineCard()">关闭</button></div><p class="mu">这个 App 是从网上下载的，macOS 给它加了隔离标记（Gatekeeper）。一般不影响使用，但少数情况下会让系统代理设置、通知或抓包证书被拒。移除后更省心。</p><p class="mt8"><b>打开「终端」粘贴运行这一行（按回车）即可：</b></p><pre style="background:rgba(0,0,0,.06);padding:10px 12px;border-radius:8px;overflow:auto;font-size:13px;word-break:break-all;white-space:pre-wrap"><code id="quarantineCmd">'+esc(cmd)+'</code></pre><div class="fl g8 fw mt12"><button class="bt bt-s" onclick="copyQuarantineCmd()">复制命令</button><button class="bt bt-o bt-s" onclick="dismissQuarantineCard()">我已执行，不再提示</button></div><p class="mu mt12">运行后重启本工具即可。仅对本工具生效，不动其它 App。</p></div>';
ov.classList.remove('hid');ov.style.display='flex';ov.onclick=e=>{if(e.target===ov)dismissQuarantineCard()}}catch(e){}}
function dismissQuarantineCard(){const ov=el('quarantineOv');if(ov){ov.classList.add('hid');ov.style.display='none'}try{localStorage.setItem('sushiro_q_dismissed','1')}catch(e){}}
function copyQuarantineCmd(){const c=el('quarantineCmd');if(!c)return;const txt=c.textContent||'';if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(txt).then(()=>toast('命令已复制，去终端粘贴运行'),()=>fallbackCopy(txt))}else{fallbackCopy(txt)}}
function fallbackCopy(txt){try{const ta=document.createElement('textarea');ta.value=txt;document.body.appendChild(ta);ta.select();document.execCommand('copy');document.body.removeChild(ta);toast('命令已复制，去终端粘贴运行')}catch(e){toast('复制失败，请手动选中上方命令复制')}}
function isRun(){return ['capturing','booking','sniping'].includes(es.status)}
function awzPeek(){try{const s=JSON.parse(localStorage.getItem('sushiro_wizard_state')||'null');if(!s)return null;const c=s.cap||{};return{step:s.step||1,fields:need.filter(k=>c[k]).length}}catch(e){return null}}
function renderSetupCard(){
 const card=el('setupCard'),list=el('setupList');if(!card||!list)return;
 const aw=awzPeek(),items=[];
 const authS=hc?((ah&&ah.status==='stale')?'warn':'ok'):'warn';
 const hasStores=(pr.selected_stores||[]).length>0;
 items.push({t:'常用门店',d:hasStores?('已选 '+pr.selected_stores.length+' 家，各页面自动带入'):'选好后看排队、预测、日历都不用重选',s:hasStores?'ok':'warn',a:hasStores?null:{l:'去选店',f:'openGuestStorePicker()'}});
 items.push({t:'寿司郎通行证 🎫',d:hc?(authS==='warn'?'可能已失效，建议重新获取':'已就绪'):(aw&&aw.fields>0?('拿到一半（'+aw.fields+'/'+need.length+' 项），可以继续'):'抢未来预约、远程取号、读单据时才需要'),s:authS,a:hc?(authS==='warn'?{l:'重新获取',f:'resetAuthAndStart()'}:null):{l:(aw&&aw.fields>0)?'继续获取':'去获取',f:'openAuthWizard()'}});
 const spOK=!!(spState.running||spState.enabled||spState.sample_runs>0);
 if(currentUIMode()==='advanced')items.push({t:'通知渠道',d:nfc?'已配置':'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:nfc?null:{l:'去配置',f:'focusNotifySettings()'}});
 if(currentUIMode()==='advanced')items.push({t:'预测数据',d:spOK?'采集中，“几点能吃上”会越来越准':'开启后到店预测更准（可选）',s:spOK?'ok':'warn',a:spOK?null:{l:'去开启',f:"openSettingsFold('fold-sm')"}});
 const allOK=items.every(x=>x.s==='ok');
 card.classList.toggle('hid',allOK);
 if(allOK)return;
 list.innerHTML=healthStripHTML(items);
}
function journeyStepHTML(kind,title,desc,state){
 const label={read:'只读',auth:'通行证',action:'会执行'}[kind]||kind;
 return'<div class="journey-step '+escA(kind)+' '+escA(state||'')+'"><span>'+esc(label)+'</span><b>'+esc(title)+'</b><small>'+esc(desc)+'</small></div>'
}
function journeyButtonsHTML(buttons){return(buttons||[]).map((b,i)=>'<button class="bt '+(i===0?'bt-r':'bt-w')+' bt-s" onclick="'+b.f+'">'+esc(b.l)+'</button>').join('')}
function renderJourneyPanel(){
 const box=el('journeyPanel');if(!box)return;
 const hasStores=(pr.selected_stores||[]).length>0,st=(ah&&ah.status)||'unknown',stale=hc&&st==='stale',running=isRun(),tickets=(activeTickets||[]).length;
 const aw=awzPeek(),authDesc=hc?(stale?'可能已失效，建议更新':'已就绪'):(aw&&aw.fields>0?('已拿到 '+aw.fields+'/'+need.length+' 项，可继续'):'抢预约前再拿');
 const actionDesc=running?'正在运行':(es.status==='error'?'需要先排障':(hc&&hasStores?'可以开始未来预约或远程取号':'先补齐门店和通行证'));
 let plan={level:'ok',mode:'只读优先',title:'今天该走哪条路',copy:'排队和预测直接用；抢预约、取号、看单据才要通行证。',buttons:[{l:'先看实时排队',f:"go('qt')"},{l:'获取通行证',f:'startAuth()'}]};
 if(tickets>0)plan={level:'ok',mode:'已有单据',title:'先看你手上的单据',copy:'你有未完成的预约或排队号，先确认避免重复。',buttons:[{l:'查看我的单据',f:"enterAdvanced('re')"},{l:'几点能吃上',f:"go('qd')"}]};
 else if(es.status==='error'){const certUAC=/机器级|LocalMachine|管理员权限|UAC|RunAs|elevated|exit code/i.test(es.message||'');plan={level:'bad',mode:'需要处理',title:'先处理这件事',copy:explainMsg(es.message||'')+' 处理前不会动你的预约或排队号。',buttons:certUAC?[{l:'重新装证书（会弹UAC，点“是”）',f:'startAuth()'},{l:'改用手机获取（更稳）',f:'closeAuthWizard();openAuthWizard();setTimeout(()=>awzDevice("ios"),50)'},{l:'打开本机诊断',f:'openDiagnostics()'}]:[{l:'打开本机诊断',f:'openDiagnostics()'},{l:hc?'重新获取通行证':'获取通行证',f:'startAuth()'}]};}
 else if(running)plan={level:'warn',mode:'运行中',title:'当前有任务正在执行',copy:'可保持页面打开；换目标前先停止当前任务。',buttons:[{l:'查看运行日志',f:"openSettingsFold('fold-lo')"},{l:'停止当前任务',f:'sE()'}]};
 else if(!hc)plan={level:'warn',mode:'只读可用',title:'先不用登录，也能看排队',copy:'排队和预测都不用登录；要抢预约或取号时再获取通行证。',buttons:[{l:'选门店看排队',f:'openGuestStorePicker()'},{l:'我要抢未来预约：获取通行证',f:'startAuth()'}]};
 else if(stale)plan={level:'bad',mode:'通行证待更新',title:'通行证可能失效了',copy:'看排队仍可用；抢预约或取号前建议重新获取通行证。',buttons:[{l:'重新获取通行证',f:'resetAuthAndStart()'},{l:'先看实时排队',f:"go('qt')"}]};
 else if(!hasStores)plan={level:'warn',mode:'还差门店',title:'通行证好了，下一步选门店',copy:'选好常用门店，各页面自动带入，不用每页重选。',buttons:[{l:'设置门店和偏好',f:'openSnPrefs()'},{l:'先看实时排队',f:"go('qt')"}]};
 else plan={level:'ok',mode:'准备就绪',title:'可以开始未来预约或取号',copy:'通行证和门店都就绪，可查日历或交给自动抢预约。',buttons:[{l:'查可约时段',f:"enterAdvanced('ca')"},{l:'自动抢预约',f:"enterAdvanced('sn')"}]};
 const steps=[
  journeyStepHTML('read','只读','排队、预测、叫号估算，直接用','ok'),
  journeyStepHTML('auth','通行证',authDesc,hc?(stale?'bad':'ok'):'warn'),
  journeyStepHTML('action','会执行',actionDesc,es.status==='error'?'bad':(running?'warn':(hc&&hasStores?'ok':'warn')))
 ];
 box.className='journey-panel mt16 '+plan.level;
 box.innerHTML='<div class="journey-head"><div><h2>'+esc(plan.title)+'</h2><p class="journey-copy">'+esc(plan.copy)+'</p></div><span class="journey-mode '+escA(plan.level)+'">'+esc(plan.mode)+'</span></div><div class="journey-steps">'+steps.join('')+'</div><div class="journey-cta">'+journeyButtonsHTML(plan.buttons)+'</div>';
}
function openGuestStorePicker(){openStorePicker({selected:(pr.selected_stores||[]).map(String),onConfirm:saveStarterStores})}
async function saveStarterStores(ids){if(!ids||!ids.length){toast('先勾选至少一家门店');return}const b={...pr,selected_stores:ids,store_priority:ids};if(!await savePrefsPayload(b,true))return;qtSelected=ids.map(String);rememberStores('sushiro_qt_stores',qtSelected);toast('已记住常用门店，看看现在排多久');go('qt')}
let activeTickets=[],activeLive={},activeLoadedAt=0,homeLiveAt=0;
function lDA(){loadActiveTickets(false);loadHomeLive(false)}
function goQtStore(id){qtSelected=[String(id)];rememberStores('sushiro_qt_stores',qtSelected);go('qt')}
async function loadHomeLive(force){
 const box=el('homeLive');if(!box)return;
 const now=Date.now();if(!force&&now-homeLiveAt<60000)return;homeLiveAt=now;
 let all=recallStores('sushiro_qt_stores');if(!all.length)all=(pr.selected_stores||[]).map(String);
 const fallback=!all.length;if(fallback&&stores.length)all=stores.map(s=>String(s.id));
 const ids=all.slice(0,3);
 if(!ids.length){box.innerHTML=hc?'<div class="empty">还没选常用门店；选好后首页直接看排队。<div class="mt8"><button class="bt bt-w bt-s" onclick="openGuestStorePicker()">选常用门店</button></div></div>':'';return}
 if(!box.innerHTML)box.innerHTML='<div class="home-live">'+ids.map(()=>'<div class="hl-card"><span class="hl-name mu">读取中…</span></div>').join('')+'</div>';
 const panels=await Promise.all(ids.map(id=>safeFetch('/api/queue/live?store='+encodeURIComponent(id),null,12000).catch(()=>null)));
 const items=panels.filter(Boolean);
 if(!items.length){box.innerHTML='<div class="empty">实时排队读取失败。<div class="mt8"><button class="bt bt-w bt-s" onclick="loadHomeLive(true)">重试</button></div></div>';return}
 const more=all.length>ids.length?'<button type="button" class="hl-card" onclick="go(\'qt\')"><span class="hl-name">还有 '+(all.length-ids.length)+' 家关注门店</span><span class="hl-num">→</span><span class="hl-sub">去「现在去吃」看全部</span></button>':'';
 box.innerHTML='<div class="home-live">'+items.map(s=>{
  const open=s.online_open||s.store_status==='OPEN';
  const eta=(s.eta_minutes!=null)?s.eta_minutes:((s.server_wait_minutes||0)>0?s.server_wait_minutes:null);
  return'<button type="button" class="hl-card" onclick="goQtStore(\''+escA(String(s.store_id))+'\')"><span class="hl-name">'+esc(s.store_name||s.store_id)+'</span><span class="hl-num '+(open?'':'closed')+'">'+(open?fmtN(s.wait_groups||0):'休')+'</span><span class="hl-sub">'+(open?('桌在等'+(eta!=null?' · 约 '+eta+' 分钟':'')+(s.called_no?' · 叫到 '+esc(String(s.called_no)):'')):'暂停营业 · 点开看详情')+'</span></button>'}).join('')+more+'</div>'+(fallback?'<p class="mu mt8">还没选常用门店，暂按通行证里你去过的门店显示。<button class="bt bt-w bt-s" onclick="openGuestStorePicker()">选常用门店</button></p>':'');
}
async function loadActiveTickets(force){
 if(!hc){activeTickets=[];renderActiveHome();return}
 const now=Date.now();
 if(!force&&now-activeLoadedAt<60000){renderActiveHome();return}
 activeLoadedAt=now;
 try{const d=await safeFetch('/api/reservations',null,15000);const items=(Array.isArray(d)?d:(d.items||[]));activeTickets=items.filter(r=>{const st=String(r.status||'').toUpperCase();return!/CANCEL|EXPIRE|FINISH|SEATED|DONE/.test(st)})}catch(e){activeTickets=[]}
 renderActiveHome();
 const seen=new Set();
 for(const r of activeTickets){
  if(recordKind(r)!=='net_ticket')continue;
  const id=String(r.monitored_store_id||r.storeId||'');
  if(!id||seen.has(id))continue;seen.add(id);
  try{activeLive[id]=await safeFetch('/api/queue/live?store='+encodeURIComponent(id),null,12000)}catch(e){}
 }
 if(seen.size)renderActiveHome();
}
function renderActiveHome(){
 const box=el('activeHome');if(!box)return;
 const list=activeTickets||[],show=hc&&list.length>0;
 box.innerHTML=show?list.map(ticketHeroHTML).join(''):'';
 const hero=el('heroBox');if(hero)hero.classList.toggle('hid',show);
 renderJourneyPanel();
}
function ticketHeroHTML(r){
 const kind=recordKind(r),storeId=String(r.monitored_store_id||r.storeId||''),store=r.store_name||storeDisplayName(storeId)||storeId;
 if(kind==='net_ticket'){
  const live=activeLive[storeId]||null,lines=[];
  if(live&&live.called_no)lines.push('现在叫到 '+esc(String(live.called_no)));
  if(r.wait>0)lines.push('你前面还有约 '+fmtN(r.wait)+' 桌');
  else if(live&&live.wait_groups>0)lines.push('店内在等约 '+fmtN(live.wait_groups)+' 桌');
  if(live&&live.eta_minutes!=null)lines.push('约等待 '+live.eta_minutes+' 分钟');
  const no=String(r.number||'-');
  const myNo=parseInt(String(r.number||'').replace(/\D+/g,''),10),calledNo=live&&live.called_no?parseInt(String(live.called_no).replace(/\D+/g,''),10):0;
  const todayKey=new Date(),tk=todayKey.getFullYear()+String(todayKey.getMonth()+1).padStart(2,'0')+String(todayKey.getDate()).padStart(2,'0');
  const isToday=!r.queueDate||String(r.queueDate).slice(0,8)===tk;
  const passed=isToday&&myNo>0&&calledNo>myNo;
  const rescueCard=passed?('<div class="th-rescue"><div class="th-rescue-t">⚠️ 你的 '+esc(myNo)+' 号可能已经过号了</div><div class="th-rescue-b">当前叫到 '+esc(calledNo)+'，已超过你的号码。寿司郎过号不会自动叫到，需要重新取号——到店现场取号最稳，或点右边的远程取号（取号后请尽快到店）。</div><div class="th-rescue-acts"><button class="bt bt-r bt-s" onclick="takeTicket(\''+escA(storeId)+'\')">重新取号</button><button class="bt bt-w bt-s" onclick="openTicketForecast(\''+escA(storeId)+'\',\''+escA(no)+'\')">看几点能吃上</button></div></div>'):'';
 return'<div class="ticket-hero"><div class="th-eyebrow">🎫 你正在排：'+esc(store)+'</div><div class="th-no">'+esc(no)+'</div>'+rescueCard+'<div class="th-line">'+(lines.length?lines.join(' · '):'点下方按钮看“几点能吃上”')+'</div><div class="th-sub">'+esc(r.checkedIn?'已签到':'未签到')+' · 进度以寿司郎小程序为准</div><div class="th-acts"><button class="bt bt-w" onclick="openTicketForecast(\''+escA(storeId)+'\',\''+escA(no)+'\')">⏱ 几点能吃上 / 设提醒</button><button class="bt bt-ghost" onclick="enterAdvanced(\'re\')">查看单据</button><button class="bt bt-ghost advanced-only" onclick="cancelNetTicket()">取消排队号…</button></div></div>';
 }
 const when=r.slot_label||[r.queueDate,fT(r.start),r.end?'-'+fT(r.end):''].filter(Boolean).join(' ');
 return'<div class="ticket-hero"><div class="th-eyebrow">📅 你有一个预约：'+esc(store)+'</div><div class="th-no">'+esc(when||String(r.number||'-'))+'</div><div class="th-line">'+esc(recordStatusText(r,kind))+(r.number?' · 预约号 '+esc(String(r.number)):'')+'</div><div class="th-sub">预约号不参与当天叫号进度；到点前记得出发。</div><div class="th-acts"><button class="bt bt-w" onclick="enterAdvanced(\'re\')">查看单据</button><button class="bt bt-ghost" onclick="go(\'qt\')">看门店现场排队</button></div></div>';
}
function seedQueueForecastStoreFromQueue(){const id=(qtSelected&&qtSelected[0])?String(qtSelected[0]):'';if(id){qdSelected=[id];rememberStores('sushiro_qd_store',qdSelected)}}
function openPickupForecastFromQueue(){seedQueueForecastStoreFromQueue();const oldNo=el('qdTargetNo');if(oldNo)oldNo.value='';setPlanDir('pickup');go('qd');setTimeout(()=>{setQueuePredictionMode('pickup');el('qdNowTicketCard')?.scrollIntoView({behavior:'smooth',block:'start'});setPickupToNow(true);applyPlanDir();runPlanCalcDebounced();el('qpPickup')?.focus()},100)}
function openTicketForecastFromQueue(){seedQueueForecastStoreFromQueue();go('qd');setTimeout(()=>{setQueuePredictionMode('ticket');el('qdExistingTicketCard')?.scrollIntoView({behavior:'smooth',block:'start'});const t=el('qdTargetNo');if(t){t.focus();t.select?.()}},100)}
function openTicketForecast(storeId,no){qdSelected=storeId?[String(storeId)]:[];rememberStores('sushiro_qd_store',qdSelected);const t=el('qdTargetNo'),n=parseInt(String(no||'').replace(/\D+/g,''),10);if(t)t.value=n>0?n:'';go('qd');setTimeout(()=>setQueuePredictionMode('ticket'),0)}
function explainMsg(m){m=String(m||'');if(/机器级|LocalMachine|管理员权限|UAC|RunAs|elevated|exit code/i.test(m))return'Windows 机器级证书没装上：PC 微信只读机器级证书库，装它时会弹 UAC 请求管理员权限，点「是」即可。被拒或关掉就会失败——重新获取通行证会再弹一次，这次点同意。';if(/证书|trust|certificate/i.test(m))return'证书问题：先到设置页刷新诊断，确认 CA 证书已信任；失败后可重新获取通行证。';if(/代理|proxy/i.test(m))return'代理问题：先点击设置页的“修复代理”，再重新获取通行证。';if(/401|403|凭证|认证|token|auth/i.test(m))return'通行证过期：重新获取通行证后再启动。';if(/network|timeout|超时|不可达|connection/i.test(m))return'网络问题：确认网络可访问寿司郎接口，稍后重试。';if(/门店|store/i.test(m))return'门店配置问题：检查设置页的预约/取号门店是否仍在可用列表中。';return'先查看设置页本机诊断和日志，处理红色项后重试。'}
function wechatLightHTML(w){if(!w)return'';let cls,txt,btn='';if(w.restarted&&w.running){cls='ok';txt='PC 微信已重新打开 ✓ 请在寿司郎小程序里点一次排队或预约'}else if(w.restarted){cls='ok';txt='检测到 PC 微信已重启'}else if(w.running){cls='warn';txt='检测到 PC 微信正在运行——请彻底退出（任务栏右键退出，不是最小化）后重新打开';btn=' <button class="bt bt-o bt-s" onclick="killWeChat()">一键结束微信</button>'}else{cls='bad';txt='没检测到 PC 微信在运行，请打开 PC 微信'}return'<p class="wechat-light '+cls+'">'+esc(txt)+btn+'</p>'}
// captureProgressHTML 渲染采集阶段进度条。基于后端 stage 枚举（preparing_cert/...）高亮当前阶段。
// 各阶段：装证书→起代理→设系统代理→抓包(等微信)→自检。已完成打绿勾，当前高亮，未来灰。
function captureProgressHTML(s){
  if(!s||s.status!=='capturing'||!s.stage||s.stage==='idle')return'';
  const steps=[
    {k:'cert',label:'装证书',stages:['preparing_cert','installing_cert_currentuser','installing_cert_localmachine_uac']},
    {k:'proxy',label:'起代理',stages:['starting_proxy']},
    {k:'sysproxy',label:'设系统代理',stages:['setting_system_proxy']},
    {k:'capture',label:'获取信息',stages:['waiting_capture']},
    {k:'probe',label:'自检',stages:['probing']}
  ];
  const order=['preparing_cert','installing_cert_currentuser','installing_cert_localmachine_uac','starting_proxy','setting_system_proxy','waiting_capture','probing','done'];
  const curIdx=order.indexOf(s.stage);
  let cells=steps.map(st=>{
    const stIdx=Math.max.apply(null,st.stages.map(x=>order.indexOf(x)));
    let state='pending';
    if(curIdx>stIdx)state='done';
    else if(st.stages.indexOf(s.stage)>=0)state='current';
    const icon=state==='done'?'✓':(state==='current'?'●':'○');
    return '<div class="cstep '+state+'"><span class="cstep-ic">'+icon+'</span><span class="cstep-lb">'+st.label+'</span></div>';
  }).join('<span class="cstep-arrow">›</span>');
  let sub='';
  if(s.stage==='waiting_capture'&&s.capture){
    const got=countCaptured(s.capture);
    sub='<p class="cstep-sub">已抓到 '+got+'/8 个字段'+(got<8?'，还差几个——请真的排队/预约一次（之后可取消），再点一次门店':'，正在自检…')+'</p>';
  }else if(s.stage==='installing_cert_localmachine_uac'){
    sub='<p class="cstep-sub warn">马上会弹出系统窗口请求管理员权限，请点「是」（装机器级证书必须）</p>';
  }
  return '<div class="cprogress">'+cells+'</div>'+sub;
}
function countCaptured(c){if(!c)return 0;let n=0;['x_app_code','query_auth','reservation_auth','user_agent','referer','wechat_id','phone_number','store_ids'].forEach(k=>{if(c[k])n++});return n}
// errorFromKind 用后端 error_kind 枚举生成人话文案 + 出路按钮，替代 explainMsg 正则猜。
function errorFromKind(s){
  const k=s.error_kind,m=s.message||'';
  if(k==='cert_uac_declined')return{t:'Windows 机器级证书没装上',d:'刚才弹出的系统窗口你没点是。点下面按钮重装，弹出时务必点「是」（PC 微信只读机器级证书库，必须管理员权限）。',btn:'重新装证书',act:'startAuth()'};
  if(k==='cert_locked')return{t:'钥匙串被锁住了',d:'macOS 钥匙串锁定，证书装不进去。在终端运行 security unlock-keychain 解锁后，点下面按钮重试。',btn:'我已解锁，重试',act:'startAuth()'};
  if(k==='cert_install_failed')return{t:'证书没装上',d:'证书安装失败：'+esc(m)+'。可到设置页诊断看详情，或重试。',btn:'重新装证书',act:'startAuth()'};
  if(k==='proxy_failed')return{t:'系统代理没设上',d:'设置系统代理失败：'+esc(m)+'。先到设置页点「修复代理」清理残留，再重试。',btn:'修复代理',act:'repairP()'};
  if(k==='quic_block_failed')return{t:'微信可能走旁路了',d:m||'Windows QUIC 屏蔽失败，微信可能用 UDP 绕过代理导致获取不到必要信息。建议重启微信再试，仍不行改用手机获取。',btn:'重新获取',act:'startAuth()'};
  if(k==='auth_stale')return{t:'通行证过期了',d:'通行证过期或被手机端登录顶掉。点下面重新获取。',btn:'重新获取通行证',act:'startAuth()'};
  if(k==='network')return{t:'网络问题',d:'连不上寿司郎接口：'+esc(m)+'。确认网络后重试。',btn:'重试',act:'startAuth()'};
  return{t:'需要处理',d:explainMsg(m),btn:'重新获取通行证',act:'startAuth()'};
}
function uD(){
  const b=el('bm'),bc=el('bc'),nc=el('nc'),pick=el('heroPick'),title=el('heroTitle'),copy=el('heroCopy'),badge=el('heroBadge');
  const run=isRun();
  b.disabled=run;
  b.classList.remove('hid');
  bc.className='bt bt-w';
  bc.classList.remove('hid');
  bc.textContent='获取通行证';
  bc.onclick=startAuth;
  pick.classList.add('hid');
  nc.classList.add('hid');nc.textContent='';
  renderSetupCard();
  renderActiveHome();
  if(es.status==='capturing'){
    badge.textContent='正在获取通行证';title.textContent='按引导打开寿司郎小程序';copy.textContent='只需要打开小程序让页面加载一次，不要提交预约，也不要取消任何订单。获取到必要信息后下方进度会自动点亮。';
    b.textContent='获取中';b.className='bt bt-y bt-l';b.onclick=sC;
    bc.classList.add('hid');
  }else if(es.status==='booking'||es.status==='sniping'){
    badge.textContent='正在执行';title.textContent=es.status==='sniping'?'蹲未来预约时段运行中':'自动抢预约运行中';copy.textContent=es.message||'页面可以保持打开；抢到未来预约后会保存记录、发送通知并停止。';
    b.textContent='运行中';b.className='bt bt-r bt-l';b.onclick=sB;
    bc.classList.add('hid');
  }else if(es.status==='success'){
    badge.textContent='已成功';title.textContent='已拿到预约 🍣';copy.textContent=es.message||'预约信息已保存。请以寿司郎小程序里的最终记录为准。';
    b.textContent='查看我的单据';b.className='bt bt-r bt-l';b.onclick=()=>enterAdvanced('re');
    bc.textContent='继续看排队';bc.onclick=()=>go('qt');
  }else if(es.status==='error'){
    badge.textContent='需要处理';title.textContent='运行遇到问题';copy.textContent='先看错误原因和建议。重新开始前，不会自动取消你的预约或排队号。';
    b.textContent=hc?'查看可约日历':'先看实时排队';b.className='bt bt-y bt-l';b.onclick=hc?(()=>enterAdvanced('ca')):(()=>go('qt'));
    bc.textContent=hc?'重新获取通行证':'获取通行证';
    bc.onclick=startAuth;
    nc.classList.remove('hid');nc.innerHTML='<b>错误</b><br><code style="word-break:break-all">'+esc(es.message||'(无错误信息)')+'</code><br><br><b>建议</b><br>'+esc(explainMsg(es.message));
  }else if(!hc){
    badge.textContent='第一次使用';title.textContent='想吃寿司郎？先看看现在排多久';copy.textContent='看门店、排队和叫号预测完全不需要通行证；只有抢未来预约、远程取号、读取我的单据才需要。';
    b.classList.add('hid');
    pick.classList.remove('hid');
    bc.classList.add('hid');
  }else{
    const hasStores=(pr.selected_stores||[]).length>0;
    if(!hasStores){
      badge.textContent='通行证已就绪';title.textContent='下一步：选门店和偏好';copy.textContent='抢未来预约前，需要先选门店、人数、桌型和时间偏好。只看排队仍然可以直接使用。';
      b.textContent='设置门店和偏好';b.className='bt bt-y bt-l';b.onclick=openSnPrefs;
      bc.textContent='先看实时排队';bc.onclick=()=>go('qt');
    }else{
      badge.textContent='准备就绪';title.textContent='今天怎么吃？';copy.textContent='通行证和门店偏好都已就绪。可以查未来可约日历直接预约；目标明确就交给自动抢预约。';
      b.textContent='查可约时段';b.className='bt bt-r bt-l';b.onclick=()=>enterAdvanced('ca');
      bc.textContent='自动抢预约';
      bc.className='bt bt-o';
      bc.onclick=()=>enterAdvanced('sn');
    }
  }
}
function uE(){
  const box=el('eb'),bs=el('bs'),s=es||{status:'idle'};
  if(!box){return}  // #eb 只在首页 DOM；SSE 在任意页面都可能触发 uE，元素不存在时直接跳过，避免 .classList 抛错中断后续状态更新
  const label={idle:'就绪',capturing:'正在获取通行证',booking:'正在抢预约',sniping:'蹲预约中',success:'预约成功',error:'需要处理'}[s.status]||s.status;
  const desc=s.message||({idle:'等待下一步。',capturing:'等待小程序请求。',booking:'正在查询未来预约时段。',sniping:'蹲未来预约窗口运行中。',success:'已保存预约信息。',error:'请查看日志。'}[s.status]||'');
  box.className='engine '+s.status+(s.status==='idle'?' hid':'');box.innerHTML='<div class="row"><span class="dot"></span><strong>'+esc(label)+'</strong></div><p>'+esc(desc)+'</p>';
  if(s.status==='booking'&&s.attempts)box.innerHTML+='<p>已查询 '+s.attempts+' 次</p>';
  if(s.status==='capturing'){
    box.innerHTML+=captureProgressHTML(s);
    if(s.warning)box.innerHTML+='<p class="cstep-sub warn">⚠ '+esc(s.warning)+'</p>';
    if(s.capture&&s.capture.wechat&&(pf==='windows'||pf==='darwin'))box.innerHTML+=wechatLightHTML(s.capture.wechat);
  }
  if(s.status==='error'){
    const ek=errorFromKind(s);
    box.innerHTML+='<div class="err-card"><b>'+esc(ek.t)+'</b><p>'+ek.d+'</p><div class="fl g8 fw"><button class="bt bt-r" onclick="'+ek.act+'">'+esc(ek.btn)+'</button><button class="bt bt-o" onclick="openDiagnostics()">打开诊断</button></div></div>';
  }
  if(bs)bs.classList.toggle('hid',!isRun());
  const cb=el('cb');
  if(s.status==='capturing'&&s.capture){if(cb)cb.classList.remove('hid');rG(s.capture)}else if(s.status!=='capturing'){if(cb)cb.classList.add('hid')}
}
function remTab(t){const once=t==='once';el('remOnce').classList.toggle('hid',!once);el('remDaily').classList.toggle('hid',once);el('remTabOnce').classList.toggle('on',once);el('remTabDaily').classList.toggle('on',!once);el('qdrCreateBtn')?.classList.toggle('hid',!once)}
function expandSnPrefs(){const t=el('snPrefsTime');if(t)t.open=true;const d=el('snPrefs');if(d){d.open=true;d.scrollIntoView({behavior:'smooth',block:'start'})}}
function scrollSnSection(id){const d=el(id);if(!d)return;d.scrollIntoView({behavior:'smooth',block:'start'});d.classList.add('sn-focus');setTimeout(()=>d.classList.remove('sn-focus'),1000)}
async function openSnPrefs(){await enterAdvanced('sn');if(cp==='sn')setTimeout(expandSnPrefs,80)}
async function openSettingsFold(id){if(currentUIMode()!=='advanced'&&ADVANCED_FOLDS.has(id)){await enterAdvanced('se');if(currentUIMode()!=='advanced')return}else go('se');setTimeout(()=>{const d=el(id);if(d){d.open=true;d.scrollIntoView({behavior:'smooth',block:'start'})}},80)}
function openDiagnostics(){openSettingsFold('fold-safe');setTimeout(()=>lD(),120)}
function focusNotifySettings(){go('se');setTimeout(()=>{const d=el('fold-notify');if(d)d.open=true;const x=el('nf');if(x){x.scrollIntoView({behavior:'smooth',block:'center'});x.focus()}},60)}
function renderSettingsStatus(){
 const box=el('settingsStatus');if(!box)return;
 const stale=hc&&ah&&ah.status==='stale';
 const softWarn=hc&&!stale&&am&&am.soft_warn;
 const ageStr=(am&&am.age_label)?am.age_label:'';
 const cloudConn=!!cloudAuth.connected,cloudCfg=!!cloudAuth.configured;
 const cloudBaseOK=!!cloudAuth.baseline_connected;
 const spOK=!!(spState&&(spState.running||spState.enabled||spState.sample_runs>0));
 const authDesc=!hc?'看排队不需要；抢未来预约、远程取号、读单据才需要':stale?'可能已失效，建议重新获取':softWarn?('已用 '+ageStr+'，接近以往平均有效期，建议提前续期'):('已就绪'+(ageStr?('，已用 '+ageStr):'')+'；接近过期会自动提醒');
 const items=[
  {t:'寿司郎通行证 🎫',d:authDesc,s:!hc?'warn':stale?'bad':softWarn?'warn':'ok',a:!hc?{l:'去获取',f:'openAuthWizard()'}:stale?{l:'重新获取',f:'resetAuthAndStart()'}:softWarn?{l:'提前续期',f:'resetAuthAndStart()'}:{l:'看我的单据',f:"enterAdvanced('re')"}},
  {t:'通知渠道',d:nfc?('已配置'+(notifyChannels.length?('：'+notifyChannels.join('、')):'')):'不配置就收不到叫号提醒和抢到通知',s:nfc?'ok':'warn',a:nfc?{l:'测试通知',f:"tN('all')"}:{l:'去配置',f:'focusNotifySettings()'}}
 ];
 if(currentUIMode()==='advanced'){
  items.push({t:'GitHub 线上基准',d:cloudBaseOK?('GitHub 已登录，线上数据库已验证，图表可叠加全国线上基准'):cloudConn?('GitHub 已登录，线上数据库待验证。验证前图表会继续优先用本机数据'):'登录后叫号预测可叠加全国线上基准（可选）',s:cloudBaseOK?'ok':'warn',a:cloudConn?{l:'退出',f:'logoutCloudAuth()'}:{l:'登录 GitHub',f:'startCloudLogin()'}});
  const calib=accCalibrated>0?('；已用实测误差校准 '+accCalibrated+' 家店'):'';
  items.push({t:'预测数据',d:(spOK?'采集中，“几点能吃上”会越来越准':'公开曲线已默认记录；想更准可开启凭证态采集')+calib,s:spOK?'ok':'warn',a:{l:'配置',f:"openSettingsFold('fold-sm')"}});
 }
 box.innerHTML=healthStripHTML(items);
}
async function checkUpdate(){try{const u=await(await fetch('/api/update')).json(),b=el('updBox');if(!b)return;if(u.current_version==='dev'){b.classList.add('hid');return}if(u.update_available){b.classList.remove('hid');b.innerHTML='<h2>版本更新</h2><div class="ps"><b>'+esc(u.latest_version)+'</b><span class="line">当前 '+esc(u.current_version)+'</span></div><a class="bt bt-w bt-s mt16" href="'+escA(u.url||'#')+'" target="_blank">打开 Release</a>'}else b.classList.add('hid')}catch(e){}}
function rG(c){const cg=el('cg');if(!cg){return}cg.innerHTML=need.map(k=>'<div class="ci '+(c[k]?'ok':'')+'">'+fieldName(k)+'</div>').join('')}
function fieldName(k){return {x_app_code:'App Code',query_auth:'查询凭证',reservation_auth:'预约凭证',user_agent:'设备信息',referer:'小程序来源',wechat_id:'微信 ID',phone_number:'手机号',store_ids:'门店'}[k]||k}
async function sC(){try{const d=await(await fetch('/api/engine/capture',{method:'POST'})).json();if(d.error)toast(d.error);await loadStatus();}catch(e){toast('启动失败')}}
async function resetAuthOnly(ask){if(ask!==false){if(!await confirmDialog({title:'重置寿司郎通行证？',body:'这会清除本机保存的寿司郎通行证，并停止未执行的自动取号计划；不会取消已经拿到的预约或排队号。\\n通行证会过期，也可能被手机端登录顶掉。重置后需要重新获取。',ok:'重置通行证',cancel:'取消'}))return false}try{const d=await safeFetch('/api/auth/reset',{method:'POST'});hc=false;ah=d.auth_health||{status:'unknown'};await loadStatus();toast(d.message||'已重置通行证');return true}catch(e){toast('重置通行证失败：'+String(e.message||e));return false}}
async function resetAuthAndStart(){if(!await resetAuthOnly(true))return;openAuthWizard()}
async function rST(){if(!await confirmDialog('重置通行证获取状态？会断开当前临时代理并清理残留，之后可点「获取通行证」手动重新连接。'))return;try{const d=await safeFetch('/api/engine/reset',{method:'POST'});if(d.error){toast(d.error);return}await loadStatus();toast('已重置通行证获取状态，点「获取通行证」可重新连接')}catch(e){toast('重置失败：'+String(e.message||e))}}
async function sB(){if(!await ensureNotifyConfigured('抢到预约'))return;if(!await confirmDialog('启动自动抢预约？\\n这会按你的门店和时段偏好尝试创建寿司郎预约；成功后会停止并保存到“我的单据”。\\n不会取消你已有的预约或排队号。'))return;try{const d=await(await fetch('/api/engine/booking',{method:'POST'})).json();if(d.error)toast(d.error);await loadStatus();}catch(e){toast('启动失败')}}
async function sE(){try{await fetch('/api/engine/stop',{method:'POST'});await loadStatus();}catch(e){}}
function startAuth(){if(hc&&(ah&&ah.status==='stale')){resetAuthAndStart();return}openAuthWizard()}
function mA(){hc?sB():startAuth()}
const MASCOT_KINDS=['salmon','maguro','saba','tamago','ebi','tako','unagi','hotate','ikura','uni','maki','kappa'];
function mascotFace(mood,fy){
 const my=fy+7;
 const eyes=mood==='sleep'?'<path d="M26 '+fy+'q3 3 6 0M40 '+fy+'q3 3 6 0" stroke="#3A3530" stroke-width="2.4" fill="none" stroke-linecap="round"/>':'<circle cx="29" cy="'+fy+'" r="2.6" fill="#3A3530"/><circle cx="43" cy="'+fy+'" r="2.6" fill="#3A3530"/>';
 const mouth=mood==='sad'?'<path d="M32 '+(my+2)+'q4 -3.5 8 0" stroke="#3A3530" stroke-width="2.2" fill="none" stroke-linecap="round"/>':mood==='happy'?'<path d="M32 '+my+'q4 4.5 8 0" stroke="#3A3530" stroke-width="2.2" fill="none" stroke-linecap="round"/>':'<path d="M33 '+(my+1)+'h6" stroke="#3A3530" stroke-width="2.2" stroke-linecap="round"/>';
 const blush=mood==='happy'?'<circle cx="23" cy="'+(fy+5)+'" r="2.4" fill="#F2A6A0" opacity=".75"/><circle cx="49" cy="'+(fy+5)+'" r="2.4" fill="#F2A6A0" opacity=".75"/>':'';
 return eyes+mouth+blush}
function mascotSVG(mood,size,kind){size=size||64;if(!kind||kind==='rand')kind=MASCOT_KINDS[Math.floor(Math.random()*MASCOT_KINDS.length)];
 const rice='<ellipse cx="36" cy="44" rx="27" ry="15" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/>';
 let body='';
 const topShape='M9 36Q36 12 63 36q1 6-5 7Q36 26 14 43q-6-1-5-7z';
 if(kind==='maki'){body='<circle cx="36" cy="32" r="27" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="36" cy="32" r="20" fill="#FFFDF6"/><circle cx="36" cy="23" r="6.5" fill="#F8875F"/><circle cx="28" cy="29" r="3.5" fill="#7FBF6C"/><circle cx="44" cy="29" r="3.5" fill="#FFD566"/>'+mascotFace(mood,37)}
 else if(kind==='kappa'){body='<circle cx="36" cy="32" r="27" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="36" cy="32" r="20" fill="#FFFDF6"/><circle cx="36" cy="23" r="7" fill="#6FB35D" stroke="#578F47" stroke-width="1.5"/><circle cx="36" cy="23" r="2.6" fill="#DFF0D6"/>'+mascotFace(mood,37)}
 else if(kind==='tamago'){body=rice+'<rect x="11" y="17" width="50" height="21" rx="10" fill="#FFD566" stroke="#E8B73F" stroke-width="2"/><rect x="31" y="13" width="10" height="30" rx="4" fill="#33433A"/>'+mascotFace(mood,41)}
 else if(kind==='ebi'){body=rice+'<path d="'+topShape+'" fill="#FB9C7C" stroke="#E27D5B" stroke-width="2" stroke-linejoin="round"/><path d="M24 32q4-4 8-5M36 26q5-1 9 0M48 27q5 1 8 4" stroke="#FFF1EA" stroke-width="3" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='maguro'){body=rice+'<path d="'+topShape+'" fill="#E8485C" stroke="#C9394B" stroke-width="2" stroke-linejoin="round"/><path d="M22 32q14 -9 28 0" stroke="#F8A8B2" stroke-width="2" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='unagi'){body=rice+'<path d="'+topShape+'" fill="#8C5A38" stroke="#6F4527" stroke-width="2" stroke-linejoin="round"/><path d="M20 33q7-5 14-6M42 26q7 0 12 4" stroke="#5C3A1F" stroke-width="2.5" fill="none" stroke-linecap="round"/><rect x="31" y="13" width="10" height="30" rx="4" fill="#33433A"/>'+mascotFace(mood,41)}
 else if(kind==='ikura'){body='<ellipse cx="36" cy="42" rx="24" ry="17" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/><rect x="12" y="18" width="48" height="22" rx="6" fill="#33433A" stroke="#27332C" stroke-width="2"/><circle cx="26" cy="15" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="38" cy="12" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="47" cy="16" r="6" fill="#FF9D45" stroke="#E8832E" stroke-width="1.5"/><circle cx="33" cy="17" r="5" fill="#FFB066" stroke="#E8832E" stroke-width="1.5"/>'+mascotFace(mood,47)}
 else if(kind==='uni'){body='<ellipse cx="36" cy="42" rx="24" ry="17" fill="#FFFDF6" stroke="#E5E0DB" stroke-width="2"/><rect x="12" y="18" width="48" height="22" rx="6" fill="#33433A" stroke="#27332C" stroke-width="2"/><path d="M20 10l-3-5M28 7l-1-5M37 6v-5M46 8l2-5M54 11l3-4" stroke="#B9842B" stroke-width="2" stroke-linecap="round"/><circle cx="26" cy="15" r="7" fill="#DFA63C" stroke="#C08A2D" stroke-width="1.5"/><circle cx="38" cy="12" r="7" fill="#E7B14A" stroke="#C08A2D" stroke-width="1.5"/><circle cx="49" cy="15" r="6" fill="#DFA63C" stroke="#C08A2D" stroke-width="1.5"/>'+mascotFace(mood,47)}
 else if(kind==='tako'){body=rice+'<path d="'+topShape+'" fill="#E89BB0" stroke="#C97891" stroke-width="2" stroke-linejoin="round"/><circle cx="27" cy="30" r="2.4" fill="#F8D8E1"/><circle cx="38" cy="27" r="2.4" fill="#F8D8E1"/><circle cx="48" cy="31" r="2.4" fill="#F8D8E1"/>'+mascotFace(mood,41)}
 else if(kind==='hotate'){body=rice+'<path d="'+topShape+'" fill="#F6E9D2" stroke="#DCC49C" stroke-width="2" stroke-linejoin="round"/><path d="M27 29q-1 5-2 9M36 26q0 6 0 12M45 29q1 5 2 9" stroke="#E3CFA8" stroke-width="2.5" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else if(kind==='saba'){body=rice+'<path d="'+topShape+'" fill="#AFC4D8" stroke="#85A0B8" stroke-width="2" stroke-linejoin="round"/><path d="M22 31q5-6 9-7M35 25q4-2 8-1M46 26q5 2 8 6" stroke="#5E7A93" stroke-width="2.4" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 else{body=rice+'<path d="'+topShape+'" fill="#F8875F" stroke="#E0744C" stroke-width="2" stroke-linejoin="round"/><path d="M20 33q16 -10 32 0" stroke="#FFD9C9" stroke-width="2" fill="none" stroke-linecap="round"/>'+mascotFace(mood,41)}
 return '<svg class="mascot" width="'+size+'" height="'+size+'" viewBox="0 0 72 64" aria-hidden="true">'+body+'</svg>'}
function mascotRowHTML(mood,size){return '<div class="mascot-row">'+MASCOT_KINDS.map(k=>mascotSVG(mood,size||44,k)).join('')+'</div>'}
function fillPageMascots(){document.querySelectorAll('.pm').forEach(x=>{if(!x.innerHTML)x.innerHTML=mascotSVG(x.dataset.mood||'happy',x.dataset.size?+x.dataset.size:34,x.dataset.kind||'rand')})}
function buildBelt(){const b=el('belt');if(!b)return;
 // 无缝循环：轨道 = 完全相同的两段，translateX(-50%) 回到起点时画面逐像素一致。
 // 一段必须铺得比视口还宽，否则宽屏右侧会露出空白。itemW = 盘子 48 + 间距 56。
 const itemW=104,need=Math.max(window.innerWidth||1280,1280)+itemW;
 let half=[];while(half.length*itemW<need)half=half.concat(MASCOT_KINDS);
 const seg=half.map(k=>'<div class="belt-item">'+mascotSVG('plain',34,k)+'<i class="plate"></i></div>').join('');
 const dur=Math.round(half.length*itemW/26); // 恒定 ~26px/s，与宽度无关
 b.innerHTML='<div class="belt-track" style="animation-duration:'+dur+'s">'+seg+seg+'</div>'}
let beltResizeT=null;
window.addEventListener('resize',()=>{clearTimeout(beltResizeT);beltResizeT=setTimeout(buildBelt,400)});
function lsGet(k){try{return localStorage.getItem(k)||''}catch(e){return''}}
function lsSet(k,v){try{localStorage.setItem(k,v)}catch(e){}}
function rememberStores(k,ids){lsSet(k,(ids||[]).join(','))}
function recallStores(k){const v=lsGet(k);return v?v.split(',').filter(Boolean):[]}
function openFirstUseWizard(){let ov=el('firstUse');if(!ov){ov=document.createElement('div');ov.id='firstUse';ov.className='ov';document.body.appendChild(ov)}
 ov.innerHTML='<div class="ovc" style="width:min(720px,96vw)"><div class="fl ai jb mb16"><b>第一次用，先选一条路</b><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="closeFirstUseWizard();go(\'gu\')">先看机制图</button><button class="bt bt-w bt-s" onclick="closeFirstUseWizard()">稍后</button></div></div>'
 +'<p class="mu" style="line-height:1.75;margin-top:-6px">看排队和预测都是只读；约未来、远程取号、读单据时再获取通行证。</p>'
 +'<div class="first-use-grid">'
 +'<button class="first-use-card read" type="button" onclick="closeFirstUseWizard();openGuestStorePicker()"><span>今天去吃</span><b>选门店看排队</b><small>搜城市或门店名，先看哪家排得少。</small></button>'
 +'<button class="first-use-card read" type="button" onclick="closeFirstUseWizard();go(\'qd\')"><span>我有号码</span><b>算这个号几点能吃上</b><small>填当天排队号，按叫号进度估时间。</small></button>'
 +'<button class="first-use-card auth" type="button" onclick="closeFirstUseWizard();currentUIMode()===\'advanced\'?go(\'ca\'):enterAdvanced(\'ca\')"><span>想约未来</span><b>查未来预约</b><small>先看日历，要提交预约前再确认。</small></button>'
 +'</div></div>';
 ov.onclick=e=>{if(e.target===ov)closeFirstUseWizard()};
 ov.classList.remove('hid');ov.style.display='flex'}
function closeFirstUseWizard(){const ov=el('firstUse');if(ov){ov.classList.add('hid');ov.style.display='none'}}
let authWizPoll=null;
let awz={step:1,device:'',cap:null};
function awzSave(){try{localStorage.setItem('sushiro_wizard_state',JSON.stringify({step:awz.step,device:awz.device,cap:awz.cap}))}catch(e){}}
function awzClear(){awz={step:1,device:'',cap:null};try{localStorage.removeItem('sushiro_wizard_state');localStorage.removeItem('sushiro_wizard_draft')}catch(e){}}
function awzGo(n){awz.step=n;awzSave();authWizStep(n)}
function awzDevice(d){awz.device=d;awz.step=2;awzSave();authWizStep(2)}
function awzDraft(v){try{localStorage.setItem('sushiro_wizard_draft',v)}catch(e){}}
function awzStartPC(){closeAuthWizard();sC();go('da');toast('已启动 PC 微信自动捕获：打开 PC 微信里的寿司郎小程序，点一次门店，再真的排队/预约一下（之后可取消）')}
function openAuthWizard(){let ov=el('authWiz');if(!ov){ov=document.createElement('div');ov.id='authWiz';ov.className='ov';document.body.appendChild(ov)}
 try{const s=JSON.parse(localStorage.getItem('sushiro_wizard_state')||'null');if(s&&s.step)awz={step:s.step,device:s.device||'',cap:s.cap||null}}catch(e){}
 if(awz.step>1&&awz.step<5&&!awz.device)awz.step=1;
 if(awz.step===5)awz.step=4;
 ov.classList.remove('hid');ov.style.display='flex';authWizStep(awz.step)}
function closeAuthWizard(){const ov=el('authWiz');if(ov){ov.classList.add('hid');ov.style.display='none'}if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}fetch('/api/mobile-auth/stop',{method:'POST',headers:{'X-Sushiro-CSRF':csrfToken}}).catch(()=>{})}
const AWZ_STEPS=['选设备','抓一次','传到电脑','粘贴解析','验证'];
function awzBar(cur){return'<div class="wsteps">'+AWZ_STEPS.map((s,i)=>{const n=i+1;return'<div class="wstep '+(n<cur?'done':n===cur?'on':'')+'"><i>'+(n<cur?'✓':n)+'</i>'+s+'</div>'}).join('')+'</div>'}
function authWizShell(cur,body){return'<div class="ovc"><div class="fl ai jb mb16"><b>获取通行证 🎫 <span class="mu" style="font-weight:400">约 3 分钟 · 全程只在本机处理</span></b><button class="bt bt-w bt-s" onclick="closeAuthWizard()">稍后再说</button></div>'+(cur?awzBar(cur):'')+'<div style="overflow:auto">'+body+'</div></div>'}
// authCaptureFlowSVG 画"两类请求"分步图——解决"抓不全"的视觉化方案：
// 门店请求带查询auth+UA+referer，排队/预约请求带预约auth+wechatId+手机号，两者都要抓。
function authCaptureFlowSVG(){return ''+
'<svg viewBox="0 0 520 240" class="awz-flow" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="凭证采集流程">'+
'<defs><marker id="awzArr" markerWidth="8" markerHeight="8" refX="6" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="#B81C22"/></marker></defs>'+
'<rect x="10" y="20" width="150" height="62" rx="10" fill="#FFF1F1" stroke="#B81C22" stroke-width="1.5"/>'+
'<text x="85" y="42" text-anchor="middle" font-size="13" font-weight="700" fill="#B81C22">① 点一家门店</text>'+
'<text x="85" y="60" text-anchor="middle" font-size="11" fill="#666">产生「查询请求」</text>'+
'<text x="85" y="74" text-anchor="middle" font-size="10" fill="#999">查询auth · UA · referer</text>'+
'<rect x="185" y="20" width="150" height="62" rx="10" fill="#ECF7EF" stroke="#21823F" stroke-width="1.5"/>'+
'<text x="260" y="42" text-anchor="middle" font-size="13" font-weight="700" fill="#21823F">② 排队或预约一下</text>'+
'<text x="260" y="60" text-anchor="middle" font-size="11" fill="#666">产生「预约请求」（之后可取消）</text>'+
'<text x="260" y="74" text-anchor="middle" font-size="10" fill="#999">预约auth · 微信ID · 手机号</text>'+
'<path d="M85,82 L85,108 L260,108 L260,82" fill="none" stroke="#B81C22" stroke-width="1.5" marker-end="url(#awzArr)"/>'+
'<rect x="170" y="115" width="170" height="50" rx="10" fill="#FBFAF8" stroke="#999" stroke-width="1.5"/>'+
'<text x="255" y="137" text-anchor="middle" font-size="12" font-weight="700" fill="#333">③ 导出 cURL / 请求头</text>'+
'<text x="255" y="153" text-anchor="middle" font-size="10" fill="#999">两条请求都选上</text>'+
'<path d="M255,165 L255,183" fill="none" stroke="#B81C22" stroke-width="1.5" marker-end="url(#awzArr)"/>'+
'<rect x="155" y="185" width="200" height="40" rx="10" fill="#FFF5D8" stroke="#B67800" stroke-width="1.5"/>'+
'<text x="255" y="210" text-anchor="middle" font-size="12" font-weight="700" fill="#6F4B00">④ 粘贴到电脑 → 自动填好</text>'+
'<text x="455" y="50" text-anchor="middle" font-size="11" fill="#B81C22" font-weight="700">⚠</text>'+
'<text x="455" y="66" text-anchor="middle" font-size="10" fill="#B81C22">只抓一类</text>'+
'<text x="455" y="80" text-anchor="middle" font-size="10" fill="#B81C22">就抓不全</text>'+
'</svg>'}
function awzChecklistHTML(){const c=awz.cap||{},got=need.filter(k=>c[k]).length;return'<div class="mu mt8" style="font-weight:800">字段捕获进度 '+got+'/'+need.length+'</div><div class="cg mt8">'+need.map(k=>'<div class="ci '+(c[k]?'ok':'')+'">'+(c[k]?'✓ ':'')+fieldName(k)+'</div>').join('')+'</div>'}
function awzToolHint(){const d=awz.device;return d==='android'?'安装 <b>Reqable</b>（推荐，免费）或 <b>HttpCanary</b>，按引导装好并信任证书，开启抓包。<br><span class="mu">⚠ 安卓 7+ 需把 CA 装成系统证书（用 Magisk 模块最省事），否则抓不到 HTTPS。</span>':'App Store 安装 <b>Reqable</b>（推荐，免费社区版够用）或 <b>HTTP Catcher</b>，按引导装好 CA 并在「设置→通用→关于本机→证书信任设置」里完全信任，然后开启抓包。<br><span class="mu">⚠ 别用 Stream——它抓微信小程序的提交请求（POST）经常抓不全。</span>'}
function authWizStep(step){const ov=el('authWiz');if(!ov)return;if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}
  if(step===1){
   const intro='<h3 style="margin:0 0 4px">第 1 步：怎么拿？</h3><p class="mu">通行证是寿司郎小程序和服务器对话用的身份凭证，抢预约、远程取号、读单据都靠它。原始字段只保存在本机，不会上传。</p>';
   const pasteFirst='<div class="fl g8 fw mt16"><button class="bt bt-r bt-l" onclick="awzGo(4)">📋 已有抓包内容？直接粘贴（最简单）</button></div><p class="mu mt8">从抓包工具/别人那里拿到请求内容（cURL/JSON/请求头都行）就粘进来，免装证书免设代理。没有的话，下面按设备走自动抓：</p>';
   const phones='<button class="bt bt-r" onclick="awzDevice(\'ios\')">📱 iPhone 手机抓包</button><button class="bt bt-r" onclick="awzDevice(\'android\')">🤖 安卓手机抓包</button>';
   const autoHint='<div class="why">💡 手机和电脑连同一个 Wi-Fi、路由器没开隔离？可以试 <button class="bt bt-w bt-s" onclick="authWizStep(\'auto\')">同 Wi-Fi 自动代理抓</button>，手机不用装抓包工具。</div>';
   const body=pf==='windows'
    ?intro+pasteFirst+'<div class="wnum"><b class="n">!</b><div><b>Windows 上的 PC 微信抓不到小程序请求</b>，需要用手机拿一次，两条路任选：手机抓包（最稳），或同 Wi-Fi 自动代理。</div></div><div class="fl g8 fw mt8">'+phones+'</div>'+autoHint
    :intro+pasteFirst+'<div class="fl g8 fw mt8"><button class="bt bt-r" onclick="awzStartPC()">💻 PC 微信自动抓（本机最省事）</button></div><div class="fl g8 fw mt8">'+phones+'</div>'+autoHint;
   ov.innerHTML=authWizShell(1,body)}
  else if(step===2){ov.innerHTML=authWizShell(2,'<h3 style="margin:0 0 4px">第 2 步：在手机上“抓一次”</h3><p class="mu">'+awzToolHint()+'</p>'+authCaptureFlowSVG()+'<div class="wnum"><b class="n">1</b><div>打开微信里的<b>寿司郎小程序</b></div></div><div class="wnum"><b class="n">2</b><div>随便点开一家门店 <span class="mu">← 这一下产生「查询请求」</span></div></div><div class="wnum"><b class="n">3</b><div>找一家店<b>真的排队或预约一下</b> <span class="mu">← 这下产生「预约请求」（含微信ID/手机号）；抓到后再去取消即可</span></div></div><div class="why">💡 为什么要点两次？门店查询和排队/预约是两类请求，各含通行证的一半信息，缺一不可。光看「我的预约」列表不行，得真的提交一次排队/预约。</div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(1)">← 上一步</button><button class="bt bt-r" onclick="awzGo(3)">我点完了，下一步 →</button></div>')}
  else if(step===3){ov.innerHTML=authWizShell(3,'<h3 style="margin:0 0 4px">第 3 步：把抓到的内容传到电脑</h3><div class="wnum"><b class="n">1</b><div>在抓包工具里找到 <code>crm-cn-prd.sushiro.com.cn</code> 的请求——<b>第 2 步点门店、排队/预约产生的两条都要选</b>（长按多选），少一条就抓不全</div></div><div class="wnum"><b class="n">2</b><div>导出 / 复制成 <b>cURL</b>（首选，含完整请求头和提交内容）或 <b>HAR</b>。Reqable/HttpCanary：长按请求 → 分享/导出 →「复制为 cURL」</div></div><div class="wnum"><b class="n">3</b><div>手机微信搜「<b>文件传输助手</b>」发给它 → 电脑微信打开同一会话复制</div></div><div class="why">💡 手机和电脑不在同一网络也没关系，文件传输助手走微信通道。两条请求的内容都粘进下一步即可（不用分开粘）。</div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(2)">← 上一步</button><button class="bt bt-r" onclick="awzGo(4)">内容已复制，去粘贴 →</button></div>')}
  else if(step===4){let draft='';try{draft=localStorage.getItem('sushiro_wizard_draft')||''}catch(e){}
   ov.innerHTML=authWizShell(4,'<h3 style="margin:0 0 4px">第 4 步：粘贴并解析</h3><p class="mu">支持 JSON / cURL / 原始请求头。第一次没抓齐也没关系：<b>不要清空</b>，把新抓的内容接着粘在后面，再点一次解析。</p><div class="fg mt8"><label>抓包内容</label><textarea id="awImport" oninput="awzDraft(this.value)" placeholder="粘贴包含 X-App-Code、Authorization、User-Agent、Referer、wechatId、phoneNumber、storeId 的请求…"></textarea></div><div id="awChecklist">'+awzChecklistHTML()+'</div><div id="awImportState" class="diag-detail mt8 hid"></div><div class="fl ai jb mt16"><button class="bt bt-w bt-s" onclick="awzGo(3)">← 上一步</button><button class="bt bt-r" onclick="authWizImport()">解析并保存 →</button></div>');
   const ta=el('awImport');if(ta&&draft)ta.value=draft}
  else if(step===5){ov.innerHTML=authWizShell(5,'<div id="awVerify"></div>');awzVerify()}
  else if(step==='auto'){ov.innerHTML=authWizShell(0,'<h3 style="margin:0 0 4px">自动代理抓（同 Wi-Fi）</h3><p class="mu">手机不用装抓包工具：电脑临时帮手机“看一眼”寿司郎的网络请求（本机 MITM 代理，只解密寿司郎域名，其他流量不读取）。跟着信号灯走：</p><div id="awAutoStages"></div><div id="awAuto" class="mt8"><span class="mu">正在启动…</span></div><div class="fl g8 fw mt16"><button class="bt bt-w bt-s" onclick="awzGo(1)">← 返回选设备</button><button class="bt bt-w bt-s" onclick="closeAuthWizard()">停止并关闭</button></div>');authWizStartAuto()}}
async function authWizStartAuto(){try{const d=await safeFetch('/api/mobile-auth/start',{method:'POST'},12000);authWizRenderAuto(d);if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}authWizPoll=setInterval(authWizPollAuto,2500)}catch(e){const b=el('awAuto');if(b)b.innerHTML='<span class="bad">启动失败：'+esc(String(e.message||e))+'</span>'}}
async function authWizPollAuto(){try{const d=await safeFetch('/api/mobile-auth');authWizRenderAuto(d);if(d.saved||d.config_complete){if(authWizPoll){clearInterval(authWizPoll);authWizPoll=null}await loadStatus();toast('已捕获完成！记得把手机 Wi-Fi 代理改回关闭。');awz.step=5;awzSave();authWizStep(5)}}catch(e){}}
function awzAutoStages(d){const cap=d.capture||{},anyField=need.some(k=>cap[k]),done=!!(d.saved||d.config_complete);const st=[['电脑侧服务已启动，二维码可扫',!!d.active],['捕获到小程序请求',anyField],['字段齐全，已保存',done]];return st.map(x=>'<div class="strip"><span class="st '+(x[1]?'ok':'warn')+'">'+(x[1]?'✓':'…')+'</span><div><b>'+esc(x[0])+'</b></div></div>').join('')}
function authWizRenderAuto(d){const b=el('awAuto'),sg=el('awAutoStages');if(sg)sg.innerHTML=awzAutoStages(d);if(!b)return;const urls=d.guide_urls||[],hosts=d.hosts||[];b.innerHTML='<div class="wnum"><b class="n">1</b><div>手机微信「扫一扫」右侧二维码打开引导页，按页面提示<b>安装并信任 CA 证书</b>（iPhone 还需在 设置→通用→关于本机→证书信任设置 里完全信任）</div></div><div class="wnum"><b class="n">2</b><div>把手机 Wi-Fi 的 HTTP 代理设为下方 <b>电脑IP:端口</b></div></div><div class="wnum"><b class="n">3</b><div>彻底关掉再打开微信，进寿司郎小程序点一次门店，再真的排队/预约一下（之后可取消）</div></div><div class="mt8" style="text-align:center">'+((d.active&&d.qr_svg)?d.qr_svg:'<span class="mu">二维码加载中…</span>')+'</div><div class="ps mt8">'+(urls.length?'<b>扫码或手机浏览器打开：</b><br>'+urls.map(u=>'<code>'+esc(u)+'</code>').join('<br>'):'')+'<div class="mu mt8"><b>Wi-Fi 代理：</b>'+hosts.map(h=>'<code>'+esc(h)+':'+esc(d.proxy_port||'')+'</code>').join(' ')+'</div><div class="mu mt8">扫码打不开 / 连不上？多半是路由器开了 AP（客户端）隔离，<button class="bt bt-w bt-s" onclick="awzDevice(awz.device||\'ios\')">改用手动抓（更稳）</button></div></div><div class="diag-detail mt8">'+esc(d.message||'')+'</div>'}
async function authWizImport(){const txt=(el('awImport')?.value||'').trim();if(!txt){toast('请先粘贴抓到的内容');return}const st=el('awImportState');if(st){st.classList.remove('hid');st.innerHTML='解析中…'}
 try{const d=await safeFetch('/api/auth/import',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({text:txt})},15000);
  const cap={};need.forEach(k=>{cap[k]=!!(d.capture&&d.capture[k])});awz.cap=cap;awzSave();
  const ck=el('awChecklist');if(ck)ck.innerHTML=awzChecklistHTML();
  if(d.saved){await loadStatus();awz.step=5;awzSave();authWizStep(5);return}
  const miss=d.missing||[],fix=[];
  if(miss.some(x=>/预约|微信|手机/.test(x)))fix.push('回到第 2 步，真的排队/预约一次（之后可取消）');
  if(miss.some(x=>/查询|Referer|门店/i.test(x)))fix.push('回到第 2 步，再点一次门店/排队');
  if(st)st.innerHTML='<span class="bad">还差一点，缺：</span>'+esc(miss.join('、')||'未知')+'<br><span class="mu">'+(fix.length?esc(fix.join('；'))+'，把新抓的内容接着粘在后面（不要清空），再点解析。':'再补一段包含缺失字段的请求，接着粘在后面即可。')+'</span>'
 }catch(e){if(st)st.innerHTML='<span class="bad">导入失败：'+esc(String(e.message||e))+'</span>'}}
function awzCelebrateHTML(){return'<div class="celebrate">'+mascotRowHTML('happy',50)+'<h3 style="margin:10px 0 4px;font-size:20px">通行证已生效！🍣</h3><p class="mu">抢预约、远程取号、读取我的单据都解锁了。原始凭证只保存在本机。</p><div class="fl g8 fw mt16" style="justify-content:center"><button class="bt bt-r" onclick="closeAuthWizard();enterAdvanced(\'ca\')">去约一个</button><button class="bt bt-w" onclick="closeAuthWizard();go(\'qt\')">先看排队</button><button class="bt bt-w bt-s" onclick="closeAuthWizard()">完成</button></div></div>'}
function awzConfetti(){const host=document.querySelector('#authWiz .celebrate');if(!host)return;const colors=['#B81C22','#F2A93B','#21823F','#F8875F','#4A90D9'];for(let i=0;i<16;i++){const s=document.createElement('span');s.className='confetti';s.style.left=(5+Math.random()*90)+'%';s.style.background=colors[i%colors.length];s.style.animationDelay=(Math.random()*0.5)+'s';host.appendChild(s);setTimeout(()=>s.remove(),2400)}}
async function awzVerify(){const box=el('awVerify');if(!box)return;box.innerHTML='<div class="mascot-wrap">'+mascotSVG('plain',64)+'</div><p class="mu" style="text-align:center">第 5 步：正在用通行证测试基础接口…</p>';
 try{const r=await fetch('/api/auth/probe',{method:'POST'}),d=await r.json();await loadStatus();
  if(d.ok){awzClear();box.innerHTML=awzCelebrateHTML();awzConfetti()}
  else{box.innerHTML='<div class="mascot-wrap">'+mascotSVG('sad',64)+'</div><div class="diag-detail bad">'+authProbeHTML(d)+'</div><div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="awzVerify()">重试</button><button class="bt bt-w bt-s" onclick="awzGo(4)">回到粘贴步骤</button></div>'}
 }catch(e){box.innerHTML='<div class="diag-detail bad">基础接口测试失败：'+esc(String(e.message||e))+'</div><div class="fl g8 fw mt16"><button class="bt bt-r bt-s" onclick="awzVerify()">重试</button><button class="bt bt-w bt-s" onclick="awzGo(4)">回到粘贴步骤</button></div>'}}

function calendarEmptyHTML(kind){const needsAuth=kind==='needs_auth';const copy=needsAuth?'约未来先看日历，再决定要不要预约或蹲点。还没有通行证也能先看今天排队。':'约未来先看日历，再决定要不要预约或蹲点。先选门店，筛掉不想看的时段；已满时段也能拿去蹲点。';const cards=needsAuth?'<button class="calendar-empty-card auth" onclick="startAuth()" type="button"><span>需要通行证</span><b>去获取通行证</b><small>查未来日历、预约和蹲点前都需要。</small></button><button class="calendar-empty-card read" onclick="go(\'qt\')" type="button"><span>今天去吃</span><b>先看今天排队</b><small>不用通行证，先看门店营业和等位。</small></button><button class="calendar-empty-card read" onclick="go(\'gu\')" type="button"><span>不确定</span><b>先看机制图</b><small>分清当天排队号、未来预约和通行证。</small></button>':'<button class="calendar-empty-card auth" onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})" type="button"><span>第一步</span><b>选择门店看日历</b><small>可以多选几家常去门店一起比。</small></button><button class="calendar-empty-card read" onclick="el(\'avOnly\').checked=true;openStorePicker({selected:selStores,onConfirm:applyCalendarStores})" type="button"><span>筛选</span><b>只看可预约</b><small>选完门店后，只留下能直接预约的时段。</small></button><button class="calendar-empty-card action" onclick="go(\'sn\')" type="button"><span>已满</span><b>已满就蹲点</b><small>没有可约时段时，去自动抢预约里添加目标。</small></button>';return'<div class="calendar-empty"><h3>'+(needsAuth?'还没拿到通行证？':'先选门店，看看未来哪天能约')+'</h3><p>'+copy+'</p><div class="calendar-empty-grid">'+cards+'</div></div>'}
async function lC(){await ensureStores();if(!stores.length){el('storeChoices').innerHTML='<span class="mu">未来预约需要通行证；今天排队不用。</span>';el('sc').innerHTML=calendarEmptyHTML('needs_auth');return}if(!selStores.length){el('sc').innerHTML=calendarEmptyHTML('no_store');return}rStoreChoices();rC()}
function rStoreChoices(){const c=el('storeChoices');c.innerHTML=stores.map(s=>'<button class="chip '+(selStores.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>togStore(b.dataset.store))}
function togStore(id){selStores=selStores.includes(id)?selStores.filter(x=>x!==id):selStores.concat(id);if(!selStores.length&&stores[0])selStores=[String(stores[0].id)];rStoreChoices();sd='';rC()}
async function rC(){if(!selStores.length)return;el('sc').innerHTML='<div class="empty">加载中…</div>';const q='stores='+encodeURIComponent(selStores.join(','))+'&available='+(el('avOnly').checked?'1':'0')+'&period='+encodeURIComponent(el('period').value||'all');try{const d=await safeFetch('/api/calendar?'+q);if(d.error){el('sc').innerHTML=loadErrBoxHTML(d.error,'rC()','日历');return}as=[];calErrs=[];(d.stores||[]).forEach(st=>{if(st.error)calErrs.push({store:st.store_name||st.store_id,error:st.error});(st.slots||[]).forEach(s=>as.push({...s,store_name:st.store_name,store_id:st.store_id}))});rDB()}catch(e){el('sc').innerHTML=loadErrBoxHTML(e,'rC()','日历')}}
function setAR(){if(arTimer){clearInterval(arTimer);arTimer=null}const sec=+el('ar').value||0;if(sec>0)arTimer=setInterval(()=>{if(cp==='ca')rC()},sec*1000)}
function fD(d){return parseInt(d.substring(4,6),10)+'/'+parseInt(d.substring(6,8),10)}
function fT(t){return t&&t.length>=4?t.substring(0,2)+':'+t.substring(2,4):t||''}
function nT(t){t=compactTime(t||'');return t.length===4?t+'00':t}
function slotMatchesPrefs(s){const dt=new Date(s.date.substring(0,4)+'-'+s.date.substring(4,6)+'-'+s.date.substring(6,8)),w=dt.getDay(),rs=w===6?(pr.saturday_slots||[]):w===0?(pr.sunday_slots||[]):(pr.weekday_slots||[]),st=nT(s.start),en=nT(s.end||s.start);return rs.some(r=>st>=nT(r.start)&&st<nT(r.end)&&en<=nT(r.end))}
function calendarErrHTML(){return calErrs.length?'<div class="errbox">'+calErrs.map(x=>'<b>'+esc(x.store)+'</b>：'+esc(x.error)).join('<br>')+'<div class="mt8"><button class="bt bt-o bt-s" onclick="startAuth()">重新获取通行证</button></div></div>':''}
function rDB(){const g={};as.forEach(s=>{if(!g[s.date])g[s.date]=[];g[s.date].push(s)});const ds=Object.keys(g).sort(),b=el('dbar');b.innerHTML='';if(!ds.length){el('sc').innerHTML=calendarErrHTML()+'<div class="empty"><div class="mascot-wrap">'+mascotSVG('sleep',56)+'</div>这几家门店当前没有放出可展示时段，晚点再来看看？也可以刷新或换一家门店。</div>';return}if(!sd||!ds.includes(sd))sd=ds[0];ds.forEach(d=>{const sl=g[d],av=sl.filter(s=>s.availability==='AVAILABLE').length,dt=new Date(d.substring(0,4)+'-'+d.substring(4,6)+'-'+d.substring(6,8)),c=document.createElement('div');c.className='dc'+(d===sd?' on':'');c.innerHTML='<div class="dw">周'+W[dt.getDay()]+'</div><div class="dd">'+fD(d)+'</div><div class="dv '+(av>0?'h':'n')+'">'+(av>0?'可约 '+av:'已满')+'</div>';c.onclick=()=>{sd=d;rDB()};b.appendChild(c)});rS(sd)}
function rS(d){const sl=as.filter(s=>s.date===d).sort((a,b)=>(a.store_name||'').localeCompare(b.store_name||'')||(a.start||'').localeCompare(b.start||'')),c=el('sc');if(!sl.length){c.innerHTML=calendarErrHTML()+'<div class="empty">无时段</div>';return}const ac=sl.filter(s=>s.availability==='AVAILABLE').length;c.innerHTML=calendarErrHTML()+'<div class="sg">'+sl.map(s=>{const a=s.availability==='AVAILABLE',m=slotMatchesPrefs(s);return'<div class="sl '+(a?'av':'fu')+'"><div class="tm">'+esc(fT(s.start))+'-'+esc(fT(s.end))+'</div><div class="ss">'+(a?'可预约':'已满')+' · '+esc(s.store_name||s.store_id||'')+(a&&m?' · 符合偏好':'')+'</div><div class="mt8">'+(a?'<button class="bt bt-r bt-s" onclick="bookSlotDirect(\''+escA(String(s.store_id||''))+'\',\''+escA(s.date)+'\',\''+escA(s.start)+'\',\''+escA(s.end||'')+'\',\''+escA(String(s.store_name||s.store_id||''))+'\');return false">预约这个时段</button>':'<button class="bt bt-w bt-s" onclick="snFromSlot(\''+escA(String(s.store_id||''))+'\',\''+escA(s.date)+'\',\''+escA(s.start)+'\',\''+escA(s.end||'')+'\');return false">蹲这个时段</button>')+'</div></div>'}).join('')+'</div><p class="mu mt8">'+sl.length+' 个时段 · '+ac+' 个可预约（可直接预约）· 已满时段可加入蹲未来预约 · '+selStores.length+' 家门店</p>'}

async function lI(){await ensureStores();const c=el('ic');c.innerHTML='<div class="skeleton" style="height:46px;border-radius:10px;margin-bottom:8px"></div><div class="skeleton" style="height:200px;border-radius:10px"></div>';try{const d=await safeFetch('/api/insights?top=12');if(d.error){c.innerHTML=loadErrBoxHTML(d.error,'lI()','历史洞察');return}const rec=d.recommendations||[],min=d.min_recommendation_observations||3;const metrics='<div class="metric">'+chip('历史样本',d.valid_snapshots||0,'ok')+chip('推荐门槛','同一时段 '+min+' 次','warn')+chip('推荐数量',rec.length,'ok')+'</div>';const rows=rec.map(r=>'<tr><td data-label="门店">'+esc(storeName(r.store_id))+'<span class="mu debug-only"><br>'+esc(r.store_id)+'</span></td><td data-label="星期">'+esc(r.weekday_name)+'</td><td data-label="时段">'+esc(fT(r.start))+'-'+esc(fT(r.end))+'</td><td data-label="开放概率">'+Math.round((r.availability_rate||0)*100)+'%</td><td data-label="售罄速度">'+(r.sold_out_minutes==null?'-':Math.round(r.sold_out_minutes)+' 分')+'</td><td data-label="样本">'+esc(r.observations)+'</td></tr>').join('');const empty=(d.valid_snapshots||0)?'<div class="empty">样本还不够稳定。保持预测准确度，等同一门店、星期、时段至少积累 '+min+' 次观察后再给推荐。<div class="mt8"><button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">去预测准确度</button></div></div>':'<div class="empty">暂无历史数据。<div class="mt8"><button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">去预测准确度</button></div></div>';c.innerHTML=metrics+(rows?'<table class="tbl tbl-cards"><thead><tr><th>门店</th><th>星期</th><th>时段</th><th>开放概率</th><th>售罄速度</th><th>样本</th></tr></thead><tbody>'+rows+'</tbody></table>':empty)}catch(e){c.innerHTML=loadErrBoxHTML(e,'lI()','历史洞察')}}

async function lQD(){await ensureStores();setQueuePredictionMode('ticket');if(!qdSelected.length){const saved=recallStores('sushiro_qd_store').slice(0,1);if(saved.length)qdSelected=saved;else if(stores.length)qdSelected=[String(stores[0].id)]}renderDashboardStores();applyPlanDir();fillNetTicketStores();loadNetTicketRoutine();await loadCloudAuth(false);await loadSampling();await loadQueueAlerts();await loadQueueAlertStatus();await loadQueueDashboard();runPlanCalc();stopQDAutoRefresh();qdAutoTimer=setInterval(()=>{if(document.hidden)return;loadQueueAdvisorCard()},45000)}
function dashboardParams(){const p=new URLSearchParams();p.set('scope',qdSelected.length?'local':'all');p.set('date_type',dashboardDateType());p.set('window','12');p.set('bucket','10');const target=parseInt(el('qdTargetNo')?.value||'',10);if(target>0)p.set('target_no',String(target));if(qdSelected.length)p.set('stores',qdSelected.slice(0,1).join(','));return p}
function dashboardDateType(){const v=el('qdDateType')?.value||'all';return['all','weekday','weekend','holiday'].includes(v)?v:'all'}
function updateQueuePredictionReadiness(){const target=parseInt(el('qdTargetNo')?.value||'',10)||0;const ready=!!qdSelected.length&&target>0;el('qdAdvisorBlock')?.classList.toggle('hid',!ready)}
function applyDashboardStores(ids){qdSelected=(ids||[]).slice(0,1).map(String);rememberStores('sushiro_qd_store',qdSelected);updateQueuePredictionReadiness();renderDashboardStores();renderReminderTemplateHint();loadQueueDashboard();loadQueueAlertStatus();runPlanCalcDebounced()}
function renderDashboardStores(){updateQueuePredictionReadiness();const c=el('qdStores');if(!c)return;if(!qdSelected.length){const target=parseInt(el('qdTargetNo')?.value||'',10);c.innerHTML='<span class="mu">'+(target>0?'已填写当天排队号：请先选择门店，避免用其他门店曲线误判。':'未指定门店：可先浏览样本最多、最新的门店；填当天排队号前建议选定门店。')+'</span>';renderTicketReminderCard();return}c.innerHTML=qdSelected.map(id=>'<button class="chip on" data-store="'+escA(String(id))+'">'+esc(storeDisplayName(id))+' ✕</button>').join('');c.querySelectorAll('.chip.on').forEach(b=>b.onclick=()=>{const id=b.dataset.store;qdSelected=qdSelected.filter(x=>x!==id);rememberStores('sushiro_qd_store',qdSelected);renderDashboardStores();renderReminderTemplateHint();loadQueueDashboard();loadQueueAlertStatus()})}
function qdReminderStore(){const id=qdSelected[0];if(!id)return null;return{id:String(id),name:storeDisplayName(id)}}
function reminderTemplatePoints(target,tpl){const presets={normal:[80,50,25],conservative:[120,90,60,30],urgent:[50,25,10]},offsets=presets[tpl]||[];return Array.from(new Set(offsets.map(n=>target-n).filter(n=>n>0&&n<=target))).sort((a,b)=>a-b)}
function reminderPointsFromInputs(target){const custom=alertNoList(el('qdrPoints')?.value||'').filter(n=>n<=target);if(custom.length)return custom.sort((a,b)=>a-b);return reminderTemplatePoints(target,el('qdrTemplate')?.value||'normal')}
function renderReminderTemplateHint(){const target=parseInt(el('qdTargetNo')?.value||'',10),input=el('qdrPoints'),tpl=el('qdrTemplate')?.value||'normal';if(input&&!(input.value||'').trim()){const pts=target>0?reminderTemplatePoints(target,tpl):[];input.placeholder=pts.length?'默认 '+pts.join(','):'如 1000,1025,1050'}renderDashboardStores();renderTicketReminderCard()}
function qaRuleThreshold(r){return(r&&r.notify_at_no)||(((r&&r.target_no)||0)-((r&&r.lead_groups)||0))||0}
function qaRuleKey(r){r=r||{};return r.type==='called_reach'?[r.store_id,r.type,r.wait_minutes||0,r.target_no||0,qaRuleThreshold(r)].join('|'):[r.store_id,r.type,r.wait_minutes||0,r.target_no||0].join('|')}
async function loadQueueAlertStatus(){try{qaStatus=await safeFetch('/api/queue/alerts/status');renderTicketReminderCard()}catch(e){renderTicketReminderCard('提醒状态加载失败：'+String(e.message||e))}}
function renderTicketReminderCard(err){
 const box=el('qdReminderStatus');if(!box)return;
 const nb=el('qdrNotifyBtn');if(nb)nb.textContent=nfc?'管理通知':'设置通知';
 if(err){box.innerHTML='<div class="ci bad">'+esc(err)+'</div>';return}
 const s=qdReminderStore(),target=parseInt(el('qdTargetNo')?.value||'',10),points=target>0?reminderPointsFromInputs(target):[],n=qaStatus.notifications||{},sampling=qaStatus.sampling||{},channels=(n.channels||[]).join('、')||'未配置',notifyClass=n.configured?'ok':'bad',sampleClass=sampling.running||sampling.daemon_running||sampling.system_auto_start?.enabled?'ok':'warn',hint=!s?'先在上方选一家门店（提醒只盯这家店的叫号）。':!target?'在上方「当天排队号」填你的号码，提醒会按节奏自动生成。':points.length?('将为 '+esc(s.name)+' · 当天排队号 '+fmtN(target)+'，在叫到 '+points.map(fmtN).join('、')+' 号时各提醒一次。'):'自定义号码无效：提醒号必须小于你的当天排队号。';
 const chips=chip('通知',channels,notifyClass)+chip('采集',sampling.running?'运行中':sampling.daemon_running?'后台运行':sampling.system_auto_start?.enabled?'已设开机采集':(sampling.message||'未持续采集'),sampleClass);
 const rules=(qaStatus.rules||[]).filter(x=>x.rule&&x.rule.type==='called_reach'&&(!s||String(x.rule.store_id)===s.id)&&(!target||x.rule.target_no===target));
 const rows=rules.length?'<div class="sg mt8">'+rules.map(x=>{
  const r=x.rule||{},cls=x.status==='fired'?'av':x.status==='due'?'fu':'av',eta=x.estimate_to_threshold_minutes!=null?(' · 预计 '+fmtN(x.estimate_to_threshold_minutes)+' 分钟到提醒点'):'',next=x.next?' · 下一条':'',key=x.key||qaRuleKey(r);
  return'<div class="sl '+cls+'"><div class="fl ai jb g8"><div class="ss">'+esc(x.label||r.label||((r.target_no||0)+'号'))+' · '+fmtN(r.target_no||0)+'号</div><button class="bt bt-o bt-s" onclick="removeQueueAlertByKey(\''+escA(key)+'\')">删除</button></div><div class="mu mt8">到/过 '+fmtN(x.threshold||qaRuleThreshold(r))+' 号提醒 · 当前 '+fmtN(x.current_called_no||0)+' · 还差 '+fmtN(x.remaining_to_threshold||0)+' 号'+eta+next+'</div><div class="mu mt8">'+esc(x.status_text||'监控中')+(r.travel_minutes?(' · 路程约 '+fmtN(r.travel_minutes)+' 分钟'):'')+' · 命中后自动删除</div></div>'
 }).join('')+'</div>':'';
 box.innerHTML='<div class="fl g8 fw">'+chips+'</div><div class="mu mt8">'+hint+'</div>'+rows
 renderReminderChannels()
}
function reminderSamplingActive(){const s=(qaStatus&&qaStatus.sampling)||{};return !!(s.running||s.daemon_running||s.system_auto_start?.enabled)}
async function ensureTicketReminderSampling(storeID){if(!hc)return '';try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const active=reminderSamplingActive(),id=String(storeID),ids=(spCfg.store_ids||[]).map(String),hasStore=ids.includes(id),nextIDs=Array.from(new Set([id].concat(ids)));if(active&&hasStore)return '';const payload={...spCfg,enabled:true,auto_start:true,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:nextIDs,use_preference_stores:false};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;if(active){await loadSampling();return 'updated'}d=await safeFetch('/api/sampling/start',{method:'POST'});spState=d.state||spState;await loadSampling();return 'started'}catch(e){toast('提醒已保存，但配置本机采集失败：'+String(e.message||e));return ''}}
async function createTicketReminder(){const s=qdReminderStore();if(!s){toast('请先选门店');return}const target=parseInt(el('qdTargetNo')?.value||'',10);if(!target){toast('请填写当天排队号');return}const points=reminderPointsFromInputs(target);if(!points.length){toast('请填写有效提醒点，且不能大于当天排队号');return}const label=(el('qdrLabel')?.value||'').trim(),travel=Math.max(0,parseInt(el('qdrTravel')?.value||'',10)||0),tpl=el('qdrTemplate')?.value||'normal';try{let base=qtAlerts||[];try{const d=await safeFetch('/api/queue/alerts');base=(d&&d.rules)||base}catch(e){}const rules=base.filter(r=>!(String(r.store_id)===s.id&&r.type==='called_reach'&&Number(r.target_no||0)===target));points.forEach(n=>rules.push({store_id:s.id,store_name:s.name,label:label,type:'called_reach',target_no:target,notify_at_no:n,lead_groups:Math.max(0,target-n),travel_minutes:travel,template:tpl,channels:qdReminderChannels.slice(),enabled:true}));const saved=await safeFetch('/api/queue/alerts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({rules:rules})});qtAlerts=(saved&&saved.rules)||rules;const samplingAction=await ensureTicketReminderSampling(s.id);await loadQueueAlertStatus();let msg='已生成 '+points.length+' 个提醒点';if(samplingAction==='started')msg+='，已启动本机采集';if(samplingAction==='updated')msg+='，已加入本机采集门店';if(!reminderSamplingActive())msg+='，需要先获取凭证并开启本机采集才会推送';toast(msg);if(!nfc){const goCfg=await confirmDialog({title:'提醒已生成，但还没配通知渠道',body:'提醒规则已保存，但通知渠道（飞书/Telegram/Bark/Server酱）没配的话，到点叫号不会推送给你——得自己盯着屏幕。现在去配一个？只需填一次。',ok:'去配置通知',cancel:'稍后'});if(goCfg)focusNotifySettings()}}catch(e){toast('生成提醒失败：'+String(e.message||e))}}
function renderDashboardSamplingCard(){
 const box=el('qdSamplingCard');if(!box)return;
 const s=spState||{},cfg=spCfg||{},q=spQueueState||{},running=!!s.running,enabled=!!(s.enabled||cfg.enabled);
 const cloudReady=!!(cloudAuth.baseline_connected||(qdDashboardData.baseline&&qdDashboardData.baseline.used));
 const cloudLoggedIn=!!cloudAuth.connected;
 const localNeedsAuth=!hc||q.needs_auth||q.auth_ok===false;
 const last=s.last_run_at?new Date(s.last_run_at).toLocaleString():'还没有',next=s.next_run_at?new Date(s.next_run_at).toLocaleString():'-',ids=(cfg.store_ids||[]).map(String),storeText=ids.length?ids.map(storeDisplayName).join('、'):'偏好门店';
 const msg=s.last_error||s.message||q.message||'开启后会在本机记录叫号、等位和可预约时段。';
 const toggle='<label class="switch"><input type="checkbox" '+(running?'checked':'')+' onchange="toggleDashboardSampling(this.checked)"> 本机持续采集</label>';
 const adv=currentUIMode()==='advanced';
 // GitHub/线上基准是进阶版功能：简化版不暴露登录入口，只讲本机采集。
 const cloudButton=cloudReady||cloudLoggedIn?'<button class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新图表</button>':'<button class="bt bt-w bt-s" onclick="startCloudLogin()">登录 GitHub 获取线上基准</button>';
 const localActions=toggle+'<button class="bt bt-w bt-s" onclick="runDashboardSampleOnce()">收集一次</button>'+(running?'<button class="bt bt-o bt-s" onclick="stopSampling()">暂停</button>':'')+(adv?'<button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">详细配置</button>':'');
 const actions=adv&&localNeedsAuth?cloudButton+'<button class="bt bt-o bt-s" onclick="startAuth()">小程序采集补强</button>':(localNeedsAuth?'<button class="bt bt-o bt-s" onclick="startAuth()">小程序采集补强</button>':localActions);
 const intro=adv?('图表走 GitHub + 线上数据库；小程序通行证只用于本机采集补强。'):'开启本机采集后，叫号曲线会越来越准。';
 const chartChip=adv?chip('图表',cloudReady?'线上基准可用':cloudLoggedIn?'GitHub 已登录，基准待验证':'登录 GitHub 获取线上基准',cloudReady?'ok':'warn'):'';
 box.innerHTML='<div><p style="margin-top:0">'+intro+'只记录 '+esc(storeText)+' 的叫号、等位和可预约时段；本机采集数据只留在本机，不上传。</p><div class="sample-state">'+chartChip+chip('本机采集',running?'运行中':enabled?'已启用':'未启动',running?'ok':enabled?'warn':'')+chip('小程序通行证',localNeedsAuth?'采集需更新':'采集可用',localNeedsAuth?'warn':'ok')+chip('样本',s.queue_snapshots||s.snapshots||0,'ok')+chip('上次',last,'ok')+chip('下次',next,'ok')+chip('最近结果',msg,s.last_error?'warn':'ok')+'</div></div><div class="curve-sampling-actions">'+actions+'</div>'
}
async function toggleDashboardSampling(on){if(on&&!hc){toast('本机持续采集需要先获取通行证');renderDashboardSamplingCard();startAuth();return}try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const ids=qdSelected.length?qdSelected.slice(0,1):(spCfg.store_ids||[]);const payload={...spCfg,enabled:!!on,auto_start:on?true:!!spCfg.auto_start,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:ids,use_preference_stores:ids.length===0};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;if(on){d=await safeFetch('/api/sampling/start',{method:'POST'});spState=d.state||spState;toast('已启动本机持续采集')}else{d=await safeFetch('/api/sampling/stop',{method:'POST'});spState=d.state||spState;toast('已暂停本机持续采集')}await loadSampling();renderDashboardSamplingCard()}catch(e){toast('采集开关失败：'+String(e.message||e));await loadSampling();renderDashboardSamplingCard()}}
async function runDashboardSampleOnce(){if(!hc){toast('本机采集需要先获取通行证');startAuth();return}try{if(!spCfg||!Object.keys(spCfg).length)await loadSampling();const ids=qdSelected.length?qdSelected.slice(0,1):(spCfg.store_ids||[]);const payload={...spCfg,enabled:true,interval_seconds:spCfg.interval_seconds||300,active_start:spCfg.active_start||'100000',active_end:spCfg.active_end||'220000',store_ids:ids,use_preference_stores:ids.length===0};let d=await safeFetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});spCfg=d.config||payload;spState=d.state||spState;d=await safeFetch('/api/sampling/once',{method:'POST'});spState=d.state||spState;const r=d.result||{};toast(r.skipped?'本轮跳过：'+(r.skip_reason||'未知原因'):'收集完成：'+(r.queue_snapshots||0)+' 条排队快照，'+(r.snapshots||0)+' 条时段');await loadSampling();renderDashboardSamplingCard()}catch(e){toast('收集失败：'+String(e.message||e));await loadSampling();renderDashboardSamplingCard()}}
async function loadQueueDashboard(){const adv=el('qdAdvisor');if(!adv)return;const token=++qdDashToken;qdRefreshToken++;adv.innerHTML='<div class="ci">正在生成到店建议…</div>';try{const d=await safeFetch('/api/queue/dashboard?'+dashboardParams().toString(),null,20000);if(token!==qdDashToken)return;qdDashboardData=d||{};renderQueueDashboard(d);renderDashboardSamplingCard()}catch(e){if(token!==qdDashToken)return;qdDashboardData={};adv.innerHTML=loadErrBoxHTML(e,'loadQueueDashboard()','到店建议');renderDashboardSamplingCard()}if(token===qdDashToken)loadQueueAdvisorCard()}

// ---------- 排队压力答案卡 + 压力主图（我有号码页顶部） ----------
function pressureClass(level){return 'press-'+(level||'unknown')}
async function loadQueueAdvisorCard(){updateQueuePredictionReadiness();const ans=el('qdAnswer'),pc=el('qdPressChart');if(!ans)return;const store=qdSelected[0]||'';if(!store){ans.innerHTML='<div class="ci">选一家门店，并填上你的当天排队号，这里直接给你「几点能吃上、几点出发」。</div>';if(pc)renderPressureChart(pc,{points:[],message:'选门店后，这里把今天的叫号进度、排队压力和你的当天排队号画在同一张图上（未选门店时仍展示全国线上历史排队趋势）。'},null,0);return}const target=parseInt(el('qdTargetNo')?.value||'',10)||0,travel=Math.max(0,parseInt(el('qdrTravel')?.value||'',10)||0);const token=++qdRefreshToken;ans.innerHTML='<div class="ci">正在读取实时排队压力…</div>';let adv=null;try{const qs='store='+encodeURIComponent(store)+(target>0?'&target_no='+target:'')+(travel>0?'&travel_minutes='+travel:'');adv=await safeFetch('/api/queue/advisor?'+qs,null,15000);if(token!==qdRefreshToken)return;renderQueueAnswer(adv,target)}catch(e){if(token!==qdRefreshToken)return;ans.innerHTML=loadErrBoxHTML(e,'loadQueueAdvisorCard()','排队压力')}
 if(pc){try{const curve=await safeFetch('/api/queue/pressure/curve?store='+encodeURIComponent(store),null,20000);if(token!==qdRefreshToken)return;renderPressureChart(pc,curve,adv,target)}catch(e){if(token!==qdRefreshToken)return;pc.innerHTML=loadErrBoxHTML(e,'loadQueueAdvisorCard()','整合走势')}}}
function renderQueueAnswer(adv,target){const ans=el('qdAnswer');if(!ans)return;const cur=adv.current||{},p=adv.pressure||{},sp=adv.speed||{},eta=adv.eta||null,nfcOk=nfc;let lead='';if(eta&&eta.remaining_groups>0&&eta.wait_minutes_range){const wr=eta.wait_minutes_range,called=fmtN(cur.called_no||0),tip=eta.estimated_called_at_range?(shortTime(eta.estimated_called_at_range.early)+'-'+shortTime(eta.estimated_called_at_range.late)):shortTime(eta.estimated_called_at);lead='你的当天排队号是 '+fmtN(target)+'，当前叫到 '+called+'，预计 '+wr.low+'-'+wr.high+' 分钟后轮到（约 '+tip+' 能吃上）。'+(eta.arrival_suggestion||'')}else if(eta&&eta.remaining_groups<=0){lead='你的当天排队号是 '+fmtN(target)+'，已经轮到或即将轮到，请尽快到店。'}else if(eta){lead=eta.arrival_suggestion||'实时和历史数据都不足，暂时无法预估能吃上的时间。'}else if(target>0){lead='当前叫到 '+fmtN(cur.called_no||0)+' 号，正在估算到你的时间…'}else{lead='当前叫到 '+fmtN(cur.called_no||0)+' 号，排队压力'+(p.label||'数据不足')+'。填上你的当天排队号，给你「几点能吃上、几点出发」。'}const s15=sp.called_per_min_15!=null?(Math.round(sp.called_per_min_15*15)+' 桌'):'数据不足';const chips=[];chips.push(answerChip('当前叫到',fmtN(cur.called_no||0)||'-',''));if(eta&&eta.remaining_groups>0)chips.push(answerChip('还差',fmtN(eta.remaining_groups)+' 号',''));chips.push(answerChip('排队压力',p.label||'数据不足',pressureClass(p.level)));chips.push(answerChip('消化趋势',p.trend_label||'数据不足',''));chips.push(answerChip('近15分钟叫号',s15,''));if(eta&&eta.source_label)chips.push(answerChip('估算依据',eta.source_label,eta.source==='official'?'press-extreme':''));if(eta&&eta.estimated_called_at_range)chips.push(answerChip('预计能吃',shortTime(eta.estimated_called_at_range.early)+'-'+shortTime(eta.estimated_called_at_range.late),''));chips.push(answerChip('通知',nfcOk?'已配置':'未配置',nfcOk?'':'press-extreme'));const reason=p.reason?'<div class="mu mt8">'+esc(p.reason)+'</div>':'',sourceNote=(eta&&eta.source_note)?'<div class="mu mt8">'+esc(eta.source_note)+'</div>':'',accNote=(eta&&eta.accuracy_note)?'<div class="mu mt8" style="color:#21823F">📈 '+esc(eta.accuracy_note)+'</div>':'',warns=(adv.warnings||[]).length?'<div class="mu mt8" style="color:#c4561a">⚠ '+(adv.warnings||[]).map(esc).join('；')+'</div>':'';ans.innerHTML='<div class="answer-lead">'+esc(lead)+'</div><div class="answer-chips">'+chips.join('')+'</div>'+reason+sourceNote+accNote+warns}
function answerChip(label,value,cls){return '<div class="answer-chip"><span>'+esc(label)+'</span><strong class="'+(cls||'')+'">'+esc(String(value))+'</strong></div>'}
function hhmmMinute(t){const m=String(t||'').match(/^(\d{1,2}):(\d{2})/);return m?parseInt(m[1],10)*60+parseInt(m[2],10):null}
function historicalCalledPoints(d){return ((d&&d.called_curve)||[]).filter(p=>hhmmMinute(p.bucket)!=null&&(p.called_no_typical||0)>0).slice().sort((a,b)=>hhmmMinute(a.bucket)-hhmmMinute(b.bucket))}
function historicalQueueTrendPoints(d){return ((d&&d.trend)||[]).map(p=>({p:p,m:hhmmMinute(p.label)!=null?hhmmMinute(p.label):hhmmMinute(p.bucket)})).filter(o=>o.m!=null&&o.m>=600&&o.m<=1320&&(o.p.total_queue_groups||0)>0).sort((a,b)=>a.m-b.m).map(o=>Object.assign({},o.p,{_m:o.m}))}
function calledCurveSourceLabel(source){return source==='remote_baseline'?'线上数据库基准':source==='local'?'本机历史采样':'历史基准'}
function queueTrendSourceLabel(source){return source==='remote_baseline'?'线上数据库':source==='local'?'本机采样':'历史'}
function renderPressureChart(box,curve,adv,target){
 if(!box)return;
 const minM=600,maxM=1320;
 const points=(curve&&curve.points||[]).filter(p=>{const m=hhmmMinute(p.time);return m!=null&&m>=minM&&m<=maxM}).slice().sort((a,b)=>hhmmMinute(a.time)-hhmmMinute(b.time));
 const hist=historicalCalledPoints(qdDashboardData);
 const trend=historicalQueueTrendPoints(qdDashboardData);
 if(!points.length&&!hist.length&&!trend.length){box.innerHTML='<div class="empty">'+esc((curve&&curve.message)||'还没有今天的本机采样曲线。开启「本机持续采集」后会逐步补齐；现在可看上面的实时答案卡。')+'</div>';return}
 const trendMax=Math.max(1,...trend.map(t=>t.total_queue_groups||0));
 const calledMax=Math.max(10,...points.map(p=>p.called_no||0),...hist.flatMap(p=>[p.called_no_slow||0,p.called_no_typical||0,p.called_no_fast||0]),target>0?target:0);
 const w=1040,h=286,l=52,r=52,t=28,b=40,maxCalled=calledMax,x=m=>l+((m-minM))/(maxM-minM)*(w-l-r),yCall=v=>h-b-(v/maxCalled)*(h-t-b),yPress=s=>h-b-(Math.min(100,Math.max(0,s))/100)*(h-t-b);
 let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="xMidYMid meet" style="width:100%;aspect-ratio:'+w+'/'+h+'">';
 svg+='<text class="chart-axis-title" x="'+(l-6)+'" y="'+(t-8)+'" text-anchor="end">叫号</text><text class="chart-axis-title" x="'+(w-r+6)+'" y="'+(t-8)+'" text-anchor="start">压力</text>';
 for(let i=0;i<=4;i++){const yy=t+i*(h-t-b)/4,val=Math.round(maxCalled*(4-i)/4);svg+='<line class="chart-grid" x1="'+l+'" y1="'+yy+'" x2="'+(w-r)+'" y2="'+yy+'"></line><text class="chart-label" x="'+(l-8)+'" y="'+(yy+4)+'" text-anchor="end">'+fmtN(val)+(i===0?' 号':'')+'</text><text class="chart-label" x="'+(w-r+8)+'" y="'+(yy+4)+'" text-anchor="start" fill="#888">'+(100-25*i)+'</text>'}
 for(let hh=10;hh<=22;hh+=2){const xx=x(hh*60);svg+='<line class="chart-grid" x1="'+xx+'" y1="'+t+'" x2="'+xx+'" y2="'+(h-b)+'" opacity=".55"></line><text class="chart-label" x="'+xx+'" y="'+(h-9)+'" text-anchor="middle">'+(hh<10?'0':'')+hh+':00</text>'}
 svg+='<line class="chart-axis" x1="'+l+'" y1="'+(h-b)+'" x2="'+(w-r)+'" y2="'+(h-b)+'"></line>';
 const pressArea=points.map(p=>x(hhmmMinute(p.time))+','+yPress(p.pressure_score||0));
 if(pressArea.length){const base=(h-b);svg+='<polygon points="'+l+','+base+' '+pressArea.join(' ')+' '+(w-r)+','+base+'" fill="rgba(120,120,152,.18)" stroke="rgba(120,120,152,.5)" stroke-width="1"></polygon>'}
 const histPts=hist.map(p=>x(hhmmMinute(p.bucket))+','+yCall(p.called_no_typical||0));
 if(histPts.length>1)svg+='<polyline points="'+histPts.join(' ')+'" fill="none" stroke="var(--blue)" stroke-width="2.4" stroke-dasharray="7 5" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>';
 const yTrend=g=>h-b-(Math.min(trendMax,Math.max(0,g))/trendMax)*(h-t-b);
 const trendPts=trend.map(t=>x(t._m)+','+yTrend(t.total_queue_groups||0));
 if(trendPts.length>1)svg+='<polyline points="'+trendPts.join(' ')+'" fill="none" stroke="var(--green)" stroke-width="2.4" stroke-linejoin="round" stroke-linecap="round" stroke-dasharray="2 4" vector-effect="non-scaling-stroke"></polyline>';
 const callPts=points.filter(p=>(p.called_no||0)>0);
 let stepPath='';
 callPts.forEach((p,i)=>{const cx=x(hhmmMinute(p.time)),cy=yCall(p.called_no);stepPath+=(i===0?'M':'L')+cx+','+cy+' ';if(i<callPts.length-1){const nx=x(hhmmMinute(callPts[i+1].time));stepPath+='L'+nx+','+cy+' '}});
 if(stepPath)svg+='<path d="'+stepPath+'" fill="none" stroke="var(--red)" stroke-width="3" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></path>';
 else if(callPts.length===1){const cx=x(hhmmMinute(callPts[0].time)),cy=yCall(callPts[0].called_no);svg+='<circle cx="'+cx+'" cy="'+cy+'" r="5" fill="#B81C22" stroke="#fff" stroke-width="2"></circle>'}
 const nowMin=(()=>{const dd=new Date();return dd.getHours()*60+dd.getMinutes()})();
 if(nowMin>=minM&&nowMin<=maxM){const nx=x(nowMin);svg+='<line x1="'+nx+'" y1="'+t+'" x2="'+nx+'" y2="'+(h-b)+'" stroke="var(--red)" stroke-width="1.4" opacity=".8"></line><text class="chart-label" x="'+(nx+4)+'" y="'+(t+10)+'" fill="var(--red)">现在</text>'}
 else{svg+='<text class="chart-label" x="'+(l+6)+'" y="'+(t+12)+'" fill="#aaa">（非营业时间 10:00-22:00，图不更新）</text>'}
 if(target>0){const my=target<=maxCalled?yCall(target):t;svg+='<line x1="'+l+'" y1="'+my+'" x2="'+(w-r)+'" y2="'+my+'" stroke="var(--red)" stroke-width="1.4" stroke-dasharray="4 4" opacity=".9"></line><text class="chart-label" x="'+(w-r-4)+'" y="'+(my-4)+'" text-anchor="end" fill="var(--red)">'+(target>maxCalled?'我的当天排队号 '+fmtN(target)+'（较靠后）':'我的当天排队号 '+fmtN(target))+'</text>'}
 // 预测就餐区间带：有 ETA 时间区间时，在 x 轴对应时段画半透明绿带 + 顶部标注「预计几点能吃上」。
 const er=(adv&&adv.eta&&adv.eta.estimated_called_at_range)?adv.eta.estimated_called_at_range:null;
 if(er&&target>0){const eM=hhmmMinute(shortTime(er.early)),lM=hhmmMinute(shortTime(er.late));if(eM!=null&&lM!=null&&lM>=minM&&eM<=maxM){const xe=x(Math.max(minM,eM)),xl=x(Math.min(maxM,lM));if(xl>xe){svg+='<rect x="'+xe+'" y="'+t+'" width="'+(xl-xe)+'" height="'+(h-b-t)+'" fill="rgba(33,130,63,.14)"></rect>';svg+='<line x1="'+xe+'" y1="'+t+'" x2="'+xe+'" y2="'+(h-b)+'" stroke="var(--green)" stroke-width="1.2" stroke-dasharray="3 3" opacity=".7"></line><line x1="'+xl+'" y1="'+t+'" x2="'+xl+'" y2="'+(h-b)+'" stroke="var(--green)" stroke-width="1.2" stroke-dasharray="3 3" opacity=".7"></line>';const lab='预计 '+shortTime(er.early)+'-'+shortTime(er.late)+' 能吃上'+(adv.eta.remaining_groups>0?('（还差 '+fmtN(adv.eta.remaining_groups)+' 号）'):'');svg+='<text class="chart-label" x="'+((xe+xl)/2)+'" y="'+(t+12)+'" text-anchor="middle" fill="var(--green)" font-weight="900">'+esc(lab)+'</text>'}}}
 const etaTip=(adv&&adv.eta&&adv.eta.estimated_called_at_range)?('\n预计能吃上：'+shortTime(adv.eta.estimated_called_at_range.early)+'-'+shortTime(adv.eta.estimated_called_at_range.late)):'';
 hist.forEach(p=>{const cx=x(hhmmMinute(p.bucket)),cy=yCall(p.called_no_typical||0),tip=p.bucket+'\n历史典型叫到：'+fmtN(p.called_no_typical||0)+'\n保守/偏快：'+fmtN(p.called_no_slow||0)+' / '+fmtN(p.called_no_fast||0)+'\n样本：'+fmtN(p.sample_count||0)+' · '+fmtN(p.day_count||0)+' 天\n来源：'+calledCurveSourceLabel(p.source)+(p.confidence?'\n置信度：'+p.confidence:'');svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="3" fill="#fff" stroke="var(--blue)" stroke-width="1.8"></circle></g>'});
 if(trendPts.length>1)trend.forEach(p=>{const cx=x(p._m),cy=yTrend(p.total_queue_groups||0),tip=(p.label||p.bucket)+'\n历史排队桌数：'+fmtN(p.total_queue_groups||0)+'\n历史等待：'+fmtN(p.total_wait_minutes||0)+' 分\n样本数：'+fmtN(p.sample_count||0)+'\n来源：'+queueTrendSourceLabel(p.source);svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="2.6" fill="#fff" stroke="var(--green)" stroke-width="1.6"></circle></g>'});
 callPts.forEach((p,i)=>{const cx=x(hhmmMinute(p.time)),cy=yCall(p.called_no),s15=p.called_speed_15!=null?(Math.round(p.called_speed_15*15)+' 桌'):'数据不足',tip=p.time+'\n当前叫到：'+fmtN(p.called_no)+'\n排队压力：'+pressureLabelCN(p.pressure_level)+'\n等待桌数：'+fmtN(p.waiting_groups||0)+'\n官方等待：'+fmtN(p.official_wait_minutes||0)+' 分\n近15分钟叫号：'+s15+'\n来源：'+pressureSourceLabel(p.source)+(p.confidence?'\n置信度：'+p.confidence:'')+(i===callPts.length-1?etaTip:'');svg+='<g class="chart-hot" data-tip="'+escA(tip)+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="'+(i===callPts.length-1?5:3.5)+'" fill="'+(i===callPts.length-1?'#B81C22':'#fff')+'" stroke="#B81C22" stroke-width="2"></circle></g>'});
 svg+='</svg>';
 const notes=[];if(curve&&curve.message)notes.push(curve.message);if(hist.length&&qdDashboardData.called_summary&&qdDashboardData.called_summary.message)notes.push('历史推算线：'+qdDashboardData.called_summary.message);
 if(trendPts.length>1)notes.push('历史排队趋势：绿色虚线是 '+queueTrendSourceLabel(trend[0].source)+'的 '+(trend.length)+' 个时间窗的排队桌数走势（与叫号轴量纲不同，仅看高低）。');
 const note=notes.length?'<div class="mu mt8">'+esc(notes.join(' '))+'</div>':'';
 box.innerHTML=svg+'<div class="chart-legend"><span class="legend-line">今日叫号</span><span class="legend-history">历史叫号</span>'+(trendPts.length>1?'<span class="legend-turso-trend">排队桌数趋势</span>':'')+'<span class="legend-pressure">排队压力</span><span class="legend-now">现在</span><span class="legend-mine">我的号</span><span class="mu">10:00-22:00</span></div>'+note
}
function pressureSourceLabel(source){return {local:'本机采样',remote_latest:'线上最新',remote_baseline:'线上基准'}[source]||'未知'}
function pressureLabelCN(level){return {low:'低',medium:'中',high:'高',extreme:'极高'}[level]||'数据不足'}
function riskLabelCN(r){return {low:'风险低',medium:'风险中',high:'风险高'}[r]||'风险未知'}
function riskClass(r){return {low:'press-low',medium:'press-medium',high:'press-extreme'}[r]||'press-unknown'}
// ---------- 取号→几点吃 ----------
// 时间换算方向：pickup=几点取号→几点吃；meal=想几点吃→几点取号。用 localStorage 记忆，避免依赖被移除的 select。
function planDir(){try{return localStorage.getItem('sushiro_plan_dir')==='meal'?'meal':'pickup'}catch(e){return 'pickup'}}
function setQueuePredictionMode(mode){mode=mode==='pickup'?'pickup':'ticket';const now=el('qdNowTicketCard'),ticket=el('qdExistingTicketCard');if(now)now.classList.toggle('hid',mode!=='pickup');if(ticket)ticket.classList.toggle('hid',mode!=='ticket');el('qdModeTicket')?.classList.toggle('on',mode==='ticket');el('qdModePickup')?.classList.toggle('on',mode==='pickup');if(mode==='pickup'){setPlanDir('pickup');applyPlanDir()}}
function setPlanDir(d){try{localStorage.setItem('sushiro_plan_dir',d==='meal'?'meal':'pickup')}catch(e){}}
function currentTimeInputValue(){const d=new Date(),hh=String(d.getHours()).padStart(2,'0'),mm=String(d.getMinutes()).padStart(2,'0');return hh+':'+mm}
function setPickupToNow(force){const p=el('qpPickup');if(p&&(force||!p.value))p.value=currentTimeInputValue()}
function useNowForPickupPlan(){setPlanDir('pickup');applyPlanDir();setPickupToNow(true);runPlanCalcDebounced()}
function applyPlanDir(){const d=planDir();el('qpPickupWrap').classList.toggle('hid',d!=='pickup');el('qwMealWrap').classList.toggle('hid',d!=='meal');el('qwTravelWrap').classList.toggle('hid',d!=='meal');if(d==='pickup')setPickupToNow(false);const t=el('planTitle');const s=el('planSub');if(t)t.textContent=d==='meal'?'想几点吃，倒推几点取号':'现在取号，几点能吃上';if(s)s.textContent=d==='meal'?'填想吃的时间，算出建议取号时间（倒推，结果仅供参考）':'默认按当前时间估算，适合还没拿号、想判断现在去不去。'}
function swapPlanDir(){setPlanDir(planDir()==='meal'?'pickup':'meal');applyPlanDir();runPlanCalcDebounced()}
let _planCalcTimer=null
function runPlanCalcDebounced(){clearTimeout(_planCalcTimer);_planCalcTimer=setTimeout(runPlanCalc,300)}
function onPlanDirChange(){applyPlanDir();runPlanCalcDebounced()}
function runPlanCalc(){planDir()==='meal'?loadQueueMealPlan():loadQueuePickupPlan()}
async function loadQueuePickupPlan(){const ans=el('qpAnswer');if(!ans)return;const store=qdSelected[0];if(!store){ans.innerHTML='<div class="ci">先在上方选一家门店。</div>';return}setPickupToNow(false);const pickup=(el('qpPickup')?.value||'').replace(':','');ans.innerHTML='<div class="ci">正在估算…</div>';try{const d=await safeFetch('/api/queue/plan?store='+encodeURIComponent(store)+'&pickup='+encodeURIComponent(pickup),null,15000);renderPickupPlan(d)}catch(e){ans.innerHTML=loadErrBoxHTML(e,'loadQueuePickupPlan()','取号规划')}}
function renderPickupPlan(d){const ans=el('qpAnswer');if(!ans)return;if(d.message&&!d.meal_range){ans.innerHTML='<div class="answer-lead">'+esc(d.message)+'</div>';return}const wr=d.wait_minutes_range||{},mr=d.meal_range||{},lead='如果 '+esc(d.pickup)+' 取号，预计 '+esc(mr.early||'?')+'-'+esc(mr.late||'?')+' 吃上（等待约 '+(wr.low||0)+'-'+(wr.high||0)+' 分钟）。';const chips=[answerChip('推荐就餐',esc((mr.early||'?')+'-'+(mr.late||'?')),''),answerChip('预计等待',(wr.low||0)+'-'+(wr.high||0)+' 分',''),answerChip('风险',riskLabelCN(d.risk),riskClass(d.risk))].join('');ans.innerHTML='<div class="answer-lead">'+lead+'</div><div class="answer-chips">'+chips+'</div>'+(d.basis?'<details class="plan-basis mt8"><summary>为什么这么算</summary><div class="mu mt8">'+esc(d.basis)+'</div></details>':'')}
// ---------- 想几点吃→几点取号 ----------
async function loadQueueMealPlan(){const ans=el('qpAnswer');if(!ans)return;const store=qdSelected[0];if(!store){ans.innerHTML='<div class="ci">先在上方选一家门店。</div>';return}const meal=(el('qwMeal')?.value||'').replace(':',''),travel=Math.max(0,parseInt(el('qwTravel')?.value||'',10)||0);ans.innerHTML='<div class="ci">正在倒推…</div>';try{const d=await safeFetch('/api/queue/plan?store='+encodeURIComponent(store)+'&target_meal='+encodeURIComponent(meal)+(travel>0?'&travel_minutes='+travel:''),null,15000);renderMealPlan(d)}catch(e){ans.innerHTML=loadErrBoxHTML(e,'loadQueueMealPlan()','取号倒推')}}
function renderMealPlan(d){const ans=el('qpAnswer');if(!ans)return;if(d.message&&!d.recommend_pickup_range){ans.innerHTML='<div class="answer-lead">'+esc(d.message)+'</div>';return}const rp=d.recommend_pickup_range||{},wr=d.wait_minutes_range||{},lead='想 '+esc(d.target_meal)+' 吃，建议 '+esc(rp.early||d.stable_pickup||'?')+'-'+esc(rp.late||d.stable_pickup||'?')+' 取号。'+(d.latest_pickup?(' 最晚别拖过 '+esc(d.latest_pickup)+'。'):'');const chips=[answerChip('建议取号',esc((rp.early||'?')+'-'+(rp.late||'?')),''),answerChip('偏稳取号',esc(d.stable_pickup||'-'),''),answerChip('最晚取号',esc(d.latest_pickup||'-'),''),answerChip('预计等待',(wr.low||0)+'-'+(wr.high||0)+' 分',''),answerChip('风险',riskLabelCN(d.risk),riskClass(d.risk))].join('');ans.innerHTML='<div class="answer-lead">'+esc(lead)+'</div><div class="answer-chips">'+chips+'</div>'+(d.basis?'<details class="plan-basis mt8"><summary>为什么这么算</summary><div class="mu mt8">'+esc(d.basis)+'</div></details>':'')+'<div class="mu mt8">⚠ 倒推按历史等待估的；取号后前面可能被插队，实际等待可能 ±15 分钟，别把建议取号时间当死线。</div>'}
function renderQueueDashboard(d){renderDashboardAdvisor(d.advisor||{});renderDashboardInsights(d);renderDashboardDataSource(d)}
function dashboardBaselineStatusHTML(d){const b=(d&&d.baseline)||{};const configured=!!b.configured,authenticated=!!b.authenticated,used=!!b.used,rollupCount=Number(b.rollup_count||0),latestCount=Number(b.latest_count||0),rollup=fmtN(rollupCount),latest=fmtN(latestCount);let title,lines=[],cls='ok';if(used){title='本次图表已使用线上数据库基准';lines.push('来源：线上数据库基准');if(rollupCount||latestCount)lines.push('聚合样本 '+rollup+' 条，最新明细 '+latest+' 条');else lines.push('基准已响应，暂无样本');cls='ok'}else if(authenticated){title='仍在用本机数据，线上数据库基准未参与本次图表';lines.push('GitHub 已登录，但线上数据库还没验证成功，需在设置页确认。');cls='warn'}else if(configured){title='本次图表用本机数据，GitHub 尚未登录';lines.push('云端服务已配置；登录 GitHub 后可验证线上基准并叠加参考。');cls='warn'}else{title='本次图表用本机数据，未配置线上基准';lines.push('可在「设置」登录 GitHub 并验证线上基准后，叠加全国线上参考。');cls='warn'}const ws=(d&&d.warnings)||[];if(ws.length){cls=cls==='ok'?'warn':cls;lines.push('注意：'+ws.join('；'));if(ws.some(w=>/明细|基准|曲线/.test(w)))lines.push('这能解释为什么基准可用、但叫号曲线仍没明细。')}return{cls:cls,html:'<b>📊 图表数据来源</b><p>'+esc(title)+'</p><div class="data-source-lines">'+lines.map(l=>'<span>'+esc(l)+'</span>').join('')+'</div>'}}
function renderDashboardDataSource(d){const box=el('qdDataSource');if(!box)return;if(currentUIMode()!=='advanced'){box.className='data-source mt16 hid';box.innerHTML='';return}const s=dashboardBaselineStatusHTML(d);box.className='data-source mt16 '+(s.cls||'');box.innerHTML=s.html||''}
function renderDashboardInsights(d){const heat=el('qdHeatmap'),wk=el('qdWeekday'),tr=el('qdTrend'),cc=el('qdCalledCurve');if(heat)heat.innerHTML=renderHeatmapHTML(d.heatmap||[]);
if(wk)wk.innerHTML=renderWeekdayHTML(d.weekday_profiles||[]);
if(tr)tr.innerHTML=renderTrendHTML(d.trend||[]);
if(cc)cc.innerHTML=renderCalledCurveHTML(d.called_curve||[])}
function fmtFloat(v,dft){return(v==null||isNaN(v))?dft:Math.round(Number(v))}
function busyClass(rate){if(rate==null)return'';const r=Number(rate);if(r>=0.65)return'hot';if(r>=0.35)return'warm';if(r>=0.15)return'mild';return''}
function renderHeatmapHTML(pts){if(!pts.length)return emptyTrendHTML('这家店还没有热力图数据');const days={};pts.forEach(p=>{days[p.weekday]=days[p.weekday]||{name:p.weekday_name,buckets:{}};days[p.weekday].buckets[p.bucket]=p});const wkOrder=[1,2,3,4,5,6,0];const buckets=[];pts.forEach(p=>{if(!buckets.includes(p.bucket))buckets.push(p.bucket)});buckets.sort();const head='<tr><th>时段</th>'+buckets.map(b=>'<th>'+esc(b)+'</th>').join('')+'</tr>';const rows=wkOrder.filter(w=>days[w]).map(w=>{const dn=days[w];return'<tr><td>'+esc(dn.name)+'</td>'+buckets.map(b=>{const p=dn.buckets[b];if(!p)return'<td></td>';const rate=p.busy_rate;const cls=busyClass(rate);const tip=(p.weekday_name||'')+' '+esc(b)+'\n平均等位 '+fmtFloat(p.wait_minutes_avg,'-')+' 桌数 '+fmtFloat(p.queue_groups_avg,'-')+'\n忙率 '+Math.round((Number(rate)||0)*100)+'% 样本 '+p.sample_count;return'<td title="'+escA(tip)+'"><span class="heat-cell '+cls+'">'+fmtFloat(p.queue_groups_avg,'-')+'</span></td>'}).join('')+'</tr>'}).join('');return'<p class="ph-sub" style="margin:0 0 8px">单元格数字=平均排队桌数，颜色越红代表该时段越忙；悬停看等位/忙率/样本。</p><div class="heat-wrap"><table class="heat"><thead>'+head+'</thead><tbody>'+rows+'</tbody></table></div>'}
function renderWeekdayHTML(profiles){if(!profiles.length)return'';const order=[1,2,3,4,5,6,0];const sorted=order.map(w=>profiles.find(p=>p.weekday===w)).filter(Boolean);return'<p class="ph-sub" style="margin:0 0 8px">工作日画像：平均排队桌数 / 平均等位 / 高峰时段。</p><div class="weekday-strip">'+sorted.map(p=>'<div class="weekday-card"><b>'+esc(p.weekday_name||'')+'</b><span>平均桌数 '+fmtFloat(p.queue_groups_avg,'-')+' · 等位 '+fmtFloat(p.wait_minutes_avg,'-')+' 分'+(p.peak_bucket?'<br>高峰约 '+esc(p.peak_bucket)+'（'+fmtFloat(p.peak_queue_groups,'-')+' 桌）':'')+'<br>样本 '+p.sample_count+' · '+esc(p.confidence||'')+'</span></div>').join('')+'</div>'}
function renderTrendHTML(trend){if(!trend.length)return emptyTrendHTML();const top=trend.slice(-12);const maxG=Math.max(1,...top.map(t=>t.total_queue_groups||0));return'<p class="ph-sub" style="margin:0 0 8px">近段排队趋势条（按采样窗口）。</p><div class="rank-list">'+top.map(t=>{const pct=Math.round((t.total_queue_groups||0)/maxG*100);return'<div class="rank-row"><b>'+esc(t.label||t.bucket)+'</b><span>桌数 '+fmtN(t.total_queue_groups)+' · 等位 '+fmtN(t.total_wait_minutes)+' 分 · 样本 '+t.sample_count+'</span><strong style="font-size:14px">'+fmtN(t.total_queue_groups)+'</strong></div>'}).join('')+'</div>'}
// emptyTrendHTML 在某门店还没有叫号趋势/热力图数据时，按登录/采集状态给出差异化引导：
// 趋势数据有两个来源——登录 GitHub 拉线上数据库基准（全国聚合，开箱即用）、或本机采集（更准但需积累）。
// 没数据时主推这两条路，已做的就不再重复推。headOverride 用于热力图等不同标题。
function emptyTrendHTML(headOverride){
	const sp=spState||{},samplingOn=!!(sp.running||sp.enabled||sp.sample_runs>0);
	let head=headOverride||'这家店还没有叫号趋势数据',copy='',btns=[];
	// 简化版不暴露 GitHub/线上基准：统一引导到本机采集。
	if(currentUIMode()!=='advanced'){
		if(samplingOn){
			head='正在采集，叫号趋势会逐步补齐';
			copy='本机采集已在运行，但这家店的样本还不够画出趋势。多用几次、或在店里多待一会儿，数据就会上来。';
		}else{
			copy='开启本机采集后，这家店的叫号趋势会随着使用越来越准。';
			btns.push('<button class="bt bt-r bt-s" onclick="openSettingsFold(\'fold-sm\')">开启本机采集</button>');
		}
		return '<div class="empty"><div class="mascot-wrap"><span class="pm" data-kind="plain" data-size="48"></span></div><b>'+esc(head)+'</b><p class="mt8" style="margin-bottom:12px">'+esc(copy)+'</p><div class="fl g8 fw">'+btns.join('')+'</div></div>';
	}
	const cloudLoggedIn=!!(cloudAuth&&cloudAuth.connected);
	const cloudReady=!!(cloudAuth&&cloudAuth.baseline_connected)||(qdDashboardData.baseline&&qdDashboardData.baseline.used);
	if(cloudReady){
		// 线上基准已参与但仍无该店趋势：多半是这家店线上样本也少，或本机窗内没采到。
		head='线上基准暂无这家店的叫号趋势';
		copy='线上数据库里这家店的样本还较少。开本机采集能补这家店的实时叫号，越用越准。';
		btns.push('<button class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新图表</button>');
		if(!samplingOn)btns.push('<button class="bt bt-r bt-s" onclick="openSettingsFold(\'fold-sm\')">开启本机采集</button>');
	}else if(cloudLoggedIn){
		head='GitHub 已登录，线上基准待验证';
		copy='登录信息已收到，但线上数据库还没验证连通。验证后这家店的叫号趋势会从这里出来；也可同时开本机采集补强。';
		btns.push('<button class="bt bt-w bt-s" onclick="testCloudAuth()">验证连接</button>');
		btns.push('<button class="bt bt-o bt-s" onclick="openSettingsFold(\'fold-sm\')">开启本机采集</button>');
	}else if(samplingOn){
		head='正在采集，叫号趋势会逐步补齐';
		copy='本机采集已在运行，但这家店的样本还不够画出趋势。多用几次、或在店里多待一会儿，数据就会上来。想立刻看到这家店的历史规律，可登录 GitHub 拉线上数据库基准。';
		btns.push('<button class="bt bt-r bt-s" onclick="startCloudLogin()">登录 GitHub 获取线上数据</button>');
	}else{
		copy='叫号趋势有两个来源：登录 GitHub 拉线上数据库（全国聚合，开箱即有）、或开本机采集（更准，需积累）。任选一个就能看到数据。';
		btns.push('<button class="bt bt-r bt-s" onclick="startCloudLogin()">登录 GitHub 获取线上数据</button>');
		btns.push('<button class="bt bt-w bt-s" onclick="openSettingsFold(\'fold-sm\')">开启本机采集</button>');
	}
	return '<div class="empty"><div class="mascot-wrap"><span class="pm" data-kind="plain" data-size="48"></span></div><b>'+esc(head)+'</b><p class="mt8" style="margin-bottom:12px">'+esc(copy)+'</p><div class="fl g8 fw">'+btns.join('')+'</div></div>';
}
function renderCalledCurveHTML(curve){const pts=(curve||[]).filter(p=>hhmmMinute(p.bucket)!=null&&(p.called_no_typical||p.called_no_slow||p.called_no_fast||p.latest_called_no||p.today_projected_no)>0).slice().sort((a,b)=>hhmmMinute(a.bucket)-hhmmMinute(b.bucket));if(pts.length<1)return emptyTrendHTML('这家店还没有叫号曲线');const maxDay=Math.max(0,...pts.map(p=>p.day_count||0));if(maxDay<=1)return renderTodayCalledProgressHTML(pts);const minM=600,maxM=1320;const inRange=pts.filter(p=>{const m=hhmmMinute(p.bucket);return m>=minM&&m<=maxM});const useP=inRange.length>=2?inRange:pts;const maxNo=Math.max(10,...useP.flatMap(p=>[p.called_no_typical||0,p.called_no_slow||0,p.called_no_fast||0,p.today_projected_no||0]));const w=1000,h=260,l=54,r=24,t=24,b=38,x=m=>l+((m-minM))/(maxM-minM)*(w-l-r),y=v=>h-b-(Math.min(maxNo,v)/(maxNo))*(h-t-b);let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="xMidYMid meet" style="width:100%;aspect-ratio:'+w+'/'+h+'">';for(let i=0;i<=4;i++){const yy=t+i*(h-t-b)/4,val=Math.round(maxNo*(4-i)/4);svg+='<line class="chart-grid" x1="'+l+'" y1="'+yy+'" x2="'+(w-r)+'" y2="'+yy+'"></line><text class="chart-label" x="'+(l-8)+'" y="'+(yy+4)+'" text-anchor="end">'+fmtN(val)+(i===0?' 号':'')+'</text>'}for(let hh=10;hh<=22;hh+=2){const xx=x(hh*60);svg+='<line class="chart-grid" x1="'+xx+'" y1="'+t+'" x2="'+xx+'" y2="'+(h-b)+'" opacity=".55"></line><text class="chart-label" x="'+xx+'" y="'+(h-9)+'" text-anchor="middle">'+(hh<10?'0':'')+hh+':00</text>'}svg+='<line class="chart-axis" x1="'+l+'" y1="'+(h-b)+'" x2="'+(w-r)+'" y2="'+(h-b)+'"></line>';const poly=(key,color,dash)=>{const ps=useP.filter(p=>(p[key]||0)>0).map(p=>x(hhmmMinute(p.bucket))+','+y(p[key]));if(ps.length<2)return'';return'<polyline points="'+ps.join(' ')+'" fill="none" stroke="'+color+'" stroke-width="2.4" '+dash+' stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>'};svg+=poly('called_no_slow','var(--blue)','stroke-dasharray="5 4"');svg+=poly('called_no_fast','#E0A800','stroke-dasharray="2 4"');svg+=poly('called_no_typical','var(--green)','');svg+=poly('today_projected_no','#ff8c00','stroke-dasharray="6 4"');useP.forEach(p=>{const cx=x(hhmmMinute(p.bucket)),tipBase=(p.bucket||'')+(p.called_no_typical?'\n历史典型叫到：'+fmtN(p.called_no_typical||0)+' 号\n保守/偏快：'+fmtN(p.called_no_slow||0)+' / '+fmtN(p.called_no_fast||0)+' 号\n样本：'+fmtN(p.sample_count||0)+' · '+fmtN(p.day_count||0)+' 天':'');const tipProj=(p.today_projected_no?'\n推测叫到：约 '+fmtN(p.today_projected_no)+' 号（按今天速度外推）':'');const tip=escA(tipBase+tipProj);const cy=y(p.called_no_typical||p.today_projected_no||0);const isProj=!p.called_no_typical&&p.today_projected_no;svg+='<g class="chart-hot" data-tip="'+tip+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="'+(isProj?'3':'3.5')+'" fill="'+(isProj?'#ff8c00':'#fff')+'" stroke="'+(isProj?'#ff8c00':'var(--green)')+'" stroke-width="1.8"></circle></g>'});svg+='</svg>';return'<p class="ph-sub" style="margin:0 0 8px">历史叫号曲线 + 推测未来：绿线=历史典型叫到几号，蓝虚线=保守（慢），黄虚线=偏快，橙虚线=按今天速度推测接下来叫到几号；悬停看详情。</p><div class="chart">'+svg+'</div>'}
function renderTodayCalledProgressHTML(pts){
 // 历史段（今天已采到的叫号）+ 外推段（按今天速度推测的未来叫号）。
 const hist=pts.filter(p=>(p.latest_called_no||p.called_no_typical||0)>0);
 const proj=pts.filter(p=>(p.today_projected_no||0)>0).sort((a,b)=>hhmmMinute(a.bucket)-hhmmMinute(b.bucket));
 if(hist.length<1&&proj.length<1)return emptyTrendHTML('这家店还没有叫号曲线');
 const allM=[];hist.forEach(p=>allM.push(hhmmMinute(p.bucket)));proj.forEach(p=>allM.push(hhmmMinute(p.bucket)));
 let minM=Math.min(...allM),maxM=Math.max(...allM);
 if(maxM-minM<30)maxM=minM+30;
 const allNo=[];hist.forEach(p=>allNo.push(p.latest_called_no||p.called_no_typical||0));proj.forEach(p=>allNo.push(p.today_projected_no));
 let minNo=Math.min(...allNo),maxNo=Math.max(...allNo);
 const w=1000,h=260,l=54,r=24,t=24,b=38,x=m=>l+((m-minM))/Math.max(1,maxM-minM)*(w-l-r),y=v=>h-b-((Math.min(maxNo,v)-minNo+1)/(maxNo-minNo+1))*(h-t-b);
 let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="xMidYMid meet" style="width:100%;aspect-ratio:'+w+'/'+h+'">';
 for(let i=0;i<=4;i++){const yy=t+i*(h-t-b)/4,val=Math.round(maxNo-(maxNo-minNo)*i/4);svg+='<line class="chart-grid" x1="'+l+'" y1="'+yy+'" x2="'+(w-r)+'" y2="'+yy+'"></line><text class="chart-label" x="'+(l-8)+'" y="'+(yy+4)+'" text-anchor="end">'+fmtN(val)+' 号</text>'}
 const span=Math.max(1,maxM-minM),steps=Math.min(5,Math.max(1,Math.ceil(span/15)));
 for(let i=0;i<=steps;i++){const m=Math.round((minM+span*i/steps)/15)*15,xx=x(m);const hh=Math.floor(m/60),mm=m%60;svg+='<line class="chart-grid" x1="'+xx+'" y1="'+t+'" x2="'+xx+'" y2="'+(h-b)+'" opacity=".55"></line><text class="chart-label" x="'+xx+'" y="'+(h-9)+'" text-anchor="middle">'+(hh<10?'0':'')+hh+':'+(mm<10?'0':'')+mm+'</text>'}
 svg+='<line class="chart-axis" x1="'+l+'" y1="'+(h-b)+'" x2="'+(w-r)+'" y2="'+(h-b)+'"></line>';
 // 绿线：今天已采到的叫号进度。
 const hps=hist.map(p=>x(hhmmMinute(p.bucket))+','+y(p.latest_called_no||p.called_no_typical||0));
 if(hps.length>=2)svg+='<polyline points="'+hps.join(' ')+'" fill="none" stroke="var(--green)" stroke-width="2.6" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>';
 // 橙虚线：推测未来叫号。从最后一个历史点接到外推序列，让「现在→未来」连贯。
 if(proj.length>=1){
  const startPts=[];if(hist.length>=1){const last=hist[hist.length-1];startPts.push(x(hhmmMinute(last.bucket))+','+y(last.latest_called_no||last.called_no_typical||0));}
  proj.forEach(p=>startPts.push(x(hhmmMinute(p.bucket))+','+y(p.today_projected_no)));
  if(startPts.length>=2)svg+='<polyline points="'+startPts.join(' ')+'" fill="none" stroke="#ff8c00" stroke-width="2.4" stroke-dasharray="6 4" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"></polyline>';
 }
 hist.forEach(p=>{const v=p.latest_called_no||p.called_no_typical||0;const cx=x(hhmmMinute(p.bucket)),cy=y(v);const tip=escA((p.bucket||'')+'\n当前叫到：'+fmtN(v)+' 号'+(p.latest_queue_groups?'\n在等 '+fmtN(p.latest_queue_groups)+' 桌':'')+(p.latest_wait_minutes?'\n等位 '+fmtN(p.latest_wait_minutes)+' 分':''));svg+='<g class="chart-hot" data-tip="'+tip+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="3.5" fill="#fff" stroke="var(--green)" stroke-width="2"></circle></g>'});
 proj.forEach(p=>{const cx=x(hhmmMinute(p.bucket)),cy=y(p.today_projected_no);const tip=escA((p.bucket||'')+'\n推测叫到：约 '+fmtN(p.today_projected_no)+' 号\n（按今天叫号速度外推，仅供参考）');svg+='<g class="chart-hot" data-tip="'+tip+'" onmousemove="dashTip(event,this)" onclick="dashTip(event,this)" onmouseleave="hideDashTip()"><circle cx="'+cx+'" cy="'+cy+'" r="3" fill="#ff8c00" stroke="#ff8c00" stroke-width="1.5" opacity="0.85"></circle></g>'});
 svg+='</svg>';
 const note='绿线=今天已采到的叫号，橙虚线=按今天叫号速度推测的接下来几点叫到几号。继续开着采集，过几天这里会叠加「按历史规律」的全天曲线。';
 return'<p class="ph-sub" style="margin:0 0 8px">今日叫号进度 + 推测未来：绿线是今天实际叫号，橙虚线是按当前速度推接下来叫到几号。悬停看详情。</p><div class="chart">'+svg+'</div><p class="mu mt8">'+esc(note)+'</p>';
}
function renderDashboardAdvisor(a){const box=el('qdAdvisor');if(!box)return;a=a||{};const state=a.state||'empty',bad=state==='passed'||state==='empty',warn=state==='uncovered',cls=bad?'bad':warn?'warn':state==='milestones'?'muted':'';const source=a.source==='remote_baseline'?'线上基准':a.source?'本机记录':'无数据',conf=confText(a.confidence||'none'),target=a.target_no?('当天排队号 '+fmtN(a.target_no)):'未输入号码',miles=(a.milestones||[]).slice(0,3).map(m=>'<div class="advisor-point"><span>'+esc(m.label||'时间点')+'</span><b>'+esc(m.bucket||'-')+'</b><strong>'+fmtN(m.called_no_typical||0)+'号</strong></div>').join('');let side=miles||'<div class="advisor-point"><span>提示</span><b>选门店</b><strong>补数据</strong></div>';box.innerHTML='<div class="advisor-card '+cls+'"><div class="advisor-main"><span class="advisor-eyebrow">'+esc(target)+' · '+esc(source)+' · 可信度'+esc(conf)+'</span><h3>'+esc(a.headline||'还不能判断能吃上的时间')+'</h3><p>'+esc(a.copy||'先选一个门店；如果没有曲线，开启本机采集后会逐步变准。')+'</p>'+(a.arrival_label?'<p><b>到店建议：</b>'+esc(a.arrival_label)+'</p>':'')+'</div><div class="advisor-milestones">'+side+'</div></div>'}
function fmtN(v){return Number(v||0).toLocaleString('zh-CN')}
function trendDeltaText(v){return(v>0?'↑ '+fmtN(v):v<0?'↓ '+fmtN(Math.abs(v)):'平稳')}
function shortTime(v){if(!v)return'-';const d=new Date(v);if(Number.isNaN(d.getTime()))return String(v).slice(11,16)||String(v);return d.toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit',hour12:false})}
function dashTip(e,node){let t=el('dashTip');if(!t){t=document.createElement('div');t.id='dashTip';t.className='dash-tip';document.body.appendChild(t)}t.textContent=node.getAttribute('data-tip')||'';t.style.display='block';const x=Math.min(window.innerWidth-280,e.clientX+14),y=Math.min(window.innerHeight-170,e.clientY+14);t.style.left=Math.max(8,x)+'px';t.style.top=Math.max(8,y)+'px'}
function hideDashTip(){const t=el('dashTip');if(t)t.style.display='none'}
function toggleHint(btn){let p=btn.nextElementSibling;if(!p||!p.classList.contains('hint-pop')){p=document.createElement('div');p.className='hint-pop';p.textContent=btn.getAttribute('data-hint')||'';btn.after(p)}const open=p.style.display==='block';document.querySelectorAll('.hint-pop').forEach(h=>{if(h!==p)h.style.display='none'});if(open){p.style.display='none';return}p.style.display='block';const r=btn.getBoundingClientRect(),pw=p.offsetWidth,ph=p.offsetHeight;let x=r.left,y=r.bottom+6;if(x+pw>window.innerWidth-8)x=Math.max(8,window.innerWidth-pw-8);if(y+ph>window.innerHeight-8)y=Math.max(8,r.top-ph-6);p.style.left=x+'px';p.style.top=y+'px'}
document.addEventListener('click',e=>{if(!e.target.closest('.hint-btn'))document.querySelectorAll('.hint-pop').forEach(h=>h.style.display='none')})
function queueStarterHTML(){return'<div class="queue-starter"><h3>先选一家常去门店</h3><p>现在去吃页只看你关注的门店，避免一上来被全国门店淹没。看排队不用通行证。</p><div class="queue-starter-grid"><button class="queue-starter-card read" onclick="openStorePicker({selected:qtSelected,onConfirm:applyQueueStores})" type="button"><span>今天去吃</span><b>选门店看排队</b><small>搜城市或门店名，勾选常去门店。</small></button><button class="queue-starter-card read" onclick="go(\'qd\')" type="button"><span>我有号码</span><b>算几点能吃上</b><small>已经拿到当天排队号，直接填号码估时间。</small></button><button class="queue-starter-card read" onclick="go(\'gu\')" type="button"><span>第一次用</span><b>先看机制图</b><small>分清当天排队号、未来预约和通行证。</small></button></div></div>'}
async function lQT(){await ensureStores();initQueueTrendFilters();renderQueueTrendStores();await refreshQueueView()}
function initQueueTrendFilters(){if(!qtSelected.length){const saved=recallStores('sushiro_qt_stores');if(saved.length)qtSelected=saved;else if((pr.selected_stores||[]).length)qtSelected=(pr.selected_stores||[]).map(String);else if(stores.length)qtSelected=[String(stores[0].id)]}}
function renderQueueTrendStores(){const c=el('qtStores');if(!c)return;if(!qtSelected.length){c.innerHTML='<span class="mu">尚未选择门店，点上方「选择门店（全国）」从全国门店里挑。</span>';return}c.innerHTML=qtSelected.map(id=>'<button class="chip on" data-store="'+escA(String(id))+'">'+esc(storeDisplayName(id))+' ✕</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{const id=b.dataset.store;qtSelected=qtSelected.filter(x=>x!==id);renderQueueTrendStores();refreshQueueView()})}
function applyQueueStores(ids){qtSelected=(ids||[]).map(String);rememberStores('sushiro_qt_stores',qtSelected);renderQueueTrendStores();refreshQueueView()}
function applyCalendarStores(ids){selStores=(ids||[]).map(String);rStoreChoices();rC()}
let allStoresCache=null;
async function ensureAllStores(){if(allStoresCache)return allStoresCache;try{const d=await safeFetch('/api/queue/stores');allStoresCache=d.stores||[]}catch(e){allStoresCache=[]}return allStoresCache}
function storeDisplayName(id){id=String(id);const c=(allStoresCache||[]).find(s=>String(s.id)===id);if(c)return c.name||id;const a=(stores||[]).find(s=>String(s.id)===id);if(a)return a.nickname||a.name||id;const p=(qtPanels||[]).find(x=>String(x.store_id)===id);if(p)return p.store_name||id;const t=(qtTrendStores||[]).find(x=>String(x.store_id)===id);if(t)return t.store_name||id;return id}
function openStorePicker(opts){
  opts=opts||{};
  let ov=el('storePicker');
  if(!ov){ov=document.createElement('div');ov.id='storePicker';ov.className='ov';document.body.appendChild(ov)}
  ov._sel=new Set((opts.selected||[]).map(String));
  ov._multi=opts.multi!==false;
  ov._onConfirm=opts.onConfirm||function(){};
  ov.innerHTML='<div class="ovc"><div class="fl ai jb mb16"><b>选择门店（全国）</b><button class="bt bt-w bt-s" type="button" onclick="closeStorePicker()">关闭</button></div><input id="spSearch" type="search" enterkeyhint="search" autocomplete="off" placeholder="搜城市 / 门店名 / 区" oninput="renderStorePickerList()" onkeydown="if(event.key===\'Enter\'){event.preventDefault();confirmStorePicker()}"><div id="spList" class="splist mt8"><span class="mu">加载中…</span></div><div class="fl ai jb mt16"><span class="mu" id="spCount"></span><button class="bt bt-r" type="button" onclick="confirmStorePicker()">确定</button></div></div>';
  ov.onclick=e=>{if(e.target===ov)closeStorePicker()};
  ov.classList.remove('hid');
  ov.style.display='flex';
  document.body.style.overflow='hidden';
  ensureAllStores().then(()=>{
    renderStorePickerList();
    const s=el('spSearch');
    if(s){try{s.focus({preventScroll:true})}catch(err){try{s.focus()}catch(e){}}}
  });
}
function closeStorePicker(){
  const ov=el('storePicker');
  if(ov){ov.classList.add('hid');ov.style.display='none'}
  document.body.style.overflow='';
}
function renderStorePickerList(){const ov=el('storePicker');if(!ov)return;const sel=ov._sel,q=(el('spSearch').value||'').trim().toLowerCase();const list=(allStoresCache||[]).filter(s=>{if(!q)return true;return[s.name,s.nameKana,s.area,s.address].some(v=>String(v||'').toLowerCase().includes(q))});const groups={};list.forEach(s=>{const city=s.nameKana||s.area||'其他';(groups[city]=groups[city]||[]).push(s)});const cities=Object.keys(groups).sort();el('spList').innerHTML=cities.map(city=>'<div class="spgroup"><div class="spcity">'+esc(city)+' <span class="mu">('+groups[city].length+')</span></div>'+groups[city].map(s=>{const id=String(s.id),on=sel.has(id),wait=s.wait||0,open=s.storeStatus==='OPEN',tk=String(s.netTicketStatus||'').toUpperCase(),tkOpen=tk==='ONLINE'||tk.indexOf('OPEN')>=0;const badges='<span class="spb '+(open?'ok':'mut')+'">'+(open?'营业':'休息')+'</span>'+(wait>0?'<span class="spb warn">等位'+wait+'分</span>':'')+(tkOpen?'<span class="spb ok">可取号</span>':'');return'<label class="sprow'+(on?' on':'')+'"><input type="checkbox" '+(on?'checked':'')+' onchange="toggleStorePick(\''+escA(id)+'\',this.checked)"><div class="spname">'+esc(s.name||id)+'<div class="mu">'+esc(s.area||'')+'</div></div><div class="spbs">'+badges+'</div></label>'}).join('')+'</div>').join('')||'<span class="mu">没找到匹配门店</span>';el('spCount').textContent='已选 '+sel.size+' 家'}
function toggleStorePick(id,on){const ov=el('storePicker');if(!ov)return;if(!ov._multi){ov._sel.clear();if(on)ov._sel.add(String(id));renderStorePickerList();return}if(on)ov._sel.add(String(id));else ov._sel.delete(String(id));el('spCount').textContent='已选 '+ov._sel.size+' 家'}
function confirmStorePicker(){const ov=el('storePicker');if(!ov)return;const ids=Array.from(ov._sel);closeStorePicker();(ov._onConfirm||function(){})(ids)}
function onNtModeChange(){const m=el('ntMode')?el('ntMode').value:'time',w=el('ntTimeWrap');if(w)w.classList.toggle('hid',m==='on_open')}
async function refreshQueueView(){await loadQueueLive();loadQueueRecommend();await loadNetTicketRoutine();await loadNetTicketPlan();await loadQueueAlerts();await loadQueueAlertStatus()}
// 多门店排队压力推荐：复用单店 advisor，按压力从低到高排序。
async function loadQueueRecommend(){const box=el('qtRecommend');if(!box)return;const ids=(qtSelected||[]).slice(0,6);if(ids.length<2){box.innerHTML='';return}box.innerHTML='<div class="ci">正在比较各店排队压力…</div>';try{const advs=await Promise.all(ids.map(id=>safeFetch('/api/queue/advisor?store='+encodeURIComponent(id)).catch(()=>null)));const items=advs.filter(Boolean).map(a=>{const p=a.pressure||{},c=a.current||{};return{name:a.store_name||a.store_id,level:p.level||'unknown',score:p.level==='unknown'?9999:(p.score||0),label:p.label||'数据不足',trend:p.trend_label||'',wait:c.official_wait_minutes||0,groups:c.waiting_groups||0,open:(c.store_status||'').toUpperCase()==='OPEN'}});if(!items.length){box.innerHTML='';return}items.sort((a,b)=>a.score-b.score);const best=items[0];const cards=items.map((x,i)=>'<div class="rec-card'+(i===0&&x.level!=='unknown'?' rec-best':'')+'"><div class="fl ai jb g8"><b>'+esc(x.name)+'</b><span class="answer-chip" style="padding:2px 8px"><strong class="'+pressureClass(x.level)+'">'+esc(x.label)+'</strong></span></div><div class="mu mt8">'+(x.level==='unknown'?'实时数据不足':('前面约 '+fmtN(x.groups)+' 桌 · 官方等待 '+fmtN(x.wait)+' 分'+(x.trend?' · '+esc(x.trend):'')))+'</div></div>').join('');const lead=best.level==='unknown'?'各店实时数据暂不足，先看下方实时排队。':('压力最低：<b>'+esc(best.name)+'</b>（'+esc(best.label)+'），更可能快点吃上。');box.innerHTML='<div class="cd-t" style="margin-bottom:8px">去哪家更快 <span class="tag read">只读</span></div><div class="answer-lead" style="font-size:15px">'+lead+'</div><div class="rec-grid mt8">'+cards+'</div>'}catch(e){box.innerHTML=''}}
let qtAlerts=[];
// qdReminderChannels 是「当次提醒」卡片上用户勾选的 per-level 通道（英文 key：feishu/telegram/bark/serverchan）。
// 为空 = 全通道（兼容老配置）。与全局只读的 notifyChannels（中文名、展示用）区分。
let qdReminderChannels=[];
const REMINDER_CHANNEL_KEYS=[['feishu','飞书'],['telegram','Telegram'],['bark','Bark'],['serverchan','Server酱']];
function reminderChannelsAvailable(){return REMINDER_CHANNEL_KEYS.filter(([k])=>notifyChannels.some(n=>n===k||n===({feishu:'飞书',telegram:'Telegram',bark:'Bark',serverchan:'Server酱'})[k]))}
function renderReminderChannels(){const box=el('remChannels'),hint=el('remChannelsHint');if(!box)return;const avail=reminderChannelsAvailable();if(!avail.length){box.innerHTML='';if(hint)hint.textContent='还没配置任何通知渠道，将走桌面通知；到设置页配一个更稳。';return}box.innerHTML=avail.map(([k,n])=>'<button type="button" class="chip '+(qdReminderChannels.includes(k)?'on':'')+'" data-ch="'+k+'">'+esc(n)+'</button>').join('');if(hint)hint.textContent=qdReminderChannels.length?('仅发：'+qdReminderChannels.map(k=>REMINDER_CHANNEL_KEYS.find(x=>x[0]===k)?.[1]||k).join('、')):'不选 = 全部已配通道都发';box.querySelectorAll('button[data-ch]').forEach(b=>b.onclick=()=>togReminderChannel(b.dataset.ch))}
function togReminderChannel(k){const i=qdReminderChannels.indexOf(k);if(i>=0)qdReminderChannels.splice(i,1);else qdReminderChannels.push(k);renderReminderChannels()}
async function loadQueueAlerts(){try{const d=await safeFetch('/api/queue/alerts');qtAlerts=(d&&d.rules)||[];renderTicketReminderCard()}catch(e){}}
function alertNoList(v){return Array.from(new Set(String(v||'').split(/[，,\s]+/).map(x=>parseInt(x,10)).filter(x=>x>0)))}
async function removeQueueAlertByKey(key){try{let base=qtAlerts||[];try{const d=await safeFetch('/api/queue/alerts');base=(d&&d.rules)||base}catch(e){}const before=base.length;qtAlerts=base.filter(r=>qaRuleKey(r)!==key);if(qtAlerts.length===before){toast('没有找到这条提醒');return}await saveQueueAlerts();toast('已删除提醒')}catch(e){toast('删除提醒失败：'+String(e.message||e))}}
async function saveQueueAlerts(){try{const d=await safeFetch('/api/queue/alerts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({rules:qtAlerts})});qtAlerts=(d&&d.rules)||qtAlerts;await loadQueueAlertStatus()}catch(e){toast('保存提醒失败')}}
async function loadQueueLive(){const box=el('qtLive');if(!box)return;box.innerHTML='<div class="skeleton skk"></div>';if(qtSelected.length){try{const ids=qtSelected.slice(0,6);const panels=await Promise.all(ids.map(id=>safeFetch('/api/queue/live?store='+encodeURIComponent(id)).catch(()=>null)));qtPanels=panels.filter(Boolean);renderQueueLivePanels(qtPanels);fillNetTicketStores()}catch(e){box.innerHTML=loadErrBoxHTML(e,'loadQueueLive()','实时排队')}return}qtPanels=[];fillNetTicketStores();box.innerHTML=queueStarterHTML()}
let qtPanels=[],ntPlan={},ntRoutine={};
function netTimeDisp(hhmm){hhmm=String(hhmm||'').replace(/\D/g,'').slice(0,4);while(hhmm.length<4)hhmm='0'+hhmm;return hhmm.slice(0,2)+':'+hhmm.slice(2,4)}
function fillNetTicketStores(){const ids=(qtSelected&&qtSelected.length)?qtSelected.map(String):(qdSelected&&qdSelected.length?qdSelected.map(String):qtPanels.map(p=>String(p.store_id)));const opts=ids.length?ids.map(id=>{const p=qtPanels.find(x=>String(x.store_id)===id);const nm=p?(p.store_name||id):storeDisplayName(id);return'<option value="'+escA(id)+'">'+esc(nm)+'</option>'}).join(''):'<option value="">先在上方选关注门店</option>';const sel=el('ntStore');if(sel){const prev=sel.value||(ntPlan&&ntPlan.store_id?String(ntPlan.store_id):'');sel.innerHTML=opts;if(prev&&ids.includes(prev))sel.value=prev}const rsel=el('nrStore');if(rsel){const prev=rsel.value||(ntRoutine&&ntRoutine.store_id?String(ntRoutine.store_id):'');rsel.innerHTML=opts;if(prev&&ids.includes(prev))rsel.value=prev}}
async function loadNetTicketPlan(){try{const p=await safeFetch('/api/queue/ticket/plan');ntPlan=p||{};fillNetTicketStores();if(el('ntTime')&&p.target_time)el('ntTime').value=netTimeDisp(p.target_time);if(el('ntStore')&&p.store_id)el('ntStore').value=String(p.store_id);if(el('ntMode'))el('ntMode').value=(p.trigger_mode==='on_open')?'on_open':'time';onNtModeChange();renderNetTicketStatus(p)}catch(e){}}
async function loadNetTicketRoutine(){try{const d=await safeFetch('/api/queue/ticket/routine');ntRoutine=(d&&d.routine)||{};if(d&&d.plan)ntPlan=d.plan;fillNetTicketStores();if(el('nrStore')&&ntRoutine.store_id)el('nrStore').value=String(ntRoutine.store_id);if(el('nrMeal')&&ntRoutine.target_meal_time)el('nrMeal').value=netTimeDisp(ntRoutine.target_meal_time);if(el('nrTravel'))el('nrTravel').value=ntRoutine.travel_minutes||0;if(el('nrSafety'))el('nrSafety').value=(ntRoutine.notify_before_minutes==null?(ntRoutine.safety_minutes==null?10:ntRoutine.safety_minutes):ntRoutine.notify_before_minutes);renderNetTicketRoutineStatus(ntRoutine)}catch(e){const b=el('nrStatus');if(b)b.innerHTML='<span class="mu">每日提醒状态读取失败：</span><code style="word-break:break-all">'+esc(String(e.message||e))+'</code> <button class="bt bt-w bt-s" onclick="loadNetTicketRoutine()">重试</button>'}}
function renderNetTicketRoutineStatus(r){
 const box=el('nrStatus');if(!box)return;r=r||{};
 if(!r.enabled){box.innerHTML='<span class="mu">未开启每日取号提醒。开启后会按目标就餐时间倒推取号窗口，并提前提醒你手动取号；样本不足时不会乱提醒。</span>';return}
 const store=esc(r.store_name||storeDisplayName(r.store_id)||r.store_id||''),meal=r.target_meal_time?netTimeDisp(r.target_meal_time):'-',pickup=r.planned_pickup_time||'',pickEnd=r.planned_pickup_end_time||'',reminder=r.reminder_time||'',range=r.recommend_pickup_range?(r.recommend_pickup_range.early+'-'+r.recommend_pickup_range.late):'',window=pickup?(pickup+(pickEnd&&pickEnd!==pickup?'-'+pickEnd:'')):'',wait=r.wait_minutes_range?('预计等 '+r.wait_minutes_range.low+'-'+r.wait_minutes_range.high+' 分钟'):'等待样本不足',risk=r.risk==='high'?'风险偏高':r.risk==='medium'?'风险中等':r.risk==='low'?'风险较低':'风险待确认';
 let head='',detail='';
 switch(r.status){
  case'armed':head='已开启：今天 '+(reminder||'?')+' 提醒你取号';detail=store+' · 目标 '+meal+' 吃 · 建议取号 '+(window||range||'待确认')+' · '+wait+' · '+risk;break;
  case'needs_notify':head='已开启：需要先配置通知';detail=r.last_error||'每日取号提醒只是提醒你手动取号，不配置通知渠道就无法按时提醒。';break;
  case'waiting_data':head='已开启：等待历史样本';detail=r.last_error||'这家店样本不足，暂不提醒。去“预测准确度”开启本机采集后会自动补齐。';break;
  case'missed':head='今天已错过提醒窗口';detail=r.last_error||'每日取号提醒明天会重新规划提醒时间。';break;
  case'notified':head='今天已提醒取号';detail=store+' · 建议取号 '+(window||range||'待确认')+' · 目标 '+meal+' 吃。';break;
  case'done':head='今天已经取到号';detail=r.last_error||'如果你已经手动取到号，可以到“我有号码”继续做叫号预测。';break;
  case'error':head='每日取号提醒保存失败';detail=r.last_error||'未知错误';break;
  default:head='已开启：等待下一次规划';detail='目标 '+meal+' 吃，后台会按历史等待倒推提醒时间。'
 }
 const notifyBtn=r.status==='needs_notify'?'<button class="bt bt-r bt-s" onclick="focusNotifySettings()">配置通知</button>':'';
 box.innerHTML='<b>'+esc(head)+'</b><div class="mu mt8">'+esc(detail)+(r.basis?'<br>'+esc(r.basis):'')+'</div><div class="fl g8 fw mt8">'+notifyBtn+'<button class="bt bt-w bt-s" onclick="openSettingsFold(&quot;fold-sm&quot;)">提升预测准确度</button><button class="bt bt-w bt-s" onclick="refreshNetTicketRoutineNow()">重新试算今天</button></div>'
}
function renderNetTicketStatus(p){
 const box=el('ntStatus');if(!box)return;p=p||{};
 const store=esc(p.store_name||p.store_id||''),tt=p.target_time?netTimeDisp(p.target_time):'';
 if(!p.enabled){box.innerHTML=!hc?'<div class="notice">自动取号需要寿司郎通行证。现在还没配置——点下方「获取通行证」后，才能定时或一开放就自动远程取号。</div><div class="fl g8 fw mt8"><button class="bt bt-r bt-s" onclick="startAuth()">获取通行证</button></div>':'<span class="mu">选门店和时间，点「启用」即可设置自动取号计划；这不是只读功能，启用前会再次确认。</span>';return}
 switch(p.status){
  case 'success':box.innerHTML='<b>已自动取号 '+esc(p.number||'(详见我的单据)')+'</b><div class="mu mt8">'+store+' · 电脑已停止当天取号轮询；现在用手机寿司郎小程序查看排队信息更稳。</div>';break;
  case 'issued_unknown':box.innerHTML='<b>⚠️ 官方提示已经发过号，但本地号码未知</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'不要重复取号，请用手机寿司郎小程序查看排队号。')+'<br>电脑已停止当天取号轮询，避免影响手机端查看。</div>';break;
  case 'retrying':box.innerHTML='<b>⏳ 取号暂未成功，窗口内继续重试</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'如果提示通行证需要刷新，请先重新获取')+'</div>';break;
  case 'error':{const authErr=/E010|error\\.server|凭证|认证/.test(String(p.last_error||''));box.innerHTML='<b>⚠️ 取号失败</b><div class="mu mt8">'+store+' '+tt+'：'+esc(p.last_error||'未知错误')+'<br>'+(authErr?'寿司郎通行证会过期或被手机端登录顶掉，请先重置通行证。':'改时间后重新启用可重试。')+'</div>'+(authErr?'<div class="mt8"><button class="bt bt-r bt-s" onclick="resetAuthAndStart()">重置并重新获取</button></div>':'');break;}
  case 'expired':box.innerHTML='<b>⏰ 未在窗口内取到号</b><div class="mu mt8">'+store+' '+tt+'：超时已放弃，可重新启用。</div>';break;
  default:box.innerHTML='<b>⏳ 已设定：'+tt+' 自动取号</b><div class="mu mt8">'+store+' · 到点(约 '+tt+')自动远程取号并发一次通知。取到后电脑会停止当天轮询。</div>';
 }
}
async function refreshNetTicketRoutineNow(){if(!ntRoutine||!ntRoutine.enabled){await loadNetTicketRoutine();return}const before=ntRoutine.notify_before_minutes==null?(ntRoutine.safety_minutes==null?10:ntRoutine.safety_minutes):ntRoutine.notify_before_minutes;try{const d=await safeFetch('/api/queue/ticket/routine',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:true,store:ntRoutine.store_id,store_name:ntRoutine.store_name||storeDisplayName(ntRoutine.store_id),target_meal_time:ntRoutine.target_meal_time,travel_minutes:ntRoutine.travel_minutes||0,notify_before_minutes:before})});if(d.error){toast(d.error);return}ntRoutine=d.routine||{};if(d.plan)ntPlan=d.plan;fillNetTicketStores();renderNetTicketRoutineStatus(ntRoutine);renderNetTicketStatus(ntPlan);toast('已重新试算今天')}catch(e){toast('重新试算失败')}}
async function saveNetTicketRoutine(enabled){const sel=el('nrStore'),store=sel?sel.value:'',meal=el('nrMeal')?.value||'',travel=Math.max(0,parseInt(el('nrTravel')?.value||'0',10)||0),before=Math.max(0,parseInt(el('nrSafety')?.value||'0',10)||0);if(enabled){if(!store){toast('请先选门店');return}if(!meal){toast('请填想几点吃');return}if(!nfc){toast('启用每日取号提醒前必须先配置通知渠道');focusNotifySettings();return}if(!await confirmDialog('启用每日取号提醒？\\n每天会按目标就餐时间倒推取号窗口，并提前提醒你手动取号。\\n不会自动向寿司郎提交取号请求。'))return}else if(ntRoutine&&ntRoutine.enabled){if(!await confirmDialog('关闭每日取号提醒？\\n这只会停止未来提醒，不会取消已经拿到的排队号。'))return}const sn=(sel&&sel.selectedOptions[0])?sel.selectedOptions[0].textContent:'';try{const d=await safeFetch('/api/queue/ticket/routine',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:enabled,store:store,store_name:sn,target_meal_time:compactTime(meal),travel_minutes:travel,notify_before_minutes:before})});if(d.error){toast(d.error);return}ntRoutine=d.routine||{};if(d.plan)ntPlan=d.plan;fillNetTicketStores();renderNetTicketRoutineStatus(ntRoutine);renderNetTicketStatus(ntPlan);toast(enabled?'已开启每日取号提醒':'已关闭每日取号提醒')}catch(e){toast('保存每日取号提醒失败')}}
async function saveNetTicketPlan(enabled){const sel=el('ntStore'),tEl=el('ntTime'),modeEl=el('ntMode'),store=sel?sel.value:'',mode=modeEl?modeEl.value:'time',t=tEl?tEl.value:'';if(enabled){if(!store){toast('请先选门店');return}if(mode==='time'&&!t){toast('请填取号时间');return}const tip=mode==='on_open'?'门店一开放线上取号就会自动远程取号。':'到 '+t+' 会自动远程取号。';if(!await confirmDialog('启用自动取号计划？\\n'+tip+'\\n取到号后请尽快到店；这不是只读功能。'))return}else if(ntPlan&&ntPlan.enabled){if(!await confirmDialog('取消自动取号计划？\\n这只会停止本工具未来自动取号，不会取消已经拿到的排队号。'))return}const sn=(sel&&sel.selectedOptions[0])?sel.selectedOptions[0].textContent:'',tt=t?t.replace(':',''):'';try{const p=await safeFetch('/api/queue/ticket/plan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:enabled,store:store,store_name:sn,trigger_mode:mode,target_time:tt})});if(p.error){toast(p.error);return}ntPlan=p;renderNetTicketStatus(p);toast(enabled?(mode==='on_open'?'已启用：门店一开放线上取号就自动取号':('已启用定时取号：'+netTimeDisp(tt)+' 自动取号')):'已取消取号计划')}catch(e){toast('保存失败')}}
async function recoverNetTicketStatus(){try{const d=await safeFetch('/api/queue/ticket/status',null,12000);const t=d.ticket||{},p=d.plan||{};ntPlan=p;renderNetTicketStatus(p);lR();toast('已恢复当前排队号：'+(t.number||p.number||'(详见我的单据)'))}catch(e){toast('恢复失败：'+String(e.message||e))}}
async function cancelNetTicket(){if(!await confirmDialog('危险操作：取消当前排队号？\\n这会取消寿司郎小程序里的排队号，取消后不可恢复。\\n如果你只是想停止本工具，请点“取消计划”或“停止”。'))return;try{const d=await safeFetch('/api/queue/ticket/cancel',{method:'POST'});if(d.error){toast('取消失败：'+d.error);return}toast('已取消排队号');await loadNetTicketPlan();loadActiveTickets(true);if(typeof lR==='function')lR()}catch(e){toast('取消失败：'+String(e.message||e))}}
function sparkSVG(arr){if(!arr||arr.length<2)return'';const w=140,h=34,mn=Math.min(...arr),mx=Math.max(...arr),rg=(mx-mn)||1,n=arr.length,dx=w/(n-1);const pts=arr.map((v,i)=>(i*dx).toFixed(1)+','+(h-3-((v-mn)/rg)*(h-6)).toFixed(1)).join(' ');return'<svg class="spark" viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="none"><polyline points="'+pts+'"/></svg>'}
function queueLiveRaw(v){return String(v||'').trim().toUpperCase()}
function queueLiveOpen(s){return !!(s&&(s.online_open||queueLiveRaw(s.store_status||s.storeStatus)==='OPEN'))}
function queueStoreStatusLabel(v){const raw=queueLiveRaw(v);if(!raw)return'';if(raw==='OPEN')return'营业中';if(raw==='CLOSED'||raw==='OFFLINE_CLOSED')return'暂停营业';if(raw.indexOf('OPEN')>=0)return'营业中';if(raw.indexOf('CLOSE')>=0||raw.indexOf('OFFLINE')>=0)return'暂停营业';return''}
function queueTicketStatusLabel(v,open){const raw=queueLiveRaw(v);if(open)return'线上可取号';if(raw.indexOf('OPEN')>=0||raw.indexOf('ONLINE')>=0)return'线上可取号';if(raw.indexOf('PAUSE')>=0||raw.indexOf('OFFLINE')>=0||raw.indexOf('CLOSE')>=0)return'线上取号暂停';return'线上取号暂停'}
function queueLiveStatusLabel(s){const open=queueLiveOpen(s),parts=[],store=queueStoreStatusLabel(s&&(s.store_status||s.storeStatus)),ticket=queueTicketStatusLabel(s&&(s.net_ticket_status||s.netTicketStatus),open);if(store)parts.push(store);if(ticket&&!parts.includes(ticket))parts.push(ticket);return parts.join(' · ')||'状态待更新'}
function queueLiveWaitMinutes(s,open){if(!open)return null;const vals=[s&&s.eta_minutes,s&&s.server_wait_minutes,s&&s.wait];for(const v of vals){const n=Number(v);if(Number.isFinite(n)&&n>0)return Math.round(n)}const g=Number(s&&s.wait_groups);if(Number.isFinite(g)&&g>0)return Math.ceil(g*2);if(Number.isFinite(g)&&g===0)return 0;return null}
function queueLiveWaitHigh(m){return Math.max(m+10,m+Math.floor(m/2))}
function queueLiveTimeAt(s,min){let base=new Date(s&&s.observed_at?s.observed_at:Date.now());if(Number.isNaN(base.getTime()))base=new Date();const d=new Date(base.getTime()+Math.max(0,min)*60000);return d.toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit',hour12:false})}
function queueLiveEtaText(s,open){if(!open)return'暂停';const m=queueLiveWaitMinutes(s,open);if(m==null)return'—';return m+'-'+queueLiveWaitHigh(m)+' 分钟'}
function queueLiveNowMealText(s,open){if(!open)return'暂停取号';const m=queueLiveWaitMinutes(s,open);if(m==null)return'待刷新';return queueLiveTimeAt(s,m)+'-'+queueLiveTimeAt(s,queueLiveWaitHigh(m))}
function waitLevel(s){const eta=(s.eta_minutes!=null)?s.eta_minutes:(s.server_wait_minutes||0),cap=s.wait_time_cap||180,pct=eta<=0?0:Math.max(5,Math.min(100,Math.round(eta/cap*100))),lvl=eta<=0?'g':eta<=30?'g':eta<=90?'y':'r';return{eta:eta,pct:pct,lvl:lvl}}
function renderQueueLivePanels(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="empty">还没拿到实时排队数据，请刷新或换一家门店。<div class="mt8"><button class="bt bt-w bt-s" onclick="refreshQueueView()">重试</button></div></div>';return}const note=currentUIMode()==='advanced'?'门店、叫号、在等桌数为公开实时信息；远程取号是会执行操作的实验性功能，确认后才会提交。':'门店、叫号、在等桌数为公开实时信息；简化版保持只读，不会替你取号。';box.innerHTML='<div class="queue-live-grid">'+rows.map(s=>{const open=queueLiveOpen(s),card=open?'open':'closed',status=open?'可取号':'暂停取号',statusMeta=queueLiveStatusLabel(s),etaTxt=queueLiveEtaText(s,open),mealTxt=queueLiveNowMealText(s,open),called15=s.called_15m!=null?('+'+s.called_15m):'待收集',rate=s.rate_per_min!=null?(s.rate_per_min.toFixed(1)+' 桌/分'):'待收集',wl=waitLevel(s),trend=(s.called_15m>0)?'↑':'';return'<article class="queue-live-card '+card+'"><div class="queue-live-top"><div class="queue-live-name"><b>'+esc(s.store_name||s.store_id)+'</b><span>'+esc(statusMeta)+'</span></div><span class="queue-status '+(open?'ok':'bad')+'">'+esc(status)+'</span></div><div class="queue-live-main"><div class="queue-call"><span>当前叫号</span><strong>'+esc(s.called_no||'—')+' <em>'+esc(trend)+'</em></strong></div><div class="queue-spark">'+(sparkSVG(s.spark)||'<span class="mu">小折线待收集</span>')+'</div></div><div class="queue-metrics"><div class="queue-metric"><span>前面</span><b>'+fmtN(s.wait_groups||0)+' 桌</b></div><div class="queue-metric"><span>现在取号</span><b>'+esc(mealTxt)+'</b></div><div class="queue-metric"><span>近15分钟</span><b>'+esc(called15)+'</b></div></div><div class="queue-meter" title="拥挤度"><i class="lv-'+wl.lvl+'" style="width:'+wl.pct+'%"></i></div><div class="queue-live-foot"><span>预计等待 '+esc(etaTxt)+' · 均速 '+esc(rate)+' · 拥挤度 '+wl.pct+'%'+(s.tables_capacity?(' · 桌位 '+s.tables_capacity+(s.counters_capacity?(' / 吧台 '+s.counters_capacity):'')):'')+'</span><button class="bt bt-o bt-s advanced-only" onclick="takeTicket(\''+escA(String(s.store_id||''))+'\')">远程取号</button></div></article>'}).join('')+'</div><p class="queue-live-note">'+esc(note)+'</p>'}
function renderQueueLive(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="empty">还没拿到门店排队数据。点上方「选择门店（全国）」搜索城市或门店名，手动选择关注门店。</div>';return}box.innerHTML='<div class="sg">'+rows.map(s=>{const open=queueLiveOpen(s),groups=(s.groupQueuesCount==null?0:s.groupQueuesCount),statusMeta=queueLiveStatusLabel(s),etaTxt=queueLiveEtaText(s,open),mealTxt=queueLiveNowMealText(s,open),cls=open?'av':'full';return'<div class="sl '+cls+'"><div class="tm">'+(open?('现在取号约 '+esc(mealTxt)+' 吃上'):'暂停取号')+'</div><div class="ss">'+esc(s.name||s.id)+' · '+esc(s.nameKana||s.area||'')+'</div><div class="mu mt8">在等 '+groups+' 桌 · 预计等待 '+esc(etaTxt)+' · '+esc(statusMeta)+(open&&s.waitTimeCap?'<br>预估上限 '+esc(s.waitTimeCap)+' 分钟':'')+'</div></div>'}).join('')+'</div><p class="mu mt8">选中上方关注门店即可查看实时叫号、近15分钟叫号与均速。</p>'}
function queueStatusText(q){if(!q)return'未知';if(q.needs_auth)return'通行证需更新';if(q.needs_background)return'需开启';if(q.needs_data_refresh)return'需更新';return'正常'}
function queueTypeName(t){return t==='weekday'?'工作日':t==='workday'?'调休工作日':t==='weekend'?'周末':t==='holiday'?'节假日':t}
function confText(v){return v==='high'?'高':v==='medium'?'中':v==='low'?'低':'无'}

async function lSm(){await ensureStores();await loadSampling();loadAccuracyReport()}
/* loadAccuracyReport：渲染各店「预测 vs 实际」实测误差。bias>0=通常偏晚(低估等待)，<0=偏早。 */
async function loadAccuracyReport(){const box=el('accReport');if(!box)return;try{const d=await safeFetch('/api/queue/accuracy');const st=d.stores||[];accCalibrated=st.filter(s=>s.samples>=4).length;renderSettingsStatus();if(!st.length){box.innerHTML='<div class="empty">还没有可对账的样本。填号预测、等叫到后会自动积累。</div>';return}const rows=st.map(s=>{const dir=s.bias_min>=5?'通常偏晚':s.bias_min<=-5?'通常偏早':'基本居中';return'<tr><td data-label="门店">'+esc(storeName(s.store_id))+'</td><td data-label="平均误差">±'+Math.round(s.mae_min)+' 分</td><td data-label="偏向">'+dir+'</td><td data-label="最差">'+Math.round(s.worst_min)+' 分</td><td data-label="样本">'+esc(s.samples)+'</td></tr>'}).join('');box.innerHTML='<table class="tbl tbl-cards"><thead><tr><th>门店</th><th>平均误差</th><th>偏向</th><th>最差</th><th>样本</th></tr></thead><tbody>'+rows+'</tbody></table><p class="ps mt8">样本达 '+4+' 条后，会用实测误差自动校准该店的预测区间。</p>'}catch(e){box.innerHTML=loadErrBoxHTML(e,'loadAccuracyReport()','预测准确度')}}
async function loadSampling(){try{const d=await(await fetch('/api/sampling')).json();spCfg=d.config||{};spState=d.state||{};spAutoStart=d.autostart||{};spQueueState=d.queue_state||{};fillSamplingForm();renderSamplingStores();renderSamplingState();renderDashboardSamplingCard()}catch(e){const ss=el('sampleState');if(ss)ss.innerHTML='<div class="ci bad">预测准确度状态加载失败</div>';renderDashboardSamplingCard()}}
function fillSamplingForm(){const set=(id,fn)=>{const e=el(id);if(e)fn(e)};set('spEnabled',e=>e.checked=!!spCfg.enabled);set('spAuto',e=>e.checked=!!spCfg.auto_start);set('spInterval',e=>e.value=spCfg.interval_seconds||300);set('spStart',e=>e.value=timeInputValue(spCfg.active_start||'100000'));set('spEnd',e=>e.value=timeInputValue(spCfg.active_end||'220000'))}
function renderSamplingStores(){const c=el('samplingStores'),h=el('sampleStoreHint');if(!c)return;if(!stores.length){c.innerHTML='<span class="mu">本机采集需要寿司郎认证；只看实时排队不用。</span>';if(h)h.textContent='先获取凭证后，才能记录你常去门店的历史变化。';return}const chosen=(spCfg.store_ids||[]).map(String);c.innerHTML=stores.map(s=>'<button class="chip '+(chosen.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{b.classList.toggle('on');renderSamplingStoreHint()});renderSamplingStoreHint()}
function renderSamplingStoreHint(){const h=el('sampleStoreHint');if(!h)return;const chosen=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);if(chosen.length){h.textContent='当前记录 '+chosen.length+' 家指定门店。';return}const pref=(pr.selected_stores||[]).map(storeName).filter(Boolean);h.textContent=pref.length?'当前跟随预约/取号门店：'+pref.join('、'):'当前跟随凭证里保存的门店。'}
function samplingPayload(){const ids=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);return{enabled:el('spEnabled').checked,auto_start:el('spAuto').checked,interval_seconds:+el('spInterval').value||300,active_start:compactTime(el('spStart').value||'10:00'),active_end:compactTime(el('spEnd').value||'22:00'),store_ids:ids,use_preference_stores:ids.length===0}}
function renderSamplingState(){const s=spState||{},a=spAutoStart||{},q=spQueueState||{},next=s.next_run_at?new Date(s.next_run_at).toLocaleString():'-',last=s.last_run_at?new Date(s.last_run_at).toLocaleString():'-',msg=s.last_error||s.message||q.message||'无',bad=(s.last_error||q.needs_auth)&&!/跳过|时间窗|暂无|正在运行/.test(s.last_error||'');el('sampleState').innerHTML=chip('状态',s.running?'运行中':(s.enabled?'已启用':'未启动'),s.running?'ok':s.enabled?'warn':'')+chip('开机自启动',a.enabled?'已配置':a.supported?'未配置':'不支持',a.enabled?'ok':'warn')+chip('下次',next,'ok')+chip('上次',last,'ok')+chip('样本',s.snapshots||0,'ok')+chip('门店失败',s.store_errors||0,(s.store_errors||0)?'warn':'ok')+chip('凭证',q.auth_ok?'可用':'需更新',q.auth_ok?'ok':'bad')+chip('最近结果',msg,bad?'bad':'ok');renderSettingsStatus()}
async function saveSampling(quiet){spCfg=samplingPayload();try{const d=await(await fetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(spCfg)})).json();if(d.error){if(!quiet)toast(d.error);return false}spCfg=d.config||spCfg;spState=d.state||spState;renderSamplingStores();renderSamplingState();renderDashboardSamplingCard();if(!quiet)toast(spState.running?'预测配置已保存，后台已按新配置重启':'预测配置已保存');return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function startSampling(){if(el('spEnabled'))el('spEnabled').checked=true;if(!await saveSampling(true))return;try{const d=await(await fetch('/api/sampling/start',{method:'POST'})).json();if(d.error){toast(d.error);return}spState=d.state||spState;await loadSampling();toast('已开启本机持续采集')}catch(e){toast('启动失败')}}
async function stopSampling(){try{const d=await(await fetch('/api/sampling/stop',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();renderDashboardSamplingCard()}catch(e){toast('停止失败')}}
async function runSampleOnce(){if(!await saveSampling(true))return;const box=el('sampleResult');box.classList.remove('hid');box.textContent='收集中';try{const d=await(await fetch('/api/sampling/once',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();renderDashboardSamplingCard();const r=d.result||{};box.innerHTML=r.skipped?'本轮跳过：'+esc(r.skip_reason):'<b>收集完成</b><br>'+esc((r.stores||[]).map(x=>{const parts=[];parts.push(x.error||((x.slots||0)+' 条时段'));if(x.queue_observed)parts.push('排队 '+(x.queue_wait_groups||0)+' 组');else if(x.queue_error)parts.push('排队失败');return(x.store_name||x.store_id)+': '+parts.join('，')}).join('\\n')).replaceAll('\\n','<br>');if(cp==='qd')toast(r.skipped?'本轮跳过：'+(r.skip_reason||'未知原因'):'收集完成：'+(r.queue_snapshots||0)+' 条排队快照，'+(r.snapshots||0)+' 条时段')}catch(e){box.innerHTML='收集失败';renderDashboardSamplingCard()}}
function usePrefSamplingStores(){document.querySelectorAll('#samplingStores .chip').forEach(x=>x.classList.remove('on'));renderSamplingStoreHint()}
async function setBootSampling(enabled){try{const d=await(await fetch('/api/sampling/autostart',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})})).json();if(d.error){toast(d.error);return}spAutoStart=d.autostart||{};if(cp==='se'||cp==='qd')await loadSampling();toast(enabled?'已配置开机自启动':'已取消开机自启动')}catch(e){toast('操作失败')}}

let pendingSnTarget=null;
function snFromSlot(store_id,date,start,end){pendingSnTarget={store_id:String(store_id),date:String(date),start_after:String(start),start_before:String(end||start)};go('sn')}
async function bookSlotDirect(store_id,date,start,end,store_name){const when=fT(start)+(end?'-'+fT(end):'');if(!await confirmDialog({title:'直接预约这个时段',body:'会向寿司郎提交预约：\\n'+(store_name||store_id)+'\\n'+fD(date)+' '+when+'\\n这是会执行操作，不是只读查看；成功后可在「我的单据」查看。',ok:'确认预约',cancel:'再想想'}))return;await submitGuard('book:'+store_id+':'+date+':'+start,async()=>{try{const d=await safeFetch('/api/engine/booking',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({store:String(store_id),date:String(date),start:String(start),end:String(end||'')})});if(d.error){toast('预约失败：'+d.error);return}toast('已开始预约这个时段，进度看首页或我的单据');await loadStatus();go('da')}catch(e){toast('预约失败：'+String(e.message||e))}})}
async function lSn(){await ensureStores();if(!el('snRows').children.length)addSn();await loadSnPlan();if(pendingSnTarget){const t=pendingSnTarget;pendingSnTarget=null;const rows=el('snRows');if(rows.children.length===1&&!rows.querySelector('input').value)rows.innerHTML='';addSn(t);rows.lastElementChild?.scrollIntoView({block:'center'})}}
async function ensureStores(){if(stores.length)return;try{stores=await(await fetch('/api/stores')).json();selStores=stores.map(s=>String(s.id));}catch(e){}}
function storeOpts(v){return stores.map(s=>'<option value="'+escA(String(s.id))+'" '+(String(s.id)===String(v)?'selected':'')+'>'+esc(s.nickname||s.name||s.id)+'</option>').join('')}
function dateInputValue(v){v=String(v||'');return /^\d{8}$/.test(v)?v.slice(0,4)+'-'+v.slice(4,6)+'-'+v.slice(6,8):v}
function timeInputValue(v){v=String(v||'');return /^\d{6}$/.test(v)?v.slice(0,2)+':'+v.slice(2,4):/^\d{4}$/.test(v)?v.slice(0,2)+':'+v.slice(2,4):v}
function compactDate(v){v=String(v||'').trim();return /^\d{4}-\d{2}-\d{2}$/.test(v)?v.replaceAll('-',''):v}
function compactTime(v){v=String(v||'').trim();return /^\d{2}:\d{2}$/.test(v)?v.replace(':',''):v.replaceAll(':','')}
function validDate8(v){if(!/^\d{8}$/.test(v))return false;const d=new Date(v.slice(0,4)+'-'+v.slice(4,6)+'-'+v.slice(6,8));return !Number.isNaN(d.getTime())&&d.toISOString().slice(0,10).replaceAll('-','')===v}
function timeSec(v){v=compactTime(v);if(!/^(?:\d{4}|\d{6})$/.test(v))return -1;const h=+v.slice(0,2),m=+v.slice(2,4),s=v.length===6?+v.slice(4,6):0;return h>=0&&h<=23&&m>=0&&m<=59&&s>=0&&s<=59?h*3600+m*60+s:-1}
function addSnErr(row,msg){const d=document.createElement('div');d.className='inline-err';d.textContent=msg;row.appendChild(d)}
function addSn(t={}){const c=el('snRows'),d=document.createElement('div');d.className='sn-row';d.innerHTML='<div class="fg"><label>日期</label><input type="date" value="'+escA(dateInputValue(t.date||''))+'"></div><div class="fg"><label>最早</label><input type="time" value="'+escA(timeInputValue(t.start_after||'1930'))+'"></div><div class="fg"><label>最晚</label><input type="time" value="'+escA(timeInputValue(t.start_before||'2030'))+'"></div><div class="fg"><label>门店</label><select>'+storeOpts(t.store_id||stores[0]?.id||'')+'</select></div><button class="bt bt-o bt-s" onclick="this.parentElement.remove()">删除</button>';c.appendChild(d)}
function readSnTargets(){const rows=Array.from(el('snRows').children);let ok=true;const targets=[];rows.forEach(r=>{r.querySelector('.inline-err')?.remove();const i=r.querySelectorAll('input'),s=r.querySelector('select'),date=compactDate(i[0].value),start=compactTime(i[1].value),end=compactTime(i[2].value),ss=timeSec(start),es=timeSec(end);if(!date&&!start&&!end&&!s.value)return;if(!validDate8(date)){ok=false;addSnErr(r,'日期无效');return}if(ss<0||es<0){ok=false;addSnErr(r,'时间无效');return}if(ss>=es){ok=false;addSnErr(r,'最晚时间必须晚于最早时间');return}if(!s.value){ok=false;addSnErr(r,'请选择门店');return}targets.push({date,start_after:start,start_before:end,store_id:s.value})});return{ok,targets}}
function snTargets(){const r=readSnTargets();return r.ok?r.targets:[]}
async function saveSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){toast('请至少添加一个有效目标时段');return}try{const d=await(await fetch('/api/sniper/plan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){toast(d.error);return}renderSnPlan(d.plan);toast('已保存蹲未来预约计划')}catch(e){toast('保存失败')}}
async function loadSnPlan(){try{const d=await(await fetch('/api/sniper/plan')).json();if(d.targets?.length){el('snRows').innerHTML='';d.targets.forEach(addSn)}renderSnPlan(d)}catch(e){}}
function renderSnPlan(p){const c=el('snPlan'),ts=p?.targets||[];if(!ts.length){c.innerHTML='<div class="empty">还没有蹲未来预约目标。点“添加目标时段”，填日期、门店和时间窗。</div>';return}c.innerHTML='<table class="tbl"><thead><tr><th>目标时段</th><th>开放窗口</th><th>状态</th><th>尝试</th><th>最后错误</th></tr></thead><tbody>'+ts.map(t=>'<tr><td>'+esc(t.store_id)+'<br>'+esc(t.date)+' '+esc(fT(t.start_after))+'-'+esc(fT(t.start_before))+'</td><td>'+esc(t.open_at?new Date(t.open_at).toLocaleString():'-')+'<br>'+(t.countdown_seconds>0?Math.ceil(t.countdown_seconds/60)+' 分钟后':'窗口内/已结束')+'</td><td>'+esc(t.status||'-')+'</td><td>'+esc(t.attempts||0)+'</td><td>'+esc(t.last_error||'')+'</td></tr>').join('')+'</tbody></table>'}
async function startSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){toast('请至少添加一个有效目标时段');return}if(!await ensureNotifyConfigured('抢到未来预约'))return;if(!await confirmDialog('启动蹲未来预约时段？\\n到开放窗口会自动尝试创建未来预约；抢到后会停止。\\n不会取消已有预约或排队号。'))return;await submitGuard('startSn',async()=>{try{const d=await(await fetch('/api/sniper/start',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){toast(d.error);return}await loadStatus();await loadSnPlan();toast('蹲未来预约计划已启动，抢到的预约会出现在“我的单据”')}catch(e){toast('启动失败')}})}

function recordsEmptyHTML(kind){const needsAuth=kind==='needs_auth';return'<div class="record-empty"><h3>'+(needsAuth?'已有预约或排队号？':'还没有单据')+'</h3><p>我的单据只用来看已经成功的预约或排队号。还没开始的话，先从今天排队或未来预约进入。</p><div class="record-empty-grid"><button class="record-empty-card read" onclick="go(\'qt\')" type="button"><span>今天去吃</span><b>先看今天排队</b><small>看营业、等位、当前叫号，不需要通行证。</small></button><button class="record-empty-card auth" onclick="enterAdvanced(\'ca\')" type="button"><span>约未来</span><b>去约未来</b><small>先查日期和时段，提交预约前再确认。</small></button><button class="record-empty-card auth" onclick="startAuth()" type="button"><span>已有单据</span><b>获取通行证查看</b><small>已经预约或取号后，用通行证读取单据。</small></button></div></div>'}
async function lR(){const c=el('rc');if(!hc){c.innerHTML=recordsEmptyHTML('needs_auth');return}c.innerHTML='<div class="empty">正在读取你的预约和排队号。</div>';try{const d=await safeFetch('/api/reservations');if(d.error){loadStatus();c.innerHTML=loadErrBoxHTML(d.error,'lR()','我的单据');return}const items=Array.isArray(d)?d:(d.items||[]);if(!items.length){c.innerHTML=recordsEmptyHTML('empty');return}c.innerHTML='<div class="sg">'+items.map(r=>{const when=r.slot_label||[r.queueDate,fT(r.start),r.end?'-'+fT(r.end):''].filter(Boolean).join(' '),store=r.store_name||r.monitored_store_id||r.storeId||'',kind=recordKind(r);const extra=[];if(kind==='net_ticket'&&r.wait>0)extra.push('前面 '+r.wait+' 桌');if(kind==='net_ticket')extra.push(r.checkedIn?'已签到':'未签到');if(kind==='reservation')extra.push('预约时间优先');extra.push(kind==='net_ticket'?'排队号':kind==='reservation'?'预约':'类型待确认');const cancel=cancelActionHTML(r,kind);return'<div class="sl av"><div class="tm">'+esc(r.number||'-')+'</div><div class="ss">'+esc(recordStatusText(r,kind))+(store?' · '+esc(store):'')+'</div><div class="mu mt8">'+esc(when||'时间待确认')+'<br>'+esc(extra.join(' · '))+'<br>#'+esc(r.ticketId||'')+'</div>'+cancel+'</div>'}).join('')+'</div>'+localRecordsFooter(d,items)}catch(e){loadStatus();c.innerHTML=loadErrBoxHTML(e,'lR()','我的单据')}}
/* localRecordsFooter：检测到本机遗留/补录记录时，给一个「清除本地遗留记录」入口（不动官方真实单据）。 */
function localRecordsFooter(d,items){const local=(d&&d.unavailable===true)||(items||[]).some(r=>/本地/.test(String(r.status||'')));if(!local)return'';return'<div class="mu mt12">'+((d&&d.message)?esc(d.message)+'<br>':'')+'有过去遗留、已经没用的本地记录？<button class="bt bt-w bt-s" onclick="clearLocalReservations()">清除本地遗留记录</button></div>'}
async function clearLocalReservations(){if(!await confirmDialog({title:'清除本机保存的预约/排队号记录？',body:'只清掉本机缓存的记录（含过去遗留的），不会取消寿司郎小程序里的真实预约或排队号；下次会从官方重新同步。',ok:'清除',cancel:'取消'}))return;try{const d=await safeFetch('/api/reservations/local/clear',{method:'POST'});toast(d.message||'已清除本地记录');lR()}catch(e){toast('清除失败：'+String(e.message||e))}}
function hasReservationSchedule(r){return!!(r.slot_label||r.start||r.end)}
function recordKind(r){const k=String(r.kind||'').toLowerCase();if(k==='reservation'||k==='reservation_ticket')return'reservation';if(hasReservationSchedule(r))return'reservation';if(k==='net_ticket'||k==='netticket')return'net_ticket';if(r.wait>0||String(r.status||'').toUpperCase()==='WAITING')return'net_ticket';return'unknown'}
function recordStatusText(r,kind){const s=String(r.status||'').trim(),u=s.toUpperCase();if(kind==='reservation'){if(u==='WAITING')return'预约待到店';if(u==='RESERVED')return'已确认预约';if(u==='CHECKED_IN')return'已签到预约';return s||'已确认预约'}if(kind==='net_ticket'){if(u==='WAITING')return'排队中';if(u==='CALLED')return'已叫号';return s||'排队号'}return s||'-'}
function cancelActionHTML(r,kind){if(kind==='net_ticket')return'<div class="mt8"><button class="bt bt-o bt-s" onclick="cancelNetTicket()">取消排队号</button></div>';if(kind==='reservation'&&r.ticketId)return'<div class="mt8"><button class="bt bt-o bt-s" onclick="cancelTicket('+r.ticketId+',&quot;reservation&quot;)">取消预约</button></div>';return'<div class="mu mt8">为避免误取消，类型未确认的记录不提供取消按钮。</div>'}
async function cancelTicket(id,kind){if(kind!=='reservation'){toast('安全保护：排队号请使用“取消排队号”，不会走预约取消接口。');return}if(!await confirmDialog('危险操作：取消当前预约？\\n这会取消寿司郎小程序里的预约单，取消后不可恢复。\\n如果你只是想刷新状态，请不要点确认。'))return;try{const d=await(await fetch('/api/reservations/cancel',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({ticket_id:id,kind:'reservation'})})).json();if(d.error){toast('取消失败：'+d.error);return}toast('已取消预约');lR()}catch(e){toast('取消失败')}}
async function takeTicket(id){if(!await confirmDialog('现在远程取号？\\n这会向寿司郎提交取号请求，不是只读查看。\\n取号后请尽快到店，过号会作废。'))return;await submitGuard('takeTicket',async()=>{try{const d=await(await fetch('/api/queue/ticket',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({store:String(id)})})).json();if(d.error){toast('取号失败：'+d.error);return}const t=d.ticket||{};toast('已取号 '+(t.number||'(详见我的单据)')+'，请到“我的单据”查看');loadActiveTickets(true)}catch(e){toast('取号失败')}})}

async function ensurePrefsLoaded(){if(prefsLoaded&&Object.keys(pr||{}).length)return pr;if(!prefsLoading){prefsLoading=(async()=>{const d=await(await fetch('/api/preferences')).json();pr=d;prefsLoaded=true;return d})().finally(()=>{prefsLoading=null})}return prefsLoading}
async function lP(){try{const modeSeq=uiModeSeq;await ensurePrefsLoaded();if(modeSeq===uiModeSeq)cacheUIMode(pr.ui_mode==='advanced'?'advanced':'simple');fF(pr);dP(pr);renderBookingStores();uD();applyUIMode()}catch(e){applyUIMode()}}
function fF(p){el('pa').value=p.adult||2;el('pc').value=p.child||0;el('pt').value=p.table_type||'T';if(el('pphone'))el('pphone').value=p.phone_number||'';el('ppm').value=p.day_priority_mode||'date';el('pst').value=p.slot_strategy||'earliest';el('ptm').value=p.target_time||'1930';rT('wd',p.weekday_slots||[]);rT('sa',p.saturday_slots||[]);rT('su',p.sunday_slots||[])}
function rangeText(rs){return !rs||!rs.length?'不预约':rs.map(r=>fT(String(r.start||''))+'-'+fT(String(r.end||''))).join('、')}
function priText(v){return v==='weekend_first'?'周末优先':v==='weekday_first'?'工作日优先':'按日期优先'}
function stratText(v,t){return v==='latest'?'最晚可约':v==='closest'?'接近 '+fT(t||'1930'):'最早可约'}
function dP(p){const people=(p.adult||2)+' 成人'+((p.child||0)>0?' · '+p.child+' 儿童':'');const table=(p.table_type||'T')==='C'?'吧台':'桌位',pri=priText(p.day_priority_mode),str=stratText(p.slot_strategy,p.target_time);const notifyHint=(hc&&!nfc)?'<span class="line" style="color:#b81c22">⚠ 未配置通知，抢到预约 / 叫号提醒不会推送 —— <a href="#" onclick="focusNotifySettings();return false" style="color:#b81c22;text-decoration:underline">去设置</a></span>':'';el('ps').innerHTML='<b>'+esc(people)+'</b> · '+esc(table)+'<span class="line">优先级：'+esc(pri)+' · '+esc(str)+'</span><span class="line">工作日：'+esc(rangeText(p.weekday_slots))+'</span><span class="line">周六：'+esc(rangeText(p.saturday_slots))+'</span><span class="line">周日：'+esc(rangeText(p.sunday_slots))+'</span>'+notifyHint}
function storeName(id){const s=stores.find(x=>String(x.id)===String(id));return s?(s.nickname||s.name||s.id):id}
function orderedStoreIDs(){const all=stores.map(s=>String(s.id)),sel=(pr.selected_stores||[]).map(String).filter(id=>all.includes(id)),base=(pr.store_priority||[]).map(String).filter(id=>all.includes(id));let order=[];base.forEach(id=>{if(!order.includes(id))order.push(id)});sel.forEach(id=>{if(!order.includes(id))order.push(id)});all.forEach(id=>{if(!order.includes(id))order.push(id)});return{all,selected:sel.length?sel:all,order}}
async function searchStores(){const q=(el('storeSearch')?.value||'').trim(),box=el('storeSearchResults');if(!box)return;if(!q){box.innerHTML='<span class="mu">输入城市或门店名再搜。</span>';return}box.innerHTML='<span class="mu">搜索中…</span>';try{const d=await safeFetch('/api/queue/stores?limit=24&q='+encodeURIComponent(q));const list=d.stores||[];if(!list.length){box.innerHTML='<span class="mu">没找到匹配门店，换个关键词试试。</span>';return}const have=new Set(stores.map(s=>String(s.id)));box.innerHTML='<div class="store-result-grid">'+list.map(s=>{const id=String(s.id),added=have.has(id),nm=String(s.name||id);return'<div class="sl av"><div class="ss"><b>'+esc(nm)+'</b></div><div class="mu mt8">'+esc([s.nameKana,s.area].filter(Boolean).join(' · ')||'门店 '+id)+'</div><div class="mt8">'+(added?'<button class="bt bt-w bt-s" disabled>已添加</button>':'<button class="bt bt-r bt-s" onclick="addStoreFromSearch(\''+escA(id)+'\',\''+escA(nm)+'\')">添加</button>')+'</div></div>'}).join('')+'</div>'}catch(e){box.innerHTML='<div class="ci bad">搜索失败</div>'}}
async function addStoreFromSearch(id,name){id=String(id);if(!stores.some(s=>String(s.id)===id))stores.push({id:id,name:name,nickname:name});pr.selected_stores=(pr.selected_stores||[]).map(String);if(!pr.selected_stores.includes(id))pr.selected_stores.push(id);pr.store_priority=(pr.store_priority||[]).map(String);if(!pr.store_priority.includes(id))pr.store_priority.push(id);renderBookingStores();if(el('storeChoices'))rStoreChoices();await savePrefsPayload(prefsPayload(),true);searchStores()}
function renderBookingStores(){const box=el('bookingStores');if(!box)return;if(!stores.length){box.innerHTML='<span class="mu">获取通行证后可在此选择门店</span>';return}const data=orderedStoreIDs(),set=new Set(data.selected);box.innerHTML=data.order.map(id=>'<div class="store-row" data-store="'+escA(id)+'"><input type="checkbox" '+(set.has(id)?'checked':'')+'><div><b>'+esc(storeName(id))+'</b><span>'+esc(id)+'</span></div><button type="button" class="ico" title="上移" aria-label="上移门店优先级" onclick="moveStoreRow(this,-1)">↑</button><button type="button" class="ico" title="下移" aria-label="下移门店优先级" onclick="moveStoreRow(this,1)">↓</button></div>').join('')}
function moveStoreRow(btn,dir){const r=btn.closest('.store-row'),p=r.parentElement;if(dir<0&&r.previousElementSibling)p.insertBefore(r,r.previousElementSibling);if(dir>0&&r.nextElementSibling)p.insertBefore(r.nextElementSibling,r)}
function bookingStoresFromUI(){const rows=Array.from(document.querySelectorAll('#bookingStores .store-row')),selected=[];rows.forEach(r=>{if(r.querySelector('input').checked)selected.push(r.dataset.store)});return{selected_stores:selected,store_priority:selected}}
function applyPreset(k){const set=(pm,st,tm,wd,sa,su)=>{el('ppm').value=pm;el('pst').value=st;el('ptm').value=tm;rT('wd',wd);rT('sa',sa);rT('su',su)};if(k==='weekday_dinner')set('weekday_first','closest','1930',[{start:'1900',end:'2030'}],[],[]);else if(k==='weekend_lunch')set('weekend_first','earliest','1130',[],[{start:'1030',end:'1300'}],[{start:'1030',end:'1300'}]);else if(k==='weekend_dinner')set('weekend_first','closest','1930',[],[{start:'1830',end:'2030'}],[{start:'1830',end:'2030'}]);else if(k==='any_available')set('date','earliest','1930',[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}]);toast('已套用策略模板，请点击保存偏好')}
function rT(k,rs){const c=el(k);c.innerHTML='';(rs||[]).forEach(r=>{const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" value="'+escA(r.start||'')+'" placeholder="1930"><span class="sp">至</span><input type="text" value="'+escA(r.end||'')+'" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)});if(!rs||!rs.length)c.innerHTML='<span class="mu">不预约</span>'}
function aT(k){const c=el(k);if(c.querySelector('.mu'))c.innerHTML='';const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" placeholder="1930"><span class="sp">至</span><input type="text" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)}
function gT(k){const ip=document.querySelectorAll('#'+k+' input'),r=[];for(let i=0;i<ip.length;i+=2){const s=ip[i].value.trim(),e=ip[i+1]?ip[i+1].value.trim():'';if(s||e)r.push({start:s,end:e})}return r}
function prefsPayload(){const st=bookingStoresFromUI();return{ui_mode:currentUIMode(),adult:+el('pa').value||2,child:+el('pc').value||0,table_type:el('pt').value||'T',phone_number:(el('pphone')?.value||'').trim(),selected_stores:st.selected_stores,store_priority:st.store_priority,day_priority_mode:el('ppm').value||'date',day_priority:pr.day_priority||['saturday','sunday','weekday'],slot_strategy:el('pst').value||'earliest',target_time:el('ptm').value.trim()||'1930',weekday_slots:gT('wd'),saturday_slots:gT('sa'),sunday_slots:gT('su')}}
async function savePrefsPayload(b,quiet){const modeSeq=uiModeSeq;try{const d=await(await fetch('/api/preferences',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)toast(d.error);return false}pr=d.preferences||b;prefsLoaded=true;const serverMode=pr.ui_mode==='advanced'?'advanced':'simple';if(modeSeq===uiModeSeq||serverMode===currentUIMode())cacheUIMode(serverMode);else pr={...pr,ui_mode:currentUIMode()};fF(pr);dP(pr);renderBookingStores();uD();applyUIMode();if(!quiet)toast('已保存');return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function sP(){const b=prefsPayload();if(stores.length&&!b.selected_stores.length){toast('请至少选择一家预约/取号门店');return false}return savePrefsPayload(b,false)}
async function saveCalendarStoresAsPrefs(){if(!selStores.length){toast('请先选择门店');return}await lP();const b={...pr,selected_stores:selStores.slice(),store_priority:selStores.slice()};if(await savePrefsPayload(b,true))toast('已保存为预约/取号门店优先级')}
function renderCloudAuth(d){cloudAuth=d||{};const st=el('cloudState');if(el('cloudUrl')&&!el('cloudUrl').value)el('cloudUrl').value=cloudAuth.base_url||'';const cfg=!!cloudAuth.configured,conn=!!cloudAuth.connected,user=cloudAuth.user_login||'',baseOK=!!cloudAuth.baseline_connected,baseCount=(cloudAuth.baseline_rollup_count||0)+(cloudAuth.baseline_latest_count||0),baseText=baseOK?(baseCount?('已验证 '+baseCount+' 条'):'已响应，暂无样本'):(conn?'未验证':'待登录'),msg=cloudAuth.last_error?('<br><span class="bad">'+esc(cloudAuth.last_error)+'</span>'):'',who=conn?(esc(user||'GitHub')+(cloudAuth.expires_at?(' · 到期 '+esc(shortTime(cloudAuth.expires_at))):'')):(cfg?'待登录 GitHub':'未连接');if(st)st.innerHTML=chip('线上基准',cfg?'服务已配置':'未连接',cfg?'ok':'warn')+chip('GitHub',who,conn?'ok':cfg?'warn':'warn')+chip('线上数据库',baseText,baseOK?'ok':conn?'warn':'warn')+chip('本机保存','只保存应用会话，不保存数据库密钥',conn?'ok':'warn')+msg+'<div class="mu mt8">'+esc(cloudAuth.provider_message||'登录只用于线上基准，不影响寿司郎认证和本机取号。')+'</div>';renderSettingsStatus()}
async function loadCloudAuth(verify){try{renderCloudAuth(await safeFetch('/api/cloud/auth'+(verify?'?verify=1':''),null,12000))}catch(e){const st=el('cloudState');if(st)st.innerHTML='<span class="bad">加载云端状态失败：'+esc(String(e.message||e))+'</span>'}}
async function saveCloudAuth(){const base=(el('cloudUrl')?.value||'').trim();try{const d=await safeFetch('/api/cloud/auth',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({base_url:base})});renderCloudAuth(d);toast(base?'已保存云端服务地址':'已清空云端服务地址');return true}catch(e){toast('保存云端设置失败：'+String(e.message||e));return false}}
async function startCloudLogin(){const base=(el('cloudUrl')?.value||'').trim();if(base&&base!==(cloudAuth.base_url||'')){if(!await saveCloudAuth())return}if(!(cloudAuth.configured||base)){toast('还没有配置云端服务地址；自建服务请在「GitHub 登录与线上基准」高级折叠里填写。');return}location.href='/api/cloud/auth/start'}
async function testCloudAuth(){try{renderCloudAuth(await safeFetch('/api/cloud/auth/test',{method:'POST'},15000));toast('GitHub 与线上数据库连接正常')}catch(e){await loadCloudAuth(true);toast('云端连接失败：'+String(e.message||e))}}
async function logoutCloudAuth(){if(!await confirmDialog('退出云端 GitHub 会话？\\n只会清空本机保存的云端 session，不影响寿司郎凭证和本机数据。'))return;try{renderCloudAuth(await safeFetch('/api/cloud/auth/logout',{method:'POST'}));toast('已退出云端')}catch(e){toast('退出云端失败：'+String(e.message||e))}}
async function lS(){await lP();await ensureStores();renderBookingStores();try{const c=await(await fetch('/api/config')).json();el('nf').value=c.feishu?.webhook||'';el('ntt').value=c.telegram?.token||'';el('ntc').value=c.telegram?.chat_id||'';el('nbu').value=c.bark?.url||'';el('nbk').value=c.bark?.key||'';el('ns').value=c.server_chan?.key||'';notifyChannels=[];if(c.feishu?.webhook)notifyChannels.push('飞书');if(c.telegram?.token&&c.telegram?.chat_id)notifyChannels.push('Telegram');if(c.bark?.url&&c.bark?.key)notifyChannels.push('Bark');if(c.server_chan?.key)notifyChannels.push('Server酱');nfc=notifyChannels.length>0;renderSettingsStatus()}catch(e){}const verifyCloud=cloudVerifyOnLoad;cloudVerifyOnLoad=false;await loadCloudAuth(verifyCloud);await loadMobileAuth();lD()}
async function sN(quiet){const b={feishu:{webhook:el('nf').value.trim()},telegram:{token:el('ntt').value.trim(),chat_id:el('ntc').value.trim()},bark:{url:el('nbu').value.trim(),key:el('nbk').value.trim()},server_chan:{key:el('ns').value.trim()}};try{const d=await(await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)toast(d.error);return false}notifyChannels=[];if(b.feishu.webhook)notifyChannels.push('飞书');if(b.telegram.token&&b.telegram.chat_id)notifyChannels.push('Telegram');if(b.bark.url&&b.bark.key)notifyChannels.push('Bark');if(b.server_chan.key)notifyChannels.push('Server酱');nfc=notifyChannels.length>0;renderSettingsStatus();if(!quiet){toast('已保存');loadStatus().then(()=>{if(pr&&pr.adult!==undefined)dP(pr)})}return true}catch(e){if(!quiet)toast('保存失败');return false}}
async function tN(ch){if(!await sN(true))return;try{const r=await fetch('/api/notifications/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({channel:ch||'all'})}),d=await r.json();if(d.error){toast(d.error);return}const bad=(d.results||[]).filter(x=>!x.ok).map(x=>x.channel+': '+x.error);toast(bad.length?'已先保存当前表单，部分发送失败：\n'+bad.join('\n'):'已先保存当前表单，测试通知已发送')}catch(e){toast('发送失败')}}
function mobileUaTime(t){try{return t?new Date(t).toLocaleString('zh-CN',{hour12:false}):'-'}catch(e){return t||'-'}}
function capLine(c){if(!c)return'<span class="bad">尚未开始</span>';const rows=[['X-App-Code',c.x_app_code],['查询凭证',c.query_auth],['User-Agent',c.user_agent],['Referer',c.referer],['预约凭证',c.reservation_auth],['微信ID',c.wechat_id],['手机号',c.phone_number],['门店',c.store_ids]];return rows.map(x=>'<span class="'+(x[1]?'ok':'bad')+'">'+esc(x[1]?'✓ ':'⏳ ')+esc(x[0])+'</span>').join(' · ')+'<br><b>完整状态：</b>'+(c.complete?'<span class="ok">已完整</span>':'<span class="bad">未完整</span>')}
function renderMobileAuth(d){const st=el('mobileAuthState');if(!st)return;const active=!!d.active,cap=d.capture||null,logs=d.logs||[];let html='<b>'+esc(active?'手机获取中（请从「获取通行证」继续或停止）':(d.saved?'已保存':'未运行'))+'</b><br>'+esc(d.message||'')+(active?'<br>失效时间：'+esc(mobileUaTime(d.expires)):'')+'<br>CA：<code>'+esc(d.ca_path||'')+'</code><br>'+capLine(cap);if(logs.length)html+='<br><b>最近日志</b><br>'+logs.slice(-6).map(l=>esc((l.time||'')+' '+(l.message||''))).join('<br>');st.innerHTML=html}
async function loadMobileAuth(){try{renderMobileAuth(await safeFetch('/api/mobile-auth'))}catch(e){const st=el('mobileAuthState');if(st)st.innerHTML='<span class="bad">加载手机凭证状态失败：'+esc(String(e.message||e))+'</span>'}}
function chip(t,s,c){return'<div class="ci '+c+'">'+esc(t)+'：'+esc(s)+'</div>'}
function diagnosticAdvice(d){
 const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},chain=d.proxy_chain||{},net=d.network||{},eng=d.engine||{},isWin=(d.platform||{}).goos==='windows';
 const certUntrusted=isWin?(cert.cert_exists&&(!cert.current_user_trusted||!cert.local_machine_trusted)):(cert.cert_exists&&!cert.trusted);
 if(pm.stale)return{level:'bad',title:'先修复代理残留',body:'系统代理里还有上次留下的寿司郎代理。先修复代理，再重新获取通行证或启动任务。',buttons:[{l:'修复代理',f:'repairP()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(!cfg.complete)return{level:'bad',title:'先获取通行证',body:'抢预约、远程取号和读取我的单据需要完整通行证。看排队仍然可以直接用。',buttons:[{l:'获取通行证',f:'startAuth()'},{l:'先看排队',f:"go('qt')"}]};
 if(certUntrusted)return{level:'bad',title:'先信任证书',body:'证书未被系统完整信任，寿司郎小程序可能获取不到必要信息。按向导重新获取通行证并允许安装证书。',buttons:[{l:'重新获取通行证',f:'resetAuthAndStart()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(chain.checked&&!chain.ok)return{level:'bad',title:'先处理代理链路',body:'本机代理链路自检失败。保留本页诊断信息，再修复代理或发给开发者排查。',buttons:[{l:'修复代理',f:'repairP()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(net.reachable===false)return{level:'warn',title:'先确认网络',body:'当前访问寿司郎接口失败，可能是网络、地区或临时接口波动。确认网络后刷新诊断。',buttons:[{l:'刷新诊断',f:'lD()'},{l:'复制诊断',f:'copyDiag()'}]};
 if(!cfg.store_count)return{level:'warn',title:'先选常用门店',body:'选好门店后，排队、预测、可约日历和自动抢预约都会自动带入，体验会顺很多。',buttons:[{l:'选门店',f:'openGuestStorePicker()'},{l:'改预约/取号偏好',f:'openSnPrefs()'}]};
 if(!(cfg.notification_channels||[]).length)return{level:'warn',title:'建议配置通知',body:'不配置通知也能使用，但叫号提醒和抢到预约不会主动推送。',buttons:[{l:'去配置通知',f:'focusNotifySettings()'},{l:'暂时不用',f:"go('da')"}]};
 if(eng.status==='error')return{level:'bad',title:'先看运行错误',body:explainMsg(eng.message||'运行遇到问题。处理红色项后再重新启动任务。'),buttons:[{l:'查看日志',f:"openSettingsFold('fold-lo')"},{l:'复制诊断',f:'copyDiag()'}]};
 return{level:'ok',title:'本机状态正常',body:'通行证、代理、网络和通知都没有明显阻塞项。可以回首页继续查排队、预约或自动抢。',buttons:[{l:'回首页',f:"go('da')"},{l:'查可约时段',f:"enterAdvanced('ca')"}]}
}
function renderDiagnosticNext(d){
 const box=el('diagNext');if(!box)return;
 const a=diagnosticAdvice(d),buttons=journeyButtonsHTML(a.buttons);
 box.className='diag-next '+a.level;
 box.innerHTML='<h3>'+esc(a.title)+'</h3><p>'+esc(a.body)+'</p>'+(buttons?'<div class="fl g8 fw mt8">'+buttons+'</div>':'');
}
function diagDetail(d){const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},net=d.network||{},logs=(d.engine_log_tail||[]).concat((d.log_tail||[]).map(x=>({time:'',message:x}))),ports=d.ports||[],isWin=(d.platform||{}).goos==='windows';const badPorts=ports.filter(p=>!p.available&&!p.current&&!p.fallback_port).map(p=>p.name+': '+(p.error||'占用')),portNotes=ports.filter(p=>p.note).map(p=>p.name+': '+p.note),chainLines=(chain.probes||[]).map(p=>p.name+': '+(p.ok?'正常':p.skipped?'跳过':'异常')+(p.detail?'（'+p.detail+'）':''));let html='<b>下一步建议</b><br>';if(!cfg.complete)html+='先重新获取通行证。<br>';if(isWin&&cert.cert_exists&&!cert.current_user_trusted&&!cert.local_machine_trusted)html+='证书已生成但未信任，请重新获取通行证并允许管理员权限安装证书。<br>';if(isWin&&cert.current_user_trusted&&!cert.local_machine_trusted)html+='Windows 机器级证书未信任，PC 微信可能拒绝访问；请重新获取通行证并允许管理员权限。<br>';if(isWin&&!cert.current_user_trusted&&cert.local_machine_trusted)html+='Windows 当前用户证书未信任，请重新获取通行证补齐证书信任。<br>';if(!isWin&&cert.cert_exists&&!cert.trusted)html+='证书已生成但未信任，请重新获取通行证触发安装。<br>';if(chain.checked&&!chain.ok)html+='代理链路自检失败，请保留本页信息发给开发者。<br>';if(pm.stale)html+='发现代理残留，请先点“修复代理”。<br>';if(!net.reachable)html+='寿司郎网络不可达，先确认网络或稍后重试。<br>';html+='<br><b>证书</b>：<code>'+esc(cert.cert_path||'-')+'</code>'+(cert.trust_error?'<br>'+esc(cert.trust_error):'')+(isWin&&(cert.current_user_trusted||cert.local_machine_trusted)?'<br>CurrentUser='+esc(String(!!cert.current_user_trusted))+'；LocalMachine='+esc(String(!!cert.local_machine_trusted))+'；Disallowed='+esc(String(!!cert.disallowed)):'');if(badPorts.length||portNotes.length)html+='<br><b>端口</b>：'+esc(badPorts.concat(portNotes).join('；'));if((sp.summary||[]).length)html+='<br><b>系统代理</b>：'+esc(sp.summary.join('；'));html+='<br><b>代理链路</b>：'+esc(chain.summary||'未检查')+(chainLines.length?'<br>'+esc(chainLines.join('；')):'');if(logs.length)html+='<br><b>最近日志</b><br>'+logs.slice(-8).map(l=>esc((l.time||'')+' '+(l.message||''))).join('<br>');return html}
async function lD(){
 const box=el('dg'),detail=el('ddetail'),next=el('diagNext');if(!box)return;
 box.innerHTML='<div class="ci">诊断中…</div>';
 if(next){next.className='diag-next warn';next.innerHTML='<h3>先处理这件事</h3><p>正在检查通行证、代理、证书、网络和通知。</p>'}
 if(detail)detail.classList.add('hid');
 try{
  const d=await safeFetch('/api/diagnostics',null,20000);lastDiag=d;renderDiagnosticNext(d);
  const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},eng=d.engine||{},net=d.network||{},dp=d.ports||[],isWin=(d.platform||{}).goos==='windows';
  const miss=(cfg.missing||[]).join('、'),portIssues=dp.filter(p=>p.in_use&&!p.current&&!p.fallback_port).map(p=>p.name),portNotes=dp.filter(p=>p.note).map(p=>p.note),portText=portIssues.length?portIssues.join('、'):(portNotes.length?portNotes.join('、'):'默认端口可用'),certText=isWin?(cert.local_machine_trusted?'机器级已信任':cert.current_user_trusted?'用户级已信任':(cert.cert_exists?'未信任':'未生成')):(cert.trusted?'已信任':cert.cert_exists?'未信任':'未生成'),certClass=isWin?(cert.local_machine_trusted?'ok':cert.current_user_trusted?'warn':'bad'):(cert.trusted?'ok':'bad');
  const items=[];
  items.push(chip('凭证参数',cfg.complete?'完整':(miss||'未捕获'),cfg.complete?'ok':'bad'));
  items.push(chip('门店',cfg.store_count?cfg.store_count+' 个':'未选择',cfg.store_count?'ok':'bad'));
  items.push(chip('证书',certText,certClass));
  items.push(chip('端口',portText,portIssues.length?'bad':portNotes.length?'warn':'ok'));
  items.push(chip('代理残留',pm.stale?'发现残留':pm.active?'运行中':'未发现',pm.stale?'bad':pm.active?'warn':'ok'));
  items.push(chip('系统代理',sp.available?'可读取':'不可读取',sp.available?'ok':'warn'));
  items.push(chip('代理链路',chain.checked?(chain.ok?'正常':'异常'):'未运行',chain.checked?(chain.ok?'ok':'bad'):'warn'));
  items.push(chip('网络',net.reachable?'寿司郎可达':'不可达',net.reachable?'ok':'bad'));
  items.push(chip('通知',cfg.notification_channels?.length?cfg.notification_channels.join('、'):'未配置',cfg.notification_channels?.length?'ok':'warn'));
  items.push(chip('引擎',eng.status||'idle',eng.status==='error'?'bad':(eng.status==='booking'||eng.status==='capturing'||eng.status==='sniping')?'warn':'ok'));
  box.innerHTML=items.join('');
  if(detail){detail.innerHTML=diagDetail(d);detail.classList.remove('hid')}
 }catch(e){
  if(next){next.className='diag-next bad';next.innerHTML='<h3>先处理这件事</h3><p>诊断没有跑通。先确认本机服务还在运行，再重试或复制错误信息。</p><div class="fl g8 fw mt8"><button class="bt bt-w bt-s" onclick="lD()">重试</button></div>'}
  box.innerHTML=loadErrBoxHTML(e,'lD()','诊断')
 }
}
async function copyDiag(){if(!lastDiag)await lD();if(!lastDiag){toast('暂无诊断信息');return}const text=JSON.stringify(lastDiag,null,2);try{if(navigator.clipboard&&navigator.clipboard.writeText)await navigator.clipboard.writeText(text);else{const t=document.createElement('textarea');t.value=text;t.style.position='fixed';t.style.left='-9999px';document.body.appendChild(t);t.select();document.execCommand('copy');t.remove()}toast('已复制诊断信息')}catch(e){toast('复制失败，请手动选择诊断详情')}}
function authProbeHTML(d){const rs=d.results||[],ad=d.advice||[];let html='<b>基础接口自检</b>：'+(d.ok?'通过':'失败')+(d.store_id?'<br><b>门店</b>：'+esc(d.store||d.store_id)+' <code>'+esc(d.store_id)+'</code>':'');if(rs.length)html+='<br>'+rs.map(r=>esc(r.name||'-')+'：'+(r.ok?'正常':r.skipped?'跳过':'异常')+(r.status?' HTTP '+r.status:'')+(r.latency_ms?' '+r.latency_ms+'ms':'')+(r.detail?'（'+esc(r.detail)+'）':'')).join('<br>');if(ad.length)html+='<br><b>下一步</b><br>'+ad.map(esc).join('<br>');return html}
async function testAuthProbe(){const detail=el('ddetail');if(detail){detail.classList.remove('hid');detail.innerHTML='基础接口测试中...'}try{const r=await fetch('/api/auth/probe',{method:'POST'}),d=await r.json();if(detail){let h=authProbeHTML(d);if(!d.ok){const rec=recommendRecapturePath();h+='<div class="fl ai g8 fw mt8"><button class="bt bt-r bt-s" onclick="'+escA(rec.fn)+'">'+esc(rec.label)+'</button><span class="mu">'+esc(rec.hint)+'</span></div>'}detail.innerHTML=h}if(!d.ok)toast('基础接口未通过，已推荐续期方式')}catch(e){if(detail)detail.innerHTML='基础接口测试失败：'+esc(String(e));toast('基础接口测试失败')}}
async function checkCert(){const box=el('certCheckState');if(box){box.classList.remove('hid');box.innerHTML='证书自检中…'}try{const d=await(await fetch('/api/cert/check')).json();if(!box)return;const ok=!!(d.cert_exists&&d.trusted),win=pf==='windows';let detail='';if(!d.cert_exists)detail='获取通行证用的 CA 证书还没生成——点「获取通行证」走一遍会自动生成。';else if(!d.trusted)detail=win?((d.current_user_trusted||d.local_machine_trusted)?'证书已装到系统，但 PC 微信要装到「本地计算机」才认；重新获取通行证时会再弹一次 UAC，这次点「是」。':'证书没装进系统信任库。重新获取通行证时会弹 UAC，点「是」即可装上；被拒就会获取不到必要信息。'):'证书已生成但系统没信任。macOS 到「钥匙串访问」找 Sushiro CA 设为始终信任，Windows 重新获取时会弹 UAC 点「是」。';else detail='CA 证书已生成并信任 ✓ 通行证获取链路就绪。';if(d.trust_error&&d.trust_error.indexOf('not implemented')<0)detail+='（'+esc(d.trust_error)+'）';box.className='diag-detail mt8 '+(ok?'ok':'bad');box.innerHTML='<div class="fl ai g8"><span class="ci '+(ok?'ok':'bad')+'">'+(ok?'证书正常':'证书异常')+'</span><span class="mu">'+detail+'</span></div>'}catch(e){if(box){box.className='diag-detail mt8 bad';box.innerHTML='证书自检失败：'+esc(String(e))}toast('证书自检失败')}}
/* recommendRecapturePath：凭证失效/自检未过时，按上次采集方式推荐最省事的续期路径。
   capture_method 来自 auth_meta（/api/status 已返回）；mobile_proxy/未记录→手动粘贴（最低门槛），pc_wechat 非 Windows→PC 微信，Windows→手机抓包。 */
function recommendRecapturePath(){
  const m=am&&am.capture_method,win=pf==='windows';
  if(m==='pc_wechat'&&!win)return{label:'用 PC 微信自动抓',fn:'awzStartPC()',hint:'上次就是 PC 微信抓的，本机再抓一次最快'};
  if(m==='import')return{label:'再粘贴一次',fn:'startAuth();awzGo(4)',hint:'上次是手动粘贴，继续粘新内容'};
  return{label:win?'去手机抓包':'手动粘贴 / 手机抓包',fn:'startAuth()',hint:'手机抓最稳；或从抓包工具直接粘贴（免装证书）'}
}
/* verifyAuthTicket：真实取号验证。会动账号（取号后立即取消），所以先确认。 */
async function verifyAuthTicket(){
  if(!await confirmDialog({title:'验证通行证（真实取号测试）？',body:'会找一家正在开放线上取号的门店，用你的通行证真实取号、然后立即取消，以此确认通行证还能不能取号。\\n如果你当前已有排队号，则只读状态、不会动它。',ok:'开始验证',cancel:'取消'}))return;
  const box=el('authVerifyState');if(box){box.classList.remove('hid');box.innerHTML='正在取号验证…（找开放门店 → 取号 → 立即取消）'}
  try{const d=await safeFetch('/api/auth/verify',{method:'POST'},20000);
    let cls=d.valid?'ok':(d.ok?'bad':'warn');
    const rec=!d.valid?recommendRecapturePath():null;
    if(box)box.innerHTML='<div class="ci '+cls+'">'+(d.valid?'✓ ':d.ok?'✕ ':'! ')+esc(d.message||'')+'</div>'+(d.detail?'<div class="mu mt8"><code style="word-break:break-all">'+esc(d.detail)+'</code></div>':'')+(rec?'<div class="fl ai g8 fw mt8"><button class="bt bt-r bt-s" onclick="'+escA(rec.fn)+'">'+esc(rec.label)+'</button><span class="mu">'+esc(rec.hint)+'</span></div>':'');
    await loadStatus();
    toast(d.valid?'凭证有效':d.ok?('凭证已失效，已推荐续期方式'):'未能判定，请稍后重试',d.valid?'ok':d.ok?'warn':'');
  }catch(e){if(box)box.innerHTML='<div class="ci bad">验证失败：'+esc(String(e.message||e))+'</div>';toast('验证失败')}
}
async function repairP(){try{const d=await(await fetch('/api/repair-proxy',{method:'POST'})).json();toast(d.ok?'代理已恢复':'修复失败，请看 doctor');lD()}catch(e){toast('修复失败')}}
async function stopProcesses(){if(!await confirmDialog('将恢复代理、停止后台抢预约/本机采集，并退出当前应用窗口。之后就可以删除 exe 或安装目录。继续？'))return;try{const r=await fetch('/api/processes/stop',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({include_self:true})}),d=await r.json();toast(d.ok?'已发送停止请求，当前应用即将退出':'部分进程未停止，请稍后再试或重启电脑')}catch(e){toast('已发送停止请求，当前应用即将退出')}}
async function uninstallAll(){if(!await confirmDialog('将恢复代理、移除证书并清理本地敏感数据。继续？'))return;try{const d=await(await fetch('/api/uninstall',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({all:true,certificates:true,system_cert:true})})).json();toast(d.ok?'已清理':'部分清理失败，请看 doctor');lD()}catch(e){toast('清理失败')}}
async function killWeChat(){if(!await confirmDialog({title:'结束 PC 微信？',body:'会强制关闭所有微信进程（WeChat、WeChatAppEx 等）。注意：未发送的消息、未保存的草稿、正在传输的文件可能丢失，正在用的小程序会闪退。结束后请重新打开 PC 微信，进寿司郎小程序点一次排队或预约。',ok:'结束微信',cancel:'取消',danger:true}))return;try{const r=await fetch('/api/wechat/kill',{method:'POST',headers:{'Content-Type':'application/json','X-Sushiro-CSRF':csrfToken}});const d=await r.json();if(r.status===403){toast('CSRF 校验失败，请刷新页面后重试');return}toast(d.ok?('已结束 '+countWeChatOK(d)+' 个微信进程，请重新打开 PC 微信'):'没有检测到微信进程，或部分未结束——可手动在任务管理器关闭')}catch(e){toast('结束微信失败：'+String((e&&e.message)||e))}}
function countWeChatOK(d){try{return(d.results||[]).filter(x=>x.status==='ok').length}catch(e){return 0}}

async function lL(){try{const ls=await(await fetch('/api/engine/logs')).json(),v=el('lv');v.innerHTML=(ls||[]).map(l=>'<div class="ll '+(l.level==='error'?'er':'')+'"><span class="lt">'+esc(l.time)+'</span><span class="lm">'+esc(l.message)+'</span></div>').join('');v.scrollTop=v.scrollHeight}catch(e){}}

// ===== MCP 助手（设置页折叠卡）=====
let mcpState={enabled:false,auto_start:false,turso_configured:false,python_ready:false,claude_config_written:false};
async function lMCP(){await loadMCP()}
async function loadMCP(){try{const d=await safeFetch('/api/mcp');mcpState=d||{};renderMCPCard()}catch(e){const c=el('mcpCard');if(c)c.innerHTML='<div class="ci bad">MCP 状态加载失败</div>'}}
function renderMCPCard(){const c=el('mcpCard');if(!c)return;const s=mcpState||{};
  const en=s.enabled?'checked':'',as=s.auto_start?'checked':'';
  const py=s.python_ready?'<span class="ci ok">Python 依赖已就绪</span>':'<span class="ci warn">首次启用会自动装 Python 依赖（联网，约几十秒）</span>';
  const cfg=s.claude_config_written?'<span class="ci ok">已注册到 Claude Desktop</span>':'<span class="ci mu">未注册（装了 Claude Desktop 并启用后自动写）</span>';
  const turso=s.turso_configured?'<span class="ci ok">数据库已配</span>':'<span class="ci warn">未配 数据库只读密钥（查数据工具不可用）</span>';
  const tokenPlaceholder=s.turso_configured?'已保存（保密不回显，重填可覆盖）':'去 turso.tech 控制台为该库建只读 token';
  c.innerHTML='<div class="fl g8 fw mb16">'+py+cfg+turso+'</div>'
    +'<label class="check" style="width:100%;justify-content:flex-start"><input type="checkbox" id="mcpEnable" '+en+' onchange="toggleMCP(this.checked)">启用 MCP 助手</label>'
    +'<p class="ps mt8 mb16">启用后桌面端会自动准备 Python 环境并把 sushiro 注册到 Claude Desktop；重启 Claude Desktop 后，就能在对话里让 AI 帮你查排队、看预约、给到店建议。</p>'
    +'<div class="fg"><label>数据库地址</label><input id="mcpDBURL" value="'+esc(s.turso_url||'libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io')+'"></div>'
    +'<div class="fg"><label>数据库只读密钥</label><input id="mcpDBToken" type="password" placeholder="'+esc(tokenPlaceholder)+'"></div>'
    +'<div class="fl g8 fw mb16"><button class="bt bt-r bt-s" onclick="saveMCPConfig()">保存数据库配置</button></div>'
    +'<label class="check" style="width:100%;justify-content:flex-start"><input type="checkbox" id="mcpAutoStart" '+as+' onchange="toggleMCPAutostart(this.checked)">开机自动准备 MCP 环境</label>'
    +(s.message?'<p class="mu mt12">'+esc(s.message)+'</p>':'');
}
async function toggleMCP(on){try{const r=await safeFetch('/api/mcp',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:on})});if(r.error){toast(r.error);await loadMCP();return}toast(on?'正在准备 MCP 环境（首次装依赖可能需几十秒）…':'已禁用 MCP');await loadMCP()}catch(e){toast('操作失败：'+String(e.message||e))}}
async function saveMCPConfig(){const url=(el('mcpDBURL')?.value||'').trim(),tok=(el('mcpDBToken')?.value||'').trim();if(!tok){toast('请填 数据库只读密钥');return}try{const r=await safeFetch('/api/mcp',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({turso_url:url,turso_token:tok})});if(r.error){toast(r.error);return}toast('数据库配置已保存');await loadMCP()}catch(e){toast('保存失败：'+String(e.message||e))}}
async function toggleMCPAutostart(on){try{const r=await safeFetch('/api/mcp/autostart',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:on})});if(r.error){toast(r.error);await loadMCP();return}toast(on?'已设开机自启':'已取消自启');await loadMCP()}catch(e){toast('操作失败：'+String(e.message||e))}}
function aL(e){const v=el('lv');if(!v)return;const d=document.createElement('div');d.className='ll '+(e.level==='error'?'er':'');d.innerHTML='<span class="lt">'+esc(e.time)+'</span><span class="lm">'+esc(e.message)+'</span>';v.appendChild(d);const lo=el('fold-lo');if(cp==='se'&&lo&&lo.open)v.scrollTop=v.scrollHeight}
function sse(){if(cE)cE.close();const s=new EventSource('/api/events');cE=s;s.onopen=()=>{loadStatus()};s.addEventListener('engine',e=>{try{es=JSON.parse(e.data);uE();uD();if(cp==='sn'||['idle','success','error'].includes(es.status))loadSnPlan();if(es.status==='success'&&typeof lR==='function')lR();if(['idle','success','error'].includes(es.status))loadStatus()}catch(x){}});s.addEventListener('sampling',e=>{try{spState=JSON.parse(e.data);renderDashboardSamplingCard();if(cp==='se')renderSamplingState()}catch(x){}});s.addEventListener('log',e=>{try{aL(JSON.parse(e.data))}catch(x){}});s.addEventListener('calendar',e=>{try{const d=JSON.parse(e.data);if(cp==='ca'){as=[];(d.stores||[]).forEach(st=>(st.slots||[]).forEach(x=>as.push({...x,store_name:st.store_name,store_id:st.store_id})));if(as.length)rDB()}}catch(x){}});s.addEventListener('ping',()=>{});s.onerror=()=>{s.close();cE=null;setTimeout(sse,3000)}}
init();
