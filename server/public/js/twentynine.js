// ========== 29 CARD GAME ==========

let tnState = null;
let tnSelectedCard = null;
let tnShowPoints = false;
let tnPrevTricksPlayed = -1;
let tnAnimating = false;
let tnAnimTrick = null; // saved trick for animation
let tnAnimWinnerId = null;
let tnPendingState = null;

const TN_SUIT_SYMBOLS = { hearts: '♥', diamonds: '♦', clubs: '♣', spades: '♠' };
const TN_SUIT_NAMES = { hearts: 'Hearts', diamonds: 'Diamonds', clubs: 'Clubs', spades: 'Spades' };

function handleTNStateUpdate(payload) {
  tnSelectedCard = null;

  // If currently animating, just update the pending state
  if (tnAnimating) {
    tnPendingState = payload;
    tnPrevTricksPlayed = payload.tricksPlayed || 0;
    return;
  }

  // Detect if a trick was just completed
  const newTricksPlayed = payload.tricksPlayed || 0;
  const prevTricks = tnPrevTricksPlayed;

  if (payload.phase === 'playing' && prevTricks >= 0 && newTricksPlayed > prevTricks && payload.completedTricks && payload.completedTricks.length > 0) {
    // A trick just completed — animate before showing new state
    const lastTrick = payload.completedTricks[payload.completedTricks.length - 1];
    if (lastTrick && lastTrick.cards && lastTrick.cards.length === 4) {
      tnAnimTrick = lastTrick;
      tnAnimWinnerId = lastTrick.winnerId;
      tnPendingState = payload;
      tnAnimating = true;
      tnPrevTricksPlayed = newTricksPlayed;
      tnRunTrickAnimation();
      return;
    }
  }

  // Also detect round_over / game_over after last trick
  if ((payload.phase === 'round_over' || payload.phase === 'game_over') && prevTricks >= 0 && prevTricks < 8 && payload.completedTricks && payload.completedTricks.length > 0) {
    const lastTrick = payload.completedTricks[payload.completedTricks.length - 1];
    if (lastTrick && lastTrick.cards && lastTrick.cards.length === 4) {
      tnAnimTrick = lastTrick;
      tnAnimWinnerId = lastTrick.winnerId;
      tnPendingState = payload;
      tnAnimating = true;
      tnPrevTricksPlayed = payload.tricksPlayed || 0;
      tnRunTrickAnimation();
      return;
    }
  }

  tnPrevTricksPlayed = newTricksPlayed;
  tnState = payload;
  renderTNGame();
}

function tnRunTrickAnimation() {
  // Render the board with the 4 completed trick cards in the center
  // We use the OLD state's board but inject the completed trick's cards into the center
  // Actually we need to render the board based on the PENDING state but with the trick cards visible
  const myIdx = tnPendingState.players.findIndex(p => p.id === playerId);
  const seatMap = myIdx >= 0 ? [
    (myIdx + 0) % 4,
    (myIdx + 1) % 4,
    (myIdx + 2) % 4,
    (myIdx + 3) % 4
  ] : [0, 1, 2, 3];
  const positions = ['bottom', 'left', 'top', 'right'];

  // Find winner seat position
  const winnerPlayerIdx = tnPendingState.players.findIndex(p => p.id === tnAnimWinnerId);
  const winnerSeat = seatMap.indexOf(winnerPlayerIdx);
  const winnerPos = positions[winnerSeat] || 'bottom';

  // Build trick cards HTML in the center area
  const centerEl = document.querySelector('.tn-center');
  if (!centerEl) {
    // Board not rendered yet, just apply state directly
    tnAnimating = false;
    tnState = tnPendingState;
    tnPendingState = null;
    renderTNGame();
    return;
  }

  // Render the 4 cards in plus pattern
  let cardsHtml = '';
  for (let ti = 0; ti < tnAnimTrick.cards.length; ti++) {
    const tc = tnAnimTrick.cards[ti];
    if (!tc.rank) continue;
    const tPlayerID = tnAnimTrick.playerIds[ti];
    const tPlayerIdx = tnPendingState.players.findIndex(p => p.id === tPlayerID);
    const seatPos = seatMap.indexOf(tPlayerIdx);
    const posName = positions[seatPos] || 'bottom';
    cardsHtml += `<div class="tn-trick-card pos-${posName} ${tc.suit}" data-anim-card="${ti}">
      <span class="tn-card-rank">${tc.rank}</span>
      <span class="tn-card-suit">${TN_SUIT_SYMBOLS[tc.suit]}</span>
    </div>`;
  }
  centerEl.innerHTML = cardsHtml;

  // Show winner indicator
  const winnerName = tnPendingState.players[winnerPlayerIdx]?.name || '?';
  const actionArea = document.getElementById('tn-action-area');
  if (actionArea) {
    actionArea.innerHTML = `<div class="tn-waiting" style="color:#22c55e;font-weight:700">${tnEsc(winnerName)} wins this trick!</div>`;
  }

  // Phase 1: Wait 3 seconds showing all 4 cards
  setTimeout(() => {
    // Phase 2: Animate cards flipping and moving to winner
    const cards = centerEl.querySelectorAll('[data-anim-card]');
    cards.forEach(card => {
      card.classList.add('tn-anim-collect');
      card.classList.add('tn-anim-to-' + winnerPos);
    });

    // Phase 3: After animation finishes (~0.6s), render the real new state
    setTimeout(() => {
      tnAnimating = false;
      tnState = tnPendingState;
      tnPendingState = null;
      tnAnimTrick = null;
      tnAnimWinnerId = null;
      renderTNGame();
    }, 700);
  }, 3000);
}

