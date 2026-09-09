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

function loadFieldNoteIntegrity() {
  if (document.querySelector('script[data-field-note-integrity]')) return;
  const integrity = document.createElement('script');
  integrity.src = '/field-note-integrity.js';
  integrity.dataset.fieldNoteIntegrity = 'true';
  document.body.append(integrity);
}

function loadFieldNoteArchive() {
  const existing = document.querySelector('script[data-field-note-archive]');
  if (existing) { existing.addEventListener('load', loadFieldNoteIntegrity, {once:true}); return; }
  const archive = document.createElement('script');
  archive.src = '/field-note-archive.js';
  archive.dataset.fieldNoteArchive = 'true';
  archive.addEventListener('load', loadFieldNoteIntegrity, {once:true});
  document.body.append(archive);
}

function loadFieldNoteAttachmentsAsset() {
  const existing = document.querySelector('script[data-field-note-attachments]');
  if (existing) { existing.addEventListener('load', loadFieldNoteArchive, {once:true}); return; }
  const attachments = document.createElement('script');
  attachments.src = '/field-note-attachments.js';
  attachments.dataset.fieldNoteAttachments = 'true';
  attachments.addEventListener('load', loadFieldNoteArchive, {once:true});
  document.body.append(attachments);
}

function loadMapLogbook() {
  const existing = document.querySelector('script[data-map-logbook]');
  if (existing) { existing.addEventListener('load', loadFieldNoteAttachmentsAsset, {once:true}); return; }
  const mapLogbook = document.createElement('script');
  mapLogbook.src = '/map-logbook.js';
  mapLogbook.dataset.mapLogbook = 'true';
  mapLogbook.addEventListener('load', loadFieldNoteAttachmentsAsset, {once:true});
  document.body.append(mapLogbook);
}

function loadFieldNoteDashboard() {
  const existing = document.querySelector('script[data-field-note-dashboard]');
  if (existing) { existing.addEventListener('load', loadMapLogbook, {once:true}); return; }
  const dashboard = document.createElement('script');
  dashboard.src = '/field-notes-dashboard.js';
  dashboard.dataset.fieldNoteDashboard = 'true';
  dashboard.addEventListener('load', loadMapLogbook, {once:true});
  document.body.append(dashboard);
}

function loadFieldNotesAsset() {
  if (document.querySelector('#field-notes')) { loadFieldNoteDashboard(); return; }
  const existing = document.querySelector('script[data-field-notes]');
  if (existing) { existing.addEventListener('load', loadFieldNoteDashboard, {once:true}); return; }
  const notes = document.createElement('script');
  notes.src = '/field-notes.js';
  notes.dataset.fieldNotes = 'true';
  notes.addEventListener('load', loadFieldNoteDashboard, {once:true});
  document.body.append(notes);
}

function loadFieldSession() {
  const existing = document.querySelector('script[data-field-session]');
  if (existing) { existing.addEventListener('load', loadFieldNotesAsset, {once:true}); return; }
  const session = document.createElement('script');
  session.src = '/field-session.js';
  session.dataset.fieldSession = 'true';
  session.addEventListener('load', loadFieldNotesAsset, {once:true});
  document.body.append(session);
}

window.addEventListener('load', () => {
  const existingQuality = document.querySelector('script[data-field-quality]');
  if (!existingQuality) {
    const quality = document.createElement('script');
    quality.src = '/field-quality.js';
    quality.dataset.fieldQuality = 'true';
    quality.addEventListener('load', loadFieldSession, {once:true});
    document.body.append(quality);
  } else if (existingQuality.dataset.loaded === 'true') loadFieldSession();
  else existingQuality.addEventListener('load', loadFieldSession, {once:true});
  if (!document.querySelector('script[data-map-editor]')) {
    const script = document.createElement('script');
    script.src = '/map-editor.js';
    script.dataset.mapEditor = 'true';
    document.body.append(script);
  }
});
