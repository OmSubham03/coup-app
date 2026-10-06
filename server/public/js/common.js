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
let unoRoomStackingEnabled = false;
let savedSessionReconnectPending = false;
let roomIsPublic = false;
let publicJoinPending = false;
let publicRoomTimer = null;
let publicRoomRequest = null;
const GAME_CAPACITIES = { coup: 6, poker: 8, ludo: 4, nquestions: 10, commune: 10, twentynine: 4, hearts: 4, uno: 6 };

function stopPublicRoomDiscovery() {
  if (publicRoomTimer) clearInterval(publicRoomTimer);
  publicRoomTimer = null;
  if (publicRoomRequest) publicRoomRequest.abort();
  publicRoomRequest = null;
}

function startPublicRoomDiscovery(type) {
  stopPublicRoomDiscovery();
  const menu = document.getElementById('menu-' + type);
  if (!menu) return;
  let section = menu.querySelector('.available-rooms');
  if (!section) {
    section = document.createElement('section');
    section.className = 'available-rooms';
    const heading = document.createElement('h2');
    heading.className = 'section-title';
    heading.textContent = 'Available Rooms';
    const list = document.createElement('div');
    list.className = 'available-rooms-list';
    list.setAttribute('aria-live', 'polite');
    section.append(heading, list);
    menu.append(section);
  }
  refreshPublicRooms(type, section.querySelector('.available-rooms-list'));
  publicRoomTimer = setInterval(() => refreshPublicRooms(type, section.querySelector('.available-rooms-list')), 5000);
}

async function refreshPublicRooms(type, list) {
  if (publicRoomRequest || type !== currentGameType) return;
  const request = new AbortController();
  publicRoomRequest = request;
  if (!list.children.length) list.textContent = 'Loading rooms...';
  try {
    const response = await fetch(HTTP + SERVER + '/api/public-rooms?' + new URLSearchParams({ gameType: type }), { cache: 'no-store', signal: request.signal });
    if (!response.ok) throw new Error('Cannot load rooms');
    const rooms = await response.json();
    if (request.signal.aborted || type !== currentGameType) return;
    const signature = JSON.stringify(rooms);
    if (list.dataset.rooms === signature) return;
    list.dataset.rooms = signature;
    list.replaceChildren();
    if (!rooms.length) {
      const empty = document.createElement('p');
      empty.className = 'available-rooms-status';
      empty.textContent = 'No public rooms available';
      list.append(empty);
    }
    for (const room of rooms.slice(0, 5)) {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'available-room';
      const name = document.createElement('span');
      name.className = 'available-room-name';
      name.textContent = room.code;
      if (type === 'coup') {
        const variantLabel = document.createElement('span');
        variantLabel.className = 'available-room-variant';
        variantLabel.textContent = room.variant === 'inquisitor' ? 'Inquisitor' : 'Standard';
        name.append(variantLabel);
      }
      const count = document.createElement('span');
      count.className = 'available-room-count';
      count.textContent = room.members + '/' + room.capacity + ' players';
      button.append(name, count);
      button.addEventListener('click', () => {
        if (type !== currentGameType || wsConnecting) return;
        roomCode = room.code;
        if (type === 'coup') variant = room.variant;
        connectWS(null, null, null, null, true);
      });
      list.append(button);
    }
  } catch (error) {
    if (!request.signal.aborted && type === currentGameType) {
      delete list.dataset.rooms;
      list.textContent = 'Unable to load rooms. Retrying...';
    }
  } finally {
    if (publicRoomRequest === request) publicRoomRequest = null;
  }
}

function updateRoomVisibilityControl() {
  const toggle = document.getElementById('room-public-toggle');
  toggle.checked = roomIsPublic;
  toggle.disabled = hostId !== playerId || gameActive;
}

function setRoomVisibility(publicRoom) {
  document.getElementById('room-public-toggle').disabled = true;
  send('set-room-visibility', { public: publicRoom });
}

