const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '../public/js/scrabble.js'), 'utf8');
function fixture() {
  return {id: 'match', phase: 'playing', currentPlayerIdx: 0, players: [
    {id: 'p1', name: '<Alice>', score: 9, active: true},
    {id: 'p2', name: 'Bob', score: 8, active: true}
  ], board: Array(225).fill(''), premiums: Array(225).fill(''), rack: ['T','T','?'], bagCount: 8, lastAction: ''};
}

// Exercise the public state handler and delegated events without a WebSocket or browser dependency.
function harness(playerId = 'p1') {
  const sent = [], screens = [];
  const nodes = new Map();
  const ctx = vm.createContext({playerId, gameActive:false, isSpectating:false,
    hideAllActiveGames: () => screens.push('hide'), showScreen: name => screens.push(name),
    playerBadge: () => '<span>Avatar</span>', esc: str => str.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;'),
    send: (type,payload) => sent.push([type, JSON.parse(JSON.stringify(payload))])});
  vm.runInContext(source, ctx); // document intentionally unavailable at load
  ctx.document = {getElementById: id => {
    if (!nodes.has(id)) nodes.set(id, {style:{}, setAttribute(name, val) { this[name] = val; }});
    return nodes.get(id);
  }};
  const root = ctx.document.getElementById('scrabble-play-area');
  const scroll = {scrollLeft:0};
  root.contains = el => !!el?.dataset;
  root.querySelector = selector => {
    if (selector === '.sc-scroll') return scroll;
    const attr = selector === '[data-cell][tabindex="0"]' ? 'tabindex="0"' : selector.slice(1,-1);
    const tag = (root.innerHTML || '').match(new RegExp(`<(?:button|select)\\b[^>]*?\\b${attr}(?=[\\s>])[^>]*>`))?.[0];
    if (!tag) return null;
    return {dataset: Object.fromEntries([...tag.matchAll(/data-(\w+)(?:="([^"]*)")?/g)].map(m => [m[1],m[2] || ''])),
      disabled:/\bdisabled\b/.test(tag), closest() {return this;}, matches: s => s === '[data-blank]',
      focus() {ctx.document.activeElement = this;}};
  };
  const get = (key, value) => root.querySelector(`[data-${key}="${value}"]`);
  const click = (key, value) => {
    const target = get(key,value);
    assert.ok(target, `Missing ${key} ${value}`);
    root.onclick({target});
  };
  const update = state => ctx.handleScrabbleState(state);
  return {root, ctx, screens, nodes, click, get, sent, update, slot:i => click('slot',i), cell:i => click('cell',i), act:a => click('action',a)};
}

test('activates the game and renders all squares, private rack, premiums, blanks, scores, turn and avatars', () => {
  const a = harness(), s = fixture();
  s.premiums[112] = 'DW'; s.board[16] = 'a';
  a.update(s);
  assert.deepEqual(a.screens,['hide','game']);
  assert.equal(a.ctx.gameActive,true);
  assert.equal(a.ctx.isSpectating,false);
  assert.equal(a.nodes.get('lobby').style.display,'none');
  assert.equal(a.nodes.get('name-entry').style.display,'none');
  assert.equal(a.nodes.get('scrabble-active').style.display,'');
  assert.equal(a.nodes.get('scrabble-status').textContent,'Your turn');
  assert.equal((a.root.innerHTML.match(/data-cell="/g) || []).length,225);
  assert.match(a.root.innerHTML,/H8, empty, center star, double word/);
  assert.match(a.root.innerHTML,/★/);
  assert.match(a.root.innerHTML,/B2, blank as A, 0 points/);
  assert.match(a.root.innerHTML,/sc-current.*Avatar.*>You<.*>9<.* points</s);
  assert.ok(a.root.innerHTML.indexOf('Game board') < a.root.innerHTML.indexOf('Your private rack'));
  assert.doesNotMatch(a.root.innerHTML,/Swipe to see|Select a rack tile|<h3>Board/);
  assert.match(a.root.innerHTML,/Your private rack/);
  a.root.onkeydown({target:a.get('cell',112), key:'ArrowLeft', preventDefault() {}});
  assert.equal(a.ctx.document.activeElement.dataset.cell,'111');
});

test('tap to place, undo, choose a blank and submit only public placement fields', () => {
  const a = harness(), s = fixture();
  a.update(s);
  a.slot(0); a.cell(112); a.slot(1); a.cell(113); a.cell(112);
  assert.equal(a.get('slot',0).disabled,false);
  a.act('undo');
  assert.equal(a.get('action','submit').disabled,true);
  a.slot(2); a.cell(112);
  assert.equal(a.get('action','submit').disabled,true);
  const blank = a.root.querySelector('[data-blank]');
  blank.value = 'A'; a.root.onchange({target:blank});
  a.cell(112); a.slot(0); a.cell(113);
  a.act('submit');
  assert.deepEqual(a.sent,[['scrabble-play',{placements:[
    {cell:112,letter:'A',blank:true},{cell:113,letter:'T',blank:false}]}]]);
  assert.equal(s.board[112],'');
  assert.deepEqual(s.rack,['T','T','?']);
});

test('draft survives unrelated refreshes but not board, game, turn, rack or phase changes', () => {
  for (const change of [s => s.id = 'new match', s => s.board[0] = 'A', s => s.rack.reverse(),
    s => s.currentPlayerIdx = 1, s => s.phase = 'voting']) {
    const a = harness(), s = fixture();
    a.update(s); a.slot(0); a.cell(112);
    s.lastAction = 'Another player rejoined'; s.players[1].name = 'Bob Jr.';
    a.update(structuredClone(s));
    assert.match(a.root.innerHTML,/H8, T, 1 points, center star, draft/);
    change(s); a.update(s);
    assert.doesNotMatch(a.root.innerHTML,/center star, draft/);
    assert.match(a.root.innerHTML,/H8, empty, center star/);
  }
});

test('pass and exchange use the right messages, including duplicate tile multiplicity', () => {
  const a = harness(), s = fixture();
  a.update(s); a.act('pass'); a.act('swap');
  a.slot(2); a.act('cancel');
  a.act('swap'); a.slot(0); a.slot(1); a.act('exchange');
  assert.deepEqual(a.sent,[['scrabble-pass',{}],['scrabble-exchange',{tiles:['T','T']}]]);
  s.bagCount = 6; a.update(s); a.act('cancel');
  assert.equal(a.get('action','swap').disabled,true);
});

test('voting overlays the proposal and permits only an active, unapproved non-proposer to vote', () => {
  const a = harness('p2'), s = fixture();
  s.phase = 'voting'; s.pending = {id:42, words:['QUIZ'], approvals:['p1'], placements:[{cell:112,letter:'Q',blank:true}]};
  a.update(s);
  assert.match(a.root.innerHTML,/H8, blank as Q, 0 points, center star, proposed/);
  assert.match(a.root.innerHTML,/QUIZ/);
  assert.match(a.root.innerHTML,/Approved: &lt;Alice&gt;/);
  assert.equal(a.get('action','pass'),null);
  a.act('approve'); a.act('reject');
  assert.deepEqual(a.sent, [['scrabble-vote',{id:42,approve:true}],['scrabble-vote',{id:42,approve:false}]]);
  for (const id of ['spectator','p1']) {
    const viewer = harness(id); viewer.update(s);
    assert.equal(viewer.get('action','approve'),null);
    if (id === 'spectator') {
      assert.equal(viewer.ctx.isSpectating,true);
      assert.doesNotMatch(viewer.root.innerHTML,/data-slot=|Your rack/);
    }
  }
  s.pending.approvals.push('p2'); a.update(s);
  assert.equal(a.get('action','reject'),null);
  s.pending.approvals = []; s.players[1].active = false; a.update(s);
  assert.equal(a.get('action','approve'),null);
});

test('game over announces all tied winners, allows lobby return and prevents further moves', () => {
  const a = harness(), s = fixture();
  s.phase = 'game_over'; s.players[1].score = 9;
  a.update(s);
  assert.match(a.nodes.get('scrabble-status').textContent,/Game over.*<Alice> & Bob tie with 9 points/);
  assert.equal(a.get('slot',0).disabled,true);
  assert.equal(a.get('action','submit'),null);
  a.act('return'); assert.deepEqual(a.sent,[['return-to-lobby',{}]]);
});
