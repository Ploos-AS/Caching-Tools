const fieldNoteAttachmentsSection = document.createElement('section');
fieldNoteAttachmentsSection.className = 'tool';
fieldNoteAttachmentsSection.id = 'field-note-attachments';
fieldNoteAttachmentsSection.innerHTML = `
  <h2>Field-note attachments</h2>
  <p>Attach local files to a persistent field note. Files stay under <code>/data/field-note-attachments</code>. Maximum size is 10 MiB per file.</p>
  <div class="form-grid nav-grid">
    <label>Field note<select id="field-note-attachment-note"><option value="">Select a field note</option></select></label>
    <label>Attachment<input id="field-note-attachment-file" type="file" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf,text/plain,application/json,text/csv,application/gpx+xml,application/xml,text/xml"></label>
    <button type="button" id="field-note-attachment-upload">Upload attachment</button>
    <button type="button" id="field-note-attachment-refresh">Refresh attachments</button>
  </div>
  <p id="field-note-attachment-status" aria-live="polite"></p>
  <div id="field-note-attachment-list" class="result">Select a field note.</div>`;

const mapLogbookTool = document.querySelector('#map-waypoint-logbook');
const fieldNoteDashboardTool = document.querySelector('#field-note-dashboard');
const fieldNotesToolForAttachments = document.querySelector('#field-notes');
if (mapLogbookTool) mapLogbookTool.insertAdjacentElement('afterend', fieldNoteAttachmentsSection);
else if (fieldNoteDashboardTool) fieldNoteDashboardTool.insertAdjacentElement('afterend', fieldNoteAttachmentsSection);
else if (fieldNotesToolForAttachments) fieldNotesToolForAttachments.insertAdjacentElement('afterend', fieldNoteAttachmentsSection);
else document.querySelector('main')?.append(fieldNoteAttachmentsSection);

const attachmentNoteSelect = fieldNoteAttachmentsSection.querySelector('#field-note-attachment-note');
const attachmentFileInput = fieldNoteAttachmentsSection.querySelector('#field-note-attachment-file');
const attachmentStatus = fieldNoteAttachmentsSection.querySelector('#field-note-attachment-status');
const attachmentList = fieldNoteAttachmentsSection.querySelector('#field-note-attachment-list');

function attachmentSize(size) {
  const bytes = Number(size || 0);
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}

async function refreshAttachmentNoteOptions() {
  const selected = attachmentNoteSelect.value;
  const notes = await requestJSON('/api/field-notes');
  attachmentNoteSelect.replaceChildren(new Option('Select a field note', ''));
  for (const note of notes) {
    const label = `${note.occurred_at} · ${note.title}`;
    attachmentNoteSelect.append(new Option(label, note.id));
  }
  if ([...attachmentNoteSelect.options].some(option => option.value === selected)) attachmentNoteSelect.value = selected;
}

async function loadFieldNoteAttachments() {
  const noteID = attachmentNoteSelect.value;
  attachmentList.replaceChildren();
  if (!noteID) {
    attachmentList.textContent = 'Select a field note.';
    return;
  }
  try {
    const items = await requestJSON(`/api/field-notes/${encodeURIComponent(noteID)}/attachments`);
    if (!items.length) {
      attachmentList.textContent = 'No attachments for this field note.';
      return;
    }
    for (const item of items) {
      const row = document.createElement('div');
      const text = document.createElement('pre');
      text.textContent = `${item.filename}\n${item.content_type} · ${attachmentSize(item.size)}\n${item.created_at}`;
      const download = document.createElement('a');
      download.href = `/api/field-notes/${encodeURIComponent(noteID)}/attachments/${encodeURIComponent(item.id)}`;
      download.textContent = 'Download';
      const remove = document.createElement('button');
      remove.type = 'button';
      remove.textContent = 'Delete attachment';
      remove.addEventListener('click', async () => {
        try {
          await requestJSON(`/api/field-notes/${encodeURIComponent(noteID)}/attachments/${encodeURIComponent(item.id)}`, {method:'DELETE'});
          attachmentStatus.textContent = `Deleted attachment ${item.filename}.`;
          await loadFieldNoteAttachments();
        } catch (error) {
          attachmentStatus.textContent = `Attachment error: ${error.message}`;
        }
      });
      row.append(text, download, remove);
      attachmentList.append(row);
    }
  } catch (error) {
    attachmentList.textContent = `Attachment error: ${error.message}`;
  }
}

async function uploadFieldNoteAttachment() {
  const noteID = attachmentNoteSelect.value;
  const file = attachmentFileInput.files[0];
  if (!noteID) {
    attachmentStatus.textContent = 'Select a field note first.';
    return;
  }
  if (!file) {
    attachmentStatus.textContent = 'Select a file first.';
    return;
  }
  if (file.size > 10 * 1024 * 1024) {
    attachmentStatus.textContent = 'Attachment exceeds the 10 MiB limit.';
    return;
  }
  try {
    const form = new FormData();
    form.append('file', file, file.name);
    const response = await fetch(`/api/field-notes/${encodeURIComponent(noteID)}/attachments`, {method:'POST', body:form});
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || `HTTP ${response.status}`);
    attachmentFileInput.value = '';
    attachmentStatus.textContent = `Uploaded ${payload.filename} (${attachmentSize(payload.size)}).`;
    await loadFieldNoteAttachments();
  } catch (error) {
    attachmentStatus.textContent = `Attachment error: ${error.message}`;
  }
}

attachmentNoteSelect.addEventListener('change', loadFieldNoteAttachments);
fieldNoteAttachmentsSection.querySelector('#field-note-attachment-upload').addEventListener('click', uploadFieldNoteAttachment);
fieldNoteAttachmentsSection.querySelector('#field-note-attachment-refresh').addEventListener('click', loadFieldNoteAttachments);
const attachmentObservedNoteList = document.querySelector('#field-note-list');
if (attachmentObservedNoteList) {
  new MutationObserver(() => { void refreshAttachmentNoteOptions(); }).observe(attachmentObservedNoteList, {childList:true, subtree:true});
}

document.addEventListener('caching-tools:map-select', async event => {
  const detail = event.detail || {};
  if (detail.kind !== 'waypoint') return;
  const notes = await requestJSON('/api/field-notes');
  const linked = notes.find(note => note.waypoint_id === detail.id);
  if (!linked) return;
  await refreshAttachmentNoteOptions();
  attachmentNoteSelect.value = linked.id;
  await loadFieldNoteAttachments();
});

void refreshAttachmentNoteOptions().then(loadFieldNoteAttachments).catch(error => { attachmentStatus.textContent = `Attachment error: ${error.message}`; });
window.refreshFieldNoteAttachments = async () => { await refreshAttachmentNoteOptions(); await loadFieldNoteAttachments(); };
const attachmentFooter = document.querySelector('footer'); if (attachmentFooter) attachmentFooter.textContent = 'Caching Tools M1.36';
