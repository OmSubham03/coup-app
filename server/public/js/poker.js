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
  return '<div class="poker-card ' + color + '"><div>' + rank + '</div><div style="font-size:20px">' + suit + '</div></div>';
}

function renderPokerTable() {
  const ps = pokerState;
  let html = '<div class="poker-board" data-player-count="' + ps.players.length + '"><div class="poker-felt"></div><div class="poker-board-center">';

  // Pot display
  let totalPot = 0;
  if (ps.pots) {
    for (const pot of ps.pots) totalPot += pot.amount;
  }
  for (const p of ps.players) totalPot += p.currentBet || 0;
  if (totalPot > 0) {
    html += '<div class="poker-pot">Pot: ' + totalPot + '</div>';
  }

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
      html += '<div class="poker-card-placeholder"></div>';
    }
  }
  html += '</div>';

  // Last action
  if (ps.lastAction) {
    html += '<div class="poker-last-action">' + esc(ps.lastAction) + '</div>';
  }

  html += '</div>';

  // Players
  const myIndex = ps.players.findIndex(player => player.id === playerId);
  const anchor = myIndex >= 0 ? myIndex : 0;
  const seats = pokerSeatPositions(ps.players.length);
  for (let seatIndex = 0; seatIndex < ps.players.length; seatIndex++) {
    const i = (anchor + seatIndex) % ps.players.length;
    const p = ps.players[i];
    const isMe = p.id === playerId;
    const isCurrent = i === ps.currentPlayerIndex;
    const isDealer = i === ps.dealerIndex;
    let cls = 'poker-seat poker-seat-' + seats[seatIndex].side;
    if (isMe) cls += ' mine';
    if (isCurrent && !p.folded) cls += ' current';
    if (p.folded) cls += ' folded';
    if (isDealer) cls += ' dealer';

    const initials = p.name.trim().split(/\s+/).map(part => part[0]).slice(0, 2).join('').toUpperCase();
    const status = p.folded ? 'Folded' : p.allIn ? 'All in' : !p.isActive ? 'Sitting out' :
      isCurrent && ps.phase !== 'showdown' && ps.phase !== 'game_over' ? 'Turn' : '';
    html += '<div class="' + cls + '" style="--seat-y:' + seats[seatIndex].percent + '%">';
    html += '<div class="poker-seat-avatar poker-avatar-' + (seatIndex % 4) + '">' + esc(initials) + '</div>';
    html += '<div class="poker-seat-name" title="' + esc(p.name) + '">' + esc(p.name) + (isMe ? ' (You)' : '') + '</div>';
    html += '<div class="poker-seat-chips">' + p.chips + ' chips</div>';
    html += '<div class="poker-seat-status">' + status + (isDealer ? (status ? ' · ' : '') + 'Dealer' : '') + '</div>';

    // Hole cards
    html += '<div class="poker-seat-cards">';
    if (p.holeCards && p.holeCards.length > 0) {
      for (const c of p.holeCards) {
        html += renderPokerCard(c);
      }
    } else if (p.isActive && !p.folded) {
      html += '<div class="poker-card-back"></div><div class="poker-card-back"></div>';
    }

    html += '</div>';
    html += '<div class="poker-seat-bet">' + (p.currentBet > 0 ? 'Bet: ' + p.currentBet : '') + '</div>';

    // Show hand result at showdown
    if (p.hand && (ps.phase === 'showdown' || ps.phase === 'game_over')) {
      html += '<div class="poker-hand-result">' + esc(p.hand.rankName) + '</div>';
    }

    html += '</div>';
  }

  document.getElementById('poker-table-area').innerHTML = html + '</div>';
}

