import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const qualitySource = fs.readFileSync(new URL('../../cmd/caching-tools/web/field-quality.js', import.meta.url), 'utf8');
const sessionSource = fs.readFileSync(new URL('../../cmd/caching-tools/web/field-session.js', import.meta.url), 'utf8');

class FakeElement {
  constructor(tagName = 'div', document = null) {
    this.tagName = tagName.toUpperCase();
    this.document = document;
    this.children = [];
    this.listeners = new Map();
    this.elements = {};
    this.value = '';
    this.checked = false;
    this.disabled = false;
    this.textContent = '';
    this.id = '';
    this.options = [];
    this.selectedOptions = [];
  }
  set innerHTML(value) {
    this._innerHTML = value;
    const ids = [...value.matchAll(/id="([^"]+)"/g)].map(match => match[1]);
    for (const id of ids) {
      const node = new FakeElement('input', this.document);
      node.id = id;
      this.document?.register(id, node);
    }
  }
  get innerHTML() { return this._innerHTML || ''; }
  querySelector(selector) {
    if (selector.startsWith('#')) return this.document?.querySelector(selector) || null;
    return null;
  }
  append(...nodes) {
    for (const node of nodes) {
      this.children.push(node);
      if (node.tagName === 'OPTION') this.options.push(node);
    }
  }
  replaceChildren(...nodes) {
    this.children = [];
    this.options = [];
    this.append(...nodes);
  }
  addEventListener(type, listener) {
    const entries = this.listeners.get(type) || [];
    entries.push(listener);
    this.listeners.set(type, entries);
  }
  dispatchEvent(event) {
    for (const listener of this.listeners.get(event.type) || []) listener.call(this, event);
  }
  insertAdjacentElement() {}
}

class FakeDocument {
  constructor() { this.nodes = new Map(); }
  register(id, node = new FakeElement('div', this)) {
    node.id = id;
    node.document = this;
    this.nodes.set(id, node);
    return node;
  }
  querySelector(selector) {
    if (selector.startsWith('#')) return this.nodes.get(selector.slice(1)) || null;
    if (selector === 'footer') return this.nodes.get('footer') || null;
    return null;
  }
  createElement(tagName) { return new FakeElement(tagName, this); }
}

function haversine(a, b) {
  const rad = value => value * Math.PI / 180;
  const dLat = rad(b.latitude - a.latitude);
  const dLon = rad(b.longitude - a.longitude);
  const lat1 = rad(a.latitude);
  const lat2 = rad(b.latitude);
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) ** 2;
  return 6371008.8 * 2 * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h));
}

