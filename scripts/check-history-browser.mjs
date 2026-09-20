// Run only against a fresh isolated local preview. No collection, auth or ticket writes.
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
const {chromium}=createRequire(import.meta.url)('playwright');
const origin=process.env.SUSHIRO_QA_URL||'http://127.0.0.1:39871';
assert.ok(['127.0.0.1','localhost'].includes(new URL(origin).hostname),'local isolated preview required');
const output=process.env.SUSHIRO_QA_OUTPUT||'/tmp/sushiro-history-real-qa';
await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.SUSHIRO_QA_BROWSER?{executablePath:process.env.SUSHIRO_QA_BROWSER}:{})});
const context=await browser.newContext({viewport:{width:1280,height:960},locale:'zh-CN',hasTouch:true});
const page=await context.newPage(),errors=[],requests=[],external=[];
page.on('pageerror',error=>errors.push(error.message));
await context.route('**/*',route=>{
  const request=route.request();
  if(new URL(request.url()).origin!==new URL(origin).origin){external.push(request.url());return route.abort();}
  requests.push({path:new URL(request.url()).pathname,method:request.method()});
  return route.continue();
});
const shot=async name=>page.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'});
const noOverflow=async()=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'horizontal overflow');
const records=()=>page.request.get(origin+'/api/records?days=all').then(r=>r.json());
let originalSetting;
try{
  originalSetting=(await page.request.get(origin+'/api/records/settings').then(r=>r.json())).include_history;
  assert.equal(originalSetting,true,'use a fresh isolated preview');
  await page.goto(origin);
  await page.locator('#record-chart svg').waitFor();
  const initial=await records();
  assert.ok(initial.history_samples>0);assert.ok(initial.called_history_samples>0);assert.ok(initial.called_points.length>0);assert.equal(initial.local_samples,0);
  assert.equal(initial.stores.length,0);assert.ok(initial.available_stores.length>0);
  assert.match(await page.locator('#history-meta').innerText(),/2026-06-17.*2026-09-06/);
  assert.equal(requests.some(r=>r.method==='POST'),false,'entry wrote settings or began collection');
  assert.match(await page.locator('#record-overview').innerText(),/已保存\s*0/);
  assert.equal(await page.getByRole('radio',{name:'叫到几号',exact:true}).isChecked(),true);
  const firstPoint=initial.called_points[0],secondPoint=initial.called_points[1];
  await page.locator('[data-record-point="'+firstPoint.time+'"]').focus();
  assert.match(await page.locator('.chart-annotation text').textContent(),new RegExp(Math.round(firstPoint.median)+' 号'));
  await page.locator('#analysis-time').focus();await page.locator('#analysis-time').press('Home');await page.locator('#analysis-time').press('ArrowRight');
  assert.equal(await page.locator('#analysis-time').inputValue(),'1');
  assert.match(await page.locator('.chart-annotation text').textContent(),new RegExp(secondPoint.time));
  await page.locator('[data-record-point="18:00"]').hover();
  await noOverflow();await shot('history-desktop');
  await page.setViewportSize({width:390,height:844});await page.waitForTimeout(200);await noOverflow();await page.locator('[data-record-point]').last().tap();await shot('history-mobile');
  await page.setViewportSize({width:320,height:740});await page.waitForTimeout(200);await noOverflow();
  await page.setViewportSize({width:1280,height:960});
  await page.getByRole('radio',{name:'等多久',exact:true}).check();
  await page.locator('#record-chart svg').waitFor();
  assert.match(await page.locator('#analysis-readout').innerText(),/号/);
  await shot('history-wait-with-called');
  await page.getByRole('radio',{name:'叫到几号',exact:true}).check();
  await page.locator('#analysis-date').selectOption('2026-09-06');
  await page.getByText('2026-09-06 · 每半小时一条记录',{exact:true}).waitFor();
  await page.locator('#record-table summary').click();
  assert.ok(await page.locator('#record-table tbody tr').count()>0);
  assert.match(await page.locator('#record-table tbody tr').first().innerText(),/\d+ 天 \/ 1 天/);
  assert.match(await page.locator('#analysis-readout').innerText(),/当日叫到/);
  await shot('history-single-date');
  await page.locator('#analysis-days').selectOption('7');await page.getByText('这个范围没有记录',{exact:true}).waitFor();
  await page.locator('#analysis-days').selectOption('all');await page.locator('#record-chart svg').waitFor();
  await page.locator('#include-history').uncheck();await page.getByText('还没有符合条件的本机记录',{exact:true}).waitFor();
  await page.reload();await page.getByText('还没有本机记录',{exact:true}).waitFor();
  assert.equal(await page.locator('#include-history').isChecked(),false);
  const off=await records();assert.equal(off.history_samples,0);assert.equal(off.points.length,0);assert.equal(off.called_points.length,0);assert.equal(off.history.included,false);
  const exportResponse=await page.request.get(origin+'/api/records/export?days=all');assert.equal(exportResponse.status(),409);assert.match((await exportResponse.json()).error,/没有可导出的个人记录/);
  await shot('history-off');
  await page.locator('#include-history').check();await page.locator('#record-chart svg').waitFor();
  assert.deepEqual(external,[]);assert.deepEqual(errors,[]);
  assert.deepEqual(requests.filter(r=>r.method==='POST').map(r=>r.path),['/api/records/settings','/api/records/settings']);
  console.log('Real offline history QA passed: first curve, actual cutoff, date filters, personal-count isolation, persisted opt-out, empty export, desktop/mobile, no external browser requests. Screenshots: '+output);
}catch(error){
  await shot('failure');
  console.error(await page.locator('#page-records').innerText(),await page.locator('#toast').textContent());
  throw error;
}finally{
  if(originalSetting!==undefined){
    const token=await page.locator('meta[name="sushiro-csrf"]').getAttribute('content').catch(()=>null);
    if(token)await page.request.post(origin+'/api/records/settings',{headers:{'X-Sushiro-CSRF':token,Origin:new URL(origin).origin},data:{include_history:originalSetting}}).catch(()=>{});
  }
  await browser.close();
}
