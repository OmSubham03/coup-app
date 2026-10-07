package game

import (
	"fmt"
	"math/rand"
	"sort"
)

type BlackjackPhase string
type BlackjackStatus string

const (
	BlackjackPhasePlaying    BlackjackPhase  = "playing"
	BlackjackPhaseRoundOver  BlackjackPhase  = "round_over"
	BlackjackPhaseGameOver   BlackjackPhase  = "game_over"
	BlackjackStatusActive    BlackjackStatus = "active"
	BlackjackStatusStood     BlackjackStatus = "stood"
	BlackjackStatusBust      BlackjackStatus = "bust"
	BlackjackStatusBlackjack BlackjackStatus = "blackjack"
	blackjackBet                             = 50
)

type BlackjackCard struct {
	ID   string `json:"id"`
	Suit string `json:"suit"`
	Rank int    `json:"rank"`
}

type BlackjackPlayer struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Cards  []BlackjackCard `json:"cards"`
	Chips  int             `json:"chips"`
	Bet    int             `json:"bet"`
	Status BlackjackStatus `json:"status"`
	Active bool            `json:"active"`
}

type BlackjackState struct {
	Players          []BlackjackPlayer `json:"players"`
	DealerCards      []BlackjackCard   `json:"dealerCards"`
	Phase            BlackjackPhase    `json:"phase"`
	CurrentPlayerIdx int               `json:"currentPlayerIdx"`
	Round            int               `json:"round"`
	TwoDecks         bool              `json:"twoDecks"`
	LastAction       string            `json:"lastAction"`
	shoe             []BlackjackCard
}

func blackjackCreateDeck(twoDecks bool) []BlackjackCard {
	deckCount := 1
	if twoDecks {
		deckCount = 2
	}
	deck := make([]BlackjackCard, 0, 52*deckCount)
	for deckIdx := 0; deckIdx < deckCount; deckIdx++ {
		for _, suit := range []string{"clubs", "diamonds", "hearts", "spades"} {
			for rank := 1; rank <= 13; rank++ {
				deck = append(deck, BlackjackCard{ID: fmt.Sprintf("blackjack_%d_%s_%d", deckIdx, suit, rank), Suit: suit, Rank: rank})
			}
		}
	}
	return deck
}

