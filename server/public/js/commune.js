// ========== COMMUNE GAME ==========

let communeState = null;

const CM_HAND_TYPES = [
  { id: 0, name: 'High Card', short: 'High' },
  { id: 1, name: 'Pair', short: 'Pair' },
  { id: 2, name: 'Two Pair', short: '2 Pair' },
  { id: 3, name: 'Three of a Kind', short: 'Trips' },
  { id: 4, name: 'Straight', short: 'Straight' },
  { id: 6, name: 'Full House', short: 'Full H.' },
  { id: 7, name: 'Four of a Kind', short: 'Quads' },
  { id: 8, name: 'Two Trips', short: '2 Trips' },
  { id: 9, name: 'Ulta Straight', short: 'Ulta St.' },
  { id: 10, name: 'Four + Three', short: '4+3' },
  { id: 11, name: 'Five of a Kind', short: '5 Kind' }
];

const CM_RANKS = [
  { v: 3, l: '3' }, { v: 4, l: '4' }, { v: 5, l: '5' }, { v: 6, l: '6' },
  { v: 7, l: '7' }, { v: 8, l: '8' }, { v: 9, l: '9' }, { v: 10, l: '10' },
  { v: 11, l: 'J' }, { v: 12, l: 'Q' }, { v: 13, l: 'K' }, { v: 14, l: 'A' }
];

let cmSelType = -1, cmSelRank1 = 0, cmSelRank2 = 0;

function handleCommuneStateUpdate(payload) {
  communeState = payload;
  renderCommuneGame();
}

