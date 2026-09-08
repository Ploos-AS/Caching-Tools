const mapLogbookSection = document.createElement('section');
mapLogbookSection.className = 'tool';
mapLogbookSection.id = 'map-logbook';
mapLogbookSection.innerHTML = `
  <h2>Map waypoint logbook</h2>
  <p>Select a waypoint on the local map to see linked field notes and start a new note with that waypoint preselected.</p>
  <p id="map-logbook-selection">No waypoint selected.</p>
  <button type="button" id="map-logbook-new-note" disabled>New field note for selected waypoint</button>
  <pre id="map-logbook-notes" class="result">Select a waypoint on the map.</pre>`;

const dashboardTool = document.querySelector('#field-note-dashboard');
if (dashboardTool) dashboardTool.insertAdjacentElement('afterend', mapLogbookSection);
else document.querySelector('main')?.append(mapLogbookSection);

const mapLogbookSelection = mapLogbookSection.querySelector('#map-logbook-selection');
const mapLogbookNotes = mapLogbookSection.querySelector('#map-logbook-notes');
const mapLogbookNewNote = mapLogbookSection.querySelector('#map-logbook-new-note');
let mapLogbookWaypoint = null;
let mapLogbookGeneration = 0;

function formatMapLogbookNotes(items) {
  if (!items.length) return 'No field notes linked to this waypoint.';
  return items.map(item => [
    `${item.occurred_at} · ${item.title}`,
    [item.status, item.type].filter(Boolean).join(' · '),
    item.body || ''
  ].filter(Boolean).join('\n')).join('\n\n---\n\n');
}

async function showWaypointLogbook(id, name) {
  const generation = ++mapLogbookGeneration;
  mapLogbookWaypoint = {id, name};
  mapLogbookSelection.textContent = `Selected waypoint: ${name} (${id})`;
  mapLogbookNewNote.disabled = false;
  mapLogbookNotes.textContent = 'Loading linked field notes...';
  try {
    const notes = await requestJSON('/api/field-notes');
    if (generation !== mapLogbookGeneration) return;
    const linked = notes.filter(note => note.waypoint_id === id);
    mapLogbookNotes.textContent = `${linked.length} linked field note${linked.length === 1 ? '' : 's'}.\n\n${formatMapLogbookNotes(linked)}`;
  } catch (error) {
    if (generation !== mapLogbookGeneration) return;
    mapLogbookNotes.textContent = `Logbook error: ${error.message}`;
  }
}

function startWaypointFieldNote() {
  if (!mapLogbookWaypoint) return;
  resetFieldNoteForm();
  fieldNoteForm.elements['waypoint-id'].value = mapLogbookWaypoint.id;
  fieldNoteForm.elements.title.value = `Field note · ${mapLogbookWaypoint.name}`;
  fieldNoteStatus.textContent = `New field note linked to ${mapLogbookWaypoint.name}.`;
  fieldNotesSection.scrollIntoView({behavior:'smooth', block:'start'});
  fieldNoteForm.elements.title.focus();
}

mapLogbookNewNote.addEventListener('click', startWaypointFieldNote);

document.addEventListener('caching-tools:map-select', event => {
  const {kind, id, name} = event.detail || {};
  if (kind !== 'waypoint' || !id) return;
  void showWaypointLogbook(id, name || id);
});

const observedMapFieldNoteList = document.querySelector('#field-note-list');
if (observedMapFieldNoteList) {
  new MutationObserver(() => {
    if (mapLogbookWaypoint) void showWaypointLogbook(mapLogbookWaypoint.id, mapLogbookWaypoint.name);
  }).observe(observedMapFieldNoteList, {childList:true, subtree:true, characterData:true});
}

window.refreshMapWaypointLogbook = () => mapLogbookWaypoint ? showWaypointLogbook(mapLogbookWaypoint.id, mapLogbookWaypoint.name) : Promise.resolve();
const mapLogbookFooter = document.querySelector('footer'); if (mapLogbookFooter) mapLogbookFooter.textContent = 'Caching Tools M1.35';
