package game

import (
	"errors"
	"slices"
	"strings"
)

// step stays within a row on horizontal moves, including at the board edges.
func scrabbleStep(cell, delta int) int {
	next := cell + delta
	if next < 0 || next >= 225 || ((delta == 1 || delta == -1) && next/15 != cell/15) {
		return -1
	}
	return next
}

// evaluate stages a complete move. Nothing in the live state changes until commit.
func (s *ScrabbleState) evaluate(placements []ScrabblePlacement, known func(string) bool) (*ScrabbleProposal, error) {
	if len(placements) == 0 || len(placements) > 7 {
		return nil, errors.New("place 1 to 7 tiles")
	}
	board := s.Board
	rack := slices.Clone(s.racks[s.CurrentPlayerIdx])
	first := placements[0].Cell
	lo, hi := first, first
	sameRow, sameColumn, connected := true, true, false
	for _, v := range placements {
		if v.Cell < 0 || v.Cell >= 225 || board[v.Cell] != "" {
			return nil, errors.New("placement must use an empty cell")
		}
		if len(v.Letter) != 1 || v.Letter[0] < 'A' || v.Letter[0] > 'Z' {
			return nil, errors.New("letter must be A to Z")
		}
		tile, face := v.Letter, v.Letter
		if v.Blank {
			tile, face = "?", strings.ToLower(face)
		}
		idx := slices.Index(rack, tile)
		if idx < 0 {
			return nil, errors.New("tile not in rack")
		}
		rack = slices.Delete(rack, idx, idx+1)
		board[v.Cell] = face
		sameRow = sameRow && v.Cell/15 == first/15
		sameColumn = sameColumn && v.Cell%15 == first%15
		lo, hi = min(lo, v.Cell), max(hi, v.Cell)
		for _, delta := range []int{-15, -1, 1, 15} {
			if n := scrabbleStep(v.Cell, delta); n >= 0 && s.Board[n] != "" {
				connected = true
			}
		}
	}
	if !sameRow && !sameColumn {
		return nil, errors.New("tiles must lie on one line")
	}
	if s.Board[112] == "" {
		if board[112] == "" {
			return nil, errors.New("first move must cover the center")
		}
	} else if !connected {
		return nil, errors.New("move must connect to the board")
	}
	main, cross := 1, 15
	if !sameRow {
		main, cross = 15, 1
	}
	for c := lo; c <= hi; c += main {
		if board[c] == "" {
			return nil, errors.New("word contains a gap")
		}
	}

	p := &ScrabbleProposal{Placements: slices.Clone(placements), remaining: rack}
	wordCount := 0
	addWord := func(cell, delta int) {
		for {
			n := scrabbleStep(cell, -delta)
			if n < 0 || board[n] == "" {
				break
			}
			cell = n
		}
		var word strings.Builder
		sum, mult := 0, 1
		for cell >= 0 && board[cell] != "" {
			word.WriteString(strings.ToUpper(board[cell]))
			value := scrabbleValue(board[cell])
			if s.Board[cell] == "" {
				switch scrabblePremiums[cell] {
				case "DL":
					value *= 2
				case "TL":
					value *= 3
				case "DW":
					mult *= 2
				case "TW":
					mult *= 3
				}
			}
			sum += value
			cell = scrabbleStep(cell, delta)
		}
		if word.Len() < 2 {
			return
		}
		wordCount++
		p.score += sum * mult
		w := word.String()
		if !known(w) && !slices.Contains(p.Words, w) {
			p.Words = append(p.Words, w)
		}
	}
	addWord(first, main)
	for _, v := range placements {
		addWord(v.Cell, cross)
	}
	if wordCount == 0 {
		return nil, errors.New("form a word of at least two letters")
	}
	if len(placements) == 7 {
		p.score += 50
	}
	return p, nil
}