function openGameMenu(type) {
  currentGameType = type;
  document.getElementById('game-list-view').style.display = 'none';
  document.getElementById('menu-coup').style.display = type === 'coup' ? '' : 'none';
  document.getElementById('menu-poker').style.display = type === 'poker' ? '' : 'none';
  document.getElementById('menu-ludo').style.display = type === 'ludo' ? '' : 'none';
  document.getElementById('menu-nquestions').style.display = type === 'nquestions' ? '' : 'none';
  document.getElementById('menu-commune').style.display = type === 'commune' ? '' : 'none';
  document.getElementById('menu-twentynine').style.display = type === 'twentynine' ? '' : 'none';
  document.getElementById('menu-hearts').style.display = type === 'hearts' ? '' : 'none';
  document.getElementById('menu-uno').style.display = type === 'uno' ? '' : 'none';
  document.getElementById('join-variant-row').style.display = type === 'coup' ? '' : 'none';
  const gameNames = { coup: 'COUP', poker: 'POKER', ludo: 'LUDO', nquestions: '20 QUESTIONS', commune: 'COMMUNE', twentynine: '29', hearts: 'HEARTS', uno: 'UNO' };
  const gameClasses = { coup: 'coup-title', poker: 'poker-title', ludo: 'ludo-title', nquestions: 'nq-title', commune: '', twentynine: '', hearts: '', uno: 'uno-title' };
  const gameSubs = { coup: 'Bluff. Deceive. Survive.', poker: 'Texas Hold\u2019em. All In.', ludo: 'Roll. Race. Win.', nquestions: 'Correct Guess in 20 turns', commune: 'Bluff poker hands. Call the liar.', twentynine: 'Trick-taking trump card game.', hearts: 'Avoid penalty cards. Shoot the moon.', uno: 'Match colors and numbers. Play your hand first.' };
  ['join-game-title', 'entry-game-title', 'lobby-game-title'].forEach(id => {
    const el = document.getElementById(id);
    if (el) { el.textContent = gameNames[type] || type.toUpperCase(); el.className = (gameClasses[type] || '') + ' '; el.style.fontSize = '36px'; }
  });
  ['join-game-subtitle', 'entry-game-subtitle', 'lobby-game-subtitle'].forEach(id => {
    const el = document.getElementById(id);
    if (el) { el.textContent = gameSubs[type] || ''; }
  });
  startPublicRoomDiscovery(type);
}

function backToGameList() {
  stopPublicRoomDiscovery();
  document.getElementById('game-list-view').style.display = '';
  document.getElementById('menu-coup').style.display = 'none';
  document.getElementById('menu-poker').style.display = 'none';
  document.getElementById('menu-ludo').style.display = 'none';
  document.getElementById('menu-nquestions').style.display = 'none';
  document.getElementById('menu-commune').style.display = 'none';
  document.getElementById('menu-twentynine').style.display = 'none';
  document.getElementById('menu-hearts').style.display = 'none';
  document.getElementById('menu-uno').style.display = 'none';
}

