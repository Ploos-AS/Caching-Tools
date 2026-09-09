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
  <label>Promote marker<select id="field-marker-promote-select"><option value="">No manual markers</option></select></label>
  <label>Waypoint name<input id="field-marker-waypoint-name" placeholder="Field marker"></label>
  <button type="button" id="field-marker-promote">Promote marker to waypoint</button>
  <button type="button" id="field-session-restore" disabled>Restore recovered session</button>
  <button type="button" id="field-session-discard" disabled>Discard recovered session</button>
  <p id="field-session-recovery-status" aria-live="polite">No recovered session.</p>
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
const fieldMarkerPromoteSelect = sessionControls.querySelector('#field-marker-promote-select');
const fieldMarkerWaypointName = sessionControls.querySelector('#field-marker-waypoint-name');
const fieldMarkerPromote = sessionControls.querySelector('#field-marker-promote');
const fieldSessionMarkers = sessionControls.querySelector('#field-session-markers');
const fieldSessionRestore = sessionControls.querySelector('#field-session-restore');
const fieldSessionDiscard = sessionControls.querySelector('#field-session-discard');
const fieldSessionRecoveryStatus = sessionControls.querySelector('#field-session-recovery-status');
const sessionRecoveryKey = 'caching-tools.field-session.v1';
let breadcrumbRecordingPaused = false;
let recordingSegmentID = 0;
let startNewRecordingSegment = false;
let liveMarkers = [];
let recoveredSession = null;

function sessionStorage() {
  try { return window.localStorage || localStorage; } catch { return null; }
}

function validRecoveredPoint(item) {
  return item && Number.isFinite(Number(item.latitude)) && Number.isFinite(Number(item.longitude)) && typeof item.time === 'string';
}

function normalizeRecoveredSession(value) {
  if (!value || value.version !== 1 || !Array.isArray(value.breadcrumbs) || !Array.isArray(value.markers)) return null;
  if (!value.breadcrumbs.every(validRecoveredPoint) || !value.markers.every(validRecoveredPoint)) return null;
  return {
    version: 1,
    savedAt: typeof value.savedAt === 'string' ? value.savedAt : '',
    breadcrumbs: value.breadcrumbs.slice(0, 10000),
    markers: value.markers.slice(0, 500),
    paused: Boolean(value.paused),
    segmentID: Math.max(0, Number.isInteger(value.segmentID) ? value.segmentID : 0)
  };
}

function renderRecoveryStatus() {
  const available = Boolean(recoveredSession);
  fieldSessionRestore.disabled = !available;
  fieldSessionDiscard.disabled = !available;
  if (!available) {
    fieldSessionRecoveryStatus.textContent = 'No recovered session.';
    return;
  }
  const when = recoveredSession.savedAt ? ` saved ${recoveredSession.savedAt}` : '';
  fieldSessionRecoveryStatus.textContent = `Recovered session available${when}: ${recoveredSession.breadcrumbs.length} breadcrumb${recoveredSession.breadcrumbs.length === 1 ? '' : 's'}, ${recoveredSession.markers.length} marker${recoveredSession.markers.length === 1 ? '' : 's'}.`;
}

function loadRecoveredSession() {
  const storage = sessionStorage();
  if (!storage) return;
  try {
    const raw = storage.getItem(sessionRecoveryKey);
    recoveredSession = raw ? normalizeRecoveredSession(JSON.parse(raw)) : null;
    if (raw && !recoveredSession) storage.removeItem(sessionRecoveryKey);
  } catch {
    recoveredSession = null;
  }
  renderRecoveryStatus();
}

function persistFieldSession() {
  const storage = sessionStorage();
  if (!storage) return false;
  if (!liveBreadcrumbs.length && !liveMarkers.length) {
    try { storage.removeItem(sessionRecoveryKey); } catch {}
    return true;
  }
  const snapshot = {
    version: 1,
    savedAt: new Date().toISOString(),
    breadcrumbs: liveBreadcrumbs,
    markers: liveMarkers,
    paused: breadcrumbRecordingPaused,
    segmentID: recordingSegmentID
  };
  try {
    storage.setItem(sessionRecoveryKey, JSON.stringify(snapshot));
    return true;
  } catch {
    return false;
  }
}

function discardRecoveredSession() {
  const storage = sessionStorage();
  try { storage?.removeItem(sessionRecoveryKey); } catch {}
  recoveredSession = null;
  renderRecoveryStatus();
}

