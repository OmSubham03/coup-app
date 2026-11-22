let kaboState = null;
let kaboSelected = [];
let kaboTarget = '';
let kaboPending = null;
let kaboBusy = false;

function kaboResetSelection() {
  kaboSelected = [];
  kaboTarget = '';
  kaboPending = null;
  kaboBusy = false;
  const dialog = document.getElementById('kabo-confirm-dialog');
  if (dialog.open) dialog.close();
}

function handleKaboState(state) {
  const changed = !kaboState || state.revision !== kaboState.revision || state.id !== kaboState.id;
  kaboState = state;
  if (changed) kaboResetSelection();
  hideAllActiveGames();
  showScreen('game');
  gameActive = true;
  isSpectating = !state.players.some(p => p.id === playerId && p.active);
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('kabo-active').style.display = '';
  renderKabo();
}

function kaboMine() { return kaboState?.players.find(p => p.id === playerId && p.active); }
function kaboTurn() { return !isSpectating && kaboState?.players[kaboState.currentPlayerIdx]?.id === playerId; }
function kaboOver() { return ['round_over', 'game_over'].includes(kaboState?.phase); }

function kaboCard(card, owner, index) {
  if (card.empty) return '<span class="kabo-empty-slot" role="img" aria-label="Empty card position"></span>';
  const state = kaboState;
  const reveal = state.reveals.find(r => r.slotId === card.id);
  const shown = reveal?.card || card;
  const mine = owner.id === playerId;
  let enabled = false;
  if (!isSpectating && !kaboBusy && !kaboOver()) {
    if (state.phase === 'initial') enabled = mine && !kaboMine()?.ready && state.reveals.length === 0;
    else if (state.phase === 'resolve') enabled = mine && kaboTurn();
    else if (state.phase === 'power' && kaboTurn()) enabled = mine ? state.power !== 'spy' : state.power !== 'own';
    else if (state.phase === 'draw' && state.matchAllowed) enabled = true;
  }
  const selected = mine ? kaboSelected.includes(card.id) : kaboTarget === card.id;
  return '<button type="button" class="kabo-card-target' + (selected ? ' selected' : '') + '" data-kabo-slot="' + esc(card.id) +
    '" data-kabo-owner="' + esc(owner.id) + '" aria-label="' + esc((mine ? 'Your' : owner.name + "'s") + ' card ' + (index + 1) +
      (shown.rank ? ': ' + additionalCardName(shown) : ': hidden')) + '" aria-pressed="' + selected + '"' +
    (enabled ? '' : ' disabled') + '>' + additionalCardHTML(shown) + '</button>';
}

function kaboChoose(owner, slotID) {
  if (kaboBusy || isSpectating || kaboOver()) return;
  const state = kaboState;
  const p = state.players.find(p => p.id === owner && p.active);
  if (!p?.cards.some(c => c.id === slotID && !c.empty)) return;
  if (state.phase === 'initial') {
    if (owner !== playerId || kaboMine().ready || state.reveals.length) return;
    if (kaboSelected.includes(slotID)) kaboSelected = kaboSelected.filter(id => id !== slotID);
    else if (kaboSelected.length < 2) kaboSelected.push(slotID);
  } else {
    if (owner === playerId) kaboSelected = kaboSelected[0] === slotID ? [] : [slotID];
    else kaboTarget = kaboTarget === slotID ? '' : slotID;
  }
  renderKabo();
}

