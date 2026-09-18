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
export function normalizeSample(row, source, calendar) {
  const d=dateParts(row.collected_at||row.updated_at),store=Number(row.store_id);
  if(!Number.isInteger(store)||store<=0)throw new Error('Invalid store ID');
  if(source==='daily'&&(Number(row.sample_count)!==1||row.snapshot_date!==d.date||row.time_bucket!==sampleKey({store_id:store,collected_at:d.stamp}).slice(-5)))throw new Error('Daily rows must be single half-hour representatives');
  const raw=source==='daily'?row.wait_typical_minutes:row.wait_minutes;
  let wait=raw===null||raw===undefined?null:Number(raw);
  const cap=Number(row.wait_time_cap)>0?Number(row.wait_time_cap):180;
  if(wait!==null&&(!Number.isInteger(wait)||wait<0||wait>cap||Number(row.dq_anomaly)>0))wait=null;
  const calledRaw=source==='daily'?row.called_no_typical:row.display_called_no;
  let called=calledRaw===null||calledRaw===undefined?null:Number(calledRaw);
  if(source==='daily'&&Number(row.called_sample_count)>1)throw new Error('Called numbers must be daily representatives, not aggregate quantiles');
  if(called!==null&&(!Number.isSafeInteger(called)||called<=0||called>2147483647||(source==='daily'&&Number(row.called_sample_count)!==1)))called=null;
  const override=calendar.get(d.date),minutes=Number(d.time.slice(0,2))*60+Number(d.time.slice(3));
  const weekend=d.weekday===6||(d.weekday===5&&minutes>=990)||(d.weekday===0&&minutes<1320);
  return {store_id:store,collected_at:d.stamp,date_type:override||(weekend?'weekend':'weekday'),store_status:source==='daily'?(Number(row.open_count)===1?'OPEN':'CLOSED'):String(row.store_status||'UNKNOWN').toUpperCase(),wait_minutes:wait,display_called_no:called};
}

