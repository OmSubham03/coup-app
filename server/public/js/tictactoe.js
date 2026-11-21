let tttState = null;

function canMarkTTT(cell) {
  return tttState?.phase === 'playing' && !isSpectating &&
    tttState.players[tttState.currentPlayerIdx]?.id === playerId && tttState.board[cell] === '';
}

function markTTT(cell) {
  if (canMarkTTT(cell)) send('tictactoe-move', {cell});
}

function handleTTTState(state) {
  tttState = state;
  hideAllActiveGames();
  showScreen('game');
  gameActive = true;
  isSpectating = !state.players.some(p => p.id === playerId);
  document.getElementById('name-entry').style.display = 'none';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('tictactoe-active').style.display = '';
  const finished = state.phase === 'game_over';
  const turn = state.players[state.currentPlayerIdx];
  const winner = state.players.find(p => p.id === state.winnerId);
  document.getElementById('ttt-seats').innerHTML = state.players.map((p,i) =>
    `<div>${playerBadge(p)} <strong>${esc(p.name)}</strong>${p.id === playerId ? ' (You)' : ''} <b>${i ? 'O' : 'X'}</b></div>`).join('');
  document.getElementById('ttt-status').textContent = finished
    ? (winner ? `${winner.name} wins!` : 'Draw!') : (turn.id === playerId ? 'Your turn' : `${turn.name}'s turn`);
  document.getElementById('ttt-board').innerHTML = state.board.map((mark, i) =>
    `<button type="button" data-mark="${mark}" class="${state.winningLine?.includes(i) ? 'ttt-win' : ''}" aria-label="Row ${1+Math.floor(i/3)}, column ${1+i%3}: ${mark || 'empty'}" ${canMarkTTT(i) ? '' : 'disabled'} onclick="markTTT(${i})">${mark}</button>`).join('');
  document.getElementById('ttt-return').hidden = !finished || isSpectating;
}