function kaboCommand() {
  const s = kaboState;
  if (!kaboMine() || isSpectating || kaboBusy || kaboOver()) return null;
  if (s.phase === 'initial') {
    if (kaboMine().ready) return null;
    if (s.reveals.length === 2) return { kind: 'ready', label: 'Hide & ready', payload: {} };
    return kaboSelected.length === 2 ? { kind: 'initial', label: 'Peek at two', payload: { slotIds: [...kaboSelected] } } : null;
  }
  if (s.phase === 'draw' && s.matchAllowed && (kaboSelected.length || kaboTarget)) {
    if (kaboTarget && !kaboSelected.length) return null;
    return { kind: 'match', label: kaboTarget ? 'Discard & give' : 'Match & discard',
      payload: { targetId: kaboTarget || kaboSelected[0], slotId: kaboTarget ? kaboSelected[0] : '', discardId: s.discardId }, confirm: true };
  }
  if (!kaboTurn()) return null;
  if (s.phase === 'resolve') return kaboSelected.length
    ? { kind: 'replace', label: 'Replace card', payload: { slotId: kaboSelected[0] }, confirm: true }
    : s.fromDeck ? { kind: 'discard', label: 'Discard drawn', payload: {} } : null;
  if (s.phase === 'power') {
    const enough = s.power === 'own' ? kaboSelected.length : s.power === 'spy' ? !!kaboTarget : kaboSelected.length && kaboTarget;
    return enough ? { kind: 'power', label: s.power === 'swap' ? 'Swap cards' : 'Peek',
      payload: { slotId: kaboSelected[0] || '', targetId: kaboTarget }, confirm: s.power === 'swap' } : null;
  }
  if (s.phase === 'peek') return { kind: 'ack', label: 'Hide cards', payload: {} };
  return null;
}

function kaboPrompt() {
  const s = kaboState;
  if (isSpectating) return 'Spectating';
  if (s.phase === 'initial') return kaboMine().ready ? 'Waiting for players' : s.reveals.length ? 'Remember these cards, then hide them.' : 'Select two of your cards to peek.';
  if (s.phase === 'draw' && s.matchAllowed && (kaboTarget || kaboSelected.length)) {
    const target = s.players.find(p => p.cards.some(c => c.id === kaboTarget));
    return target ? (kaboSelected.length ? 'Discard ' + target.name + "'s match; give them your selected card." : 'Choose your card to give ' + target.name + '.') :
      'Discard your card only if its rank matches.';
  }
  if (!kaboTurn()) return s.matchAllowed ? 'Match the discard, or wait for the next draw.' : 'Waiting for ' + (s.players[s.currentPlayerIdx]?.name || 'players');
  if (s.phase === 'draw') return s.matchAllowed ? 'Draw, or match the discard.' : 'Choose a pile to draw.';
  if (s.phase === 'resolve') return s.callerId === playerId ? 'Kabo called. Finish your drawn card.' : s.fromDeck ? 'Replace your card or discard the draw.' : 'Replace one of your cards.';
  if (s.phase === 'power') return ({ own: 'Select your card to peek.', spy: "Select an opponent's card to peek.", swap: "Select your card and an opponent's to swap.", king: "Select your card and an opponent's to peek." })[s.power];
  if (s.phase === 'peek') return 'Remember the revealed cards. Hide when ready.';
  if (s.phase === 'decision') return 'King: swap the selected cards, or keep them.';
  return '';
}

