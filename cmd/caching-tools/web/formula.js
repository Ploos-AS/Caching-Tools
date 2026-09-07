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
    await postJSON('/api/waypoints', {
      name: 'Final coordinate',
      latitude: String(finalCoordinate.point.latitude),
      longitude: String(finalCoordinate.point.longitude),
      type: 'final',
      comment: `Formula final: ${finalCoordinate.latitude_expanded}, ${finalCoordinate.longitude_expanded}`
    });
    finalResult.textContent = `${renderFinalCoordinate(finalCoordinate)}\n\nSaved as waypoint.`;
    await loadWaypoints();
    if (window.refreshLocalMap) await window.refreshLocalMap();
  } catch (error) {
    finalResult.textContent = `${renderFinalCoordinate(finalCoordinate)}\n\nSave error: ${error.message}`;
  }
});