function createHarness() {
  const document = new FakeDocument();
  const fieldSection = document.register('field-section');
  const clear = document.register('field-breadcrumb-clear', new FakeElement('button', document));
  fieldSection.querySelector = selector => {
    if (selector === 'h3:nth-of-type(2)') return null;
    if (selector === '#field-breadcrumb-clear') return clear;
    return document.querySelector(selector);
  };
  fieldSection.append = () => {};
  const fieldSessionStats = document.register('field-session-stats');
  fieldSessionStats.insertAdjacentElement = () => {};
  document.register('footer');

  const fieldForm = {elements:{
    latitude:{value:'59.9139'},
    longitude:{value:'10.7522'},
    'target-id':{selectedOptions:[{textContent:'Test target'}]}
  }};
  const liveBreadcrumbs = [];
  let rawBreadcrumbCalls = 0;
  let started = 0;
  let mapRefreshes = 0;
  let waypointLoads = 0;
  let targetLoads = 0;
  const posts = [];

  const context = vm.createContext({
    document,
    window:{refreshLocalMap:async () => { mapRefreshes += 1; }},
    fieldSection,
    fieldSessionStats,
    fieldForm,
    liveBreadcrumbs,
    fieldLiveStatus:{textContent:''},
    addBreadcrumb(position) {
      rawBreadcrumbCalls += 1;
      liveBreadcrumbs.push({
        latitude:position.coords.latitude,
        longitude:position.coords.longitude,
        time:position.time || new Date(position.timestamp || Date.now()).toISOString()
      });
      return true;
    },
    sessionStatistics() { return null; },
    breadcrumbDistanceMeters:haversine,
    breadcrumbGPX() { if (!liveBreadcrumbs.length) throw new Error('No breadcrumb points to export.'); return '<gpx/>'; },
    startLiveNavigation() { started += 1; },
    escapeXML(value) { return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;'); },
    async postJSON(url, payload) { posts.push({url, payload}); return {id:'wp-promoted'}; },
    async loadWaypoints() { waypointLoads += 1; },
    async loadFieldTargets() { targetLoads += 1; },
    Date,
    Math,
    Number,
    Promise,
    console
  });

  vm.runInContext(qualitySource, context, {filename:'field-quality.js'});
  document.querySelector('#field-min-distance').value = '3';
  document.querySelector('#field-max-accuracy').value = '50';
  document.querySelector('#field-export-simplify').checked = false;
  document.querySelector('#field-simplify-tolerance').value = '5';
  vm.runInContext(sessionSource, context, {filename:'field-session.js'});

  return {
    context, document, fieldForm, liveBreadcrumbs, posts,
    counts:() => ({rawBreadcrumbCalls, started, mapRefreshes, waypointLoads, targetLoads})
  };
}

function run(context, expression) { return vm.runInContext(expression, context); }

function position(latitude, longitude, accuracy = 5, time = '2026-09-09T00:00:00.000Z') {
  return {coords:{latitude, longitude, accuracy}, time};
}

test('breadcrumb quality filter rejects poor accuracy and too-short movement', () => {
  const {context, liveBreadcrumbs, counts} = createHarness();
  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9139, 10.7522, 100))})`), false);
  assert.equal(liveBreadcrumbs.length, 0);
  assert.equal(counts().rawBreadcrumbCalls, 0);

  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9139, 10.7522, 5))})`), true);
  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9139001, 10.7522001, 5))})`), false);
  assert.equal(liveBreadcrumbs.length, 1);
  assert.equal(counts().rawBreadcrumbCalls, 1);
  assert.match(run(context, `fieldQualityStatus.textContent`), /Accepted 1/);
  assert.match(run(context, `fieldQualityStatus.textContent`), /rejected accuracy 1/);
  assert.match(run(context, `fieldQualityStatus.textContent`), /rejected distance 1/);
});

test('pause blocks breadcrumb recording and resume starts a new segment', () => {
  const {context, liveBreadcrumbs} = createHarness();
  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9139, 10.7522, 5, '2026-09-09T00:00:00.000Z'))})`), true);
  run(context, 'pauseBreadcrumbRecording()');
  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9149, 10.7532, 5, '2026-09-09T00:01:00.000Z'))})`), false);
  assert.equal(liveBreadcrumbs.length, 1);

  run(context, 'resumeBreadcrumbRecording()');
  assert.equal(run(context, `addBreadcrumb(${JSON.stringify(position(59.9149, 10.7532, 5, '2026-09-09T00:02:00.000Z'))})`), true);
  assert.equal(liveBreadcrumbs.length, 2);
  assert.equal(liveBreadcrumbs[0].recordingSegment, 0);
  assert.equal(liveBreadcrumbs[1].recordingSegment, 1);
  assert.match(run(context, 'fieldRecordingStatus.textContent'), /segment 2/);
});

test('manual marker uses current field position, custom type and note', () => {
  const {context, fieldForm} = createHarness();
  fieldForm.elements.latitude.value = '58.1234567';
  fieldForm.elements.longitude.value = '8.7654321';
  run(context, `fieldMarkerType.value = 'custom'; fieldMarkerCustom.value = 'viewpoint'; fieldMarkerNote.value = 'Great view'; addManualMarker();`);
  assert.equal(run(context, 'liveMarkers.length'), 1);
  assert.equal(run(context, 'liveMarkers[0].type'), 'viewpoint');
  assert.equal(run(context, 'liveMarkers[0].note'), 'Great view');
  assert.equal(run(context, 'liveMarkers[0].latitude'), 58.1234567);
  assert.equal(run(context, 'liveMarkers[0].longitude'), 8.7654321);
  assert.match(run(context, 'fieldRecordingStatus.textContent'), /Added viewpoint marker/);
});

test('promoting a manual marker persists a waypoint and refreshes local targets/map', async () => {
  const {context, posts, counts} = createHarness();
  run(context, `fieldMarkerType.value = 'cache'; fieldMarkerNote.value = 'Found spot'; addManualMarker(); fieldMarkerWaypointName.value = 'Saved field point';`);
  await run(context, 'promoteManualMarker()');

  assert.equal(posts.length, 1);
  assert.equal(posts[0].url, '/api/waypoints');
  assert.equal(posts[0].payload.name, 'Saved field point');
  assert.equal(posts[0].payload.type, 'cache');
  assert.match(posts[0].payload.comment, /Found spot/);
  assert.match(posts[0].payload.comment, /Promoted from live field session marker/);
  assert.equal(run(context, 'liveMarkers[0].promotedWaypointID'), 'wp-promoted');
  assert.deepEqual(counts(), {rawBreadcrumbCalls:0, started:0, mapRefreshes:1, waypointLoads:1, targetLoads:1});
  assert.match(run(context, 'fieldRecordingStatus.textContent'), /wp-promoted/);
});

test('session statistics exclude distance and moving time across pause boundaries', () => {
  const {context, liveBreadcrumbs} = createHarness();
  liveBreadcrumbs.push(
    {latitude:59.0, longitude:10.0, time:'2026-09-09T00:00:00.000Z', recordingSegment:0},
    {latitude:59.001, longitude:10.0, time:'2026-09-09T00:01:00.000Z', recordingSegment:0},
    {latitude:60.0, longitude:11.0, time:'2026-09-09T00:10:00.000Z', recordingSegment:1},
    {latitude:60.001, longitude:11.0, time:'2026-09-09T00:11:00.000Z', recordingSegment:1}
  );
  const stats = run(context, 'sessionStatistics()');
  assert.ok(stats.distanceM > 150 && stats.distanceM < 300);
  assert.equal(stats.movingTimeS, 120);
  assert.equal(stats.durationS, 660);
});

test('starting a new live session resets markers, pause state, segment and quality counters', () => {
  const {context, counts} = createHarness();
  run(context, `qualityAccepted = 4; qualityRejectedAccuracy = 2; qualityRejectedDistance = 3; liveMarkers.push({type:'cache'}); breadcrumbRecordingPaused = true; recordingSegmentID = 5; startLiveNavigation();`);
  assert.equal(run(context, 'liveMarkers.length'), 0);
  assert.equal(run(context, 'breadcrumbRecordingPaused'), false);
  assert.equal(run(context, 'recordingSegmentID'), 0);
  assert.equal(run(context, 'qualityAccepted'), 0);
  assert.equal(run(context, 'qualityRejectedAccuracy'), 0);
  assert.equal(run(context, 'qualityRejectedDistance'), 0);
  assert.equal(counts().started, 1);
});
