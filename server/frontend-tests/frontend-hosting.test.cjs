const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const vm = require('node:vm');
const { spawnSync } = require('node:child_process');

const serverRoot = path.resolve(__dirname, '..');
const configSource = fs.readFileSync(path.join(serverRoot, 'public/js/config.js'), 'utf8');

function resolveBackend(location, backendURL = '') {
  const context = vm.createContext({ URL, location });
  vm.runInContext(configSource, context);
  context.TIMECRUSHER_CONFIG.backendURL = backendURL;
  return vm.runInContext('getBackendURL()', context);
}

test('Backend configuration preserves same-origin hosting and local development', () => {
  const cases = [
    [{ protocol: 'http:', hostname: 'localhost', port: '51962', host: 'localhost:51962' }, 'http://localhost:8080'],
    [{ protocol: 'http:', hostname: '127.0.0.1', port: '8080', host: '127.0.0.1:8080' }, 'http://127.0.0.1:8080'],
    [{ protocol: 'https:', hostname: 'games.example.com', port: '', host: 'games.example.com' }, 'https://games.example.com'],
    [{ protocol: 'http:', hostname: '192.168.1.4', port: '8080', host: '192.168.1.4:8080' }, 'http://192.168.1.4:8080']
  ];
  for (const [location, expected] of cases) assert.equal(resolveBackend(location).origin, expected);
  assert.equal(resolveBackend(cases[2][0], 'https://backend.example.com:8443/').origin, 'https://backend.example.com:8443');
  assert.equal(resolveBackend(cases[0][0], 'http://localhost:8090').origin, 'http://localhost:8090');
});

test('Invalid and mixed-content backend configurations fail explicitly', () => {
  const location = { protocol: 'https:', hostname: 'timecrushers.pages.dev', port: '', host: 'timecrushers.pages.dev' };
  for (const backend of ['http://backend.example.com', 'wss://backend.example.com', '/api',
    'https://user:secret@backend.example.com', 'https://backend.example.com/api',
    'https://backend.example.com?key=value', 'https://backend.example.com#fragment']) {
    assert.throws(() => resolveBackend(location, backend), undefined, backend);
  }
});

