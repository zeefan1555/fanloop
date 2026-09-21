const files = document.querySelector('#files');
const editor = document.querySelector('#editor');
const title = document.querySelector('#title');
const status = document.querySelector('#status');
const save = document.querySelector('#save');
const refresh = document.querySelector('#refresh');
let selected = '';
let observed = '';
let saved = '';
let stale = false;
let saving = false;
let loadGeneration = 0;

async function json(url, options) {
  const response = await fetch(url, options);
  const body = await response.json();
  if (!response.ok) throw Object.assign(new Error(body.error?.message || 'Request failed'), { response, body });
  return body;
}

function updateState(message) {
  const dirty = editor.value !== saved;
  save.disabled = !selected || !dirty || stale || saving;
  status.textContent = message || (stale ? 'Changed on disk. Refresh before editing again.' : dirty ? 'Unsaved changes' : 'Saved');
}

async function load(path, force = false) {
  const generation = ++loadGeneration;
  const next = await json(`/api/file?path=${encodeURIComponent(path)}`);
  if (generation !== loadGeneration) return;
  const dirty = editor.value !== saved;
  if (!force && selected === path && dirty) {
    if (next.sha256 !== observed) stale = true;
    updateState();
    return;
  }
  selected = path;
  observed = next.sha256;
  saved = next.content;
  stale = false;
  editor.value = next.content;
  editor.disabled = false;
  title.textContent = path;
  document.querySelectorAll('nav button').forEach(button => button.classList.toggle('active', button.dataset.path === path));
  updateState();
}

async function loadFiles() {
  const data = await json('/api/files');
  files.replaceChildren(...data.files.map(item => {
    const button = document.createElement('button');
    button.type = 'button';
    button.dataset.path = item.path;
    button.textContent = item.path;
    button.title = item.path;
    button.addEventListener('click', () => {
      if (!confirmDiscard()) return;
      load(item.path, true).catch(showError);
    });
    return button;
  }));
}

function showError(error) { updateState(error.message); }

function confirmDiscard() {
  return editor.value === saved || window.confirm('Discard unsaved changes?');
}

editor.addEventListener('input', () => { ++loadGeneration; updateState(); });
refresh.addEventListener('click', () => selected && confirmDiscard() && load(selected, true).catch(showError));
save.addEventListener('click', async () => {
  const path = selected;
  const content = editor.value;
  ++loadGeneration;
  saving = true;
  updateState('Saving...');
  try {
    const next = await json('/api/file', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path, content, observed_sha256: observed }) });
    if (selected === path) {
      observed = next.sha256;
      saved = next.content;
      stale = false;
      updateState('Saved');
    }
  } catch (error) {
    if (selected === path) {
      if (error.response?.status === 409) stale = true;
      showError(error);
    }
  } finally {
    saving = false;
    if (selected === path) updateState();
  }
});

setInterval(() => { if (selected && !saving) load(selected).catch(showError); }, 2000);
loadFiles().catch(showError);
