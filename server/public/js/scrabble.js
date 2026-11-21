let scrabbleState = null;

const scrabbleView = (() => {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  const bonuses = {DL: 'double letter', TL: 'triple letter', DW: 'double word', TW: 'triple word'};
  const points = [1,3,3,2,1,4,2,4,1,8,5,1,3,1,1,3,10,1,1,1,1,4,4,8,4,10];
  const byId = id => document.getElementById(id);
  const area = () => byId('scrabble-play-area');
  const turn = () => scrabbleState.players[scrabbleState.currentPlayerIdx];
  const me = () => scrabbleState.players.find(p => p.id === playerId);
  const canPlay = () => me()?.active && turn()?.id === playerId && scrabbleState.phase === 'playing';
  const canExchange = () => !draft.length && scrabbleState.bagCount >= 7;
  const canVote = () => me()?.active && turn()?.id !== playerId && scrabbleState.phase === 'voting' &&
    scrabbleState.pending && !scrabbleState.pending.approvals?.includes(playerId);
  const rack = () => me() ? scrabbleState.rack || [] : [];
  const coord = cell => `${alphabet[cell % 15]}${1 + Math.floor(cell / 15)}`;
  const value = face => points[alphabet.indexOf(face)] || 0;
  let snapshot, draft = [], selected = -1, blank = '', exchanging = false, swaps = new Set(), cursor = 112;

  function reset() {
    draft = [];
    selected = -1;
    blank = '';
    exchanging = false;
    swaps.clear();
  }

  function statusText() {
    if (scrabbleState.phase === 'game_over') {
      const players = scrabbleState.players.filter(p => p.active);
      const high = Math.max(...players.map(p => p.score));
      const winners = players.filter(p => p.score === high).map(p => p.name);
      return 'Game over' + (winners.length ? ` · ${winners.join(' & ')} ${winners.length === 1 ? 'wins' : 'tie'} with ${high} points` : '');
    }
    if (scrabbleState.phase === 'voting') return `Review ${turn()?.name || 'player'}’s proposal`;
    return canPlay() ? 'Your turn' : `${turn()?.name || 'Player'}’s turn`;
  }

  function boardHTML() {
    const proposed = scrabbleState.phase === 'voting' ? scrabbleState.pending?.placements : [];
    const overlay = new Map((proposed || []).map(p => [p.cell, {...p, kind: 'proposed'}]));
    for (const p of draft) overlay.set(p.cell, {...p, kind: 'draft'});
    return scrabbleState.board.map((raw, i) => {
      const p = overlay.get(i);
      const letter = p?.letter || raw || '';
      const face = /^[a-z]$/i.test(letter) ? letter.toUpperCase() : '';
      const isBlank = p ? p.blank : /^[a-z]$/.test(raw);
      const code = scrabbleState.premiums[i];
      const bonus = Object.hasOwn(bonuses, code) ? code : '';
      const description = face ? `${isBlank ? 'blank as ' : ''}${face}, ${value(isBlank ? '?' : face)} points` : 'empty';
      const label = `${coord(i)}, ${description}${i === 112 ? ', center star' : ''}${bonus ? ', ' + bonuses[bonus] : ''}${p ? ', '+p.kind : ''}`;
      return `<button type="button" class="sc-cell ${face ? 'sc-face' : 'sc-'+bonus} ${p ? 'sc-'+p.kind : ''}"
        data-cell="${i}" tabindex="${i === cursor ? 0 : -1}" aria-label="${label}">
        ${face ? `${esc(face)}<small>${value(isBlank ? '?' : face)}</small>` : i === 112 ? '★' : bonus}</button>`;
    }).join('');
  }

  function action(name, text, enabled = true) {
    return `<button type="button" data-action="${name}" ${enabled ? '' : 'disabled'}>${text}</button>`;
  }

  function rackHTML() {
    if (!me()) return '';
    const tiles = rack().map((face, i) => {
      const used = draft.some(p => p.slot === i);
      const chosen = exchanging ? swaps.has(i) : selected === i;
      return `<button type="button" class="sc-tile" data-slot="${i}" aria-pressed="${chosen}"
        aria-label="Tile ${i+1}: ${/^[A-Z]$/.test(face) ? face : 'blank'}, ${value(face)} points${used ? ', on board' : ''}"
        ${used || !canPlay() ? 'disabled' : ''}>${esc(face)}<small>${value(face)}</small></button>`;
    }).join('');
    const chooseBlank = selected >= 0 && rack()[selected] === '?';
    return `<section aria-label="Your private rack"><div class="sc-rack">${tiles || 'Empty rack'}</div>
      ${chooseBlank ? `<label>Blank letter <select data-blank aria-label="Blank letter"><option value="">Choose A–Z</option>${[...alphabet].map(c =>
        `<option ${c === blank ? 'selected' : ''}>${c}</option>`).join('')}</select></label>` : ''}</section>`;
  }

  function controlsHTML() {
    if (scrabbleState.phase === 'game_over') return me() ? action('return','Return to lobby') : '';
    if (!canPlay()) return '';
    if (exchanging) return `<div class="sc-actions">Pick tiles
      ${action('exchange', `Swap (${swaps.size})`, swaps.size > 0 && canExchange())}
      ${action('cancel','Cancel')}</div>`;
    return `<div class="sc-actions">
      ${action('undo','Undo', draft.length > 0)}
      ${action('submit','Submit', draft.length > 0)}
      ${action('pass','Pass', !draft.length)}
      ${action('swap','Swap', canExchange())}</div>`;
  }

  function proposalHTML() {
    const p = scrabbleState.pending;
    if (scrabbleState.phase !== 'voting' || !p) return '';
    const approvals = scrabbleState.players.filter(seat => p.approvals?.includes(seat.id));
    return `<section class="sc-proposal" aria-label="Word vote">
      <strong>${(p.words || []).map(esc).join(' · ')}</strong>
      <span>${approvals.length}/${scrabbleState.players.filter(seat => seat.active).length} approved</span>
      <span class="home-sr-only">Approved: ${approvals.map(seat => esc(seat.name)).join(', ') || 'No votes yet'}</span>
      ${canVote() ? `<div class="sc-actions">${action('approve','Approve')}${action('reject','Reject')}</div>` : ''}</section>`;
  }

  function focusKey(el) {
    if (!el || !area().contains(el)) return '';
    for (const key of ['cell','slot','action','blank']) {
      if (el.dataset[key] !== undefined) return `[data-${key}="${el.dataset[key]}"]`;
    }
    return '';
  }

  function render(focus = focusKey(document.activeElement)) {
    const root = area();
    const scroll = root.querySelector('.sc-scroll')?.scrollLeft || 0;
    const status = byId('scrabble-status');
    status.setAttribute('role', 'status');
    if (status.textContent !== statusText()) status.textContent = statusText();
    root.innerHTML = `<div class="sc-strip"><ul class="sc-scores" aria-label="Scores">${scrabbleState.players.map((p,i) =>
      `<li class="${p.active && scrabbleState.phase !== 'game_over' && i === scrabbleState.currentPlayerIdx ? 'sc-current' : ''}">
      ${playerBadge(p)}<strong class="sc-name">${p.id === playerId ? 'You' : esc(p.name)}</strong><b>${Number(p.score)}<span class="home-sr-only"> points</span></b>${!p.active ? ' · out' : ''}</li>`).join('')}</ul></div>
      ${proposalHTML()}
      ${scrabbleState.lastAction?.endsWith(' rejected the proposal') ? '<small>Word rejected · try again</small>' : ''}
      <section aria-label="Game board"><div class="sc-scroll"><div class="sc-board" role="group" aria-label="15 by 15 board">${boardHTML()}</div></div></section>
      ${rackHTML()}<div class="sc-toolbar">${controlsHTML()}
      <span class="sc-bag" title="Tiles left in bag"><svg aria-hidden="true" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M9 8 7 3h10l-2 5Zm0 1c-3 3-6 6-6 9 0 3 3 3 9 3s9 0 9-3c0-3-3-6-6-9Z"/></svg>${Number(scrabbleState.bagCount)}<span class="home-sr-only"> tiles left</span></span></div>`;
    root.querySelector('.sc-scroll').scrollLeft = scroll;
    if (focus) {
      const target = root.querySelector(focus);
      (target && !target.disabled ? target : root.querySelector('[data-cell][tabindex="0"]'))?.focus({preventScroll:true});
    }
    root.onclick = click;
    root.onkeydown = keydown;
    root.onchange = event => {
      if (canPlay() && rack()[selected] === '?' && event.target.matches('[data-blank]')) {
        blank = /^[A-Z]$/.test(event.target.value) ? event.target.value : '';
      }
    };
  }

  function click(event) {
    const button = event.target.closest('button');
    if (!button || button.disabled || !area().contains(button)) return;
    const {slot, cell, action: name} = button.dataset;
    if (name === 'return' && me() && scrabbleState.phase === 'game_over') return send('return-to-lobby', {});
    if ((name === 'approve' || name === 'reject') && canVote()) {
      return send('scrabble-vote', {id:scrabbleState.pending.id, approve:name === 'approve'});
    }
    if (!canPlay()) return;
    if (slot !== undefined) {
      const i = Number(slot);
      if (!rack()[i] || draft.some(p => p.slot === i)) return;
      if (exchanging) {
        if (swaps.has(i)) swaps.delete(i); else swaps.add(i);
      } else {
        selected = selected === i ? -1 : i;
        blank = '';
      }
    } else if (cell !== undefined && !exchanging) {
      const i = Number(cell);
      if (scrabbleState.board[i] !== '') return;
      cursor = i;
      const index = draft.findIndex(p => p.cell === i);
      if (index >= 0) draft.splice(index, 1);
      else {
        const tile = rack()[selected];
        if (!tile) return;
        if (tile === '?' && !blank) return area().querySelector('[data-blank]').focus();
        draft.push({slot:selected, cell:i, letter:tile === '?' ? blank : tile, blank:tile === '?'});
        selected = -1;
        blank = '';
      }
    } else if (name === 'undo' && !exchanging) draft.pop();
    else if (name === 'submit' && !exchanging && draft.length) {
      return send('scrabble-play', {placements:draft.map(({cell,letter,blank}) => ({cell,letter,blank}))});
    } else if (name === 'pass' && !draft.length && !exchanging) return send('scrabble-pass', {});
    else if (name === 'swap' && canExchange()) {
      reset();
      exchanging = true;
    } else if (name === 'cancel' && exchanging) reset();
    else if (name === 'exchange' && exchanging && swaps.size && canExchange()) {
      return send('scrabble-exchange', {tiles:[...swaps].map(i => rack()[i])});
    } else return;
    render(name === 'swap' ? '[data-slot="0"]' : name === 'cancel' ? '[data-action="swap"]' :
      slot !== undefined && rack()[selected] === '?' ? '[data-blank]' : focusKey(button));
  }

  function keydown(event) {
    if (event.target.dataset.cell === undefined) return;
    const i = Number(event.target.dataset.cell);
    const col = i % 15;
    const next = {ArrowLeft: i - (col > 0), ArrowRight: i + (col < 14),
      ArrowUp: i >= 15 ? i - 15 : i, ArrowDown: i < 210 ? i + 15 : i,
      Home: i - col, End: i + 14 - col}[event.key];
    if (next === undefined) return;
    event.preventDefault();
    area().querySelector('[data-cell][tabindex="0"]').tabIndex = -1;
    cursor = next;
    const target = area().querySelector(`[data-cell="${next}"]`);
    target.tabIndex = 0;
    target.focus();
  }

  return function update(state) {
    const focus = focusKey(document.activeElement);
    const local = state.players.find(p => p.id === playerId);
    const key = JSON.stringify([state.id, state.phase, state.currentPlayerIdx, state.players[state.currentPlayerIdx]?.id,
      state.board, playerId, local?.active, local ? state.rack : null]);
    if (!scrabbleState || snapshot !== key) reset();
    if (!scrabbleState || state.id !== scrabbleState.id) cursor = 112;
    snapshot = key;
    scrabbleState = state;
    hideAllActiveGames();
    showScreen('game');
    gameActive = true;
    isSpectating = !local;
    byId('name-entry').style.display = 'none';
    byId('lobby').style.display = 'none';
    byId('scrabble-active').style.display = '';
    render(focus);
  };
})();

function handleScrabbleState(state) {
  scrabbleView(state);
}
