const visualMap = window.cachingToolsMap;
const visualMapSvg = visualMap.svg;
let visualPathId = '';
let visualPathKind = '';
let visualPathName = '';
let visualSelectedPoint = null;
let visualDrag = null;
let visualRenderedPaths = visualMap.paths || [];

function ensureVisualEditorControls() {
  let box = document.querySelector('#map-path-editor');
  if (box) return box;
  box = document.createElement('div');
  box.id = 'map-path-editor';
  box.className = 'result map-path-editor';
  box.innerHTML = '<strong>Visual route/track editor</strong><p id="map-path-editor-status">Select a route or track on the map.</p><div class="map-controls"><button type="button" id="map-path-delete-point" disabled>Delete selected point</button><button type="button" id="map-path-split" disabled>Split track here</button><button type="button" id="map-path-merge" disabled>Merge with next segment</button></div>';
  document.querySelector('#map-selection').insertAdjacentElement('afterend', box);
  box.querySelector('#map-path-delete-point').addEventListener('click', () => deleteVisualPoint());
  box.querySelector('#map-path-split').addEventListener('click', () => splitVisualSegment());
  box.querySelector('#map-path-merge').addEventListener('click', () => mergeVisualSegment());
  return box;
}

function visualStatus(text) {
  ensureVisualEditorControls().querySelector('#map-path-editor-status').textContent = text;
}

function clearVisualMarkers() {
  for (const node of visualMapSvg.querySelectorAll('.map-edit-point, .map-edit-label')) node.remove();
}

function pointCollections(item) {
  if (item.kind === 'route') return [{segment: 0, points: item.route || []}];
  return (item.segments || []).map((segment, index) => ({segment:index, points:segment.points || segment.Points || []}));
}

function currentVisualPath() {
  return visualRenderedPaths.find((item) => item.id === visualPathId) || null;
}

function updateVisualButtons() {
  const box = ensureVisualEditorControls();
  const selected = visualSelectedPoint;
  const item = currentVisualPath();
  box.querySelector('#map-path-delete-point').disabled = !selected;
  box.querySelector('#map-path-split').disabled = !selected || item?.kind !== 'track' || selected.point <= 0;
  box.querySelector('#map-path-merge').disabled = !selected || item?.kind !== 'track' || selected.segment + 1 >= (item?.segments || []).length;
}

function selectVisualPoint(marker, detail) {
  visualSelectedPoint = detail;
  for (const node of visualMapSvg.querySelectorAll('.map-edit-point-selected')) node.classList.remove('map-edit-point-selected');
  marker.classList.add('map-edit-point-selected');
  visualStatus(`${visualPathKind} ${visualPathName}: segment ${detail.segment + 1}, point ${detail.point + 1}. Drag to move.`);
  updateVisualButtons();
}

