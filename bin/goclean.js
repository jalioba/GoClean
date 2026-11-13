#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const ext = process.platform === 'win32' ? '.exe' : '';
const binaryPath = path.join(__dirname, 'goclean' + ext);

async function run() {
  if (!fs.existsSync(binaryPath)) {
    console.log('⚡ GoClean binary not found locally, downloading prebuilt release...');
    try {
      const installBinary = require('../scripts/install-binary.js');
      await installBinary();
    } catch (err) {
      console.error('❌ Failed to download GoClean binary automatically:', err.message);
      process.exit(1);
    }
    if (!fs.existsSync(binaryPath)) {
      console.error('❌ GoClean binary is still missing after installation attempt.');
      process.exit(1);
    }
  }

  const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: false
  });

  child.on('error', (err) => {
    console.error('❌ Failed to start GoClean process:', err.message);
    process.exit(1);
  });

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code || 0);
    }
  });
}

run();