function restoreRecoveredSession() {
  if (!recoveredSession) return false;
  liveBreadcrumbs.splice(0, liveBreadcrumbs.length, ...recoveredSession.breadcrumbs.map(item => ({...item})));
  liveMarkers = recoveredSession.markers.map(item => ({...item}));
  breadcrumbRecordingPaused = recoveredSession.paused;
  recordingSegmentID = recoveredSession.segmentID;
  startNewRecordingSegment = false;
  renderRecordingStatus();
  renderMarkers();
  fieldRecordingStatus.textContent = `Recovered ${liveBreadcrumbs.length} breadcrumbs and ${liveMarkers.length} markers. Live GPS remains stopped until explicitly started.`;
  recoveredSession = null;
  renderRecoveryStatus();
  persistFieldSession();
  if (typeof window.renderMapOverlays === 'function') window.renderMapOverlays();
  return true;
}

function renderRecordingStatus() {
  fieldRecordingPause.disabled = breadcrumbRecordingPaused;
  fieldRecordingResume.disabled = !breadcrumbRecordingPaused;
  fieldRecordingStatus.textContent = breadcrumbRecordingPaused
    ? 'Breadcrumb recording paused. Live navigation continues.'
    : `Breadcrumb recording active · segment ${recordingSegmentID + 1}.`;
}

function pauseBreadcrumbRecording() {
  if (breadcrumbRecordingPaused) return;
  breadcrumbRecordingPaused = true;
  renderRecordingStatus();
  persistFieldSession();
}

function resumeBreadcrumbRecording() {
  if (!breadcrumbRecordingPaused) return;
  breadcrumbRecordingPaused = false;
  recordingSegmentID += 1;
  startNewRecordingSegment = true;
  renderRecordingStatus();
  persistFieldSession();
}

function markerTypeValue() {
  const type = fieldMarkerType.value;
  if (type !== 'custom') return type;
  const custom = fieldMarkerCustom.value.trim();
  return custom || 'custom';
}

function suggestedWaypointName(marker, index) {
  const label = marker.type.charAt(0).toUpperCase() + marker.type.slice(1);
  return `${label} field marker ${index + 1}`;
}

function renderMarkerPromotionOptions() {
  const previous = fieldMarkerPromoteSelect.value;
  fieldMarkerPromoteSelect.replaceChildren();
  if (!liveMarkers.length) {
    const option = document.createElement('option');
    option.value = '';
    option.textContent = 'No manual markers';
    fieldMarkerPromoteSelect.append(option);
    fieldMarkerWaypointName.value = '';
    fieldMarkerPromote.disabled = true;
    return;
  }
  liveMarkers.forEach((marker, index) => {
    const option = document.createElement('option');
    option.value = String(index);
    option.textContent = `${index + 1}: ${marker.type}${marker.promotedWaypointID ? ' · saved' : ''}`;
    fieldMarkerPromoteSelect.append(option);
  });
  fieldMarkerPromoteSelect.value = [...fieldMarkerPromoteSelect.options].some(option => option.value === previous) ? previous : '0';
  const selectedIndex = Number(fieldMarkerPromoteSelect.value);
  const selected = liveMarkers[selectedIndex];
  fieldMarkerWaypointName.value = selected ? suggestedWaypointName(selected, selectedIndex) : '';
  fieldMarkerPromote.disabled = !selected || Boolean(selected.promotedWaypointID);
}

function renderMarkers() {
  renderMarkerPromotionOptions();
  if (!liveMarkers.length) {
    fieldSessionMarkers.textContent = 'No manual markers.';
    return;
  }
  const recent = liveMarkers.slice(-12);
  const offset = liveMarkers.length - recent.length;
  const lines = [`${liveMarkers.length} manual marker${liveMarkers.length === 1 ? '' : 's'} in this in-memory session.`];
  if (liveMarkers.length > recent.length) lines.push(`Showing the latest ${recent.length}:`);
  recent.forEach((marker, recentIndex) => {
    const note = marker.note ? ` · ${marker.note}` : '';
    const promoted = marker.promotedWaypointID ? ` · waypoint ${marker.promotedWaypointID}` : '';
    lines.push(`${offset + recentIndex + 1}. ${marker.time}  ${marker.type}  ${marker.latitude.toFixed(7)}, ${marker.longitude.toFixed(7)}${note}${promoted}`);
  });
  fieldSessionMarkers.textContent = lines.join('\n');
}

