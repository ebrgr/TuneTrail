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
  let searchController = null;
  let searchTimer = null;
  let searchVersion = 0;
  const artworkMaxBytes = 10 * 1024 * 1024;
  const artworkTypes = new Set(['image/jpeg', 'image/png', 'image/webp']);
  let pendingArtworkFile = null;
  let artworkPreviewURL = '';
  let pendingArtworkError = '';
  // Debounced reads never consume or mint mutation tickets, and cannot outlive their task.
  function cancelSearch() {
    clearTimeout(searchTimer);
    searchController?.abort();
    searchController = null;
    searchVersion++;
    content.querySelectorAll('.manage-search-results').forEach(section => { section.inert = false; section.removeAttribute('aria-busy'); });
  }
  let base = '';
  let listenOpener = null;
  let returnFocus = null;
  // Release the local preview so cancelled or replaced artwork never pins a blob in memory.
  function clearArtworkPreview() {
    if (artworkPreviewURL) URL.revokeObjectURL(artworkPreviewURL);
    artworkPreviewURL = '';
    content.querySelector('[data-artwork-preview]')?.remove();
  }
  // Keep client-side drop feedback aligned with the server's accepted upload envelope.
  function artworkError(fileCount, file) {
    if (fileCount !== 1) return 'Drop exactly one image file.';
    if (!artworkTypes.has(file.type)) return 'Choose a JPEG, PNG or static WebP image.';
    if (file.size > artworkMaxBytes) return 'The image must be 10 MB or smaller.';
    return '';
  }
  // Put a dropped or keyboard-selected file into the existing multipart form without submitting it.
  function applyArtworkFile(file) {
    const form = content.querySelector('form.artwork-upload');
    const input = form?.querySelector('input[type=file][name=image]');
    if (!form || !input) return;
    if (!file) {
      pendingArtworkFile = null;
      input.value = '';
      status.textContent = pendingArtworkError;
      return;
    }
    const error = artworkError(1, file);
    pendingArtworkError = error;
    if (error) {
      pendingArtworkFile = null;
      input.value = '';
      status.textContent = error;
      return;
    }
    try {
      const transfer = new DataTransfer();
      transfer.items.add(file);
      input.files = transfer.files;
    } catch {
      pendingArtworkFile = null;
      status.textContent = 'This browser cannot prepare the dropped image. Use the file picker.';
      return;
    }
    pendingArtworkFile = file;
    pendingArtworkError = '';
    const urlInput = form.querySelector('input[name=image_url]');
    if (urlInput) {
      urlInput.setAttribute('value', '');
      urlInput.value = '';
    }
    clearArtworkPreview();
    artworkPreviewURL = URL.createObjectURL(file);
    const preview = document.createElement('img');
    preview.className = 'artwork-preview';
    preview.dataset.artworkPreview = '';
    preview.alt = 'Selected artwork preview';
    preview.src = artworkPreviewURL;
    form.append(preview);
    status.textContent = 'Preview ready. Confirm to replace the artwork.';
  }
  // Apply a queued drop after the server-rendered task fragment arrives.
  function installArtworkPreview() {
    if (pendingArtworkFile) applyArtworkFile(pendingArtworkFile);
    if (pendingArtworkError) status.textContent = pendingArtworkError;
  }
  // Enhance history inside its native disclosure, keeping the three-column mobile layout.
  function listenControls() {
    document.querySelectorAll('.listen-manage').forEach(link => {
      link.hidden = true;
      const row = link.closest('tr');
      if (row.querySelector('[data-listen-controls]')) return;
      const menu = link.closest('.listen-actions');
      if (!menu) return;
      const controls = document.createElement('span');
      controls.dataset.listenControls = '';
      const action = document.createElement('a');
      action.href = link.href;
      action.textContent = document.body.dataset.i18nDeleteLabel;
      action.className = 'delete-listen';
      action.dataset.mediaTask = '';
      action.dataset.listenTask = '';
      action.setAttribute('aria-label', document.body.dataset.i18nDeleteLabel);
      controls.append(action);
      menu.append(controls);
    });
  }
  listenControls();
  const plainClick = event => event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;

  // Closing or navigating away invalidates every outstanding response, including body reads.
  function clear() {
    cancelSearch();
    activation?.abort();
    activation = null;
    content.replaceChildren();
    content.inert = false;
    status.textContent = '';
    dialog.removeAttribute('aria-busy');
    clearArtworkPreview();
    pendingArtworkFile = null;
    pendingArtworkError = '';
  }
  function failure(destination = taskURL) {
    content.replaceChildren();
    status.textContent = dialog.dataset.error;
    const link = document.createElement('a');
    link.href = destination;
    link.dataset.taskPage = '';
    link.textContent = destination === taskURL ? dialog.dataset.openPage : dialog.dataset.reloadHistory;
    content.append(link);
  }
  function allowed(url) {
    return url.origin === location.origin && url.pathname.startsWith(base + '/');
  }
  function focusTitle() { dialog.scrollTop = 0; title.focus({preventScroll: true}); }

  // Refresh the public header from its owning renderer, including artwork srcset and statistics.
  async function refresh(signal) {
    const response = await fetch(location.href, {signal, cache: 'no-store', credentials: 'same-origin'});
    if (!response.ok || response.redirected) throw Error('Header unavailable');
    const next = new DOMParser().parseFromString(await response.text(), 'text/html');
    if (signal.aborted) return;
    const current = document.querySelector('.media-heading');
    const heading = next.querySelector('.media-heading');
    if (!next.body.hasAttribute('data-manage-listening') || !heading || heading.dataset.entity !== current.dataset.entity ||
        heading.dataset.entityId !== current.dataset.entityId) throw Error('Context changed');
    current.replaceWith(heading);
    document.title = next.title;
  }

  // Replace the public snapshot after a confirmed deletion; the dialog lives outside this main.
  async function refreshHistory(signal) {
    const response = await fetch(location.href, {signal, cache:'no-store', credentials:'same-origin',
      headers:{'X-TuneTrail-History-Refresh':'1'}});
    if (!response.ok || response.redirected) throw Error('History unavailable');
    const next = new DOMParser().parseFromString(await response.text(), 'text/html');
    if (signal.aborted) return;
    if (!next.body.hasAttribute('data-manage-listening') || next.body.dataset.profileSlug !== document.body.dataset.profileSlug ||
        !next.querySelector('main')) throw Error('Context changed');
    const current = document.querySelector('main');
    const scroll = {left:scrollX, top:scrollY};
    const rows = [...current.querySelectorAll('tr[data-track-id][data-unix]')];
    const index = rows.indexOf(listenOpener);
    current.replaceWith(next.querySelector('main'));
    listenControls();
    const controls = [...document.querySelectorAll('[data-listen-task]')];
    // The next row's action is inside a closed disclosure after replacement.
    returnFocus = controls[Math.min(index, controls.length - 1)]?.closest('details')?.querySelector('summary') ||
      document.querySelector('main h1,main h3,main');
    if (returnFocus && returnFocus.tabIndex < 0) returnFocus.tabIndex = -1;
    scrollTo(scroll);
  }

  // A dialog request has a bounded lifetime and validates the fragment before inserting any form.
  async function request(url, init = {}, mutation = false) {
    if (!activation || activation.signal.aborted) return;
    cancelSearch();
    const owner = activation;
    const controller = new AbortController();
    const abort = () => controller.abort();
    owner.signal.addEventListener('abort', abort, {once: true});
    const timeout = setTimeout(abort, init.body instanceof FormData ? 60000 : 15000);
    dialog.setAttribute('aria-busy', 'true');
    status.textContent = dialog.dataset.loading;
    if (content.contains(document.activeElement)) focusTitle();
    content.inert = true;
    try {
      const response = await fetch(url, {...init, signal: controller.signal, cache: 'no-store', credentials: 'same-origin', redirect: 'manual',
        headers: {...init.headers, 'X-TuneTrail-Partial': '1', 'X-TuneTrail-Dialog': '1'}});
      if (owner !== activation || controller.signal.aborted) return;
      if (response.status === 204 && mutation && listenOpener &&
          response.headers.get('X-TuneTrail-Listen-Deleted') === listenOpener.dataset.unix) {
        // If refresh fails the deletion remains acknowledged; never offer its consumed form again.
        content.replaceChildren();
        try {
          await refreshHistory(controller.signal);
          if (owner === activation && !controller.signal.aborted) dialog.close();
        } catch {
          if (owner === activation && !owner.signal.aborted) {
            listenOpener.remove();
            failure(location.href);
            status.textContent = dialog.dataset.refreshError;
            focusTitle();
          }
        }
        return;
      }
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
      installArtworkPreview();
      focusTitle();
      if (mutation && response.ok) {
        try { await refresh(controller.signal); }
        catch { if (owner === activation && !owner.signal.aborted) status.textContent = dialog.dataset.refreshError; }
        clearArtworkPreview();
        pendingArtworkFile = null;
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
  // Open the ordinary image task and optionally carry an unsubmitted local drop into it.
  function open(link, file = null, error = '') {
    clear();
    pendingArtworkFile = file;
    pendingArtworkError = error;
    base = link.pathname.replace(/\/[^/]+$/, '');
    listenOpener = link.hasAttribute('data-listen-task') ? link.closest('tr') : null;
    returnFocus = null;
    opener = link.href;
    taskURL = link.href;
    dialog.dataset.task = new URL(taskURL).pathname.split('/').at(-1);
    title.textContent = link.dataset.tip || link.title || link.getAttribute('aria-label');
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
  // A public media page exposes this handler only for an authenticated administrator.
  document.addEventListener('dragover', event => {
    const cover = event.target.closest?.('img[data-artwork-drop]');
    if (!cover) return;
    event.preventDefault();
    cover.dataset.dropActive = '1';
  });
  document.addEventListener('dragleave', event => {
    const cover = event.target.closest?.('img[data-artwork-drop]');
    if (cover && !cover.contains(event.relatedTarget)) delete cover.dataset.dropActive;
  });
  document.addEventListener('drop', event => {
    const cover = event.target.closest?.('img[data-artwork-drop]');
    if (!cover) return;
    event.preventDefault();
    delete cover.dataset.dropActive;
    // The image task link lives in the actions nav, outside the cover's header, so look it up page-wide.
    const link = document.querySelector('a[data-artwork-task]');
    if (!link || !ready) return;
    const files = [...(event.dataTransfer?.files || [])];
    open(link, files.length === 1 ? files[0] : null, artworkError(files.length, files[0]));
  });
  // Keep Tab within task controls, including the wrap through browser chrome in Chromium.
  dialog.addEventListener('keydown', event => {
    if (event.key !== 'Tab' || event.altKey || event.ctrlKey || event.metaKey) return;
    const controls = [...dialog.querySelectorAll('a[href],button,input:not([type=hidden]),select,textarea,summary,[tabindex="0"]')]
      // Chromium retains layout boxes for inputs inside closed details; those are not focusable.
      .filter(element => element.tabIndex >= 0 && !element.matches(':disabled') && !element.closest('[inert]') && element.checkVisibility({visibilityProperty:true}));
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
    (returnFocus || link)?.focus({preventScroll: true});
    returnFocus = null;
  });
  dialog.addEventListener('click', event => {
    // Cancel closes this confirmation without issuing another request.
    if (event.target.closest('[data-task-cancel]')) { event.preventDefault(); dialog.close(); return; }
    const link = event.target.closest('a[href]');
    if (!link || link.hasAttribute('data-task-page') || link.hasAttribute('download') ||
        link.target && link.target !== '_self' || !plainClick(event) || !allowed(new URL(link.href))) return;
    event.preventDefault();
    if (dialog.hasAttribute('aria-busy')) return;
    const url = new URL(link.href);
    url.searchParams.set('partial', '1');
    request(url);
  });
  dialog.addEventListener('change', event => {
    const input = event.target;
    if (!(input instanceof HTMLInputElement) || input.name !== 'image') return;
    const file = input.files?.[0];
    if (input.files?.length !== 1 || !file) {
      applyArtworkFile(null);
      pendingArtworkError = 'Choose exactly one image file.';
      status.textContent = pendingArtworkError;
      return;
    }
    applyArtworkFile(file);
  });
  // Only results are swapped while typing: retain the field, caret and public URL.
  async function search(form) {
    cancelSearch();
    if (!activation || dialog.hasAttribute('aria-busy') || !form.isConnected) return;
    const owner = activation;
    const version = searchVersion;
    const controller = new AbortController();
    searchController = controller;
    const abort = () => controller.abort();
    owner.signal.addEventListener('abort', abort, {once:true});
    const timeout = setTimeout(abort, 10000);
    const sectionID = form.dataset.liveSearch === 'merge' ? 'merge-results' : 'artist-results';
    const section = content.querySelector('#' + sectionID);
    section?.setAttribute('aria-busy', 'true');
    if (section) section.inert = true;
    try {
      // Read the attribute so named form controls cannot shadow the URL.
      const url = new URL(form.getAttribute('action'), location.href);
      if (!allowed(url)) throw Error('Invalid search');
      url.search = new URLSearchParams(new FormData(form)).toString();
      url.searchParams.set('partial', '1');
      const response = await fetch(url, {signal:controller.signal, cache:'no-store', credentials:'same-origin', redirect:'manual',
        headers:{'X-TuneTrail-Partial':'1','X-TuneTrail-Dialog':'1','X-TuneTrail-Search':form.dataset.liveSearch,
          'X-TuneTrail-Submission':content.querySelector('input[name=submission_token]')?.value || ''}});
      if (!response.ok || response.headers.get('X-TuneTrail-Fragment') !== sectionID) throw Error('Invalid search fragment');
      const html = await response.text();
      if (owner !== activation || controller.signal.aborted || version !== searchVersion || !section?.isConnected) return;
      const template = document.createElement('template');
      template.innerHTML = html.trim();
      const next = template.content.firstElementChild;
      if (template.content.childElementCount !== 1 || next?.id !== sectionID || !next.matches('[data-section]')) throw Error('Invalid results');
      section.replaceWith(next);
      status.textContent = '';
      next.querySelector('[data-search-status]').textContent = dialog.dataset.searchUpdated;
    } catch {
      if (owner === activation && !owner.signal.aborted && version === searchVersion && section?.isConnected) {
        const message = document.createElement('p');
        message.setAttribute('role', 'status');
        message.dataset.searchStatus = '';
        message.className = 'section-error';
        message.textContent = dialog.dataset.searchError;
        section.replaceChildren(message);
        // Leave the input and ordinary Search button usable after a failed read.
        status.textContent = dialog.dataset.searchError;
      }
    } finally {
      clearTimeout(timeout);
      owner.signal.removeEventListener('abort', abort);
      if (searchController === controller) {
        if (section) section.inert = false;
        section?.removeAttribute('aria-busy');
        searchController = null;
      }
    }
  }
  dialog.addEventListener('input', event => {
    const form = event.target.closest('form[data-live-search]');
    if (!form || event.isComposing || !event.target.matches('input[name=merge_query],input[name=artist_query]')) return;
    cancelSearch();
    const section = content.querySelector('#' + (form.dataset.liveSearch === 'merge' ? 'merge-results' : 'artist-results'));
    if (section) section.inert = true;
    searchTimer = setTimeout(() => search(form), 250);
  });
  window.TuneTrailEnhance.install(dialog, {submit(form, submitter) {
    if (!activation || dialog.hasAttribute('aria-busy')) return;
    if (form.matches('[data-live-search]')) { search(form); return; }
    // A control named action shadows HTMLFormElement.action (legacy review forms).
    const url = new URL(form.getAttribute('action'), location.href);
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
    document.querySelectorAll('[data-listen-controls]').forEach(cell => cell.remove());
    try {
      const response = await fetch(location.href, {method: 'HEAD', signal: controller.signal, cache: 'no-store', credentials: 'same-origin'});
      if (validation !== controller || controller.signal.aborted) return;
      if (!response.ok || response.redirected || response.headers.get('X-TuneTrail-Manage-Listening') !== 'true' ||
          response.headers.get('X-TuneTrail-Admin') !== String('admin' in document.body.dataset) ||
          response.headers.get('X-TuneTrail-UI-Generation') !== document.body.dataset.uiGeneration) { location.reload(); return; }
      ready = true;
      listenControls();
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
    document.querySelectorAll('[data-listen-controls]').forEach(cell => cell.remove());
  });
  addEventListener('pageshow', event => { if (event.persisted) revalidate(); });
  if (performance.getEntriesByType('navigation')[0]?.type === 'back_forward') revalidate();
})();
