package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func blackjackTestCards(ranks ...int) []BlackjackCard {
	cards := make([]BlackjackCard, len(ranks))
	for cardIdx, rank := range ranks {
		cards[cardIdx] = BlackjackCard{ID: fmt.Sprintf("test_%d", cardIdx), Suit: "clubs", Rank: rank}
	}
	return cards
}

func blackjackTestRound(t *testing.T, playerCount int, ranks ...int) *BlackjackState {
	t.Helper()
	state := &BlackjackState{}
	for playerIdx := 0; playerIdx < playerCount; playerIdx++ {
		state.Players = append(state.Players, BlackjackPlayer{ID: fmt.Sprint(playerIdx), Name: fmt.Sprintf("Player %d", playerIdx), Chips: 1000, Active: true})
	}
	if err := blackjackDealRound(state, blackjackTestCards(ranks...)); err != nil {
		t.Fatal(err)
	}
	return state
}

func blackjackTestInventory(t *testing.T, state *BlackjackState) map[string]BlackjackCard {
	t.Helper()
	cards := make(map[string]BlackjackCard)
	all := append([]BlackjackCard{}, state.shoe...)
	all = append(all, state.DealerCards...)
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

func TestBlackjackScoreAcesAndNaturals(t *testing.T) {
	for _, test := range []struct {
		ranks   []int
		total   int
		natural bool
	}{
		{nil, 0, false},
		{[]int{1, 13}, 21, true},
		{[]int{12, 1}, 21, true},
		{[]int{1, 1, 9}, 21, false},
		{[]int{1, 1, 1, 8}, 21, false},
		{[]int{1, 6}, 17, false},
		{[]int{1, 6, 10}, 17, false},
		{[]int{1, 1, 10, 10}, 22, false},
		{[]int{10, 11, 12, 13}, 40, false},
	} {
		cards := blackjackTestCards(test.ranks...)
		if total := blackjackScore(cards); total != test.total || blackjackNatural(cards) != test.natural {
			t.Fatalf("ranks %v: total=%d natural=%v", test.ranks, total, blackjackNatural(cards))
		}
	}
}

func TestBlackjackInitializationAndDecks(t *testing.T) {
	for _, twoDecks := range []bool{false, true} {
		for playerCount := 1; playerCount <= 4; playerCount++ {
			players := map[string]string{}
			for playerIdx := 0; playerIdx < playerCount; playerIdx++ {
				players[fmt.Sprint(playerIdx)] = "Player"
			}
			state, err := BlackjackInitializeGame(players, twoDecks)
			if err != nil {
				t.Fatal(err)
			}
			want := 52
			if twoDecks {
				want = 104
			}
			inventory := blackjackTestInventory(t, state)
			if len(inventory) != want || state.Round != 1 || state.TwoDecks != twoDecks || !strings.Contains(state.LastAction, "fresh shuffled") {
				t.Fatalf("incorrect initial state: %+v", state)
			}
			counts := map[int]int{}
			for _, card := range inventory {
				counts[card.Rank]++
			}
			for rank := 1; rank <= 13; rank++ {
				if counts[rank] != want/13 {
					t.Fatalf("rank %d count=%d", rank, counts[rank])
				}
			}
			for _, player := range state.Players {
				if len(player.Cards) != 2 || player.Bet != 50 || !player.Active {
					t.Fatalf("incorrect initial player: %+v", player)
				}
				if state.Phase == BlackjackPhasePlaying && player.Chips != 950 {
					t.Fatal("opening wager not deducted")
				}
			}
		}
	}
	if _, err := BlackjackInitializeGame(nil, false); err == nil {
		t.Fatal("accepted zero players")
	}
	if _, err := BlackjackInitializeGame(map[string]string{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E"}, false); err == nil {
		t.Fatal("accepted five players")
	}
	if _, err := BlackjackInitializeGame(map[string]string{"": "Empty"}, false); err == nil {
		t.Fatal("accepted empty player ID")
	}
}

func TestBlackjackInitialDealerNatural(t *testing.T) {
	state := blackjackTestRound(t, 2, 1, 9, 1, 13, 8, 10, 7)
	if state.Phase != BlackjackPhaseRoundOver || len(state.DealerCards) != 2 || len(state.shoe) != 1 || state.CurrentPlayerIdx != -1 {
		t.Fatal("dealer natural did not settle immediately")
	}
	if state.Players[0].Chips != 1000 || state.Players[0].Status != BlackjackStatusBlackjack || state.Players[1].Chips != 950 {
		t.Fatalf("dealer natural payout wrong: %+v", state.Players)
	}
	if BlackjackHit(state, "1") == nil || BlackjackStand(state, "1") == nil || BlackjackDouble(state, "1") == nil {
		t.Fatal("action allowed after dealer blackjack")
	}
}

func TestBlackjackPlayerNaturalPaysThreeToTwo(t *testing.T) {
	state := blackjackTestRound(t, 1, 1, 10, 13, 7)
	if state.Phase != BlackjackPhaseRoundOver || state.Players[0].Chips != 1075 || state.Players[0].Status != BlackjackStatusBlackjack {
		t.Fatalf("natural must profit 75: %+v", state)
	}
	state = blackjackTestRound(t, 1, 10, 10, 5, 7, 6)
	if err := BlackjackHit(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.Players[0].Status != BlackjackStatusStood || state.Players[0].Chips != 1050 {
		t.Fatal("three-card 21 must be an ordinary win, not a natural")
	}
	state = blackjackTestRound(t, 2, 1, 9, 10, 13, 8, 7)
	if state.Players[0].Status != BlackjackStatusBlackjack || state.CurrentPlayerIdx != 1 || state.Players[0].Chips != 950 {
		t.Fatal("natural not skipped or paid before round settlement")
	}
	if err := BlackjackStand(state, "1"); err != nil {
		t.Fatal(err)
	}
	if state.Players[0].Chips != 1075 || state.Players[1].Chips != 1000 {
		t.Fatal("multi-player natural/push settlement incorrect")
	}
}

func TestBlackjackHitRetainsTurnAndCannotDoubleAfterward(t *testing.T) {
	state := blackjackTestRound(t, 2, 5, 9, 10, 6, 8, 7, 2, 8)
	before := blackjackTestInventory(t, state)
	if err := BlackjackHit(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.Phase != BlackjackPhasePlaying || state.CurrentPlayerIdx != 0 || state.Players[0].Status != BlackjackStatusActive || blackjackScore(state.Players[0].Cards) != 13 || state.Players[0].Chips != 950 {
		t.Fatal("nonterminal hit did not retain turn and wager")
	}
	if BlackjackDouble(state, "0") == nil || !reflect.DeepEqual(before, blackjackTestInventory(t, state)) {
		t.Fatal("double after hit accepted or deck changed")
	}
	if err := BlackjackHit(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 1 || state.Players[0].Status != BlackjackStatusStood || len(state.Players[0].Cards) != 4 {
		t.Fatal("21 did not end turn")
	}
	if err := BlackjackStand(state, "1"); err != nil {
		t.Fatal(err)
	}
	if state.Players[0].Chips != 1050 || state.Players[1].Chips != 1000 {
		t.Fatal("hit/stand round settlement incorrect")
	}
}

func TestBlackjackNaturalBeatsDealerDrawnTwentyOne(t *testing.T) {
	state := blackjackTestRound(t, 1, 1, 10, 13, 6, 5)
	if state.Phase != BlackjackPhaseRoundOver || blackjackScore(state.DealerCards) != 21 || len(state.DealerCards) != 3 || state.Players[0].Chips != 1075 {
		t.Fatal("natural must beat dealer's non-natural 21")
	}
}

func TestBlackjackDealerStandsOnAllSeventeen(t *testing.T) {
	for _, dealerRanks := range [][]int{{10, 7}, {1, 6}} {
		state := blackjackTestRound(t, 1, 10, dealerRanks[0], 8, dealerRanks[1], 10)
		if err := BlackjackStand(state, "0"); err != nil {
			t.Fatal(err)
		}
		if len(state.DealerCards) != 2 || len(state.shoe) != 1 || state.Players[0].Chips != 1050 {
			t.Fatal("dealer hit hard or soft 17")
		}
	}
	state := blackjackTestRound(t, 1, 10, 1, 8, 5, 10, 1)
	if err := BlackjackStand(state, "0"); err != nil {
		t.Fatal(err)
	}
	if blackjackScore(state.DealerCards) != 17 || len(state.DealerCards) != 4 || state.Players[0].Chips != 1050 {
		t.Fatal("dealer did not adjust soft aces and hit through 16")
	}
}

func TestBlackjackRegularPayouts(t *testing.T) {
	for _, test := range []struct {
		name  string
		ranks []int
		chips int
	}{
		{"win", []int{10, 10, 9, 8}, 1050},
		{"push", []int{10, 10, 8, 8}, 1000},
		{"loss", []int{10, 10, 7, 8}, 950},
		{"dealer bust", []int{10, 10, 7, 6, 10}, 1050},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := blackjackTestRound(t, 1, test.ranks...)
			if err := BlackjackStand(state, "0"); err != nil {
				t.Fatal(err)
			}
			if state.Players[0].Chips != test.chips || state.Phase != BlackjackPhaseRoundOver {
				t.Fatalf("incorrect payout: %+v", state.Players[0])
			}
			if BlackjackStand(state, "0") == nil || state.Players[0].Chips != test.chips {
				t.Fatal("settlement repeated")
			}
		})
	}
	state := blackjackTestRound(t, 1, 10, 10, 9, 6, 10, 10)
	if err := BlackjackHit(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.Players[0].Chips != 950 || state.Players[0].Status != BlackjackStatusBust || blackjackScore(state.DealerCards) <= 21 {
		t.Fatal("player bust must lose even when dealer also busts")
	}
}

func TestBlackjackDouble(t *testing.T) {
	for _, test := range []struct {
		name  string
		ranks []int
		chips int
		bust  bool
	}{
		{"win", []int{5, 10, 6, 8, 10}, 1100, false},
		{"loss", []int{5, 10, 6, 8, 2}, 900, false},
		{"push", []int{5, 10, 6, 8, 7}, 1000, false},
		{"bust", []int{10, 10, 9, 8, 5}, 900, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := blackjackTestRound(t, 1, test.ranks...)
			if err := BlackjackDouble(state, "0"); err != nil {
				t.Fatal(err)
			}
			player := state.Players[0]
			if player.Bet != 100 || len(player.Cards) != 3 || player.Chips != test.chips || (player.Status == BlackjackStatusBust) != test.bust || state.Phase != BlackjackPhaseRoundOver {
				t.Fatalf("incorrect double: %+v", player)
			}
		})
	}
	state := blackjackTestRound(t, 2, 5, 9, 10, 6, 8, 8, 10)
	if err := BlackjackDouble(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.CurrentPlayerIdx != 1 || state.Players[0].Chips != 900 || state.Players[0].Status != BlackjackStatusStood || BlackjackHit(state, "0") == nil {
		t.Fatal("double must draw exactly once, deduct extra wager, and end turn")
	}
}

func TestBlackjackInvalidActionsAreAtomic(t *testing.T) {
	for _, test := range []struct {
		name   string
		setup  func(*BlackjackState)
		action func(*BlackjackState) error
	}{
		{"wrong turn", func(*BlackjackState) {}, func(state *BlackjackState) error { return BlackjackHit(state, "1") }},
		{"unknown", func(*BlackjackState) {}, func(state *BlackjackState) error { return BlackjackStand(state, "unknown") }},
		{"poor double", func(state *BlackjackState) { state.Players[0].Chips = 49 }, func(state *BlackjackState) error { return BlackjackDouble(state, "0") }},
		{"double after hit", func(state *BlackjackState) {
			state.Players[0].Cards = append(state.Players[0].Cards, BlackjackCard{Rank: 2})
		}, func(state *BlackjackState) error { return BlackjackDouble(state, "0") }},
		{"empty shoe", func(state *BlackjackState) { state.shoe = nil }, func(state *BlackjackState) error { return BlackjackHit(state, "0") }},
		{"premature next round", func(*BlackjackState) {}, func(state *BlackjackState) error { return BlackjackNextRound(state, "0") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := blackjackTestRound(t, 2, 5, 9, 10, 6, 8, 8, 2)
			test.setup(state)
			before, _ := json.Marshal(state)
			shoe := append([]BlackjackCard{}, state.shoe...)
			if test.action(state) == nil {
				t.Fatal("invalid action accepted")
			}
			after, _ := json.Marshal(state)
			if string(before) != string(after) || len(shoe) != len(state.shoe) || (len(shoe) > 0 && !reflect.DeepEqual(shoe, state.shoe)) {
				t.Fatal("invalid action mutated state")
			}
		})
	}
}

func TestBlackjackFreshRoundAndBrokePlayers(t *testing.T) {
	for _, twoDecks := range []bool{false, true} {
		state := blackjackTestRound(t, 2, 10, 10, 10, 8, 7, 8)
		state.TwoDecks = twoDecks
		if err := BlackjackStand(state, "0"); err != nil {
			t.Fatal(err)
		}
		if err := BlackjackStand(state, "1"); err != nil {
			t.Fatal(err)
		}
		state.Players[1].Chips = 49
		state.shoe = []BlackjackCard{{ID: "stale-shoe", Rank: 3}}
		before := state.Players[0].Chips
		if BlackjackNextRound(state, "unknown") == nil {
			t.Fatal("unknown player started round")
		}
		if err := BlackjackNextRound(state, "0"); err != nil {
			t.Fatal(err)
		}
		want := 52
		if twoDecks {
			want = 104
		}
		inventory := blackjackTestInventory(t, state)
		if len(inventory) != want || state.Round != 2 || state.Players[1].Active || len(state.Players[1].Cards) != 0 || state.Players[1].Bet != 0 {
			t.Fatal("round did not reset shoe and skip broke player")
		}
		if _, stale := inventory["stale-shoe"]; stale || !strings.Contains(state.LastAction, "fresh shuffled") {
			t.Fatal("old shoe survived or fresh-shoe label missing")
		}
		if state.Phase == BlackjackPhasePlaying && state.Players[0].Chips != before-50 {
			t.Fatal("new fixed wager not deducted")
		}
	}
	state := blackjackTestRound(t, 1, 10, 10, 8, 8)
	if err := BlackjackStand(state, "0"); err != nil {
		t.Fatal(err)
	}
	state.Players[0].Chips = 49
	if err := BlackjackNextRound(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.Phase != BlackjackPhaseGameOver || state.CurrentPlayerIdx != -1 || state.Players[0].Active {
		t.Fatal("bankrupt game did not terminate")
	}
}

func TestBlackjackExit(t *testing.T) {
	for _, exitID := range []string{"0", "1"} {
		state := blackjackTestRound(t, 2, 5, 9, 10, 6, 8, 8)
		before := blackjackTestInventory(t, state)
		if err := BlackjackExit(state, exitID); err != nil {
			t.Fatal(err)
		}
		playerIdx := 0
		remainingID := "1"
		if exitID == "1" {
			playerIdx = 1
			remainingID = "0"
		}
		if state.Players[playerIdx].Active || state.Players[playerIdx].Chips != 950 || state.Players[playerIdx].Bet != 0 || !reflect.DeepEqual(before, blackjackTestInventory(t, state)) {
			t.Fatal("exit failed to forfeit bet, retain deck, or deactivate player")
		}
		if state.Players[state.CurrentPlayerIdx].ID != remainingID || BlackjackHit(state, exitID) == nil || BlackjackExit(state, exitID) == nil {
			t.Fatal("exit left invalid turn or allowed departed player to act")
		}
		if err := BlackjackStand(state, remainingID); err != nil {
			t.Fatal(err)
		}
		if state.Phase != BlackjackPhaseRoundOver || state.Players[playerIdx].Chips != 950 {
			t.Fatal("exited wager was paid")
		}
		if err := BlackjackExit(state, remainingID); err != nil {
			t.Fatal(err)
		}
		if state.Phase != BlackjackPhaseGameOver || state.CurrentPlayerIdx != -1 {
			t.Fatal("last exit did not end game")
		}
	}
	state := blackjackTestRound(t, 1, 5, 10, 6, 8)
	if err := BlackjackExit(state, "0"); err != nil {
		t.Fatal(err)
	}
	if state.Phase != BlackjackPhaseGameOver || state.Players[0].Chips != 950 {
		t.Fatal("last player exit did not forfeit outstanding bet")
	}
}

func TestBlackjackPrivacyAndDetachedSnapshot(t *testing.T) {
	state := blackjackTestRound(t, 2, 5, 9, 10, 6, 8, 7, 2)
	for _, viewer := range []string{"0", "1", "spectator"} {
		view := BlackjackSanitize(state, viewer)
		if view.DealerCards[0] != state.DealerCards[0] || view.DealerCards[1] != (BlackjackCard{}) || view.shoe != nil || !reflect.DeepEqual(view.Players, state.Players) {
			t.Fatal("playing privacy incorrect")
		}
		encoded, err := json.Marshal(view)
		if err != nil || strings.Contains(string(encoded), state.DealerCards[1].ID) || strings.Contains(string(encoded), state.shoe[0].ID) || strings.Contains(string(encoded), "shoe\"") {
			t.Fatalf("hidden cards leaked: %s (%v)", encoded, err)
		}
		view.Players[0].Cards[0].Rank = 99
		view.DealerCards[0].Rank = 99
	}
	if state.Players[0].Cards[0].Rank != 5 || state.DealerCards[0].Rank != 10 {
		t.Fatal("snapshot mutated authoritative state")
	}
	encoded, _ := json.Marshal(state)
	if strings.Contains(string(encoded), state.shoe[0].ID) {
		t.Fatal("internal shoe serialized in authoritative JSON")
	}
	if err := BlackjackStand(state, "0"); err != nil {
		t.Fatal(err)
	}
	if err := BlackjackStand(state, "1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(BlackjackSanitize(state, "spectator").DealerCards, state.DealerCards) {
		t.Fatal("dealer hole card not revealed at round end")
	}
	if BlackjackSanitize(nil, "0") != nil || BlackjackHit(nil, "0") == nil || BlackjackStand(nil, "0") == nil || BlackjackDouble(nil, "0") == nil || BlackjackNextRound(nil, "0") == nil || BlackjackExit(nil, "0") == nil {
		t.Fatal("nil state handling failed")
	}
}
