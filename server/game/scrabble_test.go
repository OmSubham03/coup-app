package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func scrabbleFixture(t *testing.T, racks ...string) *ScrabbleState {
	t.Helper()
	var seats []ScrabblePlayer
	for i := range racks {
		seats = append(seats, ScrabblePlayer{ID: string(rune('a' + i))})
	}
	s, err := NewScrabble(seats)
	if err != nil {
		t.Fatal(err)
	}
	s.bag = strings.Split("EEEEEEEEEEEEEE", "")
	for i, r := range racks {
		s.racks[i] = strings.Split(r, "")
	}
	return s
}

func scrabbleLine(cell, delta int, word string) []ScrabblePlacement {
	var result []ScrabblePlacement
	for _, ch := range word {
		letter := string(ch)
		result = append(result, ScrabblePlacement{cell, strings.ToUpper(letter), letter != strings.ToUpper(letter)})
		cell += delta
	}
	return result
}

func TestScrabbleBoardAndBag(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 5} {
		seats := make([]ScrabblePlayer, n)
		for i := range seats {
			seats[i].ID = string(rune('a' + i))
		}
		s, err := NewScrabble(seats)
		if n < 2 || n > 4 {
			if err == nil {
				t.Fatal("invalid seats", n)
			}
			continue
		}
		if err != nil || len(s.bag) != 100-7*n {
			t.Fatalf("deal %d: %v", n, err)
		}
		tiles := slices.Clone(s.bag)
		for _, rack := range s.racks {
			if len(rack) != 7 {
				t.Fatal("rack size")
			}
			tiles = append(tiles, rack...)
		}
		slices.Sort(tiles)
		if got := strings.Join(tiles, ""); got != "??AAAAAAAAABBCCDDDDEEEEEEEEEEEEFFGGGHHIIIIIIIIIJKLLLLMMNNNNNNOOOOOOOOPPQRRRRRRSSSSTTTTTTUUUUVVWWXYYZ" {
			t.Fatalf("wrong distribution %s", got)
		}
	}
	counts := map[string]int{}
	for _, p := range scrabblePremiums {
		counts[p]++
	}
	if counts["DW"] != 17 || counts["DL"] != 24 || counts["TW"] != 8 || counts["TL"] != 12 || scrabblePremiums[112] != "DW" {
		t.Fatal(counts)
	}
}

