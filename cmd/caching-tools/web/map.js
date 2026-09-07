const mapSvg = document.querySelector('#local-map');
const mapStatus = document.querySelector('#map-status');
const mapSelection = document.querySelector('#map-selection');
const mapNS = 'http://www.w3.org/2000/svg';
const baseWidth = 900;
const baseHeight = 520;
let currentView = {x: 0, y: 0, width: baseWidth, height: baseHeight};
let fullView = {...currentView};
let dragStart = null;
let selected = null;
let objectBounds = new Map();

function svgElement(name, attrs = {}) {
  const el = document.createElementNS(mapNS, name);
  for (const [key, value] of Object.entries(attrs)) el.setAttribute(key, String(value));
  return el;
}

function pathPoints(item) {
  if (item.kind === 'route') return [item.route || []];
  return (item.segments || []).map((segment) => segment.points || segment.Points || []);
}

function pointLat(point) { return point.latitude ?? point.Latitude; }
function pointLon(point) { return point.longitude ?? point.Longitude; }

function collectCoordinates(waypoints, paths) {
  const points = waypoints.map((w) => ({lat: w.point.latitude, lon: w.point.longitude}));
  for (const item of paths) {
    for (const segment of pathPoints(item)) {
      for (const point of segment) points.push({lat: pointLat(point), lon: pointLon(point)});
    }
  }
  return points.filter((p) => Number.isFinite(p.lat) && Number.isFinite(p.lon));
}

function mapProjection(points, width, height) {
  let minLat = Math.min(...points.map((p) => p.lat));
  let maxLat = Math.max(...points.map((p) => p.lat));
  let minLon = Math.min(...points.map((p) => p.lon));
  let maxLon = Math.max(...points.map((p) => p.lon));
  if (minLat === maxLat) { minLat -= 0.005; maxLat += 0.005; }
  if (minLon === maxLon) { minLon -= 0.005; maxLon += 0.005; }
  const padLat = (maxLat - minLat) * 0.08;
  const padLon = (maxLon - minLon) * 0.08;
  minLat -= padLat; maxLat += padLat; minLon -= padLon; maxLon += padLon;
  return (lat, lon) => ({
    x: ((lon - minLon) / (maxLon - minLon)) * width,
    y: height - ((lat - minLat) / (maxLat - minLat)) * height
  });
}

function setView(view) {
  currentView = {...view};
  mapSvg.setAttribute('viewBox', `${view.x} ${view.y} ${view.width} ${view.height}`);
}

function paddedBounds(points) {
  const xs = points.map((p) => p.x);
  const ys = points.map((p) => p.y);
  let minX = Math.min(...xs), maxX = Math.max(...xs);
  let minY = Math.min(...ys), maxY = Math.max(...ys);
  if (minX === maxX) { minX -= 20; maxX += 20; }
  if (minY === maxY) { minY -= 20; maxY += 20; }
  const padX = Math.max(20, (maxX - minX) * 0.18);
  const padY = Math.max(20, (maxY - minY) * 0.18);
  return {x: minX - padX, y: minY - padY, width: maxX - minX + 2 * padX, height: maxY - minY + 2 * padY};
}

function zoom(factor) {
  const cx = currentView.x + currentView.width / 2;
  const cy = currentView.y + currentView.height / 2;
  const width = Math.min(baseWidth * 4, Math.max(baseWidth / 32, currentView.width * factor));
  const height = width * baseHeight / baseWidth;
  setView({x: cx - width / 2, y: cy - height / 2, width, height});
}

function selectObject(kind, id, name) {
  selected = {kind, id, name};
  for (const node of mapSvg.querySelectorAll('.map-selected')) node.classList.remove('map-selected');
  for (const node of mapSvg.querySelectorAll(`[data-map-kind="${kind}"][data-map-id="${CSS.escape(id)}"]`)) node.classList.add('map-selected');
  mapSelection.textContent = `${kind}: ${name}`;
  document.dispatchEvent(new CustomEvent('caching-tools:map-select', {detail: selected}));
}

function fitSelection() {
  if (!selected) {
    mapSelection.textContent = 'Select a waypoint, route or track first.';
    return;
  }
  const bounds = objectBounds.get(`${selected.kind}:${selected.id}`);
  if (bounds) setView(bounds);
}

async function fetchMapData() {
  const [waypoints, summaries] = await Promise.all([
    requestJSON('/api/waypoints'),
    requestJSON('/api/paths')
  ]);
  const paths = await Promise.all(summaries.map((item) => requestJSON(`/api/paths/${encodeURIComponent(item.id)}`)));
  return {waypoints, paths};
}

