(() => {
  // Administration owns its form and image enhancements; More remains shared below.
  if (document.body.hasAttribute('data-admin-ui')) return;
  // Dialogs and image fallbacks; theme changes live in theme-picker.js (ISSUE-110).
  let opener;
  const text = document.body.dataset;
  function openDialog(id, button) {
    const dialog = document.getElementById(id);
    if (!dialog || document.querySelector('dialog[open]')) return;
    opener = button || document.activeElement;
    dialog.showModal();
  }
  function closeDialog(dialog) { dialog.close(); }
  document.addEventListener('click', event => {
    const trigger = event.target.closest('[data-dialog]');
    if (trigger && document.getElementById(trigger.dataset.dialog)) {
      // ISSUE-071: links retain document navigation when scripts or dialogs are unavailable.
      event.preventDefault();
      openDialog(trigger.dataset.dialog, trigger);
    }
    const close = event.target.closest('[data-close]');
    if (close) closeDialog(close.closest('dialog'));
  });
  document.querySelectorAll('dialog').forEach(dialog => {
    // Return focus to the opener; a dialog opened by a shortcut has none, so never leave
    // focus on a hidden field inside it (it would swallow the next shortcut, ISSUE-107).
    dialog.addEventListener('close', () => {
      if (dialog.open) return; // the queued close event arrived after a quick reopen
      if (opener?.isConnected && opener !== document.body) opener.focus();
      else if (dialog.contains(document.activeElement)) document.activeElement.blur();
    });
    dialog.addEventListener('click', event => {
      if (event.target !== dialog) return;
      const box = dialog.getBoundingClientRect();
      if (event.clientX < box.left || event.clientX > box.right || event.clientY < box.top || event.clientY > box.bottom) closeDialog(dialog);
    });
  });
  document.addEventListener('keydown', event => {
    if (event.ctrlKey || event.metaKey || event.altKey || event.target.closest('input,textarea,select,[contenteditable=true]')) return;
    if (event.key === '/' && document.querySelector('[data-dialog=search-dialog]') && !document.querySelector('dialog[open]')) {event.preventDefault();openDialog('search-dialog');}
  });
  const failImage = image => {
    if (image.matches('[data-avatar-image]')) {
      image.hidden = true;
      image.parentElement.querySelector('[data-avatar-fallback]').hidden = false;
    } else {
      image.classList.add('cover-failed');
      image.removeAttribute('srcset');
      image.src = text.coverFallbackUrl;
    }
  };
  // Captured error events also cover artwork inserted by an authoritative history refresh.
  document.addEventListener('error', event => {
    const image = event.target;
    if (image instanceof HTMLImageElement && image.matches('img[data-cover],img[data-avatar-image]') &&
        !image.classList.contains('cover-failed')) failImage(image);
  }, true);
  document.querySelectorAll('img[data-cover],img[data-avatar-image]').forEach(image => {
    if (image.complete && !image.naturalWidth) failImage(image);
  });
})();

/* Use the sidebar tooltip appearance on public media actions without a duplicate native title. */
(() => {
  document.documentElement.dataset.mediaTooltips = '';
  function tooltip(event) {
    const link = event.target.closest('.media-actions [data-tip]');
    if (!link) return;
    if (event.type === 'pointerover' || event.type === 'focusin') link.removeAttribute('title');
    else if (!link.matches(':hover') && !link.contains(document.activeElement)) link.title = link.dataset.tip;
  }
  ['pointerover','pointerout','focusin','focusout'].forEach(type => document.addEventListener(type, tooltip));
})();

// ISSUE-071: Escape closes the native disclosure and restores its summary focus.
document.querySelectorAll('.nav-more').forEach(more => {
  more.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || event.defaultPrevented || !more.open || document.querySelector(':popover-open,dialog[open]')) return;
    more.open = false;
    more.querySelector('summary').focus();
    event.preventDefault();
  });
});

// ISSUE-154: clicking or tapping outside the More menu closes it, without moving focus.
document.addEventListener('pointerdown', event => {
  document.querySelectorAll('.nav-more[open]').forEach(more => {
    if (!more.contains(event.target)) {
      more.open = false;
      // Let the click focus its destination before rescuing focus from hidden actions.
      requestAnimationFrame(() => {
        if (!more.open && more.contains(document.activeElement)) more.querySelector('summary').focus();
      });
    }
  });
});

// ISSUE-156: return from the theme to a visible invoker, even after outside dismissal closed More.
(() => {
  const panel = document.getElementById('theme-panel');
  if (!panel) return;
  let opener;
  let outsideTarget;
  panel.addEventListener('beforetoggle', event => {
    if (event.newState === 'open') {
      opener = document.activeElement;
      outsideTarget = null;
    }
  });
  document.addEventListener('pointerdown', event => {
    if (panel.matches(':popover-open') && !panel.contains(event.target)) outsideTarget = event.target;
  });
  panel.addEventListener('toggle', event => {
    if (event.newState !== 'closed' || !opener?.isConnected) return;
    // Queued toggle events must not steal focus after the user has already moved on.
    const active = document.activeElement;
    if (active !== opener && active !== document.body && !panel.contains(active) && active.checkVisibility()) {
      opener = null;
      return;
    }
    // Preserve focus on an outside destination the user deliberately clicked.
    const clicked = outsideTarget?.closest('a,button,input,select,textarea,summary,[tabindex]');
    if (clicked?.checkVisibility()) { opener = null; return; }
    const more = opener.closest('.nav-more');
    if (more) more.open = true;
    if (opener !== document.body && opener.checkVisibility()) opener.focus();
    else document.querySelector('.nav-more>summary')?.focus();
    opener = null;
  });
})();
