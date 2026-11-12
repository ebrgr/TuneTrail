(() => {
  let opener;
  const theme = window.KoitoTheme;
  const status = document.getElementById('theme-status');
  const text = document.body.dataset;
  const customInput = document.getElementById('custom-theme-input');
  const updateThemeButtons = () => document.querySelectorAll('[data-theme-choice]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.themeChoice === document.documentElement.dataset.theme)));
  updateThemeButtons();
  document.addEventListener('koito-theme-change', updateThemeButtons);
  // The server sends only bundled demo palettes. No external calls are needed.
  const assets = document.body.dataset.assets || '/preview-assets';
  fetch(assets + '/themes.json').then(response => { if (!response.ok) throw Error(); return response.json(); }).then(palettes => {
    customInput.value = JSON.stringify(theme.getCustom() || palettes.yuu, null, 2);
  }).catch(() => { status.textContent = text.i18nThemeLoadError; });
  function openDialog(id, button) {
    const dialog = document.getElementById(id);
    if (!dialog || document.querySelector('dialog[open]')) return;
    opener = button || document.activeElement;
    dialog.showModal();
  }
  function closeDialog(dialog) { dialog.close(); }
  document.addEventListener('click', event => {
    const trigger = event.target.closest('[data-dialog]');
    if (trigger) openDialog(trigger.dataset.dialog, trigger);
    const close = event.target.closest('[data-close]');
    if (close) closeDialog(close.closest('dialog'));
    const choice = event.target.closest('[data-theme-choice]');
    if (choice) {theme.apply(choice.dataset.themeChoice);status.textContent = text.i18nThemeApplied;}
  });
  document.querySelectorAll('dialog').forEach(dialog => {
    dialog.addEventListener('close', () => { if (opener?.isConnected) opener.focus(); });
    dialog.addEventListener('click', event => {
      if (event.target !== dialog) return;
      const box = dialog.getBoundingClientRect();
      if (event.clientX < box.left || event.clientX > box.right || event.clientY < box.top || event.clientY > box.bottom) closeDialog(dialog);
    });
  });
  document.addEventListener('keydown', event => {
    if (event.ctrlKey || event.metaKey || event.altKey || event.target.closest('input,textarea,select,[contenteditable=true]')) return;
    if (event.key === '\\' && !document.querySelector('dialog[open]')) {event.preventDefault();openDialog('settings-dialog');}
    if (event.key === '/' && document.querySelector('[data-dialog=search-dialog]') && !document.querySelector('dialog[open]')) {event.preventDefault();openDialog('search-dialog');}
  });
  function selectTab(tab, focus = false) {
    document.querySelectorAll('[role=tab]').forEach(item => {
      const active = item === tab;
      item.setAttribute('aria-selected', String(active));
      item.tabIndex = active ? 0 : -1;
      document.getElementById(item.getAttribute('aria-controls')).hidden = !active;
    });
    if (focus) tab.focus();
  }
  const tabs = [...document.querySelectorAll('[role=tab]')];
  tabs.forEach((tab,index) => {
    tab.addEventListener('click', () => selectTab(tab));
    tab.addEventListener('keydown', event => {
      let next;
      if (event.key === 'ArrowDown') next = (index+1)%tabs.length;
      if (event.key === 'ArrowUp') next = (index+tabs.length-1)%tabs.length;
      if (event.key === 'Home') next = 0;
      if (event.key === 'End') next = tabs.length-1;
      if (next !== undefined) {event.preventDefault();selectTab(tabs[next],true);}
    });
  });
  document.getElementById('reset-theme').addEventListener('click', () => {theme.reset();status.textContent = text.i18nThemeReset;});
  document.getElementById('custom-theme-form').addEventListener('submit', event => {
    event.preventDefault();
    try {
      const palette = JSON.parse(customInput.value);
      if (!theme.valid(palette)) throw Error(text.i18nThemeInvalidPalette);
      theme.apply('custom', palette);
      customInput.value = JSON.stringify(palette, null, 2);
      status.textContent = text.i18nThemeCustomApplied;
    } catch (error) {status.textContent = error instanceof SyntaxError ? text.i18nThemeInvalidJson : error.message;}
  });
  const failImage = image => {
    if (image.matches('[data-avatar-image]')) {
      image.hidden = true;
      image.parentElement.querySelector('[data-avatar-fallback]').hidden = false;
    } else {
      image.classList.add('cover-failed');
      image.removeAttribute('srcset');
      image.src = assets + '/icons/Disc3.svg';
    }
  };
  document.querySelectorAll('img[data-cover],img[data-avatar-image]').forEach(image => {
    image.addEventListener('error', () => { if (!image.classList.contains('cover-failed')) failImage(image); });
    if (image.complete && !image.naturalWidth) failImage(image);
  });
})();
