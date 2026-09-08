document.addEventListener('caching-tools:map-select', (event) => {
  const {kind, name} = event.detail || {};
  const list = kind === 'waypoint' ? document.querySelector('#waypoint-list') : document.querySelector('#path-list');
  if (!list) return;

  for (const row of list.children) row.classList.remove('list-selected');
  const rows = [...list.children];
  const selected = rows.find((row) => {
    const text = row.querySelector('pre')?.textContent || row.textContent || '';
    return kind === 'waypoint' ? text.startsWith(name) : text.includes(`${kind}: ${name}`);
  });
  if (!selected) return;
  selected.classList.add('list-selected');
  selected.scrollIntoView({behavior: 'smooth', block: 'center'});
});

function loadFieldNoteArchive() {
  if (document.querySelector('script[data-field-note-archive]')) return;
  const archive = document.createElement('script');
  archive.src = '/field-note-archive.js';
  archive.dataset.fieldNoteArchive = 'true';
  document.body.append(archive);
}

function loadFieldNoteAttachments() {
  const existing = document.querySelector('script[data-field-note-attachments]');
  if (existing) {
    existing.addEventListener('load', loadFieldNoteArchive, {once:true});
    return;
  }
  const attachments = document.createElement('script');
  attachments.src = '/field-note-attachments.js';
  attachments.dataset.fieldNoteAttachments = 'true';
  attachments.addEventListener('load', loadFieldNoteArchive, {once:true});
  document.body.append(attachments);
}

function loadMapLogbook() {
  const existing = document.querySelector('script[data-map-logbook]');
  if (existing) {
    existing.addEventListener('load', loadFieldNoteAttachments, {once:true});
    return;
  }
  const mapLogbook = document.createElement('script');
  mapLogbook.src = '/map-logbook.js';
  mapLogbook.dataset.mapLogbook = 'true';
  mapLogbook.addEventListener('load', loadFieldNoteAttachments, {once:true});
  document.body.append(mapLogbook);
}

function loadFieldNoteDashboard() {
  const existing = document.querySelector('script[data-field-note-dashboard]');
  if (existing) {
    existing.addEventListener('load', loadMapLogbook, {once:true});
    return;
  }
  const dashboard = document.createElement('script');
  dashboard.src = '/field-notes-dashboard.js';
  dashboard.dataset.fieldNoteDashboard = 'true';
  dashboard.addEventListener('load', loadMapLogbook, {once:true});
  document.body.append(dashboard);
}

function loadFieldNotes() {
  const existing = document.querySelector('script[data-field-notes]');
  if (existing) {
    existing.addEventListener('load', loadFieldNoteDashboard, {once:true});
    return;
  }
  const notes = document.createElement('script');
  notes.src = '/field-notes.js';
  notes.dataset.fieldNotes = 'true';
  notes.addEventListener('load', loadFieldNoteDashboard, {once:true});
  document.body.append(notes);
}

function loadFieldSession() {
  const existing = document.querySelector('script[data-field-session]');
  if (existing) {
    existing.addEventListener('load', loadFieldNotes, {once:true});
    return;
  }
  const session = document.createElement('script');
  session.src = '/field-session.js';
  session.dataset.fieldSession = 'true';
  session.addEventListener('load', loadFieldNotes, {once:true});
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
  } else if (existingQuality.dataset.loaded === 'true') {
    loadFieldSession();
  } else {
    existingQuality.addEventListener('load', loadFieldSession, {once:true});
  }

  if (!document.querySelector('script[data-map-editor]')) {
    const script = document.createElement('script');
    script.src = '/map-editor.js';
    script.dataset.mapEditor = 'true';
    document.body.append(script);
  }
});
