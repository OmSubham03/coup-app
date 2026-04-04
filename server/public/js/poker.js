// ========== POKER GAME ==========

const SUIT_SYMBOLS = { hearts: '♥', diamonds: '♦', clubs: '♣', spades: '♠' };
const SUIT_COLORS = { hearts: 'red', diamonds: 'red', clubs: 'black', spades: 'black' };
const RANK_DISPLAY = { 2:'2',3:'3',4:'4',5:'5',6:'6',7:'7',8:'8',9:'9',10:'10',11:'J',12:'Q',13:'K',14:'A' };
let prevCommunityCardCount = 0;

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
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = '';

  const phase = pokerState.phase.replace(/_/g, ' ');
  document.getElementById('poker-phase-display').textContent = isSpectating ? 'Spectating' : phase;
  document.getElementById('poker-hand-display').textContent = 'Hand #' + pokerState.handNumber;

  const exitBtn = document.getElementById('poker-exit-btn');
  if (isSpectating) {
    exitBtn.style.display = '';
    exitBtn.textContent = 'Stop Spectating';
  } else {
    exitBtn.style.display = (pokerState.phase === 'game_over' || pokerState.phase === 'showdown') ? 'none' : '';
    exitBtn.textContent = 'Exit Game';
  }

  renderPokerTable();
  renderPokerActions();
  renderPokerLog();
}

function renderPokerCard(card) {
  const color = SUIT_COLORS[card.suit];
  const rank = RANK_DISPLAY[card.rank] || card.rank;
  const suit = SUIT_SYMBOLS[card.suit] || '?';
  return '<div class="poker-card ' + color + '"><div>' + rank + '</div><div style="font-size:20px">' + suit + '</div></div>';
}

function renderPokerTable() {
  const ps = pokerState;
  let html = '<div class="poker-table">';

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
  for (let i = 0; i < ps.players.length; i++) {
    const p = ps.players[i];
    const isMe = p.id === playerId;
    const isCurrent = i === ps.currentPlayerIndex;
    const isDealer = i === ps.dealerIndex;
    let cls = 'poker-player';
    if (isMe) cls += ' mine';
    if (isCurrent && !p.folded) cls += ' current';
    if (p.folded) cls += ' folded';
    if (isDealer) cls += ' dealer';

    html += '<div class="' + cls + '">';
    html += '<div class="player-header"><div>';
    html += '<span class="player-name">' + esc(p.name) + '</span>';
    if (isMe) html += '<span class="badge" style="background:#22c55e">You</span>';
    if (p.allIn) html += ' <span style="color:#d4a017;font-size:12px;font-weight:700">ALL IN</span>';
    if (p.folded) html += ' <span style="color:#ef4444;font-size:12px">Folded</span>';
    if (isCurrent && !p.folded && !p.allIn && ps.phase !== 'showdown' && ps.phase !== 'game_over') html += ' <span style="color:#d4a017;font-size:12px">◄ Turn</span>';
    html += '</div><span class="poker-chips">💰 ' + p.chips + '</span></div>';

    // Hole cards
    html += '<div style="display:flex;gap:6px;align-items:center">';
    if (p.holeCards && p.holeCards.length > 0) {
      for (const c of p.holeCards) {
        html += renderPokerCard(c);
      }
    } else if (p.isActive && !p.folded) {
      html += '<div class="poker-card-back"></div><div class="poker-card-back"></div>';
    }

    if (p.currentBet > 0) {
      html += '<span class="poker-bet-info" style="margin-left:8px">Bet: ' + p.currentBet + '</span>';
    }
    html += '</div>';

    // Show hand result at showdown
    if (p.hand && (ps.phase === 'showdown' || ps.phase === 'game_over')) {
      html += '<div class="poker-hand-result">' + esc(p.hand.rankName) + '</div>';
    }

    html += '</div>';
  }

  document.getElementById('poker-table-area').innerHTML = html;
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
    area.innerHTML = '<div class="waiting" style="text-align:center"><div style="font-size:24px;margin-bottom:8px">👁️</div>Spectating' +
      (cp && ps.phase !== 'showdown' && ps.phase !== 'game_over' ? ' — ' + esc(cp.name) + '\'s turn' : '') + '</div>';
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
    html += '<div style="font-size:32px;margin-bottom:8px">🃏</div>';
    html += '<h2 style="color:#d4a017;margin-bottom:16px">Showdown!</h2>';
    if (ps.lastAction) html += '<p style="color:#e2e8f0;margin-bottom:16px">' + esc(ps.lastAction) + '</p>';
    html += '<button class="btn btn-green" onclick="send(\'poker-next-hand\')">Deal Next Hand</button>';
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

  if (!raiseAmount || raiseAmount < minRaise) raiseAmount = minRaise;
  if (raiseAmount > maxRaise) raiseAmount = maxRaise;

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
