async function requestJSON(url, options = {}) {
  const response = await fetch(url, options);
  if (response.status === 204) return null;
  const body = await response.json();
  if (!response.ok) throw new Error(body.error || `HTTP ${response.status}`);
  return body;
}

async function postJSON(url, payload) {
  return requestJSON(url, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(payload)
  });
}

function formatPoint(data) {
  return [
    `DD   ${data.lat.dd}, ${data.lon.dd}`,
    `DMM  ${data.lat.dmm}, ${data.lon.dmm}`,
    `DMS  ${data.lat.dms}, ${data.lon.dms}`
  ].join('\n');
}

const waypointForm = document.querySelector('#waypoint-form');
const waypointList = document.querySelector('#waypoint-list');
const waypointStatus = document.querySelector('#waypoint-status');
const waypointCancel = document.querySelector('#waypoint-cancel');
const gpxImportForm = document.querySelector('#gpx-import-form');
const gpxStatus = document.querySelector('#gpx-status');
const gpxInspection = document.querySelector('#gpx-inspection');
let waypointCache = [];

function resetWaypointForm() {
  waypointForm.elements['waypoint-id'].value = '';
  waypointForm.elements['waypoint-name'].value = '';
  waypointForm.elements['waypoint-type'].value = '';
  waypointForm.elements['waypoint-lat'].value = '';
  waypointForm.elements['waypoint-lon'].value = '';
  waypointForm.elements['waypoint-comment'].value = '';
  waypointCancel.hidden = true;
  document.querySelector('#waypoint-save').textContent = 'Save waypoint';
}

function waypointPayload(form) {
  return {
    name: form.get('waypoint-name'),
    latitude: form.get('waypoint-lat'),
    longitude: form.get('waypoint-lon'),
    type: form.get('waypoint-type'),
    comment: form.get('waypoint-comment')
  };
}

function renderWaypoints(items) {
  waypointCache = items;
  waypointList.replaceChildren();
  if (items.length === 0) {
    waypointList.textContent = 'No saved waypoints.';
    return;
  }
  for (const item of items) {
    const row = document.createElement('div');
    const text = document.createElement('pre');
    text.textContent = `${item.name}${item.type ? ` [${item.type}]` : ''}\n${item.point.lat.dmm}, ${item.point.lon.dmm}${item.comment ? `\n${item.comment}` : ''}`;
    const edit = document.createElement('button');
    edit.type = 'button';
    edit.textContent = 'Edit';
    edit.addEventListener('click', () => editWaypoint(item.id));
    const del = document.createElement('button');
    del.type = 'button';
    del.textContent = 'Delete';
    del.addEventListener('click', () => deleteWaypoint(item.id));
    row.append(text, edit, del);
    waypointList.append(row);
  }
}

async function loadWaypoints() {
  try {
    renderWaypoints(await requestJSON('/api/waypoints'));
  } catch (error) {
    waypointList.textContent = `Error: ${error.message}`;
  }
}

function editWaypoint(id) {
  const item = waypointCache.find((candidate) => candidate.id === id);
  if (!item) return;
  waypointForm.elements['waypoint-id'].value = item.id;
  waypointForm.elements['waypoint-name'].value = item.name;
  waypointForm.elements['waypoint-type'].value = item.type || '';
  waypointForm.elements['waypoint-lat'].value = item.point.latitude;
  waypointForm.elements['waypoint-lon'].value = item.point.longitude;
  waypointForm.elements['waypoint-comment'].value = item.comment || '';
  waypointCancel.hidden = false;
  document.querySelector('#waypoint-save').textContent = 'Update waypoint';
  waypointForm.scrollIntoView({behavior: 'smooth', block: 'center'});
}

async function deleteWaypoint(id) {
  try {
    await requestJSON(`/api/waypoints/${encodeURIComponent(id)}`, {method: 'DELETE'});
    waypointStatus.textContent = 'Waypoint deleted.';
    await loadWaypoints();
  } catch (error) {
    waypointStatus.textContent = `Error: ${error.message}`;
  }
}

waypointForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const id = form.get('waypoint-id');
  const payload = waypointPayload(form);
  try {
    if (id) {
      await requestJSON(`/api/waypoints/${encodeURIComponent(id)}`, {
        method: 'PUT', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(payload)
      });
      waypointStatus.textContent = 'Waypoint updated.';
    } else {
      await postJSON('/api/waypoints', payload);
      waypointStatus.textContent = 'Waypoint saved.';
    }
    resetWaypointForm();
    await loadWaypoints();
  } catch (error) {
    waypointStatus.textContent = `Error: ${error.message}`;
  }
});

