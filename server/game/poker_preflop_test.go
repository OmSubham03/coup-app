package game

import "testing"

func TestPokerPreflopBeforeCommunityReveal(t *testing.T) {
	state := InitializePokerGame([]struct{ ID, Name string }{{"one", "One"}, {"two", "Two"}, {"three", "Three"}}, 1000, 10, true)
	if state.Phase != PokerPhasePreflop || len(state.CommunityCards) != 0 {
		t.Fatal("new hand must start with preflop betting and no community cards")
	}
	for _, player := range state.Players {
		if len(player.HoleCards) != 2 {
			t.Fatal("each player must receive two hole cards")
		}
	}
	for actions := 0; state.Phase == PokerPhasePreflop && actions < 12; actions++ {
		if len(state.CommunityCards) != 0 {
			t.Fatal("community cards exposed before preflop completes")
		}
		player := state.Players[state.CurrentPlayerIdx]
		action := "check"
		if player.CurrentBet < state.CurrentBet {
			action = "call"
		}
		if err := PokerAction(state, player.ID, action, 0); err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PokerPhaseFlop || len(state.CommunityCards) != 3 {
		t.Fatal("completed preflop betting must reveal exactly three flop cards")
	}
}