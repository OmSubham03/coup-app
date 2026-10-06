package main

import "coup-server/game"

func (room *Room) leaveGameRoom(playerID string) error {
	if room.players[playerID] != nil {
		switch room.gameType {
		case "poker":
			if room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseGameOver {
				game.PokerVoluntaryExit(room.pokerState, playerID)
			}
		case "ludo":
			if room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseFinished {
				game.LudoVoluntaryExit(room.ludoState, playerID)
			}
		case "nquestions":
			if room.nqState != nil && room.nqState.Phase != game.NQPhaseFinished {
				game.NQVoluntaryExit(room.nqState, playerID)
			}
		case "commune":
			if room.communeState != nil && room.communeState.Phase != game.CommunePhaseFinished {
				game.CommuneVoluntaryExit(room.communeState, playerID)
			}
		case "twentynine":
			if room.tnState != nil && room.tnState.Phase != game.TN_PhaseGameOver {
				game.TNVoluntaryExit(room.tnState, playerID)
			}
		case "hearts":
			if room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseGameOver {
				game.HTVoluntaryExit(room.heartsState, playerID)
			}
		case "uno":
			if room.unoState != nil && room.unoState.Phase != game.UNOPhaseGameOver && game.UNOFindPlayer(room.unoState, playerID) >= 0 {
				if err := game.UNOForfeitPlayer(room.unoState, playerID); err != nil {
					return err
				}
			}
		default:
			if room.gameState != nil && room.gameState.Phase != game.PhaseGameOver {
				game.VoluntaryExit(room.gameState, playerID)
			}
		}
	}
	delete(room.players, playerID)
	delete(room.ludoColors, playerID)
	if timer := room.disconnectTimers[playerID]; timer != nil {
		timer.Stop()
		delete(room.disconnectTimers, playerID)
	}
	for connID, connectedPlayer := range room.connPlayer {
		if connectedPlayer != playerID {
			continue
		}
		room.sendTo(connID, OutMessage{Type: "room-left"})
		connection := room.connections[connID]
		delete(room.connPlayer, connID)
		delete(room.connections, connID)
		if connection != nil {
			connection.Close()
		}
	}
	if room.hostID == playerID {
		room.hostID = ""
		for _, connectedPlayer := range room.connPlayer {
			if room.players[connectedPlayer] != nil {
				room.hostID = connectedPlayer
				break
			}
		}
	}
	room.broadcastState()
	room.broadcastVisibility()
	room.broadcast(OutMessage{Type: "players-updated", Payload: map[string]interface{}{
		"players": room.playerList(), "hostId": room.hostID, "gameActive": room.hasStartedGame(),
		"gameType": room.gameType, "unoStackingEnabled": room.unoStackingEnabled,
	}})
	return nil
}
