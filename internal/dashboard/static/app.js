const files = document.querySelector('#files');
const editor = document.querySelector('#editor');
const title = document.querySelector('#title');
const status = document.querySelector('#status');
const refresh = document.querySelector('#refresh');
const workflowSelect = document.querySelector('#workflowSelect');
const fragmentSelect = document.querySelector('#fragmentSelect');
const viewport = document.querySelector('#viewport');
const world = document.querySelector('#world');
const edges = document.querySelector('#edges');
const nodes = document.querySelector('#nodes');
const stageBands = document.querySelector('#stageBands');
const jobLabels = document.querySelector('#jobLabels');
const terminalNode = document.querySelector('#terminalNode');
const lineNumbers = document.querySelector('#lineNumbers');
const MIN_SCALE = .8;

let fileItems = [];
let topology = null;
let selectedWorkflow = '';
let selectedStep = '';
let focusedStep = '';
let activeTab = 'step';
let selectedFragment = null;
let selected = '';
let observed = '';
let stateMessage = '';
let stateError = false;
let loadGeneration = 0;
let topologyGeneration = 0;
let topologyRequest = null;
let graphLayout = null;
let scale = 1;
let translate = { x: 0, y: 0 };

async function json(url, options) {
  const response = await fetch(url, options);
  const body = await response.json();
  if (!response.ok) throw Object.assign(new Error(body.error?.message || 'Request failed'), { response, body });
  return body;
}



function updateState(message, error = false) {
  if (message !== undefined) {
    stateMessage = message;
    stateError = error;
  }
  status.textContent = stateMessage || (selected ? 'Read-only · published configuration' : 'Ready');
  status.classList.toggle('error', stateError && Boolean(stateMessage));
  document.querySelector('#healthText').textContent = 'Published configuration';
  updateLineNumbers();
}

function updateLineNumbers() {
  const count = Math.max(editor.value.split('\n').length, 1);
  lineNumbers.textContent = Array.from({ length: count }, (_, index) => index + 1).join('\n');
}

function lineOffsets(content, fragment) {
  const lines = content.split('\n');
  const startLine = Math.max(1, Math.min(fragment.start_line, lines.length));
  const endLine = Math.max(startLine, Math.min(fragment.end_line, lines.length));
  let start = 0;
  for (let index = 0; index < startLine - 1; index += 1) start += lines[index].length + 1;
  let end = start;
  for (let index = startLine - 1; index < endLine; index += 1) end += lines[index].length + (index < lines.length - 1 ? 1 : 0);
  return { start, end, startLine, endLine };
}

function selectSourceRange(fragment) {
  if (!fragment || fragment.path !== selected) {
    document.querySelector('#selection').textContent = selected ? 'Complete file' : 'No source selected';
    return;
  }
  const range = lineOffsets(editor.value, fragment);
  editor.focus({ preventScroll: true });
  editor.setSelectionRange(range.start, range.end);
  const lineHeight = parseFloat(getComputedStyle(editor).lineHeight) || 20;
  editor.scrollTop = Math.max(0, (range.startLine - 2) * lineHeight);
  document.querySelector('#selection').textContent = `Lines ${range.startLine}-${range.endLine}`;
}

function scheduleSourceRangeSelection(fragment) {
  const generation = loadGeneration;
  requestAnimationFrame(() => {
    if (generation === loadGeneration) selectSourceRange(fragment);
  });
}

async function load(path, force = false, fragment = selectedFragment) {
  const generation = ++loadGeneration;
  const next = await json(`/api/file?path=${encodeURIComponent(path)}`);
  if (generation !== loadGeneration) return;
  if (!force && selected === path && next.sha256 === observed) {
    updateState();
    return;
  }
  selected = path;
  observed = next.sha256;
  editor.value = next.content;
  editor.disabled = false;
  title.textContent = path;
  document.querySelectorAll('#files button').forEach(button => button.classList.toggle('active', button.dataset.path === path));
  updateState('');
  scheduleSourceRangeSelection(fragment);
}

function renderFiles() {
  const visible = fileItems.filter(item => item.path.startsWith(`${selectedWorkflow}/`));
  files.replaceChildren(...visible.map(item => {
    const button = document.createElement('button');
    button.type = 'button';
    button.dataset.path = item.path;
    button.textContent = item.path.slice(selectedWorkflow.length + 1);
    button.title = item.path;
    button.addEventListener('click', () => {
      activeTab = 'file';
      selectedFragment = null;
      renderInspector();
      load(item.path, true, null).catch(showError);
    });
    return button;
  }));
}

