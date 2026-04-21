// ========== 20 QUESTIONS GAME ==========

let nqState = null;

function handleNQStateUpdate(payload) {
  nqState = payload;
  renderNQGame();
}

function renderNQGame() {
  if (!nqState) return;
  document.getElementById('game-active').style.display = 'none';
  document.getElementById('poker-active').style.display = 'none';
  document.getElementById('ludo-active').style.display = 'none';
  document.getElementById('nq-active').style.display = '';
  document.getElementById('lobby').style.display = 'none';
  document.getElementById('name-entry').style.display = 'none';

  const me = nqState.players.find(p => p.id === playerId);
  const isGiver = me && me.isGiver;
  const giver = nqState.players.find(p => p.isGiver);
  const giverName = giver ? giver.name : '???';

  // Phase display
  const phaseEl = document.getElementById('nq-phase-display');
  const turnEl = document.getElementById('nq-turn-display');
  const tableArea = document.getElementById('nq-table-area');
  const actionArea = document.getElementById('nq-action-area');

  switch (nqState.phase) {
    case 'setup':
      phaseEl.textContent = 'Setting up...';
      turnEl.textContent = '';
      renderNQSetup(tableArea, actionArea, isGiver, giverName);
      break;
    case 'asking':
      phaseEl.textContent = `Question ${nqState.currentQuestion}/${nqState.maxQuestions}`;
      turnEl.textContent = `Category: ${nqState.category.toUpperCase()}`;
      renderNQAsking(tableArea, actionArea, isGiver, me);
      break;
    case 'final_guess':
      phaseEl.textContent = 'Final Guess!';
      turnEl.textContent = `Category: ${nqState.category.toUpperCase()}`;
      renderNQFinalGuess(tableArea, actionArea, isGiver, me);
      break;
    case 'finished':
      phaseEl.textContent = 'Game Over';
      turnEl.textContent = '';
      renderNQFinished(tableArea, actionArea, isGiver);
      break;
  }

  // Render log
  const logPanel = document.getElementById('nq-log-panel');
  if (logPanel && nqState.log) {
    let html = '';
    for (const entry of nqState.log) {
      const time = new Date(entry.timestamp).toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'});
      html += `<div class="nq-log-entry"><span class="nq-log-time">${time}</span> ${esc(entry.message)}</div>`;
    }
    logPanel.innerHTML = html;
    logPanel.scrollTop = logPanel.scrollHeight;
  }
}

function renderNQSetup(tableArea, actionArea, isGiver, giverName) {
  if (isGiver) {
    tableArea.innerHTML = `
      <div class="nq-setup-card">
        <div class="nq-setup-icon">🤫</div>
        <h3>You are the Word-Giver!</h3>
        <p>Pick a category and enter a secret word for others to guess.</p>
      </div>`;
    actionArea.innerHTML = `
      <div class="nq-setup-form">
        <label class="nq-label">Category</label>
        <div class="nq-cat-picker">
          <button class="nq-cat-btn selected" data-cat="name" onclick="nqSelectCat(this)">👤 Name</button>
          <button class="nq-cat-btn" data-cat="place" onclick="nqSelectCat(this)">📍 Place</button>
          <button class="nq-cat-btn" data-cat="animal" onclick="nqSelectCat(this)">🐾 Animal</button>
          <button class="nq-cat-btn" data-cat="thing" onclick="nqSelectCat(this)">📦 Thing</button>
        </div>
        <label class="nq-label">Secret Word</label>
        <input id="nq-word-input" class="nq-input" placeholder="Enter the word..." maxlength="50" onkeydown="if(event.key==='Enter')nqSubmitWord()">
        <button class="btn nq-btn-submit" onclick="nqSubmitWord()">Start Game</button>
      </div>`;
  } else {
    tableArea.innerHTML = `
      <div class="nq-setup-card">
        <div class="nq-setup-icon">⏳</div>
        <h3>Waiting for ${esc(giverName)}</h3>
        <p>The word-giver is picking a secret word...</p>
        <div class="nq-dots"><span>.</span><span>.</span><span>.</span></div>
      </div>`;
    actionArea.innerHTML = '';
  }
}

let nqSelectedCat = 'name';
function nqSelectCat(el) {
  nqSelectedCat = el.dataset.cat;
  document.querySelectorAll('.nq-cat-btn').forEach(b => b.classList.remove('selected'));
  el.classList.add('selected');
}

function nqSubmitWord() {
  const word = document.getElementById('nq-word-input').value.trim();
  if (!word) return;
  send('nq-set-word', { category: nqSelectedCat, word: word });
}

