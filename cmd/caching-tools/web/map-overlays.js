const overlayControls = document.createElement('div');
overlayControls.className = 'map-controls';
overlayControls.id = 'map-overlay-controls';
overlayControls.innerHTML = `
  <label><input type="checkbox" id="map-overlay-labels" checked> Waypoint labels</label>
  <label><input type="checkbox" id="map-overlay-rings"> Waypoint radius</label>
  <label>Radius (m)<input type="number" id="map-overlay-radius" min="1" step="1" value="100"></label>
  <label><input type="checkbox" id="map-overlay-guidance" checked> Position / target line</label>
  <label><input type="checkbox" id="map-overlay-path-guidance" checked> Route / track guidance</label>
  <label><input type="checkbox" id="map-overlay-live-session" checked> Live breadcrumbs / markers</label>
  <button type="button" id="map-overlay-refresh">Refresh overlays</button>`;
const mapFrame = document.querySelector('#local-map')?.closest('.map-frame');
if (mapFrame) mapFrame.insertAdjacentElement('beforebegin', overlayControls);

let overlayWaypoints = [];
let overlaySelection = null;
let overlayNavigationResult = null;

function overlayLayer() {
  const svg = window.cachingToolsMap?.svg;
  if (!svg) return null;
  let layer = svg.querySelector('#map-overlay-layer');
  if (!layer) {
    layer = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    layer.id = 'map-overlay-layer';
    layer.setAttribute('pointer-events', 'none');
    svg.append(layer);
  }
  return layer;
}

function overlaySVG(name, attrs = {}) {
  const node = document.createElementNS('http://www.w3.org/2000/svg', name);
  for (const [key, value] of Object.entries(attrs)) node.setAttribute(key, String(value));
  return node;
}

function metersPerDegreeLatitude() { return 111320; }
function metersPerDegreeLongitude(latitude) { return Math.max(1, 111320 * Math.cos(latitude * Math.PI / 180)); }

function projectedRadiusMeters(project, latitude, longitude, metres) {
  const center = project(latitude, longitude);
  const east = project(latitude, longitude + metres / metersPerDegreeLongitude(latitude));
  const north = project(latitude + metres / metersPerDegreeLatitude(), longitude);
  return {
    rx: Math.max(1, Math.abs(east.x - center.x)),
    ry: Math.max(1, Math.abs(north.y - center.y))
  };
}

function fieldCurrentPosition() {
  const form = document.querySelector('#field-navigation-form');
  if (!form) return null;
  const latitude = Number(form.elements.latitude?.value);
  const longitude = Number(form.elements.longitude?.value);
  return Number.isFinite(latitude) && Number.isFinite(longitude) ? {latitude, longitude} : null;
}

function selectedWaypoint() {
  if (!overlaySelection || overlaySelection.kind !== 'waypoint') return null;
  return overlayWaypoints.find(item => item.id === overlaySelection.id) || null;
}

function responsePoint(point) {
  if (!point) return null;
  const latitude = Number(point.latitude ?? point.Latitude ?? point.lat?.decimal);
  const longitude = Number(point.longitude ?? point.Longitude ?? point.lon?.decimal);
  return Number.isFinite(latitude) && Number.isFinite(longitude) ? {latitude, longitude} : null;
}

