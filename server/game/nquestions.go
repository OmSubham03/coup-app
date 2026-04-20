package game

import (
	"fmt"
	"strings"
	"time"
)

// NQ phases
type NQPhase string

const (
	NQPhaseWaiting    NQPhase = "waiting"
	NQPhaseSetup      NQPhase = "setup"      // Word-giver picks category & word
	NQPhaseAsking     NQPhase = "asking"      // Players take turns asking questions
	NQPhaseFinalGuess NQPhase = "final_guess" // Everyone gets one final guess
	NQPhaseFinished   NQPhase = "finished"
)

type NQCategory string

const (
	NQCatName   NQCategory = "name"
	NQCatPlace  NQCategory = "place"
	NQCatAnimal NQCategory = "animal"
	NQCatThing  NQCategory = "thing"
)

type NQPlayer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsGiver    bool   `json:"isGiver"`
	HasGuessed bool   `json:"hasGuessed"` // Used in final guess phase
}

type NQQuestion struct {
	AskerID    string `json:"askerId"`
	AskerName  string `json:"askerName"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`     // "yes", "no", or a short comment
	IsGuess    bool   `json:"isGuess"`    // Was this a guess attempt?
	GuessWord  string `json:"guessWord"`  // The guessed word (if isGuess)
	Correct    bool   `json:"correct"`    // Was the guess correct?
	Turn       int    `json:"turn"`
	Timestamp  int64  `json:"timestamp"`
}

type NQState struct {
	ID               string       `json:"id"`
	Players          []NQPlayer   `json:"players"`
	Phase            NQPhase      `json:"phase"`
	MaxQuestions      int          `json:"maxQuestions"`
	Category         NQCategory   `json:"category"`
	SecretWord       string       `json:"secretWord"`       // Hidden from guessers in broadcast
	CurrentQuestion  int          `json:"currentQuestion"`  // 1-indexed, which question we're on
	CurrentAskerIdx  int          `json:"currentAskerIdx"`  // Index into non-giver players
	Questions        []NQQuestion `json:"questions"`
	WaitingForAnswer bool         `json:"waitingForAnswer"` // Giver needs to respond
	PendingQuestion  string       `json:"pendingQuestion"`  // The question waiting for answer
	PendingAskerID   string       `json:"pendingAskerId"`
	Winner           string       `json:"winner"`           // "guessers" or "giver" or ""
	CorrectGuesser   string       `json:"correctGuesser"`   // Player name who guessed correctly
	Finished         bool         `json:"finished"`
	FinalGuessIdx    int          `json:"finalGuessIdx"`    // Which non-giver player is guessing
	Log              []NQLogEntry `json:"log"`
	LastAction       string       `json:"lastAction"`
}

type NQLogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func addNQLog(state *NQState, msg string) {
	state.Log = append(state.Log, NQLogEntry{
		Timestamp: time.Now().UnixMilli(),
		Message:   msg,
	})
	state.LastAction = msg
}

func InitializeNQGame(players []struct{ ID, Name string }, maxQuestions int) *NQState {
	nqPlayers := make([]NQPlayer, len(players))
	for i, p := range players {
		nqPlayers[i] = NQPlayer{
			ID:   p.ID,
			Name: p.Name,
		}
	}
	// First player (host) is the giver
	nqPlayers[0].IsGiver = true

	state := &NQState{
		ID:           fmt.Sprintf("nq-%d", time.Now().UnixNano()),
		Players:      nqPlayers,
		Phase:        NQPhaseSetup,
		MaxQuestions:  maxQuestions,
		Questions:    []NQQuestion{},
		Log:          []NQLogEntry{},
	}
	addNQLog(state, fmt.Sprintf("%s is the word-giver! Waiting for them to pick a word...", nqPlayers[0].Name))
	return state
}

