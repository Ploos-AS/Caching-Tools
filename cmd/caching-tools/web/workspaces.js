const workspaceForm = document.querySelector('#mystery-workspace-form');
const workspaceList = document.querySelector('#mystery-workspace-list');
const workspaceStatus = document.querySelector('#mystery-workspace-status');
let editingWorkspaceId = '';

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

async function loadMysteryWorkspaces() {
  const items = await workspaceRequest('/api/mystery-workspaces');
  workspaceList.replaceChildren();
  if (!items.length) { workspaceList.textContent = 'No mystery workspaces yet.'; return; }
  for (const x of items) {
    const row = document.createElement('div');
    const pre = document.createElement('pre');
    pre.textContent = `${x.code ? x.code + ' — ' : ''}${x.title}\nVariables: ${variablesText(x.variables)}\nFinal: ${x.latitude_formula || '—'} / ${x.longitude_formula || '—'}${x.final_waypoint_id ? `\nWaypoint: ${x.final_waypoint_id}` : ''}`;
    const edit = document.createElement('button'); edit.type='button'; edit.textContent='Edit'; edit.addEventListener('click',()=>editWorkspace(x));
    const del = document.createElement('button'); del.type='button'; del.textContent='Delete'; del.addEventListener('click',async()=>{ await workspaceRequest(`/api/mystery-workspaces/${encodeURIComponent(x.id)}`,{method:'DELETE'}); await loadMysteryWorkspaces(); });
    row.append(pre, edit, del); workspaceList.append(row);
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
    resetWorkspaceForm(); await loadMysteryWorkspaces();
  } catch (error) { workspaceStatus.textContent = `Error: ${error.message}`; }
});
workspaceForm.querySelector('#mystery-workspace-cancel').addEventListener('click', resetWorkspaceForm);
loadMysteryWorkspaces().catch(error => { workspaceList.textContent = `Error: ${error.message}`; });
window.loadMysteryWorkspaces = loadMysteryWorkspaces;
