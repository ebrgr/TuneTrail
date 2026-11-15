/* Task dialogs reuse server-rendered pages and retain normal links without JavaScript. */
(() => {
  const dialog = document.getElementById('media-task-dialog');
  if (!dialog || !dialog.showModal || !window.TuneTrailEnhance) return;
  const content = dialog.querySelector('[data-task-content]');
  const status = dialog.querySelector('[data-task-status]');
  const title = dialog.querySelector('#media-task-title');
  let activation = null;
  let opener = null;
  let taskURL = '';
  let ready = true;
  let validation = null;
  const base = document.querySelector('[data-media-task]')?.pathname.replace(/\/[^/]+$/, '');
  const plainClick = event => event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;

  // Closing or navigating away invalidates every outstanding response, including body reads.
  function clear() {
    activation?.abort();
    activation = null;
    content.replaceChildren();
    content.inert = false;
    status.textContent = '';
    dialog.removeAttribute('aria-busy');
  }
  function failure() {
    content.replaceChildren();
    status.textContent = dialog.dataset.error;
    const link = document.createElement('a');
    link.href = taskURL;
    link.dataset.taskPage = '';
    link.textContent = dialog.dataset.openPage;
    content.append(link);
  }
  function allowed(url) {
    return url.origin === location.origin && url.pathname.startsWith(base + '/');
  }
  function focusTitle() { title.focus({preventScroll: true}); }

  // Refresh the public header from its owning renderer, including artwork srcset and statistics.
  async function refresh(signal) {
    const response = await fetch(location.href, {signal, cache: 'no-store', credentials: 'same-origin'});
    if (!response.ok || response.redirected) throw Error('Header unavailable');
    const next = new DOMParser().parseFromString(await response.text(), 'text/html');
    if (signal.aborted) return;
    const current = document.querySelector('.media-heading');
    const heading = next.querySelector('.media-heading');
    if (!next.body.hasAttribute('data-admin') || !heading || heading.dataset.entity !== current.dataset.entity ||
        heading.dataset.entityId !== current.dataset.entityId) throw Error('Context changed');
    current.replaceWith(heading);
    document.title = next.title;
  }

  // A dialog request has a bounded lifetime and validates the fragment before inserting any form.
  async function request(url, init = {}, mutation = false) {
    if (!activation || activation.signal.aborted) return;
    const owner = activation;
    const controller = new AbortController();
    const abort = () => controller.abort();
    owner.signal.addEventListener('abort', abort, {once: true});
    const timeout = setTimeout(abort, init.body instanceof FormData ? 60000 : 15000);
    dialog.setAttribute('aria-busy', 'true');
    status.textContent = dialog.dataset.loading;
    content.inert = true;
    try {
      const response = await fetch(url, {...init, signal: controller.signal, cache: 'no-store', credentials: 'same-origin', redirect: 'manual',
        headers: {...init.headers, 'X-TuneTrail-Partial': '1', 'X-TuneTrail-Dialog': '1'}});
      if (owner !== activation || controller.signal.aborted) return;
      const target = response.headers.get('X-TuneTrail-Navigate');
      if (response.status === 204 && target && mutation) {
        const destination = new URL(target, location.href);
        const profile = '/' + document.body.dataset.profileSlug;
        if (destination.origin !== location.origin ||
            destination.pathname !== profile && !destination.pathname.startsWith(profile + '/')) throw Error('Invalid destination');
        location.assign(destination.href);
        return;
      }
      const fragment = response.headers.get('X-TuneTrail-Fragment');
      if (!['task', 'review'].includes(fragment)) throw Error('Invalid fragment');
      const html = await response.text();
      if (owner !== activation || controller.signal.aborted) return;
      const template = document.createElement('template');
      template.innerHTML = html.trim();
      const next = template.content.firstElementChild;
      if (template.content.childElementCount !== 1 || !next || next.id !== fragment || !next.matches('[data-section]')) throw Error('Invalid structure');
      content.replaceChildren(next);
      status.textContent = next.querySelector('[data-section-message]')?.textContent.trim() || '';
      focusTitle();
      if (mutation && response.ok) {
        try { await refresh(controller.signal); }
        catch { if (owner === activation && !owner.signal.aborted) status.textContent = dialog.dataset.refreshError; }
      }
    } catch {
      // Never replay a mutation after a timeout; reopening obtains authoritative state and fresh tokens.
      if (owner === activation && !owner.signal.aborted) { failure(); focusTitle(); }
    } finally {
      clearTimeout(timeout);
      owner.signal.removeEventListener('abort', abort);
      if (owner === activation) { content.inert = false; dialog.removeAttribute('aria-busy'); }
    }
  }
  function open(link) {
    clear();
    opener = link.href;
    taskURL = link.href;
    title.textContent = link.title;
    activation = new AbortController();
    dialog.showModal();
    focusTitle();
    const url = new URL(taskURL);
    url.searchParams.set('partial', '1');
    request(url);
  }
  document.addEventListener('click', event => {
    const link = event.target.closest('a[data-media-task]');
    if (!link || !plainClick(event)) return;
    if (!ready) { event.preventDefault(); revalidate(); return; }
    event.preventDefault();
    open(link);
  });
  // Keep Tab within task controls, including the wrap through browser chrome in Chromium.
  dialog.addEventListener('keydown', event => {
    if (event.key !== 'Tab') return;
    const controls = [...dialog.querySelectorAll('a[href],button,input:not([type=hidden]),select,textarea,[tabindex="0"]')]
      .filter(element => !element.disabled && !element.closest('[inert]') && element.getClientRects().length);
    const first = controls[0];
    const last = controls.at(-1);
    if (!first) { event.preventDefault(); focusTitle(); return; }
    if (!controls.includes(document.activeElement) || event.shiftKey && document.activeElement === first ||
        !event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      (event.shiftKey ? last : first).focus({preventScroll: true});
    }
  });
  dialog.querySelector('[data-task-close]').addEventListener('click', () => dialog.close());
  dialog.addEventListener('close', () => {
    if (dialog.open) return; // A queued close event must not clear a newly opened task.
    clear();
    const link = [...document.querySelectorAll('[data-media-task]')].find(link => link.href === opener);
    link?.focus({preventScroll: true});
  });
  dialog.addEventListener('click', event => {
    const link = event.target.closest('a[href]');
    if (!link || link.hasAttribute('data-task-page') || link.hasAttribute('download') ||
        link.target && link.target !== '_self' || !plainClick(event) || !allowed(new URL(link.href))) return;
    event.preventDefault();
    if (dialog.hasAttribute('aria-busy')) return;
    const url = new URL(link.href);
    url.searchParams.set('partial', '1');
    request(url);
  });
  window.TuneTrailEnhance.install(dialog, {submit(form, submitter) {
    if (!activation || dialog.hasAttribute('aria-busy')) return;
    const url = new URL(form.action);
    if (!allowed(url)) { failure(); focusTitle(); return; }
    const data = new FormData(form);
    if (submitter?.name) data.set(submitter.name, submitter.value);
    const init = {method: form.method.toUpperCase(), headers: {}};
    if (init.method === 'GET') {
      url.search = new URLSearchParams(data).toString();
      url.searchParams.set('partial', '1');
      init.headers['X-TuneTrail-Submission'] = content.querySelector('input[name=submission_token]')?.value || '';
    } else {
      init.body = form.enctype === 'multipart/form-data' ? data : new URLSearchParams(data);
    }
    request(url, init, init.method === 'POST');
  }});

  // A suspended public page may remain visible, but never restores a sensitive dialog or its authority.
  async function revalidate() {
    validation?.abort();
    const controller = new AbortController();
    validation = controller;
    const timer = setTimeout(() => controller.abort(), 10000);
    ready = false;
    document.querySelectorAll('[data-media-task]').forEach(link => { link.hidden = true; });
    try {
      const response = await fetch(location.href, {method: 'HEAD', signal: controller.signal, cache: 'no-store', credentials: 'same-origin'});
      if (validation !== controller || controller.signal.aborted) return;
      if (!response.ok || response.redirected || response.headers.get('X-TuneTrail-Admin') !== 'true' ||
          response.headers.get('X-TuneTrail-UI-Generation') !== document.body.dataset.uiGeneration) { location.reload(); return; }
      ready = true;
      document.querySelectorAll('[data-media-task]').forEach(link => { link.hidden = false; });
    } catch { /* Keep management hidden until another successful revalidation or navigation. */ }
    finally { clearTimeout(timer); }
  }
  addEventListener('pagehide', () => {
    ready = false;
    validation?.abort();
    if (dialog.open) dialog.close();
    clear();
    document.querySelectorAll('[data-media-task]').forEach(link => { link.hidden = true; });
  });
  addEventListener('pageshow', event => { if (event.persisted) revalidate(); });
  if (performance.getEntriesByType('navigation')[0]?.type === 'back_forward') revalidate();
})();
