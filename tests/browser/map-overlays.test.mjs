import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source = fs.readFileSync(new URL('../../cmd/caching-tools/web/map-overlays.js', import.meta.url), 'utf8');

class FakeElement {
  constructor(tag = 'div') {
    this.tagName = tag.toUpperCase();
    this.children = [];
    this.attributes = new Map();
    this.listeners = new Map();
    this.dataset = {};
    this.id = '';
    this.checked = false;
    this.value = '';
    this.textContent = '';
    this.elements = {};
    this.parentNode = null;
    this.className = '';
    this.innerHTML = '';
  }
  setAttribute(name, value) {
    this.attributes.set(name, String(value));
    if (name === 'id') this.id = String(value);
  }
  getAttribute(name) { return this.attributes.get(name) ?? null; }
  addEventListener(type, fn) {
    const list = this.listeners.get(type) || [];
    list.push(fn);
    this.listeners.set(type, list);
  }
  dispatchEvent(event) {
    for (const fn of this.listeners.get(event.type) || []) fn.call(this, event);
  }
  append(...nodes) {
    for (const node of nodes) { node.parentNode = this; this.children.push(node); }
  }
  replaceChildren(...nodes) {
    this.children = [];
    this.append(...nodes);
  }
  querySelector(selector) {
    if (selector === '#map-overlay-layer') return this.children.find(node => node.id === 'map-overlay-layer') || null;
    return null;
  }
  closest(selector) { return selector === '.map-frame' ? this._closestMapFrame || null : null; }
  insertAdjacentElement(_where, node) { this._inserted = node; }
}

class FakeDocument {
  constructor() {
    this.byId = new Map();
    this.listeners = new Map();
  }
  register(id, node = new FakeElement()) { node.id = id; this.byId.set(id, node); return node; }
  querySelector(selector) { return selector.startsWith('#') ? this.byId.get(selector.slice(1)) || null : null; }
  createElement(tag) { return new FakeElement(tag); }
  createElementNS(_ns, tag) { return new FakeElement(tag); }
  addEventListener(type, fn) {
    const list = this.listeners.get(type) || [];
    list.push(fn);
    this.listeners.set(type, list);
  }
  dispatchEvent(event) { for (const fn of this.listeners.get(event.type) || []) fn.call(this, event); }
}

function createHarness({waypoints = [], navigationResult = null} = {}) {
  const document = new FakeDocument();
  const svg = new FakeElement('svg');
  const localMap = document.register('local-map', svg);
  localMap._closestMapFrame = new FakeElement('div');

  const labels = document.register('map-overlay-labels'); labels.checked = true;
  const rings = document.register('map-overlay-rings'); rings.checked = false;
  const radius = document.register('map-overlay-radius'); radius.value = '100';
  const guidance = document.register('map-overlay-guidance'); guidance.checked = true;
  const pathGuidance = document.register('map-overlay-path-guidance'); pathGuidance.checked = true;
  document.register('map-overlay-refresh');
  document.register('map-status');

  const fieldForm = document.register('field-navigation-form');
  fieldForm.elements = {
    latitude: {value:'59.9000'},
    longitude: {value:'10.7000'},
    'arrival-radius': {value:'20'}
  };

  const project = (latitude, longitude) => ({x: longitude * 100, y: latitude * -100});
  const window = {cachingToolsMap:{svg, project}};
  const requestJSON = async endpoint => {
    if (endpoint === '/api/waypoints') return waypoints;
    throw new Error(`unexpected endpoint ${endpoint}`);
  };
  const navigateCalls = [];
  const navigateField = async (latitude, longitude) => {
    navigateCalls.push([latitude, longitude]);
    if (navigationResult instanceof Error) throw navigationResult;
    return navigationResult;
  };

  const context = vm.createContext({document, window, requestJSON, navigateField, console, Promise, Math, Number, String});
  vm.runInContext(source, context, {filename:'map-overlays.js'});
  return {context, document, window, svg, navigateCalls};
}

async function flushAsync() {
  await Promise.resolve();
  await Promise.resolve();
  await new Promise(resolve => setImmediate(resolve));
}

