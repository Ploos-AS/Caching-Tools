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
let recordingSegmentID = 0;
let startNewRecordingSegment = false;
let liveMarkers = [];

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
}

function resumeBreadcrumbRecording() {
  if (!breadcrumbRecordingPaused) return;
  breadcrumbRecordingPaused = false;
  recordingSegmentID += 1;
  startNewRecordingSegment = true;
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
  const accepted = addBreadcrumbQualityFiltered(position);
  if (!accepted) return false;
  const item = liveBreadcrumbs[liveBreadcrumbs.length - 1];
  if (item) item.recordingSegment = recordingSegmentID;
  startNewRecordingSegment = false;
  return true;
};

const sessionStatisticsBeforePause = sessionStatistics;
sessionStatistics = function sessionStatisticsWithPauseSegments() {
  if (!liveBreadcrumbs.length) return null;
  let distanceM = 0;
  let movingTimeS = 0;
  let maxSpeedMPS = 0;
  let acceptedSpeedSamples = 0;
  for (let index = 1; index < liveBreadcrumbs.length; index += 1) {
    const previous = liveBreadcrumbs[index - 1];
    const current = liveBreadcrumbs[index];
    if ((previous.recordingSegment ?? 0) !== (current.recordingSegment ?? 0)) continue;
    const segmentDistance = breadcrumbDistanceMeters(previous, current);
    distanceM += segmentDistance;
    const previousTime = Date.parse(previous.time);
    const currentTime = Date.parse(current.time);
    const deltaS = (currentTime - previousTime) / 1000;
    if (!Number.isFinite(deltaS) || deltaS <= 0) continue;
    movingTimeS += deltaS;
    const speedMPS = segmentDistance / deltaS;
    if (Number.isFinite(speedMPS) && speedMPS >= 0 && speedMPS <= 100) {
      maxSpeedMPS = Math.max(maxSpeedMPS, speedMPS);
      acceptedSpeedSamples += 1;
    }
  }
  const firstTime = Date.parse(liveBreadcrumbs[0].time);
  const lastTime = Date.parse(liveBreadcrumbs[liveBreadcrumbs.length - 1].time);
  const durationS = Number.isFinite(firstTime) && Number.isFinite(lastTime) && lastTime >= firstTime ? (lastTime - firstTime) / 1000 : 0;
  const averageSpeedMPS = movingTimeS > 0 ? distanceM / movingTimeS : 0;
  return {distanceM, durationS, movingTimeS, averageSpeedMPS, maxSpeedMPS, acceptedSpeedSamples};
};
void sessionStatisticsBeforePause;

function markerGPX() {
  return liveMarkers.map((marker, index) => {
    const type = escapeXML(marker.type);
    const note = marker.note ? `<desc>${escapeXML(marker.note)}</desc>` : '';
    return `  <wpt lat="${marker.latitude.toFixed(7)}" lon="${marker.longitude.toFixed(7)}"><time>${escapeXML(marker.time)}</time><name>${escapeXML(`${marker.type} ${index + 1}`)}</name><type>${type}</type>${note}</wpt>`;
  }).join('\n');
}

function recordingSegmentsForExport() {
  const groups = [];
  for (const point of liveBreadcrumbs) {
    const segmentID = point.recordingSegment ?? 0;
    let group = groups[groups.length - 1];
    if (!group || group.segmentID !== segmentID) {
      group = {segmentID, points: []};
      groups.push(group);
    }
    group.points.push(point);
  }
  return groups;
}

const breadcrumbGPXWithQuality = breadcrumbGPX;
breadcrumbGPX = function breadcrumbGPXWithMarkersAndSegments() {
  if (!liveBreadcrumbs.length) return breadcrumbGPXWithQuality();
  const target = fieldForm.elements['target-id'].selectedOptions[0]?.textContent || 'Live navigation session';
  const tolerance = qualityNumber(fieldSimplifyTolerance, 5, true);
  const groups = recordingSegmentsForExport();
  let exportedPointCount = 0;
  const segments = groups.map(group => {
    const source = group.points;
    const exportPoints = fieldExportSimplify.checked ? simplifyBreadcrumbs(source, tolerance) : source;
    exportedPointCount += exportPoints.length;
    const points = exportPoints.map(item =>
      `      <trkpt lat="${item.latitude.toFixed(7)}" lon="${item.longitude.toFixed(7)}"><time>${escapeXML(item.time)}</time></trkpt>`
    ).join('\n');
    return `    <trkseg>\n${points}\n    </trkseg>`;
  }).join('\n');
  if (fieldExportSimplify.checked) {
    fieldRecordingStatus.textContent = `GPX simplification: ${liveBreadcrumbs.length} → ${exportedPointCount} points across ${groups.length} recording segment${groups.length === 1 ? '' : 's'}.`;
  }
  const markers = liveMarkers.length ? `${markerGPX()}\n` : '';
  return `<?xml version="1.0" encoding="UTF-8"?>\n<gpx version="1.1" creator="Caching Tools" xmlns="http://www.topografix.com/GPX/1/1">\n${markers}  <trk>\n    <name>${escapeXML(`Caching Tools session - ${target}`)}</name>\n${segments}\n  </trk>\n</gpx>\n`;
};

const startLiveNavigationWithQuality = startLiveNavigation;
startLiveNavigation = function startLiveNavigationWithSessionReset() {
  liveMarkers = [];
  breadcrumbRecordingPaused = false;
  recordingSegmentID = 0;
  startNewRecordingSegment = false;
  renderRecordingStatus();
  renderMarkers();
  startLiveNavigationWithQuality();
};

fieldRecordingPause.addEventListener('click', pauseBreadcrumbRecording);
fieldRecordingResume.addEventListener('click', resumeBreadcrumbRecording);
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
