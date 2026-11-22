const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const serverRoot = path.resolve(__dirname, '..');
const publicRoot = path.join(serverRoot, 'public');
const guides = [
  { slug: 'coup', type: 'coup', title: 'Play Coup Online with Friends | Time Crushers' },
  { slug: 'ludo', type: 'ludo', title: 'Play Ludo Online with Friends | Time Crushers' },
  { slug: 'tic-tac-toe', type: 'tictactoe', title: 'Play Tic-Tac-Toe Online with a Friend | Time Crushers' },
  { slug: 'poker', type: 'poker', title: "Play Texas Hold'em Online with Friends | Time Crushers" },
  { slug: '20-questions', type: 'nquestions', title: 'Play 20 Questions Online with Friends | Time Crushers' },
  { slug: 'commune', type: 'commune', title: 'Play Commune Online with Friends | Time Crushers' },
  { slug: 'twenty-nine', type: 'twentynine', title: 'Play 29 Online with Friends | Time Crushers' },
  { slug: 'hearts', type: 'hearts', title: 'Play Hearts Online with Friends | Time Crushers' },
  { slug: 'uno', type: 'uno', title: 'Play UNO Online with Friends | Time Crushers' },
  { slug: 'bluff', type: 'bluff', title: 'Play Bluff Online with Friends | Time Crushers' },
  { slug: 'blackjack', type: 'blackjack', title: 'Play Blackjack Online for Free | Time Crushers' },
  { slug: 'scrabble', type: 'scrabble', title: 'Play Scrabble Online with Friends | Time Crushers' },
  { slug: 'kabo', type: 'kabo', title: 'Play Kabo Online with Friends | Time Crushers' }
];

test('Static guides have original game content, unique metadata, crawlable links and only the shared theme script', () => {
  const homepage = fs.readFileSync(path.join(serverRoot, 'client.html'), 'utf8');
  const sitemap = fs.readFileSync(path.join(publicRoot, 'sitemap.xml'), 'utf8');
  const theme = fs.readFileSync(path.join(publicRoot, 'js/theme.js'), 'utf8');
  assert.equal(guides.length, 13);
  assert.doesNotMatch(homepage, /home-guides|home-guide-links/);
  assert.equal((homepage.match(/class="btn btn-sm room-guide-link"/g) || []).length, guides.length);
  assert.equal((sitemap.match(/<loc>/g) || []).length, guides.length + 1);
  assert.equal(new Set(guides.map(guide => guide.title)).size, guides.length);
  assert.deepEqual(fs.readdirSync(path.join(publicRoot, 'games')).sort(), guides.map(guide => guide.slug).sort());
  assert.doesNotMatch(theme, /\b(?:fetch|WebSocket|setInterval|serviceWorker)\b/);
  for (const guide of guides) {
    const url = '/games/' + guide.slug + '/';
    const html = fs.readFileSync(path.join(publicRoot, 'games', guide.slug, 'index.html'), 'utf8');
    assert.doesNotMatch(html, /home-guides/);
    assert.ok(homepage.includes('href="' + url + '"'));
    const setup = homepage.match(new RegExp('<div id="menu-' + guide.type + '"[\\s\\S]*?<div class="room-toolbar">([\\s\\S]*?)</div>'))[1];
    assert.equal((setup.match(/room-guide-link/g) || []).length, 1);
    assert.ok(setup.includes('href="' + url + '" target="_blank" rel="noopener noreferrer"'));
    assert.match(setup, /aria-label="Read [^"]+ game guide \(opens in a new tab\)">Game guide<\/a>/);
    assert.match(setup, /<button[^>]*onclick="backToGameList\(\)"[^>]*>/);
    const rulesCallbacks = {
      coup: 'toggleRules()', poker: 'togglePokerRules()', nquestions: 'toggleNQRules()',
      ludo: 'toggleLudoRules()', commune: 'toggleCommuneRules()', twentynine: 'toggleTNRules()',
      hearts: 'toggleHTRules()', uno: 'toggleUNORules()', bluff: 'toggleBluffRules()',
      blackjack: 'toggleBlackjackRules()', tictactoe: "toggleAdditionalRules('tictactoe')",
      scrabble: "toggleAdditionalRules('scrabble')", kabo: "toggleAdditionalRules('kabo')"
    };
    assert.ok(setup.includes('onclick="' + rulesCallbacks[guide.type] + '">Rules</button>'));
    assert.ok(sitemap.includes('<loc>https://timecrushers.pages.dev' + url + '</loc>'));
    assert.ok(html.includes('<title>' + guide.title + '</title>'));
    assert.match(html, /<meta name="description" content="[^"]{50,}">/);
    assert.ok(html.includes('<link rel="canonical" href="https://timecrushers.pages.dev' + url + '">'));
    assert.equal((html.match(/<h1\b/g) || []).length, 1);
    const ids = [...html.matchAll(/\bid="([^"]+)"/g)].map(match => match[1]);
    assert.equal(ids.length, new Set(ids).size);
    for (const id of ['join', 'rules', 'example', 'tips']) {
      assert.match(html, new RegExp('<section class="guide-section" id="' + id + '">[\\s\\S]*?<h2>'));
      assert.ok(html.includes('href="#' + id + '"'));
    }
    assert.ok(html.includes('href="/?game=' + guide.type + '"'));
    assert.ok(html.includes('https://hobbystudio.github.io/Website/Timecrushers/privacy_policy.html'));
    const scripts = [...html.matchAll(/<script\b[^>]*src="([^"]+)"[^>]*>/g)].map(match => match[1]);
    assert.deepEqual(scripts, ['/js/theme.js']);
    assert.doesNotMatch(html, /display:\s*none|app-loader|\/api\/|googlesyndication|<script>/);
    assert.match(html, /<body class="guide-page">/);
    assert.match(html, /defer onload="initHomeTheme\(\)"/);
    for (const [, reference] of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
      const link = new URL(reference, 'https://timecrushers.pages.dev' + url);
      if (link.origin !== 'https://timecrushers.pages.dev') continue;
      let target;
      if (link.pathname === '/') target = path.join(serverRoot, 'client.html');
      else if (link.pathname.endsWith('/')) target = path.join(publicRoot, link.pathname, 'index.html');
      else target = path.join(publicRoot, link.pathname);
      assert.ok(fs.existsSync(target), reference);
      if (link.hash) {
        const content = fs.readFileSync(target, 'utf8');
        assert.ok(content.includes('id="' + link.hash.slice(1) + '"'), 'Missing fragment: ' + reference);
      }
    }
  }
});

