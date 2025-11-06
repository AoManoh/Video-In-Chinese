/*
 * Frontend↔Backend smoke runbook for browser console
 * Usage:
 * 1) Ensure backend services are running (Gateway on 8081) and frontend dev server is started with VITE_USE_MOCK=false
 * 2) Open http://localhost:5173 (Upload page recommended). You may pre-select a file in the UI; otherwise this script will prompt a file picker.
 * 3) Paste this whole script into the browser console and press Enter.
 */
(async () => {
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const now = () => new Date().toISOString();

  const fetchJSON = async (url, init = {}) => {
    const headers = { Accept: 'application/json', ...(init.headers || {}) };
    const res = await fetch(url, { ...init, headers });
    const text = await res.text();
    let data = undefined;
    try { data = text ? JSON.parse(text) : undefined; } catch {}
    return { ok: res.ok, status: res.status, data, text, res };
  };

  const ensureFile = async () => {
    const existing = document.querySelector('input[type="file"]');
    if (existing && existing.files && existing.files[0]) return existing.files[0];
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'video/mp4,video/quicktime,video/x-matroska,video/x-msvideo';
    input.style.display = 'none';
    document.body.appendChild(input);
    const file = await new Promise((resolve, reject) => {
      input.onchange = () => resolve(input.files[0]);
      input.click();
    });
    input.remove();
    if (!file) throw new Error('No file selected');
    return file;
  };

  const log = (...args) => console.log('[SMOKE]', ...args);
  const warn = (...args) => console.warn('[SMOKE]', ...args);
  const err = (...args) => console.error('[SMOKE]', ...args);

  console.group('[SMOKE] Runbook started', now());

  // 1) Settings connectivity
  log('Step 1: GET /v1/settings');
  let r = await fetchJSON('/v1/settings');
  if (!r.ok) throw new Error(`GET /v1/settings failed: ${r.status} ${r.text}`);
  log('Settings OK:', r.data);

  // 2) Pick file
  log('Step 2: Select a video file (UI input or file picker)');
  const file = await ensureFile();
  log('Selected file:', { name: file.name, size: file.size, type: file.type });

  // 3) Upload
  log('Step 3: POST /v1/tasks/upload');
  const fd = new FormData();
  fd.append('file', file);
  r = await fetchJSON('/v1/tasks/upload', { method: 'POST', body: fd });
  if (!r.ok) throw new Error(`POST /v1/tasks/upload failed: ${r.status} ${r.text}`);
  const taskId = r.data?.task_id;
  if (!taskId) throw new Error('No task_id in upload response');
  log('Upload OK, task_id=', taskId);

  // 4) Poll status
  log('Step 4: Poll GET /v1/tasks/{id}/status until COMPLETED/FAILED');
  const INTERVAL = 3000; // 3s
  const TIMEOUT = 15 * 60 * 1000; // 15min
  const startTs = Date.now();
  let status = 'PENDING';
  let resultUrl = undefined;
  let errorMessage = undefined;
  while (true) {
    const pr = await fetchJSON(`/v1/tasks/${taskId}/status`);
    if (!pr.ok) warn('Poll error:', pr.status, pr.text);
    status = pr.data?.status || status;
    resultUrl = pr.data?.result_url || resultUrl;
    errorMessage = pr.data?.error_message || errorMessage;
    log('Status:', status, pr.data);
    if (status === 'COMPLETED' || status === 'FAILED') break;
    if (Date.now() - startTs > TIMEOUT) throw new Error('Polling timeout');
    await sleep(INTERVAL);
  }
  if (status === 'FAILED') throw new Error('Task FAILED: ' + (errorMessage || ''));

  // 5) Download result
  log('Step 5: Download result');
  let fileName = 'result.mp4';
  if (resultUrl) {
    try {
      const urlObj = new URL(resultUrl, location.origin);
      const parts = urlObj.pathname.split('/');
      fileName = parts[parts.length - 1] || fileName;
    } catch {}
  }
  const dlUrl = resultUrl || `/v1/tasks/download/${taskId}/${encodeURIComponent(fileName)}`;
  const dlRes = await fetch(dlUrl);
  if (!dlRes.ok) throw new Error(`Download failed: ${dlRes.status}`);
  const blob = await dlRes.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  setTimeout(() => { URL.revokeObjectURL(a.href); a.remove(); }, 5000);
  log('Download OK:', { fileName, size: blob.size, dlUrl });

  console.groupEnd();
  console.info('[SMOKE] Completed at', now());
})().catch((e) => {
  console.groupEnd?.();
  console.error('[SMOKE] FAILED', e?.message || e, e);
});

