package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func kaboFixture(t *testing.T, n int) *KaboState {
	t.Helper()
	players := []KaboPlayer{}
	for i := 0; i < n; i++ {
		players = append(players, KaboPlayer{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("Player %d", i)})
	}
	s, err := NewKabo(players)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func kaboReadyAll(t *testing.T, s *KaboState) {
	t.Helper()
	for _, p := range s.Players {
		if err := s.InitialPeek(p.ID, []string{p.Slots[0].ID, p.Slots[1].ID}); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(p.ID); err != nil {
			t.Fatal(err)
		}
	}
}
func kaboDo(t *testing.T, s *KaboState, id, kind string, a KaboAction) {
	t.Helper()
	a.Revision = s.Revision
	if err := s.Apply(id, kind, a); err != nil {
		t.Fatalf("%s: %v", kind, err)
	}
}
func kaboJSON(t *testing.T, s *KaboState, id string) string {
	t.Helper()
	b, err := json.Marshal(s.View(id))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestKaboSetupAndPrivateViews(t *testing.T) {
	for n := 2; n <= 6; n++ {
		s := kaboFixture(t, n)
		if len(s.deck)+len(s.discards)+4*n != 52 {
			t.Fatal("bad deal")
		}
		for _, p := range s.Players {
			if len(p.Slots) != 4 {
				t.Fatal("bad hand")
			}
		}
		p := s.Players[0]
		if s.InitialPeek(p.ID, []string{p.Slots[0].ID, p.Slots[0].ID}) == nil {
			t.Fatal("duplicate peek")
		}
		if s.InitialPeek(p.ID, []string{p.Slots[0].ID, s.Players[1].Slots[0].ID}) == nil {
			t.Fatal("opponent initial peek")
		}
		if err := s.InitialPeek(p.ID, []string{p.Slots[0].ID, p.Slots[1].ID}); err != nil {
			t.Fatal(err)
		}
		own := kaboJSON(t, s, p.ID)
		other := kaboJSON(t, s, "p1")
		spectator := kaboJSON(t, s, "")
		if !strings.Contains(own, `"reveals":[{`) || strings.Contains(other, `"reveals":[{`) || strings.Contains(spectator, `"reveals":[{`) {
			t.Fatal("initial privacy")
		}
		if s.InitialPeek(p.ID, []string{p.Slots[2].ID, p.Slots[3].ID}) == nil {
			t.Fatal("repeated initial peek")
		}
		if err := s.Ready(p.ID); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(kaboJSON(t, s, p.ID), `"reveals":[{`) {
			t.Fatal("remembered cards persisted")
		}
		for _, q := range s.Players[1:] {
			s.InitialPeek(q.ID, []string{q.Slots[0].ID, q.Slots[1].ID})
			s.Ready(q.ID)
		}
		if s.Phase != KaboDraw {
			t.Fatal("not ready")
		}
		for _, c := range []string{`"deck"`, `"card"`, `"initial"`, `"roundScore"`} {
			if strings.Contains(kaboJSON(t, s, ""), c+":") {
				t.Fatal("private storage serialized")
			}
		}
	}
}

func TestKaboPowersAndDiscardSource(t *testing.T) {
	for _, card := range []KaboCard{{7, "clubs"}, {8, "hearts"}, {9, "clubs"}, {10, "spades"}, {11, "hearts"}, {12, "clubs"}, {13, "spades"}, {13, "hearts"}, {4, "clubs"}} {
		s := kaboFixture(t, 2)
		kaboReadyAll(t, s)
		s.deck = append([]KaboCard{card}, s.deck...)
		kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
		if strings.Contains(kaboJSON(t, s, "p1"), `"drawn":{`) {
			t.Fatal("drawn card leak")
		}
		own, target := s.Players[0].Slots[0], s.Players[1].Slots[0]
		kaboDo(t, s, "p0", "discard", KaboAction{})
		if card.Rank < 7 || (card.Rank == 13 && card.Suit == "hearts") {
			if s.Phase != KaboDraw {
				t.Fatal("unexpected power")
			}
			continue
		}
		a := KaboAction{SlotID: own.ID, TargetID: target.ID}
		if card.Rank == 7 || card.Rank == 8 {
			a.TargetID = ""
		}
		if card.Rank == 9 || card.Rank == 10 {
			a.SlotID = ""
		}
		kaboDo(t, s, "p0", "power", a)
		if card.Rank == 11 || card.Rank == 12 {
			if s.Players[0].Slots[0].card != target.card || s.Players[1].Slots[0].card != own.card || s.Phase != KaboDraw {
				t.Fatal("blind swap")
			}
		} else {
			if s.Phase != KaboPeek || strings.Contains(kaboJSON(t, s, "p1"), `"reveals":[{`) || !strings.Contains(kaboJSON(t, s, "p0"), `"reveals":[{`) {
				t.Fatal("peek privacy")
			}
			kaboDo(t, s, "p0", "ack", KaboAction{})
			if strings.Contains(kaboJSON(t, s, "p0"), `"reveals":[{`) {
				t.Fatal("peek remained visible")
			}
			if card.Rank == 13 {
				if s.Phase != KaboDecision {
					t.Fatal("no king decision")
				}
				kaboDo(t, s, "p0", "decision", KaboAction{Swap: true})
				if s.Players[0].Slots[0].card != target.card {
					t.Fatal("king swap")
				}
			}
			if s.Phase != KaboDraw {
				t.Fatal("power did not finish")
			}
		}
	}
	s := kaboFixture(t, 2)
	kaboReadyAll(t, s)
	s.discards = []KaboCard{{13, "spades"}}
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "discard"})
	if s.Apply("p0", "discard", KaboAction{Revision: s.Revision}) == nil {
		t.Fatal("discard-taken card discarded")
	}
	kaboDo(t, s, "p0", "replace", KaboAction{SlotID: s.Players[0].Slots[0].ID})
	if s.Power != "" || s.Phase != KaboDraw {
		t.Fatal("discard power activated")
	}
}