async function refreshLocalMap() {
  try {
    const {waypoints, paths} = await fetchMapData();
    const all = collectCoordinates(waypoints, paths);
    mapSvg.replaceChildren();
    objectBounds = new Map();
    selected = null;
    mapSelection.textContent = 'Nothing selected.';
    if (all.length === 0) {
      mapStatus.textContent = 'No saved coordinates to display.';
      return;
    }

    const project = mapProjection(all, baseWidth, baseHeight);
    fullView = {x: 0, y: 0, width: baseWidth, height: baseHeight};
    setView(fullView);

    for (let i = 1; i < 6; i++) {
      mapSvg.append(svgElement('line', {x1: 0, y1: i * baseHeight / 6, x2: baseWidth, y2: i * baseHeight / 6, class: 'map-grid-line'}));
      mapSvg.append(svgElement('line', {x1: i * baseWidth / 6, y1: 0, x2: i * baseWidth / 6, y2: baseHeight, class: 'map-grid-line'}));
    }

    for (const item of paths) {
      const projected = [];
      for (const segment of pathPoints(item)) {
        const valid = segment.filter((p) => Number.isFinite(pointLat(p)) && Number.isFinite(pointLon(p)));
        if (!valid.length) continue;
        const points = valid.map((p) => project(pointLat(p), pointLon(p)));
        projected.push(...points);
        const coords = points.map((xy) => `${xy.x.toFixed(2)},${xy.y.toFixed(2)}`).join(' ');
        const polyline = svgElement('polyline', {points: coords, class: `map-path map-${item.kind} map-interactive`, tabindex: 0, 'data-map-kind': item.kind, 'data-map-id': item.id});
        const title = svgElement('title');
        title.textContent = `${item.kind}: ${item.name}`;
        polyline.append(title);
        const choose = () => selectObject(item.kind, item.id, item.name);
        polyline.addEventListener('click', choose);
        polyline.addEventListener('keydown', (event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); choose(); } });
        mapSvg.append(polyline);
      }
      if (projected.length) objectBounds.set(`${item.kind}:${item.id}`, paddedBounds(projected));
    }

    for (const item of waypoints) {
      const xy = project(item.point.latitude, item.point.longitude);
      const marker = svgElement('circle', {cx: xy.x, cy: xy.y, r: 6, class: 'map-waypoint map-interactive', tabindex: 0, 'data-map-kind': 'waypoint', 'data-map-id': item.id});
      const title = svgElement('title');
      title.textContent = item.name;
      marker.append(title);
      const choose = () => selectObject('waypoint', item.id, item.name);
      marker.addEventListener('click', choose);
      marker.addEventListener('keydown', (event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); choose(); } });
      mapSvg.append(marker);
      objectBounds.set(`waypoint:${item.id}`, paddedBounds([xy]));
    }

    mapStatus.textContent = `${waypoints.length} waypoint${waypoints.length === 1 ? '' : 's'}, ${paths.length} saved track/route object${paths.length === 1 ? '' : 's'}. Drag to pan, use wheel or buttons to zoom.`;
  } catch (error) {
    mapStatus.textContent = `Error: ${error.message}`;
  }
}

mapSvg.addEventListener('wheel', (event) => {
  event.preventDefault();
  zoom(event.deltaY < 0 ? 0.8 : 1.25);
}, {passive: false});

mapSvg.addEventListener('pointerdown', (event) => {
  mapSvg.setPointerCapture(event.pointerId);
  dragStart = {x: event.clientX, y: event.clientY, view: {...currentView}};
});
mapSvg.addEventListener('pointermove', (event) => {
  if (!dragStart) return;
  const rect = mapSvg.getBoundingClientRect();
  const dx = (event.clientX - dragStart.x) * dragStart.view.width / rect.width;
  const dy = (event.clientY - dragStart.y) * dragStart.view.height / rect.height;
  setView({...dragStart.view, x: dragStart.view.x - dx, y: dragStart.view.y - dy});
});
mapSvg.addEventListener('pointerup', () => { dragStart = null; });
mapSvg.addEventListener('pointercancel', () => { dragStart = null; });

window.refreshLocalMap = refreshLocalMap;
document.querySelector('#map-refresh').addEventListener('click', refreshLocalMap);
document.querySelector('#map-zoom-in').addEventListener('click', () => zoom(0.8));
document.querySelector('#map-zoom-out').addEventListener('click', () => zoom(1.25));
document.querySelector('#map-fit-all').addEventListener('click', () => setView(fullView));
document.querySelector('#map-fit-selection').addEventListener('click', fitSelection);
refreshLocalMap();
