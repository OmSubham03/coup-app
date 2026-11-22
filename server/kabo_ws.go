package main

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"coup-server/game"
)

func (room *Room) startKabo() error {
	if room.kaboState != nil {
		return errors.New("Return to the lobby before starting another match")
	}
	ids := make([]string, 0, len(room.players))
	for id := range room.players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	seats := []game.KaboPlayer{}
	for _, id := range ids {
		seats = append(seats, game.KaboPlayer{ID: id, Name: room.players[id].Name})
	}
	state, err := game.NewKabo(seats)
	if err == nil {
		room.kaboState = state
	}
	return err
}

func (room *Room) kaboAction(playerID string, msg InMessage) error {
	if room.kaboState == nil || !room.additionalPlayerInGame(playerID) {
		return game.ErrInvalidPlayer
	}
	if msg.Type == "kabo-next-round" || msg.Type == "kabo-end-game" {
		if playerID != room.hostID {
			return errors.New("Only the host can redeal or end the match")
		}
	}
	var action game.KaboAction
	if len(msg.Payload) == 0 || string(msg.Payload) == "null" {
		return errors.New("Kabo actions require a payload")
	}
	if err := json.Unmarshal(msg.Payload, &action); err != nil {
		return errors.New("Invalid Kabo action payload")
	}
	// Initial peeks are independent per player, so simultaneous ready messages
	// need not share a global revision.
	switch msg.Type {
	case "kabo-initial":
		return room.kaboState.InitialPeek(playerID, action.SlotIDs)
	case "kabo-ready":
		return room.kaboState.Ready(playerID)
	case "kabo-next-round":
		if action.Revision != room.kaboState.Revision {
			return errors.New("The table changed. Try again.")
		}
		return room.kaboState.NextRound()
	case "kabo-end-game":
		if action.Revision != room.kaboState.Revision {
			return errors.New("The table changed. Try again.")
		}
		return room.kaboState.EndGame()
	}
	return room.kaboState.Apply(playerID, strings.TrimPrefix(msg.Type, "kabo-"), action)
}
