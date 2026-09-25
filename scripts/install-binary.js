const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const REPO = 'jalioba/GoClean';
const VERSION = '1.0.0'; // Updated automatically during release

function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  let osName = '';
  switch (platform) {
    case 'win32':
      osName = 'windows';
      break;
    case 'darwin':
      osName = 'darwin';
      break;
    case 'linux':
      osName = 'linux';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  let archName = '';
  switch (arch) {
    case 'x64':
      archName = 'amd64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  const isWindows = platform === 'win32';
  const ext = isWindows ? '.zip' : '.tar.gz';
  const binaryName = isWindows ? 'goclean.exe' : 'goclean';
  const archiveName = `goclean_${VERSION}_${osName}_${archName}${ext}`;

  return { osName, archName, isWindows, binaryName, archiveName };
}

function downloadFile(url, destPath) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      // Follow redirects
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return downloadFile(res.headers.location, destPath).then(resolve).catch(reject);
      }

      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed with HTTP status ${res.statusCode} from ${url}`));
      }

      const file = fs.createWriteStream(destPath);
      res.pipe(file);
      file.on('finish', () => file.close(resolve));
      file.on('error', (err) => {
        fs.unlink(destPath, () => {});
        reject(err);
      });
    }).on('error', reject);
  });
}

async function main() {
  const { isWindows, binaryName, archiveName } = getPlatformInfo();
  const binDir = path.join(__dirname, '..', 'bin');
  const targetBinary = path.join(binDir, binaryName);

  if (fs.existsSync(targetBinary)) {
    return;
  }

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  const tempArchive = path.join(binDir, archiveName);
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${archiveName}`;

  console.log(`[goclean] Downloading prebuilt binary: ${archiveName}...`);
  try {
    await downloadFile(downloadUrl, tempArchive);
  } catch (err) {
    console.warn(`[goclean] Failed to download prebuilt binary (${err.message}).`);
    console.warn(`[goclean] Please install GoClean manually or check release availability.`);
    return;
  }

  console.log(`[goclean] Extracting binary to ${binDir}...`);
  try {
    execSync(`tar -xf "${tempArchive}" -C "${binDir}"`, { stdio: 'ignore' });
  } catch (err) {
    console.warn(`[goclean] Extraction failed: ${err.message}`);
  } finally {
    if (fs.existsSync(tempArchive)) {
      fs.unlinkSync(tempArchive);
    }
  }

  if (fs.existsSync(targetBinary)) {
    if (!isWindows) {
      fs.chmodSync(targetBinary, 0o755);
    }
    console.log(`[goclean] Successfully installed GoClean!`);
  }
}

if (require.main === module) {
  main().catch((err) => {
    console.error(`[goclean] Installation failed: ${err.message}`);
  });
}

module.exports = main;
