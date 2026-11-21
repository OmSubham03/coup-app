package main

import (
	"encoding/json"
	"fmt"

	"coup-server/game"
)

func (room *Room) isAdditionalGame() bool {
	return room.gameType == "bluff" || room.gameType == "blackjack" || room.gameType == "tictactoe" || room.gameType == "scrabble"
}

func (room *Room) additionalPlayerInGame(playerID string) bool {
	if state := room.scrabbleState; state != nil {
		i := state.PlayerIndex(playerID)
		return room.players[playerID] != nil && i >= 0 && state.Players[i].Active
	}
	if state := room.ticTacToeState; state != nil {
		return room.players[playerID] != nil && state.PlayerIndex(playerID) >= 0
	}
	if room.bluffState != nil {
		for _, player := range room.bluffState.Players {
			if player.ID == playerID && player.Active {
				return true
			}
		}
	}
	if room.blackjackState != nil {
		for _, player := range room.blackjackState.Players {
			if player.ID == playerID && player.Active {
				return true
			}
		}
	}
	return false
}

func (room *Room) additionalPlayerActive(playerID string) bool {
	return room.additionalPlayerInGame(playerID) &&
		(room.scrabbleState != nil && room.scrabbleState.Phase != game.ScrabbleGameOver ||
			room.ticTacToeState != nil && room.ticTacToeState.Phase == game.TicTacToePlaying ||
			room.bluffState != nil && room.bluffState.Phase != game.BluffPhaseGameOver ||
			room.blackjackState != nil && room.blackjackState.Phase != game.BlackjackPhaseGameOver)
}

func (room *Room) sendAdditionalState(connID, viewerID string) {
	if state := room.scrabbleState; state != nil {
		room.sendTo(connID, OutMessage{Type: "scrabble-state", Payload: state.View(viewerID)})
	}
	if room.ticTacToeState != nil {
		room.sendTo(connID, OutMessage{Type: "tictactoe-state", Payload: room.ticTacToeState})
	}
	if room.gameType == "bluff" && room.bluffState != nil {
		room.sendTo(connID, OutMessage{Type: "bluff-state", Payload: game.BluffSanitize(room.bluffState, viewerID)})
	}
	if room.gameType == "blackjack" && room.blackjackState != nil {
		room.sendTo(connID, OutMessage{Type: "blackjack-state", Payload: game.BlackjackSanitize(room.blackjackState, viewerID)})
	}
}

func (room *Room) broadcastAdditionalState() {
	for connID, playerID := range room.connPlayer {
		if room.players[playerID] == nil {
			playerID = ""
		}
		room.sendAdditionalState(connID, playerID)
	}
}

func (room *Room) exitAdditionalPlayer(playerID string) error {
	if !room.additionalPlayerActive(playerID) {
		return nil
	}
	if state := room.scrabbleState; state != nil {
		state.Leave(playerID)
		return nil
	}
	if state := room.ticTacToeState; state != nil {
		state.WinnerID = state.Players[1-state.PlayerIndex(playerID)].ID
		state.Phase = game.TicTacToeGameOver
		return nil
	}
	if room.gameType == "bluff" {
		return game.BluffExit(room.bluffState, playerID)
	}
	return game.BlackjackExit(room.blackjackState, playerID)
}

func (room *Room) additionalLobbyMessage() OutMessage {
	return OutMessage{Type: "players-updated", Payload: map[string]interface{}{
		"players": room.playerList(), "hostId": room.hostID, "gameType": room.gameType,
		"gameActive": room.hasStartedGame(), "twoDecks": room.twoDecks,
	}}
}