waypointCancel.addEventListener('click', resetWaypointForm);

function selectedGPXFile() {
  return document.querySelector('#gpx-file').files[0];
}

function renderGPXInspection(data) {
  const lines = [
    `GPX ${data.version}${data.creator ? ` — ${data.creator}` : ''}`,
    `Waypoints: ${data.waypoints}`,
    `Routes: ${data.routes.length}`
  ];
  for (const route of data.routes) {
    lines.push(`  ${route.name}: ${route.points} points, ${route.distance_km.toFixed(3)} km`);
  }
  lines.push(`Tracks: ${data.tracks.length}`);
  for (const track of data.tracks) {
    lines.push(`  ${track.name}: ${track.points} points, ${track.segments} segment${track.segments === 1 ? '' : 's'}, ${track.distance_km.toFixed(3)} km`);
  }
  gpxInspection.textContent = lines.join('\n');
}

document.querySelector('#gpx-inspect').addEventListener('click', async () => {
  const file = selectedGPXFile();
  if (!file) {
    gpxInspection.textContent = 'Select a GPX file first.';
    return;
  }
  try {
    const data = await requestJSON('/api/gpx/inspect', {
      method: 'POST',
      headers: {'Content-Type': 'application/gpx+xml'},
      body: file
    });
    renderGPXInspection(data);
  } catch (error) {
    gpxInspection.textContent = `Error: ${error.message}`;
  }
});

gpxImportForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const file = selectedGPXFile();
  if (!file) return;
  try {
    const data = await requestJSON('/api/gpx/waypoints/import', {
      method: 'POST',
      headers: {'Content-Type': 'application/gpx+xml'},
      body: file
    });
    gpxStatus.textContent = `Imported ${data.imported} waypoint${data.imported === 1 ? '' : 's'}.`;
    gpxImportForm.reset();
    await loadWaypoints();
  } catch (error) {
    gpxStatus.textContent = `Error: ${error.message}`;
  }
});

loadWaypoints();

document.querySelector('#convert-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const output = document.querySelector('#convert-result');
  try {
    const data = await postJSON('/api/coordinates/convert', {
      latitude: form.get('latitude'), longitude: form.get('longitude')
    });
    output.textContent = formatPoint(data);
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});

document.querySelector('#grid-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const output = document.querySelector('#grid-result');
  try {
    const data = await postJSON('/api/coordinates/grid', {
      latitude: form.get('grid-lat'), longitude: form.get('grid-lon')
    });
    output.textContent = [
      `UTM  ${data.utm.zone}${data.utm.band} ${data.utm.easting.toFixed(3)} E ${data.utm.northing.toFixed(3)} N`,
      `MGRS ${data.utm.mgrs}`,
      '',
      formatPoint(data.wgs84)
    ].join('\n');
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});

document.querySelector('#utm-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const output = document.querySelector('#utm-result');
  try {
    const data = await postJSON('/api/coordinates/from-utm', {
      zone: Number(form.get('utm-zone')),
      hemisphere: String(form.get('utm-hemisphere')).toUpperCase(),
      easting: Number(form.get('utm-easting')),
      northing: Number(form.get('utm-northing'))
    });
    output.textContent = `${formatPoint(data.wgs84)}\n\nMGRS ${data.utm.mgrs}`;
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});

document.querySelector('#nav-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const output = document.querySelector('#nav-result');
  try {
    const data = await postJSON('/api/coordinates/navigation', {
      from: {latitude: form.get('from-lat'), longitude: form.get('from-lon')},
      to: {latitude: form.get('to-lat'), longitude: form.get('to-lon')}
    });
    output.textContent = `Distance: ${data.distance_km.toFixed(3)} km\nInitial bearing: ${data.initial_bearing_deg.toFixed(2)}°`;
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});

document.querySelector('#project-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const output = document.querySelector('#project-result');
  try {
    const data = await postJSON('/api/coordinates/project', {
      from: {latitude: form.get('project-lat'), longitude: form.get('project-lon')},
      bearing_deg: Number(form.get('bearing')),
      distance_m: Number(form.get('distance'))
    });
    output.textContent = `Projected waypoint\n${formatPoint(data.to)}\n\nBearing: ${data.bearing_deg.toFixed(2)}°\nDistance: ${data.distance_m.toFixed(1)} m`;
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});
