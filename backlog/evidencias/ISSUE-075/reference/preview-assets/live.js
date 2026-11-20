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
  // Delegation survives a history snapshot refresh after deleting a listen.
  document.addEventListener('click', event => {
    const link = event.target.closest('.period-selector a');
    if (!link || body.dataset.section !== 'home') return;
    try {localStorage.setItem(`period_selection_${body.dataset.profileId}:home`,new URL(link.href).searchParams.get('period'));} catch {}
  });
  // Apply the selected display option using the existing GET form.
  const rewindForm = document.querySelector('[data-rewind-time-form]');
  rewindForm?.querySelector('input[name=show_time]')?.addEventListener('change', () => rewindForm.requestSubmit());
  // Each activation owns its requests; suspended documents must not retain authority.
  let active = null;

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

  // The server marks administrator sessions with data-admin (ISSUE-109); no role fetch is needed.
  const admin = 'admin' in body.dataset;
  const signedIn = 'signedIn' in body.dataset;
  const manageListening = 'manageListening' in body.dataset;

  // Restore authority and compare server-resolved context without trusting preference cookies.
  async function activate(restored) {
    active?.abort();
    const controller = new AbortController();
    active = controller;
    window.TuneTrailTheme?.refresh();
    if (!restored) return;
    try {
      await requestScope(controller, async signal => {
        const head = await fetch(location.href, {signal, cache: 'no-store', credentials: 'same-origin', method: 'HEAD'});
        if (signal.aborted || active !== controller) return;
        const language = head.headers.get('Content-Language');
        const generation = head.headers.get('X-TuneTrail-UI-Generation');
        const timezone = head.headers.get('X-TuneTrail-Timezone');
        const week = head.headers.get('X-TuneTrail-Week-Start');
        const session = head.headers.get('X-TuneTrail-Admin');
        // A login, logout or expiry while suspended reloads instead of showing a stale badge.
        if (head.redirected || head.status === 404 || head.status === 410 ||
            (language && language !== document.documentElement.lang) ||
            (generation && generation !== body.dataset.uiGeneration) ||
            (timezone && body.dataset.timezone && timezone !== body.dataset.timezone) ||
            (week && body.dataset.weekStart && week !== body.dataset.weekStart) ||
            (session && session !== String(admin)) ||
            head.headers.get('X-TuneTrail-Signed-In') !== String(signedIn) ||
            head.headers.get('X-TuneTrail-Manage-Listening') !== String(manageListening)) {
          location.reload(); return;
        }
      });
    } catch {
      // The dialog controller owns sensitive controls and its independent session revalidation.
    }
  }

  addEventListener('pagehide', () => {
    active?.abort();
    active = null;
  });
  addEventListener('pageshow', event => { if (event.persisted) activate(true); });

  // A history navigation may rebuild the page from the HTTP cache without the bfcache;
  // revalidate it like a restore so a stale badge or controls never survive a logout.
  activate(performance.getEntriesByType('navigation')[0]?.type === 'back_forward');
})();
