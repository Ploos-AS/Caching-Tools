import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source = fs.readFileSync(new URL('../../cmd/caching-tools/web/map-overlays.js', import.meta.url), 'utf8');

class FakeElement {
  constructor(tag = 'div') { this.tagName=tag.toUpperCase();this.children=[];this.attributes=new Map();this.listeners=new Map();this.dataset={};this.id='';this.checked=false;this.value='';this.textContent='';this.elements={};this.parentNode=null;this.className='';this.innerHTML=''; }
  setAttribute(name,value){this.attributes.set(name,String(value));if(name==='id')this.id=String(value);}
  getAttribute(name){return this.attributes.get(name)??null;}
  addEventListener(type,fn){const list=this.listeners.get(type)||[];list.push(fn);this.listeners.set(type,list);}
  dispatchEvent(event){for(const fn of this.listeners.get(event.type)||[])fn.call(this,event);}
  append(...nodes){for(const node of nodes){node.parentNode=this;this.children.push(node);}}
  replaceChildren(...nodes){this.children=[];this.append(...nodes);}
  querySelector(selector){if(selector==='#map-overlay-layer')return this.children.find(node=>node.id==='map-overlay-layer')||null;return null;}
  closest(selector){return selector==='.map-frame'?this._closestMapFrame||null:null;}
  insertAdjacentElement(_where,node){this._inserted=node;}
}
class FakeDocument {
  constructor(){this.byId=new Map();this.listeners=new Map();}
  register(id,node=new FakeElement()){node.id=id;this.byId.set(id,node);return node;}
  querySelector(selector){return selector.startsWith('#')?this.byId.get(selector.slice(1))||null:null;}
  createElement(tag){return new FakeElement(tag);}
  createElementNS(_ns,tag){return new FakeElement(tag);}
  addEventListener(type,fn){const list=this.listeners.get(type)||[];list.push(fn);this.listeners.set(type,list);}
  dispatchEvent(event){for(const fn of this.listeners.get(event.type)||[])fn.call(this,event);}
}

function createHarness({waypoints=[],navigationResult=null,breadcrumbs=[],markers=[]}={}) {
  const document=new FakeDocument(),svg=new FakeElement('svg'),localMap=document.register('local-map',svg);localMap._closestMapFrame=new FakeElement('div');
  for(const [id,checked] of [['map-overlay-labels',true],['map-overlay-rings',false],['map-overlay-guidance',true],['map-overlay-path-guidance',true],['map-overlay-live-session',true],['map-overlay-follow',false]]){const node=document.register(id);node.checked=checked;}
  const radius=document.register('map-overlay-radius');radius.value='100';document.register('map-overlay-refresh');document.register('map-overlay-fit-session');document.register('map-status');
  const fieldForm=document.register('field-navigation-form');fieldForm.elements={latitude:{value:'59.9000'},longitude:{value:'10.7000'},'arrival-radius':{value:'20'}};
  const project=(latitude,longitude)=>({x:longitude*100,y:latitude*-100});
  const centered=[],fitted=[];
  const window={cachingToolsMap:{svg,project,centerViewOn:point=>centered.push(point),fitProjectedPoints:points=>{fitted.push(points);return points.length>0;}}};
  const requestJSON=async endpoint=>{if(endpoint==='/api/waypoints')return waypoints;throw new Error(`unexpected endpoint ${endpoint}`);};
  const navigateCalls=[];const navigateField=async(latitude,longitude)=>{navigateCalls.push([latitude,longitude]);if(navigationResult instanceof Error)throw navigationResult;return navigationResult;};
  const addBreadcrumb=position=>{breadcrumbs.push({latitude:position.coords.latitude,longitude:position.coords.longitude,recordingSegment:position.segment??0});return true;};
  const renderMarkers=()=>{};
  const context=vm.createContext({document,window,requestJSON,navigateField,addBreadcrumb,renderMarkers,liveBreadcrumbs:breadcrumbs,liveMarkers:markers,console,Promise,Math,Number,String,Array});
  vm.runInContext(source,context,{filename:'map-overlays.js'});
  return {context,document,window,svg,navigateCalls,breadcrumbs,markers,centered,fitted};
}
async function flushAsync(){await Promise.resolve();await Promise.resolve();await new Promise(resolve=>setImmediate(resolve));}
function classes(layer){return layer.children.map(node=>node.getAttribute('class')||'');}
function run(context,expression){return vm.runInContext(expression,context);}

