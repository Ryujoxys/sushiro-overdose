// Builds the public queue dataset published on the `data` branch: stores, half-hour patterns,
// daily half-hour representatives and the latest public status. Read-only: the database is only
// queried with SELECT, and nothing personal (accounts, tickets, credentials) exists in the source.
import fs from 'node:fs/promises';
import path from 'node:path';
import {gunzipSync} from 'node:zlib';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {WAIT_CAP_MINUTES,dateParts,openDatabase,readPublicRecords} from './export-history-bundle.mjs';

export const SCHEMA='sushiro-public-data';
export const DATE_TYPES=['all','weekday','weekend','holiday','workday'];
const BUNDLE=fileURLToPath(new URL('../internal/app/data/queue-history-v1.json.gz',import.meta.url));

// Same interpolation as the app (queueQuantile), so published patterns match its charts.
export function quantile(values,q) {
  if(!values.length)return null;
  const sorted=[...values].sort((a,b)=>a-b);
  if(sorted.length===1)return sorted[0];
  const pos=q*(sorted.length-1),lo=Math.floor(pos),hi=Math.ceil(pos);
  return lo===hi?sorted[lo]:sorted[lo]*(1-(pos-lo))+sorted[hi]*(pos-lo);
}
const round1=value=>value===null?null:Math.round(value*10)/10;
export function halfHour(stamp) {
  const time=dateParts(stamp).time;
  return time.slice(0,3)+(Number(time.slice(3))<30?'00':'30');
}

// One row per half-hour per date type. Only open-store values count; unknown waits and
// missing called numbers are left out, never treated as zero.
export function buildPatterns(records) {
  const groups=Object.fromEntries(DATE_TYPES.map(type=>[type,new Map()]));
  for(const row of records){
    if(row.store_status!=='OPEN')continue;
    const time=halfHour(row.collected_at);
    for(const type of new Set(['all',row.date_type])){
      if(!groups[type])continue;
      if(!groups[type].has(time))groups[type].set(time,{waits:[],called:[]});
      const bucket=groups[type].get(time);
      if(row.wait_minutes!==null)bucket.waits.push(row.wait_minutes);
      if(row.display_called_no!==null)bucket.called.push(row.display_called_no);
    }
  }
  return Object.fromEntries(DATE_TYPES.map(type=>[type,[...groups[type]].sort(([a],[b])=>a.localeCompare(b)).map(([time,{waits,called}])=>({
    time,
    wait_days:waits.length,wait_p50:round1(quantile(waits,.5)),wait_p80:round1(quantile(waits,.8)),
    called_days:called.length,called_p20:round1(quantile(called,.2)),called_p50:round1(quantile(called,.5)),called_p80:round1(quantile(called,.8)),
  }))]));
}

function summarizeStore(store,records) {
  const usable=records.filter(r=>r.store_status==='OPEN'&&(r.wait_minutes!==null||r.display_called_no!==null));
  const days=rows=>new Set(rows.map(r=>r.collected_at.slice(0,10))).size;
  return {
    ...store,
    first_at:records[0]?.collected_at||null,last_at:records.at(-1)?.collected_at||null,
    days:days(usable),
    wait_days:days(usable.filter(r=>r.wait_minutes!==null)),
    called_days:days(usable.filter(r=>r.display_called_no!==null)),
  };
}

const json=value=>JSON.stringify(value)+'\n';
const sha256=text=>createHash('sha256').update(text).digest('hex');