function appendPathGuidance(layer, project, result) {
  if (!result || result.kind === 'waypoint' || !result.progress) return;
  const from = responsePoint(result.from);
  const nearest = responsePoint(result.target);
  const next = responsePoint(result.progress.next_point);
  if (!from || !nearest || !next) return;

  const here = project(from.latitude, from.longitude);
  const onPath = project(nearest.latitude, nearest.longitude);
  const nextPoint = project(next.latitude, next.longitude);
  const status = String(result.guidance?.status || 'on-route');

  layer.append(overlaySVG('line', {
    x1: here.x, y1: here.y, x2: onPath.x, y2: onPath.y,
    class: `map-overlay-cross-track map-overlay-status-${status}`
  }));
  layer.append(overlaySVG('circle', {cx:onPath.x, cy:onPath.y, r:7, class:'map-overlay-nearest'}));
  layer.append(overlaySVG('line', {
    x1:onPath.x, y1:onPath.y, x2:nextPoint.x, y2:nextPoint.y,
    class:'map-overlay-forward'
  }));
  layer.append(overlaySVG('circle', {cx:nextPoint.x, cy:nextPoint.y, r:6, class:'map-overlay-next'}));

  const nextRing = projectedRadiusMeters(project, next.latitude, next.longitude, Number(result.guidance?.arrival_radius_m) || 20);
  layer.append(overlaySVG('ellipse', {
    cx:nextPoint.x, cy:nextPoint.y, rx:nextRing.rx, ry:nextRing.ry,
    class:'map-overlay-next-arrival'
  }));

  const label = overlaySVG('text', {x:onPath.x + 10, y:onPath.y + 18, class:`map-overlay-status-label map-overlay-status-${status}`});
  const crossTrack = Number(result.cross_track_m);
  const remaining = Number(result.progress.remaining_m);
  const bearing = Number(result.progress.forward_bearing_deg);
  label.textContent = `${status} · ${Number.isFinite(crossTrack) ? crossTrack.toFixed(0) : '?'} m off · ${Number.isFinite(remaining) ? (remaining / 1000).toFixed(2) : '?'} km left · ${Number.isFinite(bearing) ? bearing.toFixed(0) : '?'}°`;
  layer.append(label);
}

function liveSessionSnapshot() {
  if (typeof liveBreadcrumbs === 'undefined') return {breadcrumbs:[], markers:[]};
  const breadcrumbs = Array.isArray(liveBreadcrumbs) ? liveBreadcrumbs : [];
  const markers = typeof liveMarkers !== 'undefined' && Array.isArray(liveMarkers) ? liveMarkers : [];
  return {breadcrumbs, markers};
}

function appendLiveSession(layer, project) {
  const {breadcrumbs, markers} = liveSessionSnapshot();
  const segments = [];
  for (const item of breadcrumbs) {
    const point = responsePoint(item);
    if (!point) continue;
    const segmentID = item.recordingSegment ?? 0;
    let segment = segments[segments.length - 1];
    if (!segment || segment.id !== segmentID) {
      segment = {id:segmentID, points:[]};
      segments.push(segment);
    }
    segment.points.push(point);
  }
  for (const segment of segments) {
    if (segment.points.length < 2) continue;
    const projected = segment.points.map(point => project(point.latitude, point.longitude));
    layer.append(overlaySVG('polyline', {
      points:projected.map(point => `${point.x},${point.y}`).join(' '),
      class:'map-overlay-live-breadcrumb',
      'data-recording-segment':segment.id
    }));
  }
  for (let index = 1; index < segments.length; index += 1) {
    const previous = segments[index - 1].points.at(-1);
    const current = segments[index].points[0];
    if (!previous || !current) continue;
    const before = project(previous.latitude, previous.longitude);
    const after = project(current.latitude, current.longitude);
    layer.append(overlaySVG('line', {
      x1:before.x, y1:before.y, x2:after.x, y2:after.y,
      class:'map-overlay-live-segment-gap'
    }));
  }
  for (const [index, marker] of markers.entries()) {
    const point = responsePoint(marker);
    if (!point) continue;
    const projected = project(point.latitude, point.longitude);
    layer.append(overlaySVG('circle', {
      cx:projected.x, cy:projected.y, r:6,
      class:`map-overlay-live-marker${marker.promotedWaypointID ? ' map-overlay-live-marker-promoted' : ''}`
    }));
    const label = overlaySVG('text', {x:projected.x + 9, y:projected.y + 15, class:'map-overlay-live-marker-label'});
    label.textContent = `${index + 1}: ${marker.type || 'marker'}`;
    layer.append(label);
  }
}

