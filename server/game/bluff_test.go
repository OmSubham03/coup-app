package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func bluffTestState() *BluffState {
	return &BluffState{
		Players: []BluffPlayer{
			{ID: "a", Name: "Alice", Active: true, Cards: []BluffCard{{ID: "ace", Suit: "clubs", Rank: 1}, {ID: "king", Suit: "hearts", Rank: 13}}},
			{ID: "b", Name: "Bob", Active: true, Cards: []BluffCard{{ID: "bob", Suit: "spades", Rank: 2}}},
			{ID: "c", Name: "Carol", Active: true, Cards: []BluffCard{{ID: "carol", Suit: "diamonds", Rank: 3}}},
		}, Phase: BluffPhasePlaying, RequiredRank: 1,
		Pile: []BluffCard{{ID: "old", Suit: "clubs", Rank: 8}},
	}
}

func bluffTestInventory(t *testing.T, state *BluffState) map[string]BluffCard {
	t.Helper()
	cards := make(map[string]BluffCard)
	all := append([]BluffCard{}, state.Pile...)
	for _, player := range state.Players {
		all = append(all, player.Cards...)
	}
	for _, card := range all {
		if _, exists := cards[card.ID]; exists {
			t.Fatalf("duplicate card %q", card.ID)
		}
		cards[card.ID] = card
	}
	return cards
}

func TestBluffInitialization(t *testing.T) {
	for _, twoDecks := range []bool{false, true} {
		for playerCount := 2; playerCount <= 6; playerCount++ {
			players := map[string]string{}
			for playerIdx := 0; playerIdx < playerCount; playerIdx++ {
				players[fmt.Sprint(playerIdx)] = fmt.Sprint(playerIdx)
			}
			state, err := BluffInitializeGame(players, twoDecks)
			if err != nil {
				t.Fatal(err)
			}
			want := 52
			if twoDecks {
				want = 104
			}
			if len(bluffTestInventory(t, state)) != want || len(state.Pile) != 0 || state.RequiredRank != 1 || state.Phase != BluffPhasePlaying || state.TwoDecks != twoDecks {
				t.Fatalf("incorrect initial state: %+v", state)
			}
			counts := map[int]int{}
			for _, player := range state.Players {
				if !player.Active || len(player.Cards) < want/playerCount || len(player.Cards) > (want+playerCount-1)/playerCount {
					t.Fatalf("unbalanced deal: %+v", player)
				}
				for _, card := range player.Cards {
					counts[card.Rank]++
				}
			}
			for rank := 1; rank <= 13; rank++ {
				if counts[rank] != want/13 {
					t.Fatalf("rank %d count = %d", rank, counts[rank])
				}
			}
		}
	}
	for _, playerCount := range []int{0, 1, 7} {
		players := map[string]string{}
		for playerIdx := 0; playerIdx < playerCount; playerIdx++ {
			players[fmt.Sprint(playerIdx)] = "Player"
		}
		if _, err := BluffInitializeGame(players, false); err == nil {
			t.Fatalf("accepted %d players", playerCount)
		}
	}
	if _, err := BluffInitializeGame(map[string]string{"": "Empty", "b": "Bob"}, false); err == nil {
		t.Fatal("accepted empty player ID")
	}
}

func TestBluffPlayValidationAtomic(t *testing.T) {
	for _, ids := range [][]string{nil, {"ace", "ace"}, {"ace", "missing"}, {"1", "2", "3", "4", "5"}} {
		state := bluffTestState()
		before, _ := json.Marshal(state)
		if err := BluffPlay(state, "a", ids); err == nil {
			t.Fatalf("accepted invalid selection %v", ids)
		}
		after, _ := json.Marshal(state)
		if string(before) != string(after) {
			t.Fatal("invalid selection mutated state")
		}
	}
	state := bluffTestState()
	if BluffPlay(state, "b", []string{"bob"}) == nil || BluffPlay(state, "unknown", []string{"ace"}) == nil {
		t.Fatal("accepted invalid actor")
	}
	state.Players[0].Cards = []BluffCard{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}}
	if err := BluffPlay(state, "a", []string{"1", "2", "3", "4"}); err != nil {
		t.Fatalf("four-card bluff rejected: %v", err)
	}
	if BluffPlay(state, "b", []string{"bob"}) == nil {
		t.Fatal("play allowed during pending challenge")
	}
}

