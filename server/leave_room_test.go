package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coup-server/game"
	"github.com/gorilla/websocket"
)

func TestLeaveFinishedRoomPreservesResultsForEveryGame(t *testing.T) {
	testCases := map[string]func(*Room) interface{}{
		"coup": func(room *Room) interface{} {
			room.gameState = &game.GameState{Phase: game.PhaseGameOver, Winner: "winner"}
			return room.gameState
		},
		"poker": func(room *Room) interface{} {
			room.pokerState = &game.PokerState{Phase: game.PokerPhaseGameOver, Winner: "winner"}
			return room.pokerState
		},
		"ludo": func(room *Room) interface{} {
			room.ludoState = &game.LudoState{Phase: game.LudoPhaseFinished, Winner: "winner"}
			return room.ludoState
		},
		"nquestions": func(room *Room) interface{} {
			room.nqState = &game.NQState{Phase: game.NQPhaseFinished}
			return room.nqState
		},
		"commune": func(room *Room) interface{} {
			room.communeState = &game.CommuneState{Phase: game.CommunePhaseFinished, Winner: "winner"}
			return room.communeState
		},
		"twentynine": func(room *Room) interface{} {
			room.tnState = &game.TwentyNineState{Phase: game.TN_PhaseGameOver}
			return room.tnState
		},
		"hearts": func(room *Room) interface{} {
			room.heartsState = &game.HeartsState{Phase: game.HT_PhaseGameOver, Winner: "winner"}
			return room.heartsState
		},
		"uno": func(room *Room) interface{} {
			room.unoState = &game.UNOState{Phase: game.UNOPhaseGameOver, WinnerID: "winner"}
			return room.unoState
		},
		"bluff": func(room *Room) interface{} {
			room.bluffState = &game.BluffState{Phase: game.BluffPhaseGameOver, WinnerName: "winner"}
			return room.bluffState
		},
		"blackjack": func(room *Room) interface{} {
			room.blackjackState = &game.BlackjackState{Phase: game.BlackjackPhaseGameOver, Round: 3}
			return room.blackjackState
		},
	}
	for gameType, initialize := range testCases {
		t.Run(gameType, func(t *testing.T) {
			room := discoveryRoom("LEAVE", gameType, false, 2)
			room.connPlayer = map[string]string{"other-connection": "player-id-1"}
			state := initialize(room)
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := room.leaveGameRoom(room.hostID); err != nil {
				t.Fatal(err)
			}
			if len(room.players) != 1 || room.hostID != "player-id-1" {
				t.Fatal("exit did not remove host and transfer ownership")
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("finished results were changed by exit")
			}
		})
	}
}

func TestLeaveActiveUNOForfeitsAndStopsDisconnectTimer(t *testing.T) {
	room := discoveryRoom("LEAVE", "uno", false, 3)
	players := []struct{ ID, Name string }{{room.hostID, "Host"}, {"player-id-1", "Guest"}, {"player-id-2", "Other"}}
	var err error
	room.unoState, err = game.UNOInitializeGame(players, false)
	if err != nil {
		t.Fatal(err)
	}
	room.connPlayer = map[string]string{"guest-connection": "player-id-1"}
	room.disconnectTimers = map[string]*time.Timer{room.hostID: time.AfterFunc(time.Hour, func() {})}
	if err := room.leaveGameRoom(room.hostID); err != nil {
		t.Fatal(err)
	}
	if len(room.unoState.Players) != 2 || len(room.players) != 2 {
		t.Fatal("active player was not forfeited and removed")
	}
	if len(room.disconnectTimers) != 0 {
		t.Fatal("departing player retained reconnect timer")
	}
	if room.hostID != "player-id-1" {
		t.Fatal("host was not transferred")
	}
	if room.unoState.CurrentPlayerIdx >= len(room.unoState.Players) {
		t.Fatal("exit left an invalid turn index")
	}
}

func TestSpectatorLeaveDoesNotForfeitPlayer(t *testing.T) {
	room := discoveryRoom("LEAVE", "uno", false, 2)
	players := []struct{ ID, Name string }{{room.hostID, "Host"}, {"player-id-1", "Guest"}}
	var err error
	room.unoState, err = game.UNOInitializeGame(players, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := room.leaveGameRoom("spectator-player-id"); err != nil {
		t.Fatal(err)
	}
	if len(room.players) != 2 || len(room.unoState.Players) != 2 {
		t.Fatal("spectator exit affected players")
	}
}

func TestLeaveRoomWebSocketAcknowledgesAfterVictory(t *testing.T) {
	code := "LEAVETEST"
	room := discoveryRoom(code, "uno", false, 1)
	room.unoState = &game.UNOState{Phase: game.UNOPhaseGameOver, WinnerID: room.hostID,
		Players: []game.UNOPlayer{{ID: room.hostID, Name: "Host"}}}
	room.connections = make(map[string]*websocket.Conn)
	room.connPlayer = make(map[string]string)
	room.disconnectTimers = make(map[string]*time.Timer)
	roomsMu.Lock()
	rooms[code] = room
	roomsMu.Unlock()
	t.Cleanup(func() { roomsMu.Lock(); delete(rooms, code); roomsMu.Unlock() })
	server := httptest.NewServer(httpHandlerForRooms())
	defer server.Close()
	address := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?room=" + code + "&gameType=uno&playerId=" + room.hostID
	connection, _, err := websocket.DefaultDialer.Dial(address, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(map[string]interface{}{"type": "leave-room"}); err != nil {
		t.Fatal(err)
	}
	connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var message OutMessage
		if err := connection.ReadJSON(&message); err != nil {
			t.Fatal(err)
		}
		if message.Type == "room-left" {
			break
		}
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	if len(room.players) != 0 || len(room.connPlayer) != 0 {
		t.Fatal("leave acknowledgement did not remove room membership")
	}
	if room.unoState.WinnerID != "host-player-id" {
		t.Fatal("winner was lost on exit")
	}
}
