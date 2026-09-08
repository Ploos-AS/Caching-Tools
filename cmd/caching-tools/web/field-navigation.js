const fieldSection = document.createElement('section');
fieldSection.className = 'tool';
fieldSection.id = 'field-navigation';
fieldSection.innerHTML = `
  <h2>Field navigation</h2>
  <p>Navigate from your current position to a saved waypoint or follow a saved route/track with nearest-path, progress, deviation and arrival guidance. All calculations stay local.</p>
  <form id="field-navigation-form" class="form-grid nav-grid">
    <label>Current latitude<input name="latitude" value="59.9139" required></label>
    <label>Current longitude<input name="longitude" value="10.7522" required></label>
    <label>Target type<select name="target-type"><option value="waypoint">Waypoint</option><option value="path">Route / track</option></select></label>
    <label>Target<select name="target-id"></select></label>
    <label>Off-route threshold (m)<input name="off-route-threshold" type="number" min="1" step="1" value="50" required></label>
    <label>Arrival radius (m)<input name="arrival-radius" type="number" min="1" step="1" value="20" required></label>
    <button type="button" id="field-use-location">Use browser location</button>
    <button type="submit">Go to / route guidance</button>
    <button type="button" id="field-live-start">Start live navigation</button>
    <button type="button" id="field-live-stop" disabled>Stop live navigation</button>
  </form>
  <p id="field-live-status" aria-live="polite">Live navigation stopped.</p>
  <pre id="field-navigation-result" class="result">Choose a saved target.</pre>
  <h3>Session statistics</h3>
  <pre id="field-session-stats" class="result">No session statistics yet.</pre>
  <h3>Session breadcrumbs</h3>
  <p>Breadcrumbs exist only in browser memory for the current live session and are never written to <code>/data</code>. Export is explicit and downloads a GPX 1.1 track locally.</p>
  <button type="button" id="field-breadcrumb-export">Export session GPX</button>
  <button type="button" id="field-breadcrumb-clear">Clear breadcrumbs</button>
  <pre id="field-breadcrumbs" class="result">No breadcrumb points.</pre>`;

const navSection = document.querySelector('#nav-form')?.closest('.tool');
if (navSection) navSection.insertAdjacentElement('beforebegin', fieldSection);
else document.querySelector('main')?.append(fieldSection);

const fieldForm = fieldSection.querySelector('#field-navigation-form');
const fieldResult = fieldSection.querySelector('#field-navigation-result');
const fieldLiveStatus = fieldSection.querySelector('#field-live-status');
const fieldSessionStats = fieldSection.querySelector('#field-session-stats');
const fieldBreadcrumbs = fieldSection.querySelector('#field-breadcrumbs');
const fieldLiveStart = fieldSection.querySelector('#field-live-start');
const fieldLiveStop = fieldSection.querySelector('#field-live-stop');
let fieldWaypoints = [];
let fieldPaths = [];
let liveWatchID = null;
let liveRequestInFlight = false;
let livePendingPosition = null;
let liveGeneration = 0;
let liveBreadcrumbs = [];

async function loadFieldTargets() {
  [fieldWaypoints, fieldPaths] = await Promise.all([requestJSON('/api/waypoints'), requestJSON('/api/paths')]);
  renderFieldTargets();
}

function renderFieldTargets() {
  const type = fieldForm.elements['target-type'].value;
  const select = fieldForm.elements['target-id'];
  select.replaceChildren();
  const items = type === 'waypoint' ? fieldWaypoints : fieldPaths;
  if (!items.length) {
    const option = document.createElement('option'); option.value=''; option.textContent = type === 'waypoint' ? 'No saved waypoints' : 'No saved routes/tracks'; select.append(option); return;
  }
  for (const item of items) {
    const option = document.createElement('option'); option.value = item.id; option.textContent = type === 'waypoint' ? item.name : `${item.kind}: ${item.summary.name}`; select.append(option);
  }
}