function renderCommuneGame() {
  if (!communeState) return;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('nq-active').style.display = 'none';
  document.getElementById('commune-active').style.display = '';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';

  const me = communeState.players.find(p => p.id === playerId);
  const phaseEl = document.getElementById('commune-phase-display');
  const turnEl = document.getElementById('commune-turn-display');
  const tableArea = document.getElementById('commune-table-area');
  const actionArea = document.getElementById('commune-action-area');

  switch (communeState.phase) {
    case 'playing':
      phaseEl.textContent = `Round ${communeState.round}`;
      turnEl.textContent = `${communeState.players.filter(p => p.isAlive).length} alive`;
      renderCommunePlaying(tableArea, actionArea, me);
      break;
    case 'called':
      phaseEl.textContent = 'Called!';
      turnEl.textContent = '';
      renderCommuneCalled(tableArea, actionArea, me);
      break;
    case 'finished':
      phaseEl.textContent = 'Game Over';
      turnEl.textContent = '';
      renderCommuneFinished(tableArea, actionArea);
      break;
  }

  const logPanel = document.getElementById('commune-log-panel');
  if (logPanel && communeState.log) {
    let html = '';
    for (const entry of communeState.log) {
      const t = new Date(entry.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += `<div class="cm-log-entry"><span class="cm-log-time">${t}</span> ${esc(entry.message)}</div>`;
    }
    logPanel.innerHTML = html;
    logPanel.scrollTop = logPanel.scrollHeight;
  }
}

// --- Card rendering ---
function cmCard(card) {
  if (!card || card.id === 'hidden') return '<div class="cm-card cm-card-hidden">?</div>';
  const suits = { hearts: '♥', diamonds: '♦', clubs: '♣', spades: '♠' };
  const red = card.suit === 'hearts' || card.suit === 'diamonds';
  const ranks = {2:'2',3:'3',4:'4',5:'5',6:'6',7:'7',8:'8',9:'9',10:'10',11:'J',12:'Q',13:'K',14:'A'};
  const wc = card.isWild ? ' cm-card-wild' : '';
  return `<div class="cm-card${wc}${red ? ' cm-red' : ''}"><span class="cm-card-rank">${ranks[card.rank]}</span><span class="cm-card-suit">${suits[card.suit]}</span></div>`;
}

function cmRankLabel(rank) {
  const n = {3:'3s',4:'4s',5:'5s',6:'6s',7:'7s',8:'8s',9:'9s',10:'10s',11:'Jacks',12:'Queens',13:'Kings',14:'Aces'};
  return n[rank] || rank;
}

// --- Playing phase ---
function renderCommunePlaying(tableArea, actionArea, me) {
  const cur = communeState.players[communeState.currentPlayerIdx];
  const isMyTurn = cur && cur.id === playerId;

  // Community cards
  let communityHtml = '<div class="cm-community"><div class="cm-hand-label">Community Cards</div><div class="cm-cards">';
  if (communeState.communityCards) {
    for (const c of communeState.communityCards) communityHtml += cmCard(c);
  }
  communityHtml += '</div></div>';

  // Players bar
  let playersHtml = '<div class="cm-players">';
  for (let i = 0; i < communeState.players.length; i++) {
    const p = communeState.players[i];
    const isCur = i === communeState.currentPlayerIdx;
    const cls = !p.isAlive ? 'cm-p-dead' : isCur ? 'cm-p-active' : '';
    const tokens = '●'.repeat(p.tokens) + '○'.repeat(5 - p.tokens);
    const cardCount = p.cards ? p.cards.length : 0;
    const isMe = p.id === playerId;
    playersHtml += `<div class="cm-player ${cls}">
      <div class="cm-p-name">${esc(p.name)}${isMe ? ' <span class="badge">You</span>' : ''}${i === communeState.dealerIdx ? ' 🎴' : ''}</div>
      <div class="cm-p-info"><span class="cm-tokens">${tokens}</span> · ${cardCount} card${cardCount !== 1 ? 's' : ''}</div>
    </div>`;
  }
  playersHtml += '</div>';

  // My cards
  let cardsHtml = '';
  if (me && me.cards) {
    cardsHtml = '<div class="cm-my-hand"><div class="cm-hand-label">Your Hand</div><div class="cm-cards">';
    for (const c of me.cards) cardsHtml += cmCard(c);
    cardsHtml += '</div></div>';
  }

  // Declarations
  let declHtml = '<div class="cm-decl-list">';
  if (communeState.declarations.length === 0) {
    declHtml += '<div class="cm-decl-empty">No declarations yet</div>';
  } else {
    for (const d of communeState.declarations) {
      declHtml += `<div class="cm-decl-item"><strong>${esc(d.playerName)}</strong>: ${esc(d.displayText)}</div>`;
    }
  }
  declHtml += '</div>';

  tableArea.innerHTML = communityHtml + playersHtml + cardsHtml + declHtml;

  // Action area
  if (isMyTurn && !isSpectating) {
    renderCommuneDeclareForm(actionArea);
  } else {
    actionArea.innerHTML = `<div class="cm-wait">${esc(cur?.name || '...')}'s turn to declare...</div>`;
  }
}

function renderCommuneDeclareForm(actionArea) {
  const lastDecl = communeState.lastDeclaration;

  // Hand type buttons
  let typesHtml = '<div class="cm-type-picker">';
  for (const ht of CM_HAND_TYPES) {
    const sel = cmSelType === ht.id ? ' selected' : '';
    // Gray out types that are too low
    let disabled = '';
    if (lastDecl && ht.id < lastDecl.handType) disabled = ' disabled';
    typesHtml += `<button class="cm-type-btn${sel}" ${disabled} onclick="cmPickType(${ht.id})">${ht.short}</button>`;
  }
  typesHtml += '</div>';

  // Rank pickers based on type
  let ranksHtml = '';
  if (cmSelType >= 0) {
    const needsRank2 = (cmSelType === 2 || cmSelType === 6 || cmSelType === 8 || cmSelType === 10); // two pair, full house, two trips, four+three
    const isStraight = (cmSelType === 4);
    const isUlta = (cmSelType === 9);
    const validRanks = isStraight ? CM_RANKS.filter(r => r.v >= 5) : CM_RANKS;
    const sameType = lastDecl && cmSelType === lastDecl.handType;

    if (isUlta) {
      ranksHtml = '<div class="cm-rank-section"><div class="cm-rank-label">3 through Ace — all 12 ranks</div></div>';
    } else {
      let r1Label = 'Rank';
      if (cmSelType === 6) r1Label = 'Three-of-a-kind rank';
      else if (cmSelType === 2 || cmSelType === 8) r1Label = 'Higher rank';
      else if (cmSelType === 10) r1Label = 'Quads rank';
      else if (isStraight) r1Label = 'High card';

      ranksHtml += '<div class="cm-rank-section"><div class="cm-rank-label">' + r1Label +
        '</div><div class="cm-rank-picker">';
      for (const r of validRanks) {
        const sel = cmSelRank1 === r.v ? ' selected' : '';
        let dis = '';
        if (sameType) {
          if (needsRank2 ? r.v < lastDecl.primaryRank : r.v <= lastDecl.primaryRank) dis = ' disabled';
        }
        ranksHtml += `<button class="cm-rank-btn${sel}"${dis} onclick="cmPickRank1(${r.v})">${r.l}</button>`;
      }
      ranksHtml += '</div></div>';

      if (needsRank2 && cmSelRank1 > 0) {
        let r2Label = 'Lower rank';
        if (cmSelType === 6) r2Label = 'Pair rank';
        else if (cmSelType === 10) r2Label = 'Trips rank';

        const r2Ranks = (cmSelType === 2 || cmSelType === 8)
          ? CM_RANKS.filter(r => r.v < cmSelRank1)
          : CM_RANKS.filter(r => r.v !== cmSelRank1);
        ranksHtml += `<div class="cm-rank-section"><div class="cm-rank-label">${r2Label}</div><div class="cm-rank-picker">`;
        for (const r of r2Ranks) {
          const sel = cmSelRank2 === r.v ? ' selected' : '';
          let dis = '';
          if (sameType && cmSelRank1 === lastDecl.primaryRank && r.v <= lastDecl.secondaryRank) dis = ' disabled';
          ranksHtml += `<button class="cm-rank-btn${sel}"${dis} onclick="cmPickRank2(${r.v})">${r.l}</button>`;
        }
        ranksHtml += '</div></div>';
      }
    }
  }

  // Preview & buttons
  const isUltaSel = cmSelType === 9;
  const needsR2 = (cmSelType === 2 || cmSelType === 6 || cmSelType === 8 || cmSelType === 10);
  const canDeclare = cmSelType >= 0 && (isUltaSel || (cmSelRank1 > 0 && (!needsR2 || cmSelRank2 > 0)));
  const previewText = canDeclare ? cmDeclPreview() : '';
  const canCall = !!lastDecl;

  let buttonsHtml = '<div class="cm-action-buttons">';
  if (canDeclare) {
    buttonsHtml += `<button class="btn cm-btn-declare" onclick="cmDeclare()">Declare: ${previewText}</button>`;
  }
  if (canCall) {
    buttonsHtml += `<button class="btn cm-btn-call" onclick="cmCallIt()">📣 Call!</button>`;
  }
  buttonsHtml += '</div>';

  actionArea.innerHTML = '<div class="cm-declare-form"><p class="cm-prompt">Your turn — declare a hand or call!</p>' +
    typesHtml + ranksHtml + buttonsHtml + '</div>';
}

function cmDeclPreview() {
  const r = { 3:'3s',4:'4s',5:'5s',6:'6s',7:'7s',8:'8s',9:'9s',10:'10s',11:'Jacks',12:'Queens',13:'Kings',14:'Aces' };
  const rs = { 3:'3',4:'4',5:'5',6:'6',7:'7',8:'8',9:'9',10:'10',11:'J',12:'Q',13:'K',14:'A' };
  switch (cmSelType) {
    case 0: return `High Card ${rs[cmSelRank1]}`;
    case 1: return `Pair of ${r[cmSelRank1]}`;
    case 2: return `Two Pair: ${r[cmSelRank1]} & ${r[cmSelRank2]}`;
    case 3: return `Three ${r[cmSelRank1]}`;
    case 4: return `Straight ${rs[cmSelRank1]}-high`;
    case 6: return `${r[cmSelRank1]} full of ${r[cmSelRank2]}`;
    case 7: return `Four ${r[cmSelRank1]}`;
    case 8: return `Two Trips: ${r[cmSelRank1]} & ${r[cmSelRank2]}`;
    case 9: return `Ulta Straight (3→A)`;
    case 10: return `Four ${r[cmSelRank1]} + Three ${r[cmSelRank2]}`;
    case 11: return `Five ${r[cmSelRank1]}`;
  }
  return '';
}

function cmPickType(t) { cmSelType = t; cmSelRank1 = 0; cmSelRank2 = 0; renderCommuneGame(); }
function cmPickRank1(r) { cmSelRank1 = r; cmSelRank2 = 0; renderCommuneGame(); }
function cmPickRank2(r) { cmSelRank2 = r; renderCommuneGame(); }

function cmDeclare() {
  send('commune-declare', { handType: cmSelType, primaryRank: cmSelRank1, secondaryRank: cmSelRank2 });
  cmSelType = -1; cmSelRank1 = 0; cmSelRank2 = 0;
}

function cmCallIt() {
  send('commune-call');
}

// --- Called phase ---
function renderCommuneCalled(tableArea, actionArea, me) {
  const caller = communeState.players[communeState.callerIdx];
  const declarer = communeState.players[communeState.lastDeclarerIdx];
  const lastDecl = communeState.declarations[communeState.declarations.length - 1];
  const loser = communeState.players[communeState.loserIdx];

  let resultClass = communeState.callResult === 'caller_loses' ? 'cm-result-present' : 'cm-result-absent';
  let resultText = communeState.callResult === 'caller_loses'
    ? `✅ Hand IS present! ${esc(caller?.name)} loses a token.`
    : `❌ Hand NOT present! ${esc(declarer?.name)} loses a token.`;

  // Community cards
  let communityHtml = '<div class="cm-community"><div class="cm-hand-label">Community Cards</div><div class="cm-cards">';
  if (communeState.communityCards) {
    for (const c of communeState.communityCards) communityHtml += cmCard(c);
  }
  communityHtml += '</div></div>';

  // Show all players' cards
  let cardsHtml = '<div class="cm-all-cards">';
  for (const p of communeState.players) {
    if (!p.cards || p.cards.length === 0) continue;
    cardsHtml += `<div class="cm-player-cards"><div class="cm-pc-name">${esc(p.name)}</div><div class="cm-cards">`;
    for (const c of p.cards) cardsHtml += cmCard(c);
    cardsHtml += '</div></div>';
  }
  cardsHtml += '</div>';

  // Declaration history
  let declHtml = '<div class="cm-decl-list">';
  for (const d of communeState.declarations) {
    declHtml += `<div class="cm-decl-item"><strong>${esc(d.playerName)}</strong>: ${esc(d.displayText)}</div>`;
  }
  declHtml += '</div>';

  tableArea.innerHTML = `
    <div class="cm-call-banner">
      <div class="cm-call-title">📣 ${esc(caller?.name)} called ${esc(declarer?.name)}'s declaration!</div>
      <div class="cm-call-decl">"${esc(lastDecl?.displayText || '')}"</div>
    </div>
    ${communityHtml}
    ${cardsHtml}
    <div class="cm-result ${resultClass}">${resultText}</div>
    ${declHtml}`;

  actionArea.innerHTML = `<button class="btn cm-btn-next" onclick="send('commune-next-hand')">Deal Next Hand</button>`;
}

// --- Finished phase ---
function renderCommuneFinished(tableArea, actionArea) {
  const winnerPlayer = communeState.players.find(p => p.name === communeState.winner);

  // Show all players' cards if we came from a call
  let cardsHtml = '';
  if (communeState.allCards && communeState.allCards.length > 0) {
    cardsHtml = '<div class="cm-all-cards">';
    for (const p of communeState.players) {
      if (!p.cards || p.cards.length === 0) continue;
      cardsHtml += `<div class="cm-player-cards"><div class="cm-pc-name">${esc(p.name)}</div><div class="cm-cards">`;
      for (const c of p.cards) cardsHtml += cmCard(c);
      cardsHtml += '</div></div>';
    }
    cardsHtml += '</div>';
  }

  // Final standings
  let standingsHtml = '<div class="cm-standings"><div class="cm-stand-title">Final Standings</div>';
  const sorted = [...communeState.players].sort((a, b) => b.tokens - a.tokens);
  for (const p of sorted) {
    const tokens = '●'.repeat(p.tokens) + '○'.repeat(5 - p.tokens);
    standingsHtml += `<div class="cm-stand-row${p.isAlive ? ' cm-stand-alive' : ''}">
      <span>${esc(p.name)}</span><span class="cm-tokens">${tokens}</span>
    </div>`;
  }
  standingsHtml += '</div>';

  tableArea.innerHTML = `
    <div class="cm-winner-card">
      <div class="cm-winner-icon">🏆</div>
      <h3>${esc(communeState.winner)} wins!</h3>
    </div>
    ${cardsHtml}${standingsHtml}`;

  const isHost = hostId === playerId;
  actionArea.innerHTML = isHost
    ? `<button class="btn cm-btn-lobby" onclick="send('return-to-lobby')">Return to Lobby</button>`
    : `<div class="cm-wait">Waiting for host...</div>`;
}

function switchCommuneTab(tab) {
  document.getElementById('cmtab-game').className = 'tab' + (tab === 'game' ? ' active' : '');
  document.getElementById('cmtab-log').className = 'tab' + (tab === 'log' ? ' active' : '');
  document.getElementById('cmtab-chat').className = 'tab' + (tab === 'chat' ? ' active' : '');
  document.getElementById('commune-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('commune-log-tab').style.display = tab === 'log' ? '' : 'none';
  document.getElementById('commune-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  if (tab === 'chat') {
    const ct = document.getElementById('cmtab-chat');
    if (ct) ct.classList.remove('chat-unread');
  }
}

function toggleCommuneRules() {
  const el = document.getElementById('commune-rules-overlay');
  el.classList.toggle('active');
}
