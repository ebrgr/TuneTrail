/* Second sidebar disclosure (ISSUE-123). Without this script the trigger is a plain link
   to administration, where the server renders the second sidebar open. */
(() => {
  const bar = document.getElementById('nav-admin');
  const toggle = document.querySelector('[data-nav-toggle]');
  if (!bar || !toggle) return;
  if (toggle.tagName === 'A') toggle.setAttribute('role', 'button');
  const set = open => {
    bar.hidden = !open;
    toggle.setAttribute('aria-expanded', String(open));
  };
  // Overlay mode (theme panel): the bar covers the content, so it starts closed even where
  // the server opens it (administration) and closes after an item is chosen.
  const overlay = () => document.documentElement.dataset.navLayout === 'overlay';
  set(!bar.hidden && !overlay());
  document.documentElement.dataset.navReady = '';
  bar.addEventListener('click', event => {
    if (overlay() && event.target.closest('a')) set(false);
  });
  // ISSUE-071: entering administration closes More so two mobile panels cannot overlap.
  const activate = () => {
    set(bar.hidden);
    if (!bar.hidden) {
      const more = toggle.closest('details');
      if (more) more.open = false;
      bar.querySelector('a,button')?.focus();
    }
  };
  toggle.addEventListener('click', event => {
    event.preventDefault();
    activate();
  });
  // A link acting as a button must also respond to the space bar.
  toggle.addEventListener('keydown', event => {
    if (event.key !== ' ') return;
    event.preventDefault();
    activate();
  });
  bar.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || event.defaultPrevented || bar.hidden) return;
    set(false);
    const more = toggle.closest('details');
    if (more) more.open = true;
    toggle.focus();
  });
})();
