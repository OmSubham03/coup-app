// ========== POKER GAME ==========

const SUIT_SYMBOLS = { hearts: '♥', diamonds: '♦', clubs: '♣', spades: '♠' };
const SUIT_COLORS = { hearts: 'red', diamonds: 'red', clubs: 'black', spades: 'black' };
const RANK_DISPLAY = { 2:'2',3:'3',4:'4',5:'5',6:'6',7:'7',8:'8',9:'9',10:'10',11:'J',12:'Q',13:'K',14:'A' };
let prevCommunityCardCount = 0;
let pokerDealHandKey = '';
let pokerDealStartedAt = 0;
let pokerDealDuration = 0;
let pokerDealTimer = null;

function switchPokerTab(tab) {
  document.getElementById('ptab-game').className = 'tab' + (tab === 'game' ? ' active' : '');
  document.getElementById('ptab-log').className = 'tab' + (tab === 'log' ? ' active' : '');
  document.getElementById('ptab-chat').className = 'tab' + (tab === 'chat' ? ' active' : '');
  document.getElementById('poker-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('poker-log-tab').style.display = tab === 'log' ? '' : 'none';
  document.getElementById('poker-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  if (tab === 'chat') document.getElementById('ptab-chat').classList.remove('chat-unread');
}

function renderPokerGame() {
  if (!pokerState) return;
  const handKey = [pokerState.id || (typeof roomCode === 'string' ? roomCode : ''), pokerState.handNumber].join(':');
  if (handKey !== pokerDealHandKey) {
    pokerDealHandKey = handKey;
    prevCommunityCardCount = 0;
    clearTimeout(pokerDealTimer);
    const reducedMotion = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
    pokerDealStartedAt = Date.now();
    pokerDealDuration = pokerState.phase === 'preflop' && !reducedMotion
      ? pokerState.players.filter(player => player.isActive && !player.folded).length * 160 + 450 : 0;
    if (pokerDealDuration) pokerDealTimer = setTimeout(() => {
      if (pokerDealHandKey !== handKey) return;
      pokerDealDuration = 0;
      if (pokerState) renderPokerGame();
    }, pokerDealDuration);
  }
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = '';

  const phase = pokerState.phase.replace(/_/g, ' ');
  document.getElementById('poker-phase-display').textContent = isSpectating ? 'Spectating' : phase;
  document.getElementById('poker-hand-display').textContent = 'Hand #' + pokerState.handNumber;

  renderPokerTable();
  renderPokerActions();
  renderPokerLog();
  animatePokerDeal();
}

function animatePokerDeal() {
  const elapsed = Date.now() - pokerDealStartedAt;
  if (!pokerDealDuration || elapsed >= pokerDealDuration || pokerState.phase !== 'preflop') return;
  const board = document.querySelector('.poker-board');
  if (!board) return;
  const bounds = board.getBoundingClientRect();
  const hands = [...board.querySelectorAll('.poker-seat-cards')].filter(hand => hand.children.length > 0);
  hands.forEach((hand, seatIndex) => [...hand.children].forEach((card, cardIndex) => {
    const position = card.getBoundingClientRect();
    card.style.setProperty('--deal-x', bounds.left + bounds.width / 2 - position.left - position.width / 2 + 'px');
    card.style.setProperty('--deal-y', bounds.top + bounds.height / 2 - position.top - position.height / 2 + 'px');
    card.style.setProperty('--deal-delay', ((cardIndex * hands.length + seatIndex) * 80 - elapsed) + 'ms');
    card.classList.add('poker-card-dealing');
  }));
  document.querySelectorAll('#poker-action-area button, #poker-action-area input').forEach(control => control.disabled = true);
  document.getElementById('poker-phase-display').textContent = 'Dealing';
}

function renderPokerCard(card) {
  const color = SUIT_COLORS[card.suit];
  const rank = RANK_DISPLAY[card.rank] || card.rank;
  const suit = SUIT_SYMBOLS[card.suit] || '?';
  return `<div class="poker-card ${color}" role="img" aria-label="${rank} of ${card.suit}"><b>${rank}</b><span>${suit}</span><b aria-hidden="true">${rank}</b></div>`;
}

const pokerStatIcons = {
  blinds: '<path d="M4 9a8 8 0 0 1 13-4l2 2m0-4v4h-4M20 15a8 8 0 0 1-13 4l-2-2m0 4v-4h4"/><circle cx="12" cy="12" r="3"/>',
  bank: '<path d="m3 8 9-5 9 5H3Zm1 12h16M6 17v-6m6 6v-6m6 6v-6"/>',
  chart: '<path d="M4 4v16h17M8 16v-4m5 4V7m5 9v-6"/>',
  chip: '<circle cx="12" cy="12" r="10" fill="#534968" stroke="#ffd160" stroke-width="4" stroke-dasharray="7 5"/><path d="m7 10 2-3h6l2 3-5 7z" fill="#fff9dc" stroke="none"/>',
  coins: '<g fill="#ffd45c" stroke="#856321" stroke-width="1"><path d="M3 13v4c0 5 18 5 18 0v-4z"/><path d="M3 9v4c0 5 18 5 18 0V9z"/><ellipse cx="12" cy="9" rx="9" ry="4"/></g>'
};

function pokerStatIcon(name) {
  return `<svg class="poker-stat-icon" aria-hidden="true" focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${pokerStatIcons[name]}</svg>`;
}

function renderPokerTable() {
  const ps = pokerState;
  let html = '<section class="poker-felt" aria-label="Community board"><div class="poker-summary">';

  // Pot display
  let totalPot = 0;
  if (ps.pots) {
    for (const pot of ps.pots) totalPot += pot.amount;
  }
  for (const p of ps.players) totalPot += p.currentBet || 0;
  html += `<div><span>${pokerStatIcon('blinds')}Blinds</span><strong>${ps.smallBlind}${ps.bigBlind ? ' / ' + ps.bigBlind : ''}</strong></div>
    <div><span>${pokerStatIcon('bank')}Pot</span><strong>${totalPot}${pokerStatIcon('chip')}</strong></div>
    <div><span>${pokerStatIcon('chart')}Table bet</span><strong>${ps.currentBet}${pokerStatIcon('coins')}</strong></div></div>`;

  // Community cards
  const cc = ps.communityCards || [];
  const newCardCount = cc.length;
  const revealFrom = newCardCount > prevCommunityCardCount ? prevCommunityCardCount : newCardCount;
  prevCommunityCardCount = newCardCount;
  html += '<div class="poker-community">';
  for (let i = 0; i < 5; i++) {
    if (i < cc.length) {
      if (i >= revealFrom) {
        html += '<div class="poker-card-flip-wrapper card-revealing">';
        html += '<div class="poker-card-back"></div>';
        html += renderPokerCard(cc[i]);
        html += '</div>';
      } else {
        html += renderPokerCard(cc[i]);
      }
    } else {
      html += '<div class="poker-card-placeholder" aria-label="Undealt card"></div>';
    }
  }
  html += '</div>';

  // Last action
  if (ps.lastAction) {
    html += '<div class="poker-last-action">' + esc(ps.lastAction) + '</div>';
  }

  html += '</section>';
  let rail = '';
  let personal = '';
  for (let i = 0; i < ps.players.length; i++) {
    const p = ps.players[i];
    const isMe = p.id === playerId;
    const isCurrent = i === ps.currentPlayerIndex;
    const isDealer = i === ps.dealerIndex;
    let cls = 'poker-seat';
    if (isMe) cls += ' mine';
    if (isCurrent && !p.folded) cls += ' current';
    if (p.folded) cls += ' folded';
    if (isDealer) cls += ' dealer';

    const initials = p.name.trim().split(/\s+/).map(part => part[0]).slice(0, 2).join('').toUpperCase();
    const status = p.folded ? 'Folded' : p.allIn ? 'All in' : !p.isActive ? 'Sitting out' :
      isCurrent && ps.phase !== 'showdown' && ps.phase !== 'game_over' ? 'Turn' : '';
    let cards = '';
    if (p.holeCards && p.holeCards.length > 0) {
      for (const c of p.holeCards) {
        cards += renderPokerCard(c);
      }
    } else if (p.isActive && !p.folded) {
      cards = '<div class="poker-card-back"></div><div class="poker-card-back"></div>';
    }

    const result = p.hand && ['showdown','game_over'].includes(ps.phase) ? esc(p.hand.rankName) : '';
    const details = [status, isDealer ? 'Dealer' : ''].filter(Boolean).join(' · ');
    const betChip = p.currentBet > 0
      ? `<span class="poker-wager" role="img" aria-label="Bet: ${p.currentBet} chips"><b>${p.currentBet}</b></span>` : '';
    rail += `<div class="${cls}"><span class="poker-avatar tone-${i % 4}">${esc(initials)}</span>
      <strong class="poker-seat-name">${esc(p.name)}${isMe ? ' (You)' : ''}</strong><span>${p.chips} chips</span>
      <small>${details}</small>${!isMe && p.holeCards?.length ? '<div class="poker-seat-cards">' + cards + '</div>' : ''}
      ${!isMe && result ? '<small>' + result + '</small>' : ''}<div class="poker-wager-slot">${betChip}</div></div>`;
    if (isMe) personal = `<section class="poker-personal"><div><p class="poker-eyebrow">YOUR HAND</p>
      <strong>${status === 'Turn' ? 'Your turn' : status || 'At the table'}</strong>
      <p>${p.chips} chips${result ? ' · ' + result : ''}</p></div><div class="poker-seat-cards">${cards}</div></section>`;
  }

  document.getElementById('poker-table-area').innerHTML = `<div class="poker-board"><section class="poker-rail" aria-label="Players at the table">${rail}</section>${html}${personal}</div>`;
}

let raiseAmount = 0;

function renderPokerActions() {
  const area = document.getElementById('poker-action-area');
  const ps = pokerState;
  const me = ps.players.find(p => p.id === playerId);
  const isCurrent = ps.players[ps.currentPlayerIndex]?.id === playerId;

  // Spectator
  if (isSpectating) {
    const cp = ps.players[ps.currentPlayerIndex];
    const alreadyPending = ps.pendingJoins && ps.pendingJoins.some(p => p.id === playerId);
    const alreadyPlayer = ps.players.some(p => p.id === playerId);
    let html = '<div class="waiting" style="text-align:center"><div style="font-size:24px;margin-bottom:8px">👁️</div>Spectating' +
      (cp && ps.phase !== 'showdown' && ps.phase !== 'game_over' ? ' — ' + esc(cp.name) + '\'s turn' : '') + '</div>';
    if (!alreadyPending && !alreadyPlayer && ps.phase !== 'game_over') {
      html += '<button class="btn btn-green" style="margin-top:12px;width:100%" onclick="send(\'poker-join-game\')">Join Next Round</button>';
    } else if (alreadyPending) {
      html += '<p style="color:#22c55e;margin-top:12px;text-align:center">✓ Joining next round...</p>';
    }
    area.innerHTML = html;
    return;
  }

  // Game over — show scoreboard
  if (ps.phase === 'game_over') {
    let html = '<div class="scoreboard"><h2>🏆 Final Scoreboard</h2>';
    if (ps.scoreboard) {
      for (let i = 0; i < ps.scoreboard.length; i++) {
        const s = ps.scoreboard[i];
        const gainClass = s.netGain > 0 ? 'score-positive' : (s.netGain < 0 ? 'score-negative' : 'score-zero');
        const prefix = s.netGain > 0 ? '+' : '';
        html += '<div class="score-row"><div><span class="score-name">' + (i === 0 ? '👑 ' : '') + esc(s.name) + '</span>';
        html += '<div class="score-detail">Final: ' + s.finalChips + ' chips</div></div>';
        html += '<span class="score-gain ' + gainClass + '">' + prefix + s.netGain + '</span></div>';
      }
    }
    html += '<button class="btn btn-green" style="margin-top:16px" onclick="send(\'return-to-lobby\')">Return to Lobby</button>';
    html += '</div>';
    area.innerHTML = html;
    return;
  }

  // Showdown
  if (ps.phase === 'showdown') {
    let html = '<div class="poker-result">';

    // Check if this player needs to decide on rebuy
    if (me && me.needsRebuy) {
      html += '<div style="font-size:32px;margin-bottom:8px">💸</div>';
      html += '<h2 style="color:#d4a017;margin-bottom:12px">Out of Chips!</h2>';
      html += '<p>Buy back in for ' + ps.buyIn + ' chips?</p>';
      html += '<button class="btn btn-green" style="margin-bottom:8px" onclick="send(\'poker-rebuy\')">Buy In (' + ps.buyIn + ' chips)</button>';
      html += '<button class="btn btn-red" onclick="send(\'poker-skip-rebuy\')">Sit Out</button>';
      html += '</div>';
      area.innerHTML = html;
      return;
    }

    // Check if waiting for other players' rebuy decisions
    if (ps.pendingRebuys) {
      const waitingFor = ps.players.filter(p => p.needsRebuy).map(p => p.name);
      html += '<div style="font-size:32px;margin-bottom:8px">⏳</div>';
      html += '<h2 style="color:#d4a017;margin-bottom:12px">Waiting for Rebuy</h2>';
      html += '<p>' + waitingFor.map(n => esc(n)).join(', ') + ' deciding...</p>';
      html += '</div>';
      area.innerHTML = html;
      return;
    }

    html += '<div style="font-size:32px;margin-bottom:8px">🃏</div>';
    html += '<h2 style="color:#d4a017;margin-bottom:16px">Showdown!</h2>';
    if (ps.lastAction) html += '<p>' + esc(ps.lastAction) + '</p>';
    html += '<div style="display:flex;gap:8px;justify-content:center;flex-wrap:wrap">';
    html += '<button class="btn btn-green" onclick="send(\'poker-next-hand\')">Deal Next Hand</button>';
    html += '<button class="btn btn-red" onclick="if(confirm(\'End game and show final scores?\'))send(\'poker-end-game\')">End Game</button>';
    html += '</div>';
    html += '</div>';
    area.innerHTML = html;
    return;
  }

  // Not my turn or I'm folded/all-in
  if (!me || me.folded || !me.isActive) {
    area.innerHTML = '<div class="waiting">You are out of this hand</div>';
    return;
  }

  if (me.allIn) {
    area.innerHTML = '<div class="waiting">You are all in — waiting for showdown</div>';
    return;
  }

  if (!isCurrent) {
    const cp = ps.players[ps.currentPlayerIndex];
    area.innerHTML = '<div class="waiting">Waiting for ' + esc(cp?.name || '?') + '</div>';
    return;
  }

  // My turn — show actions
  const canCheck = me.currentBet >= ps.currentBet;
  const callAmount = Math.min(ps.currentBet - (me.currentBet || 0), me.chips);
  const minRaise = ps.currentBet + ps.minRaise;
  const maxRaise = me.chips + (me.currentBet || 0);

  if (!raiseAmount || raiseAmount < minRaise || raiseAmount > maxRaise) raiseAmount = minRaise;

  let html = '<div class="poker-action-area">';

  // Fold
  html += '<button class="poker-action-btn poker-btn-fold" onclick="pokerAct(\'fold\')">Fold</button>';

  // Check or Call
  if (canCheck) {
    html += '<button class="poker-action-btn poker-btn-check" onclick="pokerAct(\'check\')">Check</button>';
  } else {
    if (callAmount >= me.chips) {
      html += '<button class="poker-action-btn poker-btn-allin" onclick="pokerAct(\'call\')">Call ' + callAmount + ' (All In)</button>';
    } else {
      html += '<button class="poker-action-btn poker-btn-call" onclick="pokerAct(\'call\')">Call ' + callAmount + '</button>';
    }
  }

  // Raise
  if (maxRaise > minRaise && me.chips > callAmount) {
    html += `<div class="poker-raise"><div class="poker-amount">
      <button class="poker-step" aria-label="Decrease raise" onclick="adjustRaise(-${ps.smallBlind})">−</button>
      <output id="raise-display" for="raise-slider">${raiseAmount}</output>
      <button class="poker-step" aria-label="Increase raise" onclick="adjustRaise(${ps.smallBlind})">+</button></div>
      <button class="poker-action-btn poker-btn-raise" id="raise-btn" onclick="submitPokerRaise()">${raiseAmount === maxRaise ? 'All in' : 'Raise to ' + raiseAmount}</button>
      <input type="range" class="raise-slider" id="raise-slider" aria-label="Raise total" min="${minRaise}" max="${maxRaise}" value="${raiseAmount}" oninput="updateRaise(this.value)"></div>`;
  } else if (me.chips > callAmount) {
    html += '<button class="poker-action-btn poker-btn-allin" onclick="pokerAct(\'allin\')">All in (' + me.chips + ')</button>';
  }

  html += '</div>';
  area.innerHTML = html;
}

function updateRaise(val) {
  const slider = document.getElementById('raise-slider');
  slider.value = Math.max(Number(slider.min), Math.min(Number(val), Number(slider.max)));
  raiseAmount = Number(slider.value);
  document.getElementById('raise-display').textContent = raiseAmount;
  document.getElementById('raise-btn').textContent = raiseAmount === Number(slider.max) ? 'All in' : 'Raise to ' + raiseAmount;
}

function adjustRaise(increment) {
  updateRaise(raiseAmount + increment);
}

function submitPokerRaise() {
  if (raiseAmount === Number(document.getElementById('raise-slider').max)) pokerAct('allin');
  else pokerAct('raise', raiseAmount);
}

function pokerAct(action, amount) {
  send('poker-action', { action, amount: amount || 0 });
}

function renderPokerLog() {
  const log = pokerState.log || [];
  const grouped = {};
  for (const e of log) {
    if (!grouped[e.hand]) grouped[e.hand] = [];
    grouped[e.hand].push(e);
  }
  const hands = Object.keys(grouped).map(Number).sort((a,b) => b - a);
  let html = '';
  for (const h of hands) {
    html += '<div class="log-turn">Hand #' + h + '</div>';
    for (const e of grouped[h].slice().reverse()) {
      const time = new Date(e.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += '<div class="log-entry">' + esc(e.message) + ' <time>' + time + '</time></div>';
    }
  }
  document.getElementById('poker-log-panel').innerHTML = html;
}

function togglePokerRules() { document.getElementById('poker-rules-overlay').classList.toggle('active'); }
