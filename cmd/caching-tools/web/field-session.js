const sessionControls = document.createElement('div');
sessionControls.id = 'field-session-controls';
sessionControls.className = 'form-grid nav-grid';
sessionControls.innerHTML = `
  <button type="button" id="field-recording-pause">Pause breadcrumb recording</button>
  <button type="button" id="field-recording-resume" disabled>Resume breadcrumb recording</button>
  <label>Marker type<select id="field-marker-type"><option value="cache">Cache</option><option value="trailhead">Trailhead</option><option value="note">Note</option><option value="custom">Custom</option></select></label>
  <label>Custom marker type<input id="field-marker-custom" placeholder="viewpoint"></label>
  <label>Marker note<input id="field-marker-note" placeholder="Optional field note"></label>
  <button type="button" id="field-marker-add">Add marker at current position</button>
  <button type="button" id="field-marker-clear">Clear markers</button>
  <p id="field-recording-status" aria-live="polite">Breadcrumb recording active.</p>
  <pre id="field-session-markers" class="result">No manual markers.</pre>`;

const qualityControlsNode = fieldSection.querySelector('#field-quality-controls');
if (qualityControlsNode) qualityControlsNode.insertAdjacentElement('afterend', sessionControls);
else fieldSessionStats.insertAdjacentElement('afterend', sessionControls);

const fieldRecordingPause = sessionControls.querySelector('#field-recording-pause');
const fieldRecordingResume = sessionControls.querySelector('#field-recording-resume');
const fieldRecordingStatus = sessionControls.querySelector('#field-recording-status');
const fieldMarkerType = sessionControls.querySelector('#field-marker-type');
const fieldMarkerCustom = sessionControls.querySelector('#field-marker-custom');
const fieldMarkerNote = sessionControls.querySelector('#field-marker-note');
const fieldSessionMarkers = sessionControls.querySelector('#field-session-markers');
let breadcrumbRecordingPaused = false;
let liveMarkers = [];

function renderRecordingStatus() {
  fieldRecordingPause.disabled = breadcrumbRecordingPaused;
  fieldRecordingResume.disabled = !breadcrumbRecordingPaused;
  fieldRecordingStatus.textContent = breadcrumbRecordingPaused
    ? 'Breadcrumb recording paused. Live navigation continues.'
    : 'Breadcrumb recording active.';
}

function setBreadcrumbRecordingPaused(paused) {
  breadcrumbRecordingPaused = paused;
  renderRecordingStatus();
}

function markerTypeValue() {
  const type = fieldMarkerType.value;
  if (type !== 'custom') return type;
  const custom = fieldMarkerCustom.value.trim();
  return custom || 'custom';
}

function renderMarkers() {
  if (!liveMarkers.length) {
    fieldSessionMarkers.textContent = 'No manual markers.';
    return;
  }
  const recent = liveMarkers.slice(-12);
  const lines = [`${liveMarkers.length} manual marker${liveMarkers.length === 1 ? '' : 's'} in this in-memory session.`];
  if (liveMarkers.length > recent.length) lines.push(`Showing the latest ${recent.length}:`);
  for (const marker of recent) {
    const note = marker.note ? ` · ${marker.note}` : '';
    lines.push(`${marker.time}  ${marker.type}  ${marker.latitude.toFixed(7)}, ${marker.longitude.toFixed(7)}${note}`);
  }
  fieldSessionMarkers.textContent = lines.join('\n');
}

function addManualMarker() {
  const latitude = Number(fieldForm.elements.latitude.value);
  const longitude = Number(fieldForm.elements.longitude.value);
  if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90 || !Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
    fieldRecordingStatus.textContent = 'Cannot add marker: current position is invalid.';
    return;
  }
  liveMarkers.push({
    latitude,
    longitude,
    type: markerTypeValue(),
    note: fieldMarkerNote.value.trim(),
    time: new Date().toISOString()
  });
  if (liveMarkers.length > 500) liveMarkers.shift();
  fieldMarkerNote.value = '';
  renderMarkers();
  fieldRecordingStatus.textContent = `Added ${liveMarkers[liveMarkers.length - 1].type} marker.`;
}

const addBreadcrumbQualityFiltered = addBreadcrumb;
addBreadcrumb = function addBreadcrumbWithPause(position) {
  if (breadcrumbRecordingPaused) return false;
  return addBreadcrumbQualityFiltered(position);
};

function markerGPX() {
  return liveMarkers.map((marker, index) => {
    const type = escapeXML(marker.type);
    const note = marker.note ? `<desc>${escapeXML(marker.note)}</desc>` : '';
    return `  <wpt lat="${marker.latitude.toFixed(7)}" lon="${marker.longitude.toFixed(7)}"><time>${escapeXML(marker.time)}</time><name>${escapeXML(`${marker.type} ${index + 1}`)}</name><type>${type}</type>${note}</wpt>`;
  }).join('\n');
}

const breadcrumbGPXWithQuality = breadcrumbGPX;
breadcrumbGPX = function breadcrumbGPXWithMarkers() {
  const gpx = breadcrumbGPXWithQuality();
  if (!liveMarkers.length) return gpx;
  const markerXML = markerGPX();
  return gpx.replace(/(\n  <trk>)/, `\n${markerXML}$1`);
};

const startLiveNavigationWithQuality = startLiveNavigation;
startLiveNavigation = function startLiveNavigationWithSessionReset() {
  liveMarkers = [];
  setBreadcrumbRecordingPaused(false);
  renderMarkers();
  startLiveNavigationWithQuality();
};

fieldRecordingPause.addEventListener('click', () => setBreadcrumbRecordingPaused(true));
fieldRecordingResume.addEventListener('click', () => setBreadcrumbRecordingPaused(false));
sessionControls.querySelector('#field-marker-add').addEventListener('click', addManualMarker);
sessionControls.querySelector('#field-marker-clear').addEventListener('click', () => {
  liveMarkers = [];
  renderMarkers();
  fieldRecordingStatus.textContent = 'Manual markers cleared.';
});

renderRecordingStatus();
renderMarkers();
const sessionFooter = document.querySelector('footer');
if (sessionFooter) sessionFooter.textContent = 'Caching Tools M1.29';
