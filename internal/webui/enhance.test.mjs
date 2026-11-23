import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

const script = readFileSync(new URL('../../assets/js/enhance.js', import.meta.url), 'utf8');

// ISSUE-201: load the script with no enhanced form on the page, so only the helpers are exposed.
const context = {window: {}, document: {querySelector: () => null}, URL, location: {href: 'http://tune.test/account/profile'}};
vm.runInNewContext(script, context);
const {submitURL} = context.window.TuneTrailEnhance;

// A button without the formaction attribute: browsers answer the document URL for button.formAction.
const plainButton = {formAction: 'http://tune.test/account/profile', getAttribute: () => null};

test('a button without formaction posts to the form action, not to the page URL', () => {
  const form = {action: 'http://tune.test/admin/profile/1/avatar'};
  assert.equal(submitURL(form, plainButton).href, 'http://tune.test/admin/profile/1/avatar');
});

test('a button formaction attribute wins over the form action, relative values included', () => {
  const form = {action: 'http://tune.test/admin/profile/1/relay/save'};
  const button = {formAction: 'ignored', getAttribute: name => name === 'formaction' ? '/admin/profile/1/relay/test' : null};
  assert.equal(submitURL(form, button).href, 'http://tune.test/admin/profile/1/relay/test');
});

test('a form without an action attribute posts to the page, and a missing submitter uses the form action', () => {
  assert.equal(submitURL({action: 'http://tune.test/account/profile'}, plainButton).href, 'http://tune.test/account/profile');
  assert.equal(submitURL({action: 'http://tune.test/admin/x'}, undefined).href, 'http://tune.test/admin/x');
});
