package game

import (
	"sort"
	"testing"
)

func TestScrabbleLexicon(t *testing.T) {
	if !sort.StringsAreSorted(scrabbleLexicon) || !scrabbleKnown("QUIZ") || !scrabbleKnown("at") || scrabbleKnown("ZZZZZZZ") {
		t.Fatal("embedded word list must support case-insensitive lookups of real words")
	}
}