async function main() {
  const args=process.argv.slice(2),option=name=>{const i=args.indexOf(name);return i<0?'':args[i+1]||'';};
  const config=option('--config'),out=option('--out')||'internal/app/data';
  let credentials={};
  if(config)credentials=JSON.parse(await fs.readFile(config,'utf8'));
  const url=process.env.SUSHIRO_HISTORY_DATABASE_URL||credentials.database_url;
  const token=process.env.SUSHIRO_HISTORY_DATABASE_TOKEN||credentials.auth_token;
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
  const [bounds]=await query('SELECT MIN(collected_at) AS first_at, MAX(collected_at) AS cutoff_at, MAX(id) AS max_id FROM queue_snapshots');
  if(!bounds?.first_at||!bounds?.cutoff_at)throw new Error('No public snapshots available');
  const [dailyBounds]=await query('SELECT MIN(snapshot_date) AS first_date FROM daily_store_bucket_rollups');
  const dimensions=await query('SELECT store_id, name, city, area FROM store_dimension ORDER BY store_id');
  const calendar=new Map((await query('SELECT date_key,date_type FROM holiday_calendar')).filter(r=>['workday','holiday'].includes(r.date_type)).map(r=>[r.date_key,r.date_type]));
  const byID=new Map(dimensions.map(r=>[Number(r.store_id),{id:Number(r.store_id),name:r.name,city:r.city||'',area:r.area||''}]));
  const rows=new Map(),quality={daily_rows:0,raw_representatives:0,duplicate_keys:0,unknown_waits:0,non_open:0,orphans:0};
  function accept(raw,source){
    const row=normalizeSample(raw,source,calendar);
    if(Date.parse(row.collected_at)>Date.parse(bounds.cutoff_at))throw new Error('Record beyond fixed cutoff');
    if(!byID.has(row.store_id)){quality.orphans++;return;}
    const key=sampleKey(row);if(rows.has(key))quality.duplicate_keys++;
    rows.set(key,row);
    if(source==='daily')quality.daily_rows++;else quality.raw_representatives++;
  }
  const rawStart=dateParts(bounds.first_at).date,rawEnd=dateParts(bounds.cutoff_at).date;
  const plusDays=(day,n)=>new Date(Date.parse(day+'T00:00:00Z')+n*86400000).toISOString().slice(0,10);
  if(dailyBounds?.first_date){
    for(let start=dailyBounds.first_date;start<rawStart;start=plusDays(start,7)){
      const end=[plusDays(start,7),rawStart].sort()[0];
      const batch=await query('SELECT snapshot_date,store_id,time_bucket,sample_count,open_count,wait_typical_minutes,called_sample_count,called_no_typical,updated_at FROM daily_store_bucket_rollups WHERE snapshot_date >= ? AND snapshot_date < ? ORDER BY snapshot_date,store_id,time_bucket',[start,end]);
      batch.forEach(r=>accept(r,'daily'));console.log('Archived dates '+start+'..'+end+': '+batch.length+' representatives');
    }
  }
  for(let start=rawStart;start<=rawEnd;start=plusDays(start,7)){
    const end=plusDays(start,7);
    const batch=await query(`WITH ranked AS (
      SELECT store_id,collected_at,wait_minutes,display_called_no,store_status,wait_time_cap,dq_anomaly,
        ROW_NUMBER() OVER (PARTITION BY store_id,substr(collected_at,1,13),CAST(substr(collected_at,15,2) AS INTEGER)/30
          ORDER BY collected_at DESC,CASE dq_source WHEN 'store_detail' THEN 1 ELSE 0 END DESC,id DESC) AS rank
      FROM queue_snapshots WHERE collected_at >= ? AND collected_at < ? AND collected_at <= ? AND id <= CAST(? AS INTEGER)
    ) SELECT store_id,collected_at,wait_minutes,display_called_no,store_status,wait_time_cap,dq_anomaly FROM ranked WHERE rank=1 ORDER BY collected_at,store_id`,[start+'T00:00:00+08:00',end+'T00:00:00+08:00',bounds.cutoff_at,bounds.max_id]);
    batch.forEach(r=>accept(r,'raw'));console.log('Snapshot dates '+start+'..'+end+': '+batch.length+' representatives');
  }
  const records=[...rows.values()].sort((a,b)=>a.collected_at.localeCompare(b.collected_at)||a.store_id-b.store_id);
  if(!records.length||quality.orphans||quality.duplicate_keys)throw new Error('Empty export, orphan stores or duplicate half-hour keys; inspect before packaging');
  const used=new Set(records.map(r=>r.store_id));
  const stores=[...byID.values()].filter(s=>used.has(s.id));
  quality.unknown_waits=records.filter(r=>r.wait_minutes===null).length;
  quality.non_open=records.filter(r=>r.store_status!=='OPEN').length;
  quality.usable_waits=records.filter(r=>r.store_status==='OPEN'&&r.wait_minutes!==null).length;
  quality.valid_zero_waits=records.filter(r=>r.store_status==='OPEN'&&r.wait_minutes===0).length;
  quality.usable_called_numbers=records.filter(r=>r.store_status==='OPEN'&&r.display_called_no!==null).length;
  quality.unknown_called_numbers=records.filter(r=>r.display_called_no===null).length;
  const qualityNote=quality.daily_rows?rawStart+' 前的归档记录缺少部分零等待值，早期曲线可能偏高。':'';
  const bundle={version:1,id:'public-history-'+rawEnd,timezone:'Asia/Shanghai',bucket_minutes:30,source:'maintainer_snapshot',exported_at:new Date().toISOString(),first_at:records[0].collected_at,cutoff_at:dateParts(bounds.cutoff_at).stamp,quality_note:qualityNote,stores,records};
  const json=Buffer.from(JSON.stringify(bundle)),gzip=gzipSync(json,{level:9});
  const manifest={...bundle,stores:undefined,records:undefined,store_count:stores.length,record_count:records.length,day_count:new Set(records.map(r=>r.collected_at.slice(0,10))).size,bytes:gzip.length,sha256:createHash('sha256').update(gzip).digest('hex'),quality,method:'Latest representative per store / CST date / half-hour; raw snapshots preferred, archived daily representatives retain NULL as unknown. Local overlap takes precedence at runtime.'};
  await fs.mkdir(out,{recursive:true});
  await fs.writeFile(path.join(out,'queue-history-v1.json.gz'),gzip);
  await fs.writeFile(path.join(out,'queue-history-v1.manifest.json'),JSON.stringify(manifest,null,2)+'\n');
  console.log(JSON.stringify(manifest,null,2));
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url))main().catch(e=>{console.error(e.message);process.exitCode=1;});
