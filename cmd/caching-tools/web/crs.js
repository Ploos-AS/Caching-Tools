const crsSection = document.createElement('section');
crsSection.className = 'tool';
crsSection.id = 'crs-converter';
crsSection.innerHTML = `
  <h2>Datum / CRS converter</h2>
  <p>Local WGS84 ↔ ETRS89 conversion with explicit ETRS89 / UTM zones. WGS84 ↔ ETRS89 geographic conversion is intentionally marked approximate because no coordinate epoch or plate-motion model is applied.</p>
  <form id="crs-form" class="form-grid nav-grid">
    <label>Direction
      <select name="direction">
        <option value="wgs-to-etrs-utm">WGS84 → ETRS89 / UTM</option>
        <option value="etrs-to-utm">ETRS89 → ETRS89 / UTM</option>
        <option value="utm-to-wgs">ETRS89 / UTM → WGS84</option>
        <option value="utm-to-etrs">ETRS89 / UTM → ETRS89</option>
        <option value="wgs-to-etrs">WGS84 → ETRS89</option>
        <option value="etrs-to-wgs">ETRS89 → WGS84</option>
      </select>
    </label>
    <label>ETRS89 UTM zone
      <input name="zone" type="number" min="28" max="38" value="32">
    </label>
    <label>Latitude
      <input name="latitude" value="59.9139">
    </label>
    <label>Longitude
      <input name="longitude" value="10.7522">
    </label>
    <label>Easting
      <input name="easting" type="number" step="any" value="597979.9">
    </label>
    <label>Northing
      <input name="northing" type="number" step="any" value="6643118.9">
    </label>
    <button type="submit">Convert CRS</button>
  </form>
  <pre id="crs-result" class="result">Ready.</pre>`;

const coordinateConverter = document.querySelector('#convert-form')?.closest('.tool');
if (coordinateConverter) coordinateConverter.insertAdjacentElement('beforebegin', crsSection);
else document.querySelector('main')?.append(crsSection);

function crsPayload(form) {
  const direction = form.get('direction');
  const zone = Number(form.get('zone'));
  const geographic = {latitude:String(form.get('latitude') || ''), longitude:String(form.get('longitude') || ''), zone};
  const grid = {zone, easting:Number(form.get('easting')), northing:Number(form.get('northing')), hemisphere:'N'};
  switch (direction) {
    case 'wgs-to-etrs-utm': return {source:'EPSG:4326', target:'ETRS89-UTM', ...geographic};
    case 'etrs-to-utm': return {source:'EPSG:4258', target:'ETRS89-UTM', ...geographic};
    case 'utm-to-wgs': return {source:'ETRS89-UTM', target:'EPSG:4326', ...grid};
    case 'utm-to-etrs': return {source:'ETRS89-UTM', target:'EPSG:4258', ...grid};
    case 'wgs-to-etrs': return {source:'EPSG:4326', target:'EPSG:4258', latitude:geographic.latitude, longitude:geographic.longitude};
    case 'etrs-to-wgs': return {source:'EPSG:4258', target:'EPSG:4326', latitude:geographic.latitude, longitude:geographic.longitude};
    default: throw new Error('Unsupported CRS direction');
  }
}

function renderCRS(data) {
  const lines = [`${data.source} → ${data.target}`];
  if (data.grid) lines.push(`Zone ${data.grid.zone}N · E ${data.grid.easting.toFixed(3)} · N ${data.grid.northing.toFixed(3)}`);
  if (data.point) lines.push(`DD   ${data.point.lat.dd}, ${data.point.lon.dd}`, `DMM  ${data.point.lat.dmm}, ${data.point.lon.dmm}`, `DMS  ${data.point.lat.dms}, ${data.point.lon.dms}`);
  if (data.approximate) lines.push('', 'Approximate datum relation');
  if (data.note) lines.push(data.note);
  return lines.join('\n');
}

crsSection.querySelector('#crs-form').addEventListener('submit', async event => {
  event.preventDefault();
  const output = crsSection.querySelector('#crs-result');
  try {
    const data = await postJSON('/api/coordinates/crs', crsPayload(new FormData(event.currentTarget)));
    output.textContent = renderCRS(data);
  } catch (error) {
    output.textContent = `Error: ${error.message}`;
  }
});
