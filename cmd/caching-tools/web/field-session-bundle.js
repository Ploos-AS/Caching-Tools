const fieldSessionControls = document.querySelector('#field-session-controls');
const fieldSessionBundleControls = document.createElement('div');
fieldSessionBundleControls.id = 'field-session-bundle';
fieldSessionBundleControls.className = 'map-controls';
fieldSessionBundleControls.innerHTML = `
  <button type="button" id="field-session-bundle-export">Export session JSON</button>
  <label>Import session JSON<input type="file" id="field-session-bundle-import" accept="application/json,.json"></label>
  <p id="field-session-bundle-status" aria-live="polite">Portable field-session bundle ready.</p>`;
if (fieldSessionControls) fieldSessionControls.append(fieldSessionBundleControls);

const fieldSessionBundleExport = fieldSessionBundleControls.querySelector('#field-session-bundle-export');
const fieldSessionBundleImport = fieldSessionBundleControls.querySelector('#field-session-bundle-import');
const fieldSessionBundleStatus = fieldSessionBundleControls.querySelector('#field-session-bundle-status');
const fieldSessionBundleKind = 'caching-tools.field-session';
const fieldSessionBundleVersion = 1;
const fieldSessionBundleMaxBytes = 4 * 1024 * 1024;

function portableFieldSessionBundle() {
  return {
    kind: fieldSessionBundleKind,
    version: fieldSessionBundleVersion,
    exportedAt: new Date().toISOString(),
    session: {
      breadcrumbs: liveBreadcrumbs.map(item => ({...item})),
      markers: liveMarkers.map(item => ({...item})),
      paused: Boolean(breadcrumbRecordingPaused),
      segmentID: Math.max(0, Number.isInteger(recordingSegmentID) ? recordingSegmentID : 0)
    },
    metadata: {
      target: fieldForm.elements['target-id']?.selectedOptions?.[0]?.textContent || '',
      generator: 'Caching Tools M1.53'
    }
  };
}

function normalizePortableFieldSessionBundle(value) {
  if (!value || value.kind !== fieldSessionBundleKind || value.version !== fieldSessionBundleVersion || !value.session) return null;
  const session = value.session;
  if (!Array.isArray(session.breadcrumbs) || !Array.isArray(session.markers)) return null;
  if (session.breadcrumbs.length > 10000 || session.markers.length > 500) return null;
  if (!session.breadcrumbs.every(validRecoveredPoint) || !session.markers.every(validRecoveredPoint)) return null;
  return {
    breadcrumbs: session.breadcrumbs.map(item => ({...item})),
    markers: session.markers.map(item => ({...item})),
    paused: Boolean(session.paused),
    segmentID: Math.max(0, Number.isInteger(session.segmentID) ? session.segmentID : 0)
  };
}

function importPortableFieldSessionBundle(value) {
  const session = normalizePortableFieldSessionBundle(value);
  if (!session) throw new Error('Invalid or unsupported field-session bundle.');
  liveBreadcrumbs.splice(0, liveBreadcrumbs.length, ...session.breadcrumbs);
  liveMarkers = session.markers;
  breadcrumbRecordingPaused = session.paused;
  recordingSegmentID = session.segmentID;
  startNewRecordingSegment = false;
  renderRecordingStatus();
  renderMarkers();
  if (typeof renderSessionStatistics === 'function') renderSessionStatistics();
  persistFieldSession();
  if (typeof window.renderMapOverlays === 'function') window.renderMapOverlays();
  fieldRecordingStatus.textContent = `Imported ${liveBreadcrumbs.length} breadcrumbs and ${liveMarkers.length} markers. Live GPS remains stopped until explicitly started.`;
  fieldSessionBundleStatus.textContent = 'Field-session bundle imported successfully.';
  return session;
}

function downloadPortableFieldSessionBundle() {
  const bundle = portableFieldSessionBundle();
  const text = JSON.stringify(bundle, null, 2);
  if (text.length > fieldSessionBundleMaxBytes) {
    fieldSessionBundleStatus.textContent = 'Bundle export blocked because it exceeded 4 MiB.';
    return false;
  }
  const blob = new Blob([text], {type:'application/json'});
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `caching-tools-field-session-${new Date().toISOString().replaceAll(':','-')}.json`;
  link.click();
  URL.revokeObjectURL(url);
  fieldSessionBundleStatus.textContent = `Exported ${bundle.session.breadcrumbs.length} breadcrumbs and ${bundle.session.markers.length} markers.`;
  return true;
}

async function importPortableFieldSessionFile(file) {
  if (!file) return false;
  if (Number(file.size) > fieldSessionBundleMaxBytes) throw new Error('Field-session bundle exceeds 4 MiB.');
  const text = await file.text();
  if (text.length > fieldSessionBundleMaxBytes) throw new Error('Field-session bundle exceeds 4 MiB.');
  return importPortableFieldSessionBundle(JSON.parse(text));
}

fieldSessionBundleExport?.addEventListener('click', downloadPortableFieldSessionBundle);
fieldSessionBundleImport?.addEventListener('change', async () => {
  try {
    await importPortableFieldSessionFile(fieldSessionBundleImport.files?.[0]);
  } catch (error) {
    fieldSessionBundleStatus.textContent = `Session import error: ${error.message}`;
  } finally {
    fieldSessionBundleImport.value = '';
  }
});

window.portableFieldSessionBundle = portableFieldSessionBundle;
window.importPortableFieldSessionBundle = importPortableFieldSessionBundle;
const fieldSessionBundleFooter = document.querySelector('footer');
if (fieldSessionBundleFooter) fieldSessionBundleFooter.textContent = 'Caching Tools M1.53';
