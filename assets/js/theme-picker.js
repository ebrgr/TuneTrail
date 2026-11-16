/* Theme panel shared by public pages, the preview and administration (ISSUE-110).
   The panel carries its own URLs and messages, so it works wherever the head loads it. */
(() => {
  const panel = document.querySelector('[data-theme-panel]');
  const theme = window.TuneTrailTheme;
  if (!panel || !theme) return;
  const text = panel.dataset;
  const status = panel.querySelector('[data-theme-status]');
  const customInput = panel.querySelector('#custom-theme-input');
  const say = message => { status.textContent = message; };
  // Mark the active theme for assistive technology after every change.
  const updateButtons = () => panel.querySelectorAll('[data-theme-choice]').forEach(button =>
    button.setAttribute('aria-pressed', String(button.dataset.themeChoice === document.documentElement.dataset.theme)));
  updateButtons();
  document.addEventListener('tunetrail-theme-change', updateButtons);
  // The server supplies the content-addressed palette URL of the same generation.
  fetch(text.themesUrl).then(response => { if (!response.ok) throw Error(); return response.json(); }).then(palettes => {
    customInput.value = JSON.stringify(theme.getCustom() || palettes.yuu, null, 2);
  }).catch(() => say(text.i18nLoadError));
  panel.addEventListener('click', event => {
    const choice = event.target.closest('[data-theme-choice]');
    if (choice) { theme.apply(choice.dataset.themeChoice); say(text.i18nApplied); }
    if (event.target.closest('[data-theme-reset]')) { theme.reset(); say(text.i18nReset); }
  });
  panel.querySelector('[data-custom-theme-form]').addEventListener('submit', event => {
    event.preventDefault();
    try {
      const palette = JSON.parse(customInput.value);
      if (!theme.valid(palette)) throw Error(text.i18nInvalidPalette);
      theme.apply('custom', palette);
      customInput.value = JSON.stringify(palette, null, 2);
      say(text.i18nCustomApplied);
    } catch (error) { say(error instanceof SyntaxError ? text.i18nInvalidJson : error.message); }
  });
  // "\" opens the panel from anywhere except form fields and open dialogs. Without an
  // invoker button the browser leaves focus behind, so move it into the panel and return
  // it on close (the button path is handled natively by popovertarget).
  // Push or overlay for the second sidebar (ISSUE-123), remembered on this browser only.
  const layoutChoice = panel.querySelector('[data-nav-layout-choice]');
  if (layoutChoice) {
    const current = document.documentElement.dataset.navLayout === 'overlay' ? 'overlay' : 'push';
    const selected = layoutChoice.querySelector(`input[value="${current}"]`);
    if (selected) selected.checked = true;
    layoutChoice.addEventListener('change', event => {
      const value = event.target.value === 'overlay' ? 'overlay' : 'push';
      if (value === 'overlay') document.documentElement.dataset.navLayout = 'overlay';
      else delete document.documentElement.dataset.navLayout;
      try { localStorage.setItem('nav-layout', value); } catch {}
    });
  }
  // ISSUE-154: icons-only is the default; labels are opt-in on this document (persistence: ISSUE-153).
  const labelsChoice = panel.querySelector('[data-nav-labels-choice]');
  if (labelsChoice) {
    const selected = document.documentElement.dataset.navLabels === 'labels' ? 'labels' : 'icons';
    labelsChoice.querySelector(`input[value="${selected}"]`).checked = true;
    labelsChoice.addEventListener('change', event => {
      const value = event.target.value === 'labels' ? 'labels' : 'icons';
      if (value === 'labels') document.documentElement.dataset.navLabels = 'labels';
      else delete document.documentElement.dataset.navLabels;
    });
  }
  let opener = null;
  document.addEventListener('keydown', event => {
    if (event.key !== '\\' || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.target.closest('input,textarea,select,[contenteditable=true]') || document.querySelector('dialog[open]')) return;
    if (!panel.showPopover || panel.matches(':popover-open')) return;
    event.preventDefault();
    opener = document.activeElement;
    panel.showPopover();
    (panel.querySelector('[data-theme-choice][aria-pressed=true]') || panel.querySelector('.close-dialog')).focus();
  });
  panel.addEventListener('toggle', event => {
    if (event.newState !== 'closed' || !opener) return;
    if (opener.isConnected) opener.focus();
    opener = null;
  });
})();
