const fieldNotesSection = document.createElement('section');
fieldNotesSection.className = 'tool';
fieldNotesSection.id = 'field-notes';
fieldNotesSection.innerHTML = `
  <h2>Field notes / logbook</h2>
  <p>Keep persistent local field observations under <code>/data/field-notes.json</code>. Notes can optionally reference a saved waypoint or mystery workspace.</p>
  <form id="field-note-form" class="form-grid nav-grid">
    <input name="note-id" type="hidden">
    <label>Title<input name="title" value="Field note" required></label>
    <label>Status<input name="status" placeholder="info, found, done"></label>
    <label>Type<input name="type" placeholder="cache, trail, observation"></label>
    <label>Occurred at<input name="occurred-at" type="datetime-local"></label>
    <label>Waypoint<select name="waypoint-id"><option value="">None</option></select></label>
    <label>Mystery workspace<select name="workspace-id"><option value="">None</option></select></label>
    <label>Note<textarea name="body" rows="4" placeholder="Field observation"></textarea></label>
    <button type="submit" id="field-note-save">Save field note</button>
    <button type="button" id="field-note-cancel" hidden>Cancel edit</button>
  </form>
  <h3>Search / filter</h3>
  <div id="field-note-filters" class="form-grid nav-grid">
    <label>Search<input id="field-note-search" placeholder="title, note, status, type"></label>
    <label>Status<select id="field-note-filter-status"><option value="">All statuses</option></select></label>
    <label>Type<select id="field-note-filter-type"><option value="">All types</option></select></label>
    <label>Waypoint<select id="field-note-filter-waypoint"><option value="">All waypoints</option></select></label>
    <label>Workspace<select id="field-note-filter-workspace"><option value="">All workspaces</option></select></label>
    <button type="button" id="field-note-filter-clear">Clear filters</button>
    <button type="button" id="field-note-export-json">Export filtered JSON</button>
    <button type="button" id="field-note-export-csv">Export filtered CSV</button>
  </div>
  <p id="field-note-filter-status-text" aria-live="polite"></p>
  <p id="field-note-status" aria-live="polite"></p>
  <div id="field-note-list" class="result">Loading field notes...</div>`;

const fieldNavigationSection = document.querySelector('#field-navigation');
if (fieldNavigationSection) fieldNavigationSection.insertAdjacentElement('afterend', fieldNotesSection);
else document.querySelector('main')?.append(fieldNotesSection);

const fieldNoteForm = fieldNotesSection.querySelector('#field-note-form');
const fieldNoteList = fieldNotesSection.querySelector('#field-note-list');
const fieldNoteStatus = fieldNotesSection.querySelector('#field-note-status');
const fieldNoteCancel = fieldNotesSection.querySelector('#field-note-cancel');
const fieldNoteSearch = fieldNotesSection.querySelector('#field-note-search');
const fieldNoteFilterStatus = fieldNotesSection.querySelector('#field-note-filter-status');
const fieldNoteFilterType = fieldNotesSection.querySelector('#field-note-filter-type');
const fieldNoteFilterWaypoint = fieldNotesSection.querySelector('#field-note-filter-waypoint');
const fieldNoteFilterWorkspace = fieldNotesSection.querySelector('#field-note-filter-workspace');
const fieldNoteFilterStatusText = fieldNotesSection.querySelector('#field-note-filter-status-text');
let fieldNoteCache = [];

function localDateTimeToISO(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error('Occurred at is invalid.');
  return date.toISOString();
}

function isoToLocalDateTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0,16);
}

function fieldNotePayload() {
  const form = new FormData(fieldNoteForm);
  return {
    title: String(form.get('title') || ''),
    body: String(form.get('body') || ''),
    status: String(form.get('status') || ''),
    type: String(form.get('type') || ''),
    waypoint_id: String(form.get('waypoint-id') || ''),
    workspace_id: String(form.get('workspace-id') || ''),
    occurred_at: localDateTimeToISO(String(form.get('occurred-at') || ''))
  };
}

