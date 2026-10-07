let blackjackState = null;
let blackjackAnimationRound = '';
let blackjackSeenCards = new Map();

function blackjackAnimatedCards(cards, owner, initialDeal, seatOrder, seatCount) {
  return (cards || []).map((card, index) => {
    const key = owner + ':' + index;
    const signature = [card.id || '', card.rank || '', card.suit || ''].join(':');
    const previous = blackjackSeenCards.get(key);
    blackjackSeenCards.set(key, signature);
    const animate = previous !== signature;
    const delay = initialDeal ? (index * seatCount + seatOrder) * 100 : index * (owner === 'dealer' ? 100 : 0);
    return '<div class="blackjack-card-shell' + (animate ? ' blackjack-card-unfolding' : '') + '" style="--blackjack-deal-delay:' + delay + 'ms">' +
      '<div class="blackjack-card-motion"><div class="blackjack-card-front">' + additionalCardHTML(card) + '</div><div class="blackjack-card-reverse" aria-hidden="true">' + additionalCardHTML({}) + '</div></div></div>';
  }).join('');
}

function blackjackTotal(cards) {
  let total = 0;
  let aces = 0;
  for (const card of cards || []) {
    if (!card.rank) continue;
    if (card.rank === 1) { total += 11; aces++; }
    else total += Math.min(card.rank, 10);
  }
  while (total > 21 && aces > 0) { total -= 10; aces--; }
  return total;
}
function blackjackTotalLabel(cards) {
  return (cards || []).some(card => !card.rank) ? 'Visible total: ' + blackjackTotal(cards) : 'Total: ' + blackjackTotal(cards);
}
function handleBlackjackStateUpdate(payload) {
  if (!payload) return;
  blackjackState = payload;
  isSpectating = isSpectating || !payload.players.some(player => player.id === playerId);
  renderBlackjackGame();
}
function canBlackjackAct(action) {
  const state = blackjackState;
  const mine = state?.players.find(player => player.id === playerId);
  if (isSpectating || !mine?.active) return false;
  if (action === 'next-round') return hostId === playerId && state.phase === 'round_over';
  if (state.phase !== 'playing' || state.players[state.currentPlayerIdx]?.id !== playerId || mine.status !== 'active') return false;
  return action !== 'double' || (mine.cards.length === 2 && mine.bet === 50 && mine.chips >= mine.bet);
}
function blackjackAction(action) {
  if (['hit', 'stand', 'double', 'next-round'].includes(action) && canBlackjackAct(action)) send('blackjack-' + action);
}
function blackjackResult(player, state) {
  if (!player.active) return 'Inactive';
  if (state.phase !== 'round_over') return player.status;
  const total = blackjackTotal(player.cards);
  const dealerTotal = blackjackTotal(state.dealerCards);
  const natural = player.cards.length === 2 && total === 21;
  const dealerNatural = state.dealerCards.length === 2 && dealerTotal === 21;
  if (total > 21) return 'Bust - lost';
  if (dealerNatural) return natural ? 'Push' : 'Dealer blackjack - lost';
  if (natural) return 'Blackjack - won 3:2';
  if (dealerTotal > 21 || total > dealerTotal) return 'Won';
  return total === dealerTotal ? 'Push' : 'Lost';
}
function renderBlackjackGame() {
  if (!blackjackState) return;
  const state = blackjackState;
  const roundKey = [state.id || (typeof roomCode === 'string' ? roomCode : ''), state.round].join(':');
  const initialDeal = roundKey !== blackjackAnimationRound;
  if (initialDeal) {
    blackjackAnimationRound = roundKey;
    blackjackSeenCards.clear();
  }
  showAdditionalGame('blackjack', state);
  const results = document.getElementById('blackjack-round-results');
  results.hidden = state.phase !== 'round_over' || isSpectating || !state.players.some(player => player.id === playerId && player.active);
  results.innerHTML = state.phase === 'round_over' ? '<h2>Round ' + state.round + ' results</h2>' +
    state.players.filter(player => player.id === playerId && player.active && !isSpectating).map(player => {
      const result = blackjackRoundOutcome(player, state);
      return '<div class="blackjack-outcome ' + result.kind + '"><div class="blackjack-outcome-heading"><strong>' +
        esc(player.name) + (player.id === playerId ? ' (You)' : '') + ' - ' + result.title + '</strong><b>' +
        (result.net > 0 ? '+' : '') + result.net + ' chips</b></div><p>' + esc(result.reason) + '</p></div>';
    }).join('') : '';
  document.getElementById('blackjack-phase-display').textContent = 'Round ' + state.round + ' - ' + state.phase.replaceAll('_', ' ');
  const current = state.players[state.currentPlayerIdx];
  document.getElementById('blackjack-turn-display').textContent = state.phase === 'playing' ? (current?.name || 'Dealer') + "'s turn" : '';
  document.getElementById('blackjack-dealer-cards').innerHTML = blackjackAnimatedCards(state.dealerCards, 'dealer', initialDeal, 0, state.players.length + 1);
  document.getElementById('blackjack-dealer-total').textContent = blackjackTotalLabel(state.dealerCards);
  document.getElementById('blackjack-players').innerHTML = state.players.map((player, playerIndex) =>
    '<section class="additional-player-hand' + (player.id === current?.id && state.phase === 'playing' ? ' current' : '') + '"><div class="additional-player-row"><strong>' + esc(player.name) +
    (player.id === playerId ? ' (You)' : '') + '</strong><span>' + player.cards.length + ' cards</span></div><div class="additional-cards">' + blackjackAnimatedCards(player.cards, player.id, initialDeal, playerIndex + 1, state.players.length + 1) +
    '</div><div class="additional-player-stats"><span>' + blackjackTotalLabel(player.cards) + '</span><span>Chips: ' + player.chips + '</span><span>Bet: ' + player.bet +
    '</span><strong>' + (player.id === playerId && !isSpectating ? esc(blackjackResult(player, state)) : '') + '</strong></div></section>').join('');
  ['hit', 'stand', 'double', 'next-round'].forEach(action => {
    document.getElementById('blackjack-' + action + '-btn').disabled = !canBlackjackAct(action);
  });
  document.getElementById('blackjack-next-round-btn').hidden = state.phase !== 'round_over';
  document.getElementById('blackjack-return-btn').hidden = isSpectating || state.phase !== 'game_over';
}

