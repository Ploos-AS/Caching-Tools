import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const qualitySource = fs.readFileSync(new URL('../../cmd/caching-tools/web/field-quality.js', import.meta.url), 'utf8');
const sessionSource = fs.readFileSync(new URL('../../cmd/caching-tools/web/field-session.js', import.meta.url), 'utf8');

class FakeElement {
  constructor(tagName='div',document=null){this.tagName=tagName.toUpperCase();this.document=document;this.children=[];this.listeners=new Map();this.elements={};this.value='';this.checked=false;this.disabled=false;this.textContent='';this.id='';this.options=[];this.selectedOptions=[];}
  set innerHTML(value){this._innerHTML=value;for(const match of value.matchAll(/id="([^"]+)"/g)){const node=new FakeElement('input',this.document);node.id=match[1];this.document?.register(match[1],node);}}
  get innerHTML(){return this._innerHTML||'';}
  querySelector(selector){return selector.startsWith('#')?this.document?.querySelector(selector)||null:null;}
  append(...nodes){for(const node of nodes){this.children.push(node);if(node.tagName==='OPTION')this.options.push(node);}}
  replaceChildren(...nodes){this.children=[];this.options=[];this.append(...nodes);}
  addEventListener(type,listener){const list=this.listeners.get(type)||[];list.push(listener);this.listeners.set(type,list);}
  dispatchEvent(event){for(const listener of this.listeners.get(event.type)||[])listener.call(this,event);}
  insertAdjacentElement(){}
}
class FakeDocument {
  constructor(){this.nodes=new Map();}
  register(id,node=new FakeElement('div',this)){node.id=id;node.document=this;this.nodes.set(id,node);return node;}
  querySelector(selector){if(selector.startsWith('#'))return this.nodes.get(selector.slice(1))||null;if(selector==='footer')return this.nodes.get('footer')||null;return null;}
  createElement(tagName){return new FakeElement(tagName,this);}
}
class MemoryStorage {
  constructor(seed={}){this.values=new Map(Object.entries(seed));}
  getItem(key){return this.values.has(key)?this.values.get(key):null;}
  setItem(key,value){this.values.set(key,String(value));}
  removeItem(key){this.values.delete(key);}
}
class FullStorage extends MemoryStorage {
  setItem(){throw new Error('QuotaExceededError');}
}

function haversine(a,b){const rad=value=>value*Math.PI/180,dLat=rad(b.latitude-a.latitude),dLon=rad(b.longitude-a.longitude),lat1=rad(a.latitude),lat2=rad(b.latitude);const h=Math.sin(dLat/2)**2+Math.cos(lat1)*Math.cos(lat2)*Math.sin(dLon/2)**2;return 6371008.8*2*Math.atan2(Math.sqrt(h),Math.sqrt(1-h));}

function createHarness(storage=new MemoryStorage()){
  const document=new FakeDocument(),fieldSection=document.register('field-section'),clear=document.register('field-breadcrumb-clear',new FakeElement('button',document));
  fieldSection.querySelector=selector=>selector==='#field-breadcrumb-clear'?clear:document.querySelector(selector);fieldSection.append=()=>{};
  const fieldSessionStats=document.register('field-session-stats');fieldSessionStats.insertAdjacentElement=()=>{};document.register('footer');
  const fieldForm={elements:{latitude:{value:'59.9139'},longitude:{value:'10.7522'},'target-id':{selectedOptions:[{textContent:'Recovery target'}]}}};
  const liveBreadcrumbs=[];let started=0,overlayRenders=0;
  const window={localStorage:storage,refreshLocalMap:async()=>{},renderMapOverlays:()=>{overlayRenders+=1;}};
  const context=vm.createContext({document,window,localStorage:storage,fieldSection,fieldSessionStats,fieldForm,liveBreadcrumbs,fieldLiveStatus:{textContent:''},addBreadcrumb(position){liveBreadcrumbs.push({latitude:position.coords.latitude,longitude:position.coords.longitude,time:position.time});return true;},sessionStatistics(){return null;},breadcrumbDistanceMeters:haversine,breadcrumbGPX(){if(!liveBreadcrumbs.length)throw new Error('No points');return '<gpx/>';},startLiveNavigation(){started+=1;},escapeXML:value=>String(value),postJSON:async()=>({id:'wp'}),loadWaypoints:async()=>{},loadFieldTargets:async()=>{},Date,Math,Number,Promise,JSON,console});
  vm.runInContext(qualitySource,context,{filename:'field-quality.js'});
  document.querySelector('#field-min-distance').value='3';document.querySelector('#field-max-accuracy').value='50';document.querySelector('#field-export-simplify').checked=false;document.querySelector('#field-simplify-tolerance').value='5';
  vm.runInContext(sessionSource,context,{filename:'field-session.js'});
  return {context,document,storage,liveBreadcrumbs,counts:()=>({started,overlayRenders})};
}
function run(context,expression){return vm.runInContext(expression,context);}
function position(latitude,longitude,time){return {coords:{latitude,longitude,accuracy:5},time};}

test('accepted breadcrumb autosaves a versioned recovery snapshot',()=>{
  const {context,storage}=createHarness();
  assert.equal(run(context,`addBreadcrumb(${JSON.stringify(position(59.9139,10.7522,'2026-09-09T00:00:00.000Z'))})`),true);
  const snapshot=JSON.parse(storage.getItem('caching-tools.field-session.v1'));
  assert.equal(snapshot.version,1);assert.equal(snapshot.breadcrumbs.length,1);assert.equal(snapshot.markers.length,0);assert.equal(snapshot.paused,false);assert.equal(snapshot.segmentID,0);
});

test('pause, resume and marker mutations update recovery state',()=>{
  const {context,storage}=createHarness();
  run(context,`addBreadcrumb(${JSON.stringify(position(59.9139,10.7522,'2026-09-09T00:00:00.000Z'))})`);
  run(context,'pauseBreadcrumbRecording()');let snapshot=JSON.parse(storage.getItem('caching-tools.field-session.v1'));assert.equal(snapshot.paused,true);
  run(context,'resumeBreadcrumbRecording()');snapshot=JSON.parse(storage.getItem('caching-tools.field-session.v1'));assert.equal(snapshot.paused,false);assert.equal(snapshot.segmentID,1);
  run(context,`fieldMarkerType.value='cache'; addManualMarker()`);snapshot=JSON.parse(storage.getItem('caching-tools.field-session.v1'));assert.equal(snapshot.markers.length,1);assert.equal(snapshot.markers[0].type,'cache');
});

test('reload detects fresh recovery but does not automatically restart GPS or restore points',()=>{
  const seed={version:1,savedAt:new Date().toISOString(),breadcrumbs:[{latitude:59.9,longitude:10.7,time:'2026-09-09T00:00:00.000Z',recordingSegment:0}],markers:[{latitude:59.91,longitude:10.71,time:'2026-09-09T00:01:00.000Z',type:'note',note:'x',promotedWaypointID:''}],paused:true,segmentID:2};
  const storage=new MemoryStorage({'caching-tools.field-session.v1':JSON.stringify(seed)}),{context,liveBreadcrumbs,counts}=createHarness(storage);
  assert.equal(liveBreadcrumbs.length,0);assert.equal(run(context,'liveMarkers.length'),0);assert.equal(counts().started,0);assert.equal(run(context,'fieldSessionRestore.disabled'),false);assert.match(run(context,'fieldSessionRecoveryStatus.textContent'),/1 breadcrumb, 1 marker/);
});

test('recovery status presents a human-readable age',()=>{
  const {context}=createHarness();
  assert.equal(run(context,`formatRecoveryAge('2026-09-09T10:00:00.000Z', Date.parse('2026-09-09T10:12:00.000Z'))`),'12 minutes ago');
  assert.equal(run(context,`formatRecoveryAge('2026-09-09T08:00:00.000Z', Date.parse('2026-09-09T10:00:00.000Z'))`),'2 hours ago');
  assert.equal(run(context,`formatRecoveryAge('2026-09-07T10:00:00.000Z', Date.parse('2026-09-09T10:00:00.000Z'))`),'2 days ago');
});

test('recovery older than seven days is discarded and cannot be restored',()=>{
  const seed={version:1,savedAt:'2026-08-20T00:00:00.000Z',breadcrumbs:[{latitude:59.9,longitude:10.7,time:'2026-08-20T00:00:00.000Z',recordingSegment:0}],markers:[],paused:false,segmentID:0};
  const storage=new MemoryStorage({'caching-tools.field-session.v1':JSON.stringify(seed)}),{context,storage:actual}=createHarness(storage);
  assert.equal(run(context,'recoveredSession'),null);assert.equal(actual.getItem('caching-tools.field-session.v1'),null);assert.equal(run(context,'fieldSessionRestore.disabled'),true);assert.match(run(context,'fieldSessionRecoveryStatus.textContent'),/older than 7 days/);
});

test('explicit restore recovers breadcrumbs markers pause and segment without starting GPS',()=>{
  const seed={version:1,savedAt:new Date().toISOString(),breadcrumbs:[{latitude:59.9,longitude:10.7,time:'2026-09-09T00:00:00.000Z',recordingSegment:0}],markers:[{latitude:59.91,longitude:10.71,time:'2026-09-09T00:01:00.000Z',type:'note',note:'x',promotedWaypointID:''}],paused:true,segmentID:2};
  const storage=new MemoryStorage({'caching-tools.field-session.v1':JSON.stringify(seed)}),{context,liveBreadcrumbs,counts}=createHarness(storage);
  assert.equal(run(context,'restoreRecoveredSession()'),true);assert.equal(liveBreadcrumbs.length,1);assert.equal(run(context,'liveMarkers.length'),1);assert.equal(run(context,'breadcrumbRecordingPaused'),true);assert.equal(run(context,'recordingSegmentID'),2);assert.equal(counts().started,0);assert.equal(counts().overlayRenders,1);assert.match(run(context,'fieldRecordingStatus.textContent'),/Live GPS remains stopped/);
});

test('storage quota failure is non-fatal and visible to the user',()=>{
  const {context}=createHarness(new FullStorage());
  assert.equal(run(context,`addBreadcrumb(${JSON.stringify(position(59.9139,10.7522,'2026-09-09T00:00:00.000Z'))})`),true);
  assert.match(run(context,'fieldSessionRecoveryStatus.textContent'),/storage is full or unavailable/);
  assert.equal(run(context,'liveBreadcrumbs.length'),1);
});

test('oversized recovery snapshot is rejected before storage write',()=>{
  const {context}=createHarness();
  run(context,`liveMarkers.push({latitude:59,longitude:10,time:new Date().toISOString(),type:'note',note:'x'.repeat(sessionRecoveryMaxBytes + 1),promotedWaypointID:''})`);
  assert.equal(run(context,'persistFieldSession()'),false);
  assert.match(run(context,'fieldSessionRecoveryStatus.textContent'),/exceeded 2 MiB/);
});

test('discard removes recovery and a new live session starts clean',()=>{
  const seed={version:1,savedAt:new Date().toISOString(),breadcrumbs:[{latitude:59.9,longitude:10.7,time:'2026-09-09T00:00:00.000Z'}],markers:[],paused:false,segmentID:0};
  const storage=new MemoryStorage({'caching-tools.field-session.v1':JSON.stringify(seed)}),{context,storage:actual,counts}=createHarness(storage);
  run(context,'discardRecoveredSession()');assert.equal(actual.getItem('caching-tools.field-session.v1'),null);assert.equal(run(context,'fieldSessionRestore.disabled'),true);
  run(context,`liveBreadcrumbs.push({latitude:1,longitude:2,time:'2026-09-09T00:00:00.000Z'}); liveMarkers.push({latitude:1,longitude:2,time:'2026-09-09T00:00:00.000Z',type:'note'}); startLiveNavigation()`);
  assert.equal(run(context,'liveBreadcrumbs.length'),0);assert.equal(run(context,'liveMarkers.length'),0);assert.equal(counts().started,1);assert.equal(actual.getItem('caching-tools.field-session.v1'),null);
});

test('malformed recovery is discarded safely',()=>{
  const storage=new MemoryStorage({'caching-tools.field-session.v1':'{"version":99,"breadcrumbs":[]}'}),{context,storage:actual}=createHarness(storage);
  assert.equal(run(context,'recoveredSession'),null);assert.equal(actual.getItem('caching-tools.field-session.v1'),null);assert.equal(run(context,'fieldSessionRestore.disabled'),true);
});