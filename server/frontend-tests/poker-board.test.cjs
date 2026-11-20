const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function renderer() {
  const board = { innerHTML: '' };
  const context = vm.createContext({ playerId: 'me', esc: value => String(value), document: { getElementById: () => board } });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../public/js/avatars.js'), 'utf8'), context);
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../public/js/poker.js'), 'utf8'), context);
  return { context, board };
}

test('Poker rail includes every player and spectators do not get a personal hand', () => {
  const { context, board } = renderer();
  for (let count = 2; count <= 8; count++) {
    context.pokerState = {phase:'flop', smallBlind:5, currentBet:10, players:Array.from({length:count}, (_, id)=>({id,name:'Player '+id,chips:100,isActive:true}))};
    context.renderPokerTable();
    assert.equal((board.innerHTML.match(/class="poker-seat-name"/g) || []).length, count);
    assert.doesNotMatch(board.innerHTML,/poker-personal|of spades|undefined/);
  }
});

test('Local cards appear below the community board, with correct pot and showdown state', () => {
  const { context, board } = renderer();
  context.pokerState = {
    phase: 'showdown', currentPlayerIndex: 0, dealerIndex: 1, communityCards: [], pots: [{ amount: 100 }],
    players: [
      { id: 'other', name: 'Alex', chips: 900, currentBet: 20, isActive: true, folded: true, holeCards: [] },
      { id: 'me', name: 'Morgan', chips: 1000, currentBet: 0, isActive: true, holeCards: [{ rank: 14, suit: 'spades' }], hand: { rankName: 'Pair' } }
    ]
  };
  context.renderPokerTable();
  assert.match(board.innerHTML, /poker-seat mine dealer/);
  assert.match(board.innerHTML, /poker-seat folded/);
  assert.match(board.innerHTML, /poker-community[\s\S]+poker-personal[\s\S]+A of spades/);
  assert.match(board.innerHTML, /Folded/);
  assert.equal((board.innerHTML.match(/class="poker-wager-slot"/g) || []).length, 2);
  assert.equal((board.innerHTML.match(/class="poker-wager"/g) || []).length, 1);
  assert.match(board.innerHTML, /aria-label="Bet: 20 chips"><b>20<\/b>/);
  assert.match(board.innerHTML, /Pair/);
  assert.match(board.innerHTML, /Pot<\/span><strong>120/);
  assert.equal((board.innerHTML.match(/class="poker-stat-icon" aria-hidden="true"/g) || []).length, 5);
  assert.equal((board.innerHTML.match(/poker-card-placeholder/g) || []).length, 5);
});

test('Raise stepper clamps at both bounds and submits the action shown', () => {
  const {context} = renderer();
  const controls = {'raise-slider':{min:'40',max:'97'},'raise-display':{},'raise-btn':{}};
  context.document.getElementById = id => controls[id];
  const sent = [];
  context.send = (_,payload) => sent.push([payload.action,payload.amount]);
  context.updateRaise(90);
  context.adjustRaise(10);
  assert.equal(controls['raise-btn'].textContent,'All in');
  context.submitPokerRaise();
  context.adjustRaise(-100);
  assert.equal(controls['raise-btn'].textContent,'Raise to 40');
  context.submitPokerRaise();
  assert.deepEqual(sent,[['allin',0],['raise',40]]);
});