function addManualMarker() {
  const latitude = Number(fieldForm.elements.latitude.value);
  const longitude = Number(fieldForm.elements.longitude.value);
  if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90 || !Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
    fieldRecordingStatus.textContent = 'Cannot add marker: current position is invalid.';
    return;
  }
  liveMarkers.push({latitude, longitude, type: markerTypeValue(), note: fieldMarkerNote.value.trim(), time: new Date().toISOString(), promotedWaypointID: ''});
  if (liveMarkers.length > 500) liveMarkers.shift();
  fieldMarkerNote.value = '';
  renderMarkers();
  fieldMarkerPromoteSelect.value = String(liveMarkers.length - 1);
  fieldMarkerWaypointName.value = suggestedWaypointName(liveMarkers[liveMarkers.length - 1], liveMarkers.length - 1);
  fieldMarkerPromote.disabled = false;
  fieldRecordingStatus.textContent = `Added ${liveMarkers[liveMarkers.length - 1].type} marker.`;
  persistFieldSession();
}

async function promoteManualMarker() {
  const index = Number(fieldMarkerPromoteSelect.value);
  const marker = liveMarkers[index];
  if (!marker) { fieldRecordingStatus.textContent = 'Choose a manual marker to promote.'; return; }
  if (marker.promotedWaypointID) { fieldRecordingStatus.textContent = `Marker is already saved as waypoint ${marker.promotedWaypointID}.`; return; }
  const name = fieldMarkerWaypointName.value.trim() || suggestedWaypointName(marker, index);
  const provenance = `Promoted from live field session marker recorded ${marker.time}.`;
  const comment = marker.note ? `${marker.note}\n${provenance}` : provenance;
  fieldMarkerPromote.disabled = true;
  try {
    const saved = await postJSON('/api/waypoints', {name, latitude: String(marker.latitude), longitude: String(marker.longitude), type: marker.type, comment});
    marker.promotedWaypointID = saved.id;
    renderMarkers();
    persistFieldSession();
    await Promise.all([loadWaypoints(), loadFieldTargets()]);
    if (window.refreshLocalMap) await window.refreshLocalMap();
    fieldRecordingStatus.textContent = `Promoted marker ${index + 1} to waypoint ${saved.id}.`;
  } catch (error) {
    fieldMarkerPromote.disabled = false;
    fieldRecordingStatus.textContent = `Waypoint promotion error: ${error.message}`;
  }
}

const addBreadcrumbQualityFiltered = addBreadcrumb;
addBreadcrumb = function addBreadcrumbWithPause(position) {
  if (breadcrumbRecordingPaused) return false;
  const accepted = addBreadcrumbQualityFiltered(position);
  if (!accepted) return false;
  const item = liveBreadcrumbs[liveBreadcrumbs.length - 1];
  if (item) item.recordingSegment = recordingSegmentID;
  startNewRecordingSegment = false;
  persistFieldSession();
  return true;
};

const sessionStatisticsBeforePause = sessionStatistics;
sessionStatistics = function sessionStatisticsWithPauseSegments() {
  if (!liveBreadcrumbs.length) return null;
  let distanceM = 0, movingTimeS = 0, maxSpeedMPS = 0, acceptedSpeedSamples = 0;
  for (let index = 1; index < liveBreadcrumbs.length; index += 1) {
    const previous = liveBreadcrumbs[index - 1], current = liveBreadcrumbs[index];
    if ((previous.recordingSegment ?? 0) !== (current.recordingSegment ?? 0)) continue;
    const segmentDistance = breadcrumbDistanceMeters(previous, current); distanceM += segmentDistance;
    const deltaS = (Date.parse(current.time) - Date.parse(previous.time)) / 1000;
    if (!Number.isFinite(deltaS) || deltaS <= 0) continue;
    movingTimeS += deltaS;
    const speedMPS = segmentDistance / deltaS;
    if (Number.isFinite(speedMPS) && speedMPS >= 0 && speedMPS <= 100) { maxSpeedMPS = Math.max(maxSpeedMPS, speedMPS); acceptedSpeedSamples += 1; }
  }
  const firstTime = Date.parse(liveBreadcrumbs[0].time), lastTime = Date.parse(liveBreadcrumbs[liveBreadcrumbs.length - 1].time);
  const durationS = Number.isFinite(firstTime) && Number.isFinite(lastTime) && lastTime >= firstTime ? (lastTime - firstTime) / 1000 : 0;
  return {distanceM, durationS, movingTimeS, averageSpeedMPS:movingTimeS > 0 ? distanceM / movingTimeS : 0, maxSpeedMPS, acceptedSpeedSamples};
};
void sessionStatisticsBeforePause;

