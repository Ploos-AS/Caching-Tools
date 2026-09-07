async function puzzleCall(operation, extra = {}) {
  const response = await fetch('/api/puzzle', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({operation, ...extra})
  });
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
  return data;
}

function emitPuzzleResult(operation, text) {
  document.dispatchEvent(new CustomEvent('caching-tools:puzzle-result', {detail:{operation, text}}));
}

function bindPuzzleForm(id, operation, payload, render) {
  const form = document.querySelector(id);
  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const out = form.querySelector('.puzzle-output');
    try {
      const op = typeof operation === 'function' ? operation(new FormData(form)) : operation;
      const data = await puzzleCall(op, payload(new FormData(form)));
      const text = render(data);
      out.textContent = text;
      emitPuzzleResult(op, text);
    } catch (error) {
      out.textContent = `Error: ${error.message}`;
    }
  });
}

bindPuzzleForm('#puzzle-a1z26', 'a1z26', f => ({text:f.get('text')}), d => `Values: ${d.values.join(' ')}\nSum: ${d.sum}`);
bindPuzzleForm('#puzzle-caesar', 'caesar', f => ({text:f.get('text'), shift:Number(f.get('shift'))}), d => d.result);
bindPuzzleForm('#puzzle-digits', 'digitsum', f => ({text:f.get('text')}), d => `Digit sum: ${d.digit_sum}\nDigital root: ${d.digital_root}`);
bindPuzzleForm('#puzzle-substitution', 'substitution', f => ({text:f.get('text'), alphabet:f.get('alphabet'), mapping:f.get('mapping')}), d => d.result);
bindPuzzleForm('#puzzle-morse', f => f.get('mode'), f => ({text:f.get('text')}), d => d.result);
bindPuzzleForm('#puzzle-bacon', f => f.get('mode'), f => ({text:f.get('text')}), d => d.result);
bindPuzzleForm('#puzzle-rot47', 'rot47', f => ({text:f.get('text')}), d => d.result);
bindPuzzleForm('#puzzle-base', 'base', f => ({text:f.get('text'), from_base:Number(f.get('from-base')), to_base:Number(f.get('to-base'))}), d => d.result);
bindPuzzleForm('#puzzle-keypad', 'keypad', f => ({text:f.get('text')}), d => `Values: ${d.values.join(' ')}\nSum: ${d.sum}`);
