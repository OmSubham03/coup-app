package main

import (
	"encoding/json"
	"errors"
	"sort"

	"coup-server/game"
)

func (room *Room) startScrabble() error {
	if room.scrabbleState != nil {
		return errors.New("return to the lobby before starting another game")
	}
	ids := make([]string, 0, len(room.players))
	for id := range room.players {
		if id != room.hostID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	seats := []game.ScrabblePlayer{{ID: room.hostID, Name: room.players[room.hostID].Name}}
	for _, id := range ids {
		seats = append(seats, game.ScrabblePlayer{ID: id, Name: room.players[id].Name})
	}
	state, err := game.NewScrabble(seats)
	if err == nil {
		room.scrabbleState = state
	}
	return err
}

func (room *Room) scrabbleAction(playerID string, msg InMessage) error {
	s := room.scrabbleState
	switch msg.Type {
	case "scrabble-play":
		var p struct {
			Placements []struct {
				Cell   *int   `json:"cell"`
				Letter string `json:"letter"`
				Blank  *bool  `json:"blank"`
			} `json:"placements"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return err
		}
		if len(p.Placements) > 7 {
			return errors.New("place at most 7 tiles")
		}
		placements := make([]game.ScrabblePlacement, len(p.Placements))
		for i, v := range p.Placements {
			if v.Cell == nil || v.Blank == nil {
				return errors.New("placement requires cell, letter and blank")
			}
			placements[i] = game.ScrabblePlacement{Cell: *v.Cell, Letter: v.Letter, Blank: *v.Blank}
		}
		return s.Play(playerID, placements)
	case "scrabble-vote":
		var p struct {
			ID      uint64 `json:"id"`
			Approve *bool  `json:"approve"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return err
		}
		if p.Approve == nil {
			return errors.New("vote requires approve")
		}
		return s.Vote(playerID, p.ID, *p.Approve)
	case "scrabble-exchange":
		var p struct {
			Tiles []string `json:"tiles"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return err
		}
		return s.Exchange(playerID, p.Tiles)
	case "scrabble-pass":
		return s.Pass(playerID)
	}
	return errors.New("unknown Scrabble action")
}
