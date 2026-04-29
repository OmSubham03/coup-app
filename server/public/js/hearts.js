// ========== HEARTS CARD GAME ==========

let htState = null;
let htSelectedCards = []; // for passing (multiple) or playing (single)
let htPrevTricksPlayed = -1;
let htAnimating = false;
let htAnimTrick = null;
let htAnimWinnerId = null;
let htPendingState = null;

const HT_SUIT_SYMBOLS = { hearts: '♥', diamonds: '♦', clubs: '♣', spades: '♠' };
const HT_RANK_NAMES = { 2:'2',3:'3',4:'4',5:'5',6:'6',7:'7',8:'8',9:'9',10:'10',11:'J',12:'Q',13:'K',14:'A' };

function handleHTStateUpdate(payload) {
  htSelectedCards = [];
  if (htAnimating) { htPendingState = payload; htPrevTricksPlayed = payload.tricksPlayed || 0; return; }

  const newTP = payload.tricksPlayed || 0;
  const prev = htPrevTricksPlayed;

  // Detect completed trick for animation
  if (payload.phase === 'playing' && prev >= 0 && newTP > prev && payload.completedTricks?.length > 0) {
    const lt = payload.completedTricks[payload.completedTricks.length - 1];
    if (lt?.cards?.length === 4) {
      htAnimTrick = lt; htAnimWinnerId = lt.winnerId; htPendingState = payload;
      htAnimating = true; htPrevTricksPlayed = newTP; htRunTrickAnimation(); return;
    }
  }
  if ((payload.phase === 'hand_over' || payload.phase === 'game_over') && prev >= 0 && prev < 13 && payload.completedTricks?.length > 0) {
    const lt = payload.completedTricks[payload.completedTricks.length - 1];
    if (lt?.cards?.length === 4) {
      htAnimTrick = lt; htAnimWinnerId = lt.winnerId; htPendingState = payload;
      htAnimating = true; htPrevTricksPlayed = payload.tricksPlayed || 0; htRunTrickAnimation(); return;
    }
  }

  htPrevTricksPlayed = newTP;
  htState = payload;
  renderHTGame();
}

function htRunTrickAnimation() {
  const myIdx = htPendingState.players.findIndex(p => p.id === playerId);
  const seatMap = myIdx >= 0 ? [(myIdx)%4,(myIdx+1)%4,(myIdx+2)%4,(myIdx+3)%4] : [0,1,2,3];
  const positions = ['bottom','left','top','right'];
  const winnerPI = htPendingState.players.findIndex(p => p.id === htAnimWinnerId);
  const winnerPos = positions[seatMap.indexOf(winnerPI)] || 'bottom';

  const centerEl = document.querySelector('.ht-center');
  if (!centerEl) { htAnimating = false; htState = htPendingState; htPendingState = null; renderHTGame(); return; }

  let html = '';
  for (let ti = 0; ti < htAnimTrick.cards.length; ti++) {
    const tc = htAnimTrick.cards[ti];
    if (!tc.rank) continue;
    const tpi = htPendingState.players.findIndex(p => p.id === htAnimTrick.playerIds[ti]);
    const posName = positions[seatMap.indexOf(tpi)] || 'bottom';
    html += `<div class="ht-trick-card pos-${posName} ${tc.suit}" data-anim-card="${ti}">
      <span class="ht-card-rank">${HT_RANK_NAMES[tc.rank]}</span><span class="ht-card-suit">${HT_SUIT_SYMBOLS[tc.suit]}</span></div>`;
  }
  centerEl.innerHTML = html;

  const actionArea = document.getElementById('ht-action-area');
  if (actionArea) {
    const wn = htPendingState.players[winnerPI]?.name || '?';
    actionArea.innerHTML = `<div class="ht-waiting" style="color:#dc2626;font-weight:700">${htEsc(wn)} wins this trick!</div>`;
  }

  setTimeout(() => {
    centerEl.querySelectorAll('[data-anim-card]').forEach(card => {
      card.classList.add('ht-anim-collect','ht-anim-to-' + winnerPos);
    });
    setTimeout(() => {
      htAnimating = false; htState = htPendingState; htPendingState = null;
      htAnimTrick = null; htAnimWinnerId = null; renderHTGame();
    }, 700);
  }, 3000);
}

