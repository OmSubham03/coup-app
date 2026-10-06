// ========== UNO CARD GAME ==========

let unoState = null;
let unoSelectedCardId = '';
let unoColorChoiceForCard = '';

function openUNOHouseRule(rule) {
  const rules = {
    stacking: {
      title: 'Draw-card stacking house rule',
      description: 'Stack +2 on +2, +4 on +2, +4 on +4, or a +2 matching the color chosen after +4 on +4. The draw penalty accumulates until a player accepts it. This is an optional house rule, not standard UNO.'
    },
    multiSkip: {
      title: 'Multi Skip house rule',
      description: 'The next player can stack a Skip of any color. Continue until someone has no Skip; starting with that player, skip as many turns as the number of Skips piled, following the current direction. Skipped turns can wrap around the players. Emptying a hand still ends the round. This is an optional house rule, not standard UNO.'
    }
  };
  const selected = rules[rule];
  if (!selected) return;
  document.getElementById('uno-house-rule-title').textContent = selected.title;
  document.getElementById('uno-house-rule-description').textContent = selected.description;
  const dialog = document.getElementById('uno-house-rule-dialog');
  if (!dialog.open) dialog.showModal();
}

const UNO_COLORS = ['red', 'yellow', 'green', 'blue'];
const UNO_COLOR_NAMES = { red: 'Red', yellow: 'Yellow', green: 'Green', blue: 'Blue' };
const UNO_CARD_LABELS = { skip: 'SKIP', reverse: 'REVERSE', draw_two: '+2', wild: 'WILD', wild_draw_four: '+4' };
const UNO_CARD_MARKS = { skip: '⊘', reverse: '↻', draw_two: '+2', wild: 'WILD', wild_draw_four: '+4' };

function handleUNOStateUpdate(payload) {
  unoState = payload;
  const myIdx = unoState.players.findIndex(player => player.id === playerId);
  const myHand = myIdx >= 0 ? unoState.players[myIdx].cards : [];
  if (!myHand.some(card => card.id === unoSelectedCardId)) unoSelectedCardId = '';
  renderUNOGame();
}

function renderUNOGame() {
  if (!unoState) return;
  ['game-active', 'poker-active', 'ludo-active', 'nq-active', 'commune-active', 'tn-active', 'ht-active'].forEach(id => {
    document.getElementById(id).style.display = 'none';
  });
  document.getElementById('uno-active').style.display = '';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';

  const phase = document.getElementById('uno-phase-display');
  const houseRules = [];
  if (unoState.stackingEnabled) houseRules.push('Stacking on');
  if (unoState.multiSkipEnabled) houseRules.push('Multi Skip on');
  phase.textContent = unoState.phase === 'playing'
    ? `Round ${unoState.round} · ${houseRules.length ? houseRules.join(' · ') : 'Standard rules'}`
    : unoState.phase === 'round_over' ? `Round ${unoState.round} complete`
      : 'Match complete';
  document.getElementById('uno-turn-display').textContent =
    unoState.phase === 'playing' ? `Turn ${unoState.turnNumber + 1}` : '';
  document.getElementById('uno-table-area').innerHTML = renderUNOTable();
  document.getElementById('uno-action-area').innerHTML = renderUNOActions();
}

