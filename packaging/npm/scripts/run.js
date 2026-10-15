#!/usr/bin/env node
'use strict';

const {spawn} = require('child_process');
const path = require('path');
const layout = require('./layout');

const binary = path.join(layout.dataHome(), 'current', 'bin', 'fanloop');
const child = spawn(binary, process.argv.slice(2), {stdio: 'inherit'});
child.on('error', (err) => {
  console.error(`fanloop: cannot execute ${binary}: ${err.message}`);
  process.exit(1);
});
child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code === null ? 1 : code);
});
