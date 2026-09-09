import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source = fs.readFileSync(new URL('../../cmd/caching-tools/web/field-session-bundle.js', import.meta.url), 'utf8');

class FakeElement {
  constructor(tag='div', document=null) { this.tagName=tag.toUpperCase(); this.document=document; this.children=[]; this.listeners=new Map(); this.id=''; this.value=''; this.textContent=''; this.files=[]; this.href=''; this.download=''; }
  set innerHTML(value) { this._innerHTML=value; for (const match of value.matchAll(/id="([^"]+)"/g)) { const node=new FakeElement('input', this.document); node.id=match[1]; this.document?.register(match[1], node); } }
  get innerHTML() { return this._innerHTML || ''; }
  querySelector(selector) { return selector.startsWith('#') ? this.document?.querySelector(selector) || null : null; }
  append(...nodes) { this.children.push(...nodes); }
  addEventListener(type, fn) { const list=this.listeners.get(type)||[]; list.push(fn); this.listeners.set(type,list); }
  dispatchEvent(event) { for (const fn of this.listeners.get(event.type)||[]) fn.call(this,event); }
  click() { this.clicked=true; }
}
class FakeDocument {
  constructor() { this.nodes=new Map(); }
  register(id,node=new FakeElement('div',this)) { node.id=id; node.document=this; this.nodes.set(id,node); return node; }
  querySelector(selector) { return selector.startsWith('#') ? this.nodes.get(selector.slice(1)) || null : null; }
  createElement(tag) { return new FakeElement(tag,this); }
}

function validRecoveredPoint(item) { return item && Number.isFinite(Number(item.latitude)) && Number.isFinite(Number(item.longitude)) && typeof item.time === 'string'; }

function createHarness() {
  const document=new FakeDocument();
  document.register('field-session-controls');
  const liveBreadcrumbs=[{latitude:59.9,longitude:10.7,time:'2026-09-09T00:00:00.000Z',recordingSegment:0}];
  let liveMarkers=[{latitude:59.91,longitude:10.71,time:'2026-09-09T00:01:00.000Z',type:'note',note:'hello',promotedWaypointID:''}];
  let breadcrumbRecordingPaused=true;
  let recordingSegmentID=2;
  let startNewRecordingSegment=true;
  let renderRecordingCalls=0, renderMarkerCalls=0, renderStatsCalls=0, persistCalls=0, overlayCalls=0;
  const fieldRecordingStatus={textContent:''};
  const fieldForm={elements:{'target-id':{selectedOptions:[{textContent:'Portable target'}]}}};
  const footer=document.register('footer');
  const context=vm.createContext({
    document,
    window:{renderMapOverlays(){overlayCalls+=1;}},
    liveBreadcrumbs,
    liveMarkers,
    breadcrumbRecordingPaused,
    recordingSegmentID,
    startNewRecordingSegment,
    validRecoveredPoint,
    renderRecordingStatus(){renderRecordingCalls+=1;},
    renderMarkers(){renderMarkerCalls+=1;},
    renderSessionStatistics(){renderStatsCalls+=1;},
    persistFieldSession(){persistCalls+=1;return true;},
    fieldRecordingStatus,
    fieldForm,
    Date, Math, Number, JSON, Promise, Map,
    Blob: class { constructor(parts,options){this.parts=parts;this.options=options;} },
    URL:{createObjectURL(){return 'blob:test';},revokeObjectURL(){}},
    console
  });
  vm.runInContext(source,context,{filename:'field-session-bundle.js'});
  return {context,document,footer,liveBreadcrumbs,counts:()=>({renderRecordingCalls,renderMarkerCalls,renderStatsCalls,persistCalls,overlayCalls})};
}
function run(context, expression) { return vm.runInContext(expression, context); }

test('portable bundle exports current v2 schema and metadata', () => {
  const {context,footer}=createHarness();
  const bundle=run(context,'portableFieldSessionBundle()');
  assert.equal(bundle.kind,'caching-tools.field-session');
  assert.equal(bundle.version,2);
  assert.equal(bundle.session.breadcrumbs.length,1);
  assert.equal(bundle.session.markers.length,1);
  assert.equal(bundle.session.recording.paused,true);
  assert.equal(bundle.session.recording.segmentID,2);
  assert.equal(bundle.metadata.target,'Portable target');
  assert.equal(bundle.metadata.generator,'Caching Tools M1.54');
  assert.equal(footer.textContent,'Caching Tools M1.54');
});

test('portable v2 import replaces session state without starting GPS', () => {
  const {context,liveBreadcrumbs,counts}=createHarness();
  const imported={kind:'caching-tools.field-session',version:2,session:{breadcrumbs:[{latitude:58,longitude:8,time:'2026-09-08T12:00:00.000Z',recordingSegment:4}],markers:[{latitude:58.1,longitude:8.1,time:'2026-09-08T12:01:00.000Z',type:'cache',note:'x',promotedWaypointID:''}],recording:{paused:false,segmentID:4}}};
  const result=run(context,`importPortableFieldSessionBundle(${JSON.stringify(imported)})`);
  assert.equal(liveBreadcrumbs.length,1);
  assert.equal(liveBreadcrumbs[0].latitude,58);
  assert.equal(run(context,'liveMarkers.length'),1);
  assert.equal(run(context,'liveMarkers[0].type'),'cache');
  assert.equal(run(context,'breadcrumbRecordingPaused'),false);
  assert.equal(run(context,'recordingSegmentID'),4);
  assert.equal(run(context,'startNewRecordingSegment'),false);
  assert.equal(result.bundleVersion,2);
  assert.equal(result.migratedFromVersion,null);
  assert.deepEqual(counts(),{renderRecordingCalls:1,renderMarkerCalls:1,renderStatsCalls:1,persistCalls:1,overlayCalls:1});
  assert.match(run(context,'fieldRecordingStatus.textContent'),/Live GPS remains stopped until explicitly started/);
});

test('v1 bundles migrate deterministically to v2 before import', () => {
  const {context}=createHarness();
  const legacy={kind:'caching-tools.field-session',version:1,session:{breadcrumbs:[{latitude:58,longitude:8,time:'2026-09-08T12:00:00.000Z',recordingSegment:3}],markers:[],paused:true,segmentID:3},metadata:{target:'Legacy target'}};
  const migrated=run(context,`migratePortableFieldSessionBundle(${JSON.stringify(legacy)})`);
  assert.equal(migrated.version,2);
  assert.equal(migrated.session.recording.paused,true);
  assert.equal(migrated.session.recording.segmentID,3);
  assert.equal(migrated.metadata.target,'Legacy target');
  assert.equal(migrated.metadata.migratedFromVersion,1);
  const imported=run(context,`importPortableFieldSessionBundle(${JSON.stringify(legacy)})`);
  assert.equal(imported.bundleVersion,2);
  assert.equal(imported.migratedFromVersion,1);
  assert.equal(run(context,'breadcrumbRecordingPaused'),true);
  assert.equal(run(context,'recordingSegmentID'),3);
  assert.match(run(context,'fieldSessionBundleStatus.textContent'),/Migrated bundle v1 to v2/);
});

test('future bundle versions are rejected with explicit compatibility error', () => {
  const {context}=createHarness();
  const future={kind:'caching-tools.field-session',version:99,session:{breadcrumbs:[],markers:[],recording:{paused:false,segmentID:0}}};
  assert.throws(()=>run(context,`importPortableFieldSessionBundle(${JSON.stringify(future)})`),/version 99 is newer than supported version 2/);
});

test('invalid kind, unsupported old version and invalid points are rejected', () => {
  const {context}=createHarness();
  assert.equal(run(context,`normalizePortableFieldSessionBundle({kind:'wrong',version:2,session:{breadcrumbs:[],markers:[],recording:{paused:false,segmentID:0}}})`),null);
  assert.equal(run(context,`normalizePortableFieldSessionBundle({kind:'caching-tools.field-session',version:0,session:{breadcrumbs:[],markers:[]}})`),null);
  assert.equal(run(context,`normalizePortableFieldSessionBundle({kind:'caching-tools.field-session',version:2,session:{breadcrumbs:[{latitude:'bad',longitude:1,time:'x'}],markers:[],recording:{paused:false,segmentID:0}}})`),null);
  assert.throws(()=>run(context,`importPortableFieldSessionBundle({kind:'wrong',version:2,session:{breadcrumbs:[],markers:[],recording:{paused:false,segmentID:0}}})`),/Invalid or unsupported/);
});

test('bundle import file enforces 4 MiB before reading content', async () => {
  const {context}=createHarness();
  await assert.rejects(async()=>run(context,`importPortableFieldSessionFile({size:4194305,text:async()=>''})`),/exceeds 4 MiB/);
});