function formatFieldNavigation(data) {
  const lines = [
    `${data.kind}: ${data.name}`,
    `Status: ${data.guidance.status}`,
    data.guidance.message,
    `Distance: ${data.distance_m.toFixed(1)} m (${data.distance_km.toFixed(3)} km)`,
    `Bearing: ${data.bearing_deg.toFixed(1)}°`,
    `Target: ${data.target.lat.dmm}, ${data.target.lon.dmm}`,
    `Arrival radius: ${data.guidance.arrival_radius_m.toFixed(0)} m`
  ];
  if (data.kind !== 'waypoint') lines.push(`Off-route threshold: ${data.guidance.off_route_threshold_m.toFixed(0)} m`);
  if (data.cross_track_m != null) lines.push(`Cross-track / nearest-path distance: ${data.cross_track_m.toFixed(1)} m`);
  if (data.nearest_path) lines.push(`Nearest path location: segment ${data.nearest_path.segment + 1}, edge ${data.nearest_path.edge + 1}`);
  if (data.progress) {
    lines.push(
      '',
      `Progress: ${data.progress.progress_percent.toFixed(1)}%`,
      `Along path: ${(data.progress.along_m / 1000).toFixed(3)} km of ${(data.progress.total_m / 1000).toFixed(3)} km`,
      `Remaining along path: ${(data.progress.remaining_m / 1000).toFixed(3)} km`,
      `Next point: ${data.progress.next_point.lat.dmm}, ${data.progress.next_point.lon.dmm}`,
      `Distance to next point from nearest path position: ${data.progress.next_point_distance_m.toFixed(1)} m`,
      `Forward bearing: ${data.progress.forward_bearing_deg.toFixed(1)}°`
    );
  }
  return lines.join('\n');
}

function fieldPayload(latitude, longitude) {
  const form = new FormData(fieldForm);
  const id = String(form.get('target-id') || '');
  if (!id) throw new Error('No saved target is available.');
  const payload = {
    from:{latitude:String(latitude), longitude:String(longitude)},
    off_route_threshold_m:Number(form.get('off-route-threshold')),
    arrival_radius_m:Number(form.get('arrival-radius'))
  };
  if (form.get('target-type') === 'waypoint') payload.waypoint_id = id; else payload.path_id = id;
  return payload;
}

async function navigateField(latitude, longitude) {
  return postJSON('/api/navigation/field', fieldPayload(latitude, longitude));
}

function breadcrumbDistanceMeters(a, b) {
  const radius = 6371008.8;
  const radians = value => value * Math.PI / 180;
  const lat1 = radians(a.latitude), lat2 = radians(b.latitude);
  const dLat = lat2 - lat1;
  const dLon = radians(b.longitude - a.longitude);
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) ** 2;
  return 2 * radius * Math.asin(Math.min(1, Math.sqrt(h)));
}

