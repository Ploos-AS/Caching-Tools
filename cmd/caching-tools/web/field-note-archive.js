const fieldNoteArchiveSection = document.createElement('section');
fieldNoteArchiveSection.className = 'tool';
fieldNoteArchiveSection.id = 'field-note-archive';
fieldNoteArchiveSection.innerHTML = `
  <h2>Field-note archive backup / restore</h2>
  <p>Export field notes and their local attachments together as a ZIP archive. Metadata stays in <code>manifest.json</code>; attachment bytes stay as separate files.</p>
  <div class="form-grid nav-grid">
    <a id="field-note-archive-export" href="/api/field-notes/archive">Download ZIP backup</a>
    <label>ZIP backup<input id="field-note-archive-file" type="file" accept=".zip,application/zip"></label>
    <button type="button" id="field-note-archive-import">Restore ZIP backup</button>
  </div>
  <p id="field-note-archive-status" aria-live="polite"></p>`;

const attachmentsTool = document.querySelector('#field-note-attachments');
const notesTool = document.querySelector('#field-notes');
if (attachmentsTool) attachmentsTool.insertAdjacentElement('afterend', fieldNoteArchiveSection);
else if (notesTool) notesTool.insertAdjacentElement('afterend', fieldNoteArchiveSection);
else document.querySelector('main')?.append(fieldNoteArchiveSection);

const archiveFile = fieldNoteArchiveSection.querySelector('#field-note-archive-file');
const archiveStatus = fieldNoteArchiveSection.querySelector('#field-note-archive-status');

async function restoreFieldNoteArchiveFile() {
  const file = archiveFile.files[0];
  if (!file) {
    archiveStatus.textContent = 'Select a ZIP backup first.';
    return;
  }
  if (file.size > 128 * 1024 * 1024) {
    archiveStatus.textContent = 'Archive exceeds the 128 MiB upload limit.';
    return;
  }
  try {
    archiveStatus.textContent = 'Validating and restoring archive...';
    const response = await fetch('/api/field-notes/archive', {
      method: 'POST',
      headers: {'Content-Type': 'application/zip'},
      body: file
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || `HTTP ${response.status}`);
    archiveFile.value = '';
    archiveStatus.textContent = `Restored ${payload.imported_notes} field note${payload.imported_notes === 1 ? '' : 's'} and ${payload.imported_attachments} attachment${payload.imported_attachments === 1 ? '' : 's'} with new local IDs.`;
    if (typeof window.refreshFieldNotes === 'function') await window.refreshFieldNotes();
    if (typeof window.refreshFieldNoteAttachments === 'function') await window.refreshFieldNoteAttachments();
    if (typeof window.refreshLogbookDashboard === 'function') await window.refreshLogbookDashboard();
    if (typeof window.refreshMapWaypointLogbook === 'function') await window.refreshMapWaypointLogbook();
  } catch (error) {
    archiveStatus.textContent = `Archive restore error: ${error.message}`;
  }
}

fieldNoteArchiveSection.querySelector('#field-note-archive-import').addEventListener('click', restoreFieldNoteArchiveFile);
window.restoreFieldNoteArchiveFile = restoreFieldNoteArchiveFile;
const archiveFooter = document.querySelector('footer'); if (archiveFooter) archiveFooter.textContent = 'Caching Tools M1.37';