async function loadFieldNoteReferences() {
  const [waypoints, workspaces] = await Promise.all([
    requestJSON('/api/waypoints'),
    requestJSON('/api/mystery-workspaces')
  ]);
  const waypointSelect = fieldNoteForm.elements['waypoint-id'];
  const workspaceSelect = fieldNoteForm.elements['workspace-id'];
  const selectedWaypoint = waypointSelect.value;
  const selectedWorkspace = workspaceSelect.value;
  waypointSelect.replaceChildren(new Option('None',''));
  workspaceSelect.replaceChildren(new Option('None',''));
  fieldNoteFilterWaypoint.replaceChildren(new Option('All waypoints',''));
  fieldNoteFilterWorkspace.replaceChildren(new Option('All workspaces',''));
  for (const item of waypoints) {
    waypointSelect.append(new Option(item.name, item.id));
    fieldNoteFilterWaypoint.append(new Option(item.name, item.id));
  }
  for (const item of workspaces) {
    const label = `${item.code ? item.code + ' · ' : ''}${item.title}`;
    workspaceSelect.append(new Option(label, item.id));
    fieldNoteFilterWorkspace.append(new Option(label, item.id));
  }
  waypointSelect.value = selectedWaypoint;
  workspaceSelect.value = selectedWorkspace;
}

function resetFieldNoteForm() {
  fieldNoteForm.reset();
  fieldNoteForm.elements['note-id'].value = '';
  fieldNoteForm.elements.title.value = 'Field note';
  fieldNoteCancel.hidden = true;
  fieldNotesSection.querySelector('#field-note-save').textContent = 'Save field note';
}

function noteFilterValues(items, key) {
  return [...new Set(items.map(item => String(item[key] || '').trim()).filter(Boolean))].sort((a,b) => a.localeCompare(b));
}

function refreshFieldNoteFacetOptions() {
  const selectedStatus = fieldNoteFilterStatus.value;
  const selectedType = fieldNoteFilterType.value;
  fieldNoteFilterStatus.replaceChildren(new Option('All statuses',''));
  fieldNoteFilterType.replaceChildren(new Option('All types',''));
  for (const value of noteFilterValues(fieldNoteCache, 'status')) fieldNoteFilterStatus.append(new Option(value, value));
  for (const value of noteFilterValues(fieldNoteCache, 'type')) fieldNoteFilterType.append(new Option(value, value));
  fieldNoteFilterStatus.value = selectedStatus;
  fieldNoteFilterType.value = selectedType;
}

function filteredFieldNotes() {
  const query = fieldNoteSearch.value.trim().toLowerCase();
  const status = fieldNoteFilterStatus.value;
  const type = fieldNoteFilterType.value;
  const waypointID = fieldNoteFilterWaypoint.value;
  const workspaceID = fieldNoteFilterWorkspace.value;
  return fieldNoteCache.filter(item => {
    if (status && item.status !== status) return false;
    if (type && item.type !== type) return false;
    if (waypointID && item.waypoint_id !== waypointID) return false;
    if (workspaceID && item.workspace_id !== workspaceID) return false;
    if (!query) return true;
    const haystack = [item.title, item.body, item.status, item.type, item.waypoint_id, item.workspace_id, item.occurred_at].join('\n').toLowerCase();
    return haystack.includes(query);
  });
}

function renderFieldNotes(items) {
  fieldNoteList.replaceChildren();
  fieldNoteFilterStatusText.textContent = `Showing ${items.length} of ${fieldNoteCache.length} field note${fieldNoteCache.length === 1 ? '' : 's'}.`;
  if (!items.length) {
    fieldNoteList.textContent = 'No field notes match the current filters.';
    return;
  }
  for (const item of items) {
    const row = document.createElement('div');
    const text = document.createElement('pre');
    const links = [];
    if (item.waypoint_id) links.push(`waypoint ${item.waypoint_id}`);
    if (item.workspace_id) links.push(`workspace ${item.workspace_id}`);
    text.textContent = [
      `${item.occurred_at} · ${item.title}`,
      [item.status, item.type].filter(Boolean).join(' · '),
      item.body || '',
      links.length ? `Linked: ${links.join(', ')}` : ''
    ].filter(Boolean).join('\n');
    const edit = document.createElement('button');
    edit.type = 'button'; edit.textContent = 'Edit'; edit.addEventListener('click', () => editFieldNote(item.id));
    const del = document.createElement('button');
    del.type = 'button'; del.textContent = 'Delete'; del.addEventListener('click', () => deleteFieldNote(item.id));
    row.append(text, edit, del);
    fieldNoteList.append(row);
  }
}

function applyFieldNoteFilters() {
  renderFieldNotes(filteredFieldNotes());
}

async function loadFieldNotes() {
  try {
    fieldNoteCache = await requestJSON('/api/field-notes');
    refreshFieldNoteFacetOptions();
    applyFieldNoteFilters();
  } catch (error) { fieldNoteList.textContent = `Error: ${error.message}`; }
}