function showScreen(name) {
  if (name !== 'menu') stopPublicRoomDiscovery();
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById('screen-' + name).classList.add('active');
  if (name === 'menu' && document.getElementById('game-list-view').style.display === 'none') {
    startPublicRoomDiscovery(currentGameType);
  }
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
  const bbEnabled = document.getElementById('poker-bb-toggle').classList.contains('active');
  if (buyIn < 100) { alert('Buy-in must be at least 100'); return; }
  if (sb < 1) { alert('Small blind must be at least 1'); return; }
  if (bbEnabled && sb * 2 > buyIn) { alert('Big blind (2x small blind) cannot exceed buy-in'); return; }
  currentGameType = 'poker';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create', { buyIn, smallBlind: sb, bigBlindEnabled: bbEnabled });
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

let nqNumQuestions = 20;
function nqAdjustN(delta) {
  nqNumQuestions = Math.max(5, Math.min(50, nqNumQuestions + delta));
  document.getElementById('nq-n-display').textContent = nqNumQuestions;
}

async function createNQGame() {
  currentGameType = 'nquestions';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create', null, { maxQuestions: nqNumQuestions });
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinNQWithCode() {
  const code = document.getElementById('nq-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('nq-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'nquestions';
  roomCode = code;
  connectWS();
}

async function createCommuneGame() {
  currentGameType = 'commune';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create');
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinCommuneWithCode() {
  const code = document.getElementById('commune-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('commune-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'commune';
  roomCode = code;
  connectWS();
}

async function createTNGame() {
  currentGameType = 'twentynine';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create');
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinTNWithCode() {
  const code = document.getElementById('tn-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('tn-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'twentynine';
  roomCode = code;
  connectWS();
}

function toggleTNRules() {
  const el = document.getElementById('tn-rules-overlay');
  el.style.display = el.style.display === 'block' ? 'none' : 'block';
}

async function createHTGame() {
  currentGameType = 'hearts';
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create');
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinHTWithCode() {
  const code = document.getElementById('ht-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('ht-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'hearts';
  roomCode = code;
  connectWS();
}

function toggleHTRules() {
  const el = document.getElementById('ht-rules-overlay');
  el.style.display = el.style.display === 'block' ? 'none' : 'block';
}

async function createUNOGame() {
  currentGameType = 'uno';
  unoRoomStackingEnabled = document.getElementById('uno-stacking-enabled').checked;
  try {
    const res = await fetch(HTTP + SERVER + '/api/generate-code');
    const data = await res.json();
    roomCode = data.code;
    connectWS('create', null, null, unoRoomStackingEnabled);
  } catch(e) { alert('Cannot connect to server: ' + e.message); }
}

function joinUNOWithCode() {
  const code = document.getElementById('uno-join-code').value.trim().toUpperCase();
  if (!code) return;
  const btn = document.getElementById('uno-join-btn');
  btn.disabled = true;
  btn.textContent = 'Joining...';
  currentGameType = 'uno';
  roomCode = code;
  connectWS();
}

function toggleUNORules() {
  const el = document.getElementById('uno-rules-overlay');
  el.style.display = el.style.display === 'block' ? 'none' : 'block';
}

let intentionalDisconnect = false;
let reconnectTimer = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 15;
let wsConnecting = false;

function connectWS(action, pokerConfig, nqConfig, unoStacking, publicRoom = false) {
  stopPublicRoomDiscovery();
  publicJoinPending = publicRoom;
  roomIsPublic = false;
  intentionalDisconnect = false;
  wsConnecting = true;
  showConnectingState();
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
  if (ws) ws.close();
  sessionStorage.setItem('coup_room', roomCode);
  sessionStorage.setItem('coup_variant', variant);
  sessionStorage.setItem('coup_gameType', currentGameType);
  const params = new URLSearchParams({ room: roomCode, playerId, variant, gameType: currentGameType });
  if (action === 'create' && currentGameType === 'uno') params.set('unoStacking', unoStacking ? 'true' : 'false');
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
    if (action === 'create' && currentGameType === 'nquestions' && nqConfig) {
      setTimeout(() => send('set-nq-config', nqConfig), 100);
    }
    const savedName = localStorage.getItem('coup_name');
    if (savedName) {
      document.getElementById('name-entry').style.display = 'none';
      document.getElementById('lobby').style.display = '';
      document.getElementById('lobby-code').textContent = roomCode;
      joinedName = getFullName(savedName);
      sessionStorage.setItem('coup_joinedName', joinedName);
      send('join', { playerName: joinedName, publicRoom: publicJoinPending });
    } else {
      document.getElementById('name-entry').style.display = '';
      document.getElementById('lobby').style.display = 'none';
    }
    document.getElementById('game-active').style.display = 'none';
    document.getElementById('poker-active').style.display = 'none';
    document.getElementById('ludo-active').style.display = 'none';
    document.getElementById('commune-active').style.display = 'none';
    document.getElementById('tn-active').style.display = 'none';
    document.getElementById('ht-active').style.display = 'none';
    document.getElementById('uno-active').style.display = 'none';
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
    { id: 'ludo-join-btn', text: 'Join Game' },
    { id: 'nq-join-btn', text: 'Join Game' },
    { id: 'commune-join-btn', text: 'Join Game' },
    { id: 'tn-join-btn', text: 'Join Game' },
    { id: 'ht-join-btn', text: 'Join Game' },
    { id: 'uno-join-btn', text: 'Join Game' }
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
  try {
    const msg = JSON.parse(e.data);
    switch(msg.type) {
      case 'room-left':
        finishGameExit();
        break;
      case 'room-visibility':
        roomIsPublic = !!msg.payload?.public;
        hostId = msg.payload?.hostId;
        updateRoomVisibilityControl();
        break;
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
        unoRoomStackingEnabled = !!msg.payload?.unoStackingEnabled;
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
      case 'nq-started':
        break;
      case 'nq-state':
        isSpectating = false;
        currentGameType = 'nquestions';
        handleNQStateUpdate(msg.payload);
        break;
      case 'nq-spectate':
        isSpectating = true;
        currentGameType = 'nquestions';
        handleNQStateUpdate(msg.payload);
        break;
      case 'commune-started':
        break;
      case 'commune-state':
        isSpectating = false;
        currentGameType = 'commune';
        handleCommuneStateUpdate(msg.payload);
        break;
      case 'commune-spectate':
        isSpectating = true;
        currentGameType = 'commune';
        handleCommuneStateUpdate(msg.payload);
        break;
      case 'tn-started':
        break;
      case 'tn-state':
        isSpectating = false;
        currentGameType = 'twentynine';
        handleTNStateUpdate(msg.payload);
        break;
      case 'tn-spectate':
        isSpectating = true;
        currentGameType = 'twentynine';
        handleTNStateUpdate(msg.payload);
        break;
      case 'ht-started':
        break;
      case 'ht-state':
        isSpectating = false;
        currentGameType = 'hearts';
        handleHTStateUpdate(msg.payload);
        break;
      case 'ht-spectate':
        isSpectating = true;
        currentGameType = 'hearts';
        handleHTStateUpdate(msg.payload);
        break;
      case 'uno-started':
        break;
      case 'uno-state':
        isSpectating = false;
        currentGameType = 'uno';
        handleUNOStateUpdate(msg.payload);
        break;
      case 'uno-spectate':
        isSpectating = true;
        currentGameType = 'uno';
        handleUNOStateUpdate(msg.payload);
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
          nqState = null;
          communeState = null;
          tnState = null;
          htState = null;
          unoState = null;
          isSpectating = false;
          document.getElementById('game-active').style.display = 'none';
          document.getElementById('poker-active').style.display = 'none';
          document.getElementById('ludo-active').style.display = 'none';
          document.getElementById('nq-active').style.display = 'none';
          document.getElementById('commune-active').style.display = 'none';
          document.getElementById('tn-active').style.display = 'none';
          document.getElementById('ht-active').style.display = 'none';
          document.getElementById('uno-active').style.display = 'none';
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
        if (errMsg === 'This public room is no longer available') {
          const selectedGame = currentGameType;
          disconnect();
          showScreen('menu');
          openGameMenu(selectedGame);
          alert(errMsg);
          break;
        }
        if (errMsg.includes('Incorrect Game Code') || errMsg.includes('No Session Found')) {
          document.getElementById('conn-banner').textContent = 'Game session ended.';
          document.getElementById('conn-banner').classList.add('show');
          setTimeout(() => { disconnect(); showScreen('menu'); backToGameList(); }, 1500);
        } else if (gameState || pokerState || unoState) { alert(errMsg); } else { document.getElementById('name-error').textContent = errMsg; }
        break;
    }
  } catch(err) {
    console.error('[WS] Error handling message:', err);
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

function resumeSavedSessionReconnect() {
  if (!savedSessionReconnectPending) return;
  savedSessionReconnectPending = false;
  tryReconnect();
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
  closeGameSettings();
  stopPublicRoomDiscovery();
  publicJoinPending = false;
  roomIsPublic = false;
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
  nqState = null;
  communeState = null;
  tnState = null;
  htState = null;
  unoState = null;
  isSpectating = false;
  chatMessages = [];
}

function openGameSettings() {
  document.getElementById('game-settings-room-code').textContent = roomCode || 'Unavailable';
  document.getElementById('game-settings-status').textContent = '';
  updateVoiceUI();
  const dialog = document.getElementById('game-settings-dialog');
  if (!dialog.open) dialog.showModal();
}

function closeGameSettings() {
  document.getElementById('game-settings-dialog').close();
}

async function copySettingsRoomCode() {
  const status = document.getElementById('game-settings-status');
  try {
    await navigator.clipboard.writeText(roomCode);
    status.textContent = 'Room code copied';
  } catch (error) {
    status.textContent = 'Copy unavailable. Room code: ' + roomCode;
  }
}

function openSettingsRules() {
  const rules = { coup: toggleRules, poker: togglePokerRules, ludo: toggleLudoRules,
    nquestions: toggleNQRules, commune: toggleCommuneRules, twentynine: toggleTNRules,
    hearts: toggleHTRules, uno: toggleUNORules };
  closeGameSettings();
  rules[currentGameType]?.();
}

function exitGame() {
  const states = { coup: gameState, poker: pokerState, ludo: ludoState,
    nquestions: nqState, commune: communeState, twentynine: tnState, hearts: htState, uno: unoState };
  const state = states[currentGameType];
  const finished = !state || ['game_over', 'finished'].includes(state.phase);
  const warnings = { poker: 'You will forfeit your chips.', ludo: 'Your tokens will be removed.',
    twentynine: 'Your team will forfeit.', commune: 'You will be eliminated.' };
  if (!isSpectating && !finished && !confirm('Exit game? ' + (warnings[currentGameType] || 'You will forfeit your place in this game.'))) return;
  closeGameSettings();
  if (ws?.readyState === WebSocket.OPEN) {
    send('leave-room');
  } else {
    finishGameExit();
  }
}

function finishGameExit() {
  disconnect();
  document.querySelectorAll('.rules-overlay').forEach(overlay => {
    overlay.classList.remove('active');
    overlay.style.display = '';
  });
  roomCode = '';
  joinedName = '';
  gameActive = false;
  hostId = null;
  backToGameList();
  showScreen('menu');
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
  nqState = null;
  communeState = null;
  tnState = null;
  htState = null;
  unoState = null;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('nq-active').style.display = 'none';
  document.getElementById('commune-active').style.display = 'none';
  document.getElementById('tn-active').style.display = 'none';
  document.getElementById('ht-active').style.display = 'none';
  document.getElementById('uno-active').style.display = 'none';
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
  send('join', { playerName: fullName, publicRoom: publicJoinPending });
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
  const maxPlayers = GAME_CAPACITIES[currentGameType] || 6;
  document.getElementById('player-capacity').textContent = maxPlayers;
  updateRoomVisibilityControl();
  document.getElementById('lobby-poker-config').style.display = currentGameType === 'poker' ? '' : 'none';
  document.getElementById('lobby-ludo-config').style.display = currentGameType === 'ludo' ? '' : 'none';
  document.getElementById('lobby-uno-config').style.display = currentGameType === 'uno' ? '' : 'none';
  document.getElementById('lobby-uno-config').textContent = unoRoomStackingEnabled ? 'Draw-card stacking house rules enabled' : 'Standard rules · no draw-card stacking';
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
  const minPlayers = (currentGameType === 'twentynine' || currentGameType === 'hearts') ? 4 : 2;
  document.getElementById('start-btn').disabled = players.length < minPlayers || ((currentGameType === 'twentynine' || currentGameType === 'hearts') && players.length !== 4);
  document.getElementById('lobby-wait').style.display = isHost ? 'none' : '';
  document.getElementById('lobby-spectate').style.display = gameActive ? '' : 'none';
}

function esc(s) { const d=document.createElement('div'); d.textContent=s||''; return d.innerHTML; }

// ========== CHAT ==========
let chatMessages = [];

function sendChat() {
  const isPoker = currentGameType === 'poker' && pokerState;
  const isLudo = currentGameType === 'ludo' && ludoState;
  const isNQ = currentGameType === 'nquestions' && nqState;
  const isCommune = currentGameType === 'commune' && communeState;
  const isTN = currentGameType === 'twentynine' && tnState;
  const isHT = currentGameType === 'hearts' && htState;
  const isUNO = currentGameType === 'uno' && unoState;
  const input = document.getElementById(isUNO ? 'uno-chat-input' : (isHT ? 'ht-chat-input' : (isTN ? 'tn-chat-input' : (isCommune ? 'commune-chat-input' : (isNQ ? 'nq-chat-input' : (isLudo ? 'ludo-chat-input' : (isPoker ? 'poker-chat-input' : 'chat-input')))))));
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
  const isNQ = currentGameType === 'nquestions' && nqState;
  const isCommune = currentGameType === 'commune' && communeState;
  const isTN = currentGameType === 'twentynine' && tnState;
  const isHT = currentGameType === 'hearts' && htState;
  const isUNO = currentGameType === 'uno' && unoState;
  const chatPanel = document.getElementById(isUNO ? 'uno-chat-tab' : (isHT ? 'ht-chat-tab' : (isTN ? 'tn-chat-tab' : (isCommune ? 'commune-chat-tab' : (isNQ ? 'nq-chat-tab' : (isLudo ? 'ludo-chat-tab' : (isPoker ? 'poker-chat-tab' : 'chat-tab')))))));
  const chatTab = document.getElementById(isUNO ? 'unotab-chat' : (isHT ? 'httab-chat' : (isTN ? 'tntab-chat' : (isCommune ? 'cmtab-chat' : (isNQ ? 'nqtab-chat' : (isLudo ? 'ltab-chat' : (isPoker ? 'ptab-chat' : 'tab-chat')))))));
  const notOnChat = !chatPanel || chatPanel.style.display === 'none';
  if (notOnChat && chatTab) {
    chatTab.classList.add('chat-unread');
  }

  if (notOnChat && data.senderId !== playerId) {
    showChatToast(data.senderName, data.message);
  }
}

function renderChatMessages() {
  const panels = ['chat-messages', 'poker-chat-messages', 'ludo-chat-messages', 'nq-chat-messages', 'commune-chat-messages', 'tn-chat-messages', 'ht-chat-messages', 'uno-chat-messages'];
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
    savedSessionReconnectPending = true;
  }
})();