// NQSetWord: giver picks category and secret word
func NQSetWord(state *NQState, playerID string, category string, word string) error {
	if state.Phase != NQPhaseSetup {
		return fmt.Errorf("not in setup phase")
	}
	// Only the giver can set the word
	giverIdx := -1
	for i, p := range state.Players {
		if p.IsGiver && p.ID == playerID {
			giverIdx = i
			break
		}
	}
	if giverIdx == -1 {
		return fmt.Errorf("only the word-giver can set the word")
	}

	word = strings.TrimSpace(word)
	if word == "" {
		return fmt.Errorf("word cannot be empty")
	}
	if len(word) > 50 {
		return fmt.Errorf("word is too long (max 50 characters)")
	}

	validCats := map[string]bool{"name": true, "place": true, "animal": true, "thing": true}
	if !validCats[category] {
		return fmt.Errorf("invalid category")
	}

	state.Category = NQCategory(category)
	state.SecretWord = strings.ToLower(word)
	state.Phase = NQPhaseAsking
	state.CurrentQuestion = 1
	state.CurrentAskerIdx = 0

	addNQLog(state, fmt.Sprintf("Category: %s — %d questions to guess the word!", strings.ToUpper(category), state.MaxQuestions))
	return nil
}

// getNonGiverPlayers returns the list of non-giver player indices
func getNonGiverPlayers(state *NQState) []int {
	var indices []int
	for i, p := range state.Players {
		if !p.IsGiver {
			indices = append(indices, i)
		}
	}
	return indices
}

// NQAskQuestion: a guesser asks a question or makes a guess
func NQAskQuestion(state *NQState, playerID string, question string, isGuess bool, guessWord string) error {
	if state.Phase != NQPhaseAsking {
		return fmt.Errorf("not in asking phase")
	}
	if state.WaitingForAnswer {
		return fmt.Errorf("waiting for giver's answer")
	}

	// Check it's this player's turn
	nonGivers := getNonGiverPlayers(state)
	if len(nonGivers) == 0 {
		return fmt.Errorf("no guessers available")
	}
	currentPlayerIdx := nonGivers[state.CurrentAskerIdx % len(nonGivers)]
	if state.Players[currentPlayerIdx].ID != playerID {
		return fmt.Errorf("it's not your turn to ask")
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return fmt.Errorf("question cannot be empty")
	}
	if len(question) > 200 {
		return fmt.Errorf("question is too long")
	}

	if isGuess {
		guessWord = strings.TrimSpace(guessWord)
		if guessWord == "" {
			return fmt.Errorf("guess cannot be empty")
		}
		// Check if guess is correct
		correct := strings.EqualFold(guessWord, state.SecretWord)
		q := NQQuestion{
			AskerID:   playerID,
			AskerName: state.Players[currentPlayerIdx].Name,
			Question:  question,
			IsGuess:   true,
			GuessWord: guessWord,
			Correct:   correct,
			Turn:      state.CurrentQuestion,
			Timestamp: time.Now().UnixMilli(),
		}
		if correct {
			q.Answer = "✅ CORRECT!"
			state.Questions = append(state.Questions, q)
			state.Winner = "guessers"
			state.CorrectGuesser = state.Players[currentPlayerIdx].Name
			state.Phase = NQPhaseFinished
			state.Finished = true
			addNQLog(state, fmt.Sprintf("🎉 %s guessed correctly: \"%s\"! Guessers win!", state.Players[currentPlayerIdx].Name, guessWord))
			return nil
		}
		// Wrong guess — counts as a question, giver doesn't need to answer
		q.Answer = "❌ Wrong!"
		state.Questions = append(state.Questions, q)
		addNQLog(state, fmt.Sprintf("%s guessed \"%s\" — Wrong!", state.Players[currentPlayerIdx].Name, guessWord))
		advanceNQTurn(state)
		return nil
	}

	// It's a regular question — wait for giver's answer
	state.WaitingForAnswer = true
	state.PendingQuestion = question
	state.PendingAskerID = playerID

	addNQLog(state, fmt.Sprintf("Q%d — %s: \"%s\"", state.CurrentQuestion, state.Players[currentPlayerIdx].Name, question))
	return nil
}

