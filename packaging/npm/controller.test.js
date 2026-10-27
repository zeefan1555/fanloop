'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');
const {spawnSync} = require('child_process');
const controller = require('./scripts/controller');

test('rootless launcher keeps data-home diagnostics outside the controller boundary', () => {
  const result = spawnSync(process.execPath, [path.join(__dirname, 'scripts/run.js'), 'version'], {
    env: {FANLOOP_DATA_HOME: 'relative/fanloop'}, encoding: 'utf8',
  });
  assert.equal(result.status, 1);
  assert.equal(result.stdout, '');
  assert.match(result.stderr, /FANLOOP_DATA_HOME must be absolute/);
  assert.doesNotMatch(result.stderr, /TypeError|"ok":|"INTERNAL"/);
});

test('controller argv follows the shared Cobra fixture', () => {
  const cases = JSON.parse(fs.readFileSync(path.join(__dirname,'../../internal/controller/testdata/argv.json'),'utf8'));
  for (const fixture of cases) {
    const result = controller.invocation(fixture.args);
    assert.deepEqual(result ? {root:result.root,rebind:result.rebind} : {root:'',rebind:false},{root:fixture.root,rebind:fixture.rebind},JSON.stringify(fixture.args));
    if ('dry_run' in fixture) assert.equal(result.dryRun, fixture.dry_run, JSON.stringify(fixture.args));
  }
});

test('controller journal rejects the shared wrong JSON types', () => {
  const fixture = JSON.parse(fs.readFileSync(path.join(__dirname,'../../internal/controller/testdata/journal-types.json'),'utf8'));
  assert.deepEqual(controller.decode(JSON.stringify(fixture.binding)), fixture.binding);
  for (const field of [...fixture.required_strings, ...fixture.optional_strings, 'schema_version']) {
    const values = field === 'schema_version'
      ? fixture.wrong_schema_values
      : [...fixture.wrong_string_values, [fixture.binding[field]]];
    for (const value of values) {
      const content = JSON.stringify({...fixture.binding, [field]: value});
      assert.throws(() => controller.decode(content), `${field}=${JSON.stringify(value)}`);
    }
  }
  for (const field of fixture.unknown_fields) {
    assert.throws(() => controller.decode(JSON.stringify({...fixture.binding, [field]: 1})), /invalid controller journal fields/, field);
  }
  const legacy = {...fixture.binding};
  delete legacy.release_resources_sha256;
  assert.throws(() => controller.decode(JSON.stringify(legacy)));
  for (const value of fixture.invalid_resource_digests) {
    assert.throws(() => controller.decode(JSON.stringify({...fixture.binding, release_resources_sha256: value})));
  }
  for (const field of ['prepared_at', 'committed_at']) {
    for (const value of fixture.valid_timestamps) {
      assert.doesNotThrow(() => controller.decode(JSON.stringify({...fixture.binding, [field]: value})), `${field}=${value}`);
    }
    for (const value of fixture.invalid_timestamps) {
      assert.throws(() => controller.decode(JSON.stringify({...fixture.binding, [field]: value})), `${field}=${value}`);
    }
  }
});
