// ========== COMMON: Connection, WebSocket, Screens, Profile, Chat ==========

const isLocal = location.port === '8080' || location.hostname === 'localhost';
const SERVER = isLocal ? location.hostname + ':8080' : location.host;
const HTTP = location.protocol === 'https:' ? 'https://' : 'http://';
const WS = location.protocol === 'https:' ? 'wss://' : 'ws://';
let ws = null;
let playerId = sessionStorage.getItem('coup_pid') || crypto.randomUUID();
sessionStorage.setItem('coup_pid', playerId);
let roomCode = '';
let variant = 'standard';
let gameState = null;
let pokerState = null;
let hostId = null;
let joinedName = '';
let joinVariant = 'standard';
let isSpectating = false;
let gameActive = false;
let currentGameType = 'coup';

function openGameMenu(type) {
  currentGameType = type;
  document.getElementById('game-list-view').style.display = 'none';
  document.getElementById('menu-coup').style.display = type === 'coup' ? '' : 'none';
  document.getElementById('menu-poker').style.display = type === 'poker' ? '' : 'none';
  document.getElementById('menu-ludo').style.display = type === 'ludo' ? '' : 'none';
  document.getElementById('join-variant-row').style.display = type === 'coup' ? '' : 'none';
  const gameNames = { coup: 'COUP', poker: 'POKER', ludo: 'LUDO' };
  const gameClasses = { coup: 'coup-title', poker: 'poker-title', ludo: 'ludo-title' };
  const gameSubs = { coup: 'Bluff. Deceive. Survive.', poker: 'Texas Hold\u2019em. All In.', ludo: 'Roll. Race. Win.' };
  ['join-game-title', 'entry-game-title', 'lobby-game-title'].forEach(id => {
    const el = document.getElementById(id);
    if (el) { el.textContent = gameNames[type] || type.toUpperCase(); el.className = (gameClasses[type] || '') + ' '; el.style.fontSize = '36px'; }
  });
  ['join-game-subtitle', 'entry-game-subtitle', 'lobby-game-subtitle'].forEach(id => {
    const el = document.getElementById(id);
    if (el) { el.textContent = gameSubs[type] || ''; }
  });
}

function backToGameList() {
  document.getElementById('game-list-view').style.display = '';
  document.getElementById('menu-coup').style.display = 'none';
  document.getElementById('menu-poker').style.display = 'none';
  document.getElementById('menu-ludo').style.display = 'none';
}

function showScreen(name) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById('screen-' + name).classList.add('active');
}

