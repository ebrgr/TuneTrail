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
    if (stored['koito-weekstart'] !== String(first)) {
      document.cookie = `koito-weekstart=${first}; Path=/; Max-Age=31536000; SameSite=Lax`;
      changed ||= cookies()['koito-weekstart'] === String(first);
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
  // Admin availability is resolved using the existing session-only endpoint.
  const slug = body.dataset.profileSlug;
  if (!slug) return;
  const controller = new AbortController();
  addEventListener('pagehide', () => controller.abort(), {once:true});
  fetch('/apis/web/v1/user',{signal:controller.signal}).then(async response => {
    if (!response.ok) return;
    const account = await response.json();
    if (account.role !== 'admin') return;
    document.querySelectorAll('tr[data-track-id][data-unix]').forEach(row => {
      const cell = document.createElement('td');
      const button = document.createElement('button');
      button.type='button';button.textContent='×';button.className='delete-listen';button.setAttribute('aria-label',body.dataset.i18nDeleteLabel);
      button.addEventListener('click',async () => {
        if (!confirm(body.dataset.i18nDeleteConfirm)) return;
        button.disabled=true;
        const target = new URL(`/apis/web/v1/profiles/${encodeURIComponent(slug)}/listens`,location.origin);
        target.searchParams.set('track_id',row.dataset.trackId);target.searchParams.set('unix',row.dataset.unix);
        try {
          const result=await fetch(target,{method:'DELETE',signal:controller.signal});
          if (!result.ok) throw Error(body.dataset.i18nDeleteError);
          location.reload();
        } catch(error) {
          if (error.name==='AbortError') return;
          let message=document.getElementById('listen-status');
          if (!message) {message=document.createElement('p');message.id='listen-status';message.setAttribute('role','status');row.closest('table').after(message);}
          message.textContent=body.dataset.i18nDeleteError;button.disabled=false;
        }
      });cell.append(button);row.append(cell);
    });
  }).catch(() => {});
})();
