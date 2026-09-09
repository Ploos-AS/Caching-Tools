const crsSection = document.createElement('section');
crsSection.className = 'tool';
crsSection.id = 'crs-converter';
crsSection.innerHTML = `
  <h2>Datum / CRS converter</h2>
  <p>Local WGS84 ↔ ETRS89 conversion with explicit ETRS89 / UTM EPSG zones 25828–25838. WGS84 ↔ ETRS89 geographic conversion is intentionally marked approximate because no coordinate epoch or plate-motion model is applied.</p>
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
    <label>ETRS89 UTM zone / EPSG
      <select name="zone">
        <option value="28">28N · EPSG:25828</option><option value="29">29N · EPSG:25829</option><option value="30">30N · EPSG:25830</option><option value="31">31N · EPSG:25831</option><option value="32" selected>32N · EPSG:25832</option><option value="33">33N · EPSG:25833</option><option value="34">34N · EPSG:25834</option><option value="35">35N · EPSG:25835</option><option value="36">36N · EPSG:25836</option><option value="37">37N · EPSG:25837</option><option value="38">38N · EPSG:25838</option>
      </select>
    </label>
    <label>Latitude<input name="latitude" value="59.9139"></label>
    <label>Longitude<input name="longitude" value="10.7522"></label>
    <label>Easting<input name="easting" type="number" step="any" value="597979.9"></label>
    <label>Northing<input name="northing" type="number" step="any" value="6643118.9"></label>
    <button type="submit">Convert CRS</button>
  </form>
  <pre id="crs-result" class="result">Ready.</pre>`;

const coordinateConverter = document.querySelector('#convert-form')?.closest('.tool');
if (coordinateConverter) coordinateConverter.insertAdjacentElement('beforebegin', crsSection);
else document.querySelector('main')?.append(crsSection);

function crsPayload(form) {
  const direction = form.get('direction');
  const zone = Number(form.get('zone'));
  const epsg = `EPSG:${25800 + zone}`;
  const geographic = {latitude:String(form.get('latitude') || ''), longitude:String(form.get('longitude') || '')};
  const grid = {easting:Number(form.get('easting')), northing:Number(form.get('northing')), hemisphere:'N'};
  switch (direction) {
    case 'wgs-to-etrs-utm': return {source:'EPSG:4326', target:epsg, ...geographic};
    case 'etrs-to-utm': return {source:'EPSG:4258', target:epsg, ...geographic};
    case 'utm-to-wgs': return {source:epsg, target:'EPSG:4326', ...grid};
    case 'utm-to-etrs': return {source:epsg, target:'EPSG:4258', ...grid};
    case 'wgs-to-etrs': return {source:'EPSG:4326', target:'EPSG:4258', ...geographic};
    case 'etrs-to-wgs': return {source:'EPSG:4258', target:'EPSG:4326', ...geographic};
    default: throw new Error('Unsupported CRS direction');
  }
}

function renderCRS(data) {
  const lines = [`${data.source} → ${data.target}`];
  if (data.grid) lines.push(`${data.grid.crs} · Zone ${data.grid.zone}N · E ${data.grid.easting.toFixed(3)} · N ${data.grid.northing.toFixed(3)}`);
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

const crsFooter = document.querySelector('footer');
if (crsFooter) crsFooter.textContent = 'Caching Tools M1.48';