function renderHTGame() {
  if (!htState) return;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('nq-active').style.display = 'none';
  document.getElementById('commune-active').style.display = 'none';
  document.getElementById('tn-active').style.display = 'none';
  document.getElementById('ht-active').style.display = '';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';

  const phaseEl = document.getElementById('ht-phase-display');
  const turnEl = document.getElementById('ht-turn-display');
  const tableArea = document.getElementById('ht-table-area');
  const actionArea = document.getElementById('ht-action-area');
  const myIdx = htState.players.findIndex(p => p.id === playerId);

  switch (htState.phase) {
    case 'passing': phaseEl.textContent = `Hand ${htState.handNumber} — Pass ${htState.passDirName}`; break;
    case 'playing': phaseEl.textContent = `Hand ${htState.handNumber} — Trick ${htState.tricksPlayed + 1}/13`; break;
    case 'hand_over': phaseEl.textContent = 'Hand Over'; break;
    case 'game_over': phaseEl.textContent = 'Game Over'; break;
    default: phaseEl.textContent = htState.phase;
  }
  turnEl.textContent = `Hand ${htState.handNumber}`;

  let html = renderHTScoreBar(myIdx);

  if (htState.phase === 'passing') {
    html += renderHTPassingBoard(myIdx);
    tableArea.innerHTML = html;
    actionArea.innerHTML = renderHTPassingActions(myIdx);
    return;
  }
  if (htState.phase === 'hand_over' || htState.phase === 'game_over') {
    html += renderHTResult(myIdx);
    tableArea.innerHTML = html;
    actionArea.innerHTML = '';
    return;
  }

  html += renderHTPlayingBoard(myIdx);
  tableArea.innerHTML = html;
  actionArea.innerHTML = renderHTPlayingActions(myIdx);
}

function renderHTScoreBar(myIdx) {
  let html = '<div class="ht-score-bar">';
  for (let i = 0; i < 4; i++) {
    const p = htState.players[i];
    const s = p.totalScore;
    const cls = s < 30 ? 'low' : (s < 70 ? 'mid' : 'high');
    html += `<div class="ht-score-item">
      <div class="ht-score-name${i === myIdx ? ' me' : ''}">${htEsc(p.name)}</div>
      <div class="ht-score-val ${cls}">${s}</div>
      <div class="ht-hand-pts">+${p.handPoints}</div>
    </div>`;
  }
  html += '</div>';
  return html;
}

function renderHTPassingBoard(myIdx) {
  let html = '<div style="text-align:center;padding:12px">';
  html += `<div style="color:#f59e0b;font-size:14px;font-weight:700;margin-bottom:8px">Pass 3 cards ${htState.passDirName}</div>`;
  if (myIdx >= 0) {
    const me = htState.players[myIdx];
    if (me.hasPassed) {
      html += '<div class="ht-waiting">Waiting for others to pass...</div>';
    } else {
      html += `<div style="color:#94a3b8;font-size:12px;margin-bottom:8px">Select 3 cards to pass</div>`;
    }
    html += '<div class="ht-my-cards">';
    for (const c of me.cards) {
      if (c.id.startsWith('hidden_')) continue;
      const sel = htSelectedCards.includes(c.id);
      const isPenalty = c.suit === 'hearts' || (c.suit === 'spades' && c.rank === 12);
      html += `<div class="ht-card ${c.suit}${isPenalty ? ' penalty' : ''}${sel ? ' pass-selected' : ''}${me.hasPassed ? ' disabled' : ''}" onclick="htTogglePassCard('${c.id}')">
        <span class="ht-card-rank">${HT_RANK_NAMES[c.rank]}</span><span class="ht-card-suit">${HT_SUIT_SYMBOLS[c.suit]}</span></div>`;
    }
    html += '</div>';
  }
  html += '</div>';
  return html;
}

