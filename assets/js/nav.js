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
  set(!bar.hidden);
  toggle.addEventListener('click', event => {
    event.preventDefault();
    set(bar.hidden);
  });
  // A link acting as a button must also respond to the space bar.
  toggle.addEventListener('keydown', event => {
    if (event.key !== ' ') return;
    event.preventDefault();
    set(bar.hidden);
  });
  bar.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || event.defaultPrevented || bar.hidden) return;
    set(false);
    toggle.focus();
  });
})();