// Returns every published file as {relative path: text}; meta.json lists the others with hashes.
export function buildDataset({source,stores,records,extras=[],latest=[],calendar={},collector=null,quality={},qualityNote='',cutoffAt,generatedAt,windowDays}) {
  // extras[i] (online_open, group_queues_count, wait_over_cap) belongs to records[i]; the bundle has none.
  // Waits above the official display cap are published as the cap ("180 = 180 minutes or more"),
  // not dropped: dropping them would hide exactly the busiest slots and bias plans toward short waits.
  const byStore=new Map(stores.map(s=>[s.id,[]]));
  records.forEach((row,i)=>{
    const {wait_over_cap,...extra}=extras[i]||{};
    byStore.get(row.store_id)?.push({...row,...extra,wait_minutes:wait_over_cap?WAIT_CAP_MINUTES:row.wait_minutes});
  });
  const files=new Map(),summaries=[];
  for(const store of stores){
    const rows=byStore.get(store.id).sort((a,b)=>a.collected_at.localeCompare(b.collected_at));
    if(!rows.length)continue;
    const summary=summarizeStore(store,rows);
    summaries.push(summary);
    files.set('patterns/'+store.id+'.json',json({store:{id:store.id,name:store.name,city:store.city,area:store.area},first_at:summary.first_at,last_at:summary.last_at,date_types:buildPatterns(rows)}));
    files.set('daily/'+store.id+'.json',json({store_id:store.id,fields:['collected_at','date_type','store_status','wait_minutes','display_called_no','group_queues_count','online_open'],rows:rows.map(r=>[r.collected_at,r.date_type,r.store_status,r.wait_minutes,r.display_called_no,r.group_queues_count??null,r.online_open??null])}));
  }
  files.set('stores.json',json(summaries));
  files.set('latest.json',json({source:source==='turso'?'store_latest':'none',stores:latest}));
  files.set('README.md',[
    '# Sushiro Overdose 公开排队数据',
    '',
    '寿司郎门店公开排队快照的半小时代表值与统计，由 `.github/workflows/public-data.yml` 定时从维护者数据库只读导出。',
    '不含任何账号、手机号、票号或凭证。格式说明见主分支 `docs/public-data.md`。',
    '',
    '- `meta.json`：版本、来源、时间范围、覆盖统计、采集器状态、节假日表和各文件 SHA-256',
    '- `stores.json`：门店与覆盖天数',
    '- `latest.json`：每店最近一次公开状态（不是实时）',
    '- `patterns/<门店ID>.json`：按日期类型的半小时规律（等待 P50/P80、叫号 P20/P50/P80）',
    '- `daily/<门店ID>.json`：每天每半小时的代表值',
    '',
  ].join('\n'));
  const dates=new Set(records.map(r=>r.collected_at.slice(0,10)));
  const meta={
    schema:SCHEMA,version:1,source,generated_at:generatedAt,timezone:'Asia/Shanghai',bucket_minutes:30,
    window_days:windowDays,first_at:records[0]?.collected_at||null,cutoff_at:cutoffAt,
    store_count:summaries.length,record_count:records.length,day_count:dates.size,
    quality,quality_note:qualityNote,collector,calendar,
    wait_cap_minutes:WAIT_CAP_MINUTES,
    method:'One representative per store / CST half-hour (the last frame in the slot). Patterns use open-store values only: wait P50/P80 and called-number P20/P50/P80 across days, linear interpolation. Waits at or above wait_cap_minutes mean "that long or longer". Unknown values stay null and are never filled.',
    files:Object.fromEntries([...files].sort(([a],[b])=>a.localeCompare(b)).map(([name,text])=>[name,{bytes:Buffer.byteLength(text),sha256:sha256(text)}])),
  };
  files.set('meta.json',JSON.stringify(meta,null,1)+'\n');
  return files;
}

const minusDays=(day,n)=>new Date(Date.parse(day+'T00:00:00Z')-n*86400000).toISOString().slice(0,10);

async function fromBundle(windowDays) {
  const bundle=JSON.parse(gunzipSync(await fs.readFile(BUNDLE)).toString('utf8'));
  const since=minusDays(dateParts(bundle.cutoff_at).date,windowDays-1);
  return {
    source:'bundle',stores:bundle.stores,records:bundle.records.filter(r=>r.collected_at.slice(0,10)>=since),
    cutoffAt:bundle.cutoff_at,quality:{bundle_id:bundle.id},qualityNote:bundle.quality_note||'',
    collector:null,latest:[],calendar:{},
  };
}