// NQAnswerQuestion: giver answers yes/no/comment
func NQAnswerQuestion(state *NQState, playerID string, answer string) error {
	if state.Phase != NQPhaseAsking {
		return fmt.Errorf("not in asking phase")
	}
	if !state.WaitingForAnswer {
		return fmt.Errorf("no pending question")
	}

	// Only giver can answer
	isGiver := false
	for _, p := range state.Players {
		if p.ID == playerID && p.IsGiver {
			isGiver = true
			break
		}
	}
	if !isGiver {
		return fmt.Errorf("only the word-giver can answer")
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return fmt.Errorf("answer cannot be empty")
	}
	if len(answer) > 200 {
		return fmt.Errorf("answer is too long")
	}

	// Find asker name
	askerName := ""
	for _, p := range state.Players {
		if p.ID == state.PendingAskerID {
			askerName = p.Name
			break
		}
	}

	q := NQQuestion{
		AskerID:   state.PendingAskerID,
		AskerName: askerName,
		Question:  state.PendingQuestion,
		Answer:    answer,
		Turn:      state.CurrentQuestion,
		Timestamp: time.Now().UnixMilli(),
	}
	state.Questions = append(state.Questions, q)
	state.WaitingForAnswer = false
	state.PendingQuestion = ""
	state.PendingAskerID = ""

	addNQLog(state, fmt.Sprintf("A%d — %s", state.CurrentQuestion, answer))
	advanceNQTurn(state)
	return nil
}

func advanceNQTurn(state *NQState) {
	nonGivers := getNonGiverPlayers(state)
	state.CurrentAskerIdx = (state.CurrentAskerIdx + 1) % len(nonGivers)

	// If we've gone around once, that's one question used
	if state.CurrentAskerIdx == 0 {
		state.CurrentQuestion++
	}

	if state.CurrentQuestion > state.MaxQuestions {
		// Move to final guess phase
		state.Phase = NQPhaseFinalGuess
		state.FinalGuessIdx = 0
		addNQLog(state, fmt.Sprintf("All %d questions used! Each guesser gets one final guess.", state.MaxQuestions))
	}
}

// NQFinalGuess: each guesser gets one final attempt
func NQFinalGuess(state *NQState, playerID string, guess string) error {
	if state.Phase != NQPhaseFinalGuess {
		return fmt.Errorf("not in final guess phase")
	}

	nonGivers := getNonGiverPlayers(state)
	if state.FinalGuessIdx >= len(nonGivers) {
		return fmt.Errorf("all guesses have been made")
	}

	currentIdx := nonGivers[state.FinalGuessIdx]
	if state.Players[currentIdx].ID != playerID {
		return fmt.Errorf("it's not your turn to guess")
	}

	guess = strings.TrimSpace(guess)
	if guess == "" {
		return fmt.Errorf("guess cannot be empty")
	}

	correct := strings.EqualFold(guess, state.SecretWord)
	state.Players[currentIdx].HasGuessed = true

	q := NQQuestion{
		AskerID:   playerID,
		AskerName: state.Players[currentIdx].Name,
		Question:  fmt.Sprintf("Final guess: \"%s\"", guess),
		IsGuess:   true,
		GuessWord: guess,
		Correct:   correct,
		Answer:    func() string { if correct { return "✅ CORRECT!" }; return "❌ Wrong!" }(),
		Turn:      state.MaxQuestions + 1,
		Timestamp: time.Now().UnixMilli(),
	}
	state.Questions = append(state.Questions, q)

	if correct {
		state.Winner = "guessers"
		state.CorrectGuesser = state.Players[currentIdx].Name
		state.Phase = NQPhaseFinished
		state.Finished = true
		addNQLog(state, fmt.Sprintf("🎉 %s guessed correctly: \"%s\"! Guessers win!", state.Players[currentIdx].Name, guess))
		return nil
	}

	addNQLog(state, fmt.Sprintf("%s guessed \"%s\" — Wrong!", state.Players[currentIdx].Name, guess))
	state.FinalGuessIdx++

	if state.FinalGuessIdx >= len(nonGivers) {
		// All guessers failed
		state.Winner = "giver"
		state.Phase = NQPhaseFinished
		state.Finished = true
		giverName := ""
		for _, p := range state.Players {
			if p.IsGiver {
				giverName = p.Name
				break
			}
		}
		addNQLog(state, fmt.Sprintf("Nobody guessed it! The word was \"%s\". %s wins!", state.SecretWord, giverName))
	}
	return nil
}