function renderTNGame() {
  if (!tnState) return;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('nq-active').style.display = 'none';
  document.getElementById('commune-active').style.display = 'none';
  document.getElementById('tn-active').style.display = '';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';

  const phaseEl = document.getElementById('tn-phase-display');
  const turnEl = document.getElementById('tn-turn-display');
  const tableArea = document.getElementById('tn-table-area');
  const actionArea = document.getElementById('tn-action-area');

  const myIdx = tnState.players.findIndex(p => p.id === playerId);
  const me = myIdx >= 0 ? tnState.players[myIdx] : null;

  // Phase display
  switch (tnState.phase) {
    case 'bidding': phaseEl.textContent = 'Bidding'; break;
    case 'trump_select': phaseEl.textContent = 'Select Trump'; break;
    case 'playing': phaseEl.textContent = `Round ${tnState.round} — Trick ${tnState.tricksPlayed + 1}/8`; break;
    case 'round_over': phaseEl.textContent = 'Round Over'; break;
    case 'game_over': phaseEl.textContent = 'Game Over'; break;
    default: phaseEl.textContent = tnState.phase;
  }
  turnEl.textContent = `Round ${tnState.round}`;

  // Build table
  let html = '';

  // Score bar
  html += renderTNScoreBar();

  if (tnState.phase === 'bidding') {
    html += renderTNBiddingBoard(myIdx);
    tableArea.innerHTML = html;
    actionArea.innerHTML = renderTNBiddingActions(myIdx);
    return;
  }

  if (tnState.phase === 'trump_select') {
    html += renderTNBiddingBoard(myIdx);
    tableArea.innerHTML = html;
    actionArea.innerHTML = renderTNTrumpSelectActions(myIdx);
    return;
  }

  if (tnState.phase === 'round_over' || tnState.phase === 'game_over') {
    html += renderTNRoundResult(myIdx);
    tableArea.innerHTML = html;
    actionArea.innerHTML = '';
    return;
  }

  // Playing phase - main board
  html += renderTNPlayingBoard(myIdx);
  tableArea.innerHTML = html;
  actionArea.innerHTML = renderTNPlayingActions(myIdx);
}

function renderTNScoreBar() {
  const t0 = `${tnEsc(tnState.players[0].name)} & ${tnEsc(tnState.players[2].name)}`;
  const t1 = `${tnEsc(tnState.players[1].name)} & ${tnEsc(tnState.players[3].name)}`;
  const s0 = tnState.teamScore[0];
  const s1 = tnState.teamScore[1];

  return `<div class="tn-score-bar">
    <div class="tn-team-score">
      <div class="tn-team-label">Team A</div>
      <div class="tn-game-score ${s0 > 0 ? 'positive' : (s0 < 0 ? 'negative' : 'zero')}">${s0 > 0 ? '+' : ''}${s0}</div>
      ${renderTNPips(s0)}
    </div>
    <div style="text-align:center">
      <div class="tn-vs">VS</div>
      <button class="tn-points-btn" onclick="tnTogglePoints()">Points</button>
    </div>
    <div class="tn-team-score">
      <div class="tn-team-label">Team B</div>
      <div class="tn-game-score ${s1 > 0 ? 'positive' : (s1 < 0 ? 'negative' : 'zero')}">${s1 > 0 ? '+' : ''}${s1}</div>
      ${renderTNPips(s1)}
    </div>
  </div>`;
}

