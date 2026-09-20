import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import {gunzipSync} from 'node:zlib';
import test from 'node:test';

const source=fs.readFileSync(new URL('../internal/app/webui/store_select.js',import.meta.url),'utf8');
const bundle=JSON.parse(gunzipSync(fs.readFileSync(new URL('../internal/app/data/queue-history-v1.json.gz',import.meta.url))));

function fixture(stores) {
  const nodes=new Map();
  function el(id) {
    if(!nodes.has(id))nodes.set(id,{value:'',hidden:true,innerHTML:'',textContent:'',querySelectorAll:()=>[],removeAttribute(){},setAttribute(){}});
    return nodes.get(id);
  }
  const state={analysisStores:stores,stores:new Map(stores.map(store=>[String(store.id),store]))};
  const context=vm.createContext({state,el,recordIDs:()=>[],storeName:id=>state.stores.get(id)?.name||id,
    escapeHTML:value=>String(value).replace(/[&<>"']/g,char=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]))});
  vm.runInContext(source,context);
  return {el,run:code=>vm.runInContext(code,context)};
}

test('real bundled stores are searchable by city recorded only in area',()=>{
  const f=fixture(bundle.stores);
  for(const city of ['广州','深圳']) {
    const expected=bundle.stores.filter(store=>[store.name,store.city,store.area,String(store.id)].join(' ').includes(city)).map(store=>String(store.id));
    assert.ok(bundle.stores.some(store=>!store.city&&store.area.includes(city)),'regression must exercise empty city metadata');
    assert.deepEqual(Array.from(f.run(`filterAnalysisStores(${JSON.stringify(city)})`)),expected);
  }
  const store=bundle.stores.find(store=>!store.city&&store.area.includes('广州'));
  assert.equal(f.run(`filterAnalysisStores(${JSON.stringify(store.area+' '+store.name)}).join(',')`),String(store.id));
  f.run(`renderAnalysisStoreOptions(${JSON.stringify(store.area+' '+store.name)})`);
  assert.ok(f.el('analysis-store-options').innerHTML.includes('<small>'+store.area+'</small>'),'district fallback is not displayed');
});

test('area labels are escaped and explicit city labels retain priority',()=>{
  const f=fixture([{id:1,name:'区域店',city:'',area:'广州 <天河区>'},{id:2,name:'城市店',city:'深圳',area:'南山区'}]);
  f.run("renderAnalysisStoreOptions('广州')");
  assert.match(f.el('analysis-store-options').innerHTML,/<small>广州 &lt;天河区&gt;<\/small>/);
  f.run("renderAnalysisStoreOptions('南山')");
  assert.match(f.el('analysis-store-options').innerHTML,/<small>深圳<\/small>/);
  assert.equal(f.run("filterAnalysisStores('不存在').length"),0);
});
