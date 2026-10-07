const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function renderer() {
  const board = { innerHTML: '' };
  const context = vm.createContext({ playerId: 'me', esc: value => String(value), document: { getElementById: () => board } });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../public/js/poker.js'), 'utf8'), context);
  return { context, board };
}

test('Poker seats alternate left/right after bottom/top for two through eight players', () => {
  const { context } = renderer();
  for (let count = 2; count <= 8; count++) {
    const seats = context.pokerSeatPositions(count);
    assert.equal(seats[0].side, 'bottom');
    assert.equal(seats[1].side, 'top');
    for (let index = 2; index < count; index++) assert.equal(seats[index].side, index % 2 === 0 ? 'left' : 'right');
    for (const side of ['left', 'right']) {
      const positions = seats.filter(seat => seat.side === side).map(seat => seat.percent);
      assert.equal(new Set(positions).size, positions.length);
      if (positions.length === 1) assert.equal(positions[0], 50);
      assert.ok(positions.every(position => position >= 26 && position <= 74));
    }
  }
});

test('Poker rotates local player to bottom and retains folded and showdown states', () => {
  const { context, board } = renderer();
  context.pokerState = {
    phase: 'showdown', currentPlayerIndex: 0, dealerIndex: 1, communityCards: [], pots: [{ amount: 100 }],
    players: [
      { id: 'other', name: 'Alex', chips: 900, currentBet: 20, isActive: true, folded: true, holeCards: [] },
      { id: 'me', name: 'Morgan', chips: 1000, currentBet: 0, isActive: true, holeCards: [{ rank: 14, suit: 'spades' }], hand: { rankName: 'Pair' } }
    ]
  };
  context.renderPokerTable();
  assert.match(board.innerHTML, /poker-seat-bottom mine dealer/);
  assert.match(board.innerHTML, /poker-seat-top folded/);
  assert.match(board.innerHTML, /Folded/);
  assert.match(board.innerHTML, /Pair/);
  assert.match(board.innerHTML, /Pot: 120/);
  assert.equal((board.innerHTML.match(/poker-card-placeholder/g) || []).length, 5);
});