function renderNQAsking(tableArea, actionArea, isGiver, me) {
  const nonGivers = nqState.players.filter(p => !p.isGiver);
  const currentAsker = nonGivers[nqState.currentAskerIdx % nonGivers.length];
  const isMyTurn = currentAsker && currentAsker.id === playerId;

  // Q&A thread
  let qaHtml = '<div class="nq-qa-thread">';
  for (const q of nqState.questions) {
    const isGuessClass = q.isGuess ? ' nq-guess' : '';
    const correctClass = q.correct ? ' nq-correct' : '';
    qaHtml += `<div class="nq-qa-item${isGuessClass}${correctClass}">
      <div class="nq-q"><span class="nq-q-badge">${q.isGuess ? '🎯' : 'Q' + q.turn}</span> <strong>${esc(q.askerName)}</strong>: ${esc(q.question)}</div>
      <div class="nq-a">${esc(q.answer)}</div>
    </div>`;
  }

  // Show pending question
  if (nqState.waitingForAnswer && nqState.pendingQuestion) {
    qaHtml += `<div class="nq-qa-item nq-pending">
      <div class="nq-q"><span class="nq-q-badge">Q${nqState.currentQuestion}</span> ${esc(nqState.pendingQuestion)}</div>
      <div class="nq-a nq-waiting-answer">Waiting for answer...</div>
    </div>`;
  }

  // Show pending guess
  if (nqState.waitingForGuessVerdict && nqState.pendingGuessPlayerName) {
    qaHtml += `<div class="nq-qa-item nq-guess nq-pending">
      <div class="nq-q"><span class="nq-q-badge">🎯</span> <strong>${esc(nqState.pendingGuessPlayerName)}</strong>: I think it's "${esc(nqState.pendingGuessWord)}"</div>
      <div class="nq-a nq-waiting-answer">Waiting for giver to verify...</div>
    </div>`;
  }
  qaHtml += '</div>';

  // Info bar
  const secretWordHtml = isGiver ? `<div class="nq-secret-bar">Your word: <strong>${esc(nqState.secretWord)}</strong></div>` : '';
  const categoryBadge = `<div class="nq-category-badge">${getCategoryEmoji(nqState.category)} ${nqState.category.toUpperCase()}</div>`;

  tableArea.innerHTML = secretWordHtml + categoryBadge + qaHtml;

  // Action area
  if (nqState.waitingForGuessVerdict) {
    if (isGiver) {
      actionArea.innerHTML = `
        <div class="nq-verify-form">
          <p class="nq-prompt">🎯 <strong>${esc(nqState.pendingGuessPlayerName)}</strong> guessed:</p>
          <div class="nq-guess-reveal">"${esc(nqState.pendingGuessWord)}"</div>
          <div class="nq-verify-buttons">
            <button class="btn nq-btn-correct" onclick="nqVerifyGuess(true)">✅ Correct</button>
            <button class="btn nq-btn-wrong" onclick="nqVerifyGuess(false)">❌ Wrong</button>
          </div>
        </div>`;
    } else {
      actionArea.innerHTML = `<div class="nq-wait-msg">🎯 <strong>${esc(nqState.pendingGuessPlayerName)}</strong> made a guess! Waiting for the giver to verify...</div>`;
    }
  } else if (nqState.waitingForAnswer) {
    if (isGiver) {
      actionArea.innerHTML = `
        <div class="nq-answer-form">
          <p class="nq-prompt">Answer the question:</p>
          <div class="nq-quick-answers">
            <button class="btn nq-btn-yes" onclick="nqAnswer('Yes')">Yes</button>
            <button class="btn nq-btn-no" onclick="nqAnswer('No')">No</button>
          </div>
          <div class="nq-custom-answer">
            <input id="nq-answer-input" class="nq-input" placeholder="Or type a short comment..." maxlength="200" onkeydown="if(event.key==='Enter')nqAnswerCustom()">
            <button class="btn nq-btn-comment" onclick="nqAnswerCustom()">Reply</button>
          </div>
        </div>`;
    } else {
      actionArea.innerHTML = `<div class="nq-wait-msg">Waiting for the giver to answer...</div>`;
    }
  } else if (!isGiver) {
    let html = '';
    if (isMyTurn) {
      html += `<div class="nq-ask-form">
        <p class="nq-prompt">Your turn! Ask a question.</p>
        <input id="nq-question-input" class="nq-input" placeholder="Type your question..." maxlength="200" onkeydown="if(event.key==='Enter')nqAskQuestion()">
        <button class="btn nq-btn-ask" onclick="nqAskQuestion()">Ask</button>
      </div>`;
    } else {
      html += `<div class="nq-wait-msg">${esc(currentAsker?.name || '...')}'s turn to ask...</div>`;
    }
    html += `<div class="nq-guess-section">
      <div class="nq-guess-divider">— or make a guess (costs a question!) —</div>
      <div class="nq-guess-row">
        <input id="nq-guess-input" class="nq-input" placeholder="Your guess..." maxlength="50" onkeydown="if(event.key==='Enter')nqMakeGuess()">
        <button class="btn nq-btn-guess" onclick="nqMakeGuess()">🎯 Guess</button>
      </div>
    </div>`;
    actionArea.innerHTML = html;
  } else {
    actionArea.innerHTML = `<div class="nq-wait-msg">Waiting for <strong>${esc(currentAsker?.name || '...')}</strong> to ask...</div>`;
  }

  // Auto-scroll Q&A thread
  const thread = tableArea.querySelector('.nq-qa-thread');
  if (thread) thread.scrollTop = thread.scrollHeight;
}

