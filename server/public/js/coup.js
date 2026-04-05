// ========== COUP GAME ==========

const CHAR_IMG = {
  Duke: '/textures/duke.jpg',
  Assassin: '/textures/assassin.jpg',
  Captain: '/textures/captain.jpg',
  Ambassador: '/textures/ambassador.jpg',
  Contessa: '/textures/contessa.jpg',
  Inquisitor: '/textures/inquisitor.png',
};

function renderGame() {
  if (!gameState) return;
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('game-active').style.display = '';
  document.getElementById('poker-active').style.display = 'none';

  const exitBtn = document.getElementById('exit-game-btn');
  if (isSpectating) {
    exitBtn.style.display = '';
    exitBtn.textContent = 'Exit';
  } else if (gameState.phase === 'game_over') {
    exitBtn.style.display = 'none';
  } else {
    exitBtn.style.display = '';
    exitBtn.textContent = 'Exit';
  }

  const phase = gameState.phase.replace(/_/g, ' ');
  document.getElementById('phase-display').textContent = (isSpectating ? '\ud83d\udc41 Spectating — ' : '') + phase;
  document.getElementById('turn-display').textContent = 'Turn ' + gameState.turn;

  renderPlayers();
  renderActions();
  renderLog();
}

function renderPlayers() {
  const cp = gameState.players[gameState.currentPlayerIndex];
  let html = '';
  for (const p of gameState.players) {
    const isMe = p.id === playerId;
    const isCurrent = p.id === cp.id;
    const cls = 'player-card' + (isMe ? ' mine' : '') + (isCurrent ? ' current' : '') + (!p.isAlive ? ' dead' : '');
    html += '<div class="' + cls + '">';
    html += '<div class="player-header"><div>';
    if (isCurrent) html += '👑 ';
    html += '<span class="player-name">' + esc(p.name) + '</span>';
    if (isMe) html += '<span class="badge">You</span>';
    if (isCurrent) html += ' <span style="color:#3b82f6;font-size:12px">Current Turn</span>';
    html += '</div><span class="coins">💰 ' + p.coins + '</span></div>';
    html += '<div>';
    for (const c of p.cards) {
      if (c.revealed) {
        html += '<span class="card-box card-revealed" style="background-image:url(' + CHAR_IMG[c.character] + ')"><span class="card-label">' + c.character + '</span></span>';
      } else if (isMe) {
        html += '<span class="card-box" style="background-image:url(' + CHAR_IMG[c.character] + ')"><span class="card-label">' + c.character + '</span></span>';
      } else {
        html += '<span class="card-box card-hidden"></span>';
      }
    }
    html += '</div></div>';
  }
  document.getElementById('players-board').innerHTML = html;
}

const ACTION_INFO = {
  income: { label: 'Income', desc: '+1 coin (safe)', color: 'btn-green' },
  foreign_aid: { label: 'Foreign Aid', desc: '+2 coins (blockable)', color: 'btn-blue' },
  tax: { label: 'Tax (Duke)', desc: '+3 coins', color: 'btn-purple' },
  exchange: { label: 'Exchange (Ambassador)', desc: 'Swap cards', color: 'btn-indigo' },
  inquire: { label: 'Inquire (Inquisitor)', desc: 'Draw 1, return 1', color: 'btn-emerald' },
  steal: { label: 'Steal (Captain)', desc: 'Take 2 coins', color: 'btn-pink' },
  assassinate: { label: 'Assassinate (Assassin)', desc: 'Pay 3 coins', color: 'btn-red' },
  coup: { label: 'Coup', desc: 'Pay 7 coins', color: 'btn-purple' },
  interrogate: { label: 'Interrogate (Inquisitor)', desc: 'Reveal & replace', color: 'btn-emerald' },
};

const ACTION_REQS = {
  income: {},
  foreign_aid: { canBeBlocked: true, blockingChars: ['Duke'] },
  coup: { cost: 7, needsTarget: true },
  tax: { character: 'Duke' },
  assassinate: { character: 'Assassin', cost: 3, needsTarget: true, canBeBlocked: true, blockingChars: ['Contessa'] },
  steal: { character: 'Captain', needsTarget: true, canBeBlocked: true, blockingChars: ['Captain', 'Ambassador'] },
  exchange: { character: 'Ambassador' },
  interrogate: { character: 'Inquisitor', needsTarget: true },
  inquire: { character: 'Inquisitor' },
};

function getStealBlockers() {
  return gameState.variant === 'inquisitor' ? ['Captain', 'Inquisitor'] : ['Captain', 'Ambassador'];
}