function classes(layer) {
  return layer.children.map(node => node.getAttribute('class') || '');
}

function run(context, expression) { return vm.runInContext(expression, context); }

test('waypoint selection renders current-position guidance and arrival ring', async () => {
  const waypoints = [{id:'wp-1', name:'Target', point:{latitude:60.0, longitude:10.8}}];
  const {context, document, svg} = createHarness({waypoints});
  await flushAsync();

  document.dispatchEvent({type:'caching-tools:map-select', detail:{kind:'waypoint', id:'wp-1', name:'Target'}});
  const layer = svg.querySelector('#map-overlay-layer');
  assert.ok(layer);
  const rendered = classes(layer);
  assert.ok(rendered.includes('map-overlay-position'));
  assert.ok(rendered.includes('map-overlay-guidance'));
  assert.ok(rendered.includes('map-overlay-arrival'));
  assert.ok(rendered.includes('map-overlay-label'));
});

test('wrapped navigation renders route cross-track, next-point and status overlays', async () => {
  const result = {
    kind:'route', id:'path-1', name:'Route',
    from:{latitude:59.9, longitude:10.7},
    target:{latitude:59.91, longitude:10.71},
    cross_track_m:42,
    progress:{
      next_point:{latitude:59.92, longitude:10.72},
      remaining_m:1250,
      forward_bearing_deg:87
    },
    guidance:{status:'off-route', arrival_radius_m:20}
  };
  const {context, svg, navigateCalls} = createHarness({navigationResult:result});
  await flushAsync();

  const returned = await run(context, `navigateField(59.9, 10.7)`);
  assert.equal(returned.id, 'path-1');
  assert.deepEqual(navigateCalls, [[59.9, 10.7]]);

  const layer = svg.querySelector('#map-overlay-layer');
  const rendered = classes(layer);
  assert.ok(rendered.some(value => value.includes('map-overlay-cross-track') && value.includes('map-overlay-status-off-route')));
  assert.ok(rendered.includes('map-overlay-nearest'));
  assert.ok(rendered.includes('map-overlay-forward'));
  assert.ok(rendered.includes('map-overlay-next'));
  assert.ok(rendered.includes('map-overlay-next-arrival'));
  assert.ok(rendered.some(value => value.includes('map-overlay-status-label') && value.includes('map-overlay-status-off-route')));
  const label = layer.children.find(node => (node.getAttribute('class') || '').includes('map-overlay-status-label'));
  assert.match(label.textContent, /off-route · 42 m off · 1\.25 km left · 87°/);
});

test('map selection clears stale path navigation result for another object', async () => {
  const result = {
    kind:'track', id:'path-1', from:{latitude:59.9, longitude:10.7}, target:{latitude:59.91, longitude:10.71},
    cross_track_m:5, progress:{next_point:{latitude:59.92, longitude:10.72}, remaining_m:500, forward_bearing_deg:90},
    guidance:{status:'on-route', arrival_radius_m:20}
  };
  const {context, document, svg} = createHarness({navigationResult:result});
  await flushAsync();
  await run(context, `navigateField(59.9, 10.7)`);
  assert.ok(classes(svg.querySelector('#map-overlay-layer')).includes('map-overlay-nearest'));

  document.dispatchEvent({type:'caching-tools:map-select', detail:{kind:'waypoint', id:'wp-other'}});
  assert.equal(classes(svg.querySelector('#map-overlay-layer')).includes('map-overlay-nearest'), false);
});

test('map-rendered event refreshes waypoint data and rerenders labels', async () => {
  const waypoints = [{id:'wp-1', name:'One', point:{latitude:60.0, longitude:10.8}}];
  const {document, svg} = createHarness({waypoints});
  await flushAsync();
  const before = classes(svg.querySelector('#map-overlay-layer')).filter(value => value === 'map-overlay-label').length;
  assert.equal(before, 1);

  document.dispatchEvent({type:'caching-tools:map-rendered'});
  await flushAsync();
  const after = classes(svg.querySelector('#map-overlay-layer')).filter(value => value === 'map-overlay-label').length;
  assert.equal(after, 1);
});
