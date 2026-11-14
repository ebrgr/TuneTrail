(() => {
  const body = document.body;
  const query = new URLSearchParams(location.search);
  const cookies = () => Object.fromEntries(document.cookie.split(';').map(part => part.trim().split('=').map(decodeURIComponent)));
  // Establish the browser's timezone and week layout once, before subsequent reads.
  let changed = false;
  try {
    const stored = cookies();
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (!stored.tz && tz) {
      document.cookie = `tz=${encodeURIComponent(tz)}; Path=/; Max-Age=31536000; SameSite=Lax`;
      changed = cookies().tz === tz && !query.has('tz') && body.dataset.forceTimezone !== 'true';
    }
    let first = 0;
    try { first = new Intl.Locale(navigator.language).getWeekInfo().firstDay % 7; } catch {}
    // Renamed from koito-weekstart in ISSUE-111 (no fallback, user decision).
    if (stored['tunetrail-weekstart'] !== String(first)) {
      document.cookie = `tunetrail-weekstart=${first}; Path=/; Max-Age=31536000; SameSite=Lax`;
      changed ||= cookies()['tunetrail-weekstart'] === String(first);
    }
    if (body.dataset.section === 'home' && !query.has('period')) {
      const cached = localStorage.getItem(`period_selection_${body.dataset.profileId}:home`);
      if (['day','week','month','year','all_time'].includes(cached)) {query.set('period',cached);location.replace(location.pathname+'?'+query);return;}
    }
  } catch {}
  if (changed) {location.reload();return;}
  document.querySelectorAll('.period-selector a').forEach(link => link.addEventListener('click', () => {
    if (body.dataset.section !== 'home') return;
    try {localStorage.setItem(`period_selection_${body.dataset.profileId}:home`,new URL(link.href).searchParams.get('period'));} catch {}
  }));
  // Apply the selected display option using the existing GET form.
  const rewindForm = document.querySelector('[data-rewind-time-form]');
  rewindForm?.querySelector('input[name=show_time]')?.addEventListener('change', () => rewindForm.requestSubmit());
  // Each activation owns its requests; suspended documents must not retain authority.
  const slug = body.dataset.profileSlug;
  let active = null;
  const removeControls = () => document.querySelectorAll('[data-listen-controls]').forEach(cell => cell.remove());

  // Bound request lifetimes, including body reads and sibling fetches, to one activation.
  async function requestScope(parent, operation) {
    const request = new AbortController();
    const abort = () => request.abort();
    if (parent.signal.aborted) request.abort();
    parent.signal.addEventListener('abort', abort, {once: true});
    const timeout = setTimeout(abort, 10000);
    try { return await operation(request.signal); }
    finally {
      request.abort(); // Cancel sibling work when an activation batch rejects early.
      clearTimeout(timeout);
      parent.signal.removeEventListener('abort', abort);
    }
  }

  // Install one mutation cell per row only after the active session is confirmed.
  function addControls() {
    document.querySelectorAll('tr[data-track-id][data-unix]').forEach(row => {
      const cell = document.createElement('td');
      cell.dataset.listenControls = '';
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = '×';
      button.className = 'delete-listen';
      button.setAttribute('aria-label', body.dataset.i18nDeleteLabel);
      cell.append(button);
      row.append(cell);
    });
  }

  // Restore authority and compare server-resolved context without trusting preference cookies.
  async function activate(restored) {
    active?.abort();
    removeControls();
    const controller = new AbortController();
    active = controller;
    window.TuneTrailTheme?.refresh();
    try {
      await requestScope(controller, async signal => {
        const options = {signal, cache: 'no-store', credentials: 'same-origin'};
        const account = slug ? fetch('/apis/web/v1/user', options).then(async response =>
          response.ok ? response.json() : null) : Promise.resolve(null);
        const context = restored ? fetch(location.href, {...options, method: 'HEAD'}) : Promise.resolve(null);
        const [user, head] = await Promise.all([account, context]);
        if (signal.aborted || active !== controller) return;
        if (head) {
          const language = head.headers.get('Content-Language');
          const generation = head.headers.get('X-TuneTrail-UI-Generation');
          const timezone = head.headers.get('X-TuneTrail-Timezone');
          const week = head.headers.get('X-TuneTrail-Week-Start');
          if (head.redirected || head.status === 404 || head.status === 410 ||
              (language && language !== document.documentElement.lang) ||
              (generation && generation !== body.dataset.uiGeneration) ||
              (timezone && body.dataset.timezone && timezone !== body.dataset.timezone) ||
              (week && body.dataset.weekStart && week !== body.dataset.weekStart)) {
            location.reload(); return;
          }
          if (!head.ok) return;
        }
        if (user?.role === 'admin') addControls();
      });
    } catch {
      // Keep the public snapshot available, with no mutation controls, on failure.
      if (active === controller) removeControls();
    }
  }

  addEventListener('pagehide', () => {
    active?.abort();
    active = null;
    removeControls();
  });
  addEventListener('pageshow', event => { if (event.persisted) activate(true); });

  // One delegated listener reads the current activation instead of a suspended controller.
  document.addEventListener('click', async event => {
    const button = event.target.closest('.delete-listen');
    if (!button || !active || active.signal.aborted || button.disabled) return;
    if (!confirm(body.dataset.i18nDeleteConfirm)) return;
    const controller = active;
    const row = button.closest('tr[data-track-id][data-unix]');
    if (!row) return;
    button.disabled = true;
    // A pending reload keeps the button disabled until the document unloads.
    let reloading = false;
    const target = new URL(`/apis/web/v1/profiles/${encodeURIComponent(slug)}/listens`, location.origin);
    target.searchParams.set('track_id', row.dataset.trackId);
    target.searchParams.set('unix', row.dataset.unix);
    try {
      await requestScope(controller, async signal => {
        const response = await fetch(target, {method: 'DELETE', signal, cache: 'no-store', credentials: 'same-origin'});
        if (!response.ok) throw Error(body.dataset.i18nDeleteError);
        if (active === controller && !signal.aborted) { reloading = true; location.reload(); }
      });
    } catch {
      if (active !== controller) return;
      // An active timeout has an uncertain outcome; show status but never replay the DELETE.
      let message = document.getElementById('listen-status');
      if (!message) {
        message = document.createElement('p');
        message.id = 'listen-status';
        message.setAttribute('role', 'status');
        row.closest('table').after(message);
      }
      message.textContent = body.dataset.i18nDeleteError;
    } finally {
      if (!reloading && button.isConnected && active === controller) button.disabled = false;
    }
  });
  activate(false);
})();