async function fromDatabase(query,windowDays) {
  const [bounds]=await query('SELECT MAX(slot) AS last_slot FROM queue_records');
  if(!bounds?.last_slot)throw new Error('No public records available');
  const since=minusDays(bounds.last_slot.slice(0,10),windowDays-1);
  const data=await readPublicRecords(query,{since,withExtras:true});
  const keep=data.records.map(r=>r.collected_at.slice(0,10)>=since);
  const records=data.records.filter((_,i)=>keep[i]),extras=data.extras.filter((_,i)=>keep[i]);
  const extra=new Map((await query('SELECT store_id,address,latitude,longitude,is_active FROM store_dimension')).map(r=>[Number(r.store_id),r]));
  const stores=data.stores.map(s=>{const e=extra.get(s.id)||{};return {...s,address:e.address||'',latitude:e.latitude===null||e.latitude===undefined?null:Number(e.latitude),longitude:e.longitude===null||e.longitude===undefined?null:Number(e.longitude),active:Number(e.is_active)!==0};});
  const latest=(await query('SELECT store_id,collected_at,store_status,net_ticket_status,online_open,wait_minutes,group_queues_count,display_called_no FROM store_latest ORDER BY store_id')).map(r=>({
    store_id:Number(r.store_id),collected_at:dateParts(r.collected_at).stamp,store_status:String(r.store_status||'UNKNOWN').toUpperCase(),
    online_open:Number(r.online_open)===1,net_ticket_status:r.net_ticket_status||'',
    wait_minutes:r.wait_minutes===null?null:Math.min(Number(r.wait_minutes),WAIT_CAP_MINUTES),group_queues_count:r.group_queues_count===null?null:Number(r.group_queues_count),
    display_called_no:Number(r.display_called_no)>0?Number(r.display_called_no):null,
  }));
  // Collector health tells readers whether the public interface was still being read; errors are not published.
  const runs=await query("SELECT started_at,finished_at,ok,records_written FROM collector_runs WHERE run_id NOT LIKE 'maintenance-%' ORDER BY started_at DESC LIMIT 96");
  const stamp=v=>v?dateParts(v).stamp:null,lastOK=runs.find(r=>Number(r.ok)===1);
  const collector={last_run_at:stamp(runs[0]?.started_at),last_ok_at:stamp(lastOK?.finished_at||lastOK?.started_at),recent_runs:runs.length,recent_ok:runs.filter(r=>Number(r.ok)===1).length};
  const calendar=Object.fromEntries((await query('SELECT date_key,date_type FROM holiday_calendar WHERE date_key >= ? ORDER BY date_key',[since])).filter(r=>['workday','holiday'].includes(r.date_type)).map(r=>[r.date_key,r.date_type]));
  return {source:'turso',stores,records,extras,cutoffAt:dateParts(data.bounds.cutoff_at).stamp,quality:data.quality,qualityNote:data.qualityNote,collector,latest,calendar};
}

async function writeDataset(out,files) {
  const existing=await fs.readdir(out).catch(()=>null);
  if(existing?.length){
    const meta=JSON.parse(await fs.readFile(path.join(out,'meta.json'),'utf8').catch(()=>'{}'));
    if(meta.schema!==SCHEMA)throw new Error('Output directory is not empty and is not a previous public dataset: '+out);
    for(const name of existing)await fs.rm(path.join(out,name),{recursive:true,force:true});
  }
  for(const [name,text] of files){
    await fs.mkdir(path.dirname(path.join(out,name)),{recursive:true});
    await fs.writeFile(path.join(out,name),text);
  }
}

async function main() {
  const args=process.argv.slice(2),option=name=>{const i=args.indexOf(name);return i<0?'':args[i+1]||'';};
  const out=option('--out');
  if(!out)throw new Error('Usage: node scripts/export-public-data.mjs --out DIR [--source turso|bundle] [--days 180] [--config read-only.json]');
  const windowDays=Number(option('--days')||180);
  if(!Number.isInteger(windowDays)||windowDays<1||windowDays>730)throw new Error('--days must be 1..730');
  let credentials={};
  if(option('--config'))credentials=JSON.parse(await fs.readFile(option('--config'),'utf8'));
  const url=process.env.SUSHIRO_HISTORY_DATABASE_URL||credentials.database_url,token=process.env.SUSHIRO_HISTORY_DATABASE_TOKEN||credentials.auth_token;
  const source=option('--source')||(url&&token?'turso':'');
  if(!['turso','bundle'].includes(source))throw new Error('No database credentials. Set SUSHIRO_HISTORY_DATABASE_URL / SUSHIRO_HISTORY_DATABASE_TOKEN, or pass --source bundle to publish the bundled history.');
  const data=source==='turso'?await fromDatabase(openDatabase(url,token),windowDays):await fromBundle(windowDays);
  const files=buildDataset({...data,windowDays,generatedAt:new Date().toISOString()});
  await writeDataset(out,files);
  const meta=JSON.parse(files.get('meta.json'));
  console.log(JSON.stringify({source:meta.source,first_at:meta.first_at,cutoff_at:meta.cutoff_at,stores:meta.store_count,records:meta.record_count,days:meta.day_count,collector:meta.collector,files:files.size},null,2));
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url))main().catch(e=>{console.error(e.message);process.exitCode=1;});
