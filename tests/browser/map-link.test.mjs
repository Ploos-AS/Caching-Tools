import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const source = fs.readFileSync(new URL('../../cmd/caching-tools/web/map-link.js', import.meta.url), 'utf8');

class FakeClassList {
  constructor() { this.values = new Set(); }
  add(value) { this.values.add(value); }
  remove(value) { this.values.delete(value); }
  contains(value) { return this.values.has(value); }
}

class FakeElement {
  constructor(tagName = 'div', ownerDocument = null) {
    this.tagName = tagName.toUpperCase();
    this.ownerDocument = ownerDocument;
    this.children = [];
    this.dataset = {};
    this.classList = new FakeClassList();
    this.listeners = new Map();
    this.attributes = new Map();
    this.parentNode = null;
    this.id = '';
    this.src = '';
    this.removed = false;
    this.scrolled = false;
  }
  setAttribute(name, value) {
    this.attributes.set(name, String(value));
    if (name === 'id') this.id = String(value);
    if (name.startsWith('data-')) {
      const key = name.slice(5).replace(/-([a-z])/g, (_, c) => c.toUpperCase());
      this.dataset[key] = String(value);
    }
  }
  addEventListener(type, listener, options = {}) {
    const entries = this.listeners.get(type) || [];
    entries.push({listener, once:Boolean(options?.once)});
    this.listeners.set(type, entries);
  }
  dispatchEvent(event) {
    event.target ??= this;
    const entries = [...(this.listeners.get(event.type) || [])];
    for (const entry of entries) {
      entry.listener.call(this, event);
      if (entry.once) {
        const current = this.listeners.get(event.type) || [];
        this.listeners.set(event.type, current.filter(candidate => candidate !== entry));
      }
    }
  }
  append(...nodes) {
    for (const node of nodes) {
      node.parentNode = this;
      this.children.push(node);
      this.ownerDocument?._notifyMutation(this);
    }
  }
  remove() {
    this.removed = true;
    if (!this.parentNode) return;
    const parent = this.parentNode;
    parent.children = parent.children.filter(child => child !== this);
    this.parentNode = null;
    this.ownerDocument?._notifyMutation(parent);
  }
  scrollIntoView() { this.scrolled = true; }
}

class FakeDocument {
  constructor() {
    this.listeners = new Map();
    this.body = new FakeElement('body', this);
    this.nodesById = new Map();
    this.observers = [];
  }
  register(id, node = new FakeElement('div', this)) {
    node.id = id;
    node.ownerDocument = this;
    this.nodesById.set(id, node);
    return node;
  }
  createElement(tagName) { return new FakeElement(tagName, this); }
  querySelector(selector) {
    if (selector.startsWith('#')) return this.nodesById.get(selector.slice(1)) || null;
    const match = selector.match(/^script\[([^\]]+)\]$/);
    if (match) return this.body.children.find(node => node.tagName === 'SCRIPT' && node.attributes.has(match[1])) || null;
    return null;
  }
  addEventListener(type, listener, options = {}) {
    const entries = this.listeners.get(type) || [];
    entries.push({listener, once:Boolean(options?.once)});
    this.listeners.set(type, entries);
  }
  dispatchEvent(event) {
    const entries = [...(this.listeners.get(event.type) || [])];
    for (const entry of entries) {
      entry.listener.call(this, event);
      if (entry.once) this.listeners.set(event.type, (this.listeners.get(event.type) || []).filter(candidate => candidate !== entry));
    }
  }
  _notifyMutation(target) {
    for (const observer of this.observers) if (observer.target === target) observer.callback([], observer.instance);
  }
}

function createHarness({waypoints = [], paths = []} = {}) {
  const document = new FakeDocument();
  const waypointList = document.register('waypoint-list');
  const pathList = document.register('path-list');

  const requestJSON = async endpoint => {
    if (endpoint === '/api/waypoints') return waypoints;
    if (endpoint === '/api/paths') return paths;
    throw new Error(`unexpected endpoint ${endpoint}`);
  };

  class MutationObserver {
    constructor(callback) { this.callback = callback; this.target = null; this.instance = this; }
    observe(target) { this.target = target; document.observers.push(this); }
  }

  const windowListeners = new Map();
  const window = {
    addEventListener(type, listener) {
      const entries = windowListeners.get(type) || [];
      entries.push(listener);
      windowListeners.set(type, entries);
    },
    dispatchEvent(event) { for (const listener of windowListeners.get(event.type) || []) listener(event); }
  };

  const context = vm.createContext({
    document,
    window,
    MutationObserver,
    requestJSON,
    queueMicrotask,
    Promise,
    console
  });
  vm.runInContext(source, context, {filename:'map-link.js'});
  return {context, document, window, waypointList, pathList};
}

