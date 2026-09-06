async function postJSON(url, payload) {
  const response = await fetch(url, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(payload)
  });
  const body = await response.json();
  if (!response.ok) throw new Error(body.error || `HTTP ${response.status}`);
  return body;
}

function formatPoint(data) {
  return [
    `DD   ${data.lat.dd}, ${data.lon.dd}`,
    `DMM  ${data.lat.dmm}, ${data.lon.dmm}`,
    `DMS  ${data.lat.dms}, ${data.lon.dms}`
  ].join('\n');
}

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
