package game

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
)

const (
	ScrabblePlaying  = "playing"
	ScrabbleVoting   = "voting"
	ScrabbleGameOver = "game_over"
)

type ScrabblePlayer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Active bool   `json:"active"`
}

type ScrabblePlacement struct {
	Cell   int    `json:"cell"`
	Letter string `json:"letter"`
	Blank  bool   `json:"blank"`
}

type ScrabbleProposal struct {
	Placements []ScrabblePlacement `json:"placements"`
	Words      []string            `json:"words"`
	Approvals  []string            `json:"approvals"`
	ID         uint64              `json:"id"`
	remaining  []string
	score      int
}

type ScrabbleState struct {
	ID               string            `json:"id"`
	Phase            string            `json:"phase"`
	Players          []ScrabblePlayer  `json:"players"`
	CurrentPlayerIdx int               `json:"currentPlayerIdx"`
	Board            [225]string       `json:"board"`
	LastAction       string            `json:"lastAction"`
	Pending          *ScrabbleProposal `json:"pending,omitempty"`
	bag              []string
	racks            [][]string
	scoreless        int
}

var scrabbleProposalID atomic.Uint64

var scrabbleCounts = [...]int{9, 2, 2, 4, 12, 2, 3, 2, 9, 1, 1, 4, 2, 6, 8, 2, 1, 6, 4, 6, 4, 2, 2, 1, 2, 1}
var scrabbleValues = [...]int{1, 3, 3, 2, 1, 4, 2, 4, 1, 8, 5, 1, 3, 1, 1, 3, 10, 1, 1, 1, 1, 4, 4, 8, 4, 10}

var scrabblePremiums = func() (p [225]string) {
	// The board is symmetric across both axes; include the center row/column.
	rows := [...]string{"T..d...T", ".D...t..", "..D...d.", "d..D...d", "....D...", ".t...t..", "..d...d.", "T..d...D"}
	for r, row := range rows {
		for c, symbol := range row {
			premium := map[rune]string{'T': "TW", 'D': "DW", 'd': "DL", 't': "TL"}[symbol]
			for _, rr := range []int{r, 14 - r} {
				for _, cc := range []int{c, 14 - c} {
					p[15*rr+cc] = premium
				}
			}
		}
	}
	return
}()

func scrabbleValue(tile string) int {
	if len(tile) == 1 && tile[0] >= 'A' && tile[0] <= 'Z' {
		return scrabbleValues[tile[0]-'A']
	}
	return 0 // unassigned and assigned blanks
}

func scrabbleShuffle(tiles []string) {
	rand.Shuffle(len(tiles), func(i, j int) { tiles[i], tiles[j] = tiles[j], tiles[i] })
}

func NewScrabble(players []ScrabblePlayer) (*ScrabbleState, error) {
	if len(players) < 2 || len(players) > 4 {
		return nil, errors.New("Scrabble requires 2 to 4 players")
	}
	s := &ScrabbleState{ID: uuid.NewString(), Phase: ScrabblePlaying, LastAction: "Game started", racks: make([][]string, len(players))}
	for _, p := range players {
		if p.ID == "" || s.PlayerIndex(p.ID) >= 0 {
			return nil, ErrInvalidPlayer
		}
		s.Players = append(s.Players, ScrabblePlayer{ID: p.ID, Name: p.Name, Active: true})
	}
	s.bag = []string{"?", "?"}
	for i, n := range scrabbleCounts {
		for j := 0; j < n; j++ {
			s.bag = append(s.bag, string(rune('A'+i)))
		}
	}
	scrabbleShuffle(s.bag)
	for i := range players {
		s.refill(i)
	}
	return s, nil
}