async function loadFiles() {
  const data = await json('/api/files');
  fileItems = data.files;
  const workflowIDs = [...new Set(fileItems.map(item => item.path.split('/')[0]))];
  workflowSelect.replaceChildren(...workflowIDs.map(id => {
    const option = document.createElement('option');
    option.value = id;
    option.textContent = id;
    return option;
  }));
  workflowSelect.disabled = workflowIDs.length === 0;
  selectedWorkflow = workflowIDs.includes(selectedWorkflow) ? selectedWorkflow : workflowIDs[0] || '';
  workflowSelect.value = selectedWorkflow;
  renderFiles();
  if (selectedWorkflow) await loadTopology(selectedWorkflow, true, true);
}

function flattenSteps(value) {
  return value.workflow.stages.flatMap(stage => stage.jobs.flatMap(job => job.steps.map(step => ({ ...step, stage, job }))));
}

async function loadTopology(workflowID, openSource = false, fit = false, force = false) {
  if (!openSource && topologyRequest && !force) return topologyRequest;
  const generation = ++topologyGeneration;
  const request = (async () => {
    const refresh = force ? `&refresh=${generation}` : '';
    const next = await json(`/api/topology?workflow=${encodeURIComponent(workflowID)}${refresh}`);
    if (generation !== topologyGeneration) return;
    topology = next;
    const allSteps = flattenSteps(next);
    if (!allSteps.some(step => step.id === selectedStep)) selectedStep = allSteps[0]?.id || '';
    if (!allSteps.some(step => step.id === focusedStep)) focusedStep = selectedStep;
    renderSummary();
    renderGraph(fit);
    renderInspector();
    if (!openSource) scheduleSourceRangeSelection(selectedFragment);
    document.querySelector('#graphMessage').hidden = true;
    world.hidden = false;
    if (openSource) await chooseDefaultFragment();
  })();
  topologyRequest = request;
  try {
    return await request;
  } finally {
    if (topologyRequest === request) topologyRequest = null;
  }
}

function renderSummary() {
  const allSteps = flattenSteps(topology);
  const forward = topology.routes.filter(route => route.kind !== 'return').length;
  const returns = topology.routes.filter(route => route.kind === 'return').length;
  document.querySelector('#stageCount').textContent = topology.workflow.stages.length;
  document.querySelector('#stepCount').textContent = allSteps.length;
  document.querySelector('#forwardTotal').textContent = forward;
  document.querySelector('#returnTotal').textContent = returns;
  document.querySelector('#forwardLegend').textContent = forward;
  document.querySelector('#returnLegend').textContent = returns;
  document.querySelector('#routeReadout').textContent = `${topology.routes.length} routes`;
}

function buildLayout() {
  const stageGap = 20;
  const stageWidth = 200;
  const nodeWidth = 172;
  const nodeHeight = 72;
  const rowGap = 16;
  const stageTop = 24;
  const jobHeader = 28;
  const stagePadding = 14;
  let x = 24;
  let maxHeight = 420;
  const stageValues = [];
  const jobs = [];
  const stepMap = new Map();
  topology.workflow.stages.forEach(stage => {
    let y = stageTop + 46;
    stage.jobs.forEach(job => {
      jobs.push({ ...job, stageId: stage.id, x: x + stagePadding, y });
      y += jobHeader;
      job.steps.forEach(step => {
        stepMap.set(step.id, { ...step, stage, job, x: x + stagePadding, y, width: nodeWidth, height: nodeHeight });
        y += nodeHeight + rowGap;
      });
      y += 10;
    });
    const height = Math.max(260, y - stageTop + 6);
    stageValues.push({ ...stage, x, y: stageTop, width: stageWidth, height });
    maxHeight = Math.max(maxHeight, height + stageTop + 34);
    x += stageWidth + stageGap;
  });
  return { stages: stageValues, jobs, steps: stepMap, width: Math.max(680, x + 100), height: maxHeight };
}

