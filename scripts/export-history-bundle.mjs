// Maintainer-only export. Never invoked by the app, build, CI, or release.
import fs from 'node:fs/promises';
import path from 'node:path';
import {gzipSync} from 'node:zlib';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';

export function dateParts(stamp) {
  const n=Date.parse(stamp);
  if(!Number.isFinite(n))throw new Error('Invalid source timestamp');
  const d=new Date(n+8*3600000),s=d.toISOString();
  return {date:s.slice(0,10),time:s.slice(11,16),weekday:d.getUTCDay(),stamp:s.slice(0,19)+'+08:00'};
}
export function sampleKey(row) {
  const d=dateParts(row.collected_at);
  return row.store_id+'/'+d.date+'/'+d.time.slice(0,3)+(Number(d.time.slice(3))<30?'00':'30');
}
// Official wait display cap: larger values are kept in the database as reported, but read as unknown.
export const WAIT_CAP_MINUTES=180;
// Archived days before this date came from old rollups that dropped zero waits.
export const ZERO_WAIT_GAP_BEFORE='2026-07-08';

// One queue_records row → bundle record. Only allowlisted public fields leave this function.
export function normalizeRecord(row, calendar) {
  const d=dateParts(row.collected_at),store=Number(row.store_id);
  if(!Number.isInteger(store)||store<=0)throw new Error('Invalid store ID');
  if(row.slot!==undefined&&row.slot!==d.date+'T'+sampleKey({store_id:store,collected_at:d.stamp}).slice(-5))throw new Error('Record time does not match its half-hour slot');
  let wait=row.wait_minutes===null||row.wait_minutes===undefined?null:Number(row.wait_minutes);
  if(wait!==null&&(!Number.isInteger(wait)||wait<0||wait>WAIT_CAP_MINUTES))wait=null;
  let called=row.display_called_no===null||row.display_called_no===undefined?null:Number(row.display_called_no);
  if(called!==null&&(!Number.isSafeInteger(called)||called<=0||called>2147483647))called=null;
  const override=calendar.get(d.date),minutes=Number(d.time.slice(0,2))*60+Number(d.time.slice(3));
  const weekend=d.weekday===6||(d.weekday===5&&minutes>=990)||(d.weekday===0&&minutes<1320);
  return {store_id:store,collected_at:d.stamp,date_type:override||(weekend?'weekend':'weekday'),store_status:String(row.store_status||'UNKNOWN').toUpperCase(),wait_minutes:wait,display_called_no:called};
}

