let analysisStoreMatches=[],analysisStoreActive=-1;

function analysisStoreIDs() {
  const selected=el('analysis-store').value;
  return [...new Set([...recordIDs(),...(state.analysisStores||state.records?.available_stores||state.records?.stores||[]).map(s=>String(s.id)),...(selected?[selected]:[])])];
}
function syncAnalysisStore(preferred='') {
  const select=el('analysis-store'),selected=preferred||select.value||String(state.records?.selected_store||'');
  const ids=analysisStoreIDs();
  if(selected&&!ids.includes(selected))ids.push(selected);
  const html='<option value="">选择门店</option>'+ids.map(id=>'<option value="'+escapeHTML(id)+'">'+escapeHTML(storeName(id))+'</option>').join('');
  if(select.innerHTML!==html)select.innerHTML=html;
  select.value=ids.includes(selected)?selected:'';
  if(el('analysis-store-popup').hidden)el('analysis-store-search').value=select.value?storeName(select.value):'';
}
function filterAnalysisStores(query='') {
  const terms=query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return analysisStoreIDs().filter(id=>{
    const s=state.stores.get(id)||{},haystack=[storeName(id),s.city,s.area,s.name_kana,id].filter(Boolean).join(' ').toLocaleLowerCase();
    return terms.every(term=>haystack.includes(term));
  });
}
function renderAnalysisStoreOptions(query='') {
  analysisStoreMatches=filterAnalysisStores(query);
  analysisStoreActive=analysisStoreMatches.indexOf(el('analysis-store').value);
  el('analysis-store-options').innerHTML=analysisStoreMatches.map((id,index)=>{
    const s=state.stores.get(id)||{},city=s.city||s.area||s.name_kana||'';
    return '<div id="analysis-store-option-'+index+'" class="store-option" role="option" aria-selected="'+(id===el('analysis-store').value)+'" data-store-index="'+index+'"><span>'+escapeHTML(storeName(id))+'</span>'+(city?'<small>'+escapeHTML(city)+'</small>':'')+'</div>';
  }).join('');
  el('analysis-store-empty').hidden=analysisStoreMatches.length>0;
  el('analysis-store-count').textContent=analysisStoreMatches.length+' 家门店';
  positionAnalysisStores();
  highlightAnalysisStore();
}
function positionAnalysisStores() {
  if(el('analysis-store-popup').hidden)return;
  const popup=el('analysis-store-popup'),bounds=el('analysis-store-search').getBoundingClientRect();
  const below=window.innerHeight-bounds.bottom-16,above=bounds.top-16;
  const upward=below<Math.min(300,Math.max(64,analysisStoreMatches.length*64+12))&&above>below;
  popup.classList.toggle('above',upward);
  popup.style.maxHeight=Math.max(64,Math.min(300,upward?above:below))+'px';
}
function highlightAnalysisStore() {
  const input=el('analysis-store-search');
  el('analysis-store-options').querySelectorAll('[role="option"]').forEach((node,index)=>node.classList.toggle('active',index===analysisStoreActive));
  if(analysisStoreActive>=0){
    const id='analysis-store-option-'+analysisStoreActive;
    input.setAttribute('aria-activedescendant',id);
    el(id)?.scrollIntoView({block:'nearest'});
  }else input.removeAttribute('aria-activedescendant');
}
function openAnalysisStores() {
  if(!el('analysis-store-popup').hidden)return;
  el('analysis-store-popup').hidden=false;
  el('analysis-store-search').setAttribute('aria-expanded','true');
  renderAnalysisStoreOptions();
}
function closeAnalysisStores() {
  el('analysis-store-popup').hidden=true;
  el('analysis-store-search').setAttribute('aria-expanded','false');
  el('analysis-store-search').removeAttribute('aria-activedescendant');
  const id=el('analysis-store').value;
  el('analysis-store-search').value=id?storeName(id):'';
}
async function selectAnalysisStore(id) {
  const changed=id!==el('analysis-store').value;
  syncAnalysisStore(id);closeAnalysisStores();
  if(changed){el('analysis-date').value='';await refreshRecords();}
}
function analysisStoreKeydown(event) {
  if(event.isComposing)return;
  if(event.key==='Escape'){
    if(!el('analysis-store-popup').hidden){event.preventDefault();event.stopPropagation();closeAnalysisStores();}
    return;
  }
  if(event.key==='Tab'){closeAnalysisStores();return;}
  if(!['ArrowDown','ArrowUp','Enter'].includes(event.key))return;
  if(event.key==='Enter'){
    if(el('analysis-store-popup').hidden)return;
    event.preventDefault();
    if(analysisStoreActive>=0)selectAnalysisStore(analysisStoreMatches[analysisStoreActive]);
    return;
  }
  event.preventDefault();openAnalysisStores();
  const count=analysisStoreMatches.length;
  if(count)analysisStoreActive=analysisStoreActive<0?(event.key==='ArrowDown'?0:count-1):(analysisStoreActive+(event.key==='ArrowDown'?1:-1)+count)%count;
  highlightAnalysisStore();
}
function initAnalysisStorePicker() {
  const input=el('analysis-store-search'),list=el('analysis-store-options'),picker=el('analysis-store-picker');
  input.addEventListener('focus',()=>{openAnalysisStores();input.select();});
  input.addEventListener('click',openAnalysisStores);
  input.addEventListener('input',()=>{openAnalysisStores();renderAnalysisStoreOptions(input.value);});
  input.addEventListener('keydown',analysisStoreKeydown);
  list.addEventListener('pointerdown',event=>event.preventDefault());
  list.addEventListener('click',event=>{const option=event.target.closest('[data-store-index]');if(option)selectAnalysisStore(analysisStoreMatches[Number(option.dataset.storeIndex)]);});
  picker.addEventListener('focusout',event=>{if(!picker.contains(event.relatedTarget))closeAnalysisStores();});
  document.addEventListener('pointerdown',event=>{if(!picker.contains(event.target))closeAnalysisStores();});
  window.addEventListener('resize',positionAnalysisStores);
}
