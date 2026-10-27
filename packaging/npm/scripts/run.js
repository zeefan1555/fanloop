#!/usr/bin/env node
'use strict';

const {spawn} = require('child_process');
const path = require('path');
const layout = require('./layout');

const controller = require('./controller');
let selection;
try {
  selection = controller.select(process.argv.slice(2));
} catch (err) {
  const request = controller.invocation(process.argv.slice(2));
  const failure = JSON.parse(JSON.stringify(require('./commands.json').internal_error));
  failure.envelope.error.message = err.message;
  failure.envelope.meta.command = request.command;
  failure.envelope.meta.requirement_root = request.root;
  if (request.dryRun) failure.envelope.meta.dry_run = true;
  process.stderr.write(JSON.stringify(failure.envelope, null, 2)+'\n');
  process.exit(failure.exit_code);
}
const binary = selection ? selection.binary : path.join(layout.dataHome(), 'current', 'bin', 'fanloop');
const child = spawn(binary, process.argv.slice(2), {stdio: 'inherit', env: selection ? selection.env : process.env});
for (const signal of ['SIGINT','SIGTERM','SIGHUP']) process.on(signal, () => child.kill(signal));
child.on('error', (err) => {
  console.error(`fanloop: cannot execute ${binary}: ${err.message}`);
  process.exit(1);
});
child.on('exit', (code, signal) => {
  if (signal) {
    process.removeAllListeners(signal);
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code === null ? 1 : code);
});