test('waypoint selection renders current-position guidance and arrival ring',async()=>{const waypoints=[{id:'wp-1',name:'Target',point:{latitude:60,longitude:10.8}}];const{document,svg}=createHarness({waypoints});await flushAsync();document.dispatchEvent({type:'caching-tools:map-select',detail:{kind:'waypoint',id:'wp-1',name:'Target'}});const rendered=classes(svg.querySelector('#map-overlay-layer'));assert.ok(rendered.includes('map-overlay-position'));assert.ok(rendered.includes('map-overlay-guidance'));assert.ok(rendered.includes('map-overlay-arrival'));assert.ok(rendered.includes('map-overlay-label'));});

test('wrapped navigation renders route cross-track, next-point and status overlays',async()=>{const result={kind:'route',id:'path-1',from:{latitude:59.9,longitude:10.7},target:{latitude:59.91,longitude:10.71},cross_track_m:42,progress:{next_point:{latitude:59.92,longitude:10.72},remaining_m:1250,forward_bearing_deg:87},guidance:{status:'off-route',arrival_radius_m:20}};const{context,svg,navigateCalls}=createHarness({navigationResult:result});await flushAsync();const returned=await run(context,`navigateField(59.9, 10.7)`);assert.equal(returned.id,'path-1');assert.deepEqual(navigateCalls,[[59.9,10.7]]);const layer=svg.querySelector('#map-overlay-layer'),rendered=classes(layer);assert.ok(rendered.some(value=>value.includes('map-overlay-cross-track')&&value.includes('map-overlay-status-off-route')));assert.ok(rendered.includes('map-overlay-nearest'));assert.ok(rendered.includes('map-overlay-forward'));assert.ok(rendered.includes('map-overlay-next'));assert.ok(rendered.includes('map-overlay-next-arrival'));const label=layer.children.find(node=>(node.getAttribute('class')||'').includes('map-overlay-status-label'));assert.match(label.textContent,/off-route · 42 m off · 1\.25 km left · 87°/);});

test('live breadcrumbs render as separate recording segments with visible pause gap',async()=>{const breadcrumbs=[{latitude:59.90,longitude:10.70,recordingSegment:0},{latitude:59.91,longitude:10.71,recordingSegment:0},{latitude:59.93,longitude:10.73,recordingSegment:1},{latitude:59.94,longitude:10.74,recordingSegment:1}];const{svg}=createHarness({breadcrumbs});await flushAsync();const layer=svg.querySelector('#map-overlay-layer'),rendered=classes(layer);assert.equal(rendered.filter(value=>value==='map-overlay-live-breadcrumb').length,2);assert.equal(rendered.filter(value=>value==='map-overlay-live-segment-gap').length,1);assert.deepEqual(layer.children.filter(node=>node.getAttribute('class')==='map-overlay-live-breadcrumb').map(node=>node.getAttribute('data-recording-segment')),['0','1']);});

test('manual markers render with type labels and promoted state',async()=>{const markers=[{latitude:59.91,longitude:10.71,type:'trailhead',promotedWaypointID:''},{latitude:59.92,longitude:10.72,type:'cache',promotedWaypointID:'wp-7'}];const{svg}=createHarness({markers});await flushAsync();const layer=svg.querySelector('#map-overlay-layer'),rendered=classes(layer);assert.equal(rendered.filter(value=>value.startsWith('map-overlay-live-marker')).length,4);assert.ok(rendered.includes('map-overlay-live-marker map-overlay-live-marker-promoted'));assert.deepEqual(layer.children.filter(node=>node.getAttribute('class')==='map-overlay-live-marker-label').map(node=>node.textContent),['1: trailhead','2: cache']);});