function renderHTPassingActions(myIdx) {
  if (myIdx < 0) return '';
  const me = htState.players[myIdx];
  if (me.hasPassed) return '';
  return `<div style="text-align:center">
    <button class="ht-pass-btn" ${htSelectedCards.length !== 3 ? 'disabled' : ''} onclick="htConfirmPass()">Pass ${htSelectedCards.length}/3 Cards</button>
  </div>`;
}

function renderHTPlayingBoard(myIdx) {
  const seatMap = myIdx >= 0 ? [(myIdx)%4,(myIdx+1)%4,(myIdx+2)%4,(myIdx+3)%4] : [0,1,2,3];
  const positions = ['bottom','left','top','right'];
  const posClasses = ['ht-player-bottom','ht-player-left','ht-player-top','ht-player-right'];

  let html = '<div class="ht-board">';

  // Info bar
  html += '<div style="position:absolute;top:62px;left:50%;transform:translateX(-50%);z-index:1;text-align:center">';
  html += `<div class="ht-info-bar">Trick ${htState.tricksPlayed + 1}/13`;
  if (htState.heartsBroken) html += ' · <span class="ht-hearts-broken">♥ Broken</span>';
  html += '</div></div>';

  // Other players
  for (let s = 1; s <= 3; s++) {
    const pi = seatMap[s];
    const p = htState.players[pi];
    const isActive = pi === htState.currentPlayerIdx;
    const cardCount = p.cards ? p.cards.length : 0;
    const isVertical = s === 1 || s === 3;
    html += `<div class="ht-player-spot ${posClasses[s]}">`;
    html += `<div class="ht-player-name${isActive ? ' active' : ''}">${htEsc(p.name)}(+${p.handPoints})</div>`;
    html += `<div class="ht-folded-cards${isVertical ? ' vertical' : ''}">`;
    for (let ci = 0; ci < cardCount; ci++) html += '<div class="ht-folded-card"></div>';
    html += '</div></div>';
  }

  // Center trick
  html += '<div class="ht-center">';
  if (htState.currentTrick?.cards) {
    for (let ti = 0; ti < htState.currentTrick.cards.length; ti++) {
      const tc = htState.currentTrick.cards[ti];
      if (!tc.rank) continue;
      const tpi = htState.players.findIndex(p => p.id === htState.currentTrick.playerIds[ti]);
      const posName = positions[seatMap.indexOf(tpi)] || 'bottom';
      html += `<div class="ht-trick-card pos-${posName} ${tc.suit}">
        <span class="ht-card-rank">${HT_RANK_NAMES[tc.rank]}</span><span class="ht-card-suit">${HT_SUIT_SYMBOLS[tc.suit]}</span></div>`;
    }
  }
  html += '</div>';

  // My cards
  if (myIdx >= 0) {
    const me = htState.players[myIdx];
    const isMyTurn = myIdx === htState.currentPlayerIdx;
    html += '<div class="ht-player-spot ht-player-bottom">';
    html += `<div class="ht-player-name${isMyTurn ? ' active' : ''}">${htEsc(me.name)}(+${me.handPoints})</div>`;
    html += '<div class="ht-my-cards">';
    const leadSuit = htState.currentTrick?.leadSuit;
    const hasSuit = leadSuit ? me.cards.some(c => c.suit === leadSuit) : false;
    const isFirstTrick = htState.tricksPlayed === 0;
    for (const c of me.cards) {
      if (c.id.startsWith('hidden_')) continue;
      let canPlay = isMyTurn;
      if (canPlay && isFirstTrick && htState.currentTrick?.cards?.length === 0) {
        canPlay = c.suit === 'clubs' && c.rank === 2;
      } else if (canPlay && leadSuit && hasSuit) {
        canPlay = c.suit === leadSuit;
      }
      const isPenalty = c.suit === 'hearts' || (c.suit === 'spades' && c.rank === 12);
      const selected = htSelectedCards.includes(c.id);
      html += `<div class="ht-card ${c.suit}${isPenalty ? ' penalty' : ''}${!canPlay ? ' disabled' : ''}${selected ? ' selected' : ''}" onclick="htSelectCard('${c.id}')">
        <span class="ht-card-rank">${HT_RANK_NAMES[c.rank]}</span><span class="ht-card-suit">${HT_SUIT_SYMBOLS[c.suit]}</span></div>`;
    }
    html += '</div></div>';
  }
  html += '</div>';
  return html;
}

