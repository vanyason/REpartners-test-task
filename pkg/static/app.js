// State
let knownPacks = [];

// Color palette for pack sizes — deterministic hue from value
const COLORS = [
  '#6366f1', '#8b5cf6', '#ec4899', '#f43f5e',
  '#f97316', '#eab308', '#22c55e', '#14b8a6',
  '#06b6d4', '#3b82f6',
];

function packColor(size, packs) {
  const sorted = [...packs].sort((a, b) => a - b);
  const idx = sorted.indexOf(size);
  if (idx === -1) return COLORS[0];
  return COLORS[idx % COLORS.length];
}

// Box dimensions proportional to pack size
function boxSize(packSize, allPacks) {
  const maxPack = Math.max(...allPacks);
  const minDim = 22;
  const maxDim = 48;
  const ratio = Math.sqrt(packSize / maxPack); // sqrt for gentler scaling
  const dim = Math.round(minDim + ratio * (maxDim - minDim));
  return dim;
}

// --- Pack Badges ---
function renderBadges(packs) {
  const el = document.getElementById('packBadges');
  if (!packs.length) {
    el.innerHTML = '<span class="empty-state">No pack sizes configured</span>';
    return;
  }
  el.innerHTML = packs
    .slice()
    .sort((a, b) => a - b)
    .map(p => `<span class="pack-badge" style="background:${packColor(p, packs)}">${p.toLocaleString()}</span>`)
    .join('');
}

// --- Pack Inputs ---
function renderInputs(packs) {
  const el = document.getElementById('packInputs');
  el.innerHTML = '';
  const sorted = [...packs].sort((a, b) => a - b);
  sorted.forEach(p => addPackInput(p));
}

function addPackInput(value) {
  const el = document.getElementById('packInputs');
  const group = document.createElement('div');
  group.className = 'pack-input-group';
  const input = document.createElement('input');
  input.type = 'number';
  input.min = '1';
  input.placeholder = 'Size';
  if (value !== undefined) input.value = value;
  const btn = document.createElement('button');
  btn.className = 'btn-remove';
  btn.innerHTML = '&times;';
  btn.title = 'Remove';
  btn.onclick = () => group.remove();
  group.appendChild(input);
  group.appendChild(btn);
  el.appendChild(group);
  if (value === undefined) input.focus();
}

function getInputPacks() {
  const inputs = document.querySelectorAll('#packInputs input');
  const packs = [];
  for (const inp of inputs) {
    const v = parseInt(inp.value, 10);
    if (v > 0) packs.push(v);
  }
  return packs;
}

// --- API calls ---
async function fetchPacks() {
  const res = await fetch('/api/v1/packs');
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Failed to load packs');
  return data.packs;
}

async function refreshPacks() {
  try {
    knownPacks = await fetchPacks();
    renderBadges(knownPacks);
    renderInputs(knownPacks);
    hideWarning();
  } catch (e) {
    showError('packError', e.message);
  }
}

async function savePacks() {
  const packs = getInputPacks();
  hideError('packError');
  try {
    const res = await fetch('/api/v1/packs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ packs }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Failed to save');
    knownPacks = data.packs;
    renderBadges(knownPacks);
    renderInputs(knownPacks);
    hideWarning();
    // Clear previous results since packs changed
    document.getElementById('resultSection').classList.remove('visible');
  } catch (e) {
    showError('packError', e.message);
  }
}

const CALC_TIMEOUT_MS = 7000;

function setCalcLoading(loading) {
  const overlay = document.getElementById('spinnerOverlay');
  const calcCard = document.getElementById('calcBtn').closest('.card');
  if (loading) {
    overlay.classList.add('visible');
    calcCard.classList.add('ui-locked');
  } else {
    overlay.classList.remove('visible');
    calcCard.classList.remove('ui-locked');
  }
}

