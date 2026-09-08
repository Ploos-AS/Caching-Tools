const logbookDashboard = document.createElement('section');
logbookDashboard.className = 'tool';
logbookDashboard.id = 'field-note-dashboard';
logbookDashboard.innerHTML = `
  <h2>Logbook dashboard</h2>
  <p>Local summary of persistent field-note activity. Calculated in the browser from local APIs only; no telemetry is sent anywhere.</p>
  <button type="button" id="field-note-dashboard-refresh">Refresh dashboard</button>
  <pre id="field-note-dashboard-summary" class="result">Loading logbook summary...</pre>`;

const fieldNotesTool = document.querySelector('#field-notes');
if (fieldNotesTool) fieldNotesTool.insertAdjacentElement('afterend', logbookDashboard);
else document.querySelector('main')?.append(logbookDashboard);

const dashboardSummary = logbookDashboard.querySelector('#field-note-dashboard-summary');
let dashboardRefreshPending = false;

function dashboardCountBy(items, key) {
  const counts = new Map();
  for (const item of items) {
    const value = String(item[key] || '').trim() || '(unset)';
    counts.set(value, (counts.get(value) || 0) + 1);
  }
  return [...counts.entries()].sort((a,b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

function dashboardTopLinks(notes, key, labels, limit = 5) {
  const counts = new Map();
  for (const note of notes) {
    const id = String(note[key] || '').trim();
    if (!id) continue;
    counts.set(id, (counts.get(id) || 0) + 1);
  }
  return [...counts.entries()]
    .sort((a,b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, limit)
    .map(([id,count]) => `${labels.get(id) || id}: ${count}`);
}

function dashboardActivityMonths(notes, count = 6) {
  const now = new Date();
  const months = [];
  for (let offset = count - 1; offset >= 0; offset--) {
    const date = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - offset, 1));
    months.push(`${date.getUTCFullYear()}-${String(date.getUTCMonth()+1).padStart(2,'0')}`);
  }
  const totals = new Map(months.map(month => [month,0]));
  for (const note of notes) {
    const date = new Date(note.occurred_at);
    if (Number.isNaN(date.getTime())) continue;
    const month = `${date.getUTCFullYear()}-${String(date.getUTCMonth()+1).padStart(2,'0')}`;
    if (totals.has(month)) totals.set(month, totals.get(month) + 1);
  }
  return months.map(month => `${month}: ${totals.get(month)}`);
}

function dashboardRecentCount(notes, days) {
  const cutoff = Date.now() - days * 86400000;
  return notes.filter(note => {
    const timestamp = Date.parse(note.occurred_at);
    return Number.isFinite(timestamp) && timestamp >= cutoff;
  }).length;
}

function dashboardLabels(waypoints, workspaces) {
  const waypointLabels = new Map(waypoints.map(item => [item.id, item.name || item.id]));
  const workspaceLabels = new Map(workspaces.map(item => [item.id, `${item.code ? item.code + ' · ' : ''}${item.title || item.id}`]));
  return {waypointLabels, workspaceLabels};
}

function renderLogbookDashboard(notes, waypoints, workspaces) {
  const {waypointLabels, workspaceLabels} = dashboardLabels(waypoints, workspaces);
  const statuses = dashboardCountBy(notes, 'status').map(([value,count]) => `${value}: ${count}`);
  const types = dashboardCountBy(notes, 'type').map(([value,count]) => `${value}: ${count}`);
  const recentWaypoints = dashboardTopLinks(notes, 'waypoint_id', waypointLabels);
  const recentWorkspaces = dashboardTopLinks(notes, 'workspace_id', workspaceLabels);
  const validDates = notes.map(note => Date.parse(note.occurred_at)).filter(Number.isFinite).sort((a,b) => b-a);
  const latest = validDates.length ? new Date(validDates[0]).toISOString() : 'none';
  dashboardSummary.textContent = [
    `Total field notes: ${notes.length}`,
    `Last 7 days: ${dashboardRecentCount(notes, 7)}`,
    `Last 30 days: ${dashboardRecentCount(notes, 30)}`,
    `Latest activity: ${latest}`,
    '',
    'Status distribution',
    ...(statuses.length ? statuses : ['none']),
    '',
    'Type distribution',
    ...(types.length ? types : ['none']),
    '',
    'Activity by month (UTC, last 6 months)',
    ...dashboardActivityMonths(notes),
    '',
    'Most-used linked waypoints',
    ...(recentWaypoints.length ? recentWaypoints : ['none']),
    '',
    'Most-used linked mystery workspaces',
    ...(recentWorkspaces.length ? recentWorkspaces : ['none'])
  ].join('\n');
}

async function refreshLogbookDashboard() {
  if (dashboardRefreshPending) return;
  dashboardRefreshPending = true;
  try {
    const [notes, waypoints, workspaces] = await Promise.all([
      requestJSON('/api/field-notes'),
      requestJSON('/api/waypoints'),
      requestJSON('/api/mystery-workspaces')
    ]);
    renderLogbookDashboard(notes, waypoints, workspaces);
  } catch (error) {
    dashboardSummary.textContent = `Dashboard error: ${error.message}`;
  } finally {
    dashboardRefreshPending = false;
  }
}

logbookDashboard.querySelector('#field-note-dashboard-refresh').addEventListener('click', refreshLogbookDashboard);
const observedFieldNoteList = document.querySelector('#field-note-list');
if (observedFieldNoteList) {
  new MutationObserver(() => { void refreshLogbookDashboard(); }).observe(observedFieldNoteList, {childList:true, subtree:true, characterData:true});
}
void refreshLogbookDashboard();
window.refreshLogbookDashboard = refreshLogbookDashboard;
const dashboardFooter = document.querySelector('footer'); if (dashboardFooter) dashboardFooter.textContent = 'Caching Tools M1.34';
