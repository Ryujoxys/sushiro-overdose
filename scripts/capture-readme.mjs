// Capture the real embedded history and recording controls. No API writes.
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';

const {chromium}=createRequire(import.meta.url)('playwright');
const origin=process.env.SUSHIRO_QA_URL||'http://127.0.0.1:39871';
assert.ok(['127.0.0.1','localhost'].includes(new URL(origin).hostname));
const output=new URL('../docs/images/',import.meta.url);
await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.SUSHIRO_QA_BROWSER?{executablePath:process.env.SUSHIRO_QA_BROWSER}:{})});
try {
  const page=await browser.newPage({viewport:{width:1280,height:1024},locale:'zh-CN',deviceScaleFactor:1});
  await page.route('**/*',route=>{
    const request=route.request();
    if(request.method()!=='GET'||new URL(request.url()).origin!==new URL(origin).origin)return route.abort();
    return route.continue();
  });
  await page.goto(origin);
  await page.locator('#record-chart svg').waitFor();
  await page.locator('[data-record-point="18:00"]').focus();
  await page.locator('.analysis-card').screenshot({path:path.join(output.pathname,'records.png'),animations:'disabled'});
  await page.locator('.record-layout').screenshot({path:path.join(output.pathname,'recording.png'),animations:'disabled'});
  console.log('README screenshots: '+output.pathname);
} finally {
  await browser.close();
}