func TestBluffChallengeOutcomes(t *testing.T) {
	for _, truthful := range []bool{false, true} {
		state := bluffTestState()
		before := bluffTestInventory(t, state)
		cardID := "king"
		if truthful {
			cardID = "ace"
		}
		if err := BluffPlay(state, "a", []string{cardID}); err != nil {
			t.Fatal(err)
		}
		if state.RequiredRank != 1 || state.PendingActorID != "a" || state.CurrentPlayerIdx != 1 {
			t.Fatal("claim did not remain pending")
		}
		if BluffChallenge(state, "a") == nil || BluffChallenge(state, "unknown") == nil {
			t.Fatal("invalid challenge accepted")
		}
		if err := BluffChallenge(state, "c"); err != nil {
			t.Fatalf("non-next player must be able to challenge: %v", err)
		}
		pickerIdx := 0
		if truthful {
			pickerIdx = 2
		}
		if len(state.Players[pickerIdx].Cards) != 3 || len(state.Pile) != 0 || state.CurrentPlayerIdx != 1 || state.RequiredRank != 2 || state.Phase != BluffPhasePlaying || state.PendingActorID != "" {
			t.Fatalf("wrong challenge outcome: %+v", state)
		}
		if !reflect.DeepEqual(before, bluffTestInventory(t, state)) {
			t.Fatal("challenge changed deck inventory")
		}
		if BluffChallenge(state, "b") == nil {
			t.Fatal("resolved play challenged twice")
		}
	}
}

func TestBluffMixedClaimIsLie(t *testing.T) {
	state := bluffTestState()
	if err := BluffPlay(state, "a", []string{"ace", "king"}); err != nil {
		t.Fatal(err)
	}
	if err := BluffChallenge(state, "b"); err != nil {
		t.Fatal(err)
	}
	if len(state.Players[0].Cards) != 3 || state.WinnerName != "" {
		t.Fatal("mixed claim must be a lie and must not win")
	}
}

func TestBluffWinsOnlyAfterResolution(t *testing.T) {
	for _, outcome := range []string{"accept", "accept bluff", "truthful", "liar"} {
		t.Run(outcome, func(t *testing.T) {
			state := bluffTestState()
			state.Players[0].Cards = state.Players[0].Cards[:1]
			if outcome == "liar" || outcome == "accept bluff" {
				state.Players[0].Cards[0].Rank = 13
			}
			if err := BluffPlay(state, "a", []string{"ace"}); err != nil {
				t.Fatal(err)
			}
			if state.Phase != BluffPhaseChallenge || state.WinnerName != "" {
				t.Fatal("premature win")
			}
			if BluffAccept(state, "a") == nil || BluffAccept(state, "c") == nil {
				t.Fatal("only the next active player may accept")
			}
			var err error
			if outcome == "accept" || outcome == "accept bluff" {
				err = BluffAccept(state, "b")
			} else {
				err = BluffChallenge(state, "c")
			}
			if err != nil {
				t.Fatal(err)
			}
			if outcome == "liar" {
				if state.Phase != BluffPhasePlaying || state.WinnerName != "" || len(state.Players[0].Cards) != 2 {
					t.Fatal("caught last-card bluff won")
				}
			} else if state.Phase != BluffPhaseGameOver || state.WinnerName != "Alice" {
				t.Fatal("resolved last-card play did not win")
			}
		})
	}
}

func TestBluffSequentialRanksAndActiveTurn(t *testing.T) {
	state := bluffTestState()
	for rank := 1; rank <= 13; rank++ {
		actor := &state.Players[state.CurrentPlayerIdx]
		actor.Cards = append(actor.Cards, BluffCard{ID: fmt.Sprintf("turn_%d", rank), Rank: rank})
		if state.RequiredRank != rank {
			t.Fatalf("required rank %d, want %d", state.RequiredRank, rank)
		}
		if err := BluffPlay(state, actor.ID, []string{fmt.Sprintf("turn_%d", rank)}); err != nil {
			t.Fatal(err)
		}
		if err := BluffAccept(state, state.Players[state.CurrentPlayerIdx].ID); err != nil {
			t.Fatal(err)
		}
	}
	if state.RequiredRank != 1 {
		t.Fatal("King did not wrap to Ace")
	}
	state = bluffTestState()
	state.Players[1].Active = false
	if err := BluffPlay(state, "a", []string{"ace"}); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 2 || BluffChallenge(state, "b") == nil {
		t.Fatal("inactive player was not skipped")
	}
	if err := BluffAccept(state, "c"); err != nil {
		t.Fatal(err)
	}
}