function renderKabo() {
  if (!kaboState) return;
  const s = kaboState, mine = kaboMine(), over = kaboOver();
  document.getElementById('kabo-round').textContent = 'Round ' + s.round;
  const names = ids => ids.map(id => s.players.find(p => p.id === id)?.name || '').join(', ');
  const final = document.getElementById('kabo-final');
  final.hidden = !s.callerId || over;
  final.textContent = names([s.callerId]) + ' called Kabo · ' + s.finalTurns + ' final turns';
  const opponents = s.players.filter(p => p.id !== playerId);
  document.getElementById('kabo-opponents').innerHTML = opponents.map(p =>
    '<section class="kabo-seat"><div class="kabo-identity">' + playerBadge(p) + '<strong>' + esc(p.name) + '</strong></div>' +
    '<div class="kabo-mini-hand">' + p.cards.map((c, i) => kaboCard(c, p, i)).join('') + '</div>' +
    (!p.active ? '<small>Left</small>' : over ? '<small>Round: ' + p.roundScore + ' · Total: ' + p.totalScore + '</small>' : '') + '</section>').join('');
  document.getElementById('kabo-own-name').innerHTML = mine ? playerBadge(mine) + '<strong>' + esc(mine.name) + '</strong><small>(You)</small>' : '';
  document.getElementById('kabo-turn').textContent = kaboTurn() ? 'Your turn' : '';
  document.getElementById('kabo-hand').innerHTML = mine ? mine.cards.map((c, i) => kaboCard(c, mine, i)).join('') : '';
  document.getElementById('kabo-drawn').innerHTML = s.drawn ? additionalCardHTML(s.drawn) : '';
  document.getElementById('kabo-drawn').hidden = !s.drawn;
  document.getElementById('kabo-discard-card').innerHTML = s.discard ? additionalCardHTML(s.discard) : '<span>Empty</span>';
  document.getElementById('kabo-deck').disabled = !kaboTurn() || s.phase !== 'draw' || kaboBusy;
  document.getElementById('kabo-discard-pile').disabled = !kaboTurn() || s.phase !== 'draw' || !s.discard || kaboBusy;
  document.getElementById('kabo-prompt').textContent = over ? '' : kaboPrompt();
  document.getElementById('kabo-last-action').textContent = s.lastAction;
  const command = kaboCommand(), action = document.getElementById('kabo-action');
  action.textContent = command?.label || (s.phase === 'draw' ? 'Match & discard' : 'Select cards');
  action.disabled = !command;
  action.hidden = over || s.phase === 'decision';
  document.getElementById('kabo-call').hidden = !kaboTurn() || s.phase !== 'resolve' || !!s.callerId;
  document.getElementById('kabo-call').disabled = kaboBusy;
  document.getElementById('kabo-skip').hidden = !kaboTurn() || s.phase !== 'power';
  document.getElementById('kabo-decision').hidden = !kaboTurn() || s.phase !== 'decision';
  document.getElementById('kabo-clear').hidden = !kaboSelected.length && !kaboTarget;
  const results = document.getElementById('kabo-results');
  results.hidden = !over;
  results.innerHTML = over ? '<h3>' + esc(names(s.phase === 'game_over' ? s.winners : s.roundWinners)) +
    (s.phase === 'game_over' ? ' wins the match' : ' · lowest round score') + '</h3>' +
    s.players.filter(p => p.active).map(p => '<p>' + esc(p.name) + ': <strong>' + p.roundScore + '</strong> this round · <strong>' + p.totalScore + '</strong> total</p>').join('') : '';
  for (const name of ['next-round', 'end-game']) {
    const button = document.getElementById('kabo-' + name);
    button.hidden = s.phase !== 'round_over' || isSpectating || hostId !== playerId;
  }
  document.getElementById('kabo-return').hidden = s.phase !== 'game_over' || isSpectating;
}

function kaboSend(kind, payload = {}) {
  if (!kaboState || isSpectating || kaboBusy) return;
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    document.getElementById('kabo-prompt').textContent = 'Disconnected. Wait for reconnection.';
    return;
  }
  kaboBusy = true;
  send('kabo-' + kind, { ...payload, revision: kaboState.revision });
  renderKabo();
}

function kaboConfirm(kind, payload, label) {
  kaboPending = { kind, payload, revision: kaboState.revision };
  document.getElementById('kabo-confirm-title').textContent = label;
  document.getElementById('kabo-confirm-text').textContent = kind === 'match'
    ? 'An incorrect match adds one hidden penalty card. Confirm your selected cards?'
    : kind === 'call' ? 'Finish this drawn card, then everyone else gets one final turn.' : 'Confirm your selected cards?';
  document.getElementById('kabo-confirm-dialog').showModal();
}

document.getElementById('kabo-active').addEventListener('click', event => {
  const button = event.target.closest('[data-kabo-slot]');
  if (button && !button.disabled) kaboChoose(button.dataset.kaboOwner, button.dataset.kaboSlot);
});
document.getElementById('kabo-confirm-dialog').addEventListener('close', () => { kaboPending = null; });
function kaboPrimary() {
  const command = kaboCommand();
  if (!command) return;
  if (command.confirm) kaboConfirm(command.kind, command.payload, command.label + '?');
  else kaboSend(command.kind, command.payload);
}
function kaboConfirmAction() {
  const pending = kaboPending;
  document.getElementById('kabo-confirm-dialog').close();
  kaboPending = null;
  if (!pending || pending.revision !== kaboState?.revision) {
    document.getElementById('kabo-prompt').textContent = 'The table changed. Select again.';
    return;
  }
  kaboSend(pending.kind, pending.payload);
}
function kaboError(message) {
  kaboResetSelection();
  renderKabo();
  document.getElementById('kabo-prompt').textContent = message;
}
