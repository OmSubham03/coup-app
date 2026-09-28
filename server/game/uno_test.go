package game

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUNODeckAndInitialDeal(t *testing.T) {
	deck := UNOCreateDeck()
	if len(deck) != 108 {
		t.Fatalf("UNO deck has %d cards, want 108", len(deck))
	}
	players := []struct{ ID, Name string }{{"p1", "One"}, {"p2", "Two"}, {"p3", "Three"}}
	state, err := UNOInitializeGame(players, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.DrawPile)+len(state.DiscardPile) != 108-21 {
		t.Fatalf("remaining piles contain %d cards, want 87", len(state.DrawPile)+len(state.DiscardPile))
	}
	for _, player := range state.Players {
		if len(player.Cards) != 7 {
			t.Fatalf("%s has %d initial cards, want 7", player.Name, len(player.Cards))
		}
	}
}

func TestUNOPlayAfterDrawingOnlyAllowsDrawnCard(t *testing.T) {
	state := &UNOState{
		Players: []UNOPlayer{{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "held", Color: "blue", Value: "5", Kind: "number"}}}, {ID: "p2", Name: "Two"}},
		Phase:   UNOPhasePlaying, CurrentPlayerIdx: 0, Direction: 1,
		DrawPile:    []UNOCard{{ID: "drawn", Color: "red", Value: "8", Kind: "number"}},
		DiscardPile: []UNOCard{{ID: "top", Color: "red", Value: "5", Kind: "number"}}, CurrentColor: "red",
	}
	if err := UNODrawCard(state, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := UNOPlayCard(state, "p1", "held", ""); err == nil {
		t.Fatal("playing a card from the old hand after drawing should fail")
	}
	if err := UNOPlayCard(state, "p1", "drawn", ""); err != nil {
		t.Fatalf("playing the drawn card should succeed: %v", err)
	}
}

func TestUNOWildDrawFourRequiresNoMatchingColor(t *testing.T) {
	state := &UNOState{
		Players: []UNOPlayer{{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "wild4", Color: "wild", Value: "+4", Kind: "wild_draw_four"}, {ID: "red2", Color: "red", Value: "2", Kind: "number"}}}, {ID: "p2", Name: "Two"}},
		Phase:   UNOPhasePlaying, CurrentPlayerIdx: 0, Direction: 1,
		DiscardPile: []UNOCard{{ID: "top", Color: "blue", Value: "5", Kind: "number"}}, CurrentColor: "red",
	}
	if err := UNOPlayCard(state, "p1", "wild4", "green"); err == nil {
		t.Fatal("Wild Draw Four should be rejected when the player has a matching color")
	}
}

func TestUNOStackingRulesRespectChosenColorAfterWildDrawFour(t *testing.T) {
	previous := UNOCard{Kind: "wild_draw_four"}
	if UNOCanStack(previous, UNOCard{Kind: "draw_two", Color: "red"}, "blue") {
		t.Fatal("+2 of a different color must not stack on +4")
	}
	if !UNOCanStack(previous, UNOCard{Kind: "draw_two", Color: "blue"}, "blue") {
		t.Fatal("+2 of the color chosen after +4 should be stackable")
	}
}

func TestUNOReverseActsAsSkipWithTwoPlayers(t *testing.T) {
	state := &UNOState{
		Players: []UNOPlayer{{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "reverse", Color: "red", Value: "reverse", Kind: "reverse"}}}, {ID: "p2", Name: "Two"}},
		Phase:   UNOPhasePlaying, CurrentPlayerIdx: 0, Direction: 1,
		DiscardPile: []UNOCard{{ID: "top", Color: "red", Value: "2", Kind: "number"}}, CurrentColor: "red",
	}
	if err := UNOPlayCard(state, "p1", "reverse", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 0 {
		t.Fatalf("two-player Reverse should skip the opponent; current player index = %d", state.CurrentPlayerIdx)
	}
}

func TestUNOOpeningReverseStartsWithPreviousPlayer(t *testing.T) {
	state := &UNOState{
		Players:     make([]UNOPlayer, 4),
		Phase:       UNOPhasePlaying,
		Direction:   1,
		DiscardPile: []UNOCard{},
		DrawPile:    []UNOCard{{ID: "reverse", Color: "red", Value: "reverse", Kind: "reverse"}},
	}
	unoStartWithDiscard(state)
	if state.Direction != -1 || state.CurrentPlayerIdx != 3 {
		t.Fatalf("opening Reverse should move counterclockwise to player 3; direction=%d player=%d", state.Direction, state.CurrentPlayerIdx)
	}
}

