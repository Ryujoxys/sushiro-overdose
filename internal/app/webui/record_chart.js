'use strict';

function recordTimeRange(time) {
  return time+'–'+time.slice(0,3)+(time.endsWith('00')?'29':'59');
}
function calledPointLabel(point,singleDate=false) {
  if(!point)return '暂无叫号记录';
  return (singleDate?'当日叫到':point.days>=3?'通常叫到':'记录叫到')+(singleDate?' ':'约 ')+Math.round(point.median)+' 号';
}
const recordMinute=time=>Number(time.slice(0,2))*60+Number(time.slice(3));
function recordClock(minutes) { return String(Math.floor(minutes/60)).padStart(2,'0')+':'+String(minutes%60).padStart(2,'0'); }
function chartSegments(points) {
  const segments=[];
  for(const point of points){
    const last=segments.at(-1);
    if(!last||recordMinute(point.time)-recordMinute(last.at(-1).time)>30)segments.push([point]);else last.push(point);
  }
  return segments;
}
// Official waits move in 5-minute steps, so words round to 5 and long waits read in hours.
function waitWords(value,approx=true) {
  const r=Math.round(value/5)*5;
  if(r<5)return approx?'基本不用等':'不用等';
  return (approx?'约 ':'')+(r<60?r+' 分钟':Math.floor(r/60)+' 小时'+(r%60?' '+r%60+' 分':''));
}
// Round ticks: at most five steps, top tick at or above the largest value drawn.
function recordAxisTicks(rawMax,called) {
  const steps=called?[50,100,200,250,500,1000,2000,2500,5000]:[10,15,30,60,120];
  const step=steps.find(s=>Math.ceil(rawMax/s)<=5)||steps.at(-1);
  const top=Math.max(step,Math.ceil(rawMax/step)*step);
  return Array.from({length:top/step+1},(_,i)=>i*step);
}
// Whole-hour labels at least 80px apart; never forced at the ends, so they cannot collide.
function recordHourTicks(lo,hi,plotWidth) {
  const perHour=plotWidth/Math.max(.5,(hi-lo)/60),every=[1,2,3,4,6].find(h=>h*perHour>=80)||6,ticks=[];
  for(let m=Math.ceil(lo/60)*60;m<=hi;m+=60)if(m/60%every===0)ticks.push(m);
  return ticks.length?ticks:[lo];
}
// A slot is "thin" when it has under 3 days, or under half the days of the best-covered slot.
function recordThinDays(points,singleDate) {
  return singleDate?0:Math.max(3,Math.ceil(Math.max(0,...points.map(point=>point.days))/2));
}
function recordCurrentSlot() { const now=chinaHHMM();return now.slice(0,3)+(now.slice(3)<'30'?'00':'30'); }
// One plain sentence about when the store is busy, from the multi-day wait curve only.
// Slots under 3 days are ignored, gaps are never bridged, and quiet runs need well-covered slots.
function recordSummary(data) {
  const points=data.points||[];
  if(!points.length)return (data.called_points||[]).length?{text:'这段时间没有等待记录。',note:''}:null;
  const ok=points.filter(point=>point.days>=3);
  if(ok.length<3)return {text:'等待记录还少（'+(data.days||0)+' 天），还看不出哪段时间挤。',note:''};
  const thin=recordThinDays(ok,false),peak=ok.reduce((best,point)=>point.median>best.median?point:best);
  const label=run=>run[0].time+'–'+recordClock(recordMinute(run.at(-1).time)+30),span=run=>escapeHTML(label(run));
  const waitDays=new Map(points.map(point=>[point.time,point.days])),calls=data.called_points||[],callThin=recordThinDays(calls,false);
  const blind=chartSegments(calls.filter(c=>c.days>=callThin&&(waitDays.get(c.time)||0)<3&&c.time>ok[0].time&&c.time<ok.at(-1).time));
  const prefix=blind.length?'有等待记录的时段里，':'',blindNote=blind.length?blind.map(label).join('、')+' 多数日子没有等待记录，没算进来，可能更挤，可以看叫号曲线。':'';
  if(peak.median<15)return {text:(prefix||'全天')+(peak.median<5?'通常不用等。':'通常不用久等，最多'+waitWords(peak.median)+'。'),note:blindNote};
  const segments=chartSegments(ok),segment=segments.find(s=>s.includes(peak));
  let i=segment.indexOf(peak),j=i;
  while(i>0&&segment[i-1].median>=peak.median*.8)i--;
  while(j<segment.length-1&&segment[j+1].median>=peak.median*.8)j++;
  const busy=segment.slice(i,j+1);
  let quiet=null;
  for(const s of segments){
    let run=[];
    for(const point of [...s,null]){
      if(point&&point.median<=10&&point.days>=thin&&!busy.includes(point)){run.push(point);continue;}
      if(run.length>=2&&run[0]!==ok[0]&&run.at(-1)!==ok.at(-1)&&(!quiet||run.length>quiet.length))quiet=run;
      run=[];
    }
  }
  const low=Math.min(...busy.map(p=>p.median)),high=Math.max(...busy.map(p=>p.median)),a=Math.round(low/5)*5,b=Math.round(high/5)*5;
  const wait=a===b?waitWords(low):low>=60?' '+Math.floor(low/60)+' 小时以上':' '+a+'–'+b+' 分钟';
  let text=prefix+'<b>'+span(busy)+'</b> 最挤，通常要等'+wait;
  if(quiet)text+='；<b>'+span(quiet)+'</b> 通常'+(Math.max(...quiet.map(p=>p.median))<5?'不用等':'不到 10 分钟');
  const fewest=Math.min(...busy.map(p=>p.days));
  return {text:text+'。',note:(fewest<thin?'最挤的时段只有 '+fewest+' 天有等待记录，仅供参考。':'')+blindNote};
}
function recordChartPoints() {
  return state.analysisMetric==='wait'?(state.records?.points||[]):(state.records?.called_points||[]);
}
// Only overlay today when the main curve spans several days; a single-day curve is already "today".
function recordTodayPoints(called=state.analysisMetric!=='wait') {
  const today=state.recordsToday,data=state.records;
  if(!today||el('analysis-date').value||(called?data?.called_days:data?.days)<2)return [];
  return (called?today.called_points:today.points)||[];
}
// One value format for the readout, the chart label and the point's accessible name.
function recordValueText(called,point,singleDate) {
  return called?(singleDate?'':'约 ')+Math.round(point.median)+' 号':waitWords(point.median,!singleDate);
}
function recordReadoutItem(called,point,today,singleDate,thin) {
  const label=called?(!point?'叫号':singleDate?'当日叫到':point.days>=3?'通常叫到':'记录叫到'):(singleDate?'当日要等':'通常要等');
  const todayText=today?'今天 '+Math.round(today.median)+(called?' 号':' 分钟'):'';
  if(!point)return '<div class="readout-item none"><span class="readout-label">'+label+'</span><strong>暂无记录</strong><small>这个时段没记录，不做估算</small>'+(todayText?'<small class="readout-today">'+escapeHTML(todayText)+'</small>':'')+'</div>';
  const value=recordValueText(called,point,singleDate);
  const details=[];
  if(!singleDate){
    if(called&&point.days>=3&&Math.floor(point.lower)<Math.ceil(point.upper))details.push('常见 '+Math.floor(point.lower)+'–'+Math.ceil(point.upper)+' 号');
    if(!called&&point.days>=3&&point.upper>=point.median+5)details.push('八成日子不超过 '+waitWords(point.upper,false));
    details.push(point.days<3?'只有 '+point.days+' 天记录，仅供参考':point.days+' 天记录'+(point.days<thin?'（较少），仅供参考':''));
  }
  let delta='';
  if(today){
    const diff=Math.round(today.median)-(called?Math.round(point.median):Math.round(point.median/5)*5);
    delta=called?(diff?'（比平时'+(diff>0?'多 ':'少 ')+Math.abs(diff)+' 号）':'（和平时一样）'):(Math.abs(diff)<5?'（和平时差不多）':'（比平时'+(diff>0?'多 ':'少 ')+Math.abs(diff)+' 分钟）');
  }
  return '<div class="readout-item"><span class="readout-label">'+label+'</span><strong>'+escapeHTML(value)+'</strong>'+(details.length?'<small>'+escapeHTML(details.join(' · '))+'</small>':'')+(todayText?'<small class="readout-today">'+escapeHTML(todayText+delta)+'</small>':'')+'</div>';
}
function recordAnnotationWidth(text) {
  let width=0;
  try{width=text.getComputedTextLength();}catch{}
  return (width||[...text.textContent].reduce((sum,c)=>sum+(/[　-鿿＀-￯]/.test(c)?13:7.5),0))+24;
}
function updateRecordInspection(time,source='') {
  const data=state.records,points=recordChartPoints(),index=points.findIndex(point=>point.time===time);
  if(index<0)return;
  state.analysisTime=time;
  const singleDate=!!el('analysis-date').value,wanted=point=>point.time===time;
  const called=(data.called_points||[]).find(wanted),wait=(data.points||[]).find(wanted);
  const todayCalled=recordTodayPoints(true).find(wanted),todayWait=recordTodayPoints(false).find(wanted);
  const now=!singleDate&&time===recordCurrentSlot();
  el('analysis-readout').setAttribute('aria-live',source==='pointer'?'off':'polite');
  el('analysis-readout').innerHTML='<div class="readout-head"><span class="readout-time">'+escapeHTML(recordTimeRange(time))+'</span>'+(now?'<span class="badge">现在</span>':'')+'</div>'+recordReadoutItem(true,called,todayCalled,singleDate,recordThinDays(data.called_points||[],singleDate))+recordReadoutItem(false,wait,todayWait,singleDate,recordThinDays(data.points||[],singleDate));
  const chart=el('record-chart'),svg=chart.querySelector('svg'),marker=chart.querySelector('[data-record-point="'+time+'"]');
  if(svg&&marker){
    const x=marker.dataset.x,y=marker.dataset.y,ty=marker.dataset.ty;
    const cross=svg.querySelector('.chart-crosshair'),dot=svg.querySelector('.chart-selected'),todayDot=svg.querySelector('.chart-selected-today'),tip=svg.querySelector('.chart-annotation');
    if(cross){cross.setAttribute('x1',x);cross.setAttribute('x2',x);}
    if(dot){dot.setAttribute('cx',x);dot.setAttribute('cy',y);}
    if(todayDot){todayDot.setAttribute('display',ty?'inline':'none');if(ty){todayDot.setAttribute('cx',x);todayDot.setAttribute('cy',ty);}}
    if(tip){
      const text=tip.querySelector('text'),rect=tip.querySelector('rect'),point=points[index];
      text.textContent=time+' · '+recordValueText(state.analysisMetric!=='wait',point,singleDate);
      const w=recordAnnotationWidth(text),left=Number(svg.dataset.left),right=Number(svg.dataset.right);
      rect.setAttribute('width',w.toFixed(1));text.setAttribute('x',(w/2).toFixed(1));
      tip.setAttribute('transform','translate('+Math.max(left,Math.min(right-w,Number(x)-w/2)).toFixed(1)+',0)');
    }
  }
  el('analysis-time').value=String(index);
  el('analysis-time').style.setProperty('--fill',(points.length>1?index/(points.length-1)*100:0)+'%');
  el('analysis-time').setAttribute('aria-valuetext',recordTimeRange(time)+'，'+calledPointLabel(called,singleDate)+(wait?'，等待 '+waitWords(wait.median,!singleDate):''));
  chart.querySelectorAll('[data-record-point]').forEach(node=>{
    const selected=node.dataset.recordPoint===time;
    node.classList.toggle('selected',selected);
    node.setAttribute('aria-pressed',String(selected));
    node.setAttribute('tabindex',selected?'0':'-1');
  });
}
function renderRecordTable(data,open,singleDate) {
  const called=new Map((data.called_points||[]).map(point=>[point.time,point]));
  const waits=new Map((data.points||[]).map(point=>[point.time,point]));
  const times=[...new Set([...called.keys(),...waits.keys()])].sort();
  if(!times.length){el('record-table').innerHTML='';return;}
  const none='<td class="num missing">未记录</td>',cell=text=>'<td class="num">'+escapeHTML(text)+'</td>';
  const head=singleDate?['时段','当日叫到','当日要等','记录天数（叫号 / 等待）']:['时段','通常叫到','常见范围','通常要等','八成日子不超过','记录天数（叫号 / 等待）'];
  el('record-table').innerHTML='<details'+(open?' open':'')+'><summary>每半小时明细</summary><p class="caption">每家店每天每半小时取最后一条，和内置历史重叠时用自己的记录。叫号是堂食队列最大号，不含预约；常见范围是 10 天里约 6 天落在的区间，不保证当天。等待是门店当时公布的预计时间。</p><div class="table-wrap"><table><thead><tr>'+head.map((name,i)=>'<th'+(i?' class="num"':'')+'>'+name+'</th>').join('')+'</tr></thead><tbody>'+times.map(time=>{
    const c=called.get(time),w=waits.get(time),cells=[c?cell(Math.round(c.median)+' 号'):none];
    if(!singleDate)cells.push(!c?none:c.days>=3&&Math.floor(c.lower)<Math.ceil(c.upper)?cell(Math.floor(c.lower)+'–'+Math.ceil(c.upper)+' 号'):cell('—'));
    cells.push(w?cell(Math.round(w.median)+' 分钟'):none);
    if(!singleDate)cells.push(!w?none:w.days>=3?cell(Math.round(w.upper)+' 分钟'):cell('—'));
    return '<tr><td>'+escapeHTML(recordTimeRange(time))+'</td>'+cells.join('')+cell((c?.days||0)+' 天 / '+(w?.days||0)+' 天')+'</tr>';
  }).join('')+'</tbody></table></div></details>';
}
let recordChartSignature='';
function renderRecordChart(data) {
  const called=state.analysisMetric!=='wait',points=recordChartPoints(),id=el('analysis-store').value;
  const todayPoints=recordTodayPoints(called);
  const signature=JSON.stringify([id,el('analysis-date').value,called,el('record-chart').clientWidth,data.points,data.called_points,todayPoints,data.history?.included,data.history?.quality_note,data.history_samples]);
  if(signature===recordChartSignature)return;
  recordChartSignature=signature;
  const tableOpen=el('record-table').querySelector('details')?.open;
  const singleDate=el('analysis-date').value;
  renderRecordTable(data,tableOpen,singleDate);
  el('analysis-inspection').hidden=!id||!points.length;
  if(!id||!points.length){
    const otherData=(data.points||[]).length+(data.called_points||[]).length>0;
    const title=otherData?(called?'这些时段还没有叫号记录':'这些时段还没有等待记录'):!data.history?.included?(id?'还没有符合条件的本机记录':'还没有本机记录'):!id?'看看一家店的排队规律':'这个范围没有记录';
    const hint=otherData?'可以切换上方的曲线查看已有数据。':!data.history?.included?'开始记录常去的门店，或打开内置历史。':'换个日期，或开始记录常去的门店。';
    el('record-chart').innerHTML='<div class="empty"><strong>'+title+'</strong>'+hint+'</div>';
    return;
  }
  const enough=points.filter(point=>point.days>=3).length>=3,thin=recordThinDays(points,!!singleDate);
  const width=Math.max(260,Math.min(880,el('record-chart').clientWidth||880)),height=width<520?240:280;
  const ticks=recordAxisTicks(Math.max(called?100:30,...points.map(point=>singleDate?point.median:point.upper),...todayPoints.map(point=>point.median)),called),max=ticks.at(-1);
  const left=Math.max(36,14+7*String(max).length),right=width-24,top=40,bottom=height-32;
  const times=[...points,...todayPoints].map(point=>recordMinute(point.time)),lo=Math.min(...times),hi=Math.max(lo+30,...times);
  const X=minute=>left+(minute-lo)/(hi-lo)*(right-left),x=point=>X(recordMinute(point.time)),y=value=>bottom-value/max*(bottom-top),f=n=>n.toFixed(1);
  const segments=chartSegments(points);
  const path=(key,parts=segments)=>parts.map(segment=>segment.map((point,i)=>(i?'L':'M')+f(x(point))+','+f(y(point[key]))).join(' ')).join(' ');
  const band=segments.map(segment=>segment.map((point,i)=>(i?'L':'M')+f(x(point))+','+f(y(point.upper))).join(' ')+' '+[...segment].reverse().map(point=>'L'+f(x(point))+','+f(y(point.lower))).join(' ')+' Z').join(' ');
  const grid=ticks.map(v=>'<line class="chart-grid" x1="'+left+'" x2="'+right+'" y1="'+f(y(v))+'" y2="'+f(y(v))+'"/><text class="chart-label" x="'+(left-8)+'" y="'+f(y(v)+4)+'" text-anchor="end">'+v+'</text>').join('')+'<text class="chart-label" x="'+(left-8)+'" y="20" text-anchor="end">'+(called?'号':'分钟')+'</text>';
  const xLabels=recordHourTicks(lo,hi,right-left).map(m=>'<text class="chart-label" x="'+f(X(m))+'" y="'+(height-10)+'" text-anchor="middle">'+recordClock(m)+'</text>').join('');
  const spread=singleDate?'':called?'<path class="chart-band" d="'+band+'"/>':'<path class="chart-upper" d="'+path('upper')+'"/>';
  const lonely=new Set(segments.filter(segment=>segment.length===1).map(segment=>segment[0]));
  const marked=points.filter(point=>lonely.has(point)||point.days<thin);
  const dots=marked.map(point=>'<circle class="chart-dot'+(point.days<thin?' thin':'')+'" cx="'+f(x(point))+'" cy="'+f(y(point.median))+'" r="4"/>').join('');
  const todayAt=new Map(todayPoints.map(point=>[point.time,point])),last=todayPoints.at(-1);
  let today='';
  if(last){
    const lx=x(last),ly=y(last.median),near=lx>right-40,ty=ly-20<top?ly+18:ly-8;
    today='<path class="chart-today" d="'+path('median',chartSegments(todayPoints))+'"/>'+chartSegments(todayPoints).filter(s=>s.length===1&&s[0]!==last).map(s=>'<circle class="chart-today-dot" cx="'+f(x(s[0]))+'" cy="'+f(y(s[0].median))+'" r="3"/>').join('')+'<circle class="chart-today-dot" cx="'+f(lx)+'" cy="'+f(ly)+'" r="4"/><text class="chart-label chart-direct" x="'+f(near?lx-8:lx+8)+'" y="'+f(ty)+'" text-anchor="'+(near?'end':'start')+'">今天</text>';
  }
  const focus='<line class="chart-crosshair" x1="-10" x2="-10" y1="'+top+'" y2="'+bottom+'"/><circle class="chart-selected" cx="-10" cy="-10" r="5"/>'+(last?'<circle class="chart-selected-today" cx="-10" cy="-10" r="4" display="none"/>':'');
  const xs=points.map(x);
  const hits=points.map((point,i)=>{
    const x0=i?(xs[i-1]+xs[i])/2:Math.max(0,xs[i]-12),x1=i<xs.length-1?(xs[i]+xs[i+1])/2:Math.min(width,xs[i]+12),t=todayAt.get(point.time);
    const description=recordTimeRange(point.time)+'，'+(called?calledPointLabel(point,!!singleDate):(singleDate?'当日要等 ':'通常要等 ')+waitWords(point.median,!singleDate))+(singleDate?'':'，'+point.days+' 天记录');
    return '<g class="chart-point" data-record-point="'+point.time+'" data-x="'+f(xs[i])+'" data-y="'+f(y(point.median))+'"'+(t?' data-ty="'+f(y(t.median))+'"':'')+' role="button" tabindex="-1" aria-label="'+escapeHTML(description)+'"><rect class="chart-hit" x="'+f(x0)+'" y="'+top+'" width="'+f(Math.max(1,x1-x0))+'" height="'+(bottom-top)+'" fill="transparent"/></g>';
  }).join('');
  const legend=singleDate?'':'<div class="legend"><span class="legend-median">'+(called?(enough?'通常叫到':'记录叫到'):'通常要等')+'</span>'+(called?'<span class="legend-range">常见范围</span>':'<span class="legend-upper">八成日子不超过</span>')+(last?'<span class="legend-today">今天</span>':'')+(marked.some(point=>point.days<thin)?'<span class="legend-thin">记录天数较少</span>':'')+'</div>';
  const head=(singleDate?'<p class="caption">'+escapeHTML(singleDate+' · 每半小时一条记录')+'</p>':'')+legend;
  const notes=[(!singleDate&&!enough?'记录还少，仅供参考。':'')+(singleDate?'当天的记录':'以往的规律')+(called?'，不是实时叫号。叫号是堂食队列最大号，不分桌型。':'，不是实时排队。等待是门店当时公布的预计时间。')];
  if(!called&&data.history_samples&&data.history?.quality_note)notes.push(data.history.quality_note);
  el('record-chart').innerHTML=(head?'<div class="chart-head">'+head+'</div>':'')+'<svg viewBox="0 0 '+width+' '+height+'" data-left="'+left+'" data-right="'+right+'" role="group" aria-label="'+(called?'以往的堂食叫号，横轴是时间，纵轴是号码':'以往的等待时间，横轴是时间，纵轴是分钟')+'；用左右方向键逐个时段查看，没有记录的时段不连线">'+grid+xLabels+spread+'<path class="chart-line" d="'+path('median')+'"/>'+today+dots+focus+hits+'<g class="chart-annotation" pointer-events="none" aria-hidden="true"><rect y="4" height="24" rx="12" width="120"/><text y="20.5" x="60" text-anchor="middle"></text></g></svg>'+notes.map(note=>'<p class="caption chart-note">'+escapeHTML(note)+'</p>').join('');
  el('analysis-time').max=String(points.length-1);
  const slot=recordCurrentSlot(),wanted=recordMinute(state.analysisTime||(!singleDate&&points.some(point=>point.time===slot)?slot:'18:00'));
  const chosen=points.reduce((best,point)=>Math.abs(recordMinute(point.time)-wanted)<Math.abs(recordMinute(best.time)-wanted)?point:best,points[0]);
  updateRecordInspection(chosen.time);
}

