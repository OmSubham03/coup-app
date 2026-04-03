// ========== LUDO GAME ==========

// 15x15 grid layout for Ludo board
const CELL = 100/15; // ~6.67%

function gridPos(col, row) {
  return { left: col * CELL + '%', top: row * CELL + '%' };
}

// 52 track positions as [col, row] on a 15×15 grid, clockwise from Red start
const TRACK = [
  // Red start going down: col 8, rows 0-5 (positions 0-5)
  [8,0],[8,1],[8,2],[8,3],[8,4],[8,5],
  // Top-right going right: row 6, cols 9-14 (positions 6-11)
  [9,6],[10,6],[11,6],[12,6],[13,6],[14,6],
  // Right turn down: (14,7) (position 12)
  [14,7],
  // Right-bottom going left: row 8, cols 14-9 (positions 13-18)
  [14,8],[13,8],[12,8],[11,8],[10,8],[9,8],
  // Green start going down: col 8, rows 9-14 (positions 19-24)
  [8,9],[8,10],[8,11],[8,12],[8,13],[8,14],
  // Bottom turn left: (7,14) (position 25)
  [7,14],
  // Bottom-left going up: col 6, rows 14-9 (positions 26-31)
  [6,14],[6,13],[6,12],[6,11],[6,10],[6,9],
  // Bottom-left going left: row 8, cols 5-0 (positions 32-37)
  [5,8],[4,8],[3,8],[2,8],[1,8],[0,8],
  // Left turn up: (0,7) (position 38)
  [0,7],
  // Left-top going right: row 6, cols 0-5 (positions 39-44)
  [0,6],[1,6],[2,6],[3,6],[4,6],[5,6],
  // Yellow start going up: col 6, rows 5-0 (positions 45-50)
  [6,5],[6,4],[6,3],[6,2],[6,1],[6,0],
  // Top turn right: (7,0) (position 51)
  [7,0]
];

// Home column positions for each color (6 squares toward center)
const HOME_COLS = {
  0: [[1,7],[2,7],[3,7],[4,7],[5,7],[6,7]],      // Red: left arm → center
  1: [[7,1],[7,2],[7,3],[7,4],[7,5],[7,6]],       // Green: top arm → center
  2: [[13,7],[12,7],[11,7],[10,7],[9,7],[8,7]],   // Blue: right arm → center
  3: [[7,13],[7,12],[7,11],[7,10],[7,9],[7,8]],   // Yellow: bottom arm → center
};

const COLOR_HEX = { red: '#dc2626', green: '#16a34a', blue: '#2563eb', yellow: '#eab308' };

// Track position just before home column entry for each colorIndex
const HOME_ENTRY_TRACK = [39, 0, 13, 26]; // Red, Green, Blue, Yellow

// Animation state
let animatingToken = null; // { playerIndex, tokenId }
let animationTimer = null;

// Dice dot positions for each face value (3x3 grid: tl,tc,tr,ml,mc,mr,bl,bc,br)
const DICE_DOTS = {
  1: [0,0,0, 0,1,0, 0,0,0],
  2: [0,0,1, 0,0,0, 1,0,0],
  3: [0,0,1, 0,1,0, 1,0,0],
  4: [1,0,1, 0,0,0, 1,0,1],
  5: [1,0,1, 0,1,0, 1,0,1],
  6: [1,0,1, 1,0,1, 1,0,1]
};

function renderDice(val) {
  const dots = DICE_DOTS[val] || DICE_DOTS[1];
  let h = '<div class="dice-face">';
  for (let i = 0; i < 9; i++) {
    h += dots[i] ? '<div class="dice-dot"></div>' : '<div></div>';
  }
  h += '</div>';
  return h;
}

let ludoState = null;

// Cancel any ongoing animation
function cancelAnimation() {
  if (animationTimer) {
    clearTimeout(animationTimer);
    animationTimer = null;
  }
  const wrapper = document.querySelector('.ludo-anim-wrapper');
  if (wrapper) wrapper.remove();
  animatingToken = null;
}