func TestUNOChallengeWildDrawFourOutcomes(t *testing.T) {
	newState := func(illegal bool) *UNOState {
		return &UNOState{
			Players: []UNOPlayer{
				{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "p1-card", Color: "red", Value: "1", Kind: "number"}}},
				{ID: "p2", Name: "Two", Cards: []UNOCard{{ID: "p2-card", Color: "blue", Value: "2", Kind: "number"}}},
			},
			Phase: UNOPhasePlaying, CurrentPlayerIdx: 1, Direction: 1, PendingDraw: 4,
			ChallengeAvailable: true, PendingWildDrawFourPlayerID: "p1", PendingWildDrawFourIllegal: illegal,
			DrawPile:    []UNOCard{{ID: "d1"}, {ID: "d2"}, {ID: "d3"}, {ID: "d4"}, {ID: "d5"}, {ID: "d6"}},
			DiscardPile: []UNOCard{{ID: "top", Color: "wild", Value: "+4", Kind: "wild_draw_four"}},
		}
	}

	legalChallenge := newState(true)
	if err := UNOChallengeWildDrawFour(legalChallenge, "p2"); err != nil {
		t.Fatal(err)
	}
	if len(legalChallenge.Players[0].Cards) != 5 || legalChallenge.CurrentPlayerIdx != 1 || legalChallenge.PendingDraw != 0 {
		t.Fatalf("successful challenge state is incorrect: %+v", legalChallenge)
	}

	failedChallenge := newState(false)
	if err := UNOChallengeWildDrawFour(failedChallenge, "p2"); err != nil {
		t.Fatal(err)
	}
	if len(failedChallenge.Players[1].Cards) != 7 || failedChallenge.CurrentPlayerIdx != 0 || failedChallenge.PendingDraw != 0 {
		t.Fatalf("failed challenge should draw 6 and pass turn: %+v", failedChallenge)
	}
}

func TestUNODrawingPenaltyClosesPreviousUNOCallWindow(t *testing.T) {
	state := &UNOState{
		Players: []UNOPlayer{
			{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "last", Color: "red", Value: "1", Kind: "number"}}},
			{ID: "p2", Name: "Two", Cards: []UNOCard{{ID: "held", Color: "blue", Value: "2", Kind: "number"}}},
		},
		Phase: UNOPhasePlaying, CurrentPlayerIdx: 1, Direction: 1,
		PendingDraw: 2, UncalledUNOPlayerID: "p1",
		DrawPile: []UNOCard{{ID: "d1"}, {ID: "d2"}},
	}
	if err := UNOAcceptDrawPenalty(state, "p2"); err != nil {
		t.Fatal(err)
	}
	if state.UncalledUNOPlayerID != "" {
		t.Fatal("the UNO catch window should close when the next player accepts a penalty")
	}
}

func TestUNOSanitizeHidesHandsAndDrawPile(t *testing.T) {
	state := &UNOState{
		Players: []UNOPlayer{
			{ID: "p1", Name: "One", Cards: []UNOCard{{ID: "private", Color: "red", Value: "9", Kind: "number"}}},
			{ID: "p2", Name: "Two", Cards: []UNOCard{{ID: "secret", Color: "blue", Value: "3", Kind: "number"}}, DrawnCardID: "secret"},
		},
		DrawPile:                    []UNOCard{{ID: "draw-secret"}},
		PendingWildDrawFourPlayerID: "private-id", PendingWildDrawFourIllegal: true,
	}
	sanitized := UNOSanitizeState(state, "p1")
	if sanitized.Players[0].Cards[0].ID != "private" || sanitized.Players[1].Cards[0].ID == "secret" || sanitized.Players[1].DrawnCardID != "" || sanitized.DrawPile != nil {
		t.Fatalf("sanitized state leaked or hid the wrong card information: %+v", sanitized)
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PendingWildDrawFourIllegal") || strings.Contains(string(encoded), "private-id") {
		t.Fatalf("sanitized state leaked challenge evidence: %s", encoded)
	}
}

func TestUNORoundScoringAnd500PointWin(t *testing.T) {
	state := &UNOState{Players: []UNOPlayer{
		{ID: "p1", Name: "One"},
		{ID: "p2", Name: "Two", Cards: []UNOCard{{ID: "n", Kind: "number", Value: "9"}, {ID: "a", Kind: "skip"}, {ID: "w", Kind: "wild"}}},
	}}
	unoFinishRound(state, "p1")
	if state.Players[0].Score != 79 || state.Phase != UNOPhaseRoundOver {
		t.Fatalf("round scoring = %d, phase = %s; want 79 and round_over", state.Players[0].Score, state.Phase)
	}
	state.Players[0].Score = 450
	unoFinishRound(state, "p1")
	if state.Phase != UNOPhaseGameOver || state.Players[0].Score != 529 {
		t.Fatalf("500-point match should end with score 529; got %d / %s", state.Players[0].Score, state.Phase)
	}
}
