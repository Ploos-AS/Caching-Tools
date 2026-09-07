const originalRenderPaths = window.renderPaths;

async function applyPathEdit(id, payload) {
  const result = await requestJSON(`/api/paths/${encodeURIComponent(id)}/edit`, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(payload)
  });
  document.querySelector('#path-status').textContent = `Geometry updated: ${result.summary.points} points${result.summary.segments ? `, ${result.summary.segments} segments` : ''}.`;
  await loadPaths();
  if (window.refreshLocalMap) await window.refreshLocalMap();
}

function integerPrompt(label, defaultValue = '1') {
  const raw = window.prompt(label, defaultValue);
  if (raw == null) return null;
  const value = Number(raw);
  if (!Number.isInteger(value) || value < 1) throw new Error(`${label} must be a positive integer`);
  return value - 1;
}

async function editRouteGeometry(item, full) {
  const operation = window.prompt('Route edit: delete-point or move-point', 'delete-point');
  if (!operation) return;
  if (operation === 'delete-point') {
    const point = integerPrompt(`Point to delete (1-${full.route.length})`);
    if (point == null) return;
    await applyPathEdit(item.id, {operation, point});
    return;
  }
  if (operation === 'move-point') {
    const point = integerPrompt(`Point to move (1-${full.route.length})`);
    if (point == null) return;
    const target = integerPrompt(`New position (1-${full.route.length})`);
    if (target == null) return;
    await applyPathEdit(item.id, {operation, point, target});
    return;
  }
  throw new Error('Unknown route operation');
}

async function editTrackGeometry(item, full) {
  const operation = window.prompt('Track edit: delete-point, move-point, split-segment or merge-segments', 'delete-point');
  if (!operation) return;
  const segment = integerPrompt(`Segment (1-${full.segments.length})`);
  if (segment == null) return;
  if (segment < 0 || segment >= full.segments.length) throw new Error('Segment out of range');
  const points = full.segments[segment].Points || full.segments[segment].points || [];
  if (operation === 'delete-point' || operation === 'split-segment') {
    const point = integerPrompt(`${operation === 'delete-point' ? 'Point to delete' : 'Split before point'} (1-${points.length})`);
    if (point == null) return;
    await applyPathEdit(item.id, {operation, segment, point});
    return;
  }
  if (operation === 'move-point') {
    const point = integerPrompt(`Point to move (1-${points.length})`);
    if (point == null) return;
    const target = integerPrompt(`New position (1-${points.length})`);
    if (target == null) return;
    await applyPathEdit(item.id, {operation, segment, point, target});
    return;
  }
  if (operation === 'merge-segments') {
    await applyPathEdit(item.id, {operation, segment});
    return;
  }
  throw new Error('Unknown track operation');
}

async function editPathGeometry(item) {
  try {
    const full = await requestJSON(`/api/paths/${encodeURIComponent(item.id)}`);
    if (item.kind === 'route') await editRouteGeometry(item, full);
    else await editTrackGeometry(item, full);
  } catch (error) {
    document.querySelector('#path-status').textContent = `Error: ${error.message}`;
  }
}

window.renderPaths = function renderPathsWithEditor(items) {
  originalRenderPaths(items);
  const rows = [...document.querySelector('#path-list').children];
  items.forEach((item, index) => {
    const row = rows[index];
    if (!row) return;
    const edit = document.createElement('button');
    edit.type = 'button';
    edit.textContent = 'Edit geometry';
    edit.addEventListener('click', () => editPathGeometry(item));
    row.insertBefore(edit, row.children[1] || null);
  });
};

loadPaths();
