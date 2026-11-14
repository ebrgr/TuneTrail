/* Runs before paint. Only appearance preferences are stored; never a profile. */
(() => {
  const names = ['yuu','varia','midnight','catppuccin','autumn','black','wine','pearl','rosebud','urim','match','lemon'];
  const keys = ['bg','bgSecondary','bgTertiary','fg','fgSecondary','fgTertiary','primary','primaryDim','accent','accentDim','error','warning','info','success'];
  const root = document.documentElement;
  const defaultTheme = root.dataset.theme;
  const read = key => { try { return localStorage.getItem(key); } catch { return null; } };
  const write = (key, value) => { try { value === null ? localStorage.removeItem(key) : localStorage.setItem(key, value); return true; } catch { return false; } };
  const cssName = key => '--color-' + key.replace(/[A-Z]/g, c => '-' + c.toLowerCase());
  const valid = palette => palette && typeof palette === 'object' && !Array.isArray(palette) && Object.keys(palette).length === keys.length && keys.every(key => typeof palette[key] === 'string' && /^(#[0-9a-f]{3}|#[0-9a-f]{6}|#[0-9a-f]{8})$/i.test(palette[key]));
  let custom = null;
  try { const parsed = JSON.parse(read('custom-theme')); if (parsed && typeof parsed === 'object') { const {name, ...colors} = parsed; if (valid(colors)) custom = colors; } } catch {}
  function apply(name, palette, persist = true) {
    if (name === 'custom' && !valid(palette)) return false;
    if (name !== 'custom' && !names.includes(name)) name = defaultTheme;
    keys.forEach(key => root.style.removeProperty(cssName(key)));
    root.style.removeProperty('background-color');
    root.dataset.theme = name;
    if (name === 'custom') { keys.forEach(key => root.style.setProperty(cssName(key), palette[key])); custom = palette; }
    if (persist) {
      write('theme', name);
      if (name === 'custom') write('custom-theme', JSON.stringify(palette));
      write('bgcolor', getComputedStyle(root).getPropertyValue('--color-bg').trim());
    }
    document.dispatchEvent(new CustomEvent('tunetrail-theme-change')); // ISSUE-111
    return true;
  }
  const saved = read('theme');
  apply(saved === 'custom' && custom ? 'custom' : names.includes(saved) ? saved : defaultTheme, custom, false);
  window.TuneTrailTheme = {apply, valid, keys, getCustom: () => custom, reset: () => { apply(defaultTheme, null, false); write('theme', null); write('bgcolor', null); }};
})();
