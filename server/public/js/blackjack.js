let blackjackState = null;

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
  showAdditionalGame('blackjack', state);
  document.getElementById('blackjack-phase-display').textContent = 'Round ' + state.round + ' - ' + state.phase.replaceAll('_', ' ');
  const current = state.players[state.currentPlayerIdx];
  document.getElementById('blackjack-turn-display').textContent = state.phase === 'playing' ? (current?.name || 'Dealer') + "'s turn" : '';
  document.getElementById('blackjack-dealer-cards').innerHTML = (state.dealerCards || []).map(card => additionalCardHTML(card)).join('');
  document.getElementById('blackjack-dealer-total').textContent = blackjackTotalLabel(state.dealerCards);
  document.getElementById('blackjack-players').innerHTML = state.players.map(player =>
    '<section class="additional-player-hand' + (player.id === current?.id && state.phase === 'playing' ? ' current' : '') + '"><div class="additional-player-row"><strong>' + esc(player.name) +
    (player.id === playerId ? ' (You)' : '') + '</strong><span>' + player.cards.length + ' cards</span></div><div class="additional-cards">' + player.cards.map(card => additionalCardHTML(card)).join('') +
    '</div><div class="additional-player-stats"><span>' + blackjackTotalLabel(player.cards) + '</span><span>Chips: ' + player.chips + '</span><span>Bet: ' + player.bet +
    '</span><strong>' + esc(blackjackResult(player, state)) + '</strong></div></section>').join('');
  ['hit', 'stand', 'double', 'next-round'].forEach(action => {
    document.getElementById('blackjack-' + action + '-btn').disabled = !canBlackjackAct(action);
  });
  document.getElementById('blackjack-next-round-btn').hidden = state.phase !== 'round_over';
  document.getElementById('blackjack-return-btn').hidden = isSpectating || state.phase !== 'game_over';
}