async function flushAsync() {
  await Promise.resolve();
  await Promise.resolve();
  await new Promise(resolve => setImmediate(resolve));
}

function row(document) { return new FakeElement('div', document); }

function run(context, expression) { return vm.runInContext(expression, context); }

test('map/list selection uses stable IDs when display names are duplicated', async () => {
  const waypoints = [
    {id:'wp-1', name:'Duplicate'},
    {id:'wp-2', name:'Duplicate'}
  ];
  const {context, document, waypointList} = createHarness({waypoints});
  const first = row(document);
  const second = row(document);
  waypointList.append(first, second);
  await flushAsync();

  await run(context, `selectLinkedMapRow('waypoint', 'wp-2')`);

  assert.equal(first.dataset.mapId, 'wp-1');
  assert.equal(second.dataset.mapId, 'wp-2');
  assert.equal(first.classList.contains('list-selected'), false);
  assert.equal(second.classList.contains('list-selected'), true);
  assert.equal(second.scrolled, true);
});

test('path rows retain route/track kind and select by kind plus ID', async () => {
  const paths = [
    {id:'path-1', kind:'route', summary:{name:'Same'}},
    {id:'path-2', kind:'track', summary:{name:'Same'}}
  ];
  const {context, document, pathList} = createHarness({paths});
  const route = row(document);
  const track = row(document);
  pathList.append(route, track);
  await flushAsync();

  await run(context, `selectLinkedMapRow('track', 'path-2')`);

  assert.equal(route.dataset.mapKind, 'route');
  assert.equal(track.dataset.mapKind, 'track');
  assert.equal(route.classList.contains('list-selected'), false);
  assert.equal(track.classList.contains('list-selected'), true);
});

test('ready DOM sentinel advances loader without adding a script', async () => {
  const {context, document} = createHarness();
  document.register('ready-sentinel');
  context.nextCount = 0;

  run(context, `ensureDynamicAsset({dataAttribute:'data-test', src:'/test.js', readySelector:'#ready-sentinel', next:() => { nextCount += 1; }})`);
  await flushAsync();

  assert.equal(context.nextCount, 1);
  assert.equal(document.body.children.length, 0);
});

test('already loaded script advances loader immediately', async () => {
  const {context, document} = createHarness();
  const script = document.createElement('script');
  script.setAttribute('data-test', 'true');
  script.dataset.loaderState = 'loaded';
  document.body.append(script);
  context.nextCount = 0;

  run(context, `ensureDynamicAsset({dataAttribute:'data-test', src:'/test.js', next:() => { nextCount += 1; }})`);
  await flushAsync();

  assert.equal(context.nextCount, 1);
  assert.equal(document.body.children.length, 1);
});

test('loading script continues only after load event', async () => {
  const {context, document} = createHarness();
  const script = document.createElement('script');
  script.setAttribute('data-test', 'true');
  script.dataset.loaderState = 'loading';
  document.body.append(script);
  context.nextCount = 0;

  run(context, `ensureDynamicAsset({dataAttribute:'data-test', src:'/test.js', next:() => { nextCount += 1; }})`);
  await flushAsync();
  assert.equal(context.nextCount, 0);

  script.dispatchEvent({type:'load'});
  await flushAsync();
  assert.equal(script.dataset.loaderState, 'loaded');
  assert.equal(context.nextCount, 1);
});

test('failed script is replaced and a successful retry advances', async () => {
  const {context, document} = createHarness();
  const failed = document.createElement('script');
  failed.setAttribute('data-test', 'true');
  failed.dataset.loaderState = 'failed';
  document.body.append(failed);
  context.nextCount = 0;

  run(context, `ensureDynamicAsset({dataAttribute:'data-test', src:'/retry.js', next:() => { nextCount += 1; }})`);
  const replacement = document.querySelector('script[data-test]');

  assert.notEqual(replacement, failed);
  assert.equal(failed.removed, true);
  assert.equal(replacement.src, '/retry.js');
  assert.equal(replacement.dataset.loaderState, 'loading');

  replacement.dispatchEvent({type:'load'});
  await flushAsync();
  assert.equal(replacement.dataset.loaderState, 'loaded');
  assert.equal(context.nextCount, 1);
});

test('window load creates quality and editor scripts exactly once per loader state', async () => {
  const {document, window} = createHarness();
  window.dispatchEvent({type:'load'});
  await flushAsync();

  const quality = document.querySelector('script[data-field-quality]');
  const editor = document.querySelector('script[data-map-editor]');
  assert.ok(quality);
  assert.ok(editor);
  assert.equal(quality.dataset.loaderState, 'loading');
  assert.equal(editor.dataset.loaderState, 'loading');

  window.dispatchEvent({type:'load'});
  await flushAsync();
  assert.equal(document.body.children.filter(node => node.tagName === 'SCRIPT').length, 2);
});