function renderTNPips(score) {
  let html = '<div class="tn-team-pips">';
  for (let i = 0; i < 6; i++) {
    if (score > 0 && i < score) {
      html += '<div class="tn-pip red"></div>';
    } else if (score < 0 && i < -score) {
      html += '<div class="tn-pip black"></div>';
    } else {
      html += '<div class="tn-pip"></div>';
    }
  }
  html += '</div>';
  return html;
}

function renderTNBiddingBoard(myIdx) {
  let html = '<div style="text-align:center;padding:16px">';
  // Show each player's card count and name
  for (let i = 0; i < 4; i++) {
    const p = tnState.players[i];
    const isActive = i === tnState.currentPlayerIdx;
    const isBidder = tnState.bidderPlayerId === p.id && tnState.currentBid > 0;
    html += `<div style="margin:4px 0;padding:6px;border-radius:8px;${isActive ? 'background:rgba(251,191,36,0.1);border:1px solid #fbbf24' : ''}" >
      <span class="tn-player-name${isActive ? ' active' : ''}${isBidder ? ' bidder' : ''}">
        ${tnEsc(p.name)}${i === myIdx ? ' (You)' : ''}${p.team === tnState.players[myIdx]?.team ? ' 🤝' : ''}
        ${isBidder ? ` — Bid: ${tnState.currentBid}` : ''}
      </span>
    </div>`;
  }

  // Show my cards
  if (myIdx >= 0) {
    html += '<div style="margin-top:16px"><div style="color:#94a3b8;font-size:12px;margin-bottom:4px">Your Cards</div>';
    html += '<div class="tn-my-cards">';
    for (const c of tnState.players[myIdx].cards) {
      if (c.id.startsWith('hidden_')) continue;
      html += renderTNCardHTML(c, false, false);
    }
    html += '</div></div>';
  }

  html += '</div>';
  return html;
}

function renderTNBiddingActions(myIdx) {
  if (myIdx < 0 || myIdx !== tnState.currentPlayerIdx) {
    return `<div class="tn-waiting">${tnEsc(tnState.players[tnState.currentPlayerIdx].name)} is bidding...</div>`;
  }

  const minBid = tnState.currentBid > 0 ? tnState.currentBid + 1 : tnState.minBid;
  const isFirstBidder = tnState.currentBid === 0 && myIdx === (tnState.dealerIdx + 1) % 4;

  let html = '<div class="tn-bid-area">';
  html += `<div class="tn-bid-info">Current bid: ${tnState.currentBid || 'None'}</div>`;
  html += '<div class="tn-bid-buttons">';
  for (let b = minBid; b <= 28; b++) {
    html += `<button class="tn-bid-btn" onclick="tnBid(${b})">${b}</button>`;
  }
  html += '</div>';
  if (!isFirstBidder || tnState.currentBid > 0) {
    html += `<button class="tn-pass-btn" onclick="tnPass()">Pass</button>`;
  }
  html += '</div>';
  return html;
}

function renderTNTrumpSelectActions(myIdx) {
  if (myIdx < 0 || myIdx !== tnState.bidderIdx) {
    return `<div class="tn-waiting">${tnEsc(tnState.players[tnState.bidderIdx].name)} is selecting trump...</div>`;
  }

  // Show suits that the bidder has cards for
  const mySuits = new Set();
  for (const c of tnState.players[myIdx].cards) {
    if (!c.id.startsWith('hidden_')) mySuits.add(c.suit);
  }

  let html = '<div class="tn-trump-select">';
  html += '<div style="color:#e2e8f0;font-size:14px;margin-bottom:8px">Choose trump suit (kept hidden)</div>';
  html += '<div class="tn-trump-suits">';
  for (const suit of ['hearts', 'diamonds', 'clubs', 'spades']) {
    const disabled = !mySuits.has(suit);
    html += `<button class="tn-suit-btn ${suit}" ${disabled ? 'disabled style="opacity:0.3;pointer-events:none"' : ''} onclick="tnSelectTrump('${suit}')">${TN_SUIT_SYMBOLS[suit]}</button>`;
  }
  html += '</div></div>';
  return html;
}

