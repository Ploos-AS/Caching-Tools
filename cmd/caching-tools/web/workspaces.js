const workspaceForm = document.querySelector('#mystery-workspace-form');
const workspaceList = document.querySelector('#mystery-workspace-list');
const workspaceStatus = document.querySelector('#mystery-workspace-status');
let editingWorkspaceId = '';
let activeWorkspace = null;
let latestPuzzleResult = '';

function parseWorkspaceVariables(text) {
  const out = {};
  for (const part of text.split(/[;,]+/)) {
    if (!part.trim()) continue;
    const [rawKey, rawValue] = part.split('=');
    const key = (rawKey || '').trim().toUpperCase();
    const value = Number((rawValue || '').trim());
    if (!/^[A-Z]$/.test(key) || !Number.isFinite(value)) throw new Error(`Invalid variable: ${part.trim()}`);
    out[key] = value;
  }
  return out;
}

function variablesText(vars = {}) {
  return Object.keys(vars).sort().map(k => `${k}=${vars[k]}`).join(', ');
}

async function workspaceRequest(url, options = {}) {
  const response = await fetch(url, options);
  if (response.status === 204) return null;
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
  return data;
}

function workspacePayload(form) {
  const f = new FormData(form);
  return {
    code: f.get('code'), title: f.get('title'), notes: f.get('notes'),
    variables: parseWorkspaceVariables(f.get('variables')),
    intermediate: String(f.get('intermediate') || '').split('\n').filter(Boolean),
    latitude_formula: f.get('latitude-formula'), longitude_formula: f.get('longitude-formula'),
    final_waypoint_id: f.get('final-waypoint-id')
  };
}

function resetWorkspaceForm() {
  editingWorkspaceId = '';
  workspaceForm.reset();
  workspaceForm.querySelector('[name="variables"]').value = 'A=1, B=2';
  workspaceForm.querySelector('[name="latitude-formula"]').value = 'N 59 54.[A+B]45';
  workspaceForm.querySelector('[name="longitude-formula"]').value = 'E 010 45.621';
  workspaceForm.querySelector('#mystery-workspace-save').textContent = 'Create workspace';
  workspaceForm.querySelector('#mystery-workspace-cancel').hidden = true;
}

function editWorkspace(x) {
  editingWorkspaceId = x.id;
  const set = (name, value) => { workspaceForm.querySelector(`[name="${name}"]`).value = value || ''; };
  set('code', x.code); set('title', x.title); set('notes', x.notes); set('variables', variablesText(x.variables));
  set('intermediate', (x.intermediate || []).join('\n')); set('latitude-formula', x.latitude_formula); set('longitude-formula', x.longitude_formula); set('final-waypoint-id', x.final_waypoint_id);
  workspaceForm.querySelector('#mystery-workspace-save').textContent = 'Update workspace';
  workspaceForm.querySelector('#mystery-workspace-cancel').hidden = false;
  workspaceForm.scrollIntoView({behavior:'smooth', block:'start'});
}

function activateWorkspace(x) {
  activeWorkspace = x;
  workspaceStatus.textContent = `Active workspace: ${x.code ? x.code + ' — ' : ''}${x.title}`;
  if (window.loadFinalWorkspace) window.loadFinalWorkspace(x);
  loadMysteryWorkspaces().catch(()=>{});
}

async function persistActiveWorkspace(patch = {}) {
  if (!activeWorkspace) throw new Error('Select a workspace first');
  const payload = {
    code: activeWorkspace.code || '', title: activeWorkspace.title, notes: activeWorkspace.notes || '',
    variables: activeWorkspace.variables || {}, intermediate: activeWorkspace.intermediate || [],
    latitude_formula: activeWorkspace.latitude_formula || '', longitude_formula: activeWorkspace.longitude_formula || '',
    final_waypoint_id: activeWorkspace.final_waypoint_id || '', ...patch
  };
  activeWorkspace = await workspaceRequest(`/api/mystery-workspaces/${encodeURIComponent(activeWorkspace.id)}`, {
    method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)
  });
  await loadMysteryWorkspaces();
  return activeWorkspace;
}

async function addPuzzleIntermediate(text = latestPuzzleResult) {
  const value = String(text || '').trim();
  if (!value) throw new Error('No puzzle result available');
  const next = [...(activeWorkspace?.intermediate || []), value];
  await persistActiveWorkspace({intermediate: next});
  workspaceStatus.textContent = 'Puzzle result added as intermediate result.';
}

async function setPuzzleVariable(letter, valueText = latestPuzzleResult) {
  const key = String(letter || '').trim().toUpperCase();
  if (!/^[A-Z]$/.test(key)) throw new Error('Variable must be A-Z');
  const match = String(valueText || '').match(/[+-]?(?:\d+(?:\.\d*)?|\.\d+)/);
  if (!match) throw new Error('Puzzle result contains no numeric value');
  const value = Number(match[0]);
  const variables = {...(activeWorkspace?.variables || {}), [key]: value};
  await persistActiveWorkspace({variables});
  workspaceStatus.textContent = `Saved ${key}=${value} to active workspace.`;
  if (window.loadFinalWorkspace) window.loadFinalWorkspace(activeWorkspace);
}

