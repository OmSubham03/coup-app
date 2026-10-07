package game

import (
	"fmt"
	"math/rand"
	"sort"
)

type BluffPhase string

const (
	BluffPhasePlaying   BluffPhase = "playing"
	BluffPhaseChallenge BluffPhase = "challenge"
	BluffPhaseGameOver  BluffPhase = "game_over"
)

type BluffCard struct {
	ID   string `json:"id"`
	Suit string `json:"suit"`
	Rank int    `json:"rank"`
}

type BluffPlayer struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Cards  []BluffCard `json:"cards"`
	Active bool        `json:"active"`
}

type BluffState struct {
	Players          []BluffPlayer `json:"players"`
	Phase            BluffPhase    `json:"phase"`
	CurrentPlayerIdx int           `json:"currentPlayerIdx"`
	RequiredRank     int           `json:"requiredRank"`
	Pile             []BluffCard   `json:"pile"`
	PendingActorID   string        `json:"pendingActorId"`
	WinnerName       string        `json:"winnerName"`
	LastAction       string        `json:"lastAction"`
	TwoDecks         bool          `json:"twoDecks"`
	pendingCards     []BluffCard
}

func bluffCreateDeck(twoDecks bool) []BluffCard {
	deckCount := 1
	if twoDecks {
		deckCount = 2
	}
	deck := make([]BluffCard, 0, 52*deckCount)
	for deckIdx := 0; deckIdx < deckCount; deckIdx++ {
		for _, suit := range []string{"clubs", "diamonds", "hearts", "spades"} {
			for rank := 1; rank <= 13; rank++ {
				deck = append(deck, BluffCard{ID: fmt.Sprintf("bluff_%d_%s_%d", deckIdx, suit, rank), Suit: suit, Rank: rank})
			}
		}
	}
	return deck
}

