// codeblock.js wires up runnable code fences rendered by the codeblock package.
// Each ".cb-runner" element carries its language and entry filename as data
// attributes and contains the source in its <code> element. Clicking "Run"
// posts the snippet to /api/codeblock/eval and renders the result in the
// element's output area.
(function () {
  const EVAL_PATH = '/api/codeblock/eval';
  const STYLE_ID = 'cb-runner-styles';

  function injectStyles() {
    if (document.getElementById(STYLE_ID)) return;
    const style = document.createElement('style');
    style.id = STYLE_ID;
    style.textContent =
      '.cb-runner{margin:1rem 0}' +
      '.cb-toolbar{margin:.25rem 0}' +
      '.cb-run{cursor:pointer;font:inherit;padding:.25rem .75rem;border:1px solid currentColor;border-radius:.375rem;background:transparent}' +
      '.cb-run[disabled]{opacity:.5;cursor:default}' +
      '.cb-output{margin-top:.5rem;border:1px solid #e5e7eb;border-radius:.375rem;overflow:hidden}' +
      '.cb-output-label{font-size:.75rem;font-weight:600;padding:.25rem .5rem;background:#f8fafc;border-bottom:1px solid #e5e7eb}' +
      '.cb-output-body pre{margin:0;padding:.5rem;white-space:pre-wrap;word-break:break-word}' +
      '.cb-output-body iframe{width:100%;border:0;min-height:120px;background:#fff}' +
      '.cb-error{color:#b91c1c;padding:.5rem;white-space:pre-wrap}';
    document.head.appendChild(style);
  }

  function sourceOf(runner) {
    const code = runner.querySelector('.cb-code code') || runner.querySelector('.cb-code');
    return code ? code.textContent : '';
  }

  function renderResult(body, result) {
    body.innerHTML = '';
    if (result.error) {
      const err = document.createElement('div');
      err.className = 'cb-error';
      err.textContent = result.error;
      body.appendChild(err);
      return;
    }

    const contentType = result.contentType || 'text/plain';
    if (contentType === 'text/html') {
      const iframe = document.createElement('iframe');
      iframe.setAttribute('sandbox', 'allow-same-origin');
      iframe.srcdoc = result.content || '';
      body.appendChild(iframe);
      return;
    }

    const pre = document.createElement('pre');
    pre.textContent = result.content || '';
    body.appendChild(pre);
  }

  async function run(runner) {
    const language = runner.dataset.cbLanguage;
    const entry = runner.dataset.cbEntry || 'index';
    const output = runner.querySelector('.cb-output');
    const body = runner.querySelector('.cb-output-body');
    const button = runner.querySelector('.cb-run');

    const files = {};
    files[entry] = sourceOf(runner);

    output.hidden = false;
    body.innerHTML = '<pre>Running…</pre>';
    if (button) button.disabled = true;

    try {
      const res = await fetch(EVAL_PATH, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ language: language, entry: entry, files: files }),
      });
      const result = await res.json();
      renderResult(body, result);
    } catch (e) {
      renderResult(body, { error: String(e) });
    } finally {
      if (button) button.disabled = false;
    }
  }

  function init(root) {
    injectStyles();
    (root || document).querySelectorAll('.cb-runner').forEach(function (runner) {
      if (runner.dataset.cbReady) return;
      runner.dataset.cbReady = '1';
      const button = runner.querySelector('.cb-run');
      if (button) button.addEventListener('click', function () { run(runner); });
    });
  }

  document.addEventListener('DOMContentLoaded', function () { init(document); });
  document.addEventListener('htmx:afterSwap', function (e) { init(e.target); });
})();
