async function saveIntersectionPoint(point, kind, index, status) {
  const payload = {
    name: `Intersection ${kind} ${index + 1}`,
    latitude: String(point.latitude),
    longitude: String(point.longitude),
    type: 'intersection',
    comment: `Saved from ${kind} intersection solution ${index + 1}`
  };
  try {
    await postJSON('/api/waypoints', payload);
    status.textContent = `${payload.name} saved as waypoint.`;
    if (typeof loadWaypoints === 'function') await loadWaypoints();
    if (typeof window.refreshLocalMap === 'function') await window.refreshLocalMap();
  } catch (error) {
    status.textContent = `Error: ${error.message}`;
  }
}

function renderIntersectionPoints(data, output, kind) {
  output.replaceChildren();
  if (!data.points || data.points.length === 0) {
    output.textContent = 'No solutions.';
    return;
  }
  data.points.forEach((p, i) => {
    const row = document.createElement('div');
    row.className = 'intersection-solution';
    const text = document.createElement('pre');
    text.textContent = [
      `Solution ${i + 1}`,
      `DD   ${p.lat.dd}, ${p.lon.dd}`,
      `DMM  ${p.lat.dmm}, ${p.lon.dmm}`,
      `DMS  ${p.lat.dms}, ${p.lon.dms}`
    ].join('\n');
    const save = document.createElement('button');
    save.type = 'button';
    save.textContent = `Save solution ${i + 1}`;
    const status = document.createElement('span');
    status.className = 'intersection-save-status';
    status.setAttribute('aria-live', 'polite');
    save.addEventListener('click', () => saveIntersectionPoint(p, kind, i, status));
    row.append(text, save, status);
    output.append(row);
  });
}

function bindIntersectionForm(selector, outputSelector, endpoint, kind, payloadBuilder) {
  document.querySelector(selector).addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const output = document.querySelector(outputSelector);
    try {
      const data = await postJSON(endpoint, payloadBuilder(form));
      renderIntersectionPoints(data, output, kind);
    } catch (error) {
      output.textContent = `Error: ${error.message}`;
    }
  });
}

bindIntersectionForm('#intersection-bearing-form', '#intersection-bearing-result', '/api/coordinates/intersection/bearing-bearing', 'bearing-bearing', (form) => ({
  a: {latitude: form.get('a-lat'), longitude: form.get('a-lon')},
  bearing_a_deg: Number(form.get('a-bearing')),
  b: {latitude: form.get('b-lat'), longitude: form.get('b-lon')},
  bearing_b_deg: Number(form.get('b-bearing'))
}));

bindIntersectionForm('#intersection-distance-form', '#intersection-distance-result', '/api/coordinates/intersection/bearing-distance', 'bearing-distance', (form) => ({
  from: {latitude: form.get('from-lat'), longitude: form.get('from-lon')},
  bearing_deg: Number(form.get('bearing')),
  center: {latitude: form.get('center-lat'), longitude: form.get('center-lon')},
  distance_m: Number(form.get('distance'))
}));

bindIntersectionForm('#intersection-circle-form', '#intersection-circle-result', '/api/coordinates/intersection/circle-circle', 'circle-circle', (form) => ({
  a: {latitude: form.get('a-lat'), longitude: form.get('a-lon')},
  radius_a_m: Number(form.get('a-radius')),
  b: {latitude: form.get('b-lat'), longitude: form.get('b-lon')},
  radius_b_m: Number(form.get('b-radius'))
}));