function renderGraph(fit = false) {
  if (!topology) return;
  graphLayout = buildLayout();
  world.style.width = `${graphLayout.width}px`;
  world.style.height = `${graphLayout.height}px`;
  stageBands.replaceChildren(...graphLayout.stages.map(stage => {
    const band = document.createElement('section');
    band.className = 'stage-band';
    Object.assign(band.style, { left: `${stage.x}px`, top: `${stage.y}px`, width: `${stage.width}px`, height: `${stage.height}px` });
    const heading = document.createElement('header');
    heading.innerHTML = `<strong>${escapeHTML(stage.name)}</strong><span>${stage.jobs.length} jobs</span>`;
    band.append(heading);
    return band;
  }));
  jobLabels.replaceChildren(...graphLayout.jobs.map(job => {
    const label = document.createElement('div');
    label.className = 'job-label';
    Object.assign(label.style, { left: `${job.x}px`, top: `${job.y}px` });
    label.innerHTML = `<strong>${escapeHTML(job.name)}</strong><code>${escapeHTML(job.id)}</code>`;
    return label;
  }));
  const allSteps = flattenSteps(topology);
  nodes.replaceChildren(...allSteps.map((step, index) => {
    const position = graphLayout.steps.get(step.id);
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `step-node${step.id === selectedStep ? ' active' : ''}`;
    button.dataset.id = step.id;
    Object.assign(button.style, { left: `${position.x}px`, top: `${position.y}px`, width: `${position.width}px`, height: `${position.height}px` });
    button.innerHTML = `<span class="node-index">${String(index + 1).padStart(2, '0')}</span><strong>${escapeHTML(step.name)}</strong><code>${escapeHTML(step.id)}</code><span class="node-meta">${escapeHTML(step.executor)}</span>`;
    button.addEventListener('click', event => { event.stopPropagation(); selectStep(step.id); });
    return button;
  }));
  const lastStage = graphLayout.stages.at(-1);
  Object.assign(terminalNode.style, { left: `${lastStage.x + lastStage.width + 36}px`, top: `${Math.max(150, graphLayout.height / 2 - 20)}px` });
  renderEdges();
  if (fit) requestAnimationFrame(fitGraph);
}

function routePoint(step, side, offset = 0) {
  return { x: step.x + (side === 'right' ? step.width : 0), y: step.y + step.height / 2 + offset };
}

function routeCurve(from, to, route, lane) {
  if (route.to_step_id === '__terminal__') {
    const start = routePoint(from, 'right');
    const end = { x: parseFloat(terminalNode.style.left), y: parseFloat(terminalNode.style.top) + 20 };
    return `M ${start.x} ${start.y} C ${start.x + 48} ${start.y}, ${end.x - 48} ${end.y}, ${end.x} ${end.y}`;
  }
  if (from.id === to.id) {
    const point = routePoint(from, 'right', (lane % 3) * 5);
    return `M ${point.x} ${point.y} C ${point.x + 64} ${point.y - 48}, ${point.x + 64} ${point.y + 48}, ${point.x} ${point.y + 8}`;
  }
  const isReturn = route.kind === 'return';
  const start = routePoint(from, isReturn ? 'left' : 'right', (lane % 5) - 2);
  const end = routePoint(to, isReturn ? 'right' : 'left', (lane % 5) - 2);
  const bend = Math.max(58, Math.abs(end.x - start.x) * .42);
  if (!isReturn) return `M ${start.x} ${start.y} C ${start.x + bend} ${start.y}, ${end.x - bend} ${end.y}, ${end.x} ${end.y}`;
  const lift = 46 + (lane % 7) * 7;
  return `M ${start.x} ${start.y} C ${start.x - lift} ${start.y}, ${end.x + lift} ${end.y}, ${end.x} ${end.y}`;
}

function renderEdges() {
  const namespace = 'http://www.w3.org/2000/svg';
  edges.setAttribute('viewBox', `0 0 ${graphLayout.width} ${graphLayout.height}`);
  edges.replaceChildren(makeMarkers(namespace));
  topology.routes.forEach((route, index) => {
    const from = graphLayout.steps.get(route.from_step_id);
    const to = route.to_step_id === '__terminal__' ? null : graphLayout.steps.get(route.to_step_id);
    if (!from || (route.to_step_id !== '__terminal__' && !to)) return;
    const path = document.createElementNS(namespace, 'path');
    path.setAttribute('d', routeCurve(from, to, route, index));
    path.setAttribute('class', `edge ${route.kind}`);
    path.setAttribute('marker-end', `url(#arrow-${route.kind})`);
    path.dataset.from = route.from_step_id;
    path.dataset.to = route.to_step_id;
    const label = document.createElementNS(namespace, 'title');
    label.textContent = `${route.kind}: ${route.from_step_id} → ${route.to_step_id}`;
    path.append(label);
    edges.append(path);
  });
  filterEdges();
}

