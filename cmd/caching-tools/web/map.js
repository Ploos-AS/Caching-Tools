const mapSvg = document.querySelector('#local-map');
const mapStatus = document.querySelector('#map-status');
const mapNS = 'http://www.w3.org/2000/svg';

function svgElement(name, attrs = {}) {
  const el = document.createElementNS(mapNS, name);
  for (const [key, value] of Object.entries(attrs)) el.setAttribute(key, String(value));
  return el;
}

function pathPoints(item) {
  if (item.kind === 'route') return [item.route || []];
  return (item.segments || []).map((segment) => segment.points || []);
}

function collectCoordinates(waypoints, paths) {
  const points = waypoints.map((w) => ({lat: w.point.latitude, lon: w.point.longitude}));
  for (const item of paths) {
    for (const segment of pathPoints(item)) {
      for (const point of segment) points.push({lat: point.latitude, lon: point.longitude});
    }
  }
  return points;
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
    if (all.length === 0) {
      mapStatus.textContent = 'No saved coordinates to display.';
      return;
    }

    const width = 900;
    const height = 520;
    const project = mapProjection(all, width, height);
    mapSvg.setAttribute('viewBox', `0 0 ${width} ${height}`);

    for (let i = 1; i < 6; i++) {
      mapSvg.append(svgElement('line', {x1: 0, y1: i * height / 6, x2: width, y2: i * height / 6, class: 'map-grid-line'}));
      mapSvg.append(svgElement('line', {x1: i * width / 6, y1: 0, x2: i * width / 6, y2: height, class: 'map-grid-line'}));
    }

    for (const item of paths) {
      for (const segment of pathPoints(item)) {
        if (!segment.length) continue;
        const coords = segment.map((p) => {
          const xy = project(p.latitude, p.longitude);
          return `${xy.x.toFixed(2)},${xy.y.toFixed(2)}`;
        }).join(' ');
        const polyline = svgElement('polyline', {points: coords, class: `map-path map-${item.kind}`});
        const title = svgElement('title');
        title.textContent = `${item.kind}: ${item.name}`;
        polyline.append(title);
        mapSvg.append(polyline);
      }
    }

    for (const item of waypoints) {
      const xy = project(item.point.latitude, item.point.longitude);
      const marker = svgElement('circle', {cx: xy.x, cy: xy.y, r: 6, class: 'map-waypoint'});
      const title = svgElement('title');
      title.textContent = item.name;
      marker.append(title);
      mapSvg.append(marker);
    }

    mapStatus.textContent = `${waypoints.length} waypoint${waypoints.length === 1 ? '' : 's'}, ${paths.length} saved track/route object${paths.length === 1 ? '' : 's'}. Local schematic view; no external map service used.`;
  } catch (error) {
    mapStatus.textContent = `Error: ${error.message}`;
  }
}

window.refreshLocalMap = refreshLocalMap;
document.querySelector('#map-refresh').addEventListener('click', refreshLocalMap);
refreshLocalMap();