function editFieldNote(id) {
  const item = fieldNoteCache.find(note => note.id === id);
  if (!item) return;
  fieldNoteForm.elements['note-id'].value = item.id;
  fieldNoteForm.elements.title.value = item.title;
  fieldNoteForm.elements.body.value = item.body || '';
  fieldNoteForm.elements.status.value = item.status || '';
  fieldNoteForm.elements.type.value = item.type || '';
  fieldNoteForm.elements['occurred-at'].value = isoToLocalDateTime(item.occurred_at);
  fieldNoteForm.elements['waypoint-id'].value = item.waypoint_id || '';
  fieldNoteForm.elements['workspace-id'].value = item.workspace_id || '';
  fieldNoteCancel.hidden = false;
  fieldNotesSection.querySelector('#field-note-save').textContent = 'Update field note';
}

async function deleteFieldNote(id) {
  try {
    await requestJSON(`/api/field-notes/${encodeURIComponent(id)}`, {method:'DELETE'});
    fieldNoteStatus.textContent = 'Field note deleted.';
    await loadFieldNotes();
  } catch (error) { fieldNoteStatus.textContent = `Error: ${error.message}`; }
}

function csvCell(value) {
  const text = String(value ?? '');
  return `"${text.replace(/"/g, '""')}"`;
}

function fieldNotesCSV(items) {
  const columns = ['id','occurred_at','title','status','type','body','waypoint_id','workspace_id','created_at','updated_at'];
  const rows = [columns.join(',')];
  for (const item of items) rows.push(columns.map(column => csvCell(item[column] || '')).join(','));
  return rows.join('\r\n') + '\r\n';
}

function downloadFieldNoteExport(data, type, suffix) {
  const blob = new Blob([data], {type});
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  const stamp = new Date().toISOString().replace(/[-:]/g,'').replace(/\..*$/,'').replace('T','-');
  link.href = url;
  link.download = `caching-tools-field-notes-${stamp}.${suffix}`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function exportFilteredFieldNotesJSON() {
  const items = filteredFieldNotes();
  downloadFieldNoteExport(JSON.stringify({format:'caching-tools-field-notes', version:1, exported_at:new Date().toISOString(), notes:items}, null, 2) + '\n', 'application/json;charset=utf-8', 'json');
  fieldNoteStatus.textContent = `Exported ${items.length} filtered field note${items.length === 1 ? '' : 's'} as JSON.`;
}

function exportFilteredFieldNotesCSV() {
  const items = filteredFieldNotes();
  downloadFieldNoteExport(fieldNotesCSV(items), 'text/csv;charset=utf-8', 'csv');
  fieldNoteStatus.textContent = `Exported ${items.length} filtered field note${items.length === 1 ? '' : 's'} as CSV.`;
}

fieldNoteForm.addEventListener('submit', async event => {
  event.preventDefault();
  const id = fieldNoteForm.elements['note-id'].value;
  try {
    const payload = fieldNotePayload();
    if (id) {
      await requestJSON(`/api/field-notes/${encodeURIComponent(id)}`, {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
      fieldNoteStatus.textContent = 'Field note updated.';
    } else {
      await postJSON('/api/field-notes', payload);
      fieldNoteStatus.textContent = 'Field note saved.';
    }
    resetFieldNoteForm();
    await loadFieldNotes();
  } catch (error) { fieldNoteStatus.textContent = `Error: ${error.message}`; }
});

for (const control of [fieldNoteSearch, fieldNoteFilterStatus, fieldNoteFilterType, fieldNoteFilterWaypoint, fieldNoteFilterWorkspace]) {
  control.addEventListener(control === fieldNoteSearch ? 'input' : 'change', applyFieldNoteFilters);
}
fieldNotesSection.querySelector('#field-note-filter-clear').addEventListener('click', () => {
  fieldNoteSearch.value = '';
  fieldNoteFilterStatus.value = '';
  fieldNoteFilterType.value = '';
  fieldNoteFilterWaypoint.value = '';
  fieldNoteFilterWorkspace.value = '';
  applyFieldNoteFilters();
});
fieldNotesSection.querySelector('#field-note-export-json').addEventListener('click', exportFilteredFieldNotesJSON);
fieldNotesSection.querySelector('#field-note-export-csv').addEventListener('click', exportFilteredFieldNotesCSV);
fieldNoteCancel.addEventListener('click', resetFieldNoteForm);
Promise.all([loadFieldNoteReferences(), loadFieldNotes()]).catch(error => { fieldNoteStatus.textContent = `Error: ${error.message}`; });
window.refreshFieldNotes = loadFieldNotes;
window.refreshFieldNoteReferences = loadFieldNoteReferences;
const fieldNoteFooter = document.querySelector('footer'); if (fieldNoteFooter) fieldNoteFooter.textContent = 'Caching Tools M1.32';
