const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const context = vm.createContext({});
vm.runInContext(fs.readFileSync(path.join(__dirname, '../public/js/blackjack.js'), 'utf8'), context);
const cards = ranks => ranks.map(rank => ({ rank }));

test('Blackjack unfolds only new cards and newly revealed dealer cards', () => {
  const animationContext = vm.createContext({ additionalCardHTML: card => '<div>' + (card.rank || '?') + '</div>' });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../public/js/blackjack.js'), 'utf8'), animationContext);
  const hand = [{ id: 'first', rank: 10 }, { id: 'second', rank: 7 }];
  const initial = animationContext.blackjackAnimatedCards(hand, 'player', true, 1, 2);
  assert.equal((initial.match(/blackjack-card-unfolding/g) || []).length, 2);
  assert.match(initial, /--blackjack-deal-delay:100ms/);
  assert.match(initial, /--blackjack-deal-delay:300ms/);
  assert.equal((animationContext.blackjackAnimatedCards(hand, 'player', false, 1, 2).match(/blackjack-card-unfolding/g) || []).length, 0);
  const hit = animationContext.blackjackAnimatedCards([...hand, { id: 'hit', rank: 3 }], 'player', false, 1, 2);
  assert.equal((hit.match(/blackjack-card-unfolding/g) || []).length, 1);
  animationContext.blackjackAnimatedCards([{ id: 'up', rank: 6 }, { id: 'hidden' }], 'dealer', true, 0, 2);
  const reveal = animationContext.blackjackAnimatedCards([{ id: 'up', rank: 6 }, { id: 'hole', rank: 10 }], 'dealer', false, 0, 2);
  assert.equal((reveal.match(/blackjack-card-unfolding/g) || []).length, 1);
});

test('Blackjack result reasons and net payouts follow settlement rules', () => {
  const cases = [
    { hand: [10, 9], dealer: [7, 11], kind: 'win', net: 50, reason: '19 beats dealer 17' },
    { hand: [10, 9], dealer: [10, 8, 6], kind: 'win', net: 50, reason: 'Dealer busted with 24' },
    { hand: [10, 8, 6], dealer: [10, 8, 6], kind: 'loss', net: -50, reason: 'Busted with 24' },
    { hand: [1, 13], dealer: [10, 9], kind: 'win', net: 75, reason: 'pays 3:2' },
    { hand: [1, 13], dealer: [1, 12], kind: 'push', net: 0, reason: 'Both' },
    { hand: [10, 5, 6], dealer: [1, 12], kind: 'loss', net: -50, reason: 'natural blackjack' },
    { hand: [10, 7], dealer: [10, 7], kind: 'push', net: 0, reason: 'ties dealer 17' },
    { hand: [10, 7], dealer: [10, 9], kind: 'loss', net: -50, reason: 'Dealer 19 beats your 17' },
    { hand: [10, 4, 6], dealer: [10, 8], bet: 100, kind: 'win', net: 100, reason: '20 beats dealer 18' }
  ];
  for (const entry of cases) {
    const result = context.blackjackRoundOutcome({ cards: cards(entry.hand), bet: entry.bet || 50 }, { dealerCards: cards(entry.dealer) });
    assert.equal(result.kind, entry.kind);
    assert.equal(result.net, entry.net);
    assert.ok(result.reason.includes(entry.reason), result.reason);
  }
});