func TestBluffExitPreservesCardsAndPendingClaim(t *testing.T) {
	for _, exitID := range []string{"a", "b", "c"} {
		t.Run(exitID, func(t *testing.T) {
			state := bluffTestState()
			before := bluffTestInventory(t, state)
			if err := BluffPlay(state, "a", []string{"ace"}); err != nil {
				t.Fatal(err)
			}
			if err := BluffExit(state, exitID); err != nil {
				t.Fatal(err)
			}
			if len(state.Players) != 2 || bluffFindPlayer(state, exitID) >= 0 || !reflect.DeepEqual(before, bluffTestInventory(t, state)) {
				t.Fatal("exit did not remove player and conserve cards")
			}
			if exitID == "a" {
				if state.Phase != BluffPhasePlaying || state.PendingActorID != "" || state.RequiredRank != 2 || state.Players[state.CurrentPlayerIdx].ID != "b" {
					t.Fatal("departed actor's claim was not cancelled")
				}
			} else {
				if err := BluffChallenge(state, state.Players[state.CurrentPlayerIdx].ID); err != nil {
					t.Fatal(err)
				}
				if len(state.Players[0].Cards) != 1 {
					t.Fatal("exit cards contaminated the pending claim's truth check")
				}
			}
		})
	}
	state := bluffTestState()
	state.Players = state.Players[:2]
	state.Players[0].Cards = state.Players[0].Cards[:1]
	if err := BluffPlay(state, "a", []string{"ace"}); err != nil {
		t.Fatal(err)
	}
	if err := BluffExit(state, "b"); err != nil {
		t.Fatal(err)
	}
	if state.Phase != BluffPhaseGameOver || state.WinnerName != "" {
		t.Fatal("exit must not award unresolved last-card win")
	}
}

func TestBluffExitAdjustsPlayingTurn(t *testing.T) {
	for _, test := range []struct {
		current int
		exitID  string
		nextID  string
	}{
		{0, "a", "b"},
		{1, "a", "b"},
		{1, "b", "c"},
		{2, "a", "c"},
		{2, "c", "a"},
	} {
		state := bluffTestState()
		state.CurrentPlayerIdx = test.current
		before := bluffTestInventory(t, state)
		if err := BluffExit(state, test.exitID); err != nil {
			t.Fatal(err)
		}
		if state.Players[state.CurrentPlayerIdx].ID != test.nextID || state.RequiredRank != 1 || !reflect.DeepEqual(before, bluffTestInventory(t, state)) {
			t.Fatalf("bad turn adjustment for %+v: %+v", test, state)
		}
		if BluffExit(state, test.exitID) == nil {
			t.Fatal("departed player allowed to exit twice")
		}
	}
}

func TestBluffPrivacyAndDetachedSnapshot(t *testing.T) {
	state := bluffTestState()
	if err := BluffPlay(state, "a", []string{"ace"}); err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []string{"a", "spectator"} {
		view := BluffSanitize(state, viewer)
		for _, player := range view.Players {
			for _, card := range player.Cards {
				if player.ID != viewer && card != (BluffCard{}) {
					t.Fatal("opponent card leaked")
				}
			}
		}
		for _, card := range view.Pile {
			if card != (BluffCard{}) {
				t.Fatal("pile card leaked")
			}
		}
		if view.pendingCards != nil || len(view.Pile) != 2 || len(view.Players[0].Cards) != 1 {
			t.Fatal("hidden metadata leaked or counts lost")
		}
		encoded, err := json.Marshal(view)
		if err != nil || strings.Contains(string(encoded), "ace") || strings.Contains(string(encoded), "old") || strings.Contains(string(encoded), "pendingCards") {
			t.Fatalf("private JSON leaked: %s (%v)", encoded, err)
		}
		view.Players[0].Cards[0].Rank = 99
		view.Pile[0].Rank = 99
	}
	if state.Players[0].Cards[0].Rank != 13 || state.Pile[0].Rank != 8 || state.pendingCards[0].Rank != 1 {
		t.Fatal("snapshot mutated authoritative cards")
	}
	if BluffSanitize(nil, "a") != nil || BluffPlay(nil, "a", nil) == nil || BluffAccept(nil, "a") == nil || BluffChallenge(nil, "a") == nil || BluffExit(nil, "a") == nil {
		t.Fatal("nil state handling failed")
	}
}