async function calculate() {
  const items = parseInt(document.getElementById('itemsInput').value, 10);
  hideError('calcError');
  if (isNaN(items) || items < 0) {
    showError('calcError', 'Please enter a valid non-negative number');
    return;
  }

  setCalcLoading(true);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), CALC_TIMEOUT_MS);

  try {
    const res = await fetch('/api/v1/calculate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ items }),
      signal: controller.signal,
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Calculation failed');

    // Check concurrent change
    const usedSorted = [...data.pack_sizes_used].sort((a, b) => a - b).join(',');
    const knownSorted = [...knownPacks].sort((a, b) => a - b).join(',');
    if (usedSorted !== knownSorted) {
      document.getElementById('warningBanner').classList.add('visible');
    }

    renderResult(items, data.packs, data.pack_sizes_used);
  } catch (e) {
    const msg = e.name === 'AbortError'
      ? 'Calculation timed out. Please try a smaller order or check the server.'
      : e.message;
    showError('calcError', msg);
    document.getElementById('resultSection').classList.remove('visible');
  } finally {
    clearTimeout(timeout);
    setCalcLoading(false);
  }
}

// --- Result rendering ---
function renderResult(ordered, packs, packSizes) {
  const section = document.getElementById('resultSection');

  // Compute totals
  let totalItems = 0;
  let totalPacks = 0;
  const entries = [];
  for (const [sizeStr, qty] of Object.entries(packs)) {
    const size = parseInt(sizeStr, 10);
    totalItems += size * qty;
    totalPacks += qty;
    entries.push({ size, qty });
  }
  entries.sort((a, b) => b.size - a.size);

  const surplus = totalItems - ordered;

  // Summary
  const summaryEl = document.getElementById('resultSummary');
  summaryEl.innerHTML = `
    <div class="stat">
      <span class="stat-label">Ordered</span>
      <span class="stat-value">${ordered.toLocaleString()}</span>
    </div>
    <div class="stat">
      <span class="stat-label">Packed</span>
      <span class="stat-value">${totalItems.toLocaleString()}</span>
    </div>
    <div class="stat">
      <span class="stat-label">Surplus</span>
      <span class="stat-value ${surplus > 0 ? 'surplus' : ''}">${surplus > 0 ? '+' : ''}${surplus.toLocaleString()}</span>
    </div>
    <div class="stat">
      <span class="stat-label">Packs</span>
      <span class="stat-value">${totalPacks.toLocaleString()}</span>
    </div>
  `;

  // Visualization
  const visEl = document.getElementById('packVis');
  if (!entries.length) {
    visEl.innerHTML = '<span class="empty-state">No packs needed (0 items)</span>';
    section.classList.add('visible');
    return;
  }

  const allSizes = entries.map(e => e.size);
  visEl.innerHTML = entries.map(({ size, qty }) => {
    const dim = boxSize(size, allSizes);
    const color = packColor(size, packSizes);
    const maxShow = 30; // max boxes to render per row
    const showQty = Math.min(qty, maxShow);
    let boxes = '';
    for (let i = 0; i < showQty; i++) {
      boxes += `<div class="pack-box" style="width:${dim}px;height:${dim}px;background:${color};animation-delay:${i * 0.03}s" title="${size} items"></div>`;
    }
    const overflow = qty > maxShow
      ? `<span class="pack-vis-qty-overflow">... +${(qty - maxShow).toLocaleString()} more</span>`
      : '';
    return `
      <div class="pack-vis-row">
        <div class="pack-vis-label" style="color:${color}">${size.toLocaleString()} &times; ${qty.toLocaleString()}</div>
        <div class="pack-vis-boxes">${boxes}${overflow}</div>
      </div>
    `;
  }).join('');

  section.classList.add('visible');
}

// --- Helpers ---
function showError(id, msg) {
  const el = document.getElementById(id);
  el.textContent = msg;
  el.classList.add('visible');
}

function hideError(id) {
  document.getElementById(id).classList.remove('visible');
}

function hideWarning() {
  document.getElementById('warningBanner').classList.remove('visible');
}

// Allow Enter key to trigger calculate
document.getElementById('itemsInput').addEventListener('keydown', e => {
  if (e.key === 'Enter') calculate();
});

// Init
refreshPacks();