function sessionStatistics() {
  if (!liveBreadcrumbs.length) return null;
  let distanceM = 0;
  let movingTimeS = 0;
  let maxSpeedMPS = 0;
  let acceptedSpeedSamples = 0;
  for (let index = 1; index < liveBreadcrumbs.length; index += 1) {
    const previous = liveBreadcrumbs[index - 1];
    const current = liveBreadcrumbs[index];
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
}

function formatDuration(seconds) {
  const total = Math.max(0, Math.round(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  return `${hours.toString().padStart(2,'0')}:${minutes.toString().padStart(2,'0')}:${secs.toString().padStart(2,'0')}`;
}

function renderSessionStatistics() {
  const stats = sessionStatistics();
  if (!stats) {
    fieldSessionStats.textContent = 'No session statistics yet.';
    return;
  }
  fieldSessionStats.textContent = [
    `Distance: ${(stats.distanceM / 1000).toFixed(3)} km (${stats.distanceM.toFixed(1)} m)`,
    `Session duration: ${formatDuration(stats.durationS)}`,
    `Timed movement: ${formatDuration(stats.movingTimeS)}`,
    `Average speed: ${(stats.averageSpeedMPS * 3.6).toFixed(1)} km/h`,
    `Maximum accepted segment speed: ${(stats.maxSpeedMPS * 3.6).toFixed(1)} km/h`,
    `Speed samples: ${stats.acceptedSpeedSamples}`,
    'Speed samples above 360 km/h and non-positive/invalid time deltas are ignored for maximum-speed statistics.'
  ].join('\n');
}

function renderBreadcrumbs() {
  renderSessionStatistics();
  if (!liveBreadcrumbs.length) {
    fieldBreadcrumbs.textContent = 'No breadcrumb points.';
    return;
  }
  const recent = liveBreadcrumbs.slice(-12);
  const lines = [`${liveBreadcrumbs.length} breadcrumb point${liveBreadcrumbs.length === 1 ? '' : 's'} in this in-memory session.`];
  if (liveBreadcrumbs.length > recent.length) lines.push(`Showing the latest ${recent.length}:`);
  for (const item of recent) {
    lines.push(`${item.time}  ${item.latitude.toFixed(7)}, ${item.longitude.toFixed(7)}  ±${Math.round(item.accuracy)} m`);
  }
  fieldBreadcrumbs.textContent = lines.join('\n');
}

function addBreadcrumb(position) {
  liveBreadcrumbs.push({
    latitude: position.coords.latitude,
    longitude: position.coords.longitude,
    accuracy: Number.isFinite(position.coords.accuracy) ? position.coords.accuracy : 0,
    time: new Date(position.timestamp || Date.now()).toISOString()
  });
  if (liveBreadcrumbs.length > 1000) liveBreadcrumbs.shift();
  renderBreadcrumbs();
}

function escapeXML(value) {
  return String(value).replace(/[&<>"']/g, character => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&apos;'}[character]));
}

function breadcrumbGPX() {
  if (!liveBreadcrumbs.length) throw new Error('No breadcrumb points to export.');
  const target = fieldForm.elements['target-id'].selectedOptions[0]?.textContent || 'Live navigation session';
  const points = liveBreadcrumbs.map(item =>
    `      <trkpt lat="${item.latitude.toFixed(7)}" lon="${item.longitude.toFixed(7)}"><time>${escapeXML(item.time)}</time></trkpt>`
  ).join('\n');
  return `<?xml version="1.0" encoding="UTF-8"?>\n<gpx version="1.1" creator="Caching Tools" xmlns="http://www.topografix.com/GPX/1/1">\n  <trk>\n    <name>${escapeXML(`Caching Tools session - ${target}`)}</name>\n    <trkseg>\n${points}\n    </trkseg>\n  </trk>\n</gpx>\n`;
}

function exportBreadcrumbGPX() {
  try {
    const data = breadcrumbGPX();
    const blob = new Blob([data], {type:'application/gpx+xml;charset=utf-8'});
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    const stamp = new Date().toISOString().replace(/[-:]/g,'').replace(/\..*$/,'').replace('T','-');
    link.href = url;
    link.download = `caching-tools-session-${stamp}.gpx`;
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    fieldLiveStatus.textContent = `Exported ${liveBreadcrumbs.length} breadcrumb point${liveBreadcrumbs.length === 1 ? '' : 's'} as GPX 1.1.`;
  } catch (error) {
    fieldLiveStatus.textContent = `Session export error: ${error.message}`;
  }
}

async function processLivePosition(position, generation) {
  if (generation !== liveGeneration || liveWatchID === null) return;
  livePendingPosition = position;
  if (liveRequestInFlight) return;
  liveRequestInFlight = true;
  try {
    while (livePendingPosition && generation === liveGeneration && liveWatchID !== null) {
      const current = livePendingPosition;
      livePendingPosition = null;
      fieldForm.elements.latitude.value = current.coords.latitude.toFixed(7);
      fieldForm.elements.longitude.value = current.coords.longitude.toFixed(7);
      addBreadcrumb(current);
      try {
        const result = await navigateField(current.coords.latitude, current.coords.longitude);
        if (generation === liveGeneration && liveWatchID !== null) {
          fieldResult.textContent = formatFieldNavigation(result);
          fieldLiveStatus.textContent = `Live navigation active · accuracy about ${Math.round(current.coords.accuracy)} m · ${result.guidance.status}`;
        }
      } catch (error) {
        if (generation === liveGeneration && liveWatchID !== null) fieldLiveStatus.textContent = `Live navigation error: ${error.message}`;
      }
    }
  } finally {
    liveRequestInFlight = false;
  }
}

function stopLiveNavigation(message = 'Live navigation stopped.') {
  liveGeneration += 1;
  livePendingPosition = null;
  if (liveWatchID !== null && navigator.geolocation) navigator.geolocation.clearWatch(liveWatchID);
  liveWatchID = null;
  fieldLiveStart.disabled = false;
  fieldLiveStop.disabled = true;
  fieldLiveStatus.textContent = message;
}

function startLiveNavigation() {
  if (!navigator.geolocation) { fieldLiveStatus.textContent = 'Browser geolocation is not available.'; return; }
  try {
    const form = new FormData(fieldForm);
    fieldPayload(form.get('latitude'), form.get('longitude'));
  } catch (error) {
    fieldLiveStatus.textContent = error.message;
    return;
  }
  if (liveWatchID !== null) stopLiveNavigation();
  liveBreadcrumbs = [];
  renderBreadcrumbs();
  liveGeneration += 1;
  const generation = liveGeneration;
  fieldLiveStart.disabled = true;
  fieldLiveStop.disabled = false;
  fieldLiveStatus.textContent = 'Starting live navigation…';
  liveWatchID = navigator.geolocation.watchPosition(
    position => processLivePosition(position, generation),
    error => {
      if (generation !== liveGeneration) return;
      stopLiveNavigation(`Live navigation location error: ${error.message}`);
    },
    {enableHighAccuracy:true, maximumAge:2000, timeout:15000}
  );
}

fieldForm.elements['target-type'].addEventListener('change', renderFieldTargets);
fieldForm.addEventListener('submit', async event => {
  event.preventDefault();
  const form = new FormData(fieldForm);
  try {
    fieldResult.textContent = formatFieldNavigation(await navigateField(form.get('latitude'), form.get('longitude')));
  } catch (error) {
    fieldResult.textContent = `Error: ${error.message}`;
  }
});

fieldSection.querySelector('#field-use-location').addEventListener('click', () => {
  if (!navigator.geolocation) { fieldResult.textContent = 'Browser geolocation is not available.'; return; }
  navigator.geolocation.getCurrentPosition(position => {
    fieldForm.elements.latitude.value = position.coords.latitude.toFixed(7);
    fieldForm.elements.longitude.value = position.coords.longitude.toFixed(7);
    fieldResult.textContent = `Current position loaded (accuracy about ${Math.round(position.coords.accuracy)} m).`;
  }, error => { fieldResult.textContent = `Location error: ${error.message}`; }, {enableHighAccuracy:true, maximumAge:5000, timeout:10000});
});

fieldLiveStart.addEventListener('click', startLiveNavigation);
fieldLiveStop.addEventListener('click', () => stopLiveNavigation());
fieldSection.querySelector('#field-breadcrumb-export').addEventListener('click', exportBreadcrumbGPX);
fieldSection.querySelector('#field-breadcrumb-clear').addEventListener('click', () => {
  liveBreadcrumbs = [];
  renderBreadcrumbs();
});

document.addEventListener('caching-tools:map-select', event => {
  const detail = event.detail || {};
  if (detail.kind === 'waypoint') fieldForm.elements['target-type'].value = 'waypoint';
  else if (detail.kind === 'route' || detail.kind === 'track') fieldForm.elements['target-type'].value = 'path';
  else return;
  renderFieldTargets();
  fieldForm.elements['target-id'].value = detail.id || '';
});

window.addEventListener('pagehide', () => { if (liveWatchID !== null) stopLiveNavigation(); });
loadFieldTargets().catch(error => { fieldResult.textContent = `Error: ${error.message}`; });
const fieldFooter = document.querySelector('footer'); if (fieldFooter) fieldFooter.textContent = 'Caching Tools M1.27';
