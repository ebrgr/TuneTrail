/* Instant search (ISSUE-107): results while typing in the search dialog, as an ARIA
   combobox. Without this script the form opens the full search page. */
(() => {
  const form = document.querySelector('[data-instant-search]');
  if (!form) return;
  const input = form.querySelector('input[name=q]');
  const box = document.querySelector('[data-search-results]');
  const status = document.querySelector('[data-search-status]');
  let timer, request = null, active = -1;
  const options = () => [...box.querySelectorAll('[role=option]')];
  const setActive = index => {
    const list = options();
    list.forEach((option, i) => option.setAttribute('aria-selected', String(i === index)));
    active = index;
    if (list[index]) {
      input.setAttribute('aria-activedescendant', list[index].id);
      list[index].scrollIntoView({block: 'nearest'});
    } else input.removeAttribute('aria-activedescendant');
  };
  const clear = () => {
    clearTimeout(timer);
    request?.abort();
    request = null;
    box.replaceChildren();
    input.setAttribute('aria-expanded', 'false');
    setActive(-1);
  };
  // Only the latest query may fill the list; older or aborted responses are discarded.
  async function load(q) {
    request?.abort();
    const controller = new AbortController();
    request = controller;
    const url = new URL(form.action, location.origin);
    url.searchParams.set('q', q);
    url.searchParams.set('partial', '1');
    try {
      const response = await fetch(url, {headers: {'X-TuneTrail-Partial': '1'}, credentials: 'same-origin', signal: controller.signal});
      const html = response.ok ? await response.text() : '';
      if (request !== controller || input.value.trim() !== q) return;
      box.innerHTML = html; // same-origin fragment, escaped by html/template on the server
      setActive(-1);
      input.setAttribute('aria-expanded', String(options().length > 0));
      // Clear first so an identical message (e.g. two "No results") is announced again.
      status.textContent = '';
      requestAnimationFrame(() => { status.textContent = box.querySelector('[data-search-summary]')?.textContent.trim() || ''; });
    } catch (error) {
      if (error.name !== 'AbortError' && request === controller) status.textContent = '';
    }
  }
  input.addEventListener('input', () => {
    clearTimeout(timer);
    setActive(-1); // Enter must not open an option of the previous list

    const q = input.value.trim();
    if (!q) { clear(); status.textContent = ''; return; }
    timer = setTimeout(() => load(q), 120);
  });
  input.addEventListener('keydown', event => {
    if (event.isComposing) return; // Enter confirming an IME composition
    const list = options();
    if (event.key === 'ArrowDown' && list.length) { event.preventDefault(); setActive((active + 1) % list.length); }
    else if (event.key === 'ArrowUp' && list.length) { event.preventDefault(); setActive(active <= 0 ? list.length - 1 : active - 1); }
    else if (event.key === 'Enter' && list[active]) { event.preventDefault(); location.href = list[active].querySelector('a').href; }
    // The first Escape closes the list; the next one closes the dialog (a search field
    // would otherwise spend it clearing its text).
    else if (event.key === 'Escape') {
      event.preventDefault();
      if (input.getAttribute('aria-expanded') === 'true') clear();
      else { input.blur(); form.closest('dialog')?.close(); }
    }
  });
  // Closing the dialog starts the next search from an empty field. The close event is
  // queued, so skip it when the dialog was already reopened (text typed since then).
  const dialog = form.closest('dialog');
  dialog?.addEventListener('close', () => {
    if (dialog.open) return;
    clear();
    input.value = '';
    status.textContent = '';
  });
  addEventListener('pagehide', () => request?.abort());
  addEventListener('pageshow', event => { if (event.persisted) { clear(); status.textContent = ''; } });
})();