function renderTNPlayingBoard(myIdx) {
  // Map seat indices relative to me (bottom=me, left=next, top=across, right=prev)
  const seatMap = myIdx >= 0 ? [
    (myIdx + 0) % 4, // bottom (me)
    (myIdx + 1) % 4, // left
    (myIdx + 2) % 4, // top (partner)
    (myIdx + 3) % 4  // right
  ] : [0, 1, 2, 3];

  const positions = ['bottom', 'left', 'top', 'right'];
  const posClasses = ['tn-player-bottom', 'tn-player-left', 'tn-player-top', 'tn-player-right'];

  let html = '<div class="tn-board">';

  // Info bar (bid, trump, etc) — positioned below the top player
  html += '<div style="position:absolute;top:100px;left:50%;transform:translateX(-50%);z-index:1;text-align:center">';
  html += `<div style="color:#94a3b8;font-size:10px">Bid: <span style="color:#fbbf24;font-weight:700">${tnState.currentBid}</span> by ${tnEsc(tnState.players[tnState.bidderIdx].name)}</div>`;
  if (tnState.trumpRevealed) {
    html += `<div class="tn-trump-revealed">${TN_SUIT_SYMBOLS[tnState.trumpSuit]} ${TN_SUIT_NAMES[tnState.trumpSuit]}</div>`;
  } else if (myIdx === tnState.bidderIdx && tnState.trumpSuit) {
    html += `<div style="color:#a78bfa;font-size:11px">Trump: ${TN_SUIT_SYMBOLS[tnState.trumpSuit]} (hidden)</div>`;
  } else {
    html += '<div style="color:#a78bfa;font-size:11px">Trump: ???</div>';
  }
  if (tnState.pairDeclared) {
    html += `<div style="color:#f59e0b;font-size:11px;font-weight:700">Pair by ${tnEsc(tnState.pairPlayerName)}!</div>`;
  }
  html += '</div>';

  // Other players (top, left, right) - folded cards
  for (let s = 1; s <= 3; s++) {
    const pi = seatMap[s];
    const p = tnState.players[pi];
    const isActive = pi === tnState.currentPlayerIdx;
    const cardCount = p.cards ? p.cards.length : 0;
    const isVertical = s === 1 || s === 3;
    const isBidder = tnState.bidderPlayerId === p.id;

    html += `<div class="tn-player-spot ${posClasses[s]}">`;
    html += `<div class="tn-player-name${isActive ? ' active' : ''}${isBidder ? ' bidder' : ''}${p.team === tnState.players[myIdx >= 0 ? myIdx : 0].team ? ' partner' : ''}">`;
    html += `${tnEsc(p.name)}(${tnState.teamHands[p.team]})`;
    html += '</div>';
    html += `<div class="tn-folded-cards${isVertical ? ' vertical' : ''}">`;
    for (let ci = 0; ci < cardCount; ci++) {
      html += '<div class="tn-folded-card"></div>';
    }
    html += '</div></div>';
  }

  // Center area - current trick cards in plus pattern
  html += '<div class="tn-center">';
  if (tnState.currentTrick && tnState.currentTrick.cards) {
    for (let ti = 0; ti < tnState.currentTrick.cards.length; ti++) {
      const tc = tnState.currentTrick.cards[ti];
      if (!tc.rank) continue;
      const tPlayerID = tnState.currentTrick.playerIds[ti];
      // Find which seat position this player is relative to me
      const tPlayerIdx = tnState.players.findIndex(p => p.id === tPlayerID);
      const seatPos = seatMap.indexOf(tPlayerIdx);
      const posName = positions[seatPos] || 'bottom';
      html += `<div class="tn-trick-card pos-${posName} ${tc.suit}">
        <span class="tn-card-rank">${tc.rank}</span>
        <span class="tn-card-suit">${TN_SUIT_SYMBOLS[tc.suit]}</span>
      </div>`;
    }
  }
  html += '</div>';

  // My cards (bottom) - visible
  if (myIdx >= 0) {
    const me = tnState.players[myIdx];
    const isMyTurn = myIdx === tnState.currentPlayerIdx;
    const isBidder = tnState.bidderPlayerId === me.id;
    html += `<div class="tn-player-spot tn-player-bottom">`;
    html += `<div class="tn-player-name${isMyTurn ? ' active' : ''}${isBidder ? ' bidder' : ''}">`;
    html += `${tnEsc(me.name)}(${tnState.teamHands[me.team]})`;
    html += '</div>';
    html += '<div class="tn-my-cards">';
    const leadSuit = tnState.currentTrick?.leadSuit;
    const hasSuit = leadSuit ? me.cards.some(c => c.suit === leadSuit) : false;
    for (const c of me.cards) {
      if (c.id.startsWith('hidden_')) continue;
      const canPlay = isMyTurn && (!leadSuit || c.suit === leadSuit || !hasSuit);
      const isTrump = tnState.trumpRevealed && c.suit === tnState.trumpSuit;
      const selected = tnSelectedCard === c.id;
      html += `<div class="tn-card ${c.suit}${isTrump ? ' trump-card' : ''}${!canPlay ? ' disabled' : ''}${selected ? ' selected' : ''}" onclick="tnSelectCardToPlay('${c.id}')">
        <span class="tn-card-rank">${c.rank}</span>
        <span class="tn-card-suit">${TN_SUIT_SYMBOLS[c.suit]}</span>
      </div>`;
    }
    html += '</div></div>';
  }

  html += '</div>';

  // Points overlay
  if (tnShowPoints) {
    html += renderTNPointsOverlay();
  }

  return html;
}