function makeMarkers(namespace) {
  const defs = document.createElementNS(namespace, 'defs');
  [['forward', '#16866f'], ['terminal', '#657081'], ['return', '#c46b28']].forEach(([id, color]) => {
    const marker = document.createElementNS(namespace, 'marker');
    marker.setAttribute('id', `arrow-${id}`);
    marker.setAttribute('markerWidth', '8');
    marker.setAttribute('markerHeight', '8');
    marker.setAttribute('refX', '7');
    marker.setAttribute('refY', '4');
    marker.setAttribute('orient', 'auto');
    const path = document.createElementNS(namespace, 'path');
    path.setAttribute('d', 'M0,0 L8,4 L0,8 Z');
    path.setAttribute('fill', color);
    marker.append(path);
    defs.append(marker);
  });
  return defs;
}

function filterEdges() {
  const showForward = document.querySelector('#showForward').checked;
  const showReturn = document.querySelector('#showReturn').checked;
  document.querySelectorAll('.edge').forEach(path => {
    const hidden = (path.classList.contains('return') && !showReturn) || (!path.classList.contains('return') && !showForward);
    const related = path.dataset.from === focusedStep || path.dataset.to === focusedStep;
    path.classList.toggle('hidden', hidden);
    path.classList.toggle('related', !hidden && related);
    path.classList.toggle('unrelated', !hidden && focusedStep && !related);
  });
  document.querySelectorAll('.step-node').forEach(node => {
    const related = [...document.querySelectorAll('.edge.related')].some(path => path.dataset.from === node.dataset.id || path.dataset.to === node.dataset.id);
    node.classList.toggle('dim', focusedStep && node.dataset.id !== focusedStep && !related);
  });
}

function currentStep() {
  return topology ? flattenSteps(topology).find(step => step.id === selectedStep) : null;
}

function currentFragments() {
  return topology?.fragments?.[selectedStep]?.[activeTab] || [];
}

function renderInspector() {
  const step = currentStep();
  if (!step) return;
  document.querySelector('#stageName').textContent = `${step.stage.name} / ${step.job.name}`;
  document.querySelector('#stepName').textContent = step.name;
  document.querySelector('#stepId').textContent = step.id;
  document.querySelector('#executor').textContent = step.executor;
  const fragments = topology.fragments[step.id];
  document.querySelector('#forwardCount').textContent = fragments.forward.length;
  document.querySelector('#returnCount').textContent = fragments.return.length;
  document.querySelector('#skillCount').textContent = fragments.skills.length;
  document.querySelectorAll('#tabs button').forEach(button => button.classList.toggle('active', button.dataset.tab === activeTab));
  renderFragmentSelect();
}

function renderFragmentSelect() {
  const values = currentFragments();
  fragmentSelect.replaceChildren(...values.map(fragment => {
    const option = document.createElement('option');
    option.value = fragment.id;
    option.textContent = `${fragment.label} · ${fragment.path}:${fragment.start_line}`;
    return option;
  }));
  fragmentSelect.disabled = values.length === 0;
  if (selectedFragment) {
    const match = values.find(fragment => fragment.id === selectedFragment.id);
    selectedFragment = match || values[0] || null;
  } else {
    selectedFragment = values[0] || null;
  }
  if (selectedFragment) fragmentSelect.value = selectedFragment.id;
}

async function chooseDefaultFragment() {
  renderInspector();
  if (!selectedFragment) {
    editor.disabled = true;
    title.textContent = `No ${activeTab} source for this Step`;
    document.querySelector('#selection').textContent = 'No source selected';
    return;
  }
  await load(selectedFragment.path, true, selectedFragment);
}

async function selectStep(stepID) {
  if (stepID === selectedStep) {
    if (activeTab === 'file') {
      activeTab = 'step';
      selectedFragment = null;
      renderInspector();
      await chooseDefaultFragment().catch(showError);
    }
    focusedStep = stepID;
    document.querySelector('#canvasHint').textContent = `Focused on ${currentStep().name}`;
    filterEdges();
    return;
  }

  selectedStep = stepID;
  focusedStep = stepID;
  activeTab = 'step';
  selectedFragment = null;
  renderGraph();
  renderInspector();
  document.querySelector('#canvasHint').textContent = `Focused on ${currentStep().name}`;
  await chooseDefaultFragment().catch(showError);
}

