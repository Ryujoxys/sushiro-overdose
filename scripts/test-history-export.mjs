import assert from 'node:assert/strict';
import {test} from 'node:test';
import {dateParts,normalizeSample,sampleKey} from './export-history-bundle.mjs';

const raw={store_id:'1',collected_at:'2026-09-04T16:45:00+08:00',wait_minutes:0,store_status:'OPEN',wait_time_cap:180,dq_anomaly:0};
const normalize=(row,source='raw',calendar=new Map())=>normalizeSample(row,source,calendar);

test('preserve real zero, unknowns, and invalid waits separately',()=>{
  assert.equal(normalize(raw).wait_minutes,0);
  for(const value of [null,undefined,-1,181,1.5,'invalid'])assert.equal(normalize({...raw,wait_minutes:value}).wait_minutes,null);
  assert.equal(normalize({...raw,wait_minutes:30,dq_anomaly:1}).wait_minutes,null);
  assert.equal(normalize({...raw,wait_minutes:190,wait_time_cap:200}).wait_minutes,190);
});

test('archived daily representatives must agree with date and half-hour',()=>{
  const daily={store_id:1,updated_at:raw.collected_at,snapshot_date:'2026-09-04',time_bucket:'16:30',sample_count:1,open_count:1,wait_typical_minutes:null};
  assert.equal(normalize(daily,'daily').wait_minutes,null);
  assert.equal(normalize({...daily,wait_typical_minutes:0},'daily').wait_minutes,0);
  assert.throws(()=>normalize({...daily,sample_count:2},'daily'));
  assert.throws(()=>normalize({...daily,snapshot_date:'2026-09-05'},'daily'));
  assert.throws(()=>normalize({...daily,time_bucket:'16:00'},'daily'));
});

test('CST keys and calendar overrides do not use machine timezone',()=>{
  assert.equal(sampleKey(raw),'1/2026-09-04/16:30');
  assert.equal(sampleKey({...raw,collected_at:'2026-09-04T08:45:00Z'}),sampleKey(raw));
  assert.equal(dateParts('2026-09-04T16:45:00Z').date,'2026-09-05');
  assert.equal(normalize(raw).date_type,'weekend');
  assert.equal(normalize({...raw,collected_at:'2026-09-04T16:29:00+08:00'}).date_type,'weekday');
  assert.equal(normalize({...raw,collected_at:'2026-09-06T22:00:00+08:00'}).date_type,'weekday');
  assert.equal(normalize(raw,'raw',new Map([['2026-09-04','workday']])).date_type,'workday');
  assert.throws(()=>normalize({...raw,store_id:0}));
  assert.throws(()=>normalize({...raw,collected_at:'invalid'}));
});

test('the packet schema is allowlisted, not a copy of source rows',()=>{
  const out=normalize({...raw,authorization:'secret',phone:'secret',number:'secret',database_url:'secret'});
  assert.deepEqual(Object.keys(out).sort(),['collected_at','date_type','display_called_no','store_id','store_status','wait_minutes']);
  assert.equal(JSON.stringify(out).includes('secret'),false);
});

test('called numbers are actual positive daily values, independent of wait validity',()=>{
  const out=normalize({...raw,wait_minutes:null,display_called_no:123});
  assert.equal(out.wait_minutes,null);assert.equal(out.display_called_no,123);
  for(const value of [null,undefined,0,-1,1.5,'invalid',2147483648])assert.equal(normalize({...raw,display_called_no:value}).display_called_no,null);
  const daily={store_id:1,updated_at:raw.collected_at,snapshot_date:'2026-09-04',time_bucket:'16:30',sample_count:1,open_count:1,called_sample_count:1,called_no_typical:456};
  assert.equal(normalize(daily,'daily').display_called_no,456);
  assert.equal(normalize({...daily,called_sample_count:0},'daily').display_called_no,null);
  assert.throws(()=>normalize({...daily,called_sample_count:2},'daily'));
});