test('Pages build publishes complete static assets, replaces only build config and rejects invalid inputs', t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'timecrusher-pages-test-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.copyFileSync(path.join(serverRoot, 'build-frontend.cjs'), path.join(root, 'build-frontend.cjs'));
  fs.copyFileSync(path.join(serverRoot, 'client.html'), path.join(root, 'client.html'));
  fs.cpSync(path.join(serverRoot, 'public'), path.join(root, 'public'), { recursive: true });
  const env = { ...process.env };
  delete env.BACKEND_URL;
  const build = environment => spawnSync(process.execPath, [path.join(root, 'build-frontend.cjs')], {
    env: environment, encoding: 'utf8'
  });
  const first = build(env);
  assert.equal(first.status, 0, first.stderr);
  const output = path.join(root, 'dist');
  const html = fs.readFileSync(path.join(output, 'index.html'), 'utf8');
  assert.equal(html, fs.readFileSync(path.join(root, 'client.html'), 'utf8'));
  const head = html.match(/<head>([\s\S]*?)<\/head>/)[1];
  assert.equal((html.match(/name="google-adsense-account"/g) || []).length, 1);
  assert.ok(head.includes('<meta name="google-adsense-account" content="ca-pub-2847092601328775">'));
  assert.equal((html.match(/name="google-site-verification"/g) || []).length, 1);
  assert.ok(head.includes('<meta name="google-site-verification" content="CsEBsvodVkwPHxb2Tx7tUOZIN0hGEMG_VE8N21X1ljM">'));
  assert.match(head, /<title>Time Crushers - Free Multiplayer Card &amp; Board Games<\/title>/);
  assert.match(head, /<meta name="description" content="Play free multiplayer card, board, and party games on Time Crushers\./);
  assert.equal((head.match(/rel="canonical"/g) || []).length, 1);
  assert.ok(head.includes('<link rel="canonical" href="https://timecrushers.pages.dev/">'));
  assert.match(html, /class="home-hero-description">Time Crushers/);
  assert.doesNotMatch(html, /<script\b[^>]*src="[^"]*(?:googlesyndication\.com|adsbygoogle)/i);
  const robots = fs.readFileSync(path.join(output, 'robots.txt'), 'utf8');
  assert.match(robots, /^User-agent: \*\r?\nAllow: \//);
  assert.match(robots, /Sitemap: https:\/\/timecrushers\.pages\.dev\/sitemap\.xml/);
  const sitemap = fs.readFileSync(path.join(output, 'sitemap.xml'), 'utf8');
  assert.match(sitemap, /<urlset xmlns="http:\/\/www\.sitemaps\.org\/schemas\/sitemap\/0\.9">/);
  const locations = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map(match => match[1]);
  assert.deepEqual(locations, [
    'https://timecrushers.pages.dev/',
    'https://timecrushers.pages.dev/games/coup/',
    'https://timecrushers.pages.dev/games/ludo/',
    'https://timecrushers.pages.dev/games/tic-tac-toe/',
    'https://timecrushers.pages.dev/games/poker/',
    'https://timecrushers.pages.dev/games/20-questions/',
    'https://timecrushers.pages.dev/games/commune/',
    'https://timecrushers.pages.dev/games/twenty-nine/',
    'https://timecrushers.pages.dev/games/hearts/',
    'https://timecrushers.pages.dev/games/uno/',
    'https://timecrushers.pages.dev/games/bluff/',
    'https://timecrushers.pages.dev/games/blackjack/',
    'https://timecrushers.pages.dev/games/scrabble/',
    'https://timecrushers.pages.dev/games/kabo/'
  ]);
  for (const [, asset] of html.matchAll(/(?:src|href)="(\/[^"#?]+)"/g)) {
    assert.ok(fs.existsSync(path.join(output, asset)), 'Missing published asset: ' + asset);
  }
  assert.ok(html.indexOf('src="/js/config.js"') < html.indexOf('src="/js/common.js"'));
  const configPath = path.join(output, 'js', 'config.js');
  assert.match(fs.readFileSync(configPath, 'utf8'), /backendURL":"https:\/\/timecrushers\.politeflower-423ed73a\.eastus\.azurecontainerapps\.io"/);
  for (const file of ['main.go', 'go.mod', 'server.exe', 'Dockerfile', 'frontend-tests', 'client.html']) {
    assert.equal(fs.existsSync(path.join(output, file)), false, file);
  }
  const headers = fs.readFileSync(path.join(output, '_headers'), 'utf8');
  for (const route of ['/', '/index.html', '/games/*', '/js/*', '/css/*', '/sw.js']) {
    assert.ok(headers.includes(route + '\n  Cache-Control: no-cache, no-store, must-revalidate'));
  }
  fs.writeFileSync(path.join(output, 'stale.txt'), 'Old deployment asset');
  const custom = build({ ...env, BACKEND_URL: 'https://backend.example.com:8443/' });
  assert.equal(custom.status, 0, custom.stderr);
  assert.equal(fs.existsSync(path.join(output, 'stale.txt')), false);
  const generated = fs.readFileSync(configPath, 'utf8');
  const context = vm.createContext({ URL, location: { protocol: 'https:', host: 'timecrushers.pages.dev', hostname: 'timecrushers.pages.dev' } });
  vm.runInContext(generated, context);
  assert.equal(vm.runInContext('getBackendURL().origin', context), 'https://backend.example.com:8443');
  assert.equal(fs.readFileSync(path.join(root, 'public/js/config.js'), 'utf8'), configSource);
  for (const value of ['', '/api', 'http://localhost:8080', 'wss://backend.example.com',
    'https://user:secret@backend.example.com', 'https://backend.example.com/api',
    'https://backend.example.com?q=1', 'https://backend.example.com#fragment']) {
    const invalid = build({ ...env, BACKEND_URL: value });
    assert.notEqual(invalid.status, 0, value);
    assert.match(invalid.stderr, /Frontend build failed:/);
    assert.equal(fs.readFileSync(configPath, 'utf8'), generated);
  }
  fs.writeFileSync(path.join(root, 'public/js/config.js'), '');
  const missing = build(env);
  assert.notEqual(missing.status, 0);
  assert.match(missing.stderr, /config placeholder is missing/);
});

test('Service worker never intercepts backend, code or non-GET requests, but still caches artwork', async () => {
  const handlers = new Map();
  const context = vm.createContext({
    URL,
    self: {
      location: { origin: 'https://timecrushers.pages.dev' },
      addEventListener: (name, callback) => handlers.set(name, callback)
    },
    caches: { match: async () => undefined, open: async () => ({ put() {} }) },
    fetch: async () => ({ ok: true, clone() { return this; } })
  });
  vm.runInContext(fs.readFileSync(path.join(serverRoot, 'public/sw.js'), 'utf8'), context);
  const dispatch = (url, method = 'GET') => {
    const event = { request: { url, method }, respondWith(response) { this.response = response; } };
    handlers.get('fetch')(event);
    return event;
  };
  for (const url of [
    'https://backend.example.com/api/room-stats',
    'https://backend.example.com/textures/duke.jpg',
    'https://timecrushers.pages.dev/',
    'https://timecrushers.pages.dev/api/public-rooms',
    'https://timecrushers.pages.dev/ws',
    'https://timecrushers.pages.dev/js/config.js',
    'https://timecrushers.pages.dev/css/home.css',
    'https://timecrushers.pages.dev/games/coup/',
    'https://timecrushers.pages.dev/sitemap.xml',
    'https://timecrushers.pages.dev/robots.txt'
  ]) assert.equal(dispatch(url).response, undefined, url);
  assert.equal(dispatch('https://timecrushers.pages.dev/icons/icon-192.png', 'POST').response, undefined);
  assert.equal((await dispatch('https://timecrushers.pages.dev/icons/icon-192.png').response).ok, true);
});
