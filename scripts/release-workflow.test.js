const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { it } = require("node:test");

const workflow = fs.readFileSync(path.resolve(__dirname, "../.github/workflows/release.yml"), "utf8");

it("publishes every main update without a separate manual trigger", () => {
  assert.match(workflow, /^on:\n  push:\n    branches:\n      - main$/m);
  assert.doesNotMatch(workflow, /workflow_dispatch/);
  assert.doesNotMatch(workflow, /if: github\.ref/);
});
