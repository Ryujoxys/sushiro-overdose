import assert from 'node:assert/strict';
import {test} from 'node:test';
import {WAIT_CAP_MINUTES,dateParts,normalizeRecord,sampleKey} from './export-history-bundle.mjs';

const row={slot:'2026-09-04T16:30',store_id:'1',collected_at:'2026-09-04T16:45:00+08:00',wait_minutes:'0',store_status:'open',display_called_no:null,group_queues_count:'3',online_open:'1'};
const normalize=(r,calendar=new Map())=>normalizeRecord(r,calendar);

test('preserve real zero, unknowns, and waits beyond the official cap separately',()=>{
  assert.equal(normalize(row).wait_minutes,0);
  assert.equal(normalize({...row,wait_minutes:WAIT_CAP_MINUTES}).wait_minutes,WAIT_CAP_MINUTES);
  for(const value of [null,undefined,-1,WAIT_CAP_MINUTES+1,280,1.5,'invalid'])assert.equal(normalize({...row,wait_minutes:value}).wait_minutes,null);
});

test('records must sit in their own half-hour slot',()=>{
  assert.throws(()=>normalize({...row,slot:'2026-09-04T16:00'}));
  assert.throws(()=>normalize({...row,slot:'2026-09-05T16:30'}));
  assert.equal(normalize({...row,slot:undefined}).store_status,'OPEN');
});

test('CST keys and calendar overrides do not use machine timezone',()=>{
  assert.equal(sampleKey(row),'1/2026-09-04/16:30');
  assert.equal(sampleKey({...row,collected_at:'2026-09-04T08:45:00Z'}),sampleKey(row));
  assert.equal(dateParts('2026-09-04T16:45:00Z').date,'2026-09-05');
  assert.equal(normalize(row).date_type,'weekend');
  assert.equal(normalize({...row,slot:'2026-09-04T16:00',collected_at:'2026-09-04T16:29:00+08:00'}).date_type,'weekday');
  assert.equal(normalize({...row,slot:'2026-09-06T22:00',collected_at:'2026-09-06T22:00:00+08:00'}).date_type,'weekday');
  assert.equal(normalize(row,new Map([['2026-09-04','workday']])).date_type,'workday');
  assert.throws(()=>normalize({...row,store_id:0}));
  assert.throws(()=>normalize({...row,collected_at:'invalid'}));
});

test('the packet schema is allowlisted, not a copy of source rows',()=>{
  const out=normalize({...row,authorization:'secret',phone:'secret',number:'secret',database_url:'secret'});
  assert.deepEqual(Object.keys(out).sort(),['collected_at','date_type','display_called_no','store_id','store_status','wait_minutes']);
  assert.equal(JSON.stringify(out).includes('secret'),false);
});

test('called numbers are positive values, independent of wait validity',()=>{
  const out=normalize({...row,wait_minutes:null,display_called_no:'123'});
  assert.equal(out.wait_minutes,null);assert.equal(out.display_called_no,123);
  for(const value of [null,undefined,0,-1,1.5,'invalid',2147483648])assert.equal(normalize({...row,display_called_no:value}).display_called_no,null);
});