func TestScrabblePlacementAndScoring(t *testing.T) {
	s := scrabbleFixture(t, "?ATERST", "TXXXXXX")
	// Seven new tiles, center DW and DL under the first blank (worth zero).
	if err := s.Play("a", scrabbleLine(108, 1, "rATTERS")); err != nil {
		t.Fatal(err)
	}
	if s.Phase == ScrabbleVoting {
		if err := s.Vote("b", s.Pending.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	if s.Players[0].Score != 62 || s.Board[108] != "r" || s.CurrentPlayerIdx != 1 || len(s.racks[0]) != 7 {
		t.Fatalf("bad bingo: %+v", s)
	}

	// Place at (6,6), DL, where both the horizontal and vertical words score.
	s.Board[95] = "A"
	p, err := s.evaluate(scrabbleLine(96, 1, "T"), func(string) bool { return false })
	if err != nil || p.score != 6 || !slices.Equal(p.Words, []string{"AT", "TT"}) {
		t.Fatalf("cross scores/words: %+v, %v", p, err)
	}

	s = scrabbleFixture(t, "ATCCCCC", "EEEEEEE")
	s.Board[112] = "n"
	p, err = s.evaluate([]ScrabblePlacement{{113, "T", false}, {111, "A", false}}, func(string) bool { return true })
	if err != nil || p.score != 2 {
		t.Fatalf("bridging an existing blank on a spent premium: %+v %v", p, err)
	}

	// A new tile on the top-left TW, next to a tile on the DL, and extending
	// through the center-top TW. Multipliers stack; old premiums are spent.
	s.Board[3] = "A"
	s.racks[0] = strings.Split("AAAAAAA", "")
	placements := append(scrabbleLine(0, 1, "AAA"), scrabbleLine(4, 1, "AAAA")...)
	p, err = s.evaluate(placements, func(string) bool { return true })
	if err != nil || p.score != 122 {
		t.Fatalf("stacked triples: %+v %v", p, err)
	}
}

func TestScrabbleIllegalMoves(t *testing.T) {
	for _, tc := range []struct {
		name       string
		board      map[int]string
		placements []ScrabblePlacement
	}{
		{"empty", nil, nil}, {"single", nil, scrabbleLine(112, 1, "A")}, {"off center", nil, scrabbleLine(0, 1, "AT")},
		{"occupied", map[int]string{112: "A"}, scrabbleLine(112, 1, "AT")},
		{"disconnected", map[int]string{112: "A"}, scrabbleLine(0, 1, "AT")},
		{"wrap", map[int]string{112: "A", 119: "A"}, scrabbleLine(120, 1, "AT")},
		{"diagonal", nil, scrabbleLine(112, 16, "AT")}, {"gap", nil, scrabbleLine(112, 2, "AT")},
		{"duplicate", nil, scrabbleLine(112, 0, "AT")}, {"overspend", nil, scrabbleLine(112, 1, "AA")},
		{"missing blank", nil, scrabbleLine(112, 1, "aT")}, {"lowercase input", nil, []ScrabblePlacement{{112, "a", false}}},
		{"negative", nil, scrabbleLine(-1, 1, "AT")}, {"off board", nil, scrabbleLine(224, 1, "AT")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := scrabbleFixture(t, "AT", "ME")
			for i, v := range tc.board {
				s.Board[i] = v
			}
			before := s.Board
			if err := s.Play("a", tc.placements); err == nil || s.Board != before || strings.Join(s.racks[0], "") != "AT" || s.Pending != nil {
				t.Fatal("invalid move accepted or state changed")
			}
		})
	}
}

func TestScrabbleProposalAtomicity(t *testing.T) {
	s := scrabbleFixture(t, "QZABCDE", "ABCDEFG", "ABCDEFG")
	bag := slices.Clone(s.bag)
	rack := slices.Clone(s.racks[0])
	s.scoreless = 2
	play := func() uint64 {
		t.Helper()
		if err := s.Play("a", scrabbleLine(112, 1, "QZ")); err != nil {
			t.Fatal(err)
		}
		p := s.Pending
		if p == nil || !slices.Equal(p.Approvals, []string{"a"}) || !slices.Equal(p.Words, []string{"QZ"}) {
			t.Fatal("bad proposal", p)
		}
		return p.ID
	}
	id := play()
	if s.Vote("a", id, false) == nil || s.Vote("watch", id, true) == nil || s.Pass("a") == nil || s.Exchange("a", []string{"Q"}) == nil {
		t.Fatal("illegal pending action")
	}
	if err := s.Vote("b", id, true); err != nil {
		t.Fatal(err)
	}
	if s.Vote("b", id, false) == nil || s.Pending == nil {
		t.Fatal("duplicate vote")
	}
	if err := s.Vote("c", id, false); err != nil {
		t.Fatal(err)
	}
	if s.Pending != nil || s.Phase != "playing" || s.Players[0].Score != 0 || s.Board != [225]string{} || !slices.Equal(s.bag, bag) || !slices.Equal(s.racks[0], rack) || s.scoreless != 2 || s.CurrentPlayerIdx != 0 {
		t.Fatal("rejection changed committed state")
	}
	next := play()
	if next == id || s.Vote("c", id, false) == nil {
		t.Fatal("accepted stale vote")
	}
	if err := s.Vote("b", next, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Vote("c", next, true); err != nil {
		t.Fatal(err)
	}
	if s.Pending != nil || s.Board[112] != "Q" || s.Players[0].Score != 40 || s.scoreless != 0 || s.CurrentPlayerIdx != 1 || len(s.bag) != 12 {
		t.Fatal("unanimous move did not commit once")
	}
}

func TestScrabbleExchangeAndEnd(t *testing.T) {
	s := scrabbleFixture(t, "ABCD?QZ", "M")
	if s.Exchange("b", []string{"M"}) == nil || s.Exchange("a", []string{"A", "A"}) == nil {
		t.Fatal("illegal exchange")
	}
	s.bag = s.bag[:6]
	if s.Exchange("a", []string{"Z"}) == nil {
		t.Fatal("small bag")
	}
	s.bag = append(s.bag, "E")
	if err := s.Exchange("a", []string{"?", "Z"}); err != nil {
		t.Fatal(err)
	}
	bag := slices.Clone(s.bag)
	slices.Sort(bag)
	if !reflect.DeepEqual(bag, strings.Split("?EEEEEZ", "")) || strings.Join(s.racks[0], "") != "ABCDQEE" {
		t.Fatal("incorrect exchange", bag, s.racks)
	}
	for i := 0; i < 5; i++ {
		if err := s.Pass(s.Players[s.CurrentPlayerIdx].ID); err != nil {
			t.Fatal(err)
		}
	}
	if s.Phase != "game_over" || s.Players[0].Score != -21 || s.Players[1].Score != -3 || s.Pass("a") == nil {
		t.Fatal("scoreless end/penalties", s.Players)
	}
	s = scrabbleFixture(t, "AT", "?J")
	s.bag = nil
	if err := s.Play("a", scrabbleLine(112, 15, "AT")); err != nil {
		t.Fatal(err)
	}
	if s.Phase != "game_over" || s.Players[0].Score != 12 || s.Players[1].Score != -8 {
		t.Fatal("rack out settlement", s.Players)
	}
}

func TestScrabbleLeaveCancelsVote(t *testing.T) {
	s := scrabbleFixture(t, "QZ", "BYE", "AT", "NO")
	if err := s.Play("a", scrabbleLine(112, 1, "QZ")); err != nil {
		t.Fatal(err)
	}
	id := s.Pending.ID
	s.Leave("b")
	if s.Players[1].Active || s.Pending != nil || s.CurrentPlayerIdx != 0 || s.Vote("c", id, true) == nil {
		t.Fatal("voter exit did not cancel")
	}
	s.Leave("a")
	if s.CurrentPlayerIdx != 2 || s.Phase != "playing" || !slices.Equal(s.racks[2], []string{"A", "T"}) {
		t.Fatal("incorrect proposer exit")
	}
	s.Leave("d")
	if s.Phase != "game_over" {
		t.Fatal("last player standing")
	}
}