func TestKaboMatchingPenaltyTransferAndRace(t *testing.T) {
	s := kaboFixture(t, 3)
	kaboReadyAll(t, s)
	s.deck = append([]KaboCard{{5, "clubs"}}, s.deck...)
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p0", "discard", KaboAction{})
	disc := s.DiscardID
	s.Players[2].Slots[0].card = KaboCard{6, "clubs"}
	a := KaboAction{TargetID: s.Players[2].Slots[0].ID, SlotID: s.Players[0].Slots[0].ID, DiscardID: disc}
	kaboDo(t, s, "p0", "match", a)
	if len(s.Players[0].Slots) != 5 || !s.MatchOpen || s.DiscardID != disc || s.Players[2].Slots[0].card.Rank != 6 {
		t.Fatal("bad penalty")
	}
	if s.Apply("p0", "match", KaboAction{Revision: s.Revision, TargetID: a.TargetID, SlotID: a.SlotID, DiscardID: disc}) == nil {
		t.Fatal("repeat penalty spam")
	}
	s.Players[2].Slots[0].card = KaboCard{5, "hearts"}
	give := s.Players[1].Slots[0].card
	rev := s.Revision
	kaboDo(t, s, "p1", "match", KaboAction{TargetID: a.TargetID, SlotID: s.Players[1].Slots[0].ID, DiscardID: disc})
	if s.Players[1].cardCount() != 3 || len(s.Players[2].Slots) != 4 || s.Players[2].Slots[0].card != give || s.MatchOpen {
		t.Fatal("transfer incorrect")
	}
	if s.Players[1].Slots[0].card.Rank != 0 || len(s.Players[1].Slots) != 4 {
		t.Fatal("matching shifted the remaining card positions")
	}
	if s.Apply("p2", "match", KaboAction{Revision: rev, TargetID: s.Players[2].Slots[1].ID, DiscardID: disc}) == nil {
		t.Fatal("stale race accepted")
	}
	if s.Apply("p2", "match", KaboAction{Revision: s.Revision, TargetID: s.Players[2].Slots[1].ID, DiscardID: s.DiscardID}) == nil {
		t.Fatal("match chain allowed")
	}
}

