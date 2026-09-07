const finalForm = document.querySelector('#final-coordinate-form');
const finalResult = document.querySelector('#final-coordinate-result');
const finalSave = document.querySelector('#final-coordinate-save');
let finalCoordinate = null;

function parseVariableAssignments(text) {
  const variables = {};
  for (const raw of text.split(/[\n,;]+/)) {
    const entry = raw.trim();
    if (!entry) continue;
    const match = entry.match(/^([A-Za-z])\s*=\s*([+-]?(?:\d+(?:\.\d*)?|\.\d+))$/);
    if (!match) throw new Error(`Invalid variable assignment: ${entry}`);
    variables[match[1].toUpperCase()] = Number(match[2]);
  }
  return variables;
}

function variableAssignmentsText(variables = {}) {
  return Object.keys(variables).sort().map(key => `${key}=${variables[key]}`).join(', ');
}

function renderFinalCoordinate(data) {
  return [
    `Expanded: ${data.latitude_expanded}, ${data.longitude_expanded}`,
    `DD   ${data.point.lat.dd}, ${data.point.lon.dd}`,
    `DMM  ${data.point.lat.dmm}, ${data.point.lon.dmm}`,
    `DMS  ${data.point.lat.dms}, ${data.point.lon.dms}`
  ].join('\n');
}

async function solveFinalCoordinateFromForm() {
  const form = new FormData(finalForm);
  const payload = {
    variables: parseVariableAssignments(String(form.get('variables') || '')),
    latitude_formula: String(form.get('latitude-formula') || ''),
    longitude_formula: String(form.get('longitude-formula') || '')
  };
  const data = await postJSON('/api/coordinates/final', payload);
  finalCoordinate = data;
  finalResult.textContent = renderFinalCoordinate(data);
  finalSave.hidden = false;
  return data;
}

function loadFinalWorkspace(workspace) {
  if (!workspace) return;
  finalForm.querySelector('[name="variables"]').value = variableAssignmentsText(workspace.variables || {});
  finalForm.querySelector('[name="latitude-formula"]').value = workspace.latitude_formula || '';
  finalForm.querySelector('[name="longitude-formula"]').value = workspace.longitude_formula || '';
  finalResult.textContent = `Loaded ${workspace.code ? workspace.code + ' — ' : ''}${workspace.title}.`;
  solveFinalCoordinateFromForm().catch(error => {
    finalCoordinate = null;
    finalSave.hidden = true;
    finalResult.textContent = `Loaded workspace, but formula is not yet solvable: ${error.message}`;
  });
  finalForm.scrollIntoView({behavior:'smooth', block:'start'});
}

finalForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  try {
    await solveFinalCoordinateFromForm();
  } catch (error) {
    finalCoordinate = null;
    finalSave.hidden = true;
    finalResult.textContent = `Error: ${error.message}`;
  }
});

for (const input of finalForm.querySelectorAll('input, textarea')) {
  input.addEventListener('input', async () => {
    try {
      await solveFinalCoordinateFromForm();
    } catch (_) {
      finalCoordinate = null;
      finalSave.hidden = true;
    }
  });
}

finalSave.addEventListener('click', async () => {
  if (!finalCoordinate) return;
  try {
    const workspace = window.getActiveMysteryWorkspace ? window.getActiveMysteryWorkspace() : null;
    const name = workspace ? `${workspace.code ? workspace.code + ' ' : ''}${workspace.title} final` : 'Final coordinate';
    const saved = await postJSON('/api/waypoints', {
      name,
      latitude: String(finalCoordinate.point.latitude),
      longitude: String(finalCoordinate.point.longitude),
      type: 'final',
      comment: `${workspace ? `Mystery workspace ${workspace.id}. ` : ''}Formula final: ${finalCoordinate.latitude_expanded}, ${finalCoordinate.longitude_expanded}`
    });
    if (window.linkFinalWaypointToActiveWorkspace) await window.linkFinalWaypointToActiveWorkspace(saved.id);
    finalResult.textContent = `${renderFinalCoordinate(finalCoordinate)}\n\nSaved as waypoint${workspace ? ' and linked to active workspace' : ''}.`;
    await loadWaypoints();
    if (window.refreshLocalMap) await window.refreshLocalMap();
  } catch (error) {
    finalResult.textContent = `${renderFinalCoordinate(finalCoordinate)}\n\nSave error: ${error.message}`;
  }
});

window.loadFinalWorkspace = loadFinalWorkspace;
