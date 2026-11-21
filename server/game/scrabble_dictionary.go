package game

import (
	_ "embed"
	"sort"
	"strings"
)

//go:embed enable1.txt
var enableWords string

var scrabbleLexicon = strings.Fields(enableWords)

func scrabbleKnown(word string) bool {
	word = strings.ToLower(word)
	i := sort.SearchStrings(scrabbleLexicon, word)
	return i < len(scrabbleLexicon) && scrabbleLexicon[i] == word
}