function renderTNPlayingActions(myIdx) {
  if (myIdx < 0) return '<div class="tn-waiting">Spectating...</div>';

  const isMyTurn = myIdx === tnState.currentPlayerIdx;
  if (!isMyTurn) {
    return `<div class="tn-waiting">${tnEsc(tnState.players[tnState.currentPlayerIdx].name)}'s turn...</div>`;
  }

  let html = '<div class="tn-action-area">';

  // Play card button
  html += `<button class="tn-play-btn" ${!tnSelectedCard ? 'disabled' : ''} onclick="tnPlayCard()">Play Card</button>`;

  // Reveal trump button (if not revealed and can't follow suit)
  if (!tnState.trumpRevealed && tnState.currentTrick?.leadSuit) {
    const me = tnState.players[myIdx];
    const hasSuit = me.cards.some(c => c.suit === tnState.currentTrick.leadSuit);
    if (!hasSuit) {
      html += `<button class="tn-reveal-btn" onclick="tnRevealTrump()">Reveal Trump</button>`;
    }
  }

  // Declare pair button
  if (tnState.trumpRevealed && !tnState.pairDeclared) {
    const me = tnState.players[myIdx];
    const hasK = me.cards.some(c => c.suit === tnState.trumpSuit && c.rank === 'K');
    const hasQ = me.cards.some(c => c.suit === tnState.trumpSuit && c.rank === 'Q');
    if (hasK && hasQ) {
      html += `<button class="tn-pair-btn" onclick="tnDeclarePair()">Declare Pair (K+Q)</button>`;
    }
  }

  html += '</div>';
  return html;
}

function renderTNRoundResult(myIdx) {
  let html = '<div class="tn-round-result">';
  if (tnState.phase === 'game_over') {
    html += `<h3>🏆 Game Over!</h3>`;
    html += `<p style="font-size:18px;color:#22c55e;font-weight:700">${tnEsc(tnState.winner)} wins!</p>`;
  } else {
    html += `<h3>Round ${tnState.round} Result</h3>`;
  }
  html += `<p>${tnEsc(tnState.roundResult)}</p>`;

  // Show team scores
  const t0 = `${tnEsc(tnState.players[0].name)} & ${tnEsc(tnState.players[2].name)}`;
  const t1 = `${tnEsc(tnState.players[1].name)} & ${tnEsc(tnState.players[3].name)}`;
  html += `<p>Team A (${t0}): <b>${tnState.teamPoints[0]} card pts</b> | Score: <b>${tnState.teamScore[0]}</b></p>`;
  html += `<p>Team B (${t1}): <b>${tnState.teamPoints[1]} card pts</b> | Score: <b>${tnState.teamScore[1]}</b></p>`;

  if (tnState.phase === 'round_over') {
    html += `<button class="tn-next-round-btn" onclick="tnNextRound()">Next Round</button>`;
  }
  if (tnState.phase === 'game_over') {
    html += `<button class="tn-next-round-btn" style="background:#3b82f6" onclick="send('return-to-lobby')">Return to Lobby</button>`;
  }
  html += '</div>';
  return html;
}