function blackjackRoundOutcome(player, state) {
  const total = blackjackTotal(player.cards);
  const dealer = blackjackTotal(state.dealerCards);
  const natural = player.cards.length === 2 && total === 21;
  const dealerNatural = state.dealerCards.length === 2 && dealer === 21;
  if (total > 21) return { kind: 'loss', title: 'Lost', net: -player.bet, reason: 'Busted with ' + total + '. Over 21 loses, regardless of the dealer hand.' };
  if (dealerNatural && natural) return { kind: 'push', title: 'Push', net: 0, reason: 'Both you and the dealer have natural blackjack. Your bet is returned.' };
  if (dealerNatural) return { kind: 'loss', title: 'Lost', net: -player.bet, reason: 'Dealer has natural blackjack; your ' + total + ' cannot beat it.' };
  if (natural) return { kind: 'win', title: 'Blackjack!', net: Math.floor(player.bet * 3 / 2), reason: 'Two-card 21 beats the dealer. Natural blackjack pays 3:2.' };
  if (dealer > 21) return { kind: 'win', title: 'Won', net: player.bet, reason: 'Dealer busted with ' + dealer + '; your ' + total + ' stays under 22. Win pays 1:1.' };
  if (total > dealer) return { kind: 'win', title: 'Won', net: player.bet, reason: 'Your ' + total + ' beats dealer ' + dealer + '. Win pays 1:1.' };
  if (total === dealer) return { kind: 'push', title: 'Push', net: 0, reason: 'Your ' + total + ' ties dealer ' + dealer + '. Your bet is returned.' };
  return { kind: 'loss', title: 'Lost', net: -player.bet, reason: 'Dealer ' + dealer + ' beats your ' + total + '. You lose your bet.' };
}