function renderActions() {
  const area = document.getElementById('action-area');
  const me = gameState.players.find(p => p.id === playerId);
  const cp = gameState.players[gameState.currentPlayerIndex];
  const isMyTurn = cp.id === playerId;

  // Victory / Game Over
  if (gameState.phase === 'game_over') {
    const winner = gameState.players.find(p => p.id === gameState.winner);
    const isWinner = winner?.id === playerId;
    area.innerHTML = '<div class="victory"><div style="font-size:64px">' + (isWinner ? '🎉' : '💀') + '</div><h2>' +
      (isWinner ? 'Victory!' : 'Game Over') + '</h2><p style="font-size:24px;margin-bottom:24px">' +
      esc(winner?.name || '?') + ' wins!</p>' +
      (isSpectating ? '<button class="btn btn-outline" onclick="stopSpectating()">Back to Lobby</button>' : '<button class="btn btn-purple" onclick="send(\'return-to-lobby\')">' + 'Return to Lobby</button>') + '</div>';
    return;
  }

  // Spectator view
  if (isSpectating) {
    const cp = gameState.players[gameState.currentPlayerIndex];
    area.innerHTML = '<div class="spectate-banner">\ud83d\udc41 Spectating — ' + esc(cp.name) + '\'s turn</div>';
    return;
  }

  // Eliminated - spectating
  if (!me || !me.isAlive) {
    area.innerHTML = '<div class="waiting">\ud83d\udc80 You are eliminated \u2014 spectating</div>';
    return;
  }

  // Action phase
  if (gameState.phase === 'action' && isMyTurn) {
    const mustCoup = me.coins >= 10;
    const actions = gameState.variant === 'inquisitor'
      ? ['income','foreign_aid','tax','inquire','steal','interrogate','assassinate','coup']
      : ['income','foreign_aid','tax','exchange','steal','assassinate','coup'];

    let html = '<div class="panel panel-action"><h2>⚔️ Choose Your Action</h2>';
    if (mustCoup) {
      html += '<p style="color:#f59e0b;margin:8px 0">You have 10+ coins — you must Coup!</p>';
    }
    html += '<div class="action-grid">';
    for (const a of actions) {
      if (mustCoup && a !== 'coup') continue;
      const info = ACTION_INFO[a];
      const req = ACTION_REQS[a];
      const disabled = (req.cost && me.coins < req.cost) ? 'disabled' : '';
      html += '<button class="btn ' + info.color + '" ' + disabled + ' onclick="actionClick(\'' + a + '\')"><div class="action-label">' + info.label + '</div><div class="action-desc">' + info.desc + '</div></button>';
    }
    html += '</div></div>';
    area.innerHTML = html;
    return;
  }

  // Block window
  if (gameState.phase === 'block_window' && gameState.pendingAction) {
    const hasPassed = gameState.passedPlayers?.includes(playerId);
    const actor = gameState.players.find(p => p.id === gameState.pendingAction.actorId);
    const target = gameState.pendingAction.targetId ? gameState.players.find(p => p.id === gameState.pendingAction.targetId) : null;
    const canBlock = (gameState.pendingAction.type === 'foreign_aid' && gameState.pendingAction.actorId !== playerId) || (target?.id === playerId);

    if (!canBlock || hasPassed) {
      area.innerHTML = '<div class="waiting">Waiting for others to respond to ' + esc(actor?.name) + '\'s action...</div>';
      return;
    }

    const isTargeted = target?.id === playerId;
    const blockers = gameState.pendingAction.type === 'steal' ? getStealBlockers() : (ACTION_REQS[gameState.pendingAction.type]?.blockingChars || []);
    let html = '<div class="panel ' + (isTargeted ? 'panel-targeted' : 'panel-block') + '">';
    html += '<h2>' + (isTargeted ? '⚠️ You are Targeted!' : '🛡 Block Opportunity!') + '</h2>';
    html += '<p>' + esc(actor?.name) + ' is trying to ' + gameState.pendingAction.type.replace('_', ' ') + (isTargeted ? ' YOU!' : '') + '</p>';
    for (const ch of blockers) {
      html += '<button class="btn char-' + ch + '" style="margin-top:8px;display:flex;align-items:center;gap:8px;justify-content:center" onclick="send(\'block\',{type:\'block_' + gameState.pendingAction.type + '\',blockerId:\'' + playerId + '\',claimedCharacter:\'' + ch + '\',targetActionId:\'' + gameState.pendingAction.actorId + '\'})"><img src="' + CHAR_IMG[ch] + '" style="width:28px;height:28px;border-radius:50%;object-fit:cover;border:1px solid rgba(255,255,255,.5)">Block with ' + ch + '</button>';
    }
    html += '<button class="btn btn-outline" style="margin-top:8px" onclick="send(\'pass-block\')">Allow Action</button></div>';
    area.innerHTML = html;
    return;
  }

  // Challenge window
  if (gameState.phase === 'challenge_window') {
    const hasPassed = gameState.passedPlayers?.includes(playerId);
    let targetPid, claimedChar, desc;
    if (gameState.pendingBlock) {
      targetPid = gameState.pendingBlock.blockerId;
      claimedChar = gameState.pendingBlock.claimedCharacter;
      const blocker = gameState.players.find(p => p.id === targetPid);
      desc = esc(blocker?.name) + ' claims ' + claimedChar + ' to block';
    } else if (gameState.pendingAction?.claimedCharacter) {
      targetPid = gameState.pendingAction.actorId;
      claimedChar = gameState.pendingAction.claimedCharacter;
      const actor = gameState.players.find(p => p.id === targetPid);
      desc = esc(actor?.name) + ' claims to have ' + claimedChar;
    }
    if (!targetPid || !claimedChar || targetPid === playerId || hasPassed) {
      area.innerHTML = '<div class="waiting">Waiting for others to respond...</div>';
      return;
    }
    let html = '<div class="panel panel-challenge">';
    html += '<h2>⚔️ Challenge Opportunity!</h2>';
    html += '<p>' + desc + '</p>';
    html += '<p style="color:#94a3b8;font-size:13px;margin:8px 0">If you challenge and they don\'t have ' + claimedChar + ', they lose influence.</p>';
    html += '<button class="btn btn-red" onclick="send(\'challenge\',{challengerId:\'' + playerId + '\',targetPlayerId:\'' + targetPid + '\',claimedCharacter:\'' + claimedChar + '\',isBlockChallenge:' + (!!gameState.pendingBlock) + '})">Challenge ' + claimedChar + '</button>';
    html += '<button class="btn btn-outline" onclick="send(\'pass-challenge\')">Allow</button></div>';
    area.innerHTML = html;
    return;
  }

  // Lose influence
  if (gameState.phase === 'lose_influence' && gameState.pendingInfluenceLoss === playerId) {
    const unrevealed = me.cards.filter(c => !c.revealed);
    let html = '<div class="panel panel-challenge"><h2>💀 Lose Influence</h2><p>Choose a card to reveal:</p><div style="margin:12px 0">';
    for (const c of unrevealed) {
      html += '<button class="btn char-' + c.character + '" style="margin:4px 0;display:flex;align-items:center;gap:8px" onclick="send(\'lose-influence\',{cardId:\'' + c.id + '\'})">' +
        '<img src="' + CHAR_IMG[c.character] + '" style="width:32px;height:32px;border-radius:6px;object-fit:cover">' + c.character + '</button>';
    }
    html += '</div></div>';
    area.innerHTML = html;
    return;
  }

  // Exchange
  if (gameState.phase === 'exchange' && gameState.pendingAction?.actorId === playerId && gameState.pendingExchangeCards) {
    exchangeSelected = new Set();
    const unrevealed = me.cards.filter(c => !c.revealed);
    const allCards = [...unrevealed, ...gameState.pendingExchangeCards];
    const keepCount = unrevealed.length;
    area.innerHTML = '<div class="panel panel-exchange"><h2>🔄 Exchange Cards</h2><p>Select ' + keepCount + ' card(s) to keep:</p>' +
      '<div id="exchange-cards" style="margin:12px 0">' +
      allCards.map(c => '<span class="card-box card-select" style="background-image:url(' + CHAR_IMG[c.character] + ')" data-id="' + c.id + '" onclick="toggleExchange(this)"><span class="card-label">' + c.character + '</span></span>').join('') +
      '</div><button class="btn btn-green" id="exchange-confirm" data-keep="' + keepCount + '" disabled onclick="confirmExchange(' + keepCount + ')">Confirm (0/' + keepCount + ')</button></div>';
    return;
  }

  // Interrogate select
  if (gameState.phase === 'interrogate_select' && gameState.pendingInterrogate?.targetId === playerId) {
    const unrevealed = me.cards.filter(c => !c.revealed);
    let html = '<div class="panel panel-exchange"><h2>🔍 Interrogation</h2><p>Select a card to show the Inquisitor:</p><div style="margin:12px 0">';
    for (const c of unrevealed) {
      html += '<button class="btn char-' + c.character + '" style="margin:4px 0;display:flex;align-items:center;gap:8px" onclick="send(\'interrogate-select\',{cardId:\'' + c.id + '\'})">' +
        '<img src="' + CHAR_IMG[c.character] + '" style="width:32px;height:32px;border-radius:6px;object-fit:cover">' + c.character + '</button>';
    }
    html += '</div></div>';
    area.innerHTML = html;
    return;
  }

  // Interrogate decision
  if (gameState.phase === 'interrogate_decision' && gameState.pendingAction?.actorId === playerId && gameState.pendingInterrogate) {
    const target = gameState.players.find(p => p.id === gameState.pendingInterrogate.targetId);
    const card = target?.cards.find(c => c.id === gameState.pendingInterrogate.selectedCardId);
    if (card) {
      area.innerHTML = '<div class="panel panel-exchange"><h2>🔍 Interrogate Decision</h2><p>' + esc(target.name) + ' revealed:</p>' +
        '<div style="margin:12px 0"><span class="card-box" style="width:120px;height:160px;background-image:url(' + CHAR_IMG[card.character] + ')"><span class="card-label" style="font-size:14px">' + card.character + '</span></span></div>' +
        '<div class="flex-row"><button class="btn btn-green" onclick="send(\'interrogate-decision\',{decision:\'keep\'})">Keep</button>' +
        '<button class="btn btn-amber" onclick="send(\'interrogate-decision\',{decision:\'replace\'})">Replace</button></div></div>';
      return;
    }
  }

  // Default waiting
  if (gameState.phase === 'action' && !isMyTurn) {
    area.innerHTML = '<div class="waiting">Waiting for ' + esc(cp.name) + '\'s action...</div>';
  } else if (gameState.phase === 'lose_influence' && gameState.pendingInfluenceLoss !== playerId) {
    const loser = gameState.players.find(p => p.id === gameState.pendingInfluenceLoss);
    area.innerHTML = '<div class="waiting">Waiting for ' + esc(loser?.name || '?') + ' to choose a card...</div>';
  } else if (['exchange','interrogate_select','interrogate_decision'].includes(gameState.phase) && gameState.pendingAction?.actorId !== playerId && gameState.pendingInterrogate?.targetId !== playerId) {
    area.innerHTML = '<div class="waiting">Waiting for another player...</div>';
  }
}

