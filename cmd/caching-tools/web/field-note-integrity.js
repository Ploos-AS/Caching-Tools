const integritySection = document.createElement('section');
integritySection.className = 'tool';
integritySection.id = 'field-note-attachment-integrity';
integritySection.innerHTML = `
  <h2>Attachment integrity</h2>
  <p>Verify stored attachment bytes against the SHA-256 recorded when each attachment was created or restored.</p>
  <button type="button" id="attachment-integrity-run">Verify all attachments</button>
  <button type="button" id="attachment-integrity-download">Download integrity report</button>
  <p id="attachment-integrity-status" aria-live="polite"></p>
  <pre id="attachment-integrity-report" class="result">Integrity verification has not been run.</pre>`;
const archiveTool = document.querySelector('#field-note-archive');
const attachmentTool = document.querySelector('#field-note-attachments');
if (archiveTool) archiveTool.insertAdjacentElement('afterend', integritySection);
else if (attachmentTool) attachmentTool.insertAdjacentElement('afterend', integritySection);
else document.querySelector('main')?.append(integritySection);

const integrityStatus = integritySection.querySelector('#attachment-integrity-status');
const integrityOutput = integritySection.querySelector('#attachment-integrity-report');
let lastIntegrityReport = null;

function renderIntegrityReport(report) {
  const lines = [
    `Generated: ${report.generated_at}`,
    `Total: ${report.total}`,
    `OK: ${report.ok}`,
    `Mismatch: ${report.mismatch}`,
    `Missing: ${report.missing}`,
    `Unrecorded checksum: ${report.unrecorded}`,
    '',
  ];
  for (const item of report.items || []) {
    lines.push(`${String(item.status).toUpperCase()} · ${item.filename || item.attachment_id} · note ${item.note_id}`);
    if (item.expected_sha256) lines.push(`  expected ${item.expected_sha256}`);
    if (item.actual_sha256) lines.push(`  actual   ${item.actual_sha256}`);
  }
  integrityOutput.textContent = lines.join('\n');
}

async function runAttachmentIntegrity() {
  integrityStatus.textContent = 'Verifying local attachments...';
  try {
    const report = await requestJSON('/api/field-note-attachments/integrity');
    lastIntegrityReport = report;
    renderIntegrityReport(report);
    integrityStatus.textContent = report.mismatch || report.missing ? 'Integrity problems found.' : 'Integrity verification complete.';
  } catch (error) {
    integrityStatus.textContent = `Integrity error: ${error.message}`;
  }
}

function downloadIntegrityReport() {
  if (!lastIntegrityReport) {
    integrityStatus.textContent = 'Run integrity verification before downloading a report.';
    return;
  }
  const blob = new Blob([JSON.stringify(lastIntegrityReport, null, 2) + '\n'], {type:'application/json'});
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `caching-tools-attachment-integrity-${new Date().toISOString().replace(/[:.]/g, '-')}.json`;
  link.click();
  URL.revokeObjectURL(url);
}

integritySection.querySelector('#attachment-integrity-run').addEventListener('click', runAttachmentIntegrity);
integritySection.querySelector('#attachment-integrity-download').addEventListener('click', downloadIntegrityReport);
window.runAttachmentIntegrity = runAttachmentIntegrity;
const integrityFooter = document.querySelector('footer');
if (integrityFooter) integrityFooter.textContent = 'Caching Tools M1.39';
