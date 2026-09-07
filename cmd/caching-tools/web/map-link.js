document.addEventListener('caching-tools:map-select', (event) => {
  const {kind, name} = event.detail || {};
  const list = kind === 'waypoint' ? document.querySelector('#waypoint-list') : document.querySelector('#path-list');
  if (!list) return;

  for (const row of list.children) row.classList.remove('list-selected');
  const rows = [...list.children];
  const selected = rows.find((row) => {
    const text = row.querySelector('pre')?.textContent || row.textContent || '';
    return kind === 'waypoint' ? text.startsWith(name) : text.includes(`${kind}: ${name}`);
  });
  if (!selected) return;
  selected.classList.add('list-selected');
  selected.scrollIntoView({behavior: 'smooth', block: 'center'});
});

window.addEventListener('load', () => {
  if (document.querySelector('script[data-map-editor]')) return;
  const script = document.createElement('script');
  script.src = '/map-editor.js';
  script.dataset.mapEditor = 'true';
  document.body.append(script);
});