func blackjackScore(cards []BlackjackCard) int {
	total, aces := 0, 0
	for _, card := range cards {
		if card.Rank == 1 {
			total += 11
			aces++
		} else if card.Rank >= 10 {
			total += 10
		} else {
			total += card.Rank
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func blackjackNatural(cards []BlackjackCard) bool {
	return len(cards) == 2 && blackjackScore(cards) == 21
}

func blackjackSetAction(state *BlackjackState, action string) {
	deckCount := 1
	if state.TwoDecks {
		deckCount = 2
	}
	state.LastAction = fmt.Sprintf("Round %d (fresh shuffled %d-deck shoe): %s", state.Round, deckCount, action)
}

func BlackjackInitializeGame(players map[string]string, twoDecks bool) (*BlackjackState, error) {
	if len(players) < 1 || len(players) > 4 {
		return nil, fmt.Errorf("Blackjack requires 1 to 4 players")
	}
	ids := make([]string, 0, len(players))
	for id := range players {
		if id == "" {
			return nil, ErrInvalidPlayer
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	state := &BlackjackState{TwoDecks: twoDecks}
	for _, id := range ids {
		state.Players = append(state.Players, BlackjackPlayer{ID: id, Name: players[id], Chips: 1000, Active: true, Cards: []BlackjackCard{}})
	}
	if err := blackjackStartRound(state); err != nil {
		return nil, err
	}
	return state, nil
}

func blackjackStartRound(state *BlackjackState) error {
	deck := blackjackCreateDeck(state.TwoDecks)
	rand.Shuffle(len(deck), func(first, second int) { deck[first], deck[second] = deck[second], deck[first] })
	return blackjackDealRound(state, deck)
}

func blackjackDealRound(state *BlackjackState, deck []BlackjackCard) error {
	eligible := 0
	for _, player := range state.Players {
		if player.Active && player.Chips >= blackjackBet {
			eligible++
		}
	}
	if eligible > 0 && len(deck) < 2*(eligible+1) {
		return ErrActionNotAvailable
	}
	state.shoe = append([]BlackjackCard{}, deck...)
	state.DealerCards = []BlackjackCard{}
	state.CurrentPlayerIdx = -1
	state.Phase = BlackjackPhasePlaying
	for playerIdx := range state.Players {
		player := &state.Players[playerIdx]
		player.Cards = []BlackjackCard{}
		player.Bet = 0
		player.Status = BlackjackStatusStood
		if player.Active && player.Chips >= blackjackBet {
			player.Chips -= blackjackBet
			player.Bet = blackjackBet
			player.Status = BlackjackStatusActive
		} else {
			player.Active = false
		}
	}
	if eligible == 0 {
		state.Phase = BlackjackPhaseGameOver
		blackjackSetAction(state, "No active player can cover the 50-chip bet")
		return nil
	}
	state.Round++
	for dealIdx := 0; dealIdx < 2; dealIdx++ {
		for playerIdx := range state.Players {
			if state.Players[playerIdx].Active {
				state.Players[playerIdx].Cards = append(state.Players[playerIdx].Cards, state.shoe[0])
				state.shoe = state.shoe[1:]
			}
		}
		state.DealerCards = append(state.DealerCards, state.shoe[0])
		state.shoe = state.shoe[1:]
	}
	for playerIdx := range state.Players {
		if state.Players[playerIdx].Active && blackjackNatural(state.Players[playerIdx].Cards) {
			state.Players[playerIdx].Status = BlackjackStatusBlackjack
		}
	}
	blackjackSetAction(state, "Cards dealt; fixed bet 50")
	if blackjackNatural(state.DealerCards) {
		return blackjackFinishRound(state)
	}
	return blackjackAdvance(state, -1)
}

func blackjackFindPlayer(state *BlackjackState, id string) int {
	for playerIdx, player := range state.Players {
		if player.ID == id && player.Active {
			return playerIdx
		}
	}
	return -1
}

func blackjackTurnPlayer(state *BlackjackState, id string) (int, error) {
	if state == nil || state.Phase != BlackjackPhasePlaying {
		return -1, ErrWrongPhase
	}
	playerIdx := blackjackFindPlayer(state, id)
	if playerIdx < 0 {
		return -1, ErrInvalidPlayer
	}
	if playerIdx != state.CurrentPlayerIdx || state.Players[playerIdx].Status != BlackjackStatusActive {
		return -1, ErrNotYourTurn
	}
	return playerIdx, nil
}

func blackjackAdvance(state *BlackjackState, after int) error {
	for playerIdx := after + 1; playerIdx < len(state.Players); playerIdx++ {
		if state.Players[playerIdx].Active && state.Players[playerIdx].Status == BlackjackStatusActive {
			state.CurrentPlayerIdx = playerIdx
			return nil
		}
	}
	return blackjackFinishRound(state)
}

func blackjackFinishRound(state *BlackjackState) error {
	dealer := append([]BlackjackCard{}, state.DealerCards...)
	shoe := state.shoe
	for blackjackScore(dealer) < 17 {
		if len(shoe) == 0 {
			return ErrActionNotAvailable
		}
		dealer = append(dealer, shoe[0])
		shoe = shoe[1:]
	}
	state.DealerCards = dealer
	state.shoe = shoe
	dealerTotal := blackjackScore(dealer)
	dealerNatural := blackjackNatural(dealer)
	for playerIdx := range state.Players {
		player := &state.Players[playerIdx]
		if !player.Active || player.Bet == 0 {
			continue
		}
		playerTotal := blackjackScore(player.Cards)
		playerNatural := blackjackNatural(player.Cards)
		switch {
		case playerTotal > 21:
		case dealerNatural && playerNatural:
			player.Chips += player.Bet
		case dealerNatural:
		case playerNatural:
			player.Chips += player.Bet + player.Bet*3/2
		case dealerTotal > 21 || playerTotal > dealerTotal:
			player.Chips += player.Bet * 2
		case playerTotal == dealerTotal:
			player.Chips += player.Bet
		}
		if player.Status == BlackjackStatusActive {
			player.Status = BlackjackStatusStood
		}
	}
	state.Phase = BlackjackPhaseRoundOver
	state.CurrentPlayerIdx = -1
	if dealerNatural {
		blackjackSetAction(state, "Dealer blackjack; round settled")
	} else {
		blackjackSetAction(state, fmt.Sprintf("Dealer finished with %d; round settled", dealerTotal))
	}
	return nil
}

func BlackjackHit(state *BlackjackState, id string) error {
	playerIdx, err := blackjackTurnPlayer(state, id)
	if err != nil {
		return err
	}
	if len(state.shoe) == 0 {
		return ErrActionNotAvailable
	}
	player := &state.Players[playerIdx]
	player.Cards = append(player.Cards, state.shoe[0])
	state.shoe = state.shoe[1:]
	total := blackjackScore(player.Cards)
	blackjackSetAction(state, fmt.Sprintf("%s hit", player.Name))
	if total > 21 {
		player.Status = BlackjackStatusBust
	} else if total == 21 {
		player.Status = BlackjackStatusStood
	} else {
		return nil
	}
	return blackjackAdvance(state, playerIdx)
}

func BlackjackStand(state *BlackjackState, id string) error {
	playerIdx, err := blackjackTurnPlayer(state, id)
	if err != nil {
		return err
	}
	state.Players[playerIdx].Status = BlackjackStatusStood
	blackjackSetAction(state, fmt.Sprintf("%s stood", state.Players[playerIdx].Name))
	return blackjackAdvance(state, playerIdx)
}

func BlackjackDouble(state *BlackjackState, id string) error {
	playerIdx, err := blackjackTurnPlayer(state, id)
	if err != nil {
		return err
	}
	player := &state.Players[playerIdx]
	if len(player.Cards) != 2 || player.Bet != blackjackBet || len(state.shoe) == 0 {
		return ErrActionNotAvailable
	}
	if player.Chips < player.Bet {
		return ErrNotEnoughCoins
	}
	player.Chips -= player.Bet
	player.Bet *= 2
	player.Cards = append(player.Cards, state.shoe[0])
	state.shoe = state.shoe[1:]
	player.Status = BlackjackStatusStood
	if blackjackScore(player.Cards) > 21 {
		player.Status = BlackjackStatusBust
	}
	blackjackSetAction(state, fmt.Sprintf("%s doubled to %d and drew one card", player.Name, player.Bet))
	return blackjackAdvance(state, playerIdx)
}

func BlackjackNextRound(state *BlackjackState, id string) error {
	if state == nil || state.Phase != BlackjackPhaseRoundOver {
		return ErrWrongPhase
	}
	if blackjackFindPlayer(state, id) < 0 {
		return ErrInvalidPlayer
	}
	return blackjackStartRound(state)
}

func BlackjackExit(state *BlackjackState, id string) error {
	if state == nil || state.Phase == BlackjackPhaseGameOver {
		return ErrWrongPhase
	}
	playerIdx := blackjackFindPlayer(state, id)
	if playerIdx < 0 {
		return ErrInvalidPlayer
	}
	player := &state.Players[playerIdx]
	player.Active = false
	player.Status = BlackjackStatusStood
	player.Bet = 0
	blackjackSetAction(state, fmt.Sprintf("%s left; any outstanding bet is forfeited", player.Name))
	activeCount := 0
	for _, remaining := range state.Players {
		if remaining.Active {
			activeCount++
		}
	}
	if activeCount == 0 {
		state.Phase = BlackjackPhaseGameOver
		state.CurrentPlayerIdx = -1
		return nil
	}
	if state.Phase == BlackjackPhasePlaying && state.CurrentPlayerIdx == playerIdx {
		return blackjackAdvance(state, playerIdx)
	}
	return nil
}

func BlackjackSanitize(state *BlackjackState, id string) *BlackjackState {
	if state == nil {
		return nil
	}
	result := *state
	result.shoe = nil
	result.DealerCards = append([]BlackjackCard{}, state.DealerCards...)
	if state.Phase == BlackjackPhasePlaying {
		for cardIdx := 1; cardIdx < len(result.DealerCards); cardIdx++ {
			result.DealerCards[cardIdx] = BlackjackCard{}
		}
	}
	result.Players = make([]BlackjackPlayer, len(state.Players))
	for playerIdx, player := range state.Players {
		result.Players[playerIdx] = player
		result.Players[playerIdx].Cards = append([]BlackjackCard{}, player.Cards...)
	}
	return &result
}
