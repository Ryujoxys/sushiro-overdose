import assert from 'node:assert/strict';
import {test} from 'node:test';
import fs from 'node:fs';
import {gunzipSync} from 'node:zlib';
import {createHash} from 'node:crypto';
import {buildDataset,buildPatterns,halfHour,quantile} from './export-public-data.mjs';

const row=(time,extra={})=>({store_id:1,collected_at:'2026-09-04T'+time+':00+08:00',date_type:'weekday',store_status:'OPEN',wait_minutes:30,display_called_no:500,...extra});

test('quantiles interpolate like the app',()=>{
  assert.equal(quantile([40,10,30,20],.5),25);
  assert.equal(quantile([10,20,30,40],.8),34);
  assert.equal(quantile([7],.2),7);
  assert.equal(quantile([],.5),null);
});

test('half-hour buckets use China time whatever the stamp offset',()=>{
  assert.equal(halfHour('2026-09-04T10:44:00Z'),'18:30');
  assert.equal(halfHour('2026-09-04T18:29:59+08:00'),'18:00');
});

test('patterns count only open, known values and keep date types apart',()=>{
  const p=buildPatterns([
    row('18:05'),row('18:10',{collected_at:'2026-09-05T18:10:00+08:00',wait_minutes:0,display_called_no:600}),
    row('18:15',{collected_at:'2026-09-06T18:15:00+08:00',date_type:'holiday',wait_minutes:null,display_called_no:null}),
    row('18:20',{collected_at:'2026-09-07T18:20:00+08:00',store_status:'CLOSED',wait_minutes:99,display_called_no:999}),
  ]);
  const all=p.all.find(b=>b.time==='18:00');
  assert.deepEqual([all.wait_days,all.wait_p50,all.called_days,all.called_p50],[2,15,2,550],'zero wait kept, unknown and closed values dropped');
  assert.equal(p.weekday[0].wait_days,2);
  assert.deepEqual(p.holiday[0],{time:'18:00',wait_days:0,wait_p50:null,wait_p80:null,called_days:0,called_p20:null,called_p50:null,called_p80:null});
  assert.deepEqual(p.weekend,[]);
});

test('dataset lists every file with its hash and skips stores without records',()=>{
  const files=buildDataset({source:'bundle',stores:[{id:1,name:'A',city:'',area:'深圳'},{id:2,name:'B',city:'',area:''}],records:[row('18:05'),row('19:05',{collected_at:'2026-09-05T19:05:00+08:00'})],cutoffAt:'2026-09-05T19:05:00+08:00',generatedAt:'2026-10-20T00:00:00Z',windowDays:180});
  const meta=JSON.parse(files.get('meta.json'));
  assert.equal(meta.schema,'sushiro-public-data');
  assert.equal(meta.store_count,1);
  assert.ok(!files.has('patterns/2.json'),'store without records published');
  for(const [name,info] of Object.entries(meta.files)){
    assert.equal(info.sha256,createHash('sha256').update(files.get(name)).digest('hex'),name);
  }
  assert.deepEqual(JSON.parse(files.get('stores.json'))[0],{id:1,name:'A',city:'',area:'深圳',first_at:'2026-09-04T18:05:00+08:00',last_at:'2026-09-05T19:05:00+08:00',days:2,wait_days:2,called_days:2});
  assert.deepEqual(JSON.parse(files.get('daily/1.json')).rows[0],['2026-09-04T18:05:00+08:00','weekday','OPEN',30,500,null,null]);
});

test('patterns from the bundled history match the app chart readouts',()=>{
  const bundle=JSON.parse(gunzipSync(fs.readFileSync(new URL('../internal/app/data/queue-history-v1.json.gz',import.meta.url))).toString());
  const p=buildPatterns(bundle.records.filter(r=>r.store_id===1));
  const slot=p.all.find(b=>b.time==='18:00');
  assert.deepEqual([slot.called_p50,Math.floor(slot.called_p20),Math.ceil(slot.called_p80),slot.called_days],[676,535,1244,75]);
  assert.deepEqual([slot.wait_p50,slot.wait_p80,slot.wait_days],[165,175,14]);
});
