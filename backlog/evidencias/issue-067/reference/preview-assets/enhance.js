/* Media management without full reloads (ISSUE-124). Forms marked data-enhance are sent
   in the background and only their section is replaced. Without this script every form
   posts normally and the server returns to the edited section's anchor. */
(() => {
  const status = document.querySelector('[data-enhance-status]');
  if (!document.querySelector('form[data-enhance]')) return;
  let busy = false;
  // After an error page reached by a full POST (e.g. artwork), reloading would resubmit:
  // go back to the management page instead.
  const base = document.querySelector('[data-manage-base]')?.dataset.manageBase;
  const reload = () => { if (!base || location.pathname === base) location.reload(); else location.assign(base); };
  const ticket = () => document.querySelector('input[name=submission_token]')?.value || '';
  const setTicket = value => {
    if (value) document.querySelectorAll('input[name=submission_token]').forEach(input => { input.value = value; });
  };
  // Replace the section by the server's one, only if it is the expected section.
  const swap = (section, html) => {
    const template = document.createElement('template');
    template.innerHTML = html.trim();
    const next = template.content.firstElementChild;
    if (!next || next.id !== section.id) return null;
    section.replaceWith(next);
    return next;
  };
  document.addEventListener('submit', async event => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement) || !form.matches('[data-enhance]')) return;
    const section = form.closest('[data-section]');
    if (!section) return;
    event.preventDefault();
    if (busy) return; // one request at a time: the page shares a single-use ticket
    busy = true;
    const focusName = document.activeElement?.closest('form') === form ? document.activeElement.name : '';
    section.setAttribute('aria-busy', 'true');
    form.querySelectorAll('button').forEach(button => { button.disabled = true; });
    try {
      const data = new FormData(form);
      if (event.submitter?.name) data.set(event.submitter.name, event.submitter.value);
      const url = new URL(form.action, location.href);
      const init = {method: form.method.toUpperCase(), credentials: 'same-origin', redirect: 'manual',
        headers: {'X-TuneTrail-Partial': '1'}};
      if (init.method === 'GET') {
        url.search = new URLSearchParams(data).toString();
        url.searchParams.set('partial', '1');
        init.headers['X-TuneTrail-Submission'] = ticket();
      } else init.body = new URLSearchParams(data);
      const response = await fetch(url, init);
      // Sign-in, CSRF, a spent ticket or any other unexpected answer: reload, never resend.
      if (response.headers.get('X-TuneTrail-Fragment') !== section.id) { reload(); return; }
      const next = swap(section, await response.text());
      if (!next) { reload(); return; }
      setTicket(response.headers.get('X-TuneTrail-Submission'));
      // A new primary alias renames the media: keep the heading and back link in step.
      if (next.dataset.heading) document.querySelector('h1').textContent = next.dataset.heading;
      const back = document.querySelector('[data-media-back]');
      if (back && next.dataset.back) back.textContent = next.dataset.back;
      if (init.method === 'GET') {
        url.searchParams.delete('partial');
        history.replaceState(null, '', url);
        // Other forms carry the current artist query as a hidden value; keep them in step.
        const query = url.searchParams.get('artist_query');
        if (query !== null) document.querySelectorAll('input[type=hidden][name=artist_query]').forEach(input => { input.value = query; });
      }
      const message = next.querySelector('[data-section-message]')?.textContent.trim() || '';
      if (status) { status.textContent = ''; requestAnimationFrame(() => { status.textContent = message; }); }
      // Return focus to the same field, or to the section heading.
      const target = (focusName && next.querySelector(`[name="${CSS.escape(focusName)}"]:not([type=hidden])`)) || next.querySelector('h2');
      if (target) {
        if (!target.matches('input,select,textarea,button')) target.tabIndex = -1;
        target.focus({preventScroll: true});
      }
    } catch {
      reload();
    } finally {
      busy = false;
      section.removeAttribute('aria-busy');
    }
  });
})();