test('Guide rules cover implementation-specific choices rather than a generic rules template', () => {
  const coup = fs.readFileSync(path.join(publicRoot, 'games/coup/index.html'), 'utf8');
  assert.match(coup, /two coins and two hidden influence cards/);
  assert.match(coup, /ten or more coins/);
  assert.match(coup, /<strong>Inquire<\/strong>/);
  assert.match(coup, /<strong>Interrogate<\/strong>/);
  assert.match(coup, /not that Mira still has a Duke/);
  const ludo = fs.readFileSync(path.join(publicRoot, 'games/ludo/index.html'), 'utf8');
  assert.match(ludo, /rolling six, making a capture, or finishing a token/);
  assert.match(ludo, /Earlier moves from that turn are not undone/);
  assert.match(ludo, /do not create a blockade/);
  assert.match(ludo, /first player to finish all four tokens/);
  assert.match(ludo, /automatically/);
  const ttt = fs.readFileSync(path.join(publicRoot, 'games/tic-tac-toe/index.html'), 'utf8');
  assert.match(ttt, /host plays X and the guest plays O/);
  assert.match(ttt, /X always moves first/);
  assert.match(ttt, /does not automatically swap X and O/);
  assert.match(ttt, /wins by forfeit/);
  const checks = [
    ['poker', /raise.*total bet|total bet.*raising/i],
    ['20-questions', /full rotation of guessers/],
    ['commune', /all dealt cards/],
    ['commune', /six minus your remaining tokens/],
    ['twenty-nine', /after seven completed tricks/],
    ['twenty-nine', /original bid divided by two, rounded down/],
    ['hearts', /either a heart or the queen of spades breaks hearts/],
    ['uno', /Multi Skip/],
    ['uno', /first player to reach 500 points/],
    ['bluff', /only the most recent play/],
    ['blackjack', /all 17s, including soft 17/],
    ['blackjack', /fresh shuffled one- or two-deck shoe/],
    ['scrabble', /ENABLE lexicon/],
    ['scrabble', /without consuming the turn/]
  ];
  for (const [slug, rule] of checks) {
    assert.match(fs.readFileSync(path.join(publicRoot, 'games', slug, 'index.html'), 'utf8'), rule);
  }
  const homepage = fs.readFileSync(path.join(serverRoot, 'client.html'), 'utf8');
  assert.match(homepage, /6 minus tokens remaining/);
  assert.match(homepage, /full rotation of guessers/);
  assert.match(homepage, /until hearts or Q\u2660 have been played/);
});

test('Guides share persisted themes with the app and surface unavailable storage without contacting a backend', () => {
  const source = fs.readFileSync(path.join(publicRoot, 'js/theme.js'), 'utf8');
  const elements = new Map([
    ['screen-menu', { dataset: {} }],
    ['home-theme-toggle', { setAttribute(key, value) { this[key] = value; } }],
    ['guide-status', { textContent: '' }]
  ]);
  const values = new Map([['timecrusher_home_theme', 'dark']]);
  const errors = [];
  const context = vm.createContext({
    document: { getElementById: id => elements.get(id) || null },
    localStorage: { getItem: key => values.get(key), setItem: (key, value) => values.set(key, value) },
    console: { error: (...args) => errors.push(args) },
    fetch() { throw new Error('Guide must not contact the backend'); },
    WebSocket() { throw new Error('Guide must not connect to a game'); }
  });
  vm.runInContext(source, context);
  vm.runInContext('initHomeTheme()', context);
  assert.equal(elements.get('screen-menu').dataset.homeTheme, 'dark');
  assert.equal(elements.get('home-theme-toggle')['aria-pressed'], 'true');
  vm.runInContext('toggleHomeTheme()', context);
  assert.equal(values.get('timecrusher_home_theme'), 'light');
  vm.runInContext('localStorage.getItem = () => { throw new Error("Storage blocked"); }; initHomeTheme()', context);
  assert.match(elements.get('guide-status').textContent, /could not be loaded/);
  vm.runInContext('localStorage.setItem = () => { throw new Error("Storage blocked"); }; toggleHomeTheme()', context);
  assert.match(elements.get('guide-status').textContent, /could not save/);
  assert.equal(errors.length, 2);
});