func TestKaboCallFinalTurnsNegativeScoresAndRedeal(t *testing.T) {
	s := kaboFixture(t, 3)
	kaboReadyAll(t, s)
	for i := range s.Players {
		s.Players[i].Slots = []KaboSlot{{ID: uuidForTest(i), card: KaboCard{13, "hearts"}}}
	}
	s.deck = append([]KaboCard{{2, "clubs"}, {3, "clubs"}, {4, "clubs"}}, s.deck...)
	if s.Apply("p0", "call", KaboAction{Revision: s.Revision}) == nil {
		t.Fatal("called before draw")
	}
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p0", "call", KaboAction{})
	if s.FinalTurns != 2 || s.Phase != KaboResolve {
		t.Fatal("call consumed draw")
	}
	kaboDo(t, s, "p0", "discard", KaboAction{})
	if s.FinalTurns != 2 {
		t.Fatal("caller counted as final")
	}
	kaboDo(t, s, "p1", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p1", "discard", KaboAction{})
	if s.FinalTurns != 1 {
		t.Fatal("final turn not consumed")
	}
	kaboDo(t, s, "p2", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p2", "discard", KaboAction{})
	if s.Phase != KaboRoundOver || len(s.RoundWinners) != 3 {
		t.Fatal("round did not finish")
	}
	for _, p := range s.Players {
		if p.TotalScore != -1 || p.RoundScore != -1 {
			t.Fatal("negative or caller score wrong")
		}
	}
	oldID := s.Players[0].Slots[0].ID
	if err := s.NextRound(); err != nil {
		t.Fatal(err)
	}
	if s.Round != 2 || s.Players[0].TotalScore != -1 || len(s.Players[0].Slots) != 4 || s.Players[0].Slots[0].ID == oldID {
		t.Fatal("redeal lost totals")
	}
	kaboReadyAll(t, s)
	s.finishRound()
	if err := s.EndGame(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != KaboGameOver || len(s.Winners) == 0 {
		t.Fatal("manual end")
	}
}
func uuidForTest(i int) string { return fmt.Sprintf("slot-%d", i) }

func TestKaboEmptyHandAndScoreLimit(t *testing.T) {
	s := kaboFixture(t, 2)
	kaboReadyAll(t, s)
	s.Phase = KaboDraw
	s.MatchOpen = true
	s.discards = []KaboCard{{4, "clubs"}}
	s.Players[0].Slots = []KaboSlot{{ID: "last", card: KaboCard{4, "hearts"}}}
	kaboDo(t, s, "p0", "match", KaboAction{TargetID: "last", DiscardID: s.DiscardID})
	if s.Phase != KaboRoundOver || s.Players[0].RoundScore != 0 {
		t.Fatal("empty hand did not end round")
	}
	s = kaboFixture(t, 2)
	kaboReadyAll(t, s)
	s.Players[0].TotalScore = 1000
	s.finishRound()
	if s.Phase != KaboGameOver {
		t.Fatal("score threshold not applied")
	}
}

func TestKaboInvalidActionsAndDepartures(t *testing.T) {
	s := kaboFixture(t, 3)
	kaboReadyAll(t, s)
	for _, a := range []struct {
		id, kind string
		a        KaboAction
	}{
		{"watcher", "draw", KaboAction{Source: "deck"}}, {"p1", "draw", KaboAction{Source: "deck"}},
		{"p0", "draw", KaboAction{Source: "invalid"}}, {"p0", "replace", KaboAction{SlotID: "missing"}},
		{"p0", "ack", KaboAction{}}, {"p0", "decision", KaboAction{}},
	} {
		a.a.Revision = s.Revision
		if s.Apply(a.id, a.kind, a.a) == nil {
			t.Fatal("invalid action accepted")
		}
	}
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
	s.Leave("p0")
	if s.Phase != KaboDraw || s.Players[s.CurrentPlayerIdx].ID != "p1" || s.Players[0].Active {
		t.Fatal("departure stranded turn")
	}
	s.Leave("p1")
	if s.Phase != KaboGameOver || len(s.Winners) != 1 || s.Winners[0] != "p2" {
		t.Fatal("last player not handled")
	}
}

func TestKaboFinalTurnsSurviveDepartureOfAlreadyPlayedPlayer(t *testing.T) {
	s := kaboFixture(t, 4)
	kaboReadyAll(t, s)
	s.deck = append([]KaboCard{{2, "clubs"}, {3, "clubs"}, {4, "clubs"}, {5, "clubs"}}, s.deck...)
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p0", "call", KaboAction{})
	kaboDo(t, s, "p0", "discard", KaboAction{})
	kaboDo(t, s, "p1", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p1", "discard", KaboAction{})
	if s.FinalTurns != 2 {
		t.Fatal("wrong remaining turns")
	}
	s.Leave("p1")
	if s.FinalTurns != 2 {
		t.Fatal("departure of finished player consumed another player's final turn")
	}
	kaboDo(t, s, "p2", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p2", "discard", KaboAction{})
	kaboDo(t, s, "p3", "draw", KaboAction{Source: "deck"})
	kaboDo(t, s, "p3", "discard", KaboAction{})
	if s.Phase != KaboRoundOver {
		t.Fatal("round failed to finish")
	}
}

func TestKaboDeckRecyclingAndRankMatch(t *testing.T) {
	s := kaboFixture(t, 2)
	kaboReadyAll(t, s)
	s.deck = nil
	s.discards = []KaboCard{{3, "clubs"}, {4, "hearts"}, {13, "spades"}}
	kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
	if len(s.deck) != 1 || s.discards[0] != (KaboCard{13, "spades"}) {
		t.Fatal("top discard recycled")
	}
	kaboDo(t, s, "p0", "discard", KaboAction{})
	s.discards = []KaboCard{{13, "spades"}}
	s.MatchOpen = true
	s.Players[1].Slots[0].card = KaboCard{13, "hearts"}
	kaboDo(t, s, "p1", "match", KaboAction{TargetID: s.Players[1].Slots[0].ID, DiscardID: s.DiscardID})
	if s.Players[1].cardCount() != 3 {
		t.Fatal("red King failed to match black King by rank")
	}
}

func TestKaboKingKeepAndTargetDeparture(t *testing.T) {
	for _, leave := range []bool{false, true} {
		s := kaboFixture(t, 3)
		kaboReadyAll(t, s)
		s.deck = append([]KaboCard{{13, "clubs"}}, s.deck...)
		before := s.Players[0].Slots[0].card
		kaboDo(t, s, "p0", "draw", KaboAction{Source: "deck"})
		kaboDo(t, s, "p0", "discard", KaboAction{})
		kaboDo(t, s, "p0", "power", KaboAction{SlotID: s.Players[0].Slots[0].ID, TargetID: s.Players[1].Slots[0].ID})
		if leave {
			s.Leave("p1")
		} else {
			kaboDo(t, s, "p0", "ack", KaboAction{})
			kaboDo(t, s, "p0", "decision", KaboAction{Swap: false})
		}
		if s.Players[0].Slots[0].card != before || s.Phase != KaboDraw || len(s.reveals) != 0 {
			t.Fatal("king keep/departure did not resolve cleanly")
		}
	}
}