// Detect which token moved between old and new state
function detectTokenMove(oldState, newState) {
  if (!oldState || !oldState.players || !newState || !newState.players) return null;
  const cpi = oldState.currentPlayerIndex;
  if (cpi < 0 || cpi >= oldState.players.length) return null;
  const oldP = oldState.players[cpi];
  const newP = newState.players[cpi];
  if (!oldP || !newP || !oldP.tokens || !newP.tokens) return null;
  for (let ti = 0; ti < 4; ti++) {
    const oldT = oldP.tokens[ti];
    const newT = newP.tokens[ti];
    if (!oldT || !newT) continue;
    if (oldT.state !== newT.state || oldT.position !== newT.position) {
      if (newT.state === 'yard' && oldT.state !== 'yard') continue; // captured, skip
      return {
        playerIndex: cpi,
        tokenId: ti,
        color: newP.color,
        colorIndex: newP.colorIndex,
        tokenLabel: ti + 1,
        oldTokenState: oldT.state,
        oldPosition: oldT.position,
        newTokenState: newT.state,
        newPosition: newT.position
      };
    }
  }
  return null;
}

// Build array of [col, row] coordinates for the animation path
function buildAnimationPath(move) {
  const path = [];

  if (move.oldTokenState === 'yard' && move.newTokenState === 'track') {
    // Entering the board — no step animation
    return [];
  }

  if (move.oldTokenState === 'track' && move.newTokenState === 'track') {
    path.push(TRACK[move.oldPosition]);
    let pos = move.oldPosition;
    while (pos !== move.newPosition) {
      pos = (pos + 1) % 52;
      path.push(TRACK[pos]);
    }
    return path;
  }

  if (move.oldTokenState === 'track' && (move.newTokenState === 'home_col' || move.newTokenState === 'finished')) {
    path.push(TRACK[move.oldPosition]);
    const entry = HOME_ENTRY_TRACK[move.colorIndex];
    let pos = move.oldPosition;
    while (pos !== entry) {
      pos = (pos + 1) % 52;
      path.push(TRACK[pos]);
    }
    const homeCoords = HOME_COLS[move.colorIndex];
    const endH = move.newTokenState === 'finished' ? 5 : move.newPosition;
    for (let h = 0; h <= endH; h++) {
      path.push(homeCoords[h]);
    }
    return path;
  }

  if (move.oldTokenState === 'home_col' && move.newTokenState === 'home_col') {
    const homeCoords = HOME_COLS[move.colorIndex];
    for (let h = move.oldPosition; h <= move.newPosition; h++) {
      path.push(homeCoords[h]);
    }
    return path;
  }

  if (move.oldTokenState === 'home_col' && move.newTokenState === 'finished') {
    const homeCoords = HOME_COLS[move.colorIndex];
    for (let h = move.oldPosition; h < 6; h++) {
      path.push(homeCoords[h]);
    }
    return path;
  }

  return [];
}

// Animate a token stepping through the path
function animateMove(path, color, label, callback) {
  const board = document.querySelector('.ludo-board-inner');
  if (!board || path.length < 2) { callback(); return; }

  const wrapper = document.createElement('div');
  wrapper.className = 'ludo-anim-wrapper';
  const startPos = gridPos(path[0][0], path[0][1]);
  wrapper.style.left = startPos.left;
  wrapper.style.top = startPos.top;

  const tokenEl = document.createElement('div');
  tokenEl.className = 'ludo-token ludo-token-' + color;
  tokenEl.style.width = '22px';
  tokenEl.style.height = '22px';
  tokenEl.style.fontSize = '9px';
  tokenEl.textContent = label;
  wrapper.appendChild(tokenEl);
  board.appendChild(wrapper);

  // Force reflow then enable transition
  wrapper.offsetHeight;
  wrapper.style.transition = 'left 0.12s ease-out, top 0.12s ease-out';

  let step = 0;
  function nextStep() {
    step++;
    if (step >= path.length) {
      wrapper.remove();
      callback();
      return;
    }
    const pos = gridPos(path[step][0], path[step][1]);
    wrapper.style.left = pos.left;
    wrapper.style.top = pos.top;
    animationTimer = setTimeout(nextStep, 150);
  }
  animationTimer = setTimeout(nextStep, 100);
}

