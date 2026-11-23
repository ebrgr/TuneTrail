import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

const script = readFileSync(new URL('../../assets/js/subsonic-credentials.js', import.meta.url), 'utf8');

// ISSUE-127: load the script with no page, so only the pure helpers are exposed.
const context = {window: {}, document: {getElementById: () => null, querySelector: () => null}, TextEncoder, URLSearchParams, crypto: globalThis.crypto};
vm.runInNewContext(script, context);
const {md5, buildParams} = context.window.TuneTrailSubsonic;

test('md5 matches the RFC 1321 vectors', () => {
  const vectors = {
    '': 'd41d8cd98f00b204e9800998ecf8427e',
    a: '0cc175b9c0f1b6a831c399e269772661',
    abc: '900150983cd24fb0d6963f7d28e17f72',
    'message digest': 'f96b697d7cb7938d525a2f31aaf161d0',
    abcdefghijklmnopqrstuvwxyz: 'c3fcd3d76192e4007dfb496cca67e13b',
    '12345678901234567890123456789012345678901234567890123456789012345678901234567890': '57edf4a22be3c955ac49da2e2107b67a',
  };
  for (const [input, expected] of Object.entries(vectors)) assert.equal(md5(input), expected);
});

test('md5 agrees with node for padding boundaries and UTF-8', () => {
  for (const length of [54, 55, 56, 57, 63, 64, 65, 119, 120, 1000]) {
    const input = 'x'.repeat(length);
    assert.equal(md5(input), createHash('md5').update(input).digest('hex'), String(length));
  }
  assert.equal(md5('sénha-ção-✓'), createHash('md5').update('sénha-ção-✓').digest('hex'));
});

test('buildParams uses md5(password + salt), keeps extensions and drops old credentials', () => {
  const params = new URLSearchParams(buildParams('u=old&p=plain&t=x&s=y&custom=1', 'ana & co', 'segredo', 'abcd1234abcd1234'));
  assert.equal(params.get('u'), 'ana & co');
  assert.equal(params.get('s'), 'abcd1234abcd1234');
  assert.equal(params.get('t'), createHash('md5').update('segredoabcd1234abcd1234').digest('hex'));
  assert.equal(params.get('custom'), '1');
  assert.equal(params.has('p'), false);
  assert.deepEqual([...params.keys()], ['u', 't', 's', 'custom']);
});

test('the password never appears in the generated parameters', () => {
  assert.equal(buildParams('', 'ana', 'teste-senha-única', '0011223344556677').includes('teste-senha-única'), false);
});
