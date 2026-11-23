/* Progressive form enhancements; ordinary GET/POST remains available without JavaScript. */
(() => {
 const section=new URLSearchParams(location.search).get('section');if(['keys','relay','import','export'].includes(section))requestAnimationFrame(()=>document.getElementById(section==='export'?'import':section)?.scrollIntoView());
 // Preserve a visible avatar when a profile image cannot be loaded (shared sidebar avatar, ISSUE-122).
 document.querySelectorAll('[data-avatar-image]').forEach(image=>{const fallback=image.parentElement.querySelector('[data-avatar-fallback]');const fail=()=>{image.hidden=true;if(fallback)fallback.hidden=false;};image.addEventListener('error',fail);if(image.complete&&!image.naturalWidth)fail();});
 // Enhanced forms (ISSUE-124) are serialized by enhance.js instead.
 document.addEventListener('submit',event=>{const form=event.target;if(!(form instanceof HTMLFormElement)||form.method.toLowerCase()!=='post'||form.matches('[data-enhance]'))return;if(form.dataset.pending){event.preventDefault();return;}form.dataset.pending='1';setTimeout(()=>form.querySelectorAll('button').forEach(button=>button.disabled=true),0);});
 // Do not restore expired sessions or old CSRF/submission forms from the browser heap.
 addEventListener('pagehide', () => { document.body.hidden = true; });
 addEventListener('pageshow', event => {
   if (event.persisted) { location.reload(); return; }
   document.body.hidden = false;
   document.querySelectorAll('form[data-pending]').forEach(form => { delete form.dataset.pending; form.querySelectorAll('button').forEach(button => button.disabled = false); });
 });
 document.querySelectorAll('input[type=file][name=image]').forEach(input=>{const form=input.form;if(!form)return;form.addEventListener('dragover',event=>{event.preventDefault();form.dataset.dropActive='1';});form.addEventListener('dragleave',()=>delete form.dataset.dropActive);form.addEventListener('drop',event=>{event.preventDefault();delete form.dataset.dropActive;const files=event.dataTransfer?.files;if(files?.length===1){input.files=files;input.dispatchEvent(new Event('change',{bubbles:true}));input.focus();}});});
 // Delegate copying so a key card replaced by a form response works without new listeners.
 document.body.dataset.keyCopy = '';
 document.addEventListener('click', async event => {
   const button = event.target.closest('button.account-key-copy');
   if (!button) return;
   const code = button.closest('.account-key').querySelector('.account-key-value');
   try {
     await navigator.clipboard.writeText(code.textContent);
     button.textContent = document.body.dataset.i18nCopied;
   } catch {
     const range = document.createRange();
     range.selectNodeContents(code);
     const selection = getSelection();
     selection.removeAllRanges();
     selection.addRange(range);
     button.textContent = document.body.dataset.i18nCopyManual;
   }
 });
})();
