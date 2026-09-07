function renderIntersectionPoints(data) {
  return data.points.map((p, i) => [
    `Solution ${i + 1}`,
    `DD   ${p.lat.dd}, ${p.lon.dd}`,
    `DMM  ${p.lat.dmm}, ${p.lon.dmm}`,
    `DMS  ${p.lat.dms}, ${p.lon.dms}`
  ].join('\n')).join('\n\n');
}

function bindIntersectionForm(selector, outputSelector, endpoint, payloadBuilder) {
  document.querySelector(selector).addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const output = document.querySelector(outputSelector);
    try {
      const data = await postJSON(endpoint, payloadBuilder(form));
      output.textContent = renderIntersectionPoints(data);
    } catch (error) {
      output.textContent = `Error: ${error.message}`;
    }
  });
}

bindIntersectionForm('#intersection-bearing-form', '#intersection-bearing-result', '/api/coordinates/intersection/bearing-bearing', (form) => ({
  a: {latitude: form.get('a-lat'), longitude: form.get('a-lon')},
  bearing_a_deg: Number(form.get('a-bearing')),
  b: {latitude: form.get('b-lat'), longitude: form.get('b-lon')},
  bearing_b_deg: Number(form.get('b-bearing'))
}));

bindIntersectionForm('#intersection-distance-form', '#intersection-distance-result', '/api/coordinates/intersection/bearing-distance', (form) => ({
  from: {latitude: form.get('from-lat'), longitude: form.get('from-lon')},
  bearing_deg: Number(form.get('bearing')),
  center: {latitude: form.get('center-lat'), longitude: form.get('center-lon')},
  distance_m: Number(form.get('distance'))
}));

bindIntersectionForm('#intersection-circle-form', '#intersection-circle-result', '/api/coordinates/intersection/circle-circle', (form) => ({
  a: {latitude: form.get('a-lat'), longitude: form.get('a-lon')},
  radius_a_m: Number(form.get('a-radius')),
  b: {latitude: form.get('b-lat'), longitude: form.get('b-lon')},
  radius_b_m: Number(form.get('b-radius'))
}));