function renderVisualMarkers() {
  clearVisualMarkers();
  const item = currentVisualPath();
  if (!item || !visualMap.project) {
    visualSelectedPoint = null;
    updateVisualButtons();
    return;
  }

  for (const group of pointCollections(item)) {
    group.points.forEach((point, pointIndex) => {
      const latitude = visualMap.pointLat(point);
      const longitude = visualMap.pointLon(point);
      if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return;
      const xy = visualMap.project(latitude, longitude);
      const marker = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      marker.setAttribute('cx', xy.x);
      marker.setAttribute('cy', xy.y);
      marker.setAttribute('r', '7');
      marker.setAttribute('class', 'map-edit-point');
      marker.setAttribute('tabindex', '0');
      marker.dataset.segment = String(group.segment);
      marker.dataset.point = String(pointIndex);
      const detail = {segment:group.segment, point:pointIndex};
      marker.addEventListener('click', (event) => { event.stopPropagation(); selectVisualPoint(marker, detail); });
      marker.addEventListener('keydown', (event) => {
        if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); selectVisualPoint(marker, detail); }
      });
      marker.addEventListener('pointerdown', (event) => {
        event.preventDefault();
        event.stopPropagation();
        marker.setPointerCapture(event.pointerId);
        selectVisualPoint(marker, detail);
        visualDrag = {marker, pointerId:event.pointerId, detail};
      });
      marker.addEventListener('pointermove', (event) => {
        if (!visualDrag || visualDrag.pointerId !== event.pointerId) return;
        event.preventDefault(); event.stopPropagation();
        const xyNow = visualMap.clientToMap(event.clientX, event.clientY);
        marker.setAttribute('cx', xyNow.x); marker.setAttribute('cy', xyNow.y);
      });
      marker.addEventListener('pointerup', async (event) => {
        if (!visualDrag || visualDrag.pointerId !== event.pointerId) return;
        event.preventDefault(); event.stopPropagation();
        const xyNow = visualMap.clientToMap(event.clientX, event.clientY);
        const coordinate = visualMap.inverse(xyNow.x, xyNow.y);
        visualDrag = null;
        await setVisualPoint(detail, coordinate);
      });
      marker.addEventListener('pointercancel', () => { visualDrag = null; renderVisualMarkers(); });
      visualMapSvg.append(marker);
    });
  }
  updateVisualButtons();
}

async function visualEdit(payload) {
  if (!visualPathId) throw new Error('Select a route or track first');
  const updated = await requestJSON(`/api/paths/${encodeURIComponent(visualPathId)}/edit`, {
    method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)
  });
  visualStatus(`Saved ${updated.summary.name}.`);
  await loadPaths();
  await window.refreshLocalMap();
}

async function setVisualPoint(detail, coordinate) {
  try {
    await visualEdit({operation:'set-point', segment:detail.segment, point:detail.point, latitude:coordinate.latitude, longitude:coordinate.longitude});
  } catch (error) {
    visualStatus(`Error: ${error.message}`);
    renderVisualMarkers();
  }
}

async function deleteVisualPoint() {
  if (!visualSelectedPoint) return;
  try {
    await visualEdit({operation:'delete-point', segment:visualSelectedPoint.segment, point:visualSelectedPoint.point});
    visualSelectedPoint = null;
  } catch (error) { visualStatus(`Error: ${error.message}`); }
}

async function splitVisualSegment() {
  if (!visualSelectedPoint || visualPathKind !== 'track') return;
  try {
    await visualEdit({operation:'split-segment', segment:visualSelectedPoint.segment, point:visualSelectedPoint.point});
    visualSelectedPoint = null;
  } catch (error) { visualStatus(`Error: ${error.message}`); }
}

async function mergeVisualSegment() {
  if (!visualSelectedPoint || visualPathKind !== 'track') return;
  try {
    await visualEdit({operation:'merge-segments', segment:visualSelectedPoint.segment});
    visualSelectedPoint = null;
  } catch (error) { visualStatus(`Error: ${error.message}`); }
}

document.addEventListener('caching-tools:map-select', event => {
  const detail = event.detail || {};
  if (detail.kind !== 'route' && detail.kind !== 'track') {
    visualPathId = ''; visualPathKind = ''; visualPathName = ''; visualSelectedPoint = null;
    clearVisualMarkers(); visualStatus('Select a route or track on the map.'); updateVisualButtons(); return;
  }
  visualPathId = detail.id; visualPathKind = detail.kind; visualPathName = detail.name || detail.id; visualSelectedPoint = null;
  visualStatus(`${detail.kind}: ${visualPathName}. Select a point marker to edit.`);
  renderVisualMarkers();
});

document.addEventListener('caching-tools:map-rendered', event => {
  visualRenderedPaths = event.detail?.paths || [];
  if (visualPathId && !visualRenderedPaths.some(item => item.id === visualPathId)) {
    visualPathId = ''; visualPathKind = ''; visualPathName = ''; visualSelectedPoint = null;
  }
  renderVisualMarkers();
});

ensureVisualEditorControls();