func (s *ScrabbleState) PlayerIndex(id string) int {
	for i, p := range s.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func (s *ScrabbleState) View(viewer string) any {
	rack := []string{}
	if i := s.PlayerIndex(viewer); i >= 0 && s.Players[i].Active {
		rack = append(rack, s.racks[i]...)
	}
	return struct {
		*ScrabbleState
		Rack     []string    `json:"rack"`
		BagCount int         `json:"bagCount"`
		Premiums [225]string `json:"premiums"`
	}{s, rack, len(s.bag), scrabblePremiums}
}

func (s *ScrabbleState) refill(i int) {
	n := min(7-len(s.racks[i]), len(s.bag))
	s.racks[i] = append(s.racks[i], s.bag[:n]...)
	s.bag = s.bag[n:]
}

func (s *ScrabbleState) checkTurn(id string) error {
	if s.Phase != ScrabblePlaying {
		return ErrWrongPhase
	}
	if s.Players[s.CurrentPlayerIdx].ID != id {
		return ErrNotYourTurn
	}
	return nil
}

func (s *ScrabbleState) advance() {
	for {
		s.CurrentPlayerIdx = (s.CurrentPlayerIdx + 1) % len(s.Players)
		if s.Players[s.CurrentPlayerIdx].Active {
			return
		}
	}
}

func (s *ScrabbleState) endTurn(points int) {
	i := s.CurrentPlayerIdx
	if points == 0 {
		s.scoreless++
	} else {
		s.scoreless = 0
	}
	wentOut := len(s.racks[i]) == 0 && len(s.bag) == 0
	if wentOut || s.scoreless >= 6 {
		for j, p := range s.Players {
			if !p.Active {
				continue
			}
			for _, tile := range s.racks[j] {
				v := scrabbleValue(tile)
				s.Players[j].Score -= v
				if wentOut {
					s.Players[i].Score += v
				}
			}
		}
		s.Phase = ScrabbleGameOver
		return
	}
	s.advance()
}

func (s *ScrabbleState) Pass(id string) error {
	if err := s.checkTurn(id); err != nil {
		return err
	}
	s.LastAction = s.Players[s.CurrentPlayerIdx].Name + " passed"
	s.endTurn(0)
	return nil
}

func (s *ScrabbleState) Exchange(id string, tiles []string) error {
	if err := s.checkTurn(id); err != nil {
		return err
	}
	if len(s.bag) < 7 || len(tiles) < 1 || len(tiles) > 7 {
		return errors.New("exchange requires 1–7 rack tiles and at least 7 tiles in the bag")
	}
	i := s.CurrentPlayerIdx
	rack := slices.Clone(s.racks[i])
	for _, tile := range tiles {
		j := slices.Index(rack, tile)
		if j < 0 {
			return errors.New("tile not in rack")
		}
		rack = slices.Delete(rack, j, j+1)
	}
	s.racks[i] = rack
	s.refill(i)
	s.bag = append(s.bag, tiles...)
	scrabbleShuffle(s.bag)
	s.LastAction = fmt.Sprintf("%s exchanged %d tile(s)", s.Players[i].Name, len(tiles))
	s.endTurn(0)
	return nil
}

func (s *ScrabbleState) Play(id string, placements []ScrabblePlacement) error {
	if err := s.checkTurn(id); err != nil {
		return err
	}
	p, err := s.evaluate(placements, scrabbleKnown)
	if err != nil {
		return err
	}
	if len(p.Words) == 0 {
		s.commit(p)
		return nil
	}
	p.ID = scrabbleProposalID.Add(1)
	p.Approvals = []string{id}
	s.Pending, s.Phase = p, ScrabbleVoting
	s.LastAction = s.Players[s.CurrentPlayerIdx].Name + " requested word approval"
	return nil
}

func (s *ScrabbleState) Vote(id string, proposal uint64, approve bool) error {
	i := s.PlayerIndex(id)
	if i < 0 || !s.Players[i].Active {
		return ErrInvalidPlayer
	}
	p := s.Pending
	if s.Phase != ScrabbleVoting || p == nil || p.ID != proposal {
		return errors.New("no matching proposal")
	}
	if slices.Contains(p.Approvals, id) {
		return errors.New("already voted")
	}
	if !approve {
		s.Pending, s.Phase = nil, ScrabblePlaying
		s.LastAction = s.Players[i].Name + " rejected the proposal"
		return nil
	}
	p.Approvals = append(p.Approvals, id)
	for _, player := range s.Players {
		if player.Active && !slices.Contains(p.Approvals, player.ID) {
			return nil
		}
	}
	s.commit(p)
	return nil
}

func (s *ScrabbleState) commit(p *ScrabbleProposal) {
	i := s.CurrentPlayerIdx
	for _, v := range p.Placements {
		tile := v.Letter
		if v.Blank {
			tile = strings.ToLower(tile)
		}
		s.Board[v.Cell] = tile
	}
	s.racks[i] = p.remaining
	s.refill(i)
	s.Players[i].Score += p.score
	s.Pending, s.Phase = nil, ScrabblePlaying
	s.LastAction = fmt.Sprintf("%s played for %d points", s.Players[i].Name, p.score)
	s.endTurn(p.score)
}

func (s *ScrabbleState) Leave(id string) {
	i := s.PlayerIndex(id)
	if s.Phase == ScrabbleGameOver || i < 0 || !s.Players[i].Active {
		return
	}
	s.Phase, s.Pending = ScrabblePlaying, nil
	s.Players[i].Active = false
	for _, t := range s.racks[i] {
		s.Players[i].Score -= scrabbleValue(t)
	}
	s.bag = append(s.bag, s.racks[i]...)
	scrabbleShuffle(s.bag)
	s.racks[i] = nil
	s.LastAction = s.Players[i].Name + " left the game"
	if s.CurrentPlayerIdx == i {
		s.advance()
	}
	active := 0
	for _, p := range s.Players {
		if p.Active {
			active++
		}
	}
	if active == 1 {
		s.Phase = ScrabbleGameOver
	}
}
