const fs = require('node:fs');
const path = require('node:path');

const defaultBackend = 'https://timecrushers.politeflower-423ed73a.eastus.azurecontainerapps.io';

function build() {
  const backend = new URL(process.env.BACKEND_URL ?? defaultBackend);
  if (backend.protocol !== 'https:' || backend.username || backend.password ||
      backend.pathname !== '/' || backend.search || backend.hash) {
    throw new Error('BACKEND_URL must be an HTTPS origin without credentials, a path, a query, or a fragment.');
  }

  const config = fs.readFileSync(path.join(__dirname, 'public', 'js', 'config.js'), 'utf8');
  const configPlaceholder = "globalThis.TIMECRUSHER_CONFIG = { backendURL: '' };";
  if (!config.includes(configPlaceholder)) {
    throw new Error('The frontend config placeholder is missing.');
  }
  const output = path.join(__dirname, 'dist');
  fs.rmSync(output, { recursive: true, force: true });
  fs.cpSync(path.join(__dirname, 'public'), output, { recursive: true });
  fs.copyFileSync(path.join(__dirname, 'client.html'), path.join(output, 'index.html'));
  const configPath = path.join(output, 'js', 'config.js');
  fs.writeFileSync(configPath, config.replace(
    configPlaceholder,
    'globalThis.TIMECRUSHER_CONFIG = ' + JSON.stringify({ backendURL: backend.origin }) + ';'
  ));
  fs.writeFileSync(path.join(output, '_headers'), [
    '/',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    '/index.html',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    '/games/*',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    '/js/*',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    '/css/*',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    '/sw.js',
    '  Cache-Control: no-cache, no-store, must-revalidate',
    ''
  ].join('\n'));
  console.log('Cloudflare Pages frontend built in ' + output);
  console.log('Backend: ' + backend.origin);
}

try {
  build();
} catch (error) {
  console.error('Frontend build failed:', error.message);
  process.exitCode = 1;
}
