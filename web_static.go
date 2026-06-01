package main

const logoBase64 = "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"

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="sushiro-csrf" content="{{CSRF_TOKEN}}">
<title>SUSHIRO Overdose</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --red:#B81C22;--red-dark:#9F1419;--red-soft:#FFF1F1;
  --ink:#191817;--text:#282522;--sub:#66615C;--mute:#9B9691;
  --paper:#FFFFFF;--wash:#F5F3F1;--line:#E5E0DB;--line-strong:#D5CEC7;
  --green:#21823F;--green-soft:#ECF7EF;--yellow:#B67800;--yellow-soft:#FFF5D8;
  --blue:#2B5B83;--blue-soft:#EEF5FA;--shadow:0 12px 34px rgba(42,35,28,.08);
  --font:"PingFang SC","Hiragino Sans GB","Microsoft YaHei",-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
}
body{min-height:100vh;background:linear-gradient(180deg,#fff 0,#f7f5f2 260px,var(--wash) 100%);color:var(--text);font-family:var(--font);-webkit-font-smoothing:antialiased}
button,input,select{font:inherit}
.topline{height:4px;background:var(--red)}
.shell{max-width:1120px;margin:0 auto;padding:0 24px}
.hdr{position:sticky;top:0;z-index:20;background:rgba(255,255,255,.92);backdrop-filter:blur(18px);border-bottom:1px solid var(--line)}
.hdr-in{height:72px;display:flex;align-items:center;gap:18px}
.brand{display:flex;align-items:center;gap:12px;min-width:0}
.brand img{width:42px;height:42px;border-radius:50%}
.brand strong{display:block;font-size:16px;line-height:1;color:var(--ink)}
.brand span{display:block;margin-top:4px;font-size:11px;color:var(--mute);letter-spacing:.08em;text-transform:uppercase}
.nav{margin-left:auto;display:flex;gap:4px;padding:5px;background:#F0EDEA;border:1px solid var(--line);border-radius:999px}
.nav a{display:inline-flex;align-items:center;height:34px;padding:0 14px;border-radius:999px;color:var(--sub);font-size:13px;font-weight:700;text-decoration:none;white-space:nowrap}
.nav a.on{background:var(--paper);color:var(--red);box-shadow:0 2px 10px rgba(32,25,18,.08)}
.ver{margin-left:6px;padding:7px 11px;border-radius:999px;background:var(--ink);color:#fff;font-size:11px;font-weight:700}
.wrap{padding:30px 0 80px}
.grid{display:grid;grid-template-columns:minmax(0,1fr) 320px;gap:18px;align-items:start}
.hero{min-height:250px;background:var(--paper);border:1px solid var(--line);border-radius:10px;padding:30px;box-shadow:var(--shadow);position:relative;overflow:hidden}
.hero:before{content:"";position:absolute;inset:0 0 auto 0;height:6px;background:var(--red)}
.eyebrow{display:inline-flex;align-items:center;gap:8px;padding:6px 10px;border-radius:999px;background:var(--red-soft);color:var(--red);font-size:12px;font-weight:800}
.hero h1{margin-top:22px;font-size:34px;line-height:1.15;letter-spacing:0;color:var(--ink);max-width:560px}
.hero p{margin-top:12px;color:var(--sub);font-size:15px;line-height:1.8;max-width:620px}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:26px}
.bt{display:inline-flex;align-items:center;justify-content:center;height:42px;padding:0 20px;border:0;border-radius:999px;cursor:pointer;font-size:14px;font-weight:800;text-decoration:none;transition:transform .14s,box-shadow .14s,background .14s,border-color .14s}
.bt:hover{transform:translateY(-1px)}
.bt:disabled{opacity:.45;cursor:not-allowed;transform:none}
.bt-l{height:48px;padding:0 28px;font-size:15px}
.bt-s{height:34px;padding:0 14px;font-size:12px}
.bt-r{background:var(--red);color:#fff;box-shadow:0 10px 22px rgba(184,28,34,.22)}
.bt-r:hover{background:var(--red-dark)}
.bt-y{background:#F5BA24;color:#2C2418;box-shadow:0 8px 18px rgba(190,128,0,.16)}
.bt-o{background:var(--paper);color:var(--red);border:1px solid rgba(184,28,34,.35)}
.bt-w{background:var(--paper);color:var(--text);border:1px solid var(--line-strong)}
.side{display:flex;flex-direction:column;gap:14px}
.card,.cd{background:var(--paper);border:1px solid var(--line);border-radius:10px;padding:20px;box-shadow:0 8px 24px rgba(42,35,28,.05);margin-bottom:18px}
.card h2,.cd-t{font-size:12px;letter-spacing:.09em;text-transform:uppercase;color:var(--mute);font-weight:900;margin-bottom:14px}
.engine{border-radius:10px;border:1px solid var(--line);padding:16px;background:#FBFAF8}
.engine .row{display:flex;align-items:center;gap:10px}
.dot{width:9px;height:9px;border-radius:50%;background:var(--mute);box-shadow:0 0 0 4px rgba(155,150,145,.12)}
.engine strong{font-size:15px;color:var(--ink)}
.engine p{margin-top:8px;color:var(--sub);font-size:12px;line-height:1.6}
.engine.capturing .dot,.step.active .mark{background:var(--yellow)}
.engine.booking .dot,.engine.sniping .dot{background:var(--blue)}
.engine.success .dot,.step.done .mark{background:var(--green)}
.engine.error .dot{background:var(--red)}
.track{display:grid;gap:10px}
.step{display:grid;grid-template-columns:28px 1fr;gap:12px;align-items:start;padding:14px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8}
.step .mark{width:28px;height:28px;border-radius:999px;background:#D8D2CC;color:#fff;display:flex;align-items:center;justify-content:center;font-size:12px;font-weight:900}
.step b{display:block;color:var(--ink);font-size:14px;margin-bottom:3px}
.step span{display:block;color:var(--sub);font-size:12px;line-height:1.5}
.step.active{border-color:#E4C05E;background:var(--yellow-soft)}
.step.done{border-color:#B9DEC2;background:var(--green-soft)}
.notice{margin-top:16px;padding:13px 14px;border-radius:10px;background:var(--yellow-soft);border:1px solid #ECD681;color:#6F4B00;font-size:13px;line-height:1.6}
.summary{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin-top:18px}
.mini{padding:16px;border:1px solid var(--line);border-radius:10px;background:var(--paper)}
.mini span{display:block;color:var(--mute);font-size:12px;font-weight:800;margin-bottom:8px}
.mini strong{display:block;color:var(--ink);font-size:20px}
.mini p{margin-top:7px;color:var(--sub);font-size:12px;line-height:1.6}
.ps{font-size:13px;line-height:1.9;color:var(--sub)}
.ps b{color:var(--ink)}
.ps .line{display:block;margin-top:4px}
.cg{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:8px}
.ci{display:flex;align-items:center;gap:8px;min-height:38px;padding:9px 10px;border-radius:8px;background:#F3F0ED;color:var(--sub);font-size:12px;font-weight:700}
.ci:before{content:"";width:8px;height:8px;border-radius:50%;background:#C9C1BA}
.ci.ok{background:var(--green-soft);color:var(--green)}
.ci.ok:before{background:var(--green)}
.ci.bad{background:var(--red-soft);color:var(--red)}
.ci.bad:before{background:var(--red)}
.ci.warn{background:var(--yellow-soft);color:var(--yellow)}
.ci.warn:before{background:var(--yellow)}
.fg{margin-bottom:16px}
.fg label{display:block;margin-bottom:6px;color:var(--sub);font-size:12px;font-weight:800}
.fr{display:flex;gap:12px;flex-wrap:wrap}
input[type=number],input[type=text],input[type=time],input[type=date],select{width:100%;height:40px;padding:0 12px;background:#fff;border:1px solid var(--line-strong);border-radius:8px;color:var(--ink);font-size:14px}
input[type=number]{width:86px}
input:focus,select:focus{outline:0;border-color:var(--red);box-shadow:0 0 0 3px rgba(184,28,34,.08)}
.settings-grid{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:18px}
.tl{display:flex;flex-direction:column;gap:8px}
.tr{display:flex;align-items:center;gap:8px}
.tr input{width:82px;text-align:center}
.tr .sp{color:var(--mute)}
.tr .x{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:999px;color:var(--red);cursor:pointer;background:var(--red-soft);font-weight:900}
.at{display:inline-flex;margin-top:8px;color:var(--green);font-size:13px;font-weight:800;cursor:pointer}
.chips{display:flex;gap:8px;flex-wrap:wrap}
.chip{display:inline-flex;align-items:center;min-height:34px;padding:0 13px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:12px;font-weight:800;cursor:pointer}
.chip.on{background:var(--red);border-color:var(--red);color:#fff}
.check{display:inline-flex;align-items:center;gap:8px;height:40px;padding:0 12px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);font-size:13px;font-weight:800;cursor:pointer}
.check input{width:auto;height:auto}
.store-list{display:grid;gap:8px}
.store-row{display:grid;grid-template-columns:auto minmax(0,1fr) auto auto;align-items:center;gap:8px;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8}
.store-row b{font-size:13px;color:var(--ink);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.store-row span{font-size:11px;color:var(--mute)}
.ico{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border:1px solid var(--line-strong);border-radius:999px;background:#fff;color:var(--sub);cursor:pointer;font-size:13px;font-weight:900}
.preset-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(132px,1fr));gap:8px;margin-bottom:16px}
.preset{min-height:42px;padding:0 12px;border:1px solid var(--line-strong);border-radius:10px;background:#FBFAF8;color:var(--text);font-size:12px;font-weight:900;cursor:pointer}
.sample-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px}
.sample-state{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:10px;margin-top:14px}
.metric{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px;margin-bottom:14px}
.chart{min-height:260px;padding:16px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;overflow:auto}
.chart svg{width:100%;min-width:680px;height:260px;display:block}
.chart-grid{stroke:#E5E0DB;stroke-width:1}
.chart-axis{stroke:#BDB5AD;stroke-width:1.2}
.chart-label{fill:var(--mute);font-size:11px;font-weight:700}
.chart-legend{display:flex;gap:12px;flex-wrap:wrap;margin:10px 0 0;color:var(--sub);font-size:12px;font-weight:800}
.legend-line{display:inline-flex;align-items:center;gap:6px}
.legend-line:before{content:"";width:18px;height:3px;border-radius:999px;background:var(--red)}
.legend-line.global:before{background:var(--blue)}
.sn-row{display:grid;grid-template-columns:1.1fr 1fr 1fr 1.1fr auto;gap:8px;align-items:end;margin-bottom:8px}
.sn-row input,.sn-row select{height:38px}
.inline-err{grid-column:1/-1;padding:8px 10px;border-radius:8px;background:var(--red-soft);color:var(--red);font-size:12px;font-weight:800}
.tbl{width:100%;border-collapse:collapse;font-size:13px}
.tbl th,.tbl td{padding:9px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
.tbl th{color:var(--mute);font-size:11px;text-transform:uppercase;letter-spacing:.06em}
.db{display:flex;gap:8px;overflow-x:auto;padding-bottom:8px;margin:14px 0 18px}
.dc{flex:0 0 auto;min-width:76px;padding:10px 12px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;cursor:pointer;text-align:center}
.dc.on{background:var(--red);border-color:var(--red);color:#fff}
.dc .dw{font-size:11px;color:var(--mute)}.dc.on .dw{color:rgba(255,255,255,.72)}
.dc .dd{margin-top:2px;font-size:16px;font-weight:900}
.dc .dv{margin-top:3px;font-size:11px;font-weight:800}.dc .dv.h{color:var(--green)}.dc .dv.n{color:var(--red)}.dc.on .dv{color:#fff}
.sg{display:grid;grid-template-columns:repeat(auto-fill,minmax(126px,1fr));gap:9px}
.sl{padding:13px;border:1px solid var(--line);border-radius:10px;background:#F7F4F1}
.sl.av{background:var(--green-soft);border-color:#B9DEC2}
.sl.fu{opacity:.52}
.sl .tm{font-size:15px;font-weight:900;color:var(--ink)}
.sl .ss{margin-top:4px;font-size:12px;color:var(--sub)}
.lg{max-height:430px;overflow:auto;padding:14px;border-radius:10px;background:#181614;color:#E8E1DA;font-family:"SF Mono",Menlo,Consolas,monospace;font-size:12px;line-height:1.75}
.ll{display:flex;gap:10px;border-bottom:1px solid rgba(255,255,255,.06);padding:2px 0}
.ll .lt{color:#9F988F;flex:0 0 auto}.ll.er .lm{color:#FFB7B7}
.empty{padding:32px;border:1px dashed var(--line-strong);border-radius:10px;text-align:center;color:var(--mute);background:#FBFAF8}
.errbox{margin-bottom:12px;padding:12px;border:1px solid #F0B7B9;border-radius:10px;background:var(--red-soft);color:var(--red);font-size:13px;line-height:1.6}
.diag-detail{margin-top:12px;padding:14px;border:1px solid var(--line);border-radius:10px;background:#FBFAF8;color:var(--sub);font-size:12px;line-height:1.7}
.diag-detail b{color:var(--ink)}
.diag-detail code{display:inline-block;max-width:100%;overflow:auto;padding:2px 5px;border-radius:6px;background:#EEE9E4;color:var(--ink)}
.ft{padding:26px 0 46px;text-align:center;color:var(--mute);font-size:12px}.ft a{color:var(--red);text-decoration:none}
.hid{display:none!important}.mu{color:var(--mute)}.tc{text-align:center}.tg{color:var(--green)}.tre{color:var(--red)}
.mt8{margin-top:8px}.mt16{margin-top:16px}.mb16{margin-bottom:16px}
.fl{display:flex}.g8{gap:8px}.g12{gap:12px}.ai{align-items:center}.jb{justify-content:space-between}.fw{flex-wrap:wrap}
@media(max-width:900px){
  .grid,.settings-grid,.sn-row{grid-template-columns:1fr}
  .summary{grid-template-columns:1fr}
  .hdr-in{height:auto;min-height:70px;flex-wrap:wrap;padding:12px 0}
  .nav{order:3;width:100%;overflow:auto}
  .nav a{flex:1;justify-content:center}
  .ver{margin-left:auto}
}
@media(max-width:600px){
  .shell{padding:0 14px}.wrap{padding-top:18px}
  .hero{padding:24px 18px}.hero h1{font-size:27px}
  .actions .bt{width:100%}.side{gap:10px}
  .card,.cd{padding:16px}
}
</style>
</head>
<body>
<div class="topline"></div>
<header class="hdr">
  <div class="shell hdr-in">
    <div class="brand">
      <img src="data:image/png;base64,` + logoBase64 + `" alt="SUSHIRO">
      <div><strong>SUSHIRO Overdose</strong><span>reservation assistant</span></div>
    </div>
    <nav class="nav">
      <a href="#" class="on" onclick="go('da',this)">首页</a>
      <a href="#" onclick="go('ca',this)">日历</a>
      <a href="#" onclick="go('in',this)">洞察</a>
      <a href="#" onclick="go('qt',this)">到店预测</a>
      <a href="#" onclick="go('sm',this)">信息收集</a>
      <a href="#" onclick="go('sn',this)">狙击</a>
      <a href="#" onclick="go('re',this)">预约</a>
      <a href="#" onclick="go('se',this)">设置</a>
      <a href="#" onclick="go('lo',this)">日志</a>
    </nav>
    <span class="ver" id="ver">loading</span>
  </div>
</header>

<main class="shell wrap">
  <section id="p-da">
    <div class="grid">
      <div>
        <div class="hero">
          <div class="eyebrow" id="heroBadge">当前步骤</div>
          <h1 id="heroTitle">正在读取状态</h1>
          <p id="heroCopy">请稍等。</p>
          <div class="actions">
            <button class="bt bt-r bt-l" id="bm" onclick="mA()">开始</button>
            <button class="bt bt-o hid" id="bs" onclick="sE()">停止</button>
            <button class="bt bt-w" id="bc" onclick="sC()">重新获取认证</button>
          </div>
          <div id="nc" class="notice hid"></div>
        </div>
        <div id="cb" class="card hid mt16">
          <h2>认证获取进度</h2>
          <div id="cg" class="cg"></div>
        </div>
        <div class="summary">
          <div class="mini"><span>人数</span><strong id="sumPeople">2 成人</strong><p id="sumTable">桌位</p></div>
          <div class="mini"><span>选号策略</span><strong id="sumSlot">未设置</strong><p>时段范围在设置中调整</p></div>
          <div class="mini"><span>运行</span><strong id="sumRun">就绪</strong><p id="sumRunSub">等待下一步</p></div>
          <div class="mini"><span>信息收集</span><strong id="sumSampling">未启动</strong><p id="sumSamplingSub">用于到店预测</p></div>
        </div>
      </div>
      <aside class="side">
        <div id="eb" class="engine idle"><div class="row"><span class="dot"></span><strong>就绪</strong></div><p>等待操作。</p></div>
        <div class="card hid" id="updBox"></div>
        <div class="card">
          <h2>新手路径</h2>
          <div class="track">
            <div class="step" id="step-capture" style="cursor:pointer" onclick="sC()"><div class="mark">1</div><div><b>获取认证</b><span>重启 PC 微信，在小程序里点一次「排队」或「预约」。</span></div></div>
            <div class="step" id="step-prefs" style="cursor:pointer" onclick="go('se',document.querySelector('[onclick*=se]'))"><div class="mark">2</div><div><b>选门店和偏好</b><span>勾选要抢的门店、调整人数桌型和时段。</span></div></div>
            <div class="step" id="step-sampling" style="cursor:pointer" onclick="go('sm',document.querySelector('[onclick*=sm]'))"><div class="mark">3</div><div><b>信息收集（可选）</b><span>后台观察门店变化，用于判断到店时间。</span></div></div>
            <div class="step" id="step-booking" style="cursor:pointer" onclick="mA()"><div class="mark">4</div><div><b>开始抢号</b><span>按门店、日期、时段策略依次尝试。</span></div></div>
          </div>
          <div class="fl g8 fw mt16"><button class="bt bt-w bt-s" onclick="go('sm',document.querySelector('[onclick*=sm]'))">开始信息收集</button><button class="bt bt-w bt-s" onclick="go('se',document.querySelector('[onclick*=se]'))">设置抢号优先级</button></div>
        </div>
        <div class="card">
          <h2>当前偏好</h2>
          <div class="ps" id="ps"></div>
          <button class="bt bt-w bt-s mt16" onclick="go('se',document.querySelector('[onclick*=se]'))">修改设置</button>
        </div>
      </aside>
    </div>
  </section>

  <section id="p-ca" class="hid">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8">
        <div><div class="cd-t">预约日历</div></div>
        <div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="saveCalendarStoresAsPrefs()">保存为抢号门店</button><button class="bt bt-w bt-s" onclick="rC()">刷新</button><select id="ar" onchange="setAR()" style="width:auto"><option value="0">不自动刷新</option><option value="15">15 秒</option><option value="30">30 秒</option><option value="60">60 秒</option></select></div>
      </div>
      <div class="fg"><label>门店</label><div id="storeChoices" class="chips"><span class="mu">加载中</span></div></div>
      <div class="fl g8 fw mb16">
        <label class="check"><input type="checkbox" id="avOnly" onchange="rC()">只看可预约</label>
        <select id="period" onchange="rC()" style="width:auto"><option value="all">全部时段</option><option value="lunch">午餐</option><option value="dinner">晚餐</option></select>
      </div>
      <div class="db" id="dbar"></div>
      <div id="sc"><div class="empty">选择门店查看时段</div></div>
    </div>
  </section>

  <section id="p-in" class="hid">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8"><div class="cd-t" style="margin-bottom:0">历史洞察</div><button class="bt bt-w bt-s" onclick="lI()">刷新</button></div>
      <div id="ic"><div class="empty">加载中</div></div>
    </div>
  </section>

  <section id="p-qt" class="hid">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8"><div><div class="cd-t" style="margin-bottom:0">到店预测</div><p class="mu mt8">先收集你关心门店的排队和预约变化，用来判断更适合到店的时间；预测仅供参考。</p></div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="refreshQueueView()">刷新预测</button><button class="bt bt-r bt-s" onclick="setBootSampling(true)">启用开机收集</button></div></div>
      <div class="fg"><label>关注门店</label><div id="qtStores" class="chips"><span class="mu">从本地数据自动识别</span></div></div>
      <div id="qtLive" class="sample-state"><div class="ci">实时排队待加载</div></div>
      <div class="mt16" id="qtAlertCard">
        <div class="fl ai jb fw g8"><label style="margin:0">叫号提醒</label><span class="mu">后台收集运行时，命中条件会用你配置的通知渠道推送。</span></div>
        <div class="fl g8 fw mt8">
          <div class="fg"><label>类型</label><select id="qaType" onchange="onQaTypeChange()"><option value="wait_below">该取号了（预估等待）</option><option value="called_reach">快叫到我（按叫号）</option></select></div>
          <div class="fg" id="qaWaitWrap"><label>等待≤(分钟)</label><input type="number" id="qaWait" value="60"></div>
          <div class="fg hid" id="qaTargetWrap"><label>我的号</label><input type="number" id="qaTarget"></div>
          <div class="fg hid" id="qaLeadWrap"><label>提前(桌)</label><input type="number" id="qaLead" value="5"></div>
          <div class="fg" style="align-self:flex-end"><button class="bt bt-r bt-s" onclick="addQueueAlert()">新增提醒</button></div>
        </div>
        <div id="qtAlerts" class="mt8"><span class="mu">尚未设置提醒</span></div>
      </div>
      <div class="sample-grid">
        <div class="fg"><label>日期类型</label><select id="qtType" onchange="loadQueueTrends()"><option value="all">全部</option><option value="weekday">工作日</option><option value="weekend">周末</option><option value="holiday">节假日</option></select></div>
        <div class="fg"><label>开始日期</label><input type="date" id="qtFrom" onchange="loadQueueTrends()"></div>
        <div class="fg"><label>结束日期</label><input type="date" id="qtTo" onchange="loadQueueTrends()"></div>
        <div class="fg"><label>开始时间</label><input type="time" id="qtStart" value="10:00" onchange="loadQueueTrends()"></div>
        <div class="fg"><label>结束时间</label><input type="time" id="qtEnd" value="22:00" onchange="loadQueueTrends()"></div>
        <div class="fg"><label>粒度</label><select id="qtBucket" onchange="loadQueueTrends()"><option value="30">30 分钟</option><option value="60">60 分钟</option></select></div>
      </div>
      <div id="qtStatus" class="sample-state"><div class="ci">尚未加载</div></div>
      <div id="qtAdvice" class="mt16"></div>
      <div id="qtChart" class="chart mt16"><div class="empty">加载中</div></div>
      <div id="qtTable" class="mt16"></div>
    </div>
  </section>

  <section id="p-sm" class="hid">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8"><div><div class="cd-t" style="margin-bottom:0">信息收集</div><p class="mu mt8">选择你关心的门店和时间窗，后台会在本机积累预约与排队变化，帮助判断到店时间。</p></div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="runSampleOnce()">立即收集一次</button><button class="bt bt-r bt-s" onclick="startSampling()">开始信息收集</button><button class="bt bt-w bt-s" onclick="setBootSampling(true)">启用开机自启动</button><button class="bt bt-o bt-s" onclick="setBootSampling(false)">取消开机自启动</button><button class="bt bt-o bt-s" onclick="stopSampling()">暂停运行</button></div></div>
      <div class="sample-grid">
        <label class="check"><input type="checkbox" id="spEnabled">启用信息收集</label>
        <label class="check"><input type="checkbox" id="spAuto">应用启动后自动收集</label>
        <div class="fg"><label>间隔秒数</label><input type="number" id="spInterval" min="60" step="60" value="300"></div>
        <div class="fg"><label>开始</label><input type="time" id="spStart" value="10:00"></div>
        <div class="fg"><label>结束</label><input type="time" id="spEnd" value="22:00"></div>
      </div>
      <div class="fg"><label>收集门店</label><div id="samplingStores" class="chips"><span class="mu">加载中</span></div><div id="sampleStoreHint" class="ps mt8"></div></div>
      <div class="fl g8 fw"><button class="bt bt-r" onclick="saveSampling()">保存信息收集配置</button><button class="bt bt-w" onclick="usePrefSamplingStores()">跟随抢号门店</button></div>
      <div id="sampleState" class="sample-state"><div class="ci">尚未加载</div></div>
      <div id="sampleResult" class="diag-detail hid"></div>
    </div>
  </section>

  <section id="p-sn" class="hid">
    <div class="cd">
      <div class="fl ai jb mb16 fw g8"><div class="cd-t" style="margin-bottom:0">Web 狙击计划器</div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="addSn()">添加目标</button><button class="bt bt-r bt-s" onclick="saveSn()">保存计划</button><button class="bt bt-y bt-s" onclick="startSn()">启动狙击</button></div></div>
      <div id="snRows"></div>
      <div id="snPlan" class="mt16"><div class="empty">暂无计划</div></div>
    </div>
  </section>

  <section id="p-re" class="hid">
    <div class="cd"><div class="cd-t">我的预约</div><div id="rc"><div class="empty">加载中</div></div></div>
  </section>

  <section id="p-se" class="hid">
    <div class="settings-grid">
      <div class="cd">
        <div class="cd-t">预约偏好</div>
        <div class="preset-grid">
          <button class="preset" onclick="applyPreset('weekday_dinner')">工作日晚餐</button>
          <button class="preset" onclick="applyPreset('weekend_lunch')">周末午餐</button>
          <button class="preset" onclick="applyPreset('weekend_dinner')">周末晚餐</button>
          <button class="preset" onclick="applyPreset('any_available')">有号就要</button>
        </div>
        <div class="fr mb16">
          <div class="fg"><label>成人</label><input type="number" id="pa" min="0" max="10" value="2"></div>
          <div class="fg"><label>儿童</label><input type="number" id="pc" min="0" max="10" value="0"></div>
          <div class="fg"><label>桌型</label><select id="pt"><option value="T">桌位</option><option value="C">吧台</option></select></div>
        </div>
        <div class="fg"><label>抢号门店与优先级</label><div id="bookingStores" class="store-list"><span class="mu">完成认证后可选择门店</span></div><div class="ps mt8">抢预约号会按勾选门店的排序、日期优先级和时段策略依次尝试。</div></div>
        <div class="fr mb16">
          <div class="fg"><label>日期优先级</label><select id="ppm"><option value="date">按日期优先</option><option value="weekend_first">周末优先</option><option value="weekday_first">工作日优先</option></select></div>
          <div class="fg"><label>时段策略</label><select id="pst"><option value="earliest">最早可约</option><option value="latest">最晚可约</option><option value="closest">接近目标时间</option></select></div>
          <div class="fg"><label>目标时间</label><input type="text" id="ptm" placeholder="1930"></div>
        </div>
        <div class="fg"><label>工作日时段</label><div id="wd" class="tl"></div><span class="at" onclick="aT('wd')">添加时段</span></div>
        <div class="fg"><label>周六时段</label><div id="sa" class="tl"></div><span class="at" onclick="aT('sa')">添加时段</span></div>
        <div class="fg"><label>周日时段</label><div id="su" class="tl"></div><span class="at" onclick="aT('su')">添加时段</span></div>
        <button class="bt bt-r mt8" onclick="sP()">保存偏好</button>
      </div>
      <div class="cd">
        <div class="cd-t">通知渠道</div>
        <div class="fg"><label>飞书 Webhook</label><input type="text" id="nf" placeholder="https://open.feishu.cn/..."></div>
        <div class="fr"><div class="fg" style="flex:1"><label>Telegram Token</label><input type="text" id="ntt" placeholder="123456:ABC..."></div><div class="fg" style="flex:1"><label>Chat ID</label><input type="text" id="ntc" placeholder="-100..."></div></div>
        <div class="fr"><div class="fg" style="flex:1"><label>Bark URL</label><input type="text" id="nbu" placeholder="https://api.day.app"></div><div class="fg" style="flex:1"><label>Bark Key</label><input type="text" id="nbk"></div></div>
        <div class="fg"><label>Server 酱 Key</label><input type="text" id="ns" placeholder="SCT..."></div>
        <div class="fl g8 fw mt8"><button class="bt bt-r" onclick="sN()">保存通知</button><button class="bt bt-w" onclick="tN('all')">测试全部</button><button class="bt bt-w" onclick="tN('feishu')">飞书</button><button class="bt bt-w" onclick="tN('telegram')">Telegram</button><button class="bt bt-w" onclick="tN('bark')">Bark</button><button class="bt bt-w" onclick="tN('serverchan')">Server酱</button></div>
      </div>
      <div class="cd" style="grid-column:1/-1">
        <div class="fl ai jb mb16 fw g8"><div class="cd-t" style="margin-bottom:0">本机诊断</div><div class="fl g8 fw"><button class="bt bt-w bt-s" onclick="lD()">刷新</button><button class="bt bt-w bt-s" onclick="testAuthProbe()">测试基础接口</button><button class="bt bt-w bt-s" onclick="copyDiag()">复制诊断</button><button class="bt bt-w bt-s" onclick="repairP()">修复代理</button><button class="bt bt-w bt-s" onclick="stopProcesses()">停止本应用进程</button><button class="bt bt-o bt-s" onclick="uninstallAll()">卸载清理</button></div></div>
        <div id="dg" class="cg"><div class="ci">尚未加载</div></div>
        <div id="ddetail" class="diag-detail hid"></div>
      </div>
    </div>
  </section>

  <section id="p-lo" class="hid">
    <div class="cd"><div class="cd-t">运行日志</div><div class="lg" id="lv"></div></div>
  </section>
</main>
<footer class="ft">由 <a href="https://github.com/Ryujoxys/sushiro-overdose">sushiro-overdose</a> 驱动 · 非官方工具，仅供学习</footer>

<script>
let cp='da',es={status:'idle'},hc=0,as=[],sd='',pr={},pf='',cE=null,stores=[],selStores=[],calErrs=[],arTimer=null,lastDiag=null,spCfg={},spState={status:'idle'},spAutoStart={},spQueueState={},qtSelected=[],qtTrendStores=[];
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
window.fetch=(input,init)=>{
  const opt=init?{...init}:{};
  const method=String(opt.method||(input&&input.method)||'GET').toUpperCase();
  if((method==='POST'||method==='PUT')&&sameOriginRequest(input)){
    const h=new Headers(opt.headers||(input&&input.headers)||{});
    h.set('X-Sushiro-CSRF',csrfToken);
    opt.headers=h;
  }
  return rawFetch(input,opt);
};
function el(id){return document.getElementById(id)}
function esc(s){const d=document.createElement('div');d.textContent=s==null?'':String(s);return d.innerHTML}
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
function go(n,e){document.querySelectorAll('.wrap>section[id^="p-"]').forEach(p=>p.classList.add('hid'));el('p-'+n).classList.remove('hid');document.querySelectorAll('.nav a').forEach(a=>a.classList.remove('on'));if(e)e.classList.add('on');cp=n;({ca:lC,in:lI,qt:lQT,sm:lSm,sn:lSn,re:lR,se:lS,lo:lL})[n]?.();return false}
async function loadStatus(){try{const r=await(await fetch('/api/status')).json();el('ver').textContent='v'+r.version;hc=!!r.has_config;pf=r.platform||'';es=r.engine||{status:'idle'};spState=r.sampling||spState;uE();uSamplingSummary();uD();}catch(e){el('ver').textContent='offline';}}
async function init(){await loadStatus();await lP();checkUpdate();sse();}
function isRun(){return ['capturing','booking','sniping'].includes(es.status)}
function setStep(id,state){const x=el(id);x.classList.remove('active','done');if(state)x.classList.add(state)}
function explainMsg(m){m=String(m||'');if(/证书|trust|certificate/i.test(m))return'证书问题：先到设置页刷新诊断，确认 CA 证书已信任；失败后可重新获取认证。';if(/代理|proxy/i.test(m))return'代理问题：先点击设置页的“修复代理”，再重新获取认证。';if(/401|403|认证|token|auth/i.test(m))return'认证过期：重新获取认证参数后再启动。';if(/network|timeout|超时|不可达|connection/i.test(m))return'网络问题：确认网络可访问寿司郎接口，稍后重试。';if(/门店|store/i.test(m))return'门店配置问题：检查设置页的抢号门店是否仍在可用列表中。';return'先查看设置页本机诊断和日志，处理红色项后重试。'}
function uD(){
  const b=el('bm'),bc=el('bc'),nc=el('nc'),title=el('heroTitle'),copy=el('heroCopy'),badge=el('heroBadge');
  const run=isRun();
  b.disabled=run;bc.classList.toggle('hid',es.status==='capturing');
  nc.classList.add('hid');nc.textContent='';
  const _hasStores=(pr.selected_stores||[]).length>0;
  const _prefsDone=hc&&_hasStores;
  setStep('step-capture',hc?'done':'active');
  setStep('step-prefs',!hc?'':(_prefsDone?'done':'active'));
  setStep('step-sampling',spState.running?'active':(spState.sample_runs>0?'done':''));
  setStep('step-booking',!_prefsDone?'':(es.status==='success'?'done':'active'));
  if(es.status==='capturing'){
    badge.textContent='正在获取认证';title.textContent='请操作 PC 微信';copy.textContent='第一步：在任务管理器里关闭所有 WeChat 进程；第二步：重新打开 PC 微信 → 寿司郎小程序 → 选任意门店点一次「排队」或「预约」（不用真的提交）。';
    b.textContent='获取中';b.className='bt bt-y bt-l';b.onclick=sC;
  }else if(es.status==='booking'||es.status==='sniping'){
    badge.textContent='正在运行';title.textContent='正在为你查询目标时段';copy.textContent=es.message||'页面可以保持打开，成功后会保存预约并发送通知。';
    b.textContent='运行中';b.className='bt bt-r bt-l';b.onclick=sB;
  }else if(es.status==='success'){
    badge.textContent='已成功';title.textContent='预约成功';copy.textContent=es.message||'预约信息已保存。';
    b.textContent='查看预约';b.className='bt bt-r bt-l';b.onclick=()=>go('re',document.querySelector('[onclick*=re]'));
  }else if(es.status==='error'){
    badge.textContent='需要处理';title.textContent='运行遇到问题';copy.textContent='原始错误信息见下方，便于排查；按建议处理后点重试。';
    b.textContent=hc?'重新开始':'重新获取认证';b.className='bt bt-y bt-l';b.onclick=hc?sB:sC;
    nc.classList.remove('hid');nc.innerHTML='<b>错误</b><br><code style="word-break:break-all">'+esc(es.message||'(无错误信息)')+'</code><br><br><b>建议</b><br>'+esc(explainMsg(es.message));
  }else if(!hc){
    badge.textContent='第一步 · 首次设置';title.textContent='先获取认证参数';copy.textContent='点击下方按钮启动捕获代理，然后在 PC 微信里打开寿司郎小程序点一次「排队」或「预约」即可。';
    b.textContent='开始获取认证';b.className='bt bt-y bt-l';b.onclick=sC;
    nc.classList.remove('hid');nc.textContent='完成认证后会自动出现下一步指引：选门店、查日历、开始抢号。';
  }else{
    const hasStores=(pr.selected_stores||[]).length>0;
    if(!hasStores){
      badge.textContent='第二步 · 选门店';title.textContent='认证已保存，请先选抢号门店';copy.textContent='点下方按钮打开「设置」页，在「抢号门店与优先级」里勾选并排序你想抢的门店；保存后就能开始抢号。';
      b.textContent='去设置门店';b.className='bt bt-y bt-l';b.onclick=()=>go('se',document.querySelector('[onclick*=se]'));
    }else{
      badge.textContent='准备就绪';title.textContent='已选 '+(pr.selected_stores||[]).length+' 家门店，可以抢号';copy.textContent='当前认证已保存，也已选定抢号门店。可以直接开始抢号；或先去「预约日历」/「到店预测」看一眼再决定。';
      b.textContent='开始抢号';b.className='bt bt-r bt-l';b.onclick=sB;
    }
  }
}
function uE(){
  const box=el('eb'),bs=el('bs'),s=es||{status:'idle'};
  const label={idle:'就绪',capturing:'正在获取认证',booking:'正在抢号',sniping:'狙击中',success:'预约成功',error:'需要处理'}[s.status]||s.status;
  const desc=s.message||({idle:'等待下一步。',capturing:'等待小程序请求。',booking:'正在查询目标时段。',sniping:'高频窗口运行中。',success:'已保存预约信息。',error:'请查看日志。'}[s.status]||'');
  box.className='engine '+s.status;box.innerHTML='<div class="row"><span class="dot"></span><strong>'+esc(label)+'</strong></div><p>'+esc(desc)+'</p>';
  if(s.status==='booking'&&s.attempts)box.innerHTML+='<p>已查询 '+s.attempts+' 次</p>';
  bs.classList.toggle('hid',!isRun());
  el('sumRun').textContent=label;el('sumRunSub').textContent=desc;
  if(s.status==='capturing'&&s.capture){el('cb').classList.remove('hid');rG(s.capture)}else if(s.status!=='capturing'){el('cb').classList.add('hid')}
}
function uSamplingSummary(){const s=spState||{};el('sumSampling').textContent=s.running?'运行中':(s.enabled?'已启用':'未启动');el('sumSamplingSub').textContent=s.message||'用于到店预测'}
async function checkUpdate(){try{const u=await(await fetch('/api/update')).json(),b=el('updBox');if(!b)return;if(u.current_version==='dev'){b.classList.add('hid');return}if(u.update_available){b.classList.remove('hid');b.innerHTML='<h2>版本更新</h2><div class="ps"><b>'+esc(u.latest_version)+'</b><span class="line">当前 '+esc(u.current_version)+'</span></div><a class="bt bt-w bt-s mt16" href="'+escA(u.url||'#')+'" target="_blank">打开 Release</a>'}else b.classList.add('hid')}catch(e){}}
function rG(c){el('cg').innerHTML=need.map(k=>'<div class="ci '+(c[k]?'ok':'')+'">'+fieldName(k)+'</div>').join('')}
function fieldName(k){return {x_app_code:'App Code',query_auth:'查询认证',reservation_auth:'预约认证',user_agent:'设备信息',referer:'小程序来源',wechat_id:'微信 ID',phone_number:'手机号',store_ids:'门店'}[k]||k}
async function sC(){try{const d=await(await fetch('/api/engine/capture',{method:'POST'})).json();if(d.error)alert(d.error);await loadStatus();}catch(e){alert('启动失败')}}
async function sB(){try{const d=await(await fetch('/api/engine/booking',{method:'POST'})).json();if(d.error)alert(d.error);await loadStatus();}catch(e){alert('启动失败')}}
async function sE(){try{await fetch('/api/engine/stop',{method:'POST'});await loadStatus();}catch(e){}}
function mA(){hc?sB():sC()}

async function lC(){await ensureStores();if(!stores.length){el('storeChoices').innerHTML='<span class="mu">先完成认证获取</span>';el('sc').innerHTML='<div class="empty">先完成认证获取，再查看日历</div>';return}if(!selStores.length)selStores=stores.map(s=>String(s.id));rStoreChoices();rC()}
function rStoreChoices(){const c=el('storeChoices');c.innerHTML=stores.map(s=>'<button class="chip '+(selStores.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>togStore(b.dataset.store))}
function togStore(id){selStores=selStores.includes(id)?selStores.filter(x=>x!==id):selStores.concat(id);if(!selStores.length&&stores[0])selStores=[String(stores[0].id)];rStoreChoices();sd='';rC()}
async function rC(){if(!selStores.length)return;el('sc').innerHTML='<div class="empty">加载中…</div>';const q='stores='+encodeURIComponent(selStores.join(','))+'&available='+(el('avOnly').checked?'1':'0')+'&period='+encodeURIComponent(el('period').value||'all');try{const d=await safeFetch('/api/calendar?'+q);if(d.error){el('sc').innerHTML=loadErrBoxHTML(d.error,'rC()','日历');return}as=[];calErrs=[];(d.stores||[]).forEach(st=>{if(st.error)calErrs.push({store:st.store_name||st.store_id,error:st.error});(st.slots||[]).forEach(s=>as.push({...s,store_name:st.store_name,store_id:st.store_id}))});rDB()}catch(e){el('sc').innerHTML=loadErrBoxHTML(e,'rC()','日历')}}
function setAR(){if(arTimer){clearInterval(arTimer);arTimer=null}const sec=+el('ar').value||0;if(sec>0)arTimer=setInterval(()=>{if(cp==='ca')rC()},sec*1000)}
function fD(d){return parseInt(d.substring(4,6),10)+'/'+parseInt(d.substring(6,8),10)}
function fT(t){return t&&t.length>=4?t.substring(0,2)+':'+t.substring(2,4):t||''}
function nT(t){t=compactTime(t||'');return t.length===4?t+'00':t}
function slotMatchesPrefs(s){const dt=new Date(s.date.substring(0,4)+'-'+s.date.substring(4,6)+'-'+s.date.substring(6,8)),w=dt.getDay(),rs=w===6?(pr.saturday_slots||[]):w===0?(pr.sunday_slots||[]):(pr.weekday_slots||[]),st=nT(s.start),en=nT(s.end||s.start);return rs.some(r=>st>=nT(r.start)&&st<nT(r.end)&&en<=nT(r.end))}
function calendarErrHTML(){return calErrs.length?'<div class="errbox">'+calErrs.map(x=>'<b>'+esc(x.store)+'</b>：'+esc(x.error)).join('<br>')+'<div class="mt8"><button class="bt bt-o bt-s" onclick="sC()">重新获取认证</button></div></div>':''}
function rDB(){const g={};as.forEach(s=>{if(!g[s.date])g[s.date]=[];g[s.date].push(s)});const ds=Object.keys(g).sort(),b=el('dbar');b.innerHTML='';if(!ds.length){el('sc').innerHTML=calendarErrHTML()+'<div class="empty">暂无时段</div>';return}if(!sd||!ds.includes(sd))sd=ds[0];ds.forEach(d=>{const sl=g[d],av=sl.filter(s=>s.availability==='AVAILABLE').length,dt=new Date(d.substring(0,4)+'-'+d.substring(4,6)+'-'+d.substring(6,8)),c=document.createElement('div');c.className='dc'+(d===sd?' on':'');c.innerHTML='<div class="dw">周'+W[dt.getDay()]+'</div><div class="dd">'+fD(d)+'</div><div class="dv '+(av>0?'h':'n')+'">'+(av>0?'可约 '+av:'已满')+'</div>';c.onclick=()=>{sd=d;rDB()};b.appendChild(c)});rS(sd)}
function rS(d){const sl=as.filter(s=>s.date===d).sort((a,b)=>(a.store_name||'').localeCompare(b.store_name||'')||(a.start||'').localeCompare(b.start||'')),c=el('sc');if(!sl.length){c.innerHTML=calendarErrHTML()+'<div class="empty">无时段</div>';return}const ac=sl.filter(s=>s.availability==='AVAILABLE').length;c.innerHTML=calendarErrHTML()+'<div class="sg">'+sl.map(s=>{const a=s.availability==='AVAILABLE',m=slotMatchesPrefs(s);return'<div class="sl '+(a?'av':'fu')+'"><div class="tm">'+esc(fT(s.start))+'-'+esc(fT(s.end))+'</div><div class="ss">'+(a?'可预约':'已满')+' · '+esc(s.store_name||s.store_id||'')+(a&&m?' · 符合偏好':'')+'</div></div>'}).join('')+'</div><p class="mu mt8">'+sl.length+' 个时段 · '+ac+' 个可预约 · '+selStores.length+' 家门店</p>'}

async function lI(){await ensureStores();const c=el('ic');c.innerHTML='<div class="empty">分析中…</div>';try{const d=await safeFetch('/api/insights?top=12');if(d.error){c.innerHTML=loadErrBoxHTML(d.error,'lI()','历史洞察');return}const rec=d.recommendations||[],min=d.min_recommendation_observations||3;const metrics='<div class="metric">'+chip('历史样本',d.valid_snapshots||0,'ok')+chip('推荐门槛','同一时段 '+min+' 次','warn')+chip('推荐数量',rec.length,'ok')+'</div>';const rows=rec.map(r=>'<tr><td>'+esc(storeName(r.store_id))+'<br><span class="mu">'+esc(r.store_id)+'</span></td><td>'+esc(r.weekday_name)+'</td><td>'+esc(fT(r.start))+'-'+esc(fT(r.end))+'</td><td>'+Math.round((r.availability_rate||0)*100)+'%</td><td>'+(r.sold_out_minutes==null?'-':Math.round(r.sold_out_minutes)+' 分')+'</td><td>'+esc(r.observations)+'</td></tr>').join('');const empty=(d.valid_snapshots||0)?'<div class="empty">样本还不够稳定。保持信息收集，等同一门店、星期、时段至少积累 '+min+' 次观察后再给推荐。<div class="mt8"><button class="bt bt-w bt-s" onclick="go(\'sm\',document.querySelector(\'[onclick*=sm]\'))">去信息收集</button></div></div>':'<div class="empty">暂无历史数据。<div class="mt8"><button class="bt bt-w bt-s" onclick="go(\'sm\',document.querySelector(\'[onclick*=sm]\'))">去信息收集</button></div></div>';c.innerHTML=metrics+(rows?'<table class="tbl"><thead><tr><th>门店</th><th>星期</th><th>时段</th><th>开放概率</th><th>售罄速度</th><th>样本</th></tr></thead><tbody>'+rows+'</tbody></table>':empty)}catch(e){c.innerHTML=loadErrBoxHTML(e,'lI()','历史洞察')}}

async function lQT(){await ensureStores();initQueueTrendFilters();renderQueueTrendStores();await refreshQueueView()}
function initQueueTrendFilters(){const now=new Date(),from=new Date(now.getTime()-14*86400000);if(!el('qtFrom').value)el('qtFrom').value=localDateInput(from);if(!el('qtTo').value)el('qtTo').value=localDateInput(now);if(!qtSelected.length&&stores.length)qtSelected=stores.map(s=>String(s.id))}
function renderQueueTrendStores(){const c=el('qtStores');if(!c)return;const base=stores.length?stores:(qtTrendStores||[]).map(s=>({id:s.store_id,name:s.store_name,nickname:s.store_name}));if(!base.length){c.innerHTML='<span class="mu">完成认证或开始信息收集后，这里会出现可关注的门店。</span>';return}c.innerHTML=base.map(s=>'<button class="chip '+(qtSelected.includes(String(s.id||s.store_id))?'on':'')+'" data-store="'+escA(String(s.id||s.store_id))+'">'+esc(s.nickname||s.name||s.store_name||s.id||s.store_id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{const id=b.dataset.store;qtSelected=qtSelected.includes(id)?qtSelected.filter(x=>x!==id):qtSelected.concat(id);b.classList.toggle('on');refreshQueueView()})}
async function refreshQueueView(){await loadQueueLive();await loadQueueAlerts();await loadQueueTrends()}
let qtAlerts=[];
function onQaTypeChange(){const t=(el('qaType')||{}).value;if(!el('qaWaitWrap'))return;el('qaWaitWrap').classList.toggle('hid',t!=='wait_below');el('qaTargetWrap').classList.toggle('hid',t!=='called_reach');el('qaLeadWrap').classList.toggle('hid',t!=='called_reach')}
async function loadQueueAlerts(){try{const d=await safeFetch('/api/queue/alerts');qtAlerts=(d&&d.rules)||[];renderQueueAlerts()}catch(e){}}
function renderQueueAlerts(){const box=el('qtAlerts');if(!box)return;if(!qtAlerts.length){box.innerHTML='<span class="mu">尚未设置提醒。先在上方选好一个关注门店，再新增提醒。</span>';return}box.innerHTML='<div class="sg">'+qtAlerts.map((r,i)=>{const desc=r.type==='called_reach'?('快叫到我 · 我的号 '+(r.target_no||0)+' · 提前 '+(r.lead_groups||0)+' 桌'):('该取号了 · 预计等待 ≤ '+(r.wait_minutes||0)+' 分钟');return'<div class="sl '+(r.enabled?'av':'full')+'"><div class="ss">'+esc(r.store_name||r.store_id)+'</div><div class="mu mt8">'+esc(desc)+'</div><div class="fl g8 mt8 ai"><label class="check" style="margin:0"><input type="checkbox" '+(r.enabled?'checked':'')+' onchange="toggleQueueAlert('+i+',this.checked)">启用</label><button class="bt bt-o bt-s" onclick="removeQueueAlert('+i+')">删除</button></div></div>'}).join('')+'</div>'}
function currentAlertStore(){const id=qtSelected[0];if(!id)return null;const a=(stores||[]).find(x=>String(x.id)===String(id));const b=(qtTrendStores||[]).find(x=>String(x.store_id)===String(id));const name=a?(a.nickname||a.name):(b?b.store_name:String(id));return{id:String(id),name:name||String(id)}}
function addQueueAlert(){const s=currentAlertStore();if(!s){alert('请先在上方“关注门店”里选择一个门店');return}const type=(el('qaType')||{}).value||'wait_below';if(type==='called_reach'){const t=parseInt((el('qaTarget')||{}).value,10);if(!t){alert('请填写你手里的号');return}const lead=parseInt((el('qaLead')||{}).value,10)||0;qtAlerts.push({store_id:s.id,store_name:s.name,type:'called_reach',target_no:t,lead_groups:lead,enabled:true})}else{const m=parseInt((el('qaWait')||{}).value,10);if(!m){alert('请填写等待分钟阈值');return}qtAlerts.push({store_id:s.id,store_name:s.name,type:'wait_below',wait_minutes:m,enabled:true})}saveQueueAlerts()}
function toggleQueueAlert(i,on){if(qtAlerts[i]){qtAlerts[i].enabled=on;saveQueueAlerts()}}
function removeQueueAlert(i){qtAlerts.splice(i,1);saveQueueAlerts()}
async function saveQueueAlerts(){try{await fetch('/api/queue/alerts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({rules:qtAlerts})});renderQueueAlerts()}catch(e){alert('保存提醒失败')}}
async function loadQueueLive(){const box=el('qtLive');if(!box)return;box.innerHTML='<div class="ci">实时排队加载中…</div>';if(qtSelected.length){try{const ids=qtSelected.slice(0,6);const panels=await Promise.all(ids.map(id=>safeFetch('/api/queue/live?store='+encodeURIComponent(id)).catch(()=>null)));renderQueueLivePanels(panels.filter(Boolean))}catch(e){box.innerHTML='<div class="ci bad">实时排队加载失败</div>'}return}const p=new URLSearchParams();p.set('limit','8');try{const d=await safeFetch('/api/queue/stores?'+p.toString());renderQueueLive(d.stores||[])}catch(e){box.innerHTML='<div class="ci bad">实时排队加载失败</div>'}}
function renderQueueLivePanels(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="ci">暂无实时排队数据</div>';return}box.innerHTML='<div class="sg">'+rows.map(s=>{const open=s.online_open||s.store_status==='OPEN',cls=open?'av':'full';const eta=(s.eta_minutes!=null)?(s.eta_minutes+' 分钟'):(s.server_wait_minutes?(s.server_wait_minutes+' 分钟*'):'—');const c15=(s.called_15m!=null)?('近15分钟叫号 '+s.called_15m+' 个'):'近15分钟 收集中';const rate=(s.rate_per_min!=null)?('均速 '+s.rate_per_min.toFixed(1)+' 桌/分'):'均速 收集中';return'<div class="sl '+cls+'"><div class="tm">叫号 '+(s.called_no||'—')+'</div><div class="ss">'+esc(s.store_name||s.store_id)+'</div><div class="mu mt8">在等 '+(s.wait_groups||0)+' 桌 · 预计等待 '+eta+'<br>'+esc(c15)+' · '+esc(rate)+'<br>'+esc(s.store_status||'-')+' · '+esc(s.net_ticket_status||'-')+'</div></div>'}).join('')+'</div><p class="mu mt8">叫号与在等桌数为实时；近15分钟与均速来自本机后台收集，开启“开机收集”后逐步补齐。带 * 为接口预估值。</p>'}
function renderQueueLive(rows){const box=el('qtLive');if(!box)return;if(!rows.length){box.innerHTML='<div class="ci">暂无实时排队数据</div>';return}box.innerHTML='<div class="sg">'+rows.map(s=>{const wait=(s.wait==null?0:s.wait),groups=(s.groupQueuesCount==null?0:s.groupQueuesCount),status=s.storeStatus||'-',ticket=s.netTicketStatus||'-',cls=status==='OPEN'?'av':'full';return'<div class="sl '+cls+'"><div class="tm">预计 '+wait+' 分钟</div><div class="ss">'+esc(s.name||s.id)+' · '+esc(s.nameKana||s.area||'')+'</div><div class="mu mt8">在等 '+groups+' 桌 · '+esc(status)+' · '+esc(ticket)+(s.waitTimeCap?'<br>预估上限 '+esc(s.waitTimeCap)+' 分钟':'')+'</div></div>'}).join('')+'</div><p class="mu mt8">选中上方关注门店即可查看实时叫号、近15分钟叫号与均速。</p>'}
function queueTrendParams(){const p=new URLSearchParams();if(qtSelected.length)p.set('stores',qtSelected.join(','));p.set('date_type',el('qtType').value||'all');p.set('from',el('qtFrom').value||'');p.set('to',el('qtTo').value||'');p.set('start',el('qtStart').value||'10:00');p.set('end',el('qtEnd').value||'22:00');p.set('bucket',el('qtBucket').value||'30');return p}
async function loadQueueTrends(){const st=el('qtStatus'),chart=el('qtChart'),tbl=el('qtTable'),adv=el('qtAdvice');if(!st)return;st.innerHTML='<div class="ci">分析中…</div>';chart.innerHTML='<div class="empty">加载中…</div>';tbl.innerHTML='';if(adv)adv.innerHTML='';try{const d=await safeFetch('/api/queue/trends?'+queueTrendParams().toString());qtTrendStores=d.stores||qtTrendStores;if(!qtSelected.length&&!stores.length&&(d.stores||[]).length)qtSelected=d.stores.map(x=>String(x.store_id));renderQueueTrendStores();renderQueueTrend(d)}catch(e){const msg=String((e&&(e.message||e))||'(unknown)');st.innerHTML='<div class="ci bad">到店预测加载失败</div>';chart.innerHTML=loadErrBoxHTML(e,'loadQueueTrends()','到店预测')}}
function renderQueueTrend(d){const s=d.summary||{},q=d.sampling||{};el('qtStatus').innerHTML=chip('实际过号',s.actual_passed_total||0,(s.actual_samples||0)?'ok':'warn')+chip('全局过号',s.global_passed_total||0,(s.global_samples||0)?'ok':'warn')+chip('真实取号',s.session_records||0,(s.session_records||0)?'ok':'warn')+chip('公开快照',s.observation_records||0,(s.observation_records||0)?'ok':'warn')+chip('收集权限',queueStatusText(q),q.permission_status==='ok'?'ok':q.needs_auth?'bad':'warn')+chip('开机自启',q.system_auto_start?.enabled?'已配置':q.system_auto_start?.supported?'未配置':'不支持',q.system_auto_start?.enabled?'ok':'warn');renderQueueAdvice(d);renderQueueChart(d.series||[]);renderQueueTable(d.series||[])}
function renderQueueAdvice(d){const box=el('qtAdvice');if(!box)return;const rec=d.recommendations||[],q=d.sampling||{},warn=d.warnings||[];let html='';if(rec.length){html+='<div class="sg">'+rec.map(r=>{const wait=r.predicted_wait_minutes==null?'等待待确认':'预计等待 '+Math.round(r.predicted_wait_minutes)+' 分钟',meta=esc(r.date_type_name||r.date_type)+' · '+esc(r.bucket)+' · '+esc(confText(r.confidence))+'可信度';return'<div class="sl av"><div class="tm">'+esc(r.action_label||'候选时段')+'</div><div class="ss">'+esc(r.store_name||r.store_id)+' · '+meta+'</div><div class="mu mt8">'+wait+'<br>'+esc(r.reason||'预测仅供参考。')+'</div></div>'}).join('')+'</div><p class="mu mt8">预测仅供参考；每家门店会按本机收集的数据单独计算。</p>'}else{const msg=q.needs_auth?'先重新获取认证，再选择门店开始信息收集。':'先收集 2-3 次午餐、晚餐和周末时段，页面会开始给出门店级预测。';html+='<div class="empty">'+msg+'<div class="mt8">'+(q.needs_auth?'<button class="bt bt-o bt-s" onclick="sC()">重新获取认证</button>':'')+'<button class="bt bt-w bt-s" onclick="go(\'sm\',document.querySelector(\'[onclick*=sm]\'))">去信息收集</button></div></div>'}const steps=[];if(q.message)steps.push(q.message);if(warn.length&&!rec.length)steps.push(warn[0]);if(steps.length)html+='<div class="diag-detail"><b>下一步</b><br>'+esc(steps.join(' '))+'<div class="fl g8 fw mt8">'+(q.needs_auth?'<button class="bt bt-o bt-s" onclick="sC()">重新获取认证</button>':'')+'<button class="bt bt-w bt-s" onclick="go(\'sm\',document.querySelector(\'[onclick*=sm]\'))">调整信息收集</button></div></div>';box.innerHTML=html}
function queueStatusText(q){if(!q)return'未知';if(q.needs_auth)return'认证需更新';if(q.needs_background)return'需开启';if(q.needs_data_refresh)return'需更新';return'正常'}
function renderQueueChart(series){const box=el('qtChart');if(!series.length){box.innerHTML='<div class="empty">暂无曲线数据</div>';return}const buckets=[...new Set(series.map(x=>x.bucket))].sort(),types=[...new Set(series.map(x=>x.date_type))].sort((a,b)=>({weekday:1,weekend:2,holiday:3}[a]||9)-({weekday:1,weekend:2,holiday:3}[b]||9)),by={};series.forEach(x=>{const k=x.date_type+'|'+x.bucket;if(!by[k])by[k]={actual:0,global:0,name:x.date_type_name||x.date_type};by[k].actual+=x.actual_passed||0;by[k].global+=x.global_passed||0});let max=1;Object.values(by).forEach(v=>{max=Math.max(max,v.actual,v.global)});const w=720,h=230,pad=34,step=buckets.length>1?(w-pad*2)/(buckets.length-1):1,y=v=>h-pad-(v/max)*(h-pad*2),x=i=>pad+i*step,colors={weekday:'#B81C22',weekend:'#B67800',holiday:'#2B5B83'};let svg='<svg viewBox="0 0 '+w+' '+h+'" preserveAspectRatio="none"><line class="chart-axis" x1="'+pad+'" y1="'+(h-pad)+'" x2="'+(w-pad)+'" y2="'+(h-pad)+'"></line><line class="chart-axis" x1="'+pad+'" y1="'+pad+'" x2="'+pad+'" y2="'+(h-pad)+'"></line>';for(let i=0;i<=4;i++){const yy=pad+i*(h-pad*2)/4;svg+='<line class="chart-grid" x1="'+pad+'" y1="'+yy+'" x2="'+(w-pad)+'" y2="'+yy+'"></line>'}buckets.forEach((b,i)=>{svg+='<text class="chart-label" x="'+x(i)+'" y="'+(h-8)+'" text-anchor="middle">'+esc(b)+'</text>'});types.forEach(t=>{const actual=buckets.map((b,i)=>x(i)+','+y((by[t+'|'+b]||{}).actual||0)).join(' '),global=buckets.map((b,i)=>x(i)+','+y((by[t+'|'+b]||{}).global||0)).join(' '),c=colors[t]||'#555';svg+='<polyline points="'+actual+'" fill="none" stroke="'+c+'" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></polyline><polyline points="'+global+'" fill="none" stroke="'+c+'" stroke-width="2" stroke-dasharray="5 5" stroke-linecap="round" stroke-linejoin="round" opacity=".72"></polyline>'});svg+='</svg>';const legend=types.map(t=>'<span class="legend-line">'+esc(queueTypeName(t))+' 实际</span><span class="legend-line global">'+esc(queueTypeName(t))+' 全局</span>').join('');box.innerHTML=svg+'<div class="chart-legend">'+legend+'</div>'}
function queueTypeName(t){return t==='weekday'?'工作日':t==='weekend'?'周末':t==='holiday'?'节假日':t}
function renderQueueTable(series){const c=el('qtTable');if(!series.length){c.innerHTML='';return}const rows=series.map(x=>'<tr><td>'+esc(x.bucket)+'</td><td>'+esc(x.date_type_name||x.date_type)+'</td><td>'+esc(x.store_name||x.store_id)+'</td><td>'+esc(x.actual_passed||0)+'<br><span class="mu">'+esc(x.actual_samples||0)+' 样本</span></td><td>'+esc(x.global_passed||0)+'<br><span class="mu">'+esc(x.global_samples||0)+' 快照段</span></td><td>'+(x.wait_p50_minutes==null?'-':Math.round(x.wait_p50_minutes)+' 分')+'</td><td>'+Math.round((x.missed_rate||0)*100)+'%</td><td>'+esc(confText(x.confidence))+'</td></tr>').join('');c.innerHTML='<table class="tbl"><thead><tr><th>时段</th><th>日期类型</th><th>门店</th><th>实际过号</th><th>全局过号</th><th>P50等待</th><th>过号率</th><th>可信度</th></tr></thead><tbody>'+rows+'</tbody></table>'}
function confText(v){return v==='high'?'高':v==='medium'?'中':v==='low'?'低':'无'}

async function lSm(){await ensureStores();await loadSampling()}
async function loadSampling(){try{const d=await(await fetch('/api/sampling')).json();spCfg=d.config||{};spState=d.state||{};spAutoStart=d.autostart||{};spQueueState=d.queue_state||{};fillSamplingForm();renderSamplingStores();renderSamplingState();uSamplingSummary()}catch(e){el('sampleState').innerHTML='<div class="ci bad">信息收集状态加载失败</div>'}}
function fillSamplingForm(){el('spEnabled').checked=!!spCfg.enabled;el('spAuto').checked=!!spCfg.auto_start;el('spInterval').value=spCfg.interval_seconds||300;el('spStart').value=timeInputValue(spCfg.active_start||'100000');el('spEnd').value=timeInputValue(spCfg.active_end||'220000')}
function renderSamplingStores(){const c=el('samplingStores'),h=el('sampleStoreHint');if(!c)return;if(!stores.length){c.innerHTML='<span class="mu">先完成认证获取</span>';if(h)h.textContent='';return}const chosen=(spCfg.store_ids||[]).map(String);c.innerHTML=stores.map(s=>'<button class="chip '+(chosen.includes(String(s.id))?'on':'')+'" data-store="'+escA(String(s.id))+'">'+esc(s.nickname||s.name||s.id)+'</button>').join('');c.querySelectorAll('.chip').forEach(b=>b.onclick=()=>{b.classList.toggle('on');renderSamplingStoreHint()});renderSamplingStoreHint()}
function renderSamplingStoreHint(){const h=el('sampleStoreHint');if(!h)return;const chosen=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);if(chosen.length){h.textContent='当前收集 '+chosen.length+' 家指定门店。';return}const pref=(pr.selected_stores||[]).map(storeName).filter(Boolean);h.textContent=pref.length?'当前跟随抢号门店：'+pref.join('、'):'当前跟随认证里保存的门店。'}
function samplingPayload(){const ids=Array.from(document.querySelectorAll('#samplingStores .chip.on')).map(x=>x.dataset.store);return{enabled:el('spEnabled').checked,auto_start:el('spAuto').checked,interval_seconds:+el('spInterval').value||300,active_start:compactTime(el('spStart').value||'10:00'),active_end:compactTime(el('spEnd').value||'22:00'),store_ids:ids,use_preference_stores:ids.length===0}}
function renderSamplingState(){const s=spState||{},a=spAutoStart||{},q=spQueueState||{},next=s.next_run_at?new Date(s.next_run_at).toLocaleString():'-',last=s.last_run_at?new Date(s.last_run_at).toLocaleString():'-',msg=s.last_error||s.message||q.message||'无',bad=(s.last_error||q.needs_auth)&&!/跳过|时间窗|暂无|正在运行/.test(s.last_error||'');el('sampleState').innerHTML=chip('状态',s.running?'运行中':(s.enabled?'已启用':'未启动'),s.running?'ok':s.enabled?'warn':'')+chip('开机自启动',a.enabled?'已配置':a.supported?'未配置':'不支持',a.enabled?'ok':'warn')+chip('下次',next,'ok')+chip('上次',last,'ok')+chip('样本',s.snapshots||0,'ok')+chip('门店失败',s.store_errors||0,(s.store_errors||0)?'warn':'ok')+chip('认证',q.auth_ok?'可用':'需更新',q.auth_ok?'ok':'bad')+chip('最近结果',msg,bad?'bad':'ok')}
async function saveSampling(quiet){spCfg=samplingPayload();try{const d=await(await fetch('/api/sampling',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(spCfg)})).json();if(d.error){if(!quiet)alert(d.error);return false}spCfg=d.config||spCfg;spState=d.state||spState;renderSamplingStores();renderSamplingState();uSamplingSummary();if(!quiet)alert(spState.running?'信息收集配置已保存，后台已按新配置重启':'信息收集配置已保存');return true}catch(e){if(!quiet)alert('保存失败');return false}}
async function startSampling(){if(!await saveSampling(true))return;try{const d=await(await fetch('/api/sampling/start',{method:'POST'})).json();if(d.error){alert(d.error);return}spState=d.state||spState;await loadSampling();uSamplingSummary()}catch(e){alert('启动失败')}}
async function stopSampling(){try{const d=await(await fetch('/api/sampling/stop',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();uSamplingSummary()}catch(e){alert('停止失败')}}
async function runSampleOnce(){if(!await saveSampling(true))return;const box=el('sampleResult');box.classList.remove('hid');box.textContent='收集中';try{const d=await(await fetch('/api/sampling/once',{method:'POST'})).json();spState=d.state||spState;renderSamplingState();uSamplingSummary();const r=d.result||{};box.innerHTML=r.skipped?'本轮跳过：'+esc(r.skip_reason):'<b>收集完成</b><br>'+esc((r.stores||[]).map(x=>{const parts=[];parts.push(x.error||((x.slots||0)+' 条时段'));if(x.queue_observed)parts.push('排队 '+(x.queue_wait_groups||0)+' 组');else if(x.queue_error)parts.push('排队失败');return(x.store_name||x.store_id)+': '+parts.join('，')}).join('\\n')).replaceAll('\\n','<br>')}catch(e){box.innerHTML='收集失败'}}
function usePrefSamplingStores(){document.querySelectorAll('#samplingStores .chip').forEach(x=>x.classList.remove('on'));renderSamplingStoreHint()}
async function setBootSampling(enabled){try{const d=await(await fetch('/api/sampling/autostart',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled})})).json();if(d.error){alert(d.error);return}spAutoStart=d.autostart||{};if(cp==='sm')await loadSampling();if(cp==='qt')await loadQueueTrends();alert(enabled?'已配置开机自启动':'已取消开机自启动')}catch(e){alert('操作失败')}}

async function lSn(){await ensureStores();if(!el('snRows').children.length)addSn();await loadSnPlan()}
async function ensureStores(){if(stores.length)return;try{stores=await(await fetch('/api/stores')).json();selStores=stores.map(s=>String(s.id));}catch(e){}}
function storeOpts(v){return stores.map(s=>'<option value="'+escA(String(s.id))+'" '+(String(s.id)===String(v)?'selected':'')+'>'+esc(s.nickname||s.name||s.id)+'</option>').join('')}
function localDateInput(d){const p=n=>String(n).padStart(2,'0');return d.getFullYear()+'-'+p(d.getMonth()+1)+'-'+p(d.getDate())}
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
async function saveSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){alert('请至少添加一个有效目标');return}try{const d=await(await fetch('/api/sniper/plan',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){alert(d.error);return}renderSnPlan(d.plan);alert('已保存狙击计划')}catch(e){alert('保存失败')}}
async function loadSnPlan(){try{const d=await(await fetch('/api/sniper/plan')).json();if(d.targets?.length){el('snRows').innerHTML='';d.targets.forEach(addSn)}renderSnPlan(d)}catch(e){}}
function renderSnPlan(p){const c=el('snPlan'),ts=p?.targets||[];if(!ts.length){c.innerHTML='<div class="empty">暂无计划</div>';return}c.innerHTML='<table class="tbl"><thead><tr><th>目标</th><th>开放窗口</th><th>状态</th><th>尝试</th><th>最后错误</th></tr></thead><tbody>'+ts.map(t=>'<tr><td>'+esc(t.store_id)+'<br>'+esc(t.date)+' '+esc(fT(t.start_after))+'-'+esc(fT(t.start_before))+'</td><td>'+esc(t.open_at?new Date(t.open_at).toLocaleString():'-')+'<br>'+(t.countdown_seconds>0?Math.ceil(t.countdown_seconds/60)+' 分钟后':'窗口内/已结束')+'</td><td>'+esc(t.status||'-')+'</td><td>'+esc(t.attempts||0)+'</td><td>'+esc(t.last_error||'')+'</td></tr>').join('')+'</tbody></table>'}
async function startSn(){const read=readSnTargets();if(!read.ok)return;if(!read.targets.length){alert('请至少添加一个有效目标');return}try{const d=await(await fetch('/api/sniper/start',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({targets:read.targets})})).json();if(d.error){alert(d.error);return}await loadStatus();await loadSnPlan();alert('狙击计划已启动')}catch(e){alert('启动失败')}}

async function lR(){const c=el('rc');c.innerHTML='<div class="empty">加载中…</div>';try{const d=await safeFetch('/api/reservations');if(d.error){c.innerHTML=loadErrBoxHTML(d.error,'lR()','我的预约');return}if(!d.length){c.innerHTML='<div class="empty">暂无预约</div>';return}c.innerHTML='<div class="sg">'+d.map(r=>{const when=r.slot_label||[r.queueDate,fT(r.start),r.end?'-'+fT(r.end):''].filter(Boolean).join(' '),store=r.store_name||r.monitored_store_id||r.storeId||'';return'<div class="sl av"><div class="tm">'+esc(r.number||'-')+'</div><div class="ss">'+esc(r.status||'-')+(store?' · '+esc(store):'')+'</div><div class="mu mt8">'+esc(when||'时间待确认')+'<br>#'+esc(r.ticketId||'')+'</div></div>'}).join('')+'</div>'}catch(e){c.innerHTML=loadErrBoxHTML(e,'lR()','我的预约')}}

async function lP(){try{pr=await(await fetch('/api/preferences')).json();fF(pr);dP(pr);renderBookingStores();uD()}catch(e){}}
function fF(p){el('pa').value=p.adult||2;el('pc').value=p.child||0;el('pt').value=p.table_type||'T';el('ppm').value=p.day_priority_mode||'date';el('pst').value=p.slot_strategy||'earliest';el('ptm').value=p.target_time||'1930';rT('wd',p.weekday_slots||[]);rT('sa',p.saturday_slots||[]);rT('su',p.sunday_slots||[])}
function rangeText(rs){return !rs||!rs.length?'不预约':rs.map(r=>fT(String(r.start||''))+'-'+fT(String(r.end||''))).join('、')}
function priText(v){return v==='weekend_first'?'周末优先':v==='weekday_first'?'工作日优先':'按日期优先'}
function stratText(v,t){return v==='latest'?'最晚可约':v==='closest'?'接近 '+fT(t||'1930'):'最早可约'}
function dP(p){const people=(p.adult||2)+' 成人'+((p.child||0)>0?' · '+p.child+' 儿童':'');const table=(p.table_type||'T')==='C'?'吧台':'桌位',pri=priText(p.day_priority_mode),str=stratText(p.slot_strategy,p.target_time);el('sumPeople').textContent=people;el('sumTable').textContent=table;el('sumSlot').textContent=pri+' · '+str;el('ps').innerHTML='<b>'+esc(people)+'</b> · '+esc(table)+'<span class="line">优先级：'+esc(pri)+' · '+esc(str)+'</span><span class="line">工作日：'+esc(rangeText(p.weekday_slots))+'</span><span class="line">周六：'+esc(rangeText(p.saturday_slots))+'</span><span class="line">周日：'+esc(rangeText(p.sunday_slots))+'</span>'}
function storeName(id){const s=stores.find(x=>String(x.id)===String(id));return s?(s.nickname||s.name||s.id):id}
function orderedStoreIDs(){const all=stores.map(s=>String(s.id)),sel=(pr.selected_stores||[]).map(String).filter(id=>all.includes(id)),base=(pr.store_priority||[]).map(String).filter(id=>all.includes(id));let order=[];base.forEach(id=>{if(!order.includes(id))order.push(id)});sel.forEach(id=>{if(!order.includes(id))order.push(id)});all.forEach(id=>{if(!order.includes(id))order.push(id)});return{all,selected:sel.length?sel:all,order}}
function renderBookingStores(){const box=el('bookingStores');if(!box)return;if(!stores.length){box.innerHTML='<span class="mu">完成认证后可选择门店</span>';return}const data=orderedStoreIDs(),set=new Set(data.selected);box.innerHTML=data.order.map(id=>'<div class="store-row" data-store="'+escA(id)+'"><input type="checkbox" '+(set.has(id)?'checked':'')+'><div><b>'+esc(storeName(id))+'</b><span>'+esc(id)+'</span></div><button type="button" class="ico" onclick="moveStoreRow(this,-1)">↑</button><button type="button" class="ico" onclick="moveStoreRow(this,1)">↓</button></div>').join('')}
function moveStoreRow(btn,dir){const r=btn.closest('.store-row'),p=r.parentElement;if(dir<0&&r.previousElementSibling)p.insertBefore(r,r.previousElementSibling);if(dir>0&&r.nextElementSibling)p.insertBefore(r.nextElementSibling,r)}
function bookingStoresFromUI(){const rows=Array.from(document.querySelectorAll('#bookingStores .store-row')),selected=[];rows.forEach(r=>{if(r.querySelector('input').checked)selected.push(r.dataset.store)});return{selected_stores:selected,store_priority:selected}}
function applyPreset(k){const set=(pm,st,tm,wd,sa,su)=>{el('ppm').value=pm;el('pst').value=st;el('ptm').value=tm;rT('wd',wd);rT('sa',sa);rT('su',su)};if(k==='weekday_dinner')set('weekday_first','closest','1930',[{start:'1900',end:'2030'}],[],[]);else if(k==='weekend_lunch')set('weekend_first','earliest','1130',[],[{start:'1030',end:'1300'}],[{start:'1030',end:'1300'}]);else if(k==='weekend_dinner')set('weekend_first','closest','1930',[],[{start:'1830',end:'2030'}],[{start:'1830',end:'2030'}]);else if(k==='any_available')set('date','earliest','1930',[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}],[{start:'1000',end:'2200'}]);alert('已套用策略模板，请点击保存偏好')}
function rT(k,rs){const c=el(k);c.innerHTML='';(rs||[]).forEach(r=>{const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" value="'+escA(r.start||'')+'" placeholder="1930"><span class="sp">至</span><input type="text" value="'+escA(r.end||'')+'" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)});if(!rs||!rs.length)c.innerHTML='<span class="mu">不预约</span>'}
function aT(k){const c=el(k);if(c.querySelector('.mu'))c.innerHTML='';const d=document.createElement('div');d.className='tr';d.innerHTML='<input type="text" placeholder="1930"><span class="sp">至</span><input type="text" placeholder="2030"><span class="x" onclick="this.parentElement.remove()">×</span>';c.appendChild(d)}
function gT(k){const ip=document.querySelectorAll('#'+k+' input'),r=[];for(let i=0;i<ip.length;i+=2){const s=ip[i].value.trim(),e=ip[i+1]?ip[i+1].value.trim():'';if(s||e)r.push({start:s,end:e})}return r}
function prefsPayload(){const st=bookingStoresFromUI();return{adult:+el('pa').value||2,child:+el('pc').value||0,table_type:el('pt').value||'T',selected_stores:st.selected_stores,store_priority:st.store_priority,day_priority_mode:el('ppm').value||'date',day_priority:pr.day_priority||['saturday','sunday','weekday'],slot_strategy:el('pst').value||'earliest',target_time:el('ptm').value.trim()||'1930',weekday_slots:gT('wd'),saturday_slots:gT('sa'),sunday_slots:gT('su')}}
async function savePrefsPayload(b,quiet){try{const d=await(await fetch('/api/preferences',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)alert(d.error);return false}pr=d.preferences||b;fF(pr);dP(pr);renderBookingStores();uD();if(!quiet)alert('已保存');return true}catch(e){if(!quiet)alert('保存失败');return false}}
async function sP(){const b=prefsPayload();if(stores.length&&!b.selected_stores.length){alert('请至少选择一家抢号门店');return false}return savePrefsPayload(b,false)}
async function saveCalendarStoresAsPrefs(){if(!selStores.length){alert('请先选择门店');return}await lP();const b={...pr,selected_stores:selStores.slice(),store_priority:selStores.slice()};if(await savePrefsPayload(b,true))alert('已保存为抢号门店优先级')}
async function lS(){await lP();await ensureStores();renderBookingStores();try{const c=await(await fetch('/api/config')).json();el('nf').value=c.feishu?.webhook||'';el('ntt').value=c.telegram?.token||'';el('ntc').value=c.telegram?.chat_id||'';el('nbu').value=c.bark?.url||'';el('nbk').value=c.bark?.key||'';el('ns').value=c.server_chan?.key||''}catch(e){}lD()}
async function sN(quiet){const b={feishu:{webhook:el('nf').value.trim()},telegram:{token:el('ntt').value.trim(),chat_id:el('ntc').value.trim()},bark:{url:el('nbu').value.trim(),key:el('nbk').value.trim()},server_chan:{key:el('ns').value.trim()}};try{const d=await(await fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)})).json();if(d.error){if(!quiet)alert(d.error);return false}if(!quiet)alert('已保存');return true}catch(e){if(!quiet)alert('保存失败');return false}}
async function tN(ch){if(!await sN(true))return;try{const r=await fetch('/api/notifications/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({channel:ch||'all'})}),d=await r.json();if(d.error){alert(d.error);return}const bad=(d.results||[]).filter(x=>!x.ok).map(x=>x.channel+': '+x.error);alert(bad.length?'已先保存当前表单，部分发送失败：\n'+bad.join('\n'):'已先保存当前表单，测试通知已发送')}catch(e){alert('发送失败')}}
function chip(t,s,c){return'<div class="ci '+c+'">'+esc(t)+'：'+esc(s)+'</div>'}
function diagDetail(d){const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},net=d.network||{},logs=(d.engine_log_tail||[]).concat((d.log_tail||[]).map(x=>({time:'',message:x}))),ports=d.ports||[],isWin=(d.platform||{}).goos==='windows';const badPorts=ports.filter(p=>!p.available&&!p.current&&!p.fallback_port).map(p=>p.name+': '+(p.error||'占用')),portNotes=ports.filter(p=>p.note).map(p=>p.name+': '+p.note),chainLines=(chain.probes||[]).map(p=>p.name+': '+(p.ok?'正常':p.skipped?'跳过':'异常')+(p.detail?'（'+p.detail+'）':''));let html='<b>下一步建议</b><br>';if(!cfg.complete)html+='先重新获取认证参数。<br>';if(isWin&&cert.cert_exists&&!cert.current_user_trusted&&!cert.local_machine_trusted)html+='证书已生成但未信任，请重新获取认证并允许管理员权限安装证书。<br>';if(isWin&&cert.current_user_trusted&&!cert.local_machine_trusted)html+='Windows 机器级证书未信任，PC 微信可能拒绝访问；请重新获取认证并允许管理员权限。<br>';if(isWin&&!cert.current_user_trusted&&cert.local_machine_trusted)html+='Windows 当前用户证书未信任，请重新获取认证补齐证书信任。<br>';if(!isWin&&cert.cert_exists&&!cert.trusted)html+='证书已生成但未信任，请重新获取认证触发安装。<br>';if(chain.checked&&!chain.ok)html+='代理链路自检失败，请保留本页信息发给开发者。<br>';if(pm.stale)html+='发现代理残留，请先点“修复代理”。<br>';if(!net.reachable)html+='寿司郎网络不可达，先确认网络或稍后重试。<br>';html+='<br><b>证书</b>：<code>'+esc(cert.cert_path||'-')+'</code>'+(cert.trust_error?'<br>'+esc(cert.trust_error):'')+(isWin&&(cert.current_user_trusted||cert.local_machine_trusted)?'<br>CurrentUser='+esc(String(!!cert.current_user_trusted))+'；LocalMachine='+esc(String(!!cert.local_machine_trusted))+'；Disallowed='+esc(String(!!cert.disallowed)):'');if(badPorts.length||portNotes.length)html+='<br><b>端口</b>：'+esc(badPorts.concat(portNotes).join('；'));if((sp.summary||[]).length)html+='<br><b>系统代理</b>：'+esc(sp.summary.join('；'));html+='<br><b>代理链路</b>：'+esc(chain.summary||'未检查')+(chainLines.length?'<br>'+esc(chainLines.join('；')):'');if(logs.length)html+='<br><b>最近日志</b><br>'+logs.slice(-8).map(l=>esc((l.time||'')+' '+(l.message||''))).join('<br>');return html}
async function lD(){const box=el('dg'),detail=el('ddetail');if(!box)return;box.innerHTML='<div class="ci">诊断中…</div>';if(detail)detail.classList.add('hid');try{const d=await safeFetch('/api/diagnostics',null,20000);lastDiag=d;const cfg=d.config||{},cert=d.certificate||{},pm=d.proxy_marker||{},sp=d.system_proxy||{},chain=d.proxy_chain||{},eng=d.engine||{},net=d.network||{},dp=d.ports||[],isWin=(d.platform||{}).goos==='windows';const miss=(cfg.missing||[]).join('、'),portIssues=dp.filter(p=>p.in_use&&!p.current&&!p.fallback_port).map(p=>p.name),portNotes=dp.filter(p=>p.note).map(p=>p.note),portText=portIssues.length?portIssues.join('、'):(portNotes.length?portNotes.join('、'):'默认端口可用'),certText=isWin?(cert.local_machine_trusted?'机器级已信任':cert.current_user_trusted?'用户级已信任':(cert.cert_exists?'未信任':'未生成')):(cert.trusted?'已信任':cert.cert_exists?'未信任':'未生成'),certClass=isWin?(cert.local_machine_trusted?'ok':cert.current_user_trusted?'warn':'bad'):(cert.trusted?'ok':'bad');const items=[];items.push(chip('认证参数',cfg.complete?'完整':(miss||'未捕获'),cfg.complete?'ok':'bad'));items.push(chip('门店',cfg.store_count?cfg.store_count+' 个':'未选择',cfg.store_count?'ok':'bad'));items.push(chip('证书',certText,certClass));items.push(chip('端口',portText,portIssues.length?'bad':portNotes.length?'warn':'ok'));items.push(chip('代理残留',pm.stale?'发现残留':pm.active?'运行中':'未发现',pm.stale?'bad':pm.active?'warn':'ok'));items.push(chip('系统代理',sp.available?'可读取':'不可读取',sp.available?'ok':'warn'));items.push(chip('代理链路',chain.checked?(chain.ok?'正常':'异常'):'未运行',chain.checked?(chain.ok?'ok':'bad'):'warn'));items.push(chip('网络',net.reachable?'寿司郎可达':'不可达',net.reachable?'ok':'bad'));items.push(chip('通知',cfg.notification_channels?.length?cfg.notification_channels.join('、'):'未配置',cfg.notification_channels?.length?'ok':'warn'));items.push(chip('引擎',eng.status||'idle',eng.status==='error'?'bad':(eng.status==='booking'||eng.status==='capturing'||eng.status==='sniping')?'warn':'ok'));box.innerHTML=items.join('');if(detail){detail.innerHTML=diagDetail(d);detail.classList.remove('hid')}}catch(e){box.innerHTML=loadErrBoxHTML(e,'lD()','诊断')}}
async function copyDiag(){if(!lastDiag)await lD();if(!lastDiag){alert('暂无诊断信息');return}const text=JSON.stringify(lastDiag,null,2);try{if(navigator.clipboard&&navigator.clipboard.writeText)await navigator.clipboard.writeText(text);else{const t=document.createElement('textarea');t.value=text;t.style.position='fixed';t.style.left='-9999px';document.body.appendChild(t);t.select();document.execCommand('copy');t.remove()}alert('已复制诊断信息')}catch(e){alert('复制失败，请手动选择诊断详情')}}
function authProbeHTML(d){const rs=d.results||[],ad=d.advice||[];let html='<b>基础接口自检</b>：'+(d.ok?'通过':'失败')+(d.store_id?'<br><b>门店</b>：'+esc(d.store||d.store_id)+' <code>'+esc(d.store_id)+'</code>':'');if(rs.length)html+='<br>'+rs.map(r=>esc(r.name||'-')+'：'+(r.ok?'正常':r.skipped?'跳过':'异常')+(r.status?' HTTP '+r.status:'')+(r.latency_ms?' '+r.latency_ms+'ms':'')+(r.detail?'（'+esc(r.detail)+'）':'')).join('<br>');if(ad.length)html+='<br><b>下一步</b><br>'+ad.map(esc).join('<br>');return html}
async function testAuthProbe(){const detail=el('ddetail');if(detail){detail.classList.remove('hid');detail.innerHTML='基础接口测试中...'}try{const r=await fetch('/api/auth/probe',{method:'POST'}),d=await r.json();if(detail)detail.innerHTML=authProbeHTML(d);if(!d.ok)alert('基础接口未通过，详情已显示在诊断区')}catch(e){if(detail)detail.innerHTML='基础接口测试失败：'+esc(String(e));alert('基础接口测试失败')}}
async function repairP(){try{const d=await(await fetch('/api/repair-proxy',{method:'POST'})).json();alert(d.ok?'代理已恢复':'修复失败，请看 doctor');lD()}catch(e){alert('修复失败')}}
async function stopProcesses(){if(!confirm('将恢复代理、停止后台抢号/信息收集，并退出当前应用窗口。之后就可以删除 exe 或安装目录。继续？'))return;try{const r=await fetch('/api/processes/stop',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({include_self:true})}),d=await r.json();alert(d.ok?'已发送停止请求，当前应用即将退出':'部分进程未停止，请稍后再试或重启电脑')}catch(e){alert('已发送停止请求，当前应用即将退出')}}
async function uninstallAll(){if(!confirm('将恢复代理、移除证书并清理本地敏感数据。继续？'))return;try{const d=await(await fetch('/api/uninstall',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({all:true,certificates:true,system_cert:true})})).json();alert(d.ok?'已清理':'部分清理失败，请看 doctor');lD()}catch(e){alert('清理失败')}}

async function lL(){try{const ls=await(await fetch('/api/engine/logs')).json(),v=el('lv');v.innerHTML=(ls||[]).map(l=>'<div class="ll '+(l.level==='error'?'er':'')+'"><span class="lt">'+esc(l.time)+'</span><span class="lm">'+esc(l.message)+'</span></div>').join('');v.scrollTop=v.scrollHeight}catch(e){}}
function aL(e){const v=el('lv');if(!v)return;const d=document.createElement('div');d.className='ll '+(e.level==='error'?'er':'');d.innerHTML='<span class="lt">'+esc(e.time)+'</span><span class="lm">'+esc(e.message)+'</span>';v.appendChild(d);if(cp==='lo')v.scrollTop=v.scrollHeight}
function sse(){if(cE)cE.close();const s=new EventSource('/api/events');cE=s;s.onopen=()=>{loadStatus()};s.addEventListener('engine',e=>{try{es=JSON.parse(e.data);uE();uD();if(cp==='sn')loadSnPlan();if(['idle','success','error'].includes(es.status))loadStatus()}catch(x){}});s.addEventListener('sampling',e=>{try{spState=JSON.parse(e.data);uSamplingSummary();if(cp==='sm')renderSamplingState()}catch(x){}});s.addEventListener('log',e=>{try{aL(JSON.parse(e.data))}catch(x){}});s.addEventListener('calendar',e=>{try{const d=JSON.parse(e.data);if(cp==='ca'){as=[];(d.stores||[]).forEach(st=>(st.slots||[]).forEach(x=>as.push({...x,store_name:st.store_name,store_id:st.store_id})));if(as.length)rDB()}}catch(x){}});s.addEventListener('ping',()=>{});s.onerror=()=>{s.close();cE=null;setTimeout(sse,3000)}}
init();
</script>
</body></html>
`