function renderNQFinalGuess(tableArea, actionArea, isGiver, me) {
  const nonGivers = nqState.players.filter(p => !p.isGiver);
  const currentGuesser = nonGivers[nqState.finalGuessIdx];
  const isMyTurn = currentGuesser && currentGuesser.id === playerId;

  // Show all Q&A first
  let qaHtml = '<div class="nq-qa-thread">';
  for (const q of nqState.questions) {
    const isGuessClass = q.isGuess ? ' nq-guess' : '';
    const correctClass = q.correct ? ' nq-correct' : '';
    qaHtml += `<div class="nq-qa-item${isGuessClass}${correctClass}">
      <div class="nq-q"><span class="nq-q-badge">${q.isGuess ? '🎯' : 'Q' + q.turn}</span> <strong>${esc(q.askerName)}</strong>: ${esc(q.question)}</div>
      <div class="nq-a">${esc(q.answer)}</div>
    </div>`;
  }

  // Show pending guess
  if (nqState.waitingForGuessVerdict && nqState.pendingGuessPlayerName) {
    qaHtml += `<div class="nq-qa-item nq-guess nq-pending">
      <div class="nq-q"><span class="nq-q-badge">🎯</span> <strong>${esc(nqState.pendingGuessPlayerName)}</strong>: I think it's "${esc(nqState.pendingGuessWord)}"</div>
      <div class="nq-a nq-waiting-answer">Waiting for giver to verify...</div>
    </div>`;
  }
  qaHtml += '</div>';

  const secretWordHtml = isGiver ? `<div class="nq-secret-bar">Your word: <strong>${esc(nqState.secretWord)}</strong></div>` : '';
  const categoryBadge = `<div class="nq-category-badge">${getCategoryEmoji(nqState.category)} ${nqState.category.toUpperCase()}</div>`;
  tableArea.innerHTML = secretWordHtml + categoryBadge +
    `<div class="nq-final-banner">🎯 Final Guess Round!</div>` + qaHtml;

  if (nqState.waitingForGuessVerdict) {
    if (isGiver) {
      actionArea.innerHTML = `
        <div class="nq-verify-form">
          <p class="nq-prompt">🎯 <strong>${esc(nqState.pendingGuessPlayerName)}</strong> guessed:</p>
          <div class="nq-guess-reveal">"${esc(nqState.pendingGuessWord)}"</div>
          <div class="nq-verify-buttons">
            <button class="btn nq-btn-correct" onclick="nqVerifyGuess(true)">✅ Correct</button>
            <button class="btn nq-btn-wrong" onclick="nqVerifyGuess(false)">❌ Wrong</button>
          </div>
        </div>`;
    } else {
      actionArea.innerHTML = `<div class="nq-wait-msg">🎯 <strong>${esc(nqState.pendingGuessPlayerName)}</strong> made a guess! Waiting for the giver to verify...</div>`;
    }
  } else if (isMyTurn && !isGiver) {
    actionArea.innerHTML = `
      <div class="nq-ask-form">
        <p class="nq-prompt">Your final guess!</p>
        <input id="nq-final-guess-input" class="nq-input" placeholder="What's the word?" maxlength="50" onkeydown="if(event.key==='Enter')nqFinalGuess()">
        <button class="btn nq-btn-guess" onclick="nqFinalGuess()">Submit Final Guess</button>
      </div>`;
  } else if (isGiver) {
    actionArea.innerHTML = `<div class="nq-wait-msg">Waiting for <strong>${esc(currentGuesser?.name || '...')}</strong> to make their final guess...</div>`;
  } else {
    actionArea.innerHTML = `<div class="nq-wait-msg">${esc(currentGuesser?.name || '...')}'s turn to guess...</div>`;
  }

  const thread = tableArea.querySelector('.nq-qa-thread');
  if (thread) thread.scrollTop = thread.scrollHeight;
}

