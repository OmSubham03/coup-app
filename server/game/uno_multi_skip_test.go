package game

import (
	"fmt"
	"testing"
)

func multiSkipFixture(players int, enabled bool) *UNOState {
	state := &UNOState{Phase: UNOPhasePlaying, Direction: 1, MultiSkipEnabled: enabled,
		CurrentColor: "green", DiscardPile: []UNOCard{{ID: "top", Color: "green", Value: "skip", Kind: "skip"}}, Round: 1}
	for index := 0; index < players; index++ {
		state.Players = append(state.Players, UNOPlayer{ID: fmt.Sprintf("p%d", index), Name: fmt.Sprintf("Player %d", index),
			Cards: []UNOCard{{ID: fmt.Sprintf("number%d", index), Color: "blue", Value: "9", Kind: "number"}}})
	}
	state.Players[0].Cards = append(state.Players[0].Cards, UNOCard{ID: "skip0", Color: "red", Value: "skip", Kind: "skip"})
	return state
}

func addFixtureSkip(state *UNOState, index int) {
	state.Players[index].Cards = append(state.Players[index].Cards, UNOCard{ID: fmt.Sprintf("skip%d", index), Color: "yellow", Value: "skip", Kind: "skip"})
}

func TestUNOStandardSkipCanMatchAnotherSkip(t *testing.T) {
	state := multiSkipFixture(4, false)
	addFixtureSkip(state, 1)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal("standard different-color Skip-on-Skip was rejected:", err)
	}
	if state.CurrentPlayerIdx != 2 || state.PendingSkips != 0 {
		t.Fatal("standard Skip should bypass the next player, not start a chain")
	}
	if err := UNOPlayCard(state, "p1", "skip1", ""); err == nil {
		t.Fatal("standard UNO must not let the skipped player counter with Skip")
	}
}

func TestUNOMultiSkipThreeCardChainInSixPlayerGame(t *testing.T) {
	state := multiSkipFixture(6, true)
	addFixtureSkip(state, 1)
	addFixtureSkip(state, 2)
	for index := 0; index < 3; index++ {
		if err := UNOPlayCard(state, fmt.Sprintf("p%d", index), fmt.Sprintf("skip%d", index), ""); err != nil {
			t.Fatal(err)
		}
	}
	if state.CurrentPlayerIdx != 0 || state.PendingSkips != 0 {
		t.Fatal("three Skips should bypass players 3, 4, and 5, then return to player 0")
	}
}

func TestUNOMultiSkipChainSkipsCountFromFirstNonSkipHolder(t *testing.T) {
	state := multiSkipFixture(6, true)
	addFixtureSkip(state, 1)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 1 || state.PendingSkips != 1 {
		t.Fatal("next player should be able to respond with Skip")
	}
	if err := UNOPlayCard(state, "p1", "number1", ""); err == nil {
		t.Fatal("a non-Skip cannot answer a multi-skip chain")
	}
	if err := UNODrawCard(state, "p1"); err == nil {
		t.Fatal("drawing must not bypass a skip chain")
	}
	if err := UNOEndTurn(state, "p1"); err == nil {
		t.Fatal("ending turn must not bypass a skip chain")
	}
	if err := UNOPlayCard(state, "p1", "skip1", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 4 || state.PendingSkips != 0 {
		t.Fatalf("two skips should skip players 2 and 3; got player %d, pending %d", state.CurrentPlayerIdx, state.PendingSkips)
	}
}

func TestUNOMultiSkipReverseDirection(t *testing.T) {
	state := multiSkipFixture(6, true)
	state.Direction = -1
	addFixtureSkip(state, 5)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 5 {
		t.Fatal("skip response must follow current direction")
	}
	if err := UNOPlayCard(state, "p5", "skip5", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 2 || state.PendingSkips != 0 {
		t.Fatal("reverse-direction chain should skip players 4 and 3")
	}
}

func TestUNOMultiSkipTwoPlayerWraparound(t *testing.T) {
	state := multiSkipFixture(2, true)
	addFixtureSkip(state, 1)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if err := UNOPlayCard(state, "p1", "skip1", ""); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 0 || state.PendingSkips != 0 {
		t.Fatal("two skipped turns should wrap back to player 0")
	}
}

func TestUNOMultiSkipForfeitResolvesNonSkipHolder(t *testing.T) {
	state := multiSkipFixture(4, true)
	addFixtureSkip(state, 1)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if err := UNOForfeitPlayer(state, "p1"); err != nil {
		t.Fatal(err)
	}
	if state.Players[state.CurrentPlayerIdx].ID != "p3" || state.PendingSkips != 0 {
		t.Fatal("forfeit should resolve the remaining skip, not block a player with no Skip")
	}
}

func TestUNOMultiSkipForfeitInReverseDirection(t *testing.T) {
	state := multiSkipFixture(6, true)
	state.Direction = -1
	addFixtureSkip(state, 5)
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if err := UNOForfeitPlayer(state, "p5"); err != nil {
		t.Fatal(err)
	}
	if state.Players[state.CurrentPlayerIdx].ID != "p3" || state.PendingSkips != 0 {
		t.Fatal("reverse forfeit should skip player 4 and resume with player 3")
	}
}

func TestUNOMultiSkipRoundEndClearsChainAndPreservesOption(t *testing.T) {
	state := multiSkipFixture(4, true)
	state.Players[1].Cards = []UNOCard{{ID: "skip1", Color: "yellow", Value: "skip", Kind: "skip"}}
	if err := UNOPlayCard(state, "p0", "skip0", ""); err != nil {
		t.Fatal(err)
	}
	if err := UNOPlayCard(state, "p1", "skip1", ""); err != nil {
		t.Fatal(err)
	}
	if state.Phase != UNOPhaseRoundOver || state.PendingSkips != 0 {
		t.Fatal("round winner must end the skip chain")
	}
	if err := UNOStartNextRound(state); err != nil {
		t.Fatal(err)
	}
	if !state.MultiSkipEnabled || state.PendingSkips != 0 {
		t.Fatal("next round should keep the room option without a stale skip chain")
	}
}