func BluffInitializeGame(players map[string]string, twoDecks bool) (*BluffState, error) {
	if len(players) < 2 || len(players) > 6 {
		return nil, fmt.Errorf("Bluff requires 2 to 6 players")
	}
	ids := make([]string, 0, len(players))
	for id := range players {
		if id == "" {
			return nil, ErrInvalidPlayer
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state := &BluffState{Phase: BluffPhasePlaying, RequiredRank: 1, TwoDecks: twoDecks, Pile: []BluffCard{}, LastAction: "Game started; Aces required"}
	for _, id := range ids {
		state.Players = append(state.Players, BluffPlayer{ID: id, Name: players[id], Active: true, Cards: []BluffCard{}})
	}
	deck := bluffCreateDeck(twoDecks)
	rand.Shuffle(len(deck), func(first, second int) { deck[first], deck[second] = deck[second], deck[first] })
	for cardIdx, card := range deck {
		playerIdx := cardIdx % len(state.Players)
		state.Players[playerIdx].Cards = append(state.Players[playerIdx].Cards, card)
	}
	return state, nil
}

func bluffFindPlayer(state *BluffState, id string) int {
	for playerIdx, player := range state.Players {
		if player.ID == id && player.Active {
			return playerIdx
		}
	}
	return -1
}

func bluffNextActive(state *BluffState, after int) int {
	for offset := 1; offset <= len(state.Players); offset++ {
		playerIdx := (after + offset) % len(state.Players)
		if state.Players[playerIdx].Active {
			return playerIdx
		}
	}
	return -1
}

func BluffPlay(state *BluffState, id string, cardIDs []string) error {
	if state == nil || state.Phase != BluffPhasePlaying {
		return ErrWrongPhase
	}
	playerIdx := bluffFindPlayer(state, id)
	if playerIdx < 0 {
		return ErrInvalidPlayer
	}
	if playerIdx != state.CurrentPlayerIdx {
		return ErrNotYourTurn
	}
	if len(cardIDs) < 1 || len(cardIDs) > 4 {
		return ErrWrongCardCount
	}
	selected := make(map[string]bool, len(cardIDs))
	for _, cardID := range cardIDs {
		if selected[cardID] {
			return ErrInvalidCard
		}
		selected[cardID] = true
	}
	played := make([]BluffCard, 0, len(cardIDs))
	remaining := make([]BluffCard, 0, len(state.Players[playerIdx].Cards))
	for _, card := range state.Players[playerIdx].Cards {
		if selected[card.ID] {
			played = append(played, card)
		} else {
			remaining = append(remaining, card)
		}
	}
	if len(played) != len(cardIDs) {
		return ErrInvalidCard
	}
	state.Players[playerIdx].Cards = remaining
	state.Pile = append(state.Pile, played...)
	state.pendingCards = played
	state.PendingActorID = id
	state.Phase = BluffPhaseChallenge
	state.CurrentPlayerIdx = bluffNextActive(state, playerIdx)
	state.LastAction = fmt.Sprintf("%s claimed %d card(s) of rank %d", state.Players[playerIdx].Name, len(played), state.RequiredRank)
	return nil
}

func bluffResolve(state *BluffState, actorIdx int, canWin bool) {
	state.PendingActorID = ""
	state.pendingCards = nil
	state.RequiredRank = state.RequiredRank%13 + 1
	state.CurrentPlayerIdx = bluffNextActive(state, actorIdx)
	state.Phase = BluffPhasePlaying
	if canWin && len(state.Players[actorIdx].Cards) == 0 {
		state.Phase = BluffPhaseGameOver
		state.WinnerName = state.Players[actorIdx].Name
	}
}

func BluffChallenge(state *BluffState, id string) error {
	if state == nil || state.Phase != BluffPhaseChallenge {
		return ErrWrongPhase
	}
	challengerIdx := bluffFindPlayer(state, id)
	if challengerIdx < 0 {
		return ErrInvalidPlayer
	}
	if id == state.PendingActorID {
		return ErrSelfChallenge
	}
	actorIdx := bluffFindPlayer(state, state.PendingActorID)
	if actorIdx < 0 || len(state.pendingCards) == 0 {
		return ErrNoPendingAction
	}
	truthful := true
	for _, card := range state.pendingCards {
		if card.Rank != state.RequiredRank {
			truthful = false
			break
		}
	}
	pickerIdx := actorIdx
	if truthful {
		pickerIdx = challengerIdx
		state.LastAction = fmt.Sprintf("%s's claim was truthful; %s picked up the pile", state.Players[actorIdx].Name, state.Players[challengerIdx].Name)
	} else {
		state.LastAction = fmt.Sprintf("%s was bluffing and picked up the pile", state.Players[actorIdx].Name)
	}
	state.Players[pickerIdx].Cards = append(state.Players[pickerIdx].Cards, state.Pile...)
	state.Pile = []BluffCard{}
	bluffResolve(state, actorIdx, truthful)
	return nil
}

func BluffAccept(state *BluffState, id string) error {
	if state == nil || state.Phase != BluffPhaseChallenge {
		return ErrWrongPhase
	}
	playerIdx := bluffFindPlayer(state, id)
	if playerIdx < 0 {
		return ErrInvalidPlayer
	}
	actorIdx := bluffFindPlayer(state, state.PendingActorID)
	if actorIdx < 0 || len(state.pendingCards) == 0 {
		return ErrNoPendingAction
	}
	if playerIdx != bluffNextActive(state, actorIdx) || id == state.PendingActorID {
		return ErrNotYourTurn
	}
	state.LastAction = fmt.Sprintf("%s accepted %s's claim", state.Players[playerIdx].Name, state.Players[actorIdx].Name)
	bluffResolve(state, actorIdx, true)
	return nil
}

func BluffExit(state *BluffState, id string) error {
	if state == nil || state.Phase == BluffPhaseGameOver {
		return ErrWrongPhase
	}
	playerIdx := bluffFindPlayer(state, id)
	if playerIdx < 0 {
		return ErrInvalidPlayer
	}
	name := state.Players[playerIdx].Name
	state.Pile = append(state.Pile, state.Players[playerIdx].Cards...)
	actorExited := state.PendingActorID == id
	currentExited := state.CurrentPlayerIdx == playerIdx
	state.Players = append(state.Players[:playerIdx], state.Players[playerIdx+1:]...)
	if playerIdx < state.CurrentPlayerIdx {
		state.CurrentPlayerIdx--
	}
	if actorExited {
		state.PendingActorID = ""
		state.pendingCards = nil
		state.RequiredRank = state.RequiredRank%13 + 1
		state.Phase = BluffPhasePlaying
	}
	if actorExited || currentExited {
		state.CurrentPlayerIdx = bluffNextActive(state, playerIdx-1)
	}
	if state.Phase == BluffPhaseChallenge {
		state.CurrentPlayerIdx = bluffNextActive(state, bluffFindPlayer(state, state.PendingActorID))
	}
	activeCount := 0
	for _, player := range state.Players {
		if player.Active {
			activeCount++
		}
	}
	if activeCount < 2 {
		state.Phase = BluffPhaseGameOver
		state.PendingActorID = ""
		state.pendingCards = nil
	}
	state.LastAction = fmt.Sprintf("%s left; their cards were returned to the pile", name)
	return nil
}

func BluffSanitize(state *BluffState, id string) *BluffState {
	if state == nil {
		return nil
	}
	result := *state
	result.pendingCards = nil
	result.Pile = make([]BluffCard, len(state.Pile))
	result.Players = make([]BluffPlayer, len(state.Players))
	for playerIdx, player := range state.Players {
		result.Players[playerIdx] = player
		result.Players[playerIdx].Cards = make([]BluffCard, len(player.Cards))
		if player.ID == id {
			copy(result.Players[playerIdx].Cards, player.Cards)
		}
	}
	return &result
}
