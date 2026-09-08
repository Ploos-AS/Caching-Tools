const qualityControls = document.createElement('div');
qualityControls.id = 'field-quality-controls';
qualityControls.className = 'form-grid nav-grid';
qualityControls.innerHTML = `
  <label>Minimum breadcrumb distance (m)<input id="field-min-distance" type="number" min="0" step="1" value="3"></label>
  <label>Maximum GPS accuracy (m)<input id="field-max-accuracy" type="number" min="1" step="1" value="50"></label>
  <label><input id="field-export-simplify" type="checkbox"> Simplify GPX export</label>
  <label>Export simplification tolerance (m)<input id="field-simplify-tolerance" type="number" min="0" step="1" value="5"></label>
  <p id="field-quality-status" aria-live="polite">Quality filter ready.</p>`;
const breadcrumbHeading = fieldSection.querySelector('h3:nth-of-type(2)');
if (breadcrumbHeading) breadcrumbHeading.insertAdjacentElement('afterend', qualityControls);
else fieldSection.append(qualityControls);

const fieldMinDistance = qualityControls.querySelector('#field-min-distance');
const fieldMaxAccuracy = qualityControls.querySelector('#field-max-accuracy');
const fieldExportSimplify = qualityControls.querySelector('#field-export-simplify');
const fieldSimplifyTolerance = qualityControls.querySelector('#field-simplify-tolerance');
const fieldQualityStatus = qualityControls.querySelector('#field-quality-status');
let qualityAccepted = 0;
let qualityRejectedAccuracy = 0;
let qualityRejectedDistance = 0;

function qualityNumber(input, fallback, allowZero = false) {
  const value = Number(input.value);
  if (!Number.isFinite(value) || value < 0 || (!allowZero && value === 0)) return fallback;
  return value;
}

function renderQualityStatus() {
  fieldQualityStatus.textContent = `Accepted ${qualityAccepted} · rejected accuracy ${qualityRejectedAccuracy} · rejected distance ${qualityRejectedDistance}`;
}

function resetQualityCounters() {
  qualityAccepted = 0;
  qualityRejectedAccuracy = 0;
  qualityRejectedDistance = 0;
  renderQualityStatus();
}

const addBreadcrumbUnfiltered = addBreadcrumb;
addBreadcrumb = function addBreadcrumbFiltered(position) {
  const accuracy = Number(position.coords.accuracy);
  const maxAccuracy = qualityNumber(fieldMaxAccuracy, 50);
  if (!Number.isFinite(accuracy) || accuracy > maxAccuracy) {
    qualityRejectedAccuracy += 1;
    renderQualityStatus();
    return false;
  }

  if (liveBreadcrumbs.length) {
    const candidate = {
      latitude: position.coords.latitude,
      longitude: position.coords.longitude
    };
    const minimumDistance = qualityNumber(fieldMinDistance, 3, true);
    const distance = breadcrumbDistanceMeters(liveBreadcrumbs[liveBreadcrumbs.length - 1], candidate);
    if (distance < minimumDistance) {
      qualityRejectedDistance += 1;
      renderQualityStatus();
      return false;
    }
  }

  addBreadcrumbUnfiltered(position);
  qualityAccepted += 1;
  renderQualityStatus();
  return true;
};

function pointSegmentDistanceMeters(point, start, end) {
  const refLat = point.latitude * Math.PI / 180;
  const cosLat = Math.max(1e-12, Math.abs(Math.cos(refLat)));
  const radius = 6371008.8;
  const toXY = item => ({
    x: (item.longitude - point.longitude) * Math.PI / 180 * cosLat * radius,
    y: (item.latitude - point.latitude) * Math.PI / 180 * radius
  });
  const a = toXY(start);
  const b = toXY(end);
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const denominator = dx * dx + dy * dy;
  let fraction = denominator > 0 ? -(a.x * dx + a.y * dy) / denominator : 0;
  fraction = Math.max(0, Math.min(1, fraction));
  const x = a.x + fraction * dx;
  const y = a.y + fraction * dy;
  return Math.hypot(x, y);
}

function simplifyBreadcrumbs(points, toleranceM) {
  if (points.length <= 2 || toleranceM <= 0) return points.slice();
  let maxDistance = -1;
  let index = -1;
  for (let current = 1; current < points.length - 1; current += 1) {
    const distance = pointSegmentDistanceMeters(points[current], points[0], points[points.length - 1]);
    if (distance > maxDistance) {
      maxDistance = distance;
      index = current;
    }
  }
  if (maxDistance <= toleranceM || index < 0) return [points[0], points[points.length - 1]];
  const left = simplifyBreadcrumbs(points.slice(0, index + 1), toleranceM);
  const right = simplifyBreadcrumbs(points.slice(index), toleranceM);
  return left.slice(0, -1).concat(right);
}

const breadcrumbGPXUnfiltered = breadcrumbGPX;
breadcrumbGPX = function breadcrumbGPXWithQuality() {
  if (!fieldExportSimplify.checked) return breadcrumbGPXUnfiltered();
  if (!liveBreadcrumbs.length) throw new Error('No breadcrumb points to export.');
  const tolerance = qualityNumber(fieldSimplifyTolerance, 5, true);
  const exportPoints = simplifyBreadcrumbs(liveBreadcrumbs, tolerance);
  const target = fieldForm.elements['target-id'].selectedOptions[0]?.textContent || 'Live navigation session';
  const points = exportPoints.map(item =>
    `      <trkpt lat="${item.latitude.toFixed(7)}" lon="${item.longitude.toFixed(7)}"><time>${escapeXML(item.time)}</time></trkpt>`
  ).join('\n');
  fieldLiveStatus.textContent = `GPX simplification: ${liveBreadcrumbs.length} → ${exportPoints.length} points at ${tolerance.toFixed(1)} m tolerance.`;
  return `<?xml version="1.0" encoding="UTF-8"?>\n<gpx version="1.1" creator="Caching Tools" xmlns="http://www.topografix.com/GPX/1/1">\n  <trk>\n    <name>${escapeXML(`Caching Tools session - ${target}`)}</name>\n    <trkseg>\n${points}\n    </trkseg>\n  </trk>\n</gpx>\n`;
};

const startLiveNavigationUnfiltered = startLiveNavigation;
startLiveNavigation = function startLiveNavigationWithQualityReset() {
  resetQualityCounters();
  startLiveNavigationUnfiltered();
};

const clearBreadcrumbButton = fieldSection.querySelector('#field-breadcrumb-clear');
clearBreadcrumbButton.addEventListener('click', resetQualityCounters);

resetQualityCounters();
const qualityFooter = document.querySelector('footer');
if (qualityFooter) qualityFooter.textContent = 'Caching Tools M1.28';
