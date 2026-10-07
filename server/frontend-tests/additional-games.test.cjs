const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const serverRoot = path.resolve(__dirname, '..');
const html = fs.readFileSync(path.join(serverRoot, 'client.html'), 'utf8');

function harness() {
  const elements = new Map();
  const sent = [];
  const errors = [];
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
      append() {}, replaceChildren() {}, parentElement: { title: '' }
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
    location: { port: '8080', hostname: 'localhost', host: 'localhost:8080', protocol: 'http:' },
    crypto: { randomUUID: () => 'test-player' }, WebSocket: Socket,
    URLSearchParams, AbortController, navigator: {},
    console: { log() {}, error: (...args) => errors.push(args) },
    setTimeout: () => 1, clearTimeout() {}, setInterval: () => 1, clearInterval() {},
    alert() {}, confirm: () => true,
    fetch: async () => ({ ok: true, json: async () => ({ players: 0, code: 'ABCDE' }) })
  });
  context.localStorage.setItem('coup_name', 'Tester');
  function run(code) { return vm.runInContext(code, context); }
  run('let ludoState = null, nqState = null, communeState = null, tnState = null, htState = null, unoState = null; function updateVoiceUI() {} function renderGame() { document.getElementById("game-active").style.display = ""; }');
  for (const name of ['common', 'bluff', 'blackjack']) run(fs.readFileSync(path.join(serverRoot, 'public/js', name + '.js'), 'utf8'));
  run('ws = new WebSocket("test"); hostId = playerId;');
  const message = (type, payload) => run('handleWSMessage({data:' + JSON.stringify(JSON.stringify({ type, payload })) + '})');
  return { run, message, elements, sent, errors, document };
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

test('HTML readiness count, unique IDs, callbacks, assets, and default deck options', () => {
  const ids = [...html.matchAll(/\bid="([^"]+)"/g)].map(match => match[1]);
  assert.equal(new Set(ids).size, ids.length);
  const scripts = [...html.matchAll(/<script src="([^"]+)" defer onload="checkScriptsLoaded\(\)"><\/script>/g)];
  assert.equal(scripts.length, Number(html.match(/let totalScripts = (\d+)/)[1]));
  assert.equal(scripts.length, 12);
  for (const script of scripts) assert.ok(fs.existsSync(path.join(serverRoot, 'public', script[1])));
  for (const match of html.matchAll(/\bon(?:click|input|keydown|change|load)="([^"]*)"/g)) new Function('event', match[1]);
  for (const type of ['bluff', 'blackjack']) assert.ok(!new RegExp('id="' + type + '-two-decks"[^>]*checked').test(html));
  assert.ok(fs.existsSync(path.join(serverRoot, 'public/textures/card-back.svg')));
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

test('Blackjack totals handle aces, never count hidden dealer cards, and display public hands', () => {
  const app = harness();
  app.message('blackjack-state', blackjackFixture());
  assert.equal(app.elements.get('blackjack-dealer-total').textContent, 'Visible total: 11');
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