test('follow GPS centers viewport only when explicitly enabled',async()=>{const result={kind:'waypoint',id:'wp-1',from:{latitude:59.9,longitude:10.7},target:{latitude:60,longitude:10.8}};const{context,document,centered}=createHarness({navigationResult:result});await flushAsync();await run(context,`navigateField(59.9,10.7)`);assert.equal(centered.length,0);const follow=document.querySelector('#map-overlay-follow');follow.checked=true;follow.dispatchEvent({type:'change'});assert.deepEqual(centered.at(-1),{x:1070,y:-5990});await run(context,`navigateField(59.9,10.7)`);assert.equal(centered.length,2);follow.checked=false;await run(context,`navigateField(59.9,10.7)`);assert.equal(centered.length,2);});

test('fit current session includes breadcrumbs and manual markers',async()=>{const breadcrumbs=[{latitude:59.9,longitude:10.7,recordingSegment:0},{latitude:59.91,longitude:10.71,recordingSegment:0}],markers=[{latitude:59.92,longitude:10.72,type:'note'}];const{context,fitted}=createHarness({breadcrumbs,markers});await flushAsync();assert.equal(run(context,'fitCurrentSession()'),true);assert.equal(fitted.length,1);assert.equal(JSON.stringify(fitted[0].map(point=>[point.x,point.y])),JSON.stringify([[1070,-5990],[1071,-5991],[1072,-5992]]));});

test('accepted breadcrumb and marker rerender hooks update live overlay',async()=>{const{context,svg,breadcrumbs,markers}=createHarness();await flushAsync();run(context,`addBreadcrumb({coords:{latitude:59.9,longitude:10.7},segment:0})`);run(context,`addBreadcrumb({coords:{latitude:59.91,longitude:10.71},segment:0})`);assert.equal(breadcrumbs.length,2);assert.equal(classes(svg.querySelector('#map-overlay-layer')).filter(value=>value==='map-overlay-live-breadcrumb').length,1);markers.push({latitude:59.92,longitude:10.72,type:'note',promotedWaypointID:''});run(context,`renderMarkers()`);assert.ok(classes(svg.querySelector('#map-overlay-layer')).includes('map-overlay-live-marker'));});

test('map selection clears stale path navigation result for another object',async()=>{const result={kind:'track',id:'path-1',from:{latitude:59.9,longitude:10.7},target:{latitude:59.91,longitude:10.71},cross_track_m:5,progress:{next_point:{latitude:59.92,longitude:10.72},remaining_m:500,forward_bearing_deg:90},guidance:{status:'on-route',arrival_radius_m:20}};const{context,document,svg}=createHarness({navigationResult:result});await flushAsync();await run(context,`navigateField(59.9,10.7)`);assert.ok(classes(svg.querySelector('#map-overlay-layer')).includes('map-overlay-nearest'));document.dispatchEvent({type:'caching-tools:map-select',detail:{kind:'waypoint',id:'wp-other'}});assert.equal(classes(svg.querySelector('#map-overlay-layer')).includes('map-overlay-nearest'),false);});

test('map-rendered event refreshes waypoint data and rerenders labels',async()=>{const waypoints=[{id:'wp-1',name:'One',point:{latitude:60,longitude:10.8}}];const{document,svg}=createHarness({waypoints});await flushAsync();assert.equal(classes(svg.querySelector('#map-overlay-layer')).filter(value=>value==='map-overlay-label').length,1);document.dispatchEvent({type:'caching-tools:map-rendered'});await flushAsync();assert.equal(classes(svg.querySelector('#map-overlay-layer')).filter(value=>value==='map-overlay-label').length,1);});