// NQNextRound starts a new round with the next player as giver
func NQNextRound(state *NQState) {
	if state.Phase != NQPhaseFinished {
		return
	}
	// Find current giver index and rotate to next player
	giverIdx := 0
	for i, p := range state.Players {
		if p.IsGiver {
			giverIdx = i
			break
		}
	}
	nextGiverIdx := (giverIdx + 1) % len(state.Players)

	// Reset all players
	for i := range state.Players {
		state.Players[i].IsGiver = (i == nextGiverIdx)
		state.Players[i].HasGuessed = false
	}

	// Reset game state
	state.Phase = NQPhaseSetup
	state.Category = ""
	state.SecretWord = ""
	state.CurrentQuestion = 0
	state.CurrentAskerIdx = 0
	state.Questions = []NQQuestion{}
	state.WaitingForAnswer = false
	state.PendingQuestion = ""
	state.PendingAskerID = ""
	state.Winner = ""
	state.CorrectGuesser = ""
	state.Finished = false
	state.FinalGuessIdx = 0
	state.Log = []NQLogEntry{}
	state.LastAction = ""
	state.ID = fmt.Sprintf("nq-%d", time.Now().UnixNano())

	addNQLog(state, fmt.Sprintf("New round! %s is now the word-giver!", state.Players[nextGiverIdx].Name))
}

// NQVoluntaryExit removes a player from the game
func NQVoluntaryExit(state *NQState, playerID string) {
	if state.Phase == NQPhaseFinished {
		return
	}
	for i, p := range state.Players {
		if p.ID == playerID {
			if p.IsGiver {
				// Giver left — game ends
				state.Phase = NQPhaseFinished
				state.Finished = true
				state.Winner = "guessers"
				addNQLog(state, fmt.Sprintf("%s (word-giver) left the game. Game over!", p.Name))
			} else {
				// Remove guesser
				state.Players = append(state.Players[:i], state.Players[i+1:]...)
				addNQLog(state, fmt.Sprintf("%s left the game.", p.Name))

				// Check if any guessers remain
				nonGivers := getNonGiverPlayers(state)
				if len(nonGivers) == 0 {
					state.Phase = NQPhaseFinished
					state.Finished = true
					state.Winner = "giver"
					addNQLog(state, "All guessers left. Game over!")
				} else if state.Phase == NQPhaseAsking && state.CurrentAskerIdx >= len(nonGivers) {
					state.CurrentAskerIdx = 0
				} else if state.Phase == NQPhaseFinalGuess && state.FinalGuessIdx >= len(nonGivers) {
					state.Phase = NQPhaseFinished
					state.Finished = true
					state.Winner = "giver"
					addNQLog(state, fmt.Sprintf("Nobody guessed it! The word was \"%s\".", state.SecretWord))
				}
			}
			return
		}
	}
}

// SanitizeNQStateForGuessers removes secret word from state sent to guessers
func SanitizeNQStateForGuessers(state *NQState) *NQState {
	// Deep copy
	copy := *state
	playersCopy := make([]NQPlayer, len(state.Players))
	for i, p := range state.Players {
		playersCopy[i] = p
	}
	copy.Players = playersCopy
	qCopy := make([]NQQuestion, len(state.Questions))
	for i, q := range state.Questions {
		qCopy[i] = q
	}
	copy.Questions = qCopy
	logCopy := make([]NQLogEntry, len(state.Log))
	for i, l := range state.Log {
		logCopy[i] = l
	}
	copy.Log = logCopy

	// Hide secret word unless game is finished
	if copy.Phase != NQPhaseFinished {
		copy.SecretWord = ""
	}
	return &copy
}
