const mapListLinkState = {
  waypoint: {selector:'#waypoint-list', endpoint:'/api/waypoints', generation:0},
  path: {selector:'#path-list', endpoint:'/api/paths', generation:0}
};

async function annotateMapList(kind) {
  const state = mapListLinkState[kind];
  if (!state) return;
  const list = document.querySelector(state.selector);
  if (!list) return;
  const generation = ++state.generation;
  const items = await requestJSON(state.endpoint);
  if (generation !== state.generation) return;
  const rows = [...list.children];
  if (rows.length !== items.length) return;
  items.forEach((item, index) => {
    const row = rows[index];
    row.dataset.mapId = item.id;
    row.dataset.mapKind = kind === 'waypoint' ? 'waypoint' : item.kind;
  });
}

function scheduleMapListAnnotation(kind) {
  Promise.resolve().then(() => annotateMapList(kind)).catch(() => {});
}

for (const kind of ['waypoint','path']) {
  const state = mapListLinkState[kind];
  const list = document.querySelector(state.selector);
  if (!list) continue;
  new MutationObserver(() => scheduleMapListAnnotation(kind)).observe(list, {childList:true});
  scheduleMapListAnnotation(kind);
}

async function selectLinkedMapRow(kind, id) {
  const listKind = kind === 'waypoint' ? 'waypoint' : 'path';
  const state = mapListLinkState[listKind];
  const list = state ? document.querySelector(state.selector) : null;
  if (!list || !id) return;
  for (const row of list.children) row.classList.remove('list-selected');
  let selected = [...list.children].find((row) => row.dataset.mapKind === kind && row.dataset.mapId === id);
  if (!selected) {
    try { await annotateMapList(listKind); } catch (_) { return; }
    selected = [...list.children].find((row) => row.dataset.mapKind === kind && row.dataset.mapId === id);
  }
  if (!selected) return;
  selected.classList.add('list-selected');
  selected.scrollIntoView({behavior: 'smooth', block: 'center'});
}

document.addEventListener('caching-tools:map-select', (event) => {
  const {kind, id} = event.detail || {};
  void selectLinkedMapRow(kind, id);
});

function runLoaderNext(next) {
  if (typeof next === 'function') queueMicrotask(next);
}

function ensureDynamicAsset({dataAttribute, src, readySelector, next}) {
  if (readySelector && document.querySelector(readySelector)) {
    runLoaderNext(next);
    return;
  }

  const selector = `script[${dataAttribute}]`;
  let script = document.querySelector(selector);
  if (script?.dataset.loaderState === 'failed') {
    script.remove();
    script = null;
  }
  if (script) {
    if (script.dataset.loaderState === 'loaded') {
      runLoaderNext(next);
      return;
    }
    const continueOnce = () => {
      script.dataset.loaderState = 'loaded';
      runLoaderNext(next);
    };
    script.addEventListener('load', continueOnce, {once:true});
    return;
  }

  script = document.createElement('script');
  script.src = src;
  script.setAttribute(dataAttribute, 'true');
  script.dataset.loaderState = 'loading';
  script.addEventListener('load', () => {
    script.dataset.loaderState = 'loaded';
    runLoaderNext(next);
  }, {once:true});
  script.addEventListener('error', () => {
    script.dataset.loaderState = 'failed';
  }, {once:true});
  document.body.append(script);
}

function loadFieldNoteIntegrity() {
  ensureDynamicAsset({
    dataAttribute:'data-field-note-integrity',
    src:'/field-note-integrity.js',
    readySelector:'#field-note-attachment-integrity'
  });
}

function loadFieldNoteArchive() {
  ensureDynamicAsset({
    dataAttribute:'data-field-note-archive',
    src:'/field-note-archive.js',
    readySelector:'#field-note-archive',
    next:loadFieldNoteIntegrity
  });
}

function loadFieldNoteAttachmentsAsset() {
  ensureDynamicAsset({
    dataAttribute:'data-field-note-attachments',
    src:'/field-note-attachments.js',
    readySelector:'#field-note-attachments',
    next:loadFieldNoteArchive
  });
}

function loadMapLogbook() {
  ensureDynamicAsset({
    dataAttribute:'data-map-logbook',
    src:'/map-logbook.js',
    readySelector:'#map-logbook',
    next:loadFieldNoteAttachmentsAsset
  });
}

function loadFieldNoteDashboard() {
  ensureDynamicAsset({
    dataAttribute:'data-field-note-dashboard',
    src:'/field-notes-dashboard.js',
    readySelector:'#field-note-dashboard',
    next:loadMapLogbook
  });
}

function loadFieldNotesAsset() {
  ensureDynamicAsset({
    dataAttribute:'data-field-notes',
    src:'/field-notes.js',
    readySelector:'#field-notes',
    next:loadFieldNoteDashboard
  });
}

function loadFieldSessionBundle() {
  ensureDynamicAsset({
    dataAttribute:'data-field-session-bundle',
    src:'/field-session-bundle.js',
    readySelector:'#field-session-bundle',
    next:loadFieldNotesAsset
  });
}

function loadFieldSession() {
  ensureDynamicAsset({
    dataAttribute:'data-field-session',
    src:'/field-session.js',
    readySelector:'#field-session-controls',
    next:loadFieldSessionBundle
  });
}

window.addEventListener('load', () => {
  ensureDynamicAsset({
    dataAttribute:'data-field-quality',
    src:'/field-quality.js',
    readySelector:'#field-quality-controls',
    next:loadFieldSession
  });
  ensureDynamicAsset({
    dataAttribute:'data-map-editor',
    src:'/map-editor.js',
    readySelector:'#map-path-editor'
  });
});