function actionClick(action) {
  const req = ACTION_REQS[action];
  if (req.needsTarget) {
    const targets = gameState.players.filter(p => p.isAlive && p.id !== playerId);
    let html = '<div class="panel panel-action"><h2>Select Target</h2>';
    for (const t of targets) {
      html += '<button class="btn btn-outline" style="margin:4px 0;text-align:left" onclick="send(\'action\',{type:\'' + action + '\',actorId:\'' + playerId + '\',targetId:\'' + t.id + '\'})">' +
        esc(t.name) + ' — 💰' + t.coins + ' 🛡' + t.cards.filter(c=>!c.revealed).length + '</button>';
    }
    html += '<button class="btn btn-outline" style="margin-top:8px" onclick="renderActions()">Cancel</button></div>';
    document.getElementById('action-area').innerHTML = html;
  } else {
    send('action', { type: action, actorId: playerId });
  }
}

let exchangeSelected = new Set();
function toggleExchange(el) {
  const id = el.dataset.id;
  if (exchangeSelected.has(id)) { exchangeSelected.delete(id); el.classList.remove('selected'); }
  else { exchangeSelected.add(id); el.classList.add('selected'); }
  const btn = document.getElementById('exchange-confirm');
  const keepCount = parseInt(btn.dataset.keep || '1');
  btn.textContent = 'Confirm (' + exchangeSelected.size + '/' + keepCount + ')';
  btn.disabled = exchangeSelected.size !== keepCount;
}
function confirmExchange(keepCount) {
  if (exchangeSelected.size === keepCount) {
    send('exchange', { keptCardIds: [...exchangeSelected] });
    exchangeSelected.clear();
  }
}

function renderLog() {
  const log = gameState.log || [];
  const grouped = {};
  for (const e of log) {
    if (!grouped[e.turn]) grouped[e.turn] = [];
    grouped[e.turn].push(e);
  }
  const turns = Object.keys(grouped).map(Number).sort((a,b) => b - a);
  let html = '';
  for (const t of turns) {
    html += '<div class="log-turn">Turn ' + t + '</div>';
    for (const e of grouped[t].slice().reverse()) {
      const time = new Date(e.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += '<div class="log-entry">' + esc(e.message) + ' <span style="color:#475569;font-size:10px">' + time + '</span></div>';
    }
  }
  document.getElementById('log-panel').innerHTML = html;
}

function toggleRules() { document.getElementById('rules-overlay').classList.toggle('active'); }