function renderNQFinished(tableArea, actionArea, isGiver) {
  let qaHtml = '<div class="nq-qa-thread">';
  for (const q of nqState.questions) {
    const isGuessClass = q.isGuess ? ' nq-guess' : '';
    const correctClass = q.correct ? ' nq-correct' : '';
    qaHtml += `<div class="nq-qa-item${isGuessClass}${correctClass}">
      <div class="nq-q"><span class="nq-q-badge">${q.isGuess ? '🎯' : 'Q' + q.turn}</span> <strong>${esc(q.askerName)}</strong>: ${esc(q.question)}</div>
      <div class="nq-a">${esc(q.answer)}</div>
    </div>`;
  }
  qaHtml += '</div>';

  const winClass = nqState.winner === 'guessers' ? 'nq-win-guessers' : 'nq-win-giver';
  let resultHtml;
  if (nqState.winner === 'guessers') {
    const correctGuess = [...nqState.questions].reverse().find(q => q.correct);
    resultHtml = `<div class="nq-result ${winClass}">
      <div class="nq-result-icon">🎉</div>
      <h3>Guessers Win!</h3>
      <p>${esc(nqState.correctGuesser)} guessed the word!</p>
      <div class="nq-reveal-word">Guessed: <strong>${esc(correctGuess?.guessWord || '?')}</strong></div>
      <div class="nq-reveal-word">Word was: <strong>${esc(nqState.secretWord)}</strong></div>
    </div>`;
  } else {
    const giver = nqState.players.find(p => p.isGiver);
    resultHtml = `<div class="nq-result ${winClass}">
      <div class="nq-result-icon">🛡️</div>
      <h3>${esc(giver?.name || 'The Giver')} Wins!</h3>
      <p>Nobody guessed the word!</p>
      <div class="nq-reveal-word">${esc(nqState.secretWord)}</div>
    </div>`;
  }

  const categoryBadge = `<div class="nq-category-badge">${getCategoryEmoji(nqState.category)} ${nqState.category.toUpperCase()}</div>`;
  tableArea.innerHTML = resultHtml + categoryBadge + qaHtml;

  // Find next giver name
  const currentGiverIdx = nqState.players.findIndex(p => p.isGiver);
  const nextGiverIdx = (currentGiverIdx + 1) % nqState.players.length;
  const nextGiverName = nqState.players[nextGiverIdx]?.name || '?';

  const isHost = hostId === playerId;
  actionArea.innerHTML = `
    <button class="btn nq-btn-next-round" onclick="send('nq-next-round')">🔄 Next Round — ${esc(nextGiverName)} gives the word</button>
    ${isHost ? '<button class="btn nq-btn-lobby" style="margin-top:8px" onclick="send(\'return-to-lobby\')">Return to Lobby</button>' : ''}
  `;
}

function getCategoryEmoji(cat) {
  const emojis = { name: '👤', place: '📍', animal: '🐾', thing: '📦' };
  return emojis[cat] || '❓';
}

function nqAnswer(answer) {
  send('nq-answer', { answer });
}
function nqAnswerCustom() {
  const input = document.getElementById('nq-answer-input');
  const answer = input.value.trim();
  if (!answer) return;
  send('nq-answer', { answer });
}
function nqAskQuestion() {
  const input = document.getElementById('nq-question-input');
  const question = input.value.trim();
  if (!question) return;
  send('nq-ask', { question });
}
function nqMakeGuess() {
  const guessInput = document.getElementById('nq-guess-input');
  const guess = guessInput.value.trim();
  if (!guess) return;
  send('nq-guess', { guess });
}
function nqVerifyGuess(correct) {
  send('nq-verify-guess', { correct });
}
function nqFinalGuess() {
  const input = document.getElementById('nq-final-guess-input');
  const guess = input.value.trim();
  if (!guess) return;
  send('nq-guess', { guess });
}

function switchNQTab(tab) {
  document.getElementById('nqtab-game').className = 'tab' + (tab === 'game' ? ' active' : '');
  document.getElementById('nqtab-log').className = 'tab' + (tab === 'log' ? ' active' : '');
  document.getElementById('nqtab-chat').className = 'tab' + (tab === 'chat' ? ' active' : '');
  document.getElementById('nq-game-tab').style.display = tab === 'game' ? '' : 'none';
  document.getElementById('nq-log-tab').style.display = tab === 'log' ? '' : 'none';
  document.getElementById('nq-chat-tab').style.display = tab === 'chat' ? '' : 'none';
  if (tab === 'chat') {
    const chatTab = document.getElementById('nqtab-chat');
    if (chatTab) chatTab.classList.remove('chat-unread');
  }
}

function toggleNQRules() {
  const el = document.getElementById('nq-rules-overlay');
  el.style.display = el.style.display === 'flex' ? 'none' : 'flex';
}