function initRecordChart() {
  document.querySelectorAll('input[name="analysis-metric"]').forEach(input=>input.addEventListener('change',()=>{
    state.analysisMetric=input.value;
    if(state.records)renderAnalysis();
  }));
  el('analysis-time').addEventListener('input',()=>{
    const point=recordChartPoints()[Number(el('analysis-time').value)];
    if(point)updateRecordInspection(point.time,'slider');
  });
  const chart=el('record-chart');
  for(const name of ['pointerover','pointermove','focusin','click'])chart.addEventListener(name,event=>{
    let point=event.target.closest('[data-record-point]');
    const svg=event.target.closest('svg');
    // The crosshair finds the nearest half-hour, so the pointer never has to land on the line.
    if(svg&&name!=='focusin'){
      const bounds=svg.getBoundingClientRect(),x=(event.clientX-bounds.left)/bounds.width*svg.viewBox.baseVal.width;
      point=[...svg.querySelectorAll('[data-record-point]')].reduce((best,node)=>!best||Math.abs(Number(node.dataset.x)-x)<Math.abs(Number(best.dataset.x)-x)?node:best,null);
    }
    if(point)updateRecordInspection(point.dataset.recordPoint,name==='focusin'?'focus':'pointer');
  });
  chart.addEventListener('keydown',event=>{
    const point=event.target.closest('[data-record-point]');
    if(!point)return;
    if(event.key==='Enter'||event.key===' '){event.preventDefault();updateRecordInspection(point.dataset.recordPoint);return;}
    const nodes=[...chart.querySelectorAll('[data-record-point]')],moves={ArrowLeft:nodes.indexOf(point)-1,ArrowRight:nodes.indexOf(point)+1,Home:0,End:nodes.length-1};
    if(!(event.key in moves))return;
    event.preventDefault();
    nodes[Math.max(0,Math.min(nodes.length-1,moves[event.key]))].focus();
  });
}
