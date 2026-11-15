(() => {
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
    if (trigger) openDialog(trigger.dataset.dialog, trigger);
    const close = event.target.closest('[data-close]');
    if (close) closeDialog(close.closest('dialog'));
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
  document.querySelectorAll('img[data-cover],img[data-avatar-image]').forEach(image => {
    image.addEventListener('error', () => { if (!image.classList.contains('cover-failed')) failImage(image); });
    if (image.complete && !image.naturalWidth) failImage(image);
  });
})();