function renderTNPointsOverlay() {
  const t0 = `${tnEsc(tnState.players[0].name)} & ${tnEsc(tnState.players[2].name)}`;
  const t1 = `${tnEsc(tnState.players[1].name)} & ${tnEsc(tnState.players[3].name)}`;

  let adjustedTarget = tnState.targetPoints;
  if (tnState.pairDeclared) {
    if (tnState.pairDeclaredBy === tnState.biddingTeam) {
      adjustedTarget = Math.max(15, adjustedTarget - 4);
    } else {
      adjustedTarget = Math.min(28, adjustedTarget + 4);
    }
  }

  let html = `<div class="tn-points-overlay" onclick="tnTogglePoints()">
    <div class="tn-points-content" onclick="event.stopPropagation()">
      <h3>Points & Scoring</h3>
      <table class="tn-points-table">
        <tr><th></th><th>Team A</th><th>Team B</th></tr>
        <tr><td>Card Points</td><td>${tnState.teamPoints[0]}</td><td>${tnState.teamPoints[1]}</td></tr>
        <tr><td>Tricks Won</td><td>${tnState.teamHands[0]}</td><td>${tnState.teamHands[1]}</td></tr>
        <tr><td>Game Score</td><td>${tnState.teamScore[0]}</td><td>${tnState.teamScore[1]}</td></tr>
      </table>
      <div style="margin-top:12px;font-size:12px;color:#94a3b8">
        <div>Bid: <b style="color:#fbbf24">${tnState.currentBid}</b> by ${tnEsc(tnState.players[tnState.bidderIdx].name)}</div>
        <div>Bidding team needs: <b style="color:#22c55e">${adjustedTarget}</b> pts${tnState.pairDeclared ? ' (pair adjusted)' : ''}</div>
        <div>Team A: ${t0}</div>
        <div>Team B: ${t1}</div>
      </div>
      <div style="margin-top:12px;font-size:11px;color:#64748b">
        <div>Card values: J=3, 9=2, A=1, 10=1, K/Q/8/7=0</div>
        <div>Last trick bonus: +1 point</div>
        <div>Total possible: 29 points</div>
      </div>
      <button class="tn-points-close" onclick="tnTogglePoints()">Close</button>
    </div>
  </div>`;
  return html;
}

function renderTNCardHTML(card, selectable, isTrump) {
  return `<div class="tn-card ${card.suit}${isTrump ? ' trump-card' : ''}" ${selectable ? `onclick="tnSelectCardToPlay('${card.id}')"` : ''}>
    <span class="tn-card-rank">${card.rank}</span>
    <span class="tn-card-suit">${TN_SUIT_SYMBOLS[card.suit]}</span>
  </div>`;
}

// --- Actions ---
function tnBid(value) {
  send('tn-bid', { value });
}

function tnPass() {
  send('tn-pass');
}

function tnSelectTrump(suit) {
  send('tn-select-trump', { suit });
}

function tnSelectCardToPlay(cardId) {
  tnSelectedCard = tnSelectedCard === cardId ? null : cardId;
  renderTNGame();
}

function tnPlayCard() {
  if (!tnSelectedCard) return;
  send('tn-play-card', { cardId: tnSelectedCard });
  tnSelectedCard = null;
}

function tnRevealTrump() {
  send('tn-reveal-trump');
}

function tnDeclarePair() {
  send('tn-declare-pair');
}

function tnNextRound() {
  tnPrevTricksPlayed = -1;
  send('tn-next-round');
}

function tnTogglePoints() {
  tnShowPoints = !tnShowPoints;
  renderTNGame();
}

function switchTNTab(tab) {
  document.getElementById('tn-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('tn-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  document.querySelectorAll('#tn-active .tab').forEach(t => t.classList.remove('active'));
  document.getElementById('tntab-' + tab).classList.add('active');
  if (tab === 'chat') {
    const el = document.getElementById('tntab-chat');
    if (el) el.classList.remove('chat-unread');
  }
}

function tnEsc(s) { const d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML; }
