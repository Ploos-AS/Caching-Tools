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
