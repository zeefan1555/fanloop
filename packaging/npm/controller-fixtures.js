'use strict';
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');
const controller = require('./scripts/controller');
const fixtures = JSON.parse(fs.readFileSync(path.join(__dirname,'../../internal/controller/testdata/resources.json'),'utf8'));
const roots = JSON.parse(fs.readFileSync(process.argv[2],'utf8'));
for (const fixture of fixtures) {
  const args = ['flow','status','--root',roots[fixture.name]];
  if (fixture.valid) {
    assert.ok(controller.select(args),fixture.name);
  } else {
    assert.throws(() => controller.select(args),fixture.name);
  }
}