async function selectTab(tab) {
  if (tab === activeTab) return;

  activeTab = tab;
  selectedFragment = null;
  renderInspector();
  await chooseDefaultFragment().catch(showError);
}

function applyTransform() {
  world.style.transform = `translate(${translate.x}px, ${translate.y}px) scale(${scale})`;
  document.querySelector('#zoomValue').textContent = `${Math.round(scale * 100)}%`;
}

function fitGraph() {
  if (!graphLayout || world.hidden) return;
  const rect = viewport.getBoundingClientRect();
  viewport.scrollLeft = 0;
  viewport.scrollTop = 0;
  scale = Math.min(1, (rect.width - 28) / graphLayout.width, (rect.height - 24) / graphLayout.height);
  scale = Math.max(MIN_SCALE, scale);
  translate = { x: Math.max(12, (rect.width - graphLayout.width * scale) / 2), y: Math.max(10, (rect.height - graphLayout.height * scale) / 2) };
  applyTransform();
}

function showError(error) {
  updateState(error.message, true);
  document.querySelector('#graphMessage').textContent = error.message;
  document.querySelector('#graphMessage').hidden = Boolean(topology);
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[character]));
}

editor.addEventListener('scroll', () => { lineNumbers.scrollTop = editor.scrollTop; });
refresh.addEventListener('click', async () => {
  if (!selected) return;
  try {
    await load(selected, true, selectedFragment);
    await loadTopology(selectedWorkflow, false, false, true);
  } catch (error) { showError(error); }
});
workflowSelect.addEventListener('change', async event => {
  selectedWorkflow = event.target.value;
  selectedStep = '';
  focusedStep = '';
  activeTab = 'step';
  selectedFragment = null;
  renderFiles();
  try { await loadTopology(selectedWorkflow, true, true); } catch (error) { showError(error); }
});
fragmentSelect.addEventListener('change', () => {
  selectedFragment = currentFragments().find(fragment => fragment.id === fragmentSelect.value) || null;
  if (selectedFragment) load(selectedFragment.path, true, selectedFragment).catch(showError);
});
document.querySelectorAll('#tabs button').forEach(button => button.addEventListener('click', () => selectTab(button.dataset.tab)));
document.querySelectorAll('#showForward, #showReturn').forEach(input => input.addEventListener('change', filterEdges));
document.querySelector('#zoomIn').addEventListener('click', () => { scale = Math.min(1.45, scale + .1); applyTransform(); });
document.querySelector('#zoomOut').addEventListener('click', () => { scale = Math.max(MIN_SCALE, scale - .1); applyTransform(); });
document.querySelector('#fit').addEventListener('click', fitGraph);
viewport.addEventListener('click', event => {
  if (event.target.closest('.step-node')) return;
  focusedStep = '';
  document.querySelector('#canvasHint').textContent = 'All Routes visible';
  filterEdges();
});

let dragging = false;
let dragStart = null;
viewport.addEventListener('pointerdown', event => {
  if (event.target.closest('.step-node')) return;
  dragging = true;
  dragStart = { x: event.clientX - translate.x, y: event.clientY - translate.y };
  viewport.classList.add('dragging');
  viewport.setPointerCapture(event.pointerId);
});
viewport.addEventListener('pointermove', event => {
  if (!dragging) return;
  translate = { x: event.clientX - dragStart.x, y: event.clientY - dragStart.y };
  applyTransform();
});
function stopDragging(event) {
  if (!dragging) return;
  dragging = false;
  dragStart = null;
  viewport.classList.remove('dragging');
  if (viewport.hasPointerCapture(event.pointerId)) viewport.releasePointerCapture(event.pointerId);
}
viewport.addEventListener('pointerup', stopDragging);
viewport.addEventListener('pointercancel', stopDragging);
viewport.addEventListener('wheel', event => {
  event.preventDefault();
  scale = Math.max(MIN_SCALE, Math.min(1.45, scale + (event.deltaY < 0 ? .08 : -.08)));
  applyTransform();
}, { passive: false });

setInterval(async () => {
  if (selected) {
    try { await load(selected, false, selectedFragment); } catch (error) { showError(error); }
  }
  if (selectedWorkflow) {
    try { await loadTopology(selectedWorkflow, false); } catch (error) { showError(error); }
  }
}, 2000);
window.addEventListener('resize', fitGraph);
loadFiles().catch(showError);