function renderMapOverlays() {
  const project = window.cachingToolsMap?.project;
  const layer = overlayLayer();
  if (!project || !layer) return;
  layer.replaceChildren();

  if (document.querySelector('#map-overlay-rings')?.checked) {
    const radius = Math.max(1, Number(document.querySelector('#map-overlay-radius')?.value) || 100);
    for (const item of overlayWaypoints) {
      const center = project(item.point.latitude, item.point.longitude);
      const size = projectedRadiusMeters(project, item.point.latitude, item.point.longitude, radius);
      layer.append(overlaySVG('ellipse', {cx:center.x, cy:center.y, rx:size.rx, ry:size.ry, class:'map-overlay-radius'}));
    }
  }

  if (document.querySelector('#map-overlay-labels')?.checked) {
    for (const item of overlayWaypoints) {
      const point = project(item.point.latitude, item.point.longitude);
      const text = overlaySVG('text', {x:point.x + 9, y:point.y - 9, class:'map-overlay-label'});
      text.textContent = item.name;
      layer.append(text);
    }
  }

  if (document.querySelector('#map-overlay-guidance')?.checked) {
    const position = fieldCurrentPosition();
    const target = selectedWaypoint();
    if (position) {
      const here = project(position.latitude, position.longitude);
      layer.append(overlaySVG('circle', {cx:here.x, cy:here.y, r:8, class:'map-overlay-position'}));
      if (target) {
        const there = project(target.point.latitude, target.point.longitude);
        layer.append(overlaySVG('line', {x1:here.x, y1:here.y, x2:there.x, y2:there.y, class:'map-overlay-guidance'}));
        const targetRing = projectedRadiusMeters(project, target.point.latitude, target.point.longitude, Math.max(1, Number(document.querySelector('#field-navigation-form')?.elements['arrival-radius']?.value) || 20));
        layer.append(overlaySVG('ellipse', {cx:there.x, cy:there.y, rx:targetRing.rx, ry:targetRing.ry, class:'map-overlay-arrival'}));
      }
    }
  }

  if (document.querySelector('#map-overlay-path-guidance')?.checked) appendPathGuidance(layer, project, overlayNavigationResult);
  if (document.querySelector('#map-overlay-live-session')?.checked) appendLiveSession(layer, project);
}

async function refreshMapOverlays() {
  try {
    overlayWaypoints = await requestJSON('/api/waypoints');
    renderMapOverlays();
  } catch (error) {
    const status = document.querySelector('#map-status');
    if (status) status.textContent = `Overlay error: ${error.message}`;
  }
}

if (typeof navigateField === 'function') {
  const navigateFieldWithoutOverlay = navigateField;
  navigateField = async function cachingToolsNavigateFieldWithOverlay(latitude, longitude) {
    const result = await navigateFieldWithoutOverlay(latitude, longitude);
    overlayNavigationResult = result;
    renderMapOverlays();
    return result;
  };
}

if (typeof addBreadcrumb === 'function') {
  const addBreadcrumbWithoutMapOverlay = addBreadcrumb;
  addBreadcrumb = function cachingToolsAddBreadcrumbWithMapOverlay(position) {
    const accepted = addBreadcrumbWithoutMapOverlay(position);
    if (accepted !== false) renderMapOverlays();
    return accepted;
  };
}
if (typeof renderMarkers === 'function') {
  const renderMarkersWithoutMapOverlay = renderMarkers;
  renderMarkers = function cachingToolsRenderMarkersWithMapOverlay() {
    const result = renderMarkersWithoutMapOverlay();
    renderMapOverlays();
    return result;
  };
}

document.addEventListener('caching-tools:map-rendered', () => { void refreshMapOverlays(); });
document.addEventListener('caching-tools:map-select', event => {
  overlaySelection = event.detail || null;
  if (!overlaySelection || (overlayNavigationResult && overlayNavigationResult.id !== overlaySelection.id)) overlayNavigationResult = null;
  renderMapOverlays();
});
for (const id of ['map-overlay-labels','map-overlay-rings','map-overlay-radius','map-overlay-guidance','map-overlay-path-guidance','map-overlay-live-session']) {
  document.querySelector(`#${id}`)?.addEventListener('change', renderMapOverlays);
}
document.querySelector('#map-overlay-refresh')?.addEventListener('click', () => void refreshMapOverlays());
document.querySelector('#field-navigation-form')?.addEventListener('input', renderMapOverlays);

window.refreshMapOverlays = refreshMapOverlays;
window.renderMapOverlays = renderMapOverlays;
void refreshMapOverlays();