function pokerSeatPositions(count) {
  return Array.from({ length: count }, (_, index) => {
    if (index < 2) return { side: index === 0 ? 'bottom' : 'top', percent: 50 };
    const side = index % 2 === 0 ? 'left' : 'right';
    const sideCount = side === 'left' ? Math.ceil((count - 2) / 2) : Math.floor((count - 2) / 2);
    const slot = Math.floor((index - 2) / 2);
    return { side, percent: sideCount === 1 ? 50 : 26 + slot * 48 / (sideCount - 1) };
  });
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
        html += '<div style="color:#94a3b8;font-size:12px">Final: ' + s.finalChips + ' chips</div></div>';
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
    let html = '<div style="text-align:center;padding:20px">';

    // Check if this player needs to decide on rebuy
    if (me && me.needsRebuy) {
      html += '<div style="font-size:32px;margin-bottom:8px">💸</div>';
      html += '<h2 style="color:#d4a017;margin-bottom:12px">Out of Chips!</h2>';
      html += '<p style="color:#e2e8f0;margin-bottom:16px">Buy back in for ' + ps.buyIn + ' chips?</p>';
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
      html += '<p style="color:#94a3b8;margin-bottom:16px">' + waitingFor.map(n => esc(n)).join(', ') + ' deciding...</p>';
      html += '</div>';
      area.innerHTML = html;
      return;
    }

    html += '<div style="font-size:32px;margin-bottom:8px">🃏</div>';
    html += '<h2 style="color:#d4a017;margin-bottom:16px">Showdown!</h2>';
    if (ps.lastAction) html += '<p style="color:#e2e8f0;margin-bottom:16px">' + esc(ps.lastAction) + '</p>';
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
  if (me.chips > callAmount && maxRaise > ps.currentBet) {
    html += '<div style="margin-top:8px;background:#0a2e0a;border:1px solid #1a5a1a;border-radius:10px;padding:12px">';
    html += '<div class="raise-amount" id="raise-display">' + raiseAmount + '</div>';
    html += '<input type="range" class="raise-slider" id="raise-slider" min="' + minRaise + '" max="' + maxRaise + '" value="' + raiseAmount + '" step="' + ps.smallBlind + '" oninput="updateRaise(this.value)">';
    html += '<div style="display:flex;gap:4px;margin-top:8px;flex-wrap:wrap">';
    const sb = ps.smallBlind;
    const increments = [{label: '+' + sb, val: sb}, {label: '+' + (sb*5), val: sb*5}, {label: '+' + (sb*10), val: sb*10}];
    for (const inc of increments) {
      html += '<button style="flex:1;padding:6px;border:1px solid #1a5a1a;background:#0a1e0a;color:#d4a017;border-radius:6px;cursor:pointer;font-weight:600;font-size:12px" onclick="adjustRaise(' + inc.val + ',' + minRaise + ',' + maxRaise + ')">' + inc.label + '</button>';
    }
    html += '</div>';
    if (raiseAmount >= maxRaise) {
      html += '<button class="poker-action-btn poker-btn-allin" style="margin-top:8px" id="raise-btn" onclick="pokerAct(\'allin\')">All In (' + me.chips + ')</button>';
    } else {
      html += '<button class="poker-action-btn poker-btn-raise" style="margin-top:8px" id="raise-btn" onclick="pokerAct(\'raise\', raiseAmount)">Raise to ' + raiseAmount + '</button>';
    }
    html += '</div>';
  }

  // All-in button always available
  if (me.chips > 0 && !(me.chips > callAmount && maxRaise > ps.currentBet)) {
    html += '<button class="poker-action-btn poker-btn-allin" style="margin-top:8px" onclick="pokerAct(\'allin\')">All In (' + me.chips + ')</button>';
  }

  html += '</div>';
  area.innerHTML = html;
}

function updateRaise(val) {
  raiseAmount = parseInt(val);
  if (pokerState && pokerState.smallBlind > 0) {
    const sb = pokerState.smallBlind;
    raiseAmount = Math.round(raiseAmount / sb) * sb;
    if (raiseAmount < sb) raiseAmount = sb;
  }
  const display = document.getElementById('raise-display');
  if (display) display.textContent = raiseAmount;
  const btn = document.getElementById('raise-btn');
  if (btn) btn.textContent = 'Raise to ' + raiseAmount;
}

function adjustRaise(increment, minR, maxR) {
  let val = raiseAmount + increment;
  if (val < minR) val = minR;
  if (val > maxR) val = maxR;
  updateRaise(val);
  const slider = document.getElementById('raise-slider');
  if (slider) slider.value = val;
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
    html += '<div class="log-turn" style="background:#0a2e0a;color:#22c55e">Hand #' + h + '</div>';
    for (const e of grouped[h].slice().reverse()) {
      const time = new Date(e.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += '<div class="log-entry">' + esc(e.message) + ' <span style="color:#475569;font-size:10px">' + time + '</span></div>';
    }
  }
  document.getElementById('poker-log-panel').innerHTML = html;
}

function togglePokerRules() { document.getElementById('poker-rules-overlay').classList.toggle('active'); }
