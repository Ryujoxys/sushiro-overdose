'use strict';

function recordTimeRange(time) {
  return time+'–'+time.slice(0,3)+(time.endsWith('00')?'29':'59');
}
function calledPointLabel(point,singleDate=false) {
  if(!point)return '暂无叫号记录';
  return (singleDate?'当日叫到':point.days>=3?'通常叫到':'记录叫到')+(singleDate?' ':'约 ')+Math.round(point.median)+' 号';
}
function chartSegments(points) {
  const minutes=time=>Number(time.slice(0,2))*60+Number(time.slice(3));
  const segments=[];
  for(const point of points){
    const last=segments.at(-1);
    if(!last||minutes(point.time)-minutes(last.at(-1).time)>30)segments.push([point]);else last.push(point);
  }
  return segments;
}
function recordChartPoints() {
  return state.analysisMetric==='wait'?(state.records?.points||[]):(state.records?.called_points||[]);
}
function updateRecordInspection(time) {
  const data=state.records,points=recordChartPoints(),index=points.findIndex(point=>point.time===time);
  if(index<0)return;
  state.analysisTime=time;
  const called=(data.called_points||[]).find(point=>point.time===time),wait=(data.points||[]).find(point=>point.time===time);
  const singleDate=!!el('analysis-date').value;
  const label=calledPointLabel(called,singleDate);
  const number=value=>Math.round(value).toLocaleString('zh-CN');
  const calledDetail=called?(called.days>=3&&!singleDate?'常见范围 '+Math.floor(called.lower).toLocaleString('zh-CN')+'–'+Math.ceil(called.upper).toLocaleString('zh-CN')+' 号 · ':'')+called.days+' 天记录'+(!singleDate&&called.days<3?'，样本较少':''):'未记录，不补号';
  el('analysis-readout').innerHTML='<span class="readout-time">'+escapeHTML(recordTimeRange(time))+'</span><div><span>'+escapeHTML(called?singleDate?'当日叫到':called.days>=3?'通常叫到':'记录叫到':'叫号')+'</span><strong>'+escapeHTML(called?(singleDate?'':'约 ')+number(called.median)+' 号':'暂无记录')+'</strong><small>'+escapeHTML(calledDetail)+'</small></div><div><span>'+(singleDate?'当日等待':'常见等待')+'</span><strong>'+escapeHTML(wait?number(wait.median)+' 分钟':'暂无记录')+'</strong><small>'+escapeHTML(wait?wait.days+' 天记录':'未记录，不补零')+'</small></div>';
  const marker=el('record-chart').querySelector('[data-record-point="'+time+'"]'),tip=el('record-chart').querySelector('.chart-annotation'),svg=el('record-chart').querySelector('svg');
  if(marker&&tip&&svg){
    const x=Number(marker.dataset.x),y=Number(marker.dataset.y),width=svg.viewBox.baseVal.width;
    tip.setAttribute('transform','translate('+Math.max(4,Math.min(width-184,x-90))+','+(y<55?y+16:y-38)+')');
    tip.querySelector('text').textContent=time+' · '+(state.analysisMetric==='wait'?Math.round(wait.median)+' 分钟':Math.round(called.median)+' 号');
  }
  el('analysis-time').value=String(index);
  el('analysis-time').setAttribute('aria-valuetext',recordTimeRange(time)+'，'+label+(wait?'，等待 '+Math.round(wait.median)+' 分钟':''));
  el('record-chart').querySelectorAll('[data-record-point]').forEach(node=>{
    const selected=node.dataset.recordPoint===time;
    node.classList.toggle('selected',selected);
    node.setAttribute('aria-pressed',String(selected));
  });
}
function renderRecordTable(data,open) {
  const called=new Map((data.called_points||[]).map(point=>[point.time,point]));
  const waits=new Map((data.points||[]).map(point=>[point.time,point]));
  const times=[...new Set([...called.keys(),...waits.keys()])].sort();
  if(!times.length){el('record-table').innerHTML='';return;}
  el('record-table').innerHTML='<details'+(open?' open':'')+'><summary>每个时段的数据</summary><p class="caption">每店、每天、每半小时取最后一条，重叠用本机记录。叫号取堂食队列最大号，不含预约号。范围为历史 P20–P80，不是预测保证。</p><div class="table-wrap"><table><thead><tr><th>时段</th><th>叫号中位数</th><th>叫号范围</th><th>等待中位数</th><th>叫号 / 等待样本</th></tr></thead><tbody>'+times.map(time=>{
    const c=called.get(time),w=waits.get(time);
    return '<tr><td>'+escapeHTML(recordTimeRange(time))+'</td><td>'+(c?Math.round(c.median)+' 号':'未记录')+'</td><td>'+(c?Math.floor(c.lower)+'–'+Math.ceil(c.upper)+' 号':'未记录')+'</td><td>'+(w?Math.round(w.median)+' 分钟':'未记录')+'</td><td>'+(c?.days||0)+' 天 / '+(w?.days||0)+' 天</td></tr>';
  }).join('')+'</tbody></table></div></details>';
}
let recordChartSignature='';
function renderRecordChart(data) {
  const called=state.analysisMetric!=='wait',points=recordChartPoints(),id=el('analysis-store').value;
  const signature=JSON.stringify([id,el('analysis-date').value,called,el('record-chart').clientWidth,data.points,data.called_points,data.history?.included,data.history?.quality_note,data.history_samples]);
  if(signature===recordChartSignature)return;
  recordChartSignature=signature;
  const tableOpen=el('record-table').querySelector('details')?.open;
  const singleDate=el('analysis-date').value;
  renderRecordTable(data,tableOpen);
  el('analysis-inspection').hidden=!id||!points.length;
  if(!id||!points.length){
    const otherData=(data.points||[]).length+(data.called_points||[]).length>0;
    const title=otherData?(called?'这些时段还没有叫号记录':'这些时段还没有等待记录'):!data.history?.included?(id?'还没有符合条件的本机记录':'还没有本机记录'):!id?'看看一家店的排队规律':'这个范围没有记录';
    const hint=otherData?'可以切换上方的曲线查看已有数据。':!data.history?.included?'开始记录常去的门店，或打开历史数据。':'换个日期，或开始记录常去的门店。';
    el('record-chart').innerHTML='<div class="empty"><strong>'+title+'</strong>'+hint+'</div>';
    return;
  }
  const enough=points.filter(point=>point.days>=3).length>=3;
  const max=Math.max(called?100:30,...points.map(point=>point.upper));
  const width=Math.max(260,Math.min(880,el('record-chart').clientWidth||880)),height=250,left=44,top=20,bottom=210;
  const minute=time=>Number(time.slice(0,2))*60+Number(time.slice(3));
  const lo=minute(points[0].time),hi=Math.max(lo+30,minute(points.at(-1).time));
  const x=point=>left+(minute(point.time)-lo)/(hi-lo)*(width-left-25),y=value=>bottom-value/max*(bottom-top);
  const segments=chartSegments(points);
  const path=key=>segments.map(segment=>segment.map((point,i)=>(i?'L':'M')+x(point).toFixed(1)+','+y(point[key]).toFixed(1)).join(' ')).join(' ');
  const band=segments.map(segment=>segment.map((point,i)=>(i?'L':'M')+x(point).toFixed(1)+','+y(point.upper).toFixed(1)).join(' ')+' '+[...segment].reverse().map(point=>'L'+x(point).toFixed(1)+','+y(point.lower).toFixed(1)).join(' ')+' Z').join(' ');
  const grid=[0,.5,1].map(k=>'<line x1="'+left+'" x2="'+(width-20)+'" y1="'+y(k*max)+'" y2="'+y(k*max)+'" stroke="#E5E0DB"/><text x="4" y="'+(y(k*max)+4)+'" class="chart-label">'+Math.round(k*max)+'</text>').join('');
  const labels=points.filter((point,i)=>i===0||i===points.length-1||i%Math.max(1,Math.ceil(points.length/(width<500?3:5)))===0).map(point=>'<text text-anchor="middle" x="'+x(point)+'" y="240" class="chart-label">'+escapeHTML(point.time)+'</text>').join('');
  const markers=points.map(point=>{
    const description=recordTimeRange(point.time)+'，'+(called?calledPointLabel(point,!!singleDate):'等待 '+Math.round(point.median)+' 分钟')+'，'+point.days+' 天记录';
    return '<g class="chart-point" data-record-point="'+point.time+'" data-x="'+x(point)+'" data-y="'+y(point.median)+'" role="button" tabindex="0" aria-label="'+escapeHTML(description)+'"><circle cx="'+x(point)+'" cy="'+y(point.median)+'" r="13" fill="transparent"/><circle class="chart-dot" cx="'+x(point)+'" cy="'+y(point.median)+'" r="3.5" fill="#B81C22"/></g>';
  }).join('');
  const caption=(singleDate?singleDate+' · 每半小时一条记录':called?'历史叫号 / 号，不是今天的实时叫号。':'历史等待 / 分钟，不代表今天。')+(!singleDate&&!enough?' 样本较少。':'');
  el('record-chart').innerHTML='<p class="caption">'+escapeHTML(caption)+'</p>'+(!called&&data.history_samples&&data.history?.quality_note?'<p class="caption">'+escapeHTML(data.history.quality_note)+'</p>':'')+'<svg viewBox="0 0 '+width+' '+height+'" role="group" aria-label="'+(called?'历史堂食叫号':'历史等待分钟数')+'，可点选时段，缺失时段不连线">'+grid+(called?'<path d="'+band+'" fill="#B81C22" fill-opacity=".09"/>':'<path d="'+path('upper')+'" fill="none" stroke="#958780" stroke-width="2" stroke-dasharray="5 5"/>')+'<path d="'+path('median')+'" fill="none" stroke="#B81C22" stroke-width="2.5"/>'+markers+labels+'<g class="chart-annotation" pointer-events="none" aria-hidden="true"><rect width="180" height="28" rx="6" fill="#282522"/><text x="90" y="19" text-anchor="middle" fill="white" font-size="13"></text></g></svg><div class="legend'+(called?' called-range':'')+'"><span>'+(called?(singleDate?'当日叫到':enough?'通常叫到（中位数）':'记录叫号（中位数）'):'常见等待（中位数）')+'</span><span>'+(called?'历史范围（20%–80%）':'较长等待（80% 分位）')+'</span></div>';
  el('analysis-time').max=String(points.length-1);
  const wanted=minute(state.analysisTime||'18:00');
  const chosen=points.reduce((best,point)=>Math.abs(minute(point.time)-wanted)<Math.abs(minute(best.time)-wanted)?point:best,points[0]);
  updateRecordInspection(chosen.time);
}

