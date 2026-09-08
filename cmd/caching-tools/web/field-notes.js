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
  <p id="field-note-status" aria-live="polite"></p>
  <div id="field-note-list" class="result">Loading field notes...</div>`;

const fieldNavigationSection = document.querySelector('#field-navigation');
if (fieldNavigationSection) fieldNavigationSection.insertAdjacentElement('afterend', fieldNotesSection);
else document.querySelector('main')?.append(fieldNotesSection);

const fieldNoteForm = fieldNotesSection.querySelector('#field-note-form');
const fieldNoteList = fieldNotesSection.querySelector('#field-note-list');
const fieldNoteStatus = fieldNotesSection.querySelector('#field-note-status');
const fieldNoteCancel = fieldNotesSection.querySelector('#field-note-cancel');
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
  for (const item of waypoints) waypointSelect.append(new Option(item.name, item.id));
  for (const item of workspaces) workspaceSelect.append(new Option(`${item.code ? item.code + ' · ' : ''}${item.title}`, item.id));
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

function renderFieldNotes(items) {
  fieldNoteCache = items;
  fieldNoteList.replaceChildren();
  if (!items.length) {
    fieldNoteList.textContent = 'No field notes.';
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

async function loadFieldNotes() {
  try { renderFieldNotes(await requestJSON('/api/field-notes')); }
  catch (error) { fieldNoteList.textContent = `Error: ${error.message}`; }
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

fieldNoteCancel.addEventListener('click', resetFieldNoteForm);
Promise.all([loadFieldNoteReferences(), loadFieldNotes()]).catch(error => { fieldNoteStatus.textContent = `Error: ${error.message}`; });
window.refreshFieldNotes = loadFieldNotes;
window.refreshFieldNoteReferences = loadFieldNoteReferences;
const fieldNoteFooter = document.querySelector('footer'); if (fieldNoteFooter) fieldNoteFooter.textContent = 'Caching Tools M1.31';
