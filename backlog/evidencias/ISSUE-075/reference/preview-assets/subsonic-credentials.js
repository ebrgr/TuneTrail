/* ISSUE-127: builds the Subsonic u/t/s parameters from a username and password in this browser.
   The password is never put in a form, a request, storage or the page; only u, t and s leave it. */
(() => {
  // Subsonic token authentication is md5(password + salt); WebCrypto has no MD5, so it is computed here.
  // RFC 1321, over UTF-8 bytes. Returns lowercase hex.
  function md5(text) {
    const bytes = new TextEncoder().encode(text);
    const length = bytes.length;
    const padded = new Uint8Array(((length + 8) >> 6 << 6) + 64);
    padded.set(bytes);
    padded[length] = 0x80;
    const view = new DataView(padded.buffer);
    view.setUint32(padded.length - 8, (length << 3) >>> 0, true);
    view.setUint32(padded.length - 4, Math.floor(length / 0x20000000), true);
    const shifts = [7, 12, 17, 22, 5, 9, 14, 20, 4, 11, 16, 23, 6, 10, 15, 21];
    const table = Array.from({length: 64}, (_, i) => Math.floor(Math.abs(Math.sin(i + 1)) * 0x100000000) >>> 0);
    let a0 = 0x67452301, b0 = 0xefcdab89, c0 = 0x98badcfe, d0 = 0x10325476;
    for (let offset = 0; offset < padded.length; offset += 64) {
      let a = a0, b = b0, c = c0, d = d0;
      for (let i = 0; i < 64; i++) {
        let f, g;
        if (i < 16) { f = (b & c) | (~b & d); g = i; }
        else if (i < 32) { f = (d & b) | (~d & c); g = (5 * i + 1) % 16; }
        else if (i < 48) { f = b ^ c ^ d; g = (3 * i + 5) % 16; }
        else { f = c ^ (b | ~d); g = (7 * i) % 16; }
        f = (f + a + table[i] + view.getUint32(offset + g * 4, true)) >>> 0;
        a = d; d = c; c = b;
        const s = shifts[(i >> 4) * 4 + (i & 3)];
        b = (b + ((f << s) | (f >>> (32 - s)))) >>> 0;
      }
      a0 = (a0 + a) >>> 0; b0 = (b0 + b) >>> 0; c0 = (c0 + c) >>> 0; d0 = (d0 + d) >>> 0;
    }
    const out = new DataView(new ArrayBuffer(16));
    [a0, b0, c0, d0].forEach((word, i) => out.setUint32(i * 4, word, true));
    return Array.from(new Uint8Array(out.buffer), byte => byte.toString(16).padStart(2, '0')).join('');
  }

  // New random salt for each generation, 16 hex characters.
  function newSalt() {
    return Array.from(crypto.getRandomValues(new Uint8Array(8)), byte => byte.toString(16).padStart(2, '0')).join('');
  }

  // Replace u, p, t and s of the existing parameters and keep any other provider extension.
  function buildParams(existing, user, password, salt) {
    const kept = [...new URLSearchParams(existing)].filter(([key]) => !['u', 'p', 't', 's'].includes(key));
    return new URLSearchParams([['u', user], ['t', md5(password + salt)], ['s', salt], ...kept]).toString();
  }

  window.TuneTrailSubsonic = {md5, buildParams};

  const dialog = document.getElementById('subsonic-credentials');
  const open = document.querySelector('[data-subsonic-open]');
  const form = document.querySelector('form.server-settings-form');
  const field = form?.elements.KOITO_SUBSONIC_PARAMS;
  if (!dialog?.showModal || !open || !form || !field || field.disabled) return;
  const user = dialog.querySelector('[data-subsonic-user]');
  const password = dialog.querySelector('[data-subsonic-password]');
  const result = dialog.querySelector('[data-subsonic-result]');
  const filled = document.querySelector('[data-subsonic-filled]');
  const buttons = dialog.querySelectorAll('button');
  let request = null;
  let after = open; // where focus goes when the dialog closes

  function show(text, kind) {
    result.textContent = text;
    result.hidden = !text;
    result.className = 'message message-' + (kind || 'error');
    result.setAttribute('role', kind === 'error' || !kind ? 'alert' : 'status');
  }
  // The password lives only in this input and is erased whenever the dialog stops using it.
  function clear() { password.value = ''; }
  function busy(on) {
    dialog.toggleAttribute('aria-busy', on);
    buttons.forEach(button => { if (!button.hasAttribute('data-subsonic-cancel')) button.disabled = on; });
  }
  function input() {
    if (!user.value.trim() || !password.value) { show(dialog.dataset.errorEmpty); return null; }
    return {user: user.value.trim(), salt: newSalt()};
  }

  open.hidden = false;
  open.addEventListener('click', () => { show(''); dialog.showModal(); user.focus(); });
  dialog.addEventListener('close', () => { request?.abort(); clear(); show(''); busy(false); after.focus(); after = open; });
  dialog.addEventListener('click', event => { if (event.target.closest('[data-subsonic-cancel]')) dialog.close(); });
  password.addEventListener('keydown', event => { if (event.key === 'Enter') { event.preventDefault(); generate(); } });
  dialog.querySelector('[data-subsonic-generate]').addEventListener('click', generate);
  dialog.querySelector('[data-subsonic-test]').addEventListener('click', test);

  // Fill only the parameters field; saving stays an explicit action of the main form.
  function generate() {
    const values = input();
    if (!values) return;
    field.value = buildParams(field.value, values.user, password.value, values.salt);
    clear();
    if (filled) filled.hidden = false;
    after = field;
    dialog.close();
  }

  // Test with the generated parameters without touching the page field, through the existing test endpoint.
  async function test() {
    const values = input();
    if (!values) return;
    const data = new URLSearchParams(new FormData(form));
    data.set('KOITO_SUBSONIC_PARAMS', buildParams(field.value, values.user, password.value, values.salt));
    clear();
    request = new AbortController();
    busy(true);
    show(dialog.dataset.testing, 'status');
    try {
      const response = await fetch('/admin/settings/subsonic/test', {method: 'POST', body: data, credentials: 'same-origin',
        redirect: 'error', signal: request.signal, headers: {Accept: 'text/html'}});
      const page = new DOMParser().parseFromString(await response.text(), 'text/html');
      const next = page.querySelector('form.server-settings-form');
      const message = page.querySelector('#settings-notice, #settings-error');
      if (!next || !message) throw new Error('unexpected response');
      // Each response issues fresh one-use tickets for the next submission of the page form.
      ['csrf_token', 'submission_token', 'revision'].forEach(name => {
        const fresh = next.elements[name];
        if (fresh && form.elements[name]) form.elements[name].value = fresh.value;
      });
      show(message.textContent.trim(), message.id === 'settings-notice' ? 'success' : 'error');
    } catch (error) {
      if (error.name === 'AbortError') return;
      show(dialog.dataset.errorFailed);
    } finally { busy(false); request = null; }
  }
})();
