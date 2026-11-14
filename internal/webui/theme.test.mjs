import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

const script = readFileSync(new URL('./static/theme.js', import.meta.url), 'utf8');
const palettes = JSON.parse(readFileSync(new URL('./palettes.json', import.meta.url)));

// Unit harness for the before-paint script; no browser or extra dependency.
function startup(stored = {}, unavailable = false) {
  const values = new Map(Object.entries(stored));
  const vars = new Map();
  const root = {dataset: {theme:'yuu'}, style: {
    setProperty: (key,value) => vars.set(key,value),
    removeProperty: key => vars.delete(key),
  }};
  const context = {
    window: {},
    document: {documentElement:root, dispatchEvent() {}},
    localStorage: {
      getItem(key) {if (unavailable) throw Error('storage disabled'); return values.get(key) ?? null;},
      setItem(key,value) {if (unavailable) throw Error('storage disabled'); values.set(key,value);},
      removeItem(key) {if (unavailable) throw Error('storage disabled'); values.delete(key);},
    },
    CustomEvent: class {},
    getComputedStyle: () => ({getPropertyValue: key => vars.get(key) ?? palettes[root.dataset.theme]?.bg ?? ''}),
  };
  vm.runInNewContext(script, context);
  return {api:context.window.TuneTrailTheme,root,vars,values};
}

test('unavailable storage and invalid saved JSON leave a usable default', () => {
  assert.equal(startup({},true).root.dataset.theme,'yuu');
  assert.equal(startup({theme:'unknown'}).root.dataset.theme,'yuu');
  assert.equal(startup({theme:'custom','custom-theme':'{broken'}).root.dataset.theme,'yuu');
  const blocked = startup({},true);
  assert.equal(blocked.api.apply('black'),true);
  assert.equal(blocked.root.dataset.theme,'black');
});

test('custom colors survive startup and reset clears inline overrides', () => {
  const palette = {...palettes.yuu,primary:'#66bbdd'};
  const current = startup({theme:'custom','custom-theme':JSON.stringify(palette)});
  assert.equal(current.root.dataset.theme,'custom');
  assert.equal(current.vars.get('--color-primary'),'#66bbdd');
  assert.equal(current.api.apply('custom',palette),true);
  assert.equal(current.values.get('theme'),'custom');
  current.api.reset();
  assert.equal(current.root.dataset.theme,'yuu');
  assert.equal(current.vars.size,0);
  assert.equal(current.values.has('theme'),false);
});

test('incomplete and injected palettes cannot set arbitrary CSS', () => {
  const current = startup();
  assert.equal(current.api.apply('custom',{}),false);
  assert.equal(current.api.valid({...palettes.yuu,bg:'red; background:url(https://example.invalid)'}),false);
  assert.equal(current.api.valid({...palettes.yuu,unexpected:'#ffffff'}),false);
  assert.equal(current.root.dataset.theme,'yuu');
  assert.equal(current.vars.size,0);
});