function markerGPX() {
  return liveMarkers.map((marker, index) => { const type=escapeXML(marker.type); const note=marker.note?`<desc>${escapeXML(marker.note)}</desc>`:''; return `  <wpt lat="${marker.latitude.toFixed(7)}" lon="${marker.longitude.toFixed(7)}"><time>${escapeXML(marker.time)}</time><name>${escapeXML(`${marker.type} ${index + 1}`)}</name><type>${type}</type>${note}</wpt>`; }).join('\n');
}
function recordingSegmentsForExport() { const groups=[]; for(const point of liveBreadcrumbs){const segmentID=point.recordingSegment??0;let group=groups[groups.length-1];if(!group||group.segmentID!==segmentID){group={segmentID,points:[]};groups.push(group);}group.points.push(point);}return groups; }
function markerOnlyGPX() { return `<?xml version="1.0" encoding="UTF-8"?>\n<gpx version="1.1" creator="Caching Tools" xmlns="http://www.topografix.com/GPX/1/1">\n${markerGPX()}\n</gpx>\n`; }
const breadcrumbGPXWithQuality = breadcrumbGPX;
breadcrumbGPX = function breadcrumbGPXWithMarkersAndSegments() {
  if (!liveBreadcrumbs.length) { if (liveMarkers.length) return markerOnlyGPX(); return breadcrumbGPXWithQuality(); }
  const target = fieldForm.elements['target-id'].selectedOptions[0]?.textContent || 'Live navigation session';
  const tolerance = qualityNumber(fieldSimplifyTolerance, 5, true), groups = recordingSegmentsForExport(); let exportedPointCount = 0;
  const segments = groups.map(group => { const source=group.points; const exportPoints=fieldExportSimplify.checked?simplifyBreadcrumbs(source,tolerance):source; exportedPointCount+=exportPoints.length; const points=exportPoints.map(item=>`      <trkpt lat="${item.latitude.toFixed(7)}" lon="${item.longitude.toFixed(7)}"><time>${escapeXML(item.time)}</time></trkpt>`).join('\n'); return `    <trkseg>\n${points}\n    </trkseg>`; }).join('\n');
  if (fieldExportSimplify.checked) fieldRecordingStatus.textContent = `GPX simplification: ${liveBreadcrumbs.length} → ${exportedPointCount} points across ${groups.length} recording segment${groups.length === 1 ? '' : 's'}.`;
  const markers=liveMarkers.length?`${markerGPX()}\n`:'';
  return `<?xml version="1.0" encoding="UTF-8"?>\n<gpx version="1.1" creator="Caching Tools" xmlns="http://www.topografix.com/GPX/1/1">\n${markers}  <trk>\n    <name>${escapeXML(`Caching Tools session - ${target}`)}</name>\n${segments}\n  </trk>\n</gpx>\n`;
};

const startLiveNavigationWithQuality = startLiveNavigation;
startLiveNavigation = function startLiveNavigationWithSessionReset() {
  liveMarkers = [];
  liveBreadcrumbs.splice(0, liveBreadcrumbs.length);
  breadcrumbRecordingPaused = false;
  recordingSegmentID = 0;
  startNewRecordingSegment = false;
  discardRecoveredSession();
  renderRecordingStatus();
  renderMarkers();
  startLiveNavigationWithQuality();
};

fieldRecordingPause.addEventListener('click', pauseBreadcrumbRecording);
fieldRecordingResume.addEventListener('click', resumeBreadcrumbRecording);
sessionControls.querySelector('#field-marker-add').addEventListener('click', addManualMarker);
sessionControls.querySelector('#field-marker-clear').addEventListener('click', () => { liveMarkers = []; renderMarkers(); fieldRecordingStatus.textContent = 'Manual markers cleared.'; persistFieldSession(); });
fieldMarkerPromoteSelect.addEventListener('change', () => { const index=Number(fieldMarkerPromoteSelect.value),marker=liveMarkers[index];fieldMarkerWaypointName.value=marker?suggestedWaypointName(marker,index):'';fieldMarkerPromote.disabled=!marker||Boolean(marker.promotedWaypointID); });
fieldMarkerPromote.addEventListener('click', promoteManualMarker);
fieldSessionRestore.addEventListener('click', restoreRecoveredSession);
fieldSessionDiscard.addEventListener('click', discardRecoveredSession);

renderRecordingStatus();
renderMarkers();
loadRecoveredSession();
const sessionFooter = document.querySelector('footer');
if (sessionFooter) sessionFooter.textContent = 'Caching Tools M1.51';