// Handle incoming ludo state: detect move and animate
function handleLudoStateUpdate(newState) {
  cancelAnimation();
  const oldState = ludoState;
  ludoState = newState;

  if (!oldState || !oldState.players) {
    renderLudoGame();
    return;
  }

  const move = detectTokenMove(oldState, newState);
  if (!move) {
    renderLudoGame();
    return;
  }

  const path = buildAnimationPath(move);
  if (path.length < 2) {
    renderLudoGame();
    return;
  }

  // Set animating flag so renderLudoBoard hides the token at destination
  animatingToken = { playerIndex: move.playerIndex, tokenId: move.tokenId };
  renderLudoGame();

  // Run step-by-step animation
  animateMove(path, move.color, move.tokenLabel, function() {
    animatingToken = null;
    renderLudoBoard(); // Re-render to reveal token at final position
  });
}

function switchLudoTab(tab) {
  document.getElementById('ltab-game').className = 'tab' + (tab === 'game' ? ' active' : '');
  document.getElementById('ltab-log').className = 'tab' + (tab === 'log' ? ' active' : '');
  document.getElementById('ltab-chat').className = 'tab' + (tab === 'chat' ? ' active' : '');
  document.getElementById('ludo-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('ludo-log-tab').style.display = tab === 'log' ? '' : 'none';
  document.getElementById('ludo-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  if (tab === 'chat') document.getElementById('ltab-chat').classList.remove('chat-unread');
}

function renderLudoGame() {
  if (!ludoState) return;
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = '';

  const phase = ludoState.phase === 'finished' ? 'Game Over' : (ludoState.phase === 'rolling' ? 'Roll the Dice' : 'Move a Token');
  document.getElementById('ludo-phase-display').textContent = phase;
  document.getElementById('ludo-turn-display').textContent = 'Turn ' + ludoState.turnNumber;

  const exitBtn = document.getElementById('ludo-exit-btn');
  exitBtn.style.display = ludoState.phase === 'finished' ? 'none' : '';

  renderLudoBoard();
  renderLudoActions();
  renderLudoLog();
}

function renderLudoBoard() {
  const ls = ludoState;
  const cp = ls.players[ls.currentPlayerIndex];
  const isMyTurn = cp.id === playerId;

  let html = '';

  // Turn info bar above board
  const diceVal = ls.diceValue > 0 ? ls.diceValue : 1;
  html += '<div class="ludo-turn-bar">';
  html += '<div class="turn-player" style="color:' + COLOR_HEX[cp.color] + '">' + esc(cp.name) + (cp.id === playerId ? ' (You)' : '') + '</div>';
  html += '<div class="turn-dice">' + renderDice(diceVal) + '</div>';
  html += '</div>';

  // Board
  html += '<div class="ludo-board-wrapper"><div class="ludo-board"><div class="ludo-board-inner">';

  // Draw yards (corner quadrants)
  const yardColors = ['red', 'green', 'blue', 'yellow'];
  for (let ci = 0; ci < 4; ci++) {
    html += '<div class="ludo-yard ludo-yard-' + yardColors[ci] + '">';
    // Player name label inside yard
    const yardPlayer = ls.players.find(p => p.colorIndex === ci);
    if (yardPlayer) {
      html += '<div class="ludo-yard-name">' + esc(yardPlayer.name) + '</div>';
    }
    // Find player with this color index
    const player = yardPlayer;
    if (player) {
      for (let t = 0; t < 4; t++) {
        if (player.tokens[t].state === 'yard') {
          const canMove = isMyTurn && player.id === playerId && ls.phase === 'moving' && ls.movableTokens && ls.movableTokens.includes(t);
          html += '<div class="ludo-token ludo-token-' + player.color + (canMove ? ' movable' : '') + '"' +
            (canMove ? ' onclick="send(\'ludo-move\',{tokenId:' + t + '})"' : '') +
            '>' + (t+1) + '</div>';
        } else {
          html += '<div style="width:28px;height:28px"></div>';
        }
      }
    }
    html += '</div>';
  }

  // Draw center home
  html += '<div class="ludo-center">';
  html += '<div class="ludo-center-q ludo-center-red"></div>';
  html += '<div class="ludo-center-q ludo-center-green"></div>';
  html += '<div class="ludo-center-q ludo-center-yellow"></div>';
  html += '<div class="ludo-center-q ludo-center-blue"></div>';
  html += '</div>';

  // Draw track squares
  for (let i = 0; i < 52; i++) {
    const [col, row] = TRACK[i];
    const pos = gridPos(col, row);
    let sqCls = 'ludo-square';
    // Mark safe/star squares
    if ([1, 9, 14, 22, 27, 35, 40, 48].includes(i)) sqCls += ' safe';
    // Mark start squares
    if (i === 40) sqCls += ' start-red';
    if (i === 1) sqCls += ' start-green';
    if (i === 14) sqCls += ' start-blue';
    if (i === 27) sqCls += ' start-yellow';

    html += '<div class="' + sqCls + '" style="left:' + pos.left + ';top:' + pos.top + '">';

    // Draw tokens on this square
    for (let pi = 0; pi < ls.players.length; pi++) {
      const p = ls.players[pi];
      for (let t = 0; t < 4; t++) {
        if (p.tokens[t].state === 'track' && p.tokens[t].position === i) {
          const isAnim = animatingToken && animatingToken.playerIndex === pi && animatingToken.tokenId === t;
          const canMove = !isAnim && isMyTurn && p.id === playerId && ls.phase === 'moving' && ls.movableTokens && ls.movableTokens.includes(t);
          html += '<div class="ludo-token ludo-token-' + p.color + (canMove ? ' movable' : '') + '" style="width:22px;height:22px;font-size:9px' + (isAnim ? ';visibility:hidden' : '') + '"' +
            (canMove ? ' onclick="event.stopPropagation();send(\'ludo-move\',{tokenId:' + t + '})"' : '') +
            '>' + (t+1) + '</div>';
        }
      }
    }
    html += '</div>';
  }

  // Draw home column squares
  for (let pi = 0; pi < ls.players.length; pi++) {
    const p = ls.players[pi];
    const homeCoords = HOME_COLS[p.colorIndex];
    if (!homeCoords) continue;
    for (let h = 0; h < 6; h++) {
      const [col, row] = homeCoords[h];
      const pos = gridPos(col, row);
      html += '<div class="ludo-square ludo-home-' + p.color + '" style="left:' + pos.left + ';top:' + pos.top + '">';
      // Draw tokens in home column
      for (let t = 0; t < 4; t++) {
        if (p.tokens[t].state === 'home_col' && p.tokens[t].position === h) {
          const isAnim = animatingToken && animatingToken.playerIndex === pi && animatingToken.tokenId === t;
          const canMove = !isAnim && isMyTurn && p.id === playerId && ls.phase === 'moving' && ls.movableTokens && ls.movableTokens.includes(t);
          html += '<div class="ludo-token ludo-token-' + p.color + (canMove ? ' movable' : '') + '" style="width:22px;height:22px;font-size:9px' + (isAnim ? ';visibility:hidden' : '') + '"' +
            (canMove ? ' onclick="event.stopPropagation();send(\'ludo-move\',{tokenId:' + t + '})"' : '') +
            '>' + (t+1) + '</div>';
        }
      }
      html += '</div>';
    }
  }

  html += '</div></div></div>'; // close board-inner, board, wrapper

  // Last action
  if (ls.lastAction) {
    html += '<div class="ludo-last-action">' + esc(ls.lastAction) + '</div>';
  }

  document.getElementById('ludo-table-area').innerHTML = html;
}

function renderLudoActions() {
  const area = document.getElementById('ludo-action-area');
  const ls = ludoState;
  const cp = ls.players[ls.currentPlayerIndex];
  const isMyTurn = cp.id === playerId;
  const me = ls.players.find(p => p.id === playerId);

  // Game over
  if (ls.phase === 'finished') {
    const winner = ls.players.find(p => p.id === ls.winner);
    const isWinner = winner?.id === playerId;
    area.innerHTML = '<div class="victory"><div style="font-size:64px">' + (isWinner ? '🎉' : '🏆') + '</div><h2>' +
      (isWinner ? 'Victory!' : 'Game Over') + '</h2><p style="font-size:24px;margin-bottom:16px">' +
      esc(winner?.name || '?') + ' wins!</p>' +
      '<div style="margin-bottom:16px">';
    // Show finish order
    const sorted = [...ls.players].sort((a,b) => a.finishOrder - b.finishOrder);
    for (const p of sorted) {
      area.innerHTML; // just building
    }
    let orderHtml = '';
    for (const p of sorted) {
      orderHtml += '<div style="padding:6px 0;font-size:16px"><span style="color:' + COLOR_HEX[p.color] + ';font-weight:700">#' + p.finishOrder + '</span> ' + esc(p.name) + '</div>';
    }
    area.innerHTML = '<div class="victory"><div style="font-size:64px">' + (isWinner ? '🎉' : '🏆') + '</div><h2 style="color:' + COLOR_HEX[winner?.color || 'red'] + '">' +
      (isWinner ? 'Victory!' : 'Game Over') + '</h2><p style="font-size:24px;margin-bottom:16px">' +
      esc(winner?.name || '?') + ' wins!</p>' + orderHtml +
      '<button class="btn btn-purple" style="margin-top:16px" onclick="send(\'return-to-lobby\')">Return to Lobby</button></div>';
    return;
  }

  // Not my turn
  if (!isMyTurn || !me || me.finishOrder > 0) {
    area.innerHTML = '<div class="waiting">Waiting for ' + esc(cp.name) + '...</div>';
    return;
  }

  // Rolling phase
  if (ls.phase === 'rolling') {
    area.innerHTML = '<button class="ludo-move-btn ludo-roll-btn" onclick="send(\'ludo-roll\')">🎲 Roll Dice</button>';
    return;
  }

  // Moving phase — show movable tokens
  if (ls.phase === 'moving' && ls.movableTokens && ls.movableTokens.length > 0) {
    let html = '<p style="color:#94a3b8;text-align:center;margin-bottom:8px">Tap a highlighted token on the board, or pick below:</p>';
    for (const tid of ls.movableTokens) {
      const tok = me.tokens[tid];
      let desc = '';
      if (tok.state === 'yard') desc = 'Enter the board';
      else if (tok.state === 'track') desc = 'Move ' + ls.diceValue + ' spaces';
      else if (tok.state === 'home_col') desc = 'Move in home column';
      html += '<button class="ludo-move-btn" style="background:' + COLOR_HEX[me.color] + ';color:' + (me.color === 'yellow' ? '#000' : '#fff') + '" onclick="send(\'ludo-move\',{tokenId:' + tid + '})">Token ' + (tid+1) + ' — ' + desc + '</button>';
    }
    area.innerHTML = html;
    return;
  }

  area.innerHTML = '<div class="waiting">Waiting...</div>';
}

function renderLudoLog() {
  const log = ludoState.log || [];
  const grouped = {};
  for (const e of log) {
    if (!grouped[e.turn]) grouped[e.turn] = [];
    grouped[e.turn].push(e);
  }
  const turns = Object.keys(grouped).map(Number).sort((a,b) => b - a);
  let html = '';
  for (const t of turns) {
    html += '<div class="log-turn" style="background:#2d0a0a;color:#ef4444">Turn ' + t + '</div>';
    for (const e of grouped[t].slice().reverse()) {
      const time = new Date(e.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += '<div class="log-entry">' + esc(e.message) + ' <span style="color:#475569;font-size:10px">' + time + '</span></div>';
    }
  }
  document.getElementById('ludo-log-panel').innerHTML = html;
}

function toggleLudoRules() { document.getElementById('ludo-rules-overlay').classList.toggle('active'); }