// Read-only HTTPS client for the maintainer database. Only SELECT/WITH statements are sent.
export function openDatabase(url,token) {
  if(!url||!token)throw new Error('Provide read-only credentials via environment or --config; never use command-line token arguments.');
  const endpoint=new URL(url.replace(/^libsql:\/\//,'https://'));
  if(endpoint.protocol!=='https:'||endpoint.username||endpoint.password||endpoint.search||endpoint.hash)throw new Error('A plain HTTPS database origin is required');
  endpoint.pathname='/v2/pipeline';
  async function query(sql,args=[]) {
    if(!/^(SELECT|WITH)\b/.test(sql))throw new Error('Only SELECT queries are allowed');
    for(let attempt=0;attempt<3;attempt++){
      try{
        const stmt={sql,want_rows:true,args:args.map(v=>({type:'text',value:String(v)}))};
        const r=await fetch(endpoint,{method:'POST',headers:{Authorization:'Bearer '+token,'Content-Type':'application/json'},body:JSON.stringify({requests:[{type:'execute',stmt}]}),signal:AbortSignal.timeout(60000)});
        if(!r.ok)throw new Error('Database HTTP '+r.status);
        const data=await r.json(),entry=data.results?.[0];
        if(entry?.type==='error')throw new Error('Read-only query failed: '+entry.error?.code);
        const result=entry?.response?.result;if(!result)throw new Error('Incomplete database response');
        return result.rows.map(row=>Object.fromEntries(row.map((cell,n)=>[result.cols[n].name,cell.type==='null'?null:cell.value])));
      }catch(e){if(attempt===2)throw e;await new Promise(r=>setTimeout(r,1000*(attempt+1)));}
    }
  }
  return query;
}

// queue_records already holds one representative per store / CST half-hour slot.
// `since` (YYYY-MM-DD, CST) limits how far back to read; the cutoff is fixed before reading.
export async function readPublicRecords(query,{since='',withExtras=false}={}) {
  const [range]=await query('SELECT MIN(slot) AS first_slot, MAX(slot) AS last_slot FROM queue_records');
  if(!range?.first_slot)throw new Error('No public records available');
  const [bounds]=await query('SELECT MAX(collected_at) AS cutoff_at FROM queue_records WHERE slot = ?',[range.last_slot]);
  const dimensions=await query('SELECT store_id, name, city, area FROM store_dimension ORDER BY store_id');
  const calendar=new Map((await query('SELECT date_key,date_type FROM holiday_calendar')).filter(r=>['workday','holiday'].includes(r.date_type)).map(r=>[r.date_key,r.date_type]));
  const byID=new Map(dimensions.map(r=>[Number(r.store_id),{id:Number(r.store_id),name:r.name,city:r.city||'',area:r.area||''}]));
  const records=[],extras=[],quality={orphans:0,over_cap_waits:0};
  const plusDays=(day,n)=>new Date(Date.parse(day+'T00:00:00Z')+n*86400000).toISOString().slice(0,10);
  const first=range.first_slot.slice(0,10),last=range.last_slot.slice(0,10);
  for(let start=since&&since>first?since:first;start<=last;start=plusDays(start,7)){
    const batch=await query('SELECT slot,store_id,collected_at,store_status,online_open,wait_minutes,group_queues_count,display_called_no FROM queue_records WHERE slot >= ? AND slot < ? AND slot <= ? ORDER BY slot,store_id',[start,plusDays(start,7),range.last_slot]);
    for(const raw of batch){
      const row=normalizeRecord(raw,calendar);
      if(!byID.has(row.store_id)){quality.orphans++;continue;}
      if(Number(raw.wait_minutes)>WAIT_CAP_MINUTES)quality.over_cap_waits++;
      records.push(row);
      if(withExtras)extras.push({online_open:raw.online_open===null?null:Number(raw.online_open)===1,group_queues_count:raw.group_queues_count===null?null:Number(raw.group_queues_count),wait_over_cap:Number(raw.wait_minutes)>WAIT_CAP_MINUTES});
    }
    console.log('Records '+start+'..'+plusDays(start,7)+': '+batch.length);
  }
  if(!records.length||quality.orphans)throw new Error('Empty export or orphan stores; inspect before packaging');
  const used=new Set(records.map(r=>r.store_id));
  const stores=[...byID.values()].filter(s=>used.has(s.id));
  quality.unknown_waits=records.filter(r=>r.wait_minutes===null).length;
  quality.non_open=records.filter(r=>r.store_status!=='OPEN').length;
  quality.usable_waits=records.filter(r=>r.store_status==='OPEN'&&r.wait_minutes!==null).length;
  quality.valid_zero_waits=records.filter(r=>r.store_status==='OPEN'&&r.wait_minutes===0).length;
  quality.usable_called_numbers=records.filter(r=>r.store_status==='OPEN'&&r.display_called_no!==null).length;
  quality.unknown_called_numbers=records.filter(r=>r.display_called_no===null).length;
  const qualityNote=records[0].collected_at.slice(0,10)<ZERO_WAIT_GAP_BEFORE?ZERO_WAIT_GAP_BEFORE+' 前的归档记录缺少部分零等待值，早期曲线可能偏高。':'';
  return {bounds:{cutoff_at:bounds.cutoff_at},stores,records,extras,quality,qualityNote,calendar,rawEnd:last};
}

async function main() {
  const args=process.argv.slice(2),option=name=>{const i=args.indexOf(name);return i<0?'':args[i+1]||'';};
  const config=option('--config'),out=option('--out')||'internal/app/data';
  let credentials={};
  if(config)credentials=JSON.parse(await fs.readFile(config,'utf8'));
  const query=openDatabase(process.env.SUSHIRO_HISTORY_DATABASE_URL||credentials.database_url,process.env.SUSHIRO_HISTORY_DATABASE_TOKEN||credentials.auth_token);
  const {bounds,stores,records,quality,qualityNote,rawEnd}=await readPublicRecords(query);
  const bundle={version:1,id:'public-history-'+rawEnd,timezone:'Asia/Shanghai',bucket_minutes:30,source:'maintainer_snapshot',exported_at:new Date().toISOString(),first_at:records[0].collected_at,cutoff_at:dateParts(bounds.cutoff_at).stamp,quality_note:qualityNote,stores,records};
  const json=Buffer.from(JSON.stringify(bundle)),gzip=gzipSync(json,{level:9});
  const manifest={...bundle,stores:undefined,records:undefined,store_count:stores.length,record_count:records.length,day_count:new Set(records.map(r=>r.collected_at.slice(0,10))).size,bytes:gzip.length,sha256:createHash('sha256').update(gzip).digest('hex'),quality,method:'One queue_records representative per store / CST half-hour (the last frame in the slot); unknown values stay NULL, waits above '+WAIT_CAP_MINUTES+' minutes read as unknown. Local overlap takes precedence at runtime.'};
  await fs.mkdir(out,{recursive:true});
  await fs.writeFile(path.join(out,'queue-history-v1.json.gz'),gzip);
  await fs.writeFile(path.join(out,'queue-history-v1.manifest.json'),JSON.stringify(manifest,null,2)+'\n');
  console.log(JSON.stringify(manifest,null,2));
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url))main().catch(e=>{console.error(e.message);process.exitCode=1;});
