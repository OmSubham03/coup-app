const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const serverRoot = path.resolve(__dirname, '..');
const html = fs.readFileSync(path.join(serverRoot, 'client.html'), 'utf8');

function emptyRoomStats() {
  return { players: 0, games: Object.fromEntries([
    'coup', 'poker', 'ludo', 'nquestions', 'commune', 'twentynine',
    'hearts', 'uno', 'bluff', 'blackjack', 'tictactoe', 'scrabble', 'kabo'
  ].map(type => [type, { players: 0, publicRooms: 0 }])) };
}

function harness(backendURL = '', options = {}) {
  const elements = new Map();
  const sent = [];
  const errors = [];
  const requests = [];
  function element() {
    const classes = new Set();
    return {
      style: {}, dataset: {}, value: '', checked: false, disabled: false, hidden: false,
      textContent: '', innerHTML: '', open: false, children: [],
      classList: {
        add: (...names) => names.forEach(name => classes.add(name)),
        remove: (...names) => names.forEach(name => classes.delete(name)),
        contains: name => classes.has(name),
        toggle(name, force) {
          const enabled = force === undefined ? !classes.has(name) : force;
          if (enabled) classes.add(name); else classes.delete(name);
          return enabled;
        }
      },
      setAttribute(name, value) { this[name] = value; },
      addEventListener(name, callback) { this['on' + name] = callback; },
      showModal() { this.open = true; }, close() { this.open = false; }, focus() {},
      append(...children) { this.children.push(...children); },
      replaceChildren(...children) { this.children = children; },
      querySelector(selector) {
        return this.children.find(child => child.className === selector.slice(1)) || null;
      },
      parentElement: { title: '', setAttribute(name, value) { this[name] = value; } }
    };
  }
  for (const match of html.matchAll(/\bid="([^"]+)"/g)) elements.set(match[1], element());
  for (const match of html.matchAll(/<[^>]+\bid="([^"]+)"[^>]*>/g)) {
    const display = match[0].match(/style="[^"]*display:([^;" ]+)/);
    if (display) elements.get(match[1]).style.display = display[1];
  }
  const storage = () => {
    const values = new Map();
    return { getItem: key => values.get(key) || null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  };
  const document = {
    getElementById(id) {
      if (id === 'guide-status') return null;
      assert.ok(elements.has(id), 'Missing HTML element: ' + id);
      return elements.get(id);
    },
    createElement() {
      const node = element();
      Object.defineProperty(node, 'innerHTML', { get: () => String(node.textContent).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;') });
      return node;
    },
    addEventListener() {},
    querySelectorAll(selector) {
      if (selector === '.screen') return [...elements].filter(([id]) => id.startsWith('screen-')).map(([, node]) => node);
      if (selector === '#bluff-hand [data-card-id]') {
        const hand = elements.get('bluff-hand');
        if (hand.lastHTML !== hand.innerHTML) {
          hand.lastHTML = hand.innerHTML;
          hand.buttons = [...hand.innerHTML.matchAll(/data-card-id="([^"]+)"/g)].map(match => {
            const button = element();
            button.dataset.cardId = match[1];
            return button;
          });
        }
        return hand.buttons || [];
      }
      return [];
    }
  };
  class Socket {
    static OPEN = 1;
    constructor(url) { this.url = url; this.readyState = 1; }
    send(message) { sent.push(JSON.parse(message)); }
    close() { this.readyState = 3; }
  }
  const context = vm.createContext({
    document, sessionStorage: storage(), localStorage: storage(),
    location: { port: '8080', hostname: 'localhost', host: 'localhost:8080', protocol: 'http:', search: options.search || '' },
    crypto: { randomUUID: () => 'test-player' }, WebSocket: Socket,
    URL, URLSearchParams, AbortController, navigator: {},
    console: { log() {}, error: (...args) => errors.push(args) },
    setTimeout: () => 1, clearTimeout() {}, setInterval: () => 1, clearInterval() {},
    alert() {}, confirm: () => true,
    fetch: async url => {
      requests.push(url);
      return { ok: true, json: async () => url.includes('/api/public-rooms?') ? [] :
        url.endsWith('/api/room-stats') ? (options.stats || emptyRoomStats()) : ({ code: 'ABCDE' }) };
    }
  });
  context.localStorage.setItem('coup_name', 'Tester');
  for (const [key, value] of Object.entries(options.session || {})) context.sessionStorage.setItem(key, value);
  function run(code) { return vm.runInContext(code, context); }
  run('let ludoState = null, nqState = null, communeState = null, tnState = null, htState = null, unoState = null; function updateVoiceUI() {} function renderGame() { document.getElementById("game-active").style.display = ""; }');
  run(fs.readFileSync(path.join(serverRoot, 'public/js/config.js'), 'utf8'));
  context.TIMECRUSHER_CONFIG.backendURL = backendURL;
  for (const name of ['theme', 'avatars', 'common', 'bluff', 'blackjack', 'tictactoe', 'scrabble', 'kabo']) run(fs.readFileSync(path.join(serverRoot, 'public/js', name + '.js'), 'utf8'));
  run('ws = new WebSocket("test"); hostId = playerId;');
  const message = (type, payload) => run('handleWSMessage({data:' + JSON.stringify(JSON.stringify({ type, payload })) + '})');
  return { run, message, elements, sent, errors, document, requests };
}

const card = (rank, id = 'card-' + rank, suit = 'spades') => ({ rank, id, suit });
function bluffFixture(phase = 'playing') {
  return { phase, currentPlayerIdx: 0, requiredRank: 1, pile: [{ rank: 0 }], pendingActorId: '', twoDecks: true,
    players: [{ id: 'test-player', name: '<Tester>', active: true, cards: [card(2), card(3), card(4), card(5), card(6)] },
      { id: 'other', name: 'Other', active: true, cards: [{ rank: 0 }] },
      { id: 'third', name: 'Third', active: true, cards: [{ rank: 0 }] }] };
}
function blackjackFixture(phase = 'playing') {
  return { phase, currentPlayerIdx: 0, round: 1, twoDecks: false, dealerCards: [card(1), { rank: 0, id: '', suit: '' }],
    players: [{ id: 'test-player', name: 'Tester', active: true, status: 'active', cards: [card(1), card(6)], chips: 950, bet: 50 }] };
}

test('Blackjack action buttons are hidden outside the local turn', () => {
  const app = harness();
  const state = blackjackFixture();
  state.players.push({ ...state.players[0], id: 'other', name: 'Other' });
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-turn-actions').hidden, false);
  state.currentPlayerIdx = 1;
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-turn-actions').hidden, true);
  state.currentPlayerIdx = 0;
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-turn-actions').hidden, false);
  state.players[0].status = 'stood';
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-turn-actions').hidden, true);
  state.phase = 'round_over';
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-turn-actions').hidden, true);
});

test('HTML readiness count, unique IDs, callbacks, assets, and default deck options', () => {
  const ids = [...html.matchAll(/\bid="([^"]+)"/g)].map(match => match[1]);
  assert.equal(new Set(ids).size, ids.length);
  const scripts = [...html.matchAll(/<script src="([^"]+)" defer onload="checkScriptsLoaded\(\)"><\/script>/g)];
  assert.equal(scripts.length, Number(html.match(/let totalScripts = (\d+)/)[1]));
  assert.equal(scripts.length, 18);
  for (const script of scripts) assert.ok(fs.existsSync(path.join(serverRoot, 'public', script[1])));
  for (const match of html.matchAll(/\bon(?:click|input|keydown|change|load)="([^"]*)"/g)) new Function('event', match[1]);
  for (const type of ['bluff', 'blackjack']) assert.ok(!new RegExp('id="' + type + '-two-decks"[^>]*checked').test(html));
  assert.ok(fs.existsSync(path.join(serverRoot, 'public/textures/card-back.svg')));
});

test('Homepage footer links to the Studio-hosted policy and central contact section', () => {
  const footer = html.match(/<footer class="site-footer">([\s\S]*?)<\/footer>/)[1];
  const links = [
    ['https://hobbystudio.github.io/Website/Timecrushers/privacy_policy.html', 'Privacy Policy'],
    ['https://hobbystudio.github.io/Website/#contact', 'Contact Us']
  ];
  assert.match(footer, /<nav class="site-footer-links" aria-label="Information and support">/);
  for (const [url, label] of links) {
    assert.ok(footer.includes(`<a href="${url}" target="_blank" rel="noopener noreferrer">${label}</a>`));
  }
  assert.match(footer, /Made with play by Hobby Studio/);
});

test('Card activity hides each zero value while homepage and game setups display zero live', async () => {
  const stats = emptyRoomStats();
  stats.games.coup = { players: 5, publicRooms: 2 };
  stats.games.ludo = { players: 3, publicRooms: 0 };
  stats.players = 8;
  const app = harness('', { stats });
  await app.run('loadRoomPlayerCount()');
  assert.equal(app.elements.get('room-player-total').textContent, 8);
  assert.equal(app.elements.get('coup-card-rooms').textContent, '2 rooms');
  assert.equal(app.elements.get('coup-card-rooms').hidden, false);
  assert.equal(app.elements.get('coup-card-live').textContent, '5 live');
  assert.equal(app.elements.get('coup-card-live').hidden, false);
  assert.equal(app.elements.get('coup-card-activity').hidden, false);
  assert.equal(app.elements.get('ludo-card-rooms').hidden, true);
  assert.equal(app.elements.get('ludo-card-live').textContent, '3 live');
  assert.equal(app.elements.get('ludo-card-live').hidden, false);
  for (const type of Object.keys(stats.games)) {
    assert.equal(app.elements.get(type + '-setup-live').textContent, stats.games[type].players + ' live');
    if (stats.games[type].players === 0) {
      assert.equal(app.elements.get(type + '-card-activity').hidden, true);
      assert.equal(app.elements.get(type + '-card-live').hidden, true);
      assert.equal(app.elements.get(type + '-card-rooms').hidden, true);
    }
  }
  app.run('renderRoomStats(' + JSON.stringify(emptyRoomStats()) + ')');
  assert.equal(app.elements.get('room-player-total').textContent, 0);
  assert.equal(app.elements.get('coup-setup-live').textContent, '0 live');
  assert.equal(app.elements.get('coup-card-activity').hidden, true);
  assert.match(html, /id="room-player-total">--<\/strong><span>live<\/span>/);
  assert.deepEqual(app.errors, []);
});

test('Activity stats share an in-flight request and refresh once per setup open, without a timer', async () => {
  const app = harness();
  const before = app.requests.filter(url => url.endsWith('/api/room-stats')).length;
  await app.run('Promise.all([loadRoomPlayerCount(), loadRoomPlayerCount()])');
  assert.equal(app.requests.filter(url => url.endsWith('/api/room-stats')).length, before);
  app.run('setInterval = () => { throw new Error("No activity polling timer expected"); }; startPublicRoomDiscovery = () => {}');
  app.run("openGameMenu('ludo')");
  await app.run('loadRoomPlayerCount()');
  assert.equal(app.requests.filter(url => url.endsWith('/api/room-stats')).length, before + 1);
  app.run('backToGameList()');
  assert.equal(app.requests.filter(url => url.endsWith('/api/room-stats')).length, before + 1);
  assert.deepEqual(app.errors, []);
});

test('Activity fetch failures and invalid counts surface unavailable states rather than false zeros', async () => {
  const app = harness();
  await app.run('loadRoomPlayerCount()');
  app.run('renderRoomStats(' + JSON.stringify({ ...emptyRoomStats(), players: 1, games: {
    ...emptyRoomStats().games, coup: { players: 1, publicRooms: 1 }
  } }) + ')');
  const invalid = [
    null, { players: 0 }, { ...emptyRoomStats(), players: -1 },
    { ...emptyRoomStats(), players: 1 },
    { ...emptyRoomStats(), games: { ...emptyRoomStats().games, coup: { players: 0, publicRooms: 1 } } },
    { ...emptyRoomStats(), games: { ...emptyRoomStats().games, coup: { players: 1.5, publicRooms: 0 } } }
  ];
  for (const value of invalid) {
    app.run('fetch = async () => ({ok:true, json:async () => (' + JSON.stringify(value) + ')})');
    await app.run('loadRoomPlayerCount()');
    assert.equal(app.elements.get('room-player-total').textContent, '--');
    assert.equal(app.elements.get('coup-setup-live').textContent, '-- live');
    assert.equal(app.elements.get('coup-card-activity').hidden, true);
  }
  app.run('fetch = async () => ({ok:false})');
  await app.run('loadRoomPlayerCount()');
  assert.match(app.elements.get('coup-setup-live').title, /unavailable/);
  app.run('fetch = async () => {throw new Error("Offline")};');
  await app.run('loadRoomPlayerCount()');
  assert.equal(app.errors.length, invalid.length + 2);
});

test('Every game has its own card badges and setup live total without replacing guide or Rules controls', () => {
  for (const type of Object.keys(emptyRoomStats().games)) {
    assert.match(html, new RegExp('id="' + type + '-card-activity" hidden><span id="' + type + '-card-rooms" hidden></span><span id="' + type + '-card-live" hidden></span>'));
    const toolbar = html.match(new RegExp('<div id="menu-' + type + '"[\\s\\S]*?<div class="room-toolbar">([\\s\\S]*?)</div>'))[1];
    assert.ok(toolbar.includes('id="' + type + '-setup-live"'));
    assert.ok(toolbar.indexOf('id="' + type + '-setup-live"') < toolbar.indexOf('room-guide-link'));
    assert.match(toolbar, /room-guide-link/);
    assert.match(toolbar, />Rules<\/button>/);
  }
  const css = fs.readFileSync(path.join(serverRoot, 'public/css/rooms.css'), 'utf8');
  assert.match(css, /@media \(max-width: 640px\) \{[\s\S]*?\.room-setup \.room-live-count \{ display: none; \}/);
  assert.match(css, /\.room-setup \.room-guide-link \{ margin-left: auto; \}/);
});

test('Every Rules surface uses the shared theme panel without inline game colors', () => {
  const overlays = ['rules-overlay', 'poker-rules-overlay', 'ludo-rules-overlay', 'tn-rules-overlay',
    'ht-rules-overlay', 'uno-rules-overlay', 'commune-rules-overlay', 'nq-rules-overlay'];
  for (const id of overlays) {
    assert.match(html, new RegExp('<div class="rules-overlay" id="' + id + '">\\s*<div class="[^"]*rules-panel">'));
  }
  for (const id of ['bluff-rules-dialog', 'blackjack-rules-dialog', 'tictactoe-rules-dialog', 'scrabble-rules-dialog', 'uno-house-rule-dialog', 'kabo-rules-dialog', 'kabo-confirm-dialog']) {
    assert.match(html, new RegExp('<dialog id="' + id + '" class="[^"]*rules-panel"'));
  }
  assert.equal((html.match(/\brules-panel\b/g) || []).length, 15);
  assert.doesNotMatch(html, /class="[^"]*rules-content[^"]*" style="/);
  assert.doesNotMatch(html, /onclick="toggle[^"]*Rules[^"]*" style="/);
  const css = fs.readFileSync(path.join(serverRoot, 'public/css/rooms.css'), 'utf8');
  assert.match(css, /\.rules-panel \{[^}]*background: var\(--home-surface\);[^}]*color: var\(--home-ink\)/);
  assert.match(css, /\.rules-panel h3 \{[^}]*color: var\(--home-green\)/);
  assert.match(css, /\.rules-panel :is\(p,li\) \{[^}]*color: var\(--home-muted\)/);
  assert.match(css, /body:has\(#screen-menu\[data-home-theme="dark"\]\) \.rules-panel \{ color-scheme: dark; \}/);
});

test('In-game Settings Rules opens the same game rule handler and closes Settings first', () => {
  const app = harness();
  app.run('globalThis.openedRules = []');
  const callbacks = {
    coup: 'toggleRules', poker: 'togglePokerRules', ludo: 'toggleLudoRules',
    nquestions: 'toggleNQRules', commune: 'toggleCommuneRules', twentynine: 'toggleTNRules',
    hearts: 'toggleHTRules', uno: 'toggleUNORules', bluff: 'toggleBluffRules', blackjack: 'toggleBlackjackRules'
  };
  for (const [type, callback] of Object.entries(callbacks)) {
    app.run(callback + ' = () => openedRules.push("' + type + '")');
  }
  app.run('toggleAdditionalRules = type => openedRules.push(type)');
  for (const type of Object.keys(emptyRoomStats().games)) {
    app.elements.get('game-settings-dialog').open = true;
    app.run('currentGameType = "' + type + '"; openSettingsRules()');
    assert.equal(app.run('openedRules.at(-1)'), type);
    assert.equal(app.elements.get('game-settings-dialog').open, false);
  }
});

test('Guide Play now queries open the correct setup without creating a room', async () => {
  for (const game of ['coup', 'ludo', 'tictactoe', 'poker', 'nquestions', 'commune', 'twentynine', 'hearts', 'uno', 'bluff', 'blackjack', 'scrabble', 'kabo']) {
    const app = harness('', { search: '?game=' + game });
    await app.run('Promise.resolve()');
    assert.equal(app.run('currentGameType'), game);
    assert.equal(app.elements.get('menu-' + game).style.display, '');
    assert.equal(app.elements.get('game-list-view').style.display, 'none');
    assert.equal(app.run('roomCode'), '');
    assert.equal(app.run('wsConnecting'), false);
    assert.equal(app.run('savedSessionReconnectPending'), false);
    assert.ok(app.requests.some(url => url.includes('/api/public-rooms?gameType=' + game)));
    assert.ok(!app.requests.some(url => url.includes('/api/generate-code')));
    assert.deepEqual(app.errors, []);
  }
});

function kaboFixture(phase = 'draw') {
  const card = id => ({ id, rank: 0, suit: '' });
  return { id: 'kabo-match', revision: 5, round: 1, phase, currentPlayerIdx: 0,
    players: [
      { id: 'test-player', name: '<Tester>', active: true, ready: false, totalScore: 0, cards: ['a','b','c','d'].map(card) },
      { id: 'other', name: 'Alex', active: true, ready: false, totalScore: 0, cards: ['e','f','g','h'].map(card) }
    ], discard: {rank:7,suit:'hearts'}, discardId:'discard-1', drawn:null, fromDeck:true,
    reveals: [], power:'', matchOpen:true, matchAllowed:true, callerId:'', finalTurns:0,
    lastAction:'', winners:[], roundWinners:[] };
}

test('Kabo matching selects opponent and own cards, confirms atomically, and resets stale selections', () => {
  const app = harness();
  app.message('kabo-state', kaboFixture());
  assert.match(app.elements.get('kabo-opponents').innerHTML, /data-kabo-slot="e"/);
  assert.match(app.elements.get('kabo-own-name').innerHTML, /&lt;Tester&gt;/);
  assert.ok(!app.elements.get('kabo-results').innerHTML);
  app.run('kaboChoose("other","e")');
  assert.equal(app.run('kaboCommand()'), null);
  assert.match(app.elements.get('kabo-prompt').textContent, /give Alex/);
  app.run('kaboChoose(playerId,"b"); kaboPrimary()');
  assert.equal(app.elements.get('kabo-confirm-dialog').open, true);
  assert.equal(app.sent.length, 0);
  app.run('kaboConfirmAction()');
  assert.deepEqual(app.sent.at(-1), {type:'kabo-match',payload:{targetId:'e',slotId:'b',discardId:'discard-1',revision:5}});
  assert.equal(app.run('kaboBusy'), true);
  app.message('kabo-state', {...kaboFixture(), revision:6});
  assert.equal(app.run('kaboSelected.length'),0);
  assert.equal(app.run('kaboTarget'),'');
  assert.equal(app.run('kaboBusy'),false);
  app.run('kaboChoose(playerId,"a"); kaboPrimary()');
  app.message('kabo-state', {...kaboFixture(), revision:7});
  assert.equal(app.elements.get('kabo-confirm-dialog').open,false);
  app.run('kaboConfirmAction()');
  assert.equal(app.sent.length,1);
  const empty = {...kaboFixture(),revision:8};
  empty.players[0].cards[1].empty = true;
  app.message('kabo-state',empty);
  assert.match(app.elements.get('kabo-hand').innerHTML,/Empty card position/);
  app.run('kaboChoose(playerId,"b")');
  assert.equal(app.run('kaboSelected.length'),0);
});

test('Kabo initial peek, draw decision, powers and private reveal removal use the server state', () => {
  const app=harness();
  app.message('kabo-state',kaboFixture('initial'));
  app.run('kaboChoose(playerId,"a");kaboChoose(playerId,"b");kaboPrimary()');
  assert.equal(app.sent.at(-1).type,'kabo-initial');
  const revealed={...kaboFixture('initial'),revision:6,reveals:[
    {playerId:'test-player',slotId:'a',card:{rank:13,suit:'hearts'}},
    {playerId:'test-player',slotId:'b',card:{rank:3,suit:'clubs'}}
  ]};
  app.message('kabo-state',revealed);
  assert.match(app.elements.get('kabo-hand').innerHTML,/King of hearts/);
  app.run('kaboPrimary()');
  assert.equal(app.sent.at(-1).type,'kabo-ready');
  app.message('kabo-state',{...kaboFixture(),revision:7});
  assert.doesNotMatch(app.elements.get('kabo-hand').innerHTML,/King of hearts/);
  app.message('kabo-state',{...kaboFixture('resolve'),revision:8,drawn:{rank:3,suit:'clubs'}});
  assert.match(app.elements.get('kabo-drawn').innerHTML,/3 of clubs/);
  assert.equal(app.elements.get('kabo-call').hidden,false);
  app.message('kabo-state',{...kaboFixture('power'),revision:9,power:'king'});
  app.run('kaboChoose(playerId,"a");kaboChoose("other","e");kaboPrimary()');
  assert.equal(app.sent.at(-1).type,'kabo-power');
  app.message('kabo-state',{...kaboFixture('peek'),revision:10,power:'king',reveals:revealed.reveals});
  assert.equal(app.run('kaboCommand().kind'),'ack');
  app.message('kabo-state',{...kaboFixture('decision'),revision:11,power:'king'});
  assert.equal(app.elements.get('kabo-decision').hidden,false);
  assert.doesNotMatch(app.elements.get('kabo-hand').innerHTML,/King of hearts/);
});

test('Kabo spectators, errors, round scores, host controls, chat and lobby reset are consistent', () => {
  const app=harness();
  const state=kaboFixture();
  state.players[0].id='someone-else';
  app.message('kabo-state',state);
  assert.equal(app.run('isSpectating'),true);
  app.run('kaboChoose("other","e");kaboSend("draw",{source:"deck"})');
  assert.equal(app.sent.length,0);
  const over={...kaboFixture('round_over'),revision:8,roundWinners:['test-player']};
  over.players[0].roundScore=-1;over.players[0].totalScore=-1;
  over.players[1].roundScore=3;over.players[1].totalScore=3;
  app.message('kabo-state',over);
  assert.match(app.elements.get('kabo-results').innerHTML,/-1/);
  assert.equal(app.elements.get('kabo-next-round').hidden,false);
  app.elements.get('kabo-chat-input').value='Hello';
  app.run('sendChat()');
  assert.equal(app.sent.at(-1).type,'chat');
  app.message('error',{message:'The table changed'});
  assert.equal(app.elements.get('kabo-prompt').textContent,'The table changed');
  app.message('state',null);
  assert.equal(app.run('kaboState'),null);
  assert.equal(app.elements.get('kabo-active').style.display,'none');
});

test('Saved sessions take priority over game links and keep reconnect identity and variant', () => {
  const app = harness('', { search: '?game=ludo', session: {
    coup_room: 'ABCDE', coup_joinedName: 'Tester#1234', coup_gameType: 'coup', coup_variant: 'inquisitor', coup_pid: 'saved-player'
  } });
  assert.equal(app.run('currentGameType'), 'coup');
  assert.equal(app.run('roomCode'), 'ABCDE');
  assert.equal(app.run('variant'), 'inquisitor');
  assert.equal(app.run('savedSessionReconnectPending'), true);
  assert.ok(!app.requests.some(url => url.includes('/api/public-rooms')));
  app.run('resumeSavedSessionReconnect()');
  assert.match(app.run('ws.url'), /room=ABCDE.*playerId=saved-player.*variant=inquisitor.*gameType=coup/);
  assert.equal(app.run('savedSessionReconnectPending'), false);
});

test('Invalid and duplicate game queries show a notification instead of opening a broken setup', () => {
  for (const search of ['?game=unknown', '?game=', '?game=coup&game=ludo', '?game=__proto__']) {
    const app = harness('', { search });
    assert.equal(app.elements.get('home-navigation-status').hidden, false);
    assert.match(app.elements.get('home-navigation-status').textContent, /not valid/);
    assert.equal(app.errors.length, 1);
    assert.ok(!app.requests.some(url => url.includes('/api/public-rooms')));
  }
  const plain = harness('', { search: '?utm_source=studio' });
  assert.deepEqual(plain.errors, []);
  assert.equal(plain.run('roomCode'), '');
});

test('Avatar presets, initials, storage and remote identities remain independent of display names', () => {
  const app = harness();
  assert.equal(app.run('avatarPresets.size'), 9);
  assert.equal(app.run("avatarInitials('  Alex Jones#1234')"), 'AJ');
  app.run("localStorage.setItem('tc_avatar','owl'); updateProfileBar(); send('join', {playerName:'Tester#0001'})");
  assert.equal(app.elements.get('home-profile-initials').textContent, '🦉');
  assert.deepEqual(app.sent.at(-1), {type:'join',payload:{playerName:'Tester#0001',avatar:'owl'}});
  app.message('room-visibility', {avatars:{other:'fox'}});
  assert.equal(app.run("avatarFace({id:'other',name:'Alex Jones#1234'})"), '🦊');
  app.message('room-visibility', {});
  assert.equal(app.run("avatarFace({id:'other',name:'Alex Jones#1234'})"), 'AJ');
  app.run("localStorage.removeItem('tc_avatar'); updateProfileBar()");
  assert.equal(app.elements.get('home-profile-initials').textContent, 'T');
});

test('Homepage themes persist and do not replace game screen styles', () => {
  const app = harness();
  const menu = app.elements.get('screen-menu');
  const toggle = app.elements.get('home-theme-toggle');
  assert.equal(menu.dataset.homeTheme, 'light');
  assert.equal(toggle['aria-pressed'], 'false');
  app.run('toggleHomeTheme()');
  assert.equal(menu.dataset.homeTheme, 'dark');
  assert.equal(toggle['aria-pressed'], 'true');
  for (const id of ['settings-theme-toggle', 'lobby-theme-toggle', 'home-theme-toggle']) {
    assert.equal(app.elements.get(id)['aria-pressed'], 'true');
    assert.match(html, new RegExp(`<button[^>]+id="${id}"[^>]+class="home-theme-toggle"[^>]+onclick="toggleHomeTheme\\(\\)"[^>]*><span class="theme-sun"[^<]+</span><span class="theme-moon"`));
  }
  app.run('setPlayerTheme(false)');
  assert.equal(app.elements.get('lobby-theme-toggle')['aria-pressed'], 'false');
  assert.equal(toggle['aria-pressed'], 'false');
  app.run('setPlayerTheme(true)');
  assert.equal(app.run('localStorage.getItem("timecrusher_home_theme")'), 'dark');
  app.run('applyHomeTheme("light"); initHomeTheme()');
  assert.equal(menu.dataset.homeTheme, 'dark');
  app.run('startPublicRoomDiscovery = () => {}');
  for (const type of ['coup', 'uno', 'poker', 'ludo', 'nquestions', 'commune', 'twentynine', 'hearts', 'bluff', 'blackjack', 'tictactoe', 'scrabble']) {
    app.run(`openGameMenu('${type}')`);
    assert.equal(menu.classList.contains('home-active'), false);
    assert.equal(app.elements.get('menu-' + type).style.display, '');
    app.run('backToGameList()');
    assert.equal(menu.classList.contains('home-active'), true);
    assert.equal(menu.dataset.homeTheme, 'dark');
  }
  app.run('toggleHomeTheme()');
  assert.equal(menu.dataset.homeTheme, 'light');
  app.run('localStorage.getItem = () => { throw new Error("Storage blocked"); }; initHomeTheme()');
  assert.match(app.elements.get('home-search-status').textContent, /could not be loaded/);
  app.run('localStorage.setItem = () => { throw new Error("Storage blocked"); }; toggleHomeTheme()');
  assert.equal(menu.dataset.homeTheme, 'dark');
  assert.match(app.elements.get('home-search-status').textContent, /could not save/);
  assert.equal(app.errors.length, 2);
});

test('Homepage categories and case-insensitive search intersect and reset empty results', () => {
  const app = harness();
  const cards = [...html.matchAll(/<button type="button" class="game-card [^"]*" data-home-category="([^"]+)" onclick="openGameMenu\('([^']+)'\)"/g)].map(match => ({
    dataset: { homeCategory: match[1] }, textContent: match[2], hidden: false, querySelector: () => null
  }));
  const filters = ['all', 'cards', 'board', 'party'].map(filter => ({
    dataset: { homeFilter: filter }, setAttribute(name, value) { this[name] = value; }
  }));
  assert.equal(cards.length, 13);
  cards.find(card => card.textContent === 'twentynine').dataset.homeKeywords = '29 twenty nine twentynine';
  const originalQuery = app.document.querySelectorAll;
  app.document.querySelectorAll = selector => selector === '[data-home-filter]' ? filters : selector === '#home-game-list .game-card' ? cards : originalQuery(selector);
  const visible = () => cards.filter(card => !card.hidden).map(card => card.textContent);
  app.run('setHomeFilter("board")');
  assert.deepEqual(visible(), ['ludo', 'tictactoe', 'scrabble']);
  assert.equal(filters[2]['aria-pressed'], 'true');
  assert.equal(app.elements.get('home-search-status').textContent, '3 games found.');
  app.elements.get('home-search').value = '  POKER  ';
  app.run('filterHomeGames()');
  assert.deepEqual(visible(), []);
  assert.equal(app.elements.get('home-empty').hidden, false);
  app.run('setHomeFilter("cards")');
  assert.deepEqual(visible(), ['poker']);
  app.run('resetHomeGames()');
  assert.equal(visible().length, 13);
  assert.equal(app.elements.get('home-search').value, '');
  assert.equal(app.elements.get('home-empty').hidden, true);
  app.run('setHomeFilter("party")');
  assert.equal(visible().length, 6);
  app.run('resetHomeGames()');
  app.elements.get('home-search').value = '29';
  app.run('filterHomeGames()');
  assert.deepEqual(visible(), ['twentynine']);
  const pokerCard = cards.find(card => card.textContent === 'poker');
  pokerCard.textContent = '12345 livepoker';
  pokerCard.querySelector = () => ({ textContent: '12345 live' });
  app.elements.get('home-search').value = '12345';
  app.run('filterHomeGames()');
  assert.deepEqual(visible(), []);
  assert.deepEqual(app.errors, []);
});

test('Homepage and room illustrations are local and every referenced symbol exists', () => {
  const art = fs.readFileSync(path.join(serverRoot, 'public/icons/game-art.svg'), 'utf8');
  const references = [...html.matchAll(/href="\/icons\/game-art\.svg#([^"]+)"/g)];
  assert.ok(references.length > 0);
  for (const [, id] of references) assert.ok(art.includes('id="' + id + '"'), 'Missing SVG symbol: ' + id);
  for (const [, id] of art.matchAll(/href="#([^"]+)"/g)) assert.ok(art.includes('id="' + id + '"'), 'Missing nested SVG symbol: ' + id);
  assert.doesNotMatch(art, /color="url\(/, 'SVG currentColor must be a color, not a paint-server URL');
  assert.ok(html.includes('href="/css/home.css"'));
});

test('Every home game has create and join panels with its own artwork and code-entry controls', () => {
  const games = [...html.matchAll(/class="game-card [^"]*"[^>]*onclick="openGameMenu\('([^']+)'\)"/g)].map(match => match[1]);
  assert.equal(games.length, 13);
  for (const game of games) {
    const setup = html.match(new RegExp(`<div id="menu-${game}" class="room-stage room-setup"[^>]*>([\\s\\S]*?)\\n  </div>`));
    assert.ok(setup, game + ' setup');
    assert.ok(setup[1].includes(`game-art.svg#${game === 'nquestions' ? 'questions' : game}`), game + ' artwork');
    assert.match(setup[1], /class="room-panel room-create"[\s\S]+onclick="create/);
    assert.match(setup[1], /class="room-panel room-join"[\s\S]+aria-label="Five-character room code"[\s\S]+onclick="join/);
  }
  assert.ok(html.includes('id="lobby" class="room-stage room-lobby"'));
  assert.doesNotMatch(html, /id="(?:screen-game|[^"]+-active)" class="[^"]*room-stage/);
});

test('Lobby copy confirms success or provides the code when clipboard access fails', async () => {
  const app = harness();
  const status = app.elements.get('invite-status');
  app.run('roomCode = "ABCDE"; navigator.clipboard = {writeText: async value => { navigator.copied = value; }}');
  await app.run('copyRoomCode(document.getElementById("invite-status"))');
  assert.equal(app.run('navigator.copied'), 'ABCDE');
  assert.equal(status.textContent, 'Room code copied');
  app.run('navigator.clipboard.writeText = async () => { throw new Error("Permission denied"); }');
  await app.run('copyRoomCode(document.getElementById("invite-status"))');
  assert.equal(status.textContent, 'Copy unavailable. Room code: ABCDE');
});

test('Bluff selects any rank, caps selection at four, and sends owned IDs', () => {
  const app = harness();
  app.message('bluff-state', bluffFixture());
  assert.equal(app.elements.get('bluff-required-rank').textContent, 'A');
  assert.match(app.elements.get('bluff-players').innerHTML, /&lt;Tester&gt;/);
  const buttons = app.document.querySelectorAll('#bluff-hand [data-card-id]');
  buttons.slice(0, 4).forEach(button => button.onclick());
  assert.equal(buttons[0]['aria-pressed'], 'true');
  assert.equal(buttons[4].disabled, true);
  app.run('bluffPlay()');
  assert.deepEqual(app.sent.at(-1), { type: 'bluff-play', payload: { cardIds: ['card-2', 'card-3', 'card-4', 'card-5'] } });
  assert.deepEqual(app.errors, []);
});

test('Bluff all nonactors challenge, next accepts, and pending empty hand is not a win', () => {
  const app = harness();
  const state = bluffFixture('challenge');
  state.pendingActorId = 'other';
  state.players[1].cards = [];
  app.message('bluff-state', state);
  assert.match(app.elements.get('bluff-pending-actor').textContent, /awaiting resolution/);
  assert.equal(app.elements.get('bluff-return-btn').hidden, true);
  app.run('bluffAccept(); bluffChallenge()');
  assert.deepEqual(app.sent.map(message => message.type), ['bluff-accept', 'bluff-challenge']);
  state.currentPlayerIdx = 2;
  app.message('bluff-state', state);
  assert.equal(app.elements.get('bluff-accept-btn').disabled, true);
  assert.equal(app.elements.get('bluff-challenge-btn').disabled, false);
  state.pendingActorId = 'test-player';
  app.message('bluff-state', state);
  assert.equal(app.elements.get('bluff-challenge-btn').disabled, true);
  assert.deepEqual(app.errors, []);
});

test('Tic-Tac-Toe accepts only empty cells on your turn and highlights the winning line', () => {
  const app = harness();
  const state = {phase:'playing', board:Array(9).fill(''), currentPlayerIdx:0,
    players:[{id:'test-player',name:'Alex'},{id:'other',name:'River'}]};
  app.message('tictactoe-state', state);
  assert.equal(app.elements.get('ttt-status').textContent, 'Your turn');
  assert.equal((app.elements.get('ttt-board').innerHTML.match(/onclick="markTTT/g) || []).length, 9);
  app.run('markTTT(0)');
  assert.deepEqual(app.sent.at(-1), {type:'tictactoe-move',payload:{cell:0}});
  const count = app.sent.length;
  state.board[0] = 'X';
  state.currentPlayerIdx = 1;
  app.message('tictactoe-state', state);
  app.run('markTTT(1)');
  assert.equal(app.sent.length, count);
  Object.assign(state,{phase:'game_over',winnerId:'test-player',winningLine:[0,1,2]});
  app.message('tictactoe-state', state);
  assert.equal(app.elements.get('ttt-status').textContent,'Alex wins!');
  assert.equal((app.elements.get('ttt-board').innerHTML.match(/class="ttt-win"/g)||[]).length,3);
  assert.equal(app.elements.get('ttt-return').hidden,false);
  assert.deepEqual(app.errors,[]);
});

test('Blackjack totals handle aces, never count hidden dealer cards, and display public hands', () => {
  const app = harness();
  app.message('blackjack-state', blackjackFixture());
  assert.equal(app.elements.get('blackjack-dealer-total').hidden, true);
  assert.equal(app.elements.get('blackjack-dealer-total').textContent, '');
  assert.match(app.elements.get('blackjack-dealer-cards').innerHTML, /aria-label="Hidden card"/);
  assert.match(app.elements.get('blackjack-players').innerHTML, /Ace of spades/);
  assert.equal(app.run('blackjackTotal([{rank:1},{rank:1},{rank:13}])'), 12);
  app.run("blackjackAction('double')");
  assert.equal(app.sent.at(-1).type, 'blackjack-double');
  const state = blackjackFixture();
  state.players[0].cards.push(card(2));
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-double-btn').disabled, true);
  state.players[0].cards.pop(); state.players[0].chips = 49;
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-double-btn').disabled, true);
  assert.deepEqual(app.errors, []);
});

test('Blackjack next round requires active host, terminal inactive member can return, spectator cannot act', () => {
  const app = harness();
  const state = blackjackFixture('round_over');
  state.dealerCards = [card(10), card(7)];
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-dealer-total').textContent, 'Total: 17');
  assert.equal(app.elements.get('blackjack-dealer-total').hidden, false);
  assert.equal(app.elements.get('blackjack-next-round-btn').disabled, false);
  app.run("blackjackAction('next-round')");
  assert.equal(app.sent.at(-1).type, 'blackjack-next-round');
  app.message('room-visibility', { public: false, hostId: 'other', twoDecks: false });
  assert.equal(app.elements.get('blackjack-next-round-btn').disabled, true);
  state.phase = 'game_over'; state.players[0].active = false;
  app.message('blackjack-state', state);
  assert.equal(app.elements.get('blackjack-return-btn').hidden, false);
  app.run('requestSpectate()');
  app.message('blackjack-state', blackjackFixture());
  assert.equal(app.elements.get('blackjack-hit-btn').disabled, true);
  assert.equal(app.elements.get('blackjack-spectator').textContent, 'Spectating');
  assert.deepEqual(app.errors, []);
});

test('Every game chat distinguishes own and incoming bubbles with persistent sender faces', () => {
  const app = harness();
  app.message('room-visibility', {avatars:{other:'fox'}});
  app.message('chat', {senderId:'other',senderName:'Alex',message:'Good hand!',timestamp:Date.now()});
  app.message('room-visibility', {});
  app.message('chat', {senderId:app.run('playerId'),senderName:'River Song#0001',message:'Thanks!',timestamp:Date.now()});
  for (const prefix of ['', 'poker-', 'ludo-', 'nq-', 'commune-', 'tn-', 'ht-', 'uno-', 'bluff-', 'blackjack-', 'tictactoe-', 'scrabble-', 'kabo-']) {
    const html = app.elements.get(prefix + 'chat-messages').innerHTML;
    assert.match(html, /class="chat-row">[\s\S]*🦊[\s\S]*Good hand!/);
    assert.match(html, /class="chat-row outgoing">[\s\S]*>RS<[\s\S]*>You<[\s\S]*Thanks!/);
  }
});

test('Lobby limits, room deck metadata, chat routing, settings, null reset, and exit cleanup', () => {
  const app = harness();
  app.message('waiting', { gameType: 'blackjack', hostId: 'test-player', twoDecks: true, players: [{ id: 'test-player', name: 'Tester' }] });
  assert.equal(app.elements.get('start-btn').disabled, false);
  assert.equal(app.elements.get('player-capacity').textContent, 4);
  assert.equal(app.elements.get('lobby-game-title').textContent, 'BLACKJACK');
  assert.equal(app.elements.get('lobby-additional-config').textContent, 'Two decks (104 cards)');
  app.message('blackjack-state', blackjackFixture());
  app.run('openGameSettings()');
  assert.equal(app.elements.get('game-settings-decks').textContent, 'One deck (52 cards)');
  app.elements.get('blackjack-chat-input').value = 'hello';
  app.run('sendChat()');
  assert.deepEqual(app.sent.at(-1), { type: 'chat', payload: { message: 'hello' } });
  app.message('chat', { senderId: 'other', senderName: 'Friend', message: 'hi', timestamp: Date.now() });
  assert.ok(app.elements.get('blackjacktab-chat').classList.contains('chat-unread'));
  app.run("switchAdditionalTab('blackjack', 'chat')");
  assert.ok(!app.elements.get('blackjacktab-chat').classList.contains('chat-unread'));
  app.message('state', null);
  assert.equal(app.run('blackjackState'), null);
  assert.equal(app.elements.get('blackjack-active').style.display, 'none');
  app.message('bluff-state', bluffFixture());
  app.run('toggleBluffRules(); exitGame()');
  assert.equal(app.sent.at(-1).type, 'leave-room');
  app.message('room-left');
  assert.equal(app.run('bluffState'), null);
  assert.equal(app.run('bluffSelectedCards.size'), 0);
  assert.equal(app.elements.get('bluff-rules-dialog').open, false);
  assert.equal(app.elements.get('bluff-active').style.display, 'none');
  assert.deepEqual(app.errors, []);
});

test('Create alone includes twoDecks; join and reconnect preserve server setting without a query override', async () => {
  const app = harness();
  app.elements.get('bluff-two-decks').checked = true;
  await app.run('createBluffGame()');
  assert.match(app.run('ws.url'), /gameType=bluff.*twoDecks=true.*action=create/);
  app.run('ws.onopen()');
  assert.equal(app.sent.at(-1).type, 'join');
  app.elements.get('blackjack-join-code').value = 'xyz12';
  app.run('joinBlackjackWithCode()');
  assert.match(app.run('ws.url'), /room=XYZ12/);
  assert.ok(!app.run('ws.url').includes('twoDecks'));
  app.run('tryReconnect(); ws.onopen()');
  assert.ok(!app.run('ws.url').includes('twoDecks'));
  assert.equal(app.sent.at(-1).type, 'join');
  assert.deepEqual(app.errors, []);
});

test('Separate frontend hosting uses the configured backend for APIs, creation, joining and reconnects', async () => {
  const backend = 'https://games.example.com:8443';
  const app = harness(backend);
  await app.run('loadRoomPlayerCount(); createBluffGame()');
  assert.ok(app.requests.includes(backend + '/api/room-stats'));
  assert.ok(app.requests.includes(backend + '/api/generate-code'));
  assert.ok(app.requests.every(url => url.startsWith(backend + '/api/')));
  assert.ok(app.run('ws.url').startsWith('wss://games.example.com:8443/ws?'));
  app.run('ws.onopen()');
  app.elements.get('blackjack-join-code').value = 'xyz12';
  app.run('joinBlackjackWithCode()');
  assert.ok(app.run('ws.url').startsWith('wss://games.example.com:8443/ws?room=XYZ12'));
  app.run('tryReconnect(); ws.onopen()');
  assert.ok(app.run('ws.url').startsWith('wss://games.example.com:8443/ws?room=XYZ12'));
  assert.equal(app.sent.at(-1).type, 'join');
  assert.deepEqual(app.errors, []);
});

test('Every state route hides unrelated screens; Bluff starts at two and nonparticipants spectate', () => {
  const app = harness();
  app.message('waiting', { gameType: 'bluff', hostId: 'test-player', twoDecks: false, players: [{ id: 'test-player', name: 'Tester' }] });
  assert.equal(app.elements.get('start-btn').disabled, true);
  assert.equal(app.elements.get('player-capacity').textContent, 6);
  const state = bluffFixture();
  app.message('players-updated', { gameType: 'bluff', hostId: 'test-player', twoDecks: true, players: state.players });
  assert.equal(app.elements.get('start-btn').disabled, false);
  for (const [id, node] of app.elements) if (id.endsWith('-active')) node.style.display = '';
  app.message('bluff-state', state);
  for (const [id, node] of app.elements) {
    if (id.endsWith('-active') && id !== 'bluff-active') assert.equal(node.style.display, 'none', id);
  }
  app.message('state', { phase: 'playing' });
  assert.equal(app.elements.get('bluff-active').style.display, 'none');
  assert.equal(app.elements.get('game-active').style.display, '');
  state.players[0].id = 'someone-else';
  app.message('bluff-state', state);
  assert.equal(app.elements.get('bluff-spectator').textContent, 'Spectating');
  assert.equal(app.elements.get('bluff-hand-section').hidden, true);
  assert.equal(app.elements.get('bluff-play-btn').disabled, true);
  assert.deepEqual(app.errors, []);
});

test('Exit confirmation protects active Blackjack including round_over, and terminal exits do not prompt', () => {
  const app = harness();
  app.message('blackjack-state', blackjackFixture('round_over'));
  app.run('confirm = () => false; exitGame()');
  assert.equal(app.sent.length, 0);
  app.run('confirm = () => true; exitGame()');
  assert.equal(app.sent.at(-1).type, 'leave-room');
  app.message('blackjack-state', blackjackFixture('game_over'));
  app.run('confirm = () => { throw new Error("Terminal exit must not prompt"); }; exitGame()');
  assert.equal(app.sent.at(-1).type, 'leave-room');
  assert.deepEqual(app.errors, []);
});