function renderUNOTable() {
  const myIdx = unoState.players.findIndex(player => player.id === playerId);
  const myTurn = myIdx >= 0 && myIdx === unoState.currentPlayerIdx && unoState.phase === 'playing';
  let html = '<div class="uno-card-table">';
  html += '<div class="uno-opponents-row">';
  unoState.players.forEach((player, index) => {
    if (index === myIdx) return;
    const initials = unoEsc(player.name.split(/\s+/).map(part => part[0]).slice(0, 2).join('').toUpperCase());
    const active = index === unoState.currentPlayerIdx;
    html += `<div class="uno-opponent${active ? ' active' : ''}"><div class="uno-avatar uno-avatar-${UNO_COLORS[index % UNO_COLORS.length]}">${initials}</div>`;
    html += `<div class="uno-opponent-name">${unoEsc(player.name)}</div><div class="uno-opponent-meta">${player.cards.length} cards · ${player.score} pts</div>`;
    if (unoState.uncalledUnoPlayerId === player.id && !isSpectating) {
      html += `<button class="uno-catch-btn" onclick="unoCatch('${unoEsc(player.id)}')">Catch UNO</button>`;
    }
    html += '</div>';
  });
  html += '</div>';

  const topCard = unoState.discardPile[unoState.discardPile.length - 1];
  const currentPlayer = unoState.players[unoState.currentPlayerIdx];
  html += '<div class="uno-table-center">';
  html += '<div class="uno-card-stacks">';
  html += `<div class="uno-card-stack"><span class="uno-pile-caption">DRAW PILE</span><button class="uno-card-back uno-draw-pile" ${!myTurn || unoState.pendingDraw > 0 || unoState.pendingSkips > 0 || unoState.players[myIdx]?.hasDrawn ? 'disabled' : ''} onclick="send('uno-draw-card')" aria-label="Draw a card"></button></div>`;
  html += `<div class="uno-card-stack"><span class="uno-pile-caption">ON TABLE</span>${topCard ? unoCardMarkup(topCard, false, false) : ''}</div>`;
  html += '</div>';
  html += `<div class="uno-turn-banner${myTurn ? ' my-turn' : ''}"><span>${unoState.awaitingInitialColor ? 'Choose the opening color' : myTurn ? 'Your turn' : `${unoEsc(currentPlayer?.name || 'Player')}'s turn`}</span><i class="uno-swatch ${unoState.currentColor || 'wild'}"></i><b>${unoState.pendingSkips > 0 ? `${unoState.pendingSkips} skip${unoState.pendingSkips === 1 ? '' : 's'}` : unoState.pendingDraw > 0 ? `+${unoState.pendingDraw} cards` : (UNO_COLOR_NAMES[unoState.currentColor] || 'Wild')}</b></div>`;
  html += '</div>';

  if (myIdx >= 0) {
    const me = unoState.players[myIdx];
    html += `<div class="uno-my-hand-header"><div class="uno-avatar uno-avatar-self">${unoEsc(me.name.split(/\s+/).map(part => part[0]).slice(0, 2).join('').toUpperCase())}</div><div><b>${unoEsc(me.name)}${me.id === playerId ? ' · You' : ''}</b><span>${me.cards.length} cards · ${me.score} points</span></div></div>`;
    html += '<div class="uno-hand" aria-label="Your hand">';
    me.cards.forEach(card => {
      const playable = myTurn && unoIsPlayable(card, me);
      html += unoCardMarkup(card, playable, card.id === unoSelectedCardId);
    });
    html += '</div>';
  } else {
    html += '<div class="uno-my-hand-header"><b>Spectating</b></div>';
  }
  if (unoState.lastAction) html += `<p class="uno-last-action">${unoEsc(unoState.lastAction)}</p>`;
  html += '</div>';
  return html;
}

function unoCardMarkup(card, playable, selected) {
  const label = UNO_CARD_LABELS[card.kind] || card.value;
  const mark = UNO_CARD_MARKS[card.kind] || card.value;
  const color = card.color === 'wild' ? 'wild' : card.color;
  const corner = card.kind === 'number' ? card.value : mark;
  return `<button class="uno-card uno-card-${color}${playable ? ' playable' : ' disabled'}${selected ? ' selected' : ''}" ${playable ? `onclick="unoSelectCard('${unoEsc(card.id)}')"` : 'disabled'} aria-label="${unoEsc(UNO_COLOR_NAMES[card.color] || 'Wild')} ${unoEsc(label)}">
    <span class="uno-corner top">${unoEsc(corner)}</span><span class="uno-card-oval"></span><span class="uno-card-face">${unoEsc(mark)}</span><span class="uno-corner bottom">${unoEsc(corner)}</span>
  </button>`;
}

function unoIsPlayable(card, player) {
  if (unoState.pendingSkips > 0) return unoState.multiSkipEnabled && card.kind === 'skip';
  if (unoState.pendingDraw > 0) {
    if (!unoState.stackingEnabled) return false;
    const top = unoState.discardPile[unoState.discardPile.length - 1];
    if (top.kind === 'draw_two') return card.kind === 'draw_two' || card.kind === 'wild_draw_four';
    if (top.kind === 'wild_draw_four') {
      return card.kind === 'wild_draw_four' || (card.kind === 'draw_two' && card.color === unoState.currentColor);
    }
    return false;
  }
  if (player.hasDrawn && player.drawnCardId !== card.id) return false;
  if (card.kind === 'wild') return true;
  if (card.kind === 'wild_draw_four') {
    return !player.cards.some(other => other.id !== card.id && other.color === unoState.currentColor);
  }
  const top = unoState.discardPile[unoState.discardPile.length - 1];
  return card.color === unoState.currentColor || card.value === top.value;
}