function renderHTPlayingActions(myIdx) {
  if (myIdx < 0) return '<div class="ht-waiting">Spectating...</div>';
  if (myIdx !== htState.currentPlayerIdx) {
    return `<div class="ht-waiting">${htEsc(htState.players[htState.currentPlayerIdx].name)}'s turn...</div>`;
  }
  return `<div style="text-align:center">
    <button class="ht-play-btn" ${htSelectedCards.length !== 1 ? 'disabled' : ''} onclick="htPlayCard()">Play Card</button>
  </div>`;
}

function renderHTResult(myIdx) {
  let html = '<div class="ht-result">';
  if (htState.phase === 'game_over') {
    html += `<h3>🏆 Game Over!</h3><p style="color:#dc2626;font-weight:700;font-size:18px">${htEsc(htState.winner)} wins!</p>`;
  } else {
    html += `<h3>Hand ${htState.handNumber} Complete</h3>`;
  }
  html += '<table class="ht-result-table"><tr><th>Player</th><th>Hand</th><th>Total</th></tr>';
  for (const p of htState.players) {
    html += `<tr><td>${htEsc(p.name)}${p.id === playerId ? ' (You)' : ''}</td><td>+${p.handPoints}</td><td>${p.totalScore}</td></tr>`;
  }
  html += '</table>';
  if (htState.lastAction.includes('moon')) html += `<p style="color:#f59e0b;font-weight:700">${htEsc(htState.lastAction)}</p>`;
  if (htState.phase === 'hand_over') html += `<button class="ht-next-btn" onclick="htNextHand()">Next Hand</button>`;
  if (htState.phase === 'game_over') html += `<button class="ht-next-btn" style="background:#3b82f6" onclick="send('return-to-lobby')">Return to Lobby</button>`;
  html += '</div>';
  return html;
}

// Actions
function htTogglePassCard(cardId) {
  const idx = htSelectedCards.indexOf(cardId);
  if (idx >= 0) { htSelectedCards.splice(idx, 1); }
  else if (htSelectedCards.length < 3) { htSelectedCards.push(cardId); }
  renderHTGame();
}

function htConfirmPass() {
  if (htSelectedCards.length !== 3) return;
  send('ht-pass-cards', { cardIds: htSelectedCards });
  htSelectedCards = [];
}

function htSelectCard(cardId) {
  htSelectedCards = htSelectedCards[0] === cardId ? [] : [cardId];
  renderHTGame();
}

function htPlayCard() {
  if (htSelectedCards.length !== 1) return;
  send('ht-play-card', { cardId: htSelectedCards[0] });
  htSelectedCards = [];
}

function htNextHand() {
  htPrevTricksPlayed = -1;
  send('ht-next-hand');
}

function switchHTTab(tab) {
  document.getElementById('ht-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('ht-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  document.querySelectorAll('#ht-active .tab').forEach(t => t.classList.remove('active'));
  document.getElementById('httab-' + tab).classList.add('active');
  if (tab === 'chat') { const el = document.getElementById('httab-chat'); if (el) el.classList.remove('chat-unread'); }
}

function htEsc(s) { const d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML; }