function initRecordChart() {
  document.querySelectorAll('input[name="analysis-metric"]').forEach(input=>input.addEventListener('change',()=>{
    state.analysisMetric=input.value;
    if(state.records)renderAnalysis();
  }));
  el('analysis-time').addEventListener('input',()=>{
    const point=recordChartPoints()[Number(el('analysis-time').value)];
    if(point)updateRecordInspection(point.time);
  });
  const chart=el('record-chart');
  for(const name of ['pointerover','pointermove','focusin','click'])chart.addEventListener(name,event=>{
    let point=event.target.closest('[data-record-point]');
    const svg=event.target.closest('svg');
    // Touch targets overlap on narrow screens; choose time by position, not DOM paint order.
    if(svg&&name!=='focusin'){
      const bounds=svg.getBoundingClientRect(),x=(event.clientX-bounds.left)/bounds.width*svg.viewBox.baseVal.width;
      point=[...svg.querySelectorAll('[data-record-point]')].reduce((best,node)=>!best||Math.abs(Number(node.dataset.x)-x)<Math.abs(Number(best.dataset.x)-x)?node:best,null);
    }
    if(point)updateRecordInspection(point.dataset.recordPoint);
  });
  chart.addEventListener('keydown',event=>{
    const point=event.target.closest('[data-record-point]');
    if(!point)return;
    if(event.key==='Enter'||event.key===' '){event.preventDefault();updateRecordInspection(point.dataset.recordPoint);}
    if(event.key==='ArrowLeft'||event.key==='ArrowRight'){
      event.preventDefault();
      const nodes=[...chart.querySelectorAll('[data-record-point]')],index=nodes.indexOf(point)+(event.key==='ArrowRight'?1:-1);
      nodes[Math.max(0,Math.min(nodes.length-1,index))].focus();
    }
  });
}
