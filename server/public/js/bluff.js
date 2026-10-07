let bluffState = null;
let bluffSelectedCards = new Set();

const ADDITIONAL_RANKS = { 1: 'A', 11: 'J', 12: 'Q', 13: 'K' };
const ADDITIONAL_RANK_NAMES = { 1: 'Ace', 11: 'Jack', 12: 'Queen', 13: 'King' };
const ADDITIONAL_SUITS = { clubs: '&#9827;', diamonds: '&#9830;', hearts: '&#9829;', spades: '&#9824;' };

function additionalRank(rank) { return ADDITIONAL_RANKS[rank] || String(rank); }
function additionalCardName(card) {
  return card.rank ? (ADDITIONAL_RANK_NAMES[card.rank] || card.rank) + ' of ' + card.suit : 'Hidden card';
}
function additionalCardHTML(card, selectable = false, selected = false, disabled = true) {
  const hidden = !card.rank;
  const classes = 'additional-card' + (hidden ? ' additional-card-back' : '') +
    (['hearts', 'diamonds'].includes(card.suit) ? ' additional-card-red' : '') + (selected ? ' selected' : '');
  const contents = hidden ? '<span aria-hidden="true">?</span>' :
    '<span class="additional-card-rank">' + additionalRank(card.rank) + '</span><span class="additional-card-suit" aria-hidden="true">' + (ADDITIONAL_SUITS[card.suit] || '') + '</span>';
  const label = esc(additionalCardName(card));
  return selectable ? '<button type="button" class="' + classes + '" data-card-id="' + esc(card.id) +
    '" aria-label="' + label + '" aria-pressed="' + selected + '"' + (disabled ? ' disabled' : '') + '>' + contents + '</button>' :
    '<div class="' + classes + '" role="img" aria-label="' + label + '">' + contents + '</div>';
}
function showAdditionalGame(type, state) {
  hideAllActiveGames();
  showScreen('game');
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById(type + '-active').style.display = '';
  gameActive = true;
  updateAdditionalRoomSettings(state);
  document.getElementById(type + '-deck-display').textContent = state.twoDecks ? 'Two decks' : 'One deck';
  document.getElementById(type + '-spectator').textContent = isSpectating ? 'Spectating' : '';
  document.getElementById(type + '-last-action').textContent = state.lastAction || '';
  renderChatMessages();
}
function handleBluffStateUpdate(payload) {
  if (!payload) return;
  bluffState = payload;
  isSpectating = isSpectating || !payload.players.some(player => player.id === playerId && player.active);
  const hand = payload.players.find(player => player.id === playerId)?.cards || [];
  const owned = new Set(hand.filter(card => card.id).map(card => card.id));
  bluffSelectedCards = new Set([...bluffSelectedCards].filter(id => owned.has(id)));
  if (!canBluffPlay()) bluffSelectedCards.clear();
  renderBluffGame();
}
function canBluffPlay() {
  const player = bluffState?.players[bluffState.currentPlayerIdx];
  return !isSpectating && bluffState?.phase === 'playing' && player?.id === playerId && player.active;
}
function canBluffRespond(accept) {
  const mine = bluffState?.players.find(player => player.id === playerId);
  return !isSpectating && bluffState?.phase === 'challenge' && mine?.active &&
    bluffState.pendingActorId !== playerId && !!bluffState.pendingActorId &&
    (!accept || bluffState.players[bluffState.currentPlayerIdx]?.id === playerId);
}
function updateBluffSelection() {
  document.querySelectorAll('#bluff-hand [data-card-id]').forEach(button => {
    const selected = bluffSelectedCards.has(button.dataset.cardId);
    button.classList.toggle('selected', selected);
    button.setAttribute('aria-pressed', String(selected));
    button.disabled = !canBluffPlay() || (!selected && bluffSelectedCards.size >= 4);
  });
  document.getElementById('bluff-selection-count').textContent = bluffSelectedCards.size + '/4 selected';
  document.getElementById('bluff-play-btn').disabled = !canBluffPlay() || bluffSelectedCards.size < 1 || bluffSelectedCards.size > 4;
}
function bluffPlay() {
  if (!canBluffPlay() || bluffSelectedCards.size < 1 || bluffSelectedCards.size > 4) return;
  send('bluff-play', { cardIds: [...bluffSelectedCards] });
}
function bluffChallenge() { if (canBluffRespond(false)) send('bluff-challenge'); }
function bluffAccept() { if (canBluffRespond(true)) send('bluff-accept'); }
function renderBluffGame() {
  if (!bluffState) return;
  showAdditionalGame('bluff', bluffState);
  const state = bluffState;
  const current = state.players[state.currentPlayerIdx];
  const actor = state.players.find(player => player.id === state.pendingActorId);
  const mine = state.players.find(player => player.id === playerId);
  document.getElementById('bluff-phase-display').textContent = state.phase === 'game_over' ?
    (state.winnerName ? state.winnerName + ' wins' : 'Game over') : state.phase === 'challenge' ? 'Claim pending' : 'Playing';
  document.getElementById('bluff-turn-display').textContent = state.phase === 'game_over' ? '' : (current?.name || '') + (state.phase === 'challenge' ? ' may accept' : "'s turn");
  document.getElementById('bluff-required-rank').textContent = additionalRank(state.requiredRank);
  document.getElementById('bluff-pile-count').textContent = (state.pile || []).length;
  document.getElementById('bluff-pending-actor').textContent = actor ? actor.name + (actor.cards.length === 0 ? ' - empty hand, awaiting resolution' : ' - claim pending') : 'No pending claim';
  document.getElementById('bluff-players').innerHTML = state.players.map(player =>
    '<div class="additional-player-row' + (player.id === current?.id && state.phase !== 'game_over' ? ' current' : '') + '"><strong>' + esc(player.name) +
    (player.id === playerId ? ' (You)' : '') + '</strong><span>' + player.cards.length + ' cards' + (!player.active ? ' - inactive' : '') + '</span></div>').join('');
  document.getElementById('bluff-hand').innerHTML = !isSpectating && mine ? mine.cards.filter(card => card.id && card.rank).map(card =>
    additionalCardHTML(card, true, bluffSelectedCards.has(card.id), !canBluffPlay())).join('') : '';
  document.getElementById('bluff-hand-section').hidden = isSpectating || !mine;
  document.querySelectorAll('#bluff-hand [data-card-id]').forEach(button => button.addEventListener('click', () => {
    if (!canBluffPlay()) return;
    const id = button.dataset.cardId;
    if (bluffSelectedCards.has(id)) bluffSelectedCards.delete(id);
    else if (bluffSelectedCards.size < 4) bluffSelectedCards.add(id);
    updateBluffSelection();
  }));
  document.getElementById('bluff-play-btn').textContent = 'Claim ' + additionalRank(state.requiredRank);
  updateBluffSelection();
  document.getElementById('bluff-challenge-btn').disabled = !canBluffRespond(false);
  document.getElementById('bluff-accept-btn').disabled = !canBluffRespond(true);
  document.getElementById('bluff-return-btn').hidden = isSpectating || state.phase !== 'game_over';
}