function renderUNOActions() {
  const myIdx = unoState.players.findIndex(player => player.id === playerId);
  const me = myIdx >= 0 ? unoState.players[myIdx] : null;
  if (unoState.phase === 'game_over') {
    return `<div class="uno-result"><h2>${unoEsc(unoState.winnerName)} wins the match!</h2><p>${unoEsc(unoState.lastAction)}</p><button class="btn btn-blue" onclick="send('return-to-lobby')">Return to Lobby</button></div>`;
  }
  if (unoState.phase === 'round_over') {
    return `<div class="uno-result"><h2>${unoEsc(unoState.winnerName)} wins round ${unoState.round}</h2><p>${unoEsc(unoState.lastAction)}</p>${hostId === playerId ? '<button class="btn btn-green" onclick="send(\'uno-next-round\')">Deal Next Round</button>' : '<p>Waiting for the host to deal the next round.</p>'}</div>`;
  }
  if (!me) return '<div class="uno-waiting">Spectating</div>';
  if (unoState.awaitingInitialColor) {
    if (myIdx !== unoState.currentPlayerIdx) return '<div class="uno-waiting">Waiting for the opening color choice…</div>';
    return `<div class="uno-color-prompt">Choose the opening color${unoColorButtons('unoChooseInitialColor')}</div>`;
  }
  if (unoState.uncalledUnoPlayerId === playerId) {
    return '<button class="uno-action-btn uno-action-yellow" onclick="send(\'uno-say-uno\')">Call UNO</button>';
  }
  if (myIdx !== unoState.currentPlayerIdx) {
    const current = unoState.players[unoState.currentPlayerIdx];
    return `<div class="uno-waiting">${unoEsc(current?.name || 'Another player')}'s turn…</div>`;
  }

  let html = '';
  const selected = me.cards.find(card => card.id === unoSelectedCardId);
  if (unoState.pendingSkips > 0) {
    html += `<div class="uno-waiting">Skip chain: ${unoState.pendingSkips}. Play a Skip to continue.</div>`;
    if (selected && unoIsPlayable(selected, me)) html += unoPlayButton(selected);
  } else if (unoState.pendingDraw > 0) {
    html += `<button class="uno-action-btn uno-action-red" onclick="send('uno-accept-penalty')">Draw ${unoState.pendingDraw} cards</button>`;
    if (unoState.challengeAvailable) html += '<button class="uno-action-btn uno-action-blue" onclick="send(\'uno-challenge-wild-draw-four\')">Challenge +4</button>';
    if (selected && unoIsPlayable(selected, me)) html += unoPlayButton(selected);
  } else {
    if (!me.hasDrawn) html += '<button class="uno-action-btn uno-action-blue" onclick="send(\'uno-draw-card\')">Draw a card</button>';
    if (selected && unoIsPlayable(selected, me)) html += unoPlayButton(selected);
    if (me.hasDrawn) html += '<button class="uno-action-btn" onclick="send(\'uno-end-turn\')">End turn</button>';
  }
  if (me.cards.length === 1 && !me.saidUno) html += '<button class="uno-action-btn uno-action-yellow" onclick="send(\'uno-say-uno\')">Call UNO</button>';
  if (!html) html = '<div class="uno-waiting">Select a playable card or draw.</div>';
  if (unoColorChoiceForCard) html += `<div class="uno-color-prompt">Choose a color${unoColorButtons('unoChoosePlayedColor')}</div>`;
  return `<div class="uno-actions">${html}</div>`;
}

function unoPlayButton(card) {
  if (card.kind === 'wild' || card.kind === 'wild_draw_four') {
    return `<button class="uno-action-btn uno-action-green" onclick="unoColorChoiceForCard='${unoEsc(card.id)}';renderUNOGame()">Play ${unoEsc(UNO_CARD_LABELS[card.kind])}</button>`;
  }
  return `<button class="uno-action-btn uno-action-green" onclick="send('uno-play-card',{cardId:'${unoEsc(card.id)}'})">Play ${unoEsc(UNO_CARD_LABELS[card.kind] || card.value)}</button>`;
}

function unoColorButtons(handler) {
  return `<div class="uno-color-choices">${UNO_COLORS.map(color => `<button class="uno-color-choice uno-${color}" onclick="${handler}('${color}')" aria-label="Choose ${UNO_COLOR_NAMES[color]}"></button>`).join('')}</div>`;
}

function unoSelectCard(cardId) {
  unoSelectedCardId = unoSelectedCardId === cardId ? '' : cardId;
  unoColorChoiceForCard = '';
  renderUNOGame();
}

function unoChoosePlayedColor(color) {
  const cardId = unoColorChoiceForCard;
  unoColorChoiceForCard = '';
  send('uno-play-card', { cardId, chosenColor: color });
}

function unoChooseInitialColor(color) {
  send('uno-choose-initial-color', { color });
}

function unoCatch(targetPlayerId) {
  send('uno-catch-uno', { targetPlayerId });
}

function switchUNOTab(tab) {
  document.getElementById('uno-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('uno-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  document.querySelectorAll('#uno-active .tab').forEach(button => button.classList.remove('active'));
  document.getElementById('unotab-' + tab).classList.add('active');
  if (tab === 'chat') document.getElementById('unotab-chat').classList.remove('chat-unread');
}

function unoEsc(value) {
  const node = document.createElement('div');
  node.textContent = value || '';
  return node.innerHTML;
}
