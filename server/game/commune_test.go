package game

import "testing"

func TestCommuneNextOpener(t *testing.T) {
	for _, tc := range []struct {
		name        string
		rank        int
		callerLives int
		wantOpener  int
	}{
		{"caller loses", 14, 5, 0},
		{"declarer loses", 9, 5, 2},
		{"eliminated caller is skipped", 14, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := InitializeCommuneGame([]struct{ ID, Name string }{{"a", "A"}, {"b", "B"}, {"c", "C"}})
			s.CurrentPlayerIdx = 2
			s.AllCards = []CommuneCard{{Rank: tc.rank}, {Rank: tc.rank}}
			s.Players[0].Tokens = tc.callerLives
			if err := CommuneDeclare(s, "c", CommuneDeclaration{HandType: HandOnePair, PrimaryRank: 14}); err != nil {
				t.Fatal(err)
			}
			if err := CommuneCall(s, "a"); err != nil {
				t.Fatal(err)
			}
			CommuneNextHand(s)
			if s.Round != 2 || s.CurrentPlayerIdx != tc.wantOpener {
				t.Fatalf("round %d starts at seat %d; want round 2, seat %d", s.Round, s.CurrentPlayerIdx, tc.wantOpener)
			}
		})
	}
}