func (room *Room) handleAdditionalMessage(connID, playerID string, msg InMessage) bool {
	if !room.isAdditionalGame() {
		return false
	}
	var err error
	switch msg.Type {
	case "start-game":
		if playerID != room.hostID || room.players[playerID] == nil {
			err = fmt.Errorf("Only the host can start the game")
			break
		}
		if room.gameType == "scrabble" {
			err = room.startScrabble()
			break
		}
		if room.gameType == "tictactoe" {
			if room.ticTacToeState != nil {
				err = fmt.Errorf("Return to the lobby before starting a new game")
				break
			}
			if len(room.players) != 2 {
				err = fmt.Errorf("Tic-tac-toe requires exactly two players")
				break
			}
			state := &game.TicTacToeState{Phase: game.TicTacToePlaying}
			for id, player := range room.players {
				seat := 0
				if id != room.hostID {
					seat = 1
				}
				state.Players[seat] = game.TicTacToePlayer{ID: id, Name: player.Name}
			}
			room.ticTacToeState = state
			break
		}
		if room.bluffState != nil && room.bluffState.Phase != game.BluffPhaseGameOver ||
			room.blackjackState != nil && room.blackjackState.Phase != game.BlackjackPhaseGameOver {
			return true
		}
		players := make(map[string]string, len(room.players))
		for id, player := range room.players {
			players[id] = player.Name
		}
		if room.gameType == "bluff" {
			var state *game.BluffState
			state, err = game.BluffInitializeGame(players, room.twoDecks)
			if err == nil {
				room.bluffState = state
			}
		} else {
			var state *game.BlackjackState
			state, err = game.BlackjackInitializeGame(players, room.twoDecks)
			if err == nil {
				room.blackjackState = state
			}
		}
	case "get-state":
		if room.gameType == "scrabble" {
			if !room.additionalPlayerInGame(playerID) {
				playerID = ""
			}
			room.sendAdditionalState(connID, playerID)
			return true
		}
		if room.gameType != "tictactoe" {
			return false
		}
		fallthrough
	case "spectate":
		room.sendAdditionalState(connID, "")
		return true
	case "return-to-lobby":
		if room.scrabbleState != nil && room.scrabbleState.Phase != game.ScrabbleGameOver ||
			room.ticTacToeState != nil && room.ticTacToeState.Phase != game.TicTacToeGameOver ||
			room.bluffState != nil && room.bluffState.Phase != game.BluffPhaseGameOver ||
			room.blackjackState != nil && room.blackjackState.Phase != game.BlackjackPhaseGameOver {
			err = fmt.Errorf("Game is still in progress")
			break
		}
		if room.players[playerID] == nil {
			err = game.ErrInvalidPlayer
			break
		}
		room.bluffState = nil
		room.blackjackState = nil
		room.ticTacToeState = nil
		room.scrabbleState = nil
		for id, timer := range room.disconnectTimers {
			timer.Stop()
			delete(room.disconnectTimers, id)
		}
		connected := make(map[string]bool)
		for _, id := range room.connPlayer {
			connected[id] = true
		}
		for id := range room.players {
			if !connected[id] {
				delete(room.players, id)
			}
		}
		if room.players[room.hostID] == nil {
			room.hostID = ""
			for id := range room.players {
				room.hostID = id
				break
			}
		}
		room.broadcast(OutMessage{Type: "state", Payload: nil})
		room.broadcastVisibility()
		room.broadcast(room.additionalLobbyMessage())
		return true
	case "exit-game":
		if (room.gameType == "tictactoe" || room.gameType == "scrabble") && !room.additionalPlayerInGame(playerID) {
			err = game.ErrInvalidPlayer
			break
		}
		err = room.exitAdditionalPlayer(playerID)
		if err == nil {
			room.broadcastAdditionalState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, room.additionalLobbyMessage())
			return true
		}
	case "scrabble-play", "scrabble-vote", "scrabble-pass", "scrabble-exchange":
		if room.gameType != "scrabble" {
			return false
		}
		if !room.additionalPlayerInGame(playerID) {
			err = game.ErrInvalidPlayer
			break
		}
		err = room.scrabbleAction(playerID, msg)
	case "tictactoe-move":
		if room.gameType != "tictactoe" {
			return false
		}
		if !room.additionalPlayerInGame(playerID) {
			err = game.ErrInvalidPlayer
			break
		}
		var payload struct {
			Cell *int `json:"cell"`
		}
		if json.Unmarshal(msg.Payload, &payload) != nil || payload.Cell == nil {
			err = fmt.Errorf("Move requires an integer cell")
			break
		}
		err = room.ticTacToeState.Move(playerID, *payload.Cell)
	case "bluff-play", "bluff-challenge", "bluff-accept", "blackjack-hit", "blackjack-stand", "blackjack-double", "blackjack-next-round":
		if room.gameType == "tictactoe" || room.gameType == "scrabble" {
			return false
		}
		if !room.additionalPlayerActive(playerID) || room.players[playerID] == nil {
			err = game.ErrInvalidPlayer
			break
		}
		switch msg.Type {
		case "bluff-play":
			var payload struct {
				CardIDs []string `json:"cardIds"`
			}
			if err = json.Unmarshal(msg.Payload, &payload); err == nil {
				err = game.BluffPlay(room.bluffState, playerID, payload.CardIDs)
			}
		case "bluff-challenge":
			err = game.BluffChallenge(room.bluffState, playerID)
		case "bluff-accept":
			err = game.BluffAccept(room.bluffState, playerID)
		case "blackjack-hit":
			err = game.BlackjackHit(room.blackjackState, playerID)
		case "blackjack-stand":
			err = game.BlackjackStand(room.blackjackState, playerID)
		case "blackjack-double":
			err = game.BlackjackDouble(room.blackjackState, playerID)
		case "blackjack-next-round":
			if playerID != room.hostID {
				err = fmt.Errorf("Only the host can start the next round")
			} else {
				err = game.BlackjackNextRound(room.blackjackState, playerID)
			}
		}
	default:
		return false
	}
	if err != nil {
		room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
	} else {
		room.broadcastAdditionalState()
	}
	return true
}