async function linkFinalWaypointToActiveWorkspace(waypointId) {
  if (!activeWorkspace) return null;
  const saved = await persistActiveWorkspace({final_waypoint_id: waypointId});
  workspaceStatus.textContent = `Final waypoint ${waypointId} linked to ${saved.code || saved.title}.`;
  return saved;
}

function integrationControls() {
  let box = document.querySelector('#mystery-workspace-integration');
  if (box) return box;
  box = document.createElement('div');
  box.id = 'mystery-workspace-integration';
  box.className = 'result';
  box.innerHTML = '<strong>Active workspace integration</strong><p id="mystery-workspace-puzzle-result">Latest puzzle result: —</p><label>Variable <input id="mystery-workspace-variable" maxlength="1" value="A"></label> <button type="button" id="mystery-workspace-add-intermediate">Add puzzle result</button> <button type="button" id="mystery-workspace-set-variable">Set variable from result</button>';
  workspaceList.parentElement.insertBefore(box, workspaceList);
  box.querySelector('#mystery-workspace-add-intermediate').addEventListener('click', async()=>{ try { await addPuzzleIntermediate(); } catch(error) { workspaceStatus.textContent=`Error: ${error.message}`; } });
  box.querySelector('#mystery-workspace-set-variable').addEventListener('click', async()=>{ try { await setPuzzleVariable(box.querySelector('#mystery-workspace-variable').value); } catch(error) { workspaceStatus.textContent=`Error: ${error.message}`; } });
  return box;
}

async function loadMysteryWorkspaces() {
  const items = await workspaceRequest('/api/mystery-workspaces');
  workspaceList.replaceChildren();
  if (activeWorkspace) activeWorkspace = items.find(x=>x.id===activeWorkspace.id) || null;
  if (!items.length) { workspaceList.textContent = 'No mystery workspaces yet.'; return; }
  for (const x of items) {
    const row = document.createElement('div');
    const pre = document.createElement('pre');
    const active = activeWorkspace?.id === x.id ? ' [ACTIVE]' : '';
    pre.textContent = `${x.code ? x.code + ' — ' : ''}${x.title}${active}\nVariables: ${variablesText(x.variables)}\nFinal: ${x.latitude_formula || '—'} / ${x.longitude_formula || '—'}${x.final_waypoint_id ? `\nWaypoint: ${x.final_waypoint_id}` : ''}`;
    const use = document.createElement('button'); use.type='button'; use.textContent='Use in solver'; use.addEventListener('click',()=>activateWorkspace(x));
    const edit = document.createElement('button'); edit.type='button'; edit.textContent='Edit'; edit.addEventListener('click',()=>editWorkspace(x));
    const del = document.createElement('button'); del.type='button'; del.textContent='Delete'; del.addEventListener('click',async()=>{ await workspaceRequest(`/api/mystery-workspaces/${encodeURIComponent(x.id)}`,{method:'DELETE'}); if(activeWorkspace?.id===x.id)activeWorkspace=null; await loadMysteryWorkspaces(); });
    row.append(pre, use, edit, del); workspaceList.append(row);
  }
}

workspaceForm.addEventListener('submit', async event => {
  event.preventDefault();
  try {
    const payload = workspacePayload(workspaceForm);
    const url = editingWorkspaceId ? `/api/mystery-workspaces/${encodeURIComponent(editingWorkspaceId)}` : '/api/mystery-workspaces';
    const method = editingWorkspaceId ? 'PUT' : 'POST';
    const saved = await workspaceRequest(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
    workspaceStatus.textContent = `Saved ${saved.code ? saved.code + ' — ' : ''}${saved.title}.`;
    if (activeWorkspace?.id === saved.id) activeWorkspace = saved;
    resetWorkspaceForm(); await loadMysteryWorkspaces();
  } catch (error) { workspaceStatus.textContent = `Error: ${error.message}`; }
});
workspaceForm.querySelector('#mystery-workspace-cancel').addEventListener('click', resetWorkspaceForm);
document.addEventListener('caching-tools:puzzle-result', event => {
  latestPuzzleResult = String(event.detail?.text || '');
  const box = integrationControls();
  box.querySelector('#mystery-workspace-puzzle-result').textContent = `Latest puzzle result: ${latestPuzzleResult || '—'}`;
});
integrationControls();
loadMysteryWorkspaces().catch(error => { workspaceList.textContent = `Error: ${error.message}`; });
const footer = document.querySelector('footer'); if (footer) footer.textContent = 'Caching Tools M1.17';
window.loadMysteryWorkspaces = loadMysteryWorkspaces;
window.linkFinalWaypointToActiveWorkspace = linkFinalWaypointToActiveWorkspace;
window.getActiveMysteryWorkspace = () => activeWorkspace;