function switchTab(tab) {
  document.getElementById('tab-game').className = 'tab' + (tab === 'game' ? ' active' : '');
  document.getElementById('tab-log').className = 'tab' + (tab === 'log' ? ' active' : '');
  document.getElementById('tab-chat').className = 'tab' + (tab === 'chat' ? ' active' : '');
  document.getElementById('game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('log-tab').style.display = tab === 'log' ? '' : 'none';
  document.getElementById('chat-tab').style.display = tab === 'chat' ? '' : 'none';
  if (tab === 'chat') document.getElementById('tab-chat').classList.remove('chat-unread');
}

function setJoinVariant(v) {
  joinVariant = v;
  document.getElementById('join-var-standard').className = 'variant-btn' + (v === 'standard' ? ' active-std' : '');
  document.getElementById('join-var-inquisitor').className = 'variant-btn' + (v === 'inquisitor' ? ' active-inq' : '');
}

async function createGame(v) {
  variant = v;
  currentGameType = 'coup';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create');
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

async function createPokerGame() {
  const buyIn = parseInt(document.getElementById('poker-buyin').value) || 1000;
  const sb = parseInt(document.getElementById('poker-sb').value) || 10;
  if (buyIn < 100) { alert('Buy-in must be at least 100'); return; }
  if (sb < 1) { alert('Small blind must be at least 1'); return; }
  if (sb * 2 > buyIn) { alert('Big blind (2x small blind) cannot exceed buy-in'); return; }
  currentGameType = 'poker';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create', { buyIn, smallBlind: sb });
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

let selectedLudoColor = 'red';
function selectLudoColor(color) {
  selectedLudoColor = color;
  document.querySelectorAll('#ludo-color-picker .ludo-color-option').forEach(el => {
    el.classList.toggle('selected', el.dataset.color === color);
  });
}

function lobbySelectLudoColor(color) {
  selectedLudoColor = color;
  document.querySelectorAll('#lobby-ludo-colors .ludo-color-option').forEach(el => {
    el.classList.toggle('selected', el.dataset.color === color);
  });
  send('set-ludo-color', { color: color });
}

async function createLudoGame() {
  currentGameType = 'ludo';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create');
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinWithCode() {
  const code = document.getElementById('join-code').value.trim().toUpperCase();
  if (!code) return;
  roomCode = code;
  variant = joinVariant;
  connectWS();
}

let coupJoinVariant = 'standard';
function setCoupJoinVariant(v) {
  coupJoinVariant = v;
  document.getElementById('coup-join-var-standard').className = 'variant-btn' + (v === 'standard' ? ' active-std' : '');
  document.getElementById('coup-join-var-inquisitor').className = 'variant-btn' + (v === 'inquisitor' ? ' active-inq' : '');
}

function joinCoupWithCode() {
  const code = document.getElementById('coup-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('coup-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'coup';
  roomCode = code;
  variant = coupJoinVariant;
  connectWS();
}

function joinPokerWithCode() {
  const code = document.getElementById('poker-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('poker-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'poker';
  roomCode = code;
  connectWS();
}

function joinLudoWithCode() {
  const code = document.getElementById('ludo-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('ludo-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'ludo';
  roomCode = code;
  connectWS();
}

let intentionalDisconnect = false;
let reconnectTimer = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 15;
let wsConnecting = false;

function connectWS(action, pokerConfig) {
  intentionalDisconnect = false;
  wsConnecting = true;
  showConnectingState();
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
  if (ws) ws.close();
  sessionStorage.setItem('coup_room', roomCode);
  sessionStorage.setItem('coup_variant', variant);
  sessionStorage.setItem('coup_gameType', currentGameType);
  const params = new URLSearchParams({ room: roomCode, playerId, variant, gameType: currentGameType });
  if (action) params.set('action', action);
  ws = new WebSocket(WS + SERVER + '/ws?' + params);

  ws.onopen = () => {
    wsConnecting = false;
    hideConnectingState();
    // Reset join button states
    resetJoinButtons();
    showScreen('game');
    if (action === 'create' && currentGameType === 'poker' && pokerConfig) {
      setTimeout(() => send('set-poker-config', pokerConfig), 100);
    }
    const savedName = localStorage.getItem('coup_name');
    if (savedName) {
      document.getElementById('name-entry').style.display = 'none';
      document.getElementById('lobby').style.display = '';
      document.getElementById('lobby-code').textContent = roomCode;
      joinedName = getFullName(savedName);
      sessionStorage.setItem('coup_joinedName', joinedName);
      send('join', { playerName: joinedName });
    } else {
      document.getElementById('name-entry').style.display = '';
      document.getElementById('lobby').style.display = 'none';
    }
    document.getElementById('game-active').style.display = 'none';
    document.getElementById('poker-active').style.display = 'none';
    document.getElementById('ludo-active').style.display = 'none';
    document.getElementById('room-code-display').textContent = roomCode;
    document.getElementById('name-error').textContent = '';
  };

  ws.onmessage = handleWSMessage;
  ws.onclose = () => {
    wsConnecting = false;
    hideConnectingState();
    resetJoinButtons();
    if (!intentionalDisconnect && roomCode && joinedName) {
      console.log('[WS] Connection lost, reconnecting in 1s...');
      document.getElementById('conn-banner').classList.add('show');
      reconnectTimer = setTimeout(() => tryReconnect(), 1000);
    }
  };
  ws.onerror = () => {
    wsConnecting = false;
    hideConnectingState();
    resetJoinButtons();
  };
}

function showConnectingState() {
  const banner = document.getElementById('conn-banner');
  banner.textContent = 'Connecting...';
  banner.style.background = '#2563eb';
  banner.classList.add('show');
}

function hideConnectingState() {
  const banner = document.getElementById('conn-banner');
  banner.style.background = '#dc2626';
  banner.classList.remove('show');
}

function resetJoinButtons() {
  const buttons = [
    { id: 'coup-join-btn', text: 'Join Game' },
    { id: 'poker-join-btn', text: 'Join Game' },
    { id: 'ludo-join-btn', text: 'Join Game' }
  ];
  buttons.forEach(({ id, text }) => {
    const btn = document.getElementById(id);
    if (btn) {
      btn.textContent = text;
      const input = document.getElementById(id.replace('-btn', '-code'));
      if (input) btn.disabled = !input.value.trim();
    }
  });
}

function handleWSMessage(e) {
    const msg = JSON.parse(e.data);
    switch(msg.type) {
      case 'redirect':
        // Player is already in another active game
        if (msg.payload?.roomCode) {
          const banner = document.getElementById('conn-banner');
          banner.textContent = msg.payload.message || 'You are already in an active game. Redirecting...';
          banner.style.background = '#f59e0b';
          banner.classList.add('show');
          roomCode = msg.payload.roomCode;
          if (msg.payload.gameType) currentGameType = msg.payload.gameType;
          sessionStorage.setItem('coup_room', roomCode);
          sessionStorage.setItem('coup_gameType', currentGameType);
          setTimeout(() => {
            banner.classList.remove('show');
            banner.style.background = '#dc2626';
            connectWS();
          }, 2000);
        }
        break;
      case 'waiting':
      case 'players-updated':
        hostId = msg.payload?.hostId;
        gameActive = !!msg.payload?.gameActive;
        if (msg.payload?.gameType) currentGameType = msg.payload.gameType;
        renderLobby(msg.payload?.players || []);
        break;
      case 'game-started':
        gameState = msg.payload?.gameState;
        renderGame();
        break;
      case 'poker-started':
        break;
      case 'poker-state':
        pokerState = msg.payload;
        currentGameType = 'poker';
        isSpectating = false;
        renderPokerGame();
        break;
      case 'poker-spectate':
        pokerState = msg.payload;
        currentGameType = 'poker';
        isSpectating = true;
        renderPokerGame();
        break;
      case 'ludo-started':
        break;
      case 'ludo-state':
        isSpectating = false;
        handleLudoStateUpdate(msg.payload);
        currentGameType = 'ludo';
        break;
      case 'ludo-spectate':
        isSpectating = true;
        handleLudoStateUpdate(msg.payload);
        currentGameType = 'ludo';
        break;
      case 'ludo-colors':
        // Update color picker in lobby to show taken colors
        if (msg.payload) {
          const takenColors = new Set(Object.values(msg.payload));
          document.querySelectorAll('.ludo-color-option').forEach(el => {
            const c = el.dataset.color;
            const takenByOther = takenColors.has(c) && msg.payload[playerId] !== c;
            el.classList.toggle('taken', takenByOther);
          });
        }
        break;
      case 'poker-config':
        if (msg.payload) {
          document.getElementById('lobby-poker-config').style.display = '';
          document.getElementById('lobby-buyin').textContent = msg.payload.buyIn;
          document.getElementById('lobby-blinds').textContent = msg.payload.smallBlind + '/' + (msg.payload.smallBlind * 2);
        }
        break;
      case 'state':
        if (!msg.payload) {
          console.log('[STATE] Received null state, returning to lobby');
          gameState = null;
          pokerState = null;
          ludoState = null;
          isSpectating = false;
          document.getElementById('game-active').style.display = 'none';
          document.getElementById('poker-active').style.display = 'none';
          document.getElementById('ludo-active').style.display = 'none';
          document.getElementById('name-entry').style.display = 'none';
          document.getElementById('lobby').style.display = '';
          document.getElementById('lobby-code').textContent = roomCode;
          switchTab('game');
          break;
        }
        gameState = msg.payload;
        isSpectating = false;
        renderGame();
        break;
      case 'spectate-state':
        if (!isSpectating) break;
        gameState = msg.payload;
        document.getElementById('lobby').style.display = 'none';
        document.getElementById('name-entry').style.display = 'none';
        document.getElementById('game-active').style.display = '';
        renderGame();
        break;
      case 'kicked':
        alert('You were kicked');
        disconnect();
        showScreen('menu');
        break;
      case 'chat':
        appendChatMessage(msg.payload);
        break;
      case 'voice-join':
      case 'voice-leave':
      case 'voice-data':
        if (typeof handleVoiceMessage === 'function') handleVoiceMessage(msg);
        break;
      case 'error':
        const errMsg = msg.payload?.message || 'Error';
        if (errMsg.includes('Incorrect Game Code') || errMsg.includes('No Session Found')) {
          document.getElementById('conn-banner').textContent = 'Game session ended.';
          document.getElementById('conn-banner').classList.add('show');
          setTimeout(() => { disconnect(); showScreen('menu'); backToGameList(); }, 1500);
        } else if (gameState || pokerState) { alert(errMsg); } else { document.getElementById('name-error').textContent = errMsg; }
        break;
    }
}

function tryReconnect() {
  if (!roomCode || !joinedName || intentionalDisconnect) return;
  reconnectAttempts++;
  if (reconnectAttempts > MAX_RECONNECT_ATTEMPTS) {
    document.getElementById('conn-banner').textContent = 'Disconnected from game.';
    setTimeout(() => {
      document.getElementById('conn-banner').classList.remove('show');
      disconnect();
      showScreen('menu');
      backToGameList();
    }, 1500);
    return;
  }
  console.log('[WS] Attempting reconnect to room ' + roomCode + ' (attempt ' + reconnectAttempts + '/' + MAX_RECONNECT_ATTEMPTS + ')');
  document.getElementById('conn-banner').textContent = 'Connection lost — reconnecting (' + reconnectAttempts + '/' + MAX_RECONNECT_ATTEMPTS + ')...';
  const params = new URLSearchParams({ room: roomCode, playerId, variant, gameType: currentGameType });
  ws = new WebSocket(WS + SERVER + '/ws?' + params);
  ws.onopen = () => {
    console.log('[WS] Reconnected');
    reconnectAttempts = 0;
    document.getElementById('conn-banner').classList.remove('show');
    showScreen('game');
    send('join', { playerName: joinedName });
  };
  ws.onmessage = handleWSMessage;
  ws.onclose = () => {
    if (!intentionalDisconnect && roomCode && joinedName) {
      const delay = Math.min(2000 * reconnectAttempts, 10000);
      console.log('[WS] Reconnect failed, retrying in ' + delay + 'ms...');
      reconnectTimer = setTimeout(() => tryReconnect(), delay);
    }
  };
}

document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible' && roomCode && joinedName && !intentionalDisconnect) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.log('[WS] Page visible, reconnecting...');
      if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
      reconnectAttempts = 0;
      tryReconnect();
    }
  }
});

function disconnect() {
  intentionalDisconnect = true;
  reconnectAttempts = 0;
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
  document.getElementById('conn-banner').classList.remove('show');
  sessionStorage.removeItem('coup_room');
  sessionStorage.removeItem('coup_joinedName');
  sessionStorage.removeItem('coup_variant');
  sessionStorage.removeItem('coup_gameType');
  if (typeof voiceCleanup === 'function') voiceCleanup();
  if (ws) { ws.close(); ws = null; }
  gameState = null;
  pokerState = null;
  ludoState = null;
  isSpectating = false;
  chatMessages = [];
}

function exitGame() {
  if (isSpectating) {
    stopSpectating();
    return;
  }
  if (currentGameType === 'poker') {
    if (pokerState && pokerState.phase !== 'game_over' && pokerState.phase !== 'showdown') {
      if (!confirm('Exit poker? You will forfeit all your chips.')) return;
      send('exit-game');
    }
  } else if (currentGameType === 'ludo') {
    if (ludoState && ludoState.phase !== 'finished') {
      if (!confirm('Exit Ludo? Your tokens will be removed from the game.')) return;
      send('exit-game');
    }
  } else {
    if (gameState && gameState.phase !== 'game_over') {
      if (!confirm('Exit game? Both your cards will be discarded and you will return to the lobby.')) return;
      send('exit-game');
    }
  }
}

function requestSpectate() {
  isSpectating = true;
  send('spectate');
}

function stopSpectating() {
  isSpectating = false;
  gameState = null;
  pokerState = null;
  ludoState = null;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('lobby').style.display = '';
  document.getElementById('lobby-code').textContent = roomCode;
}

function submitName() {
  const name = document.getElementById('player-name').value.trim();
  if (!name) return;
  const fullName = getFullName(name);
  joinedName = fullName;
  localStorage.setItem('coup_name', name);
  sessionStorage.setItem('coup_joinedName', fullName);
  updateProfileBar();
  send('join', { playerName: fullName });
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('lobby').style.display = '';
  document.getElementById('lobby-code').textContent = roomCode;
}

function send(type, payload) {
  if (ws?.readyState === 1) ws.send(JSON.stringify({ type, payload }));
}

function renderLobby(players) {
  document.getElementById('player-count').textContent = players.length;
  const isHost = hostId === playerId;
  const maxPlayers = currentGameType === 'poker' ? 8 : (currentGameType === 'ludo' ? 4 : 6);
  document.getElementById('lobby-poker-config').style.display = currentGameType === 'poker' ? '' : 'none';
  document.getElementById('lobby-ludo-config').style.display = currentGameType === 'ludo' ? '' : 'none';
  let html = '';
  for (const p of players) {
    html += '<div class="player-list-item"><span>' + esc(p.name) +
      (p.id === hostId ? ' 👑' : '') +
      (p.id === playerId ? ' <span class="badge">You</span>' : '') +
      '</span>' +
      (isHost && p.id !== playerId ? '<button class="btn btn-red btn-sm" style="width:auto;padding:6px 12px" onclick="send(\'kick-player\',{playerId:\'' + p.id + '\'})">Kick</button>' : '') +
      '</div>';
  }
  document.getElementById('lobby-players').innerHTML = html;
  document.getElementById('start-btn').style.display = isHost ? '' : 'none';
  document.getElementById('start-btn').disabled = players.length < 2;
  document.getElementById('lobby-wait').style.display = isHost ? 'none' : '';
  document.getElementById('lobby-spectate').style.display = gameActive ? '' : 'none';
}

function esc(s) { const d=document.createElement('div'); d.textContent=s||''; return d.innerHTML; }

// ========== CHAT ==========
let chatMessages = [];

function sendChat() {
  const isPoker = currentGameType === 'poker' && pokerState;
  const isLudo = currentGameType === 'ludo' && ludoState;
  const input = document.getElementById(isLudo ? 'ludo-chat-input' : (isPoker ? 'poker-chat-input' : 'chat-input'));
  const text = input.value.trim();
  if (!text) return;
  send('chat', { message: text });
  input.value = '';
  input.focus();
}

function appendChatMessage(data) {
  chatMessages.push(data);
  if (chatMessages.length > 200) chatMessages.shift();
  renderChatMessages();

  const isPoker = currentGameType === 'poker' && pokerState;
  const isLudo = currentGameType === 'ludo' && ludoState;
  const chatPanel = document.getElementById(isLudo ? 'ludo-chat-tab' : (isPoker ? 'poker-chat-tab' : 'chat-tab'));
  const chatTab = document.getElementById(isLudo ? 'ltab-chat' : (isPoker ? 'ptab-chat' : 'tab-chat'));
  const notOnChat = !chatPanel || chatPanel.style.display === 'none';
  if (notOnChat && chatTab) {
    chatTab.classList.add('chat-unread');
  }

  if (notOnChat && data.senderId !== playerId) {
    showChatToast(data.senderName, data.message);
  }
}

function renderChatMessages() {
  const panels = ['chat-messages', 'poker-chat-messages', 'ludo-chat-messages'];
  for (const id of panels) {
    const el = document.getElementById(id);
    if (!el) continue;
    let html = '';
    for (const m of chatMessages) {
      const isMe = m.senderId === playerId;
      const time = new Date(m.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += '<div class="chat-msg"><span class="chat-msg-name' + (isMe ? ' me' : '') + '">' + esc(m.senderName) + '</span><span class="chat-msg-text">' + esc(m.message) + '</span><span class="chat-msg-time">' + time + '</span></div>';
    }
    el.innerHTML = html;
    el.scrollTop = el.scrollHeight;
  }
}

let chatToastTimer = null;
function showChatToast(name, message) {
  const toast = document.getElementById('chat-toast');
  if (!toast) return;
  const truncated = message.length > 80 ? message.substring(0, 80) + '...' : message;
  toast.innerHTML = '<span class="chat-toast-name">' + esc(name) + '</span><span class="chat-toast-text">' + esc(truncated) + '</span>';
  toast.classList.add('show');
  if (chatToastTimer) clearTimeout(chatToastTimer);
  chatToastTimer = setTimeout(() => { toast.classList.remove('show'); }, 4000);
}

// ========== PROFILE ==========

function getPlayerTag() {
  let tag = localStorage.getItem('coup_tag');
  if (!tag) {
    tag = String(Math.floor(1000 + Math.random() * 9000));
    localStorage.setItem('coup_tag', tag);
  }
  return tag;
}

function getFullName(name) {
  return name + '#' + getPlayerTag();
}

function updateProfileBar() {
  const name = localStorage.getItem('coup_name');
  const bar = document.getElementById('profile-bar');
  if (name) {
    bar.style.display = '';
    document.getElementById('profile-display-name').textContent = getFullName(name);
  } else {
    bar.style.display = 'none';
  }
}

function showNamePopup() {
  const popup = document.getElementById('name-popup-overlay');
  const input = document.getElementById('popup-name-input');
  input.value = localStorage.getItem('coup_name') || '';
  document.getElementById('popup-tag-display').textContent = '#' + getPlayerTag();
  popup.classList.add('active');
  input.focus();
}

function saveNamePopup() {
  const name = document.getElementById('popup-name-input').value.trim();
  if (!name) return;
  localStorage.setItem('coup_name', name);
  updateProfileBar();
  document.getElementById('name-popup-overlay').classList.remove('active');
}

// ========== INIT ==========
(function() {
  const saved = localStorage.getItem('coup_name');
  if (saved) {
    updateProfileBar();
  } else {
    setTimeout(() => showNamePopup(), 300);
  }
  const savedRoom = sessionStorage.getItem('coup_room');
  const savedJoinedName = sessionStorage.getItem('coup_joinedName');
  if (savedRoom && savedJoinedName) {
    roomCode = savedRoom;
    joinedName = savedJoinedName;
    variant = sessionStorage.getItem('coup_variant') || 'standard';
    currentGameType = sessionStorage.getItem('coup_gameType') || 'coup';
    console.log('[INIT] Rejoining room ' + roomCode + ' as ' + joinedName);
    tryReconnect();
  }
})();
