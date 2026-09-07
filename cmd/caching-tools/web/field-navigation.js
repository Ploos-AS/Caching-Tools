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
  </form>
  <pre id="field-navigation-result" class="result">Choose a saved target.</pre>`;

const navSection = document.querySelector('#nav-form')?.closest('.tool');
if (navSection) navSection.insertAdjacentElement('beforebegin', fieldSection);
else document.querySelector('main')?.append(fieldSection);

const fieldForm = fieldSection.querySelector('#field-navigation-form');
const fieldResult = fieldSection.querySelector('#field-navigation-result');
let fieldWaypoints = [];
let fieldPaths = [];

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

fieldForm.elements['target-type'].addEventListener('change', renderFieldTargets);
fieldForm.addEventListener('submit', async event => {
  event.preventDefault();
  const form = new FormData(fieldForm);
  const id = String(form.get('target-id') || '');
  if (!id) { fieldResult.textContent = 'No saved target is available.'; return; }
  const payload = {
    from:{latitude:String(form.get('latitude')), longitude:String(form.get('longitude'))},
    off_route_threshold_m:Number(form.get('off-route-threshold')),
    arrival_radius_m:Number(form.get('arrival-radius'))
  };
  if (form.get('target-type') === 'waypoint') payload.waypoint_id = id; else payload.path_id = id;
  try {
    fieldResult.textContent = formatFieldNavigation(await postJSON('/api/navigation/field', payload));
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

document.addEventListener('caching-tools:map-select', event => {
  const detail = event.detail || {};
  if (detail.kind === 'waypoint') fieldForm.elements['target-type'].value = 'waypoint';
  else if (detail.kind === 'route' || detail.kind === 'track') fieldForm.elements['target-type'].value = 'path';
  else return;
  renderFieldTargets();
  fieldForm.elements['target-id'].value = detail.id || '';
});

loadFieldTargets().catch(error => { fieldResult.textContent = `Error: ${error.message}`; });
const fieldFooter = document.querySelector('footer'); if (fieldFooter) fieldFooter.textContent = 'Caching Tools M1.24';
