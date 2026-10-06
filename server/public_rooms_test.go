package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coup-server/game"
	"github.com/gorilla/websocket"
)

func discoveryRoom(code, gameType string, public bool, members int) *Room {
	room := &Room{code: code, gameType: gameType, created: true, isPublic: public,
		hostID: "host-player-id", players: make(map[string]*PlayerConn)}
	for index := 0; index < members; index++ {
		playerID := fmt.Sprintf("player-id-%d", index)
		if index == 0 {
			playerID = room.hostID
		}
		room.players[playerID] = &PlayerConn{ID: playerID, Name: playerID}
	}
	return room
}

func TestPublicRoomsFilterAndLimit(t *testing.T) {
	original := rooms
	rooms = make(map[string]*Room)
	t.Cleanup(func() { rooms = original })
	for index := 0; index < 7; index++ {
		code := fmt.Sprintf("R%04d", index)
		rooms[code] = discoveryRoom(code, "uno", true, 2)
	}
	rooms["PRIVATE"] = discoveryRoom("PRIVATE", "uno", false, 1)
	rooms["FULL"] = discoveryRoom("FULL", "uno", true, 6)
	rooms["EMPTY"] = discoveryRoom("EMPTY", "uno", true, 0)
	rooms["OTHER"] = discoveryRoom("OTHER", "ludo", true, 1)
	started := discoveryRoom("STARTED", "uno", true, 2)
	started.unoState = &game.UNOState{Phase: game.UNOPhaseGameOver}
	rooms[started.code] = started
	result := listPublicRooms("uno")
	if len(result) != 5 {
		t.Fatalf("got %d rooms, want 5", len(result))
	}
	for index, room := range result {
		if room.Code != fmt.Sprintf("R%04d", index) || room.Members != 2 || room.Capacity != 6 || room.GameType != "uno" {
			t.Fatalf("unexpected listing: %+v", room)
		}
	}
	request := httptest.NewRequest("GET", "/api/public-rooms?gameType=ludo", nil)
	response := httptest.NewRecorder()
	handlePublicRooms(response, request)
	var listed []publicRoomSummary
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Code != "OTHER" {
		t.Fatalf("wrong-game results: %+v", listed)
	}
	bad := httptest.NewRecorder()
	handlePublicRooms(bad, httptest.NewRequest("GET", "/api/public-rooms?gameType=invalid", nil))
	if bad.Code != 400 {
		t.Fatalf("invalid game status = %d", bad.Code)
	}
}

func TestRoomVisibilityHostOnlyAndPrivateByDefault(t *testing.T) {
	room := discoveryRoom("ABCDE", "coup", false, 2)
	if room.availablePublicRoom() {
		t.Fatal("private room was listed")
	}
	if err := room.setPublicVisibility("player-id-1", true); err == nil {
		t.Fatal("non-host changed visibility")
	}
	if err := room.setPublicVisibility(room.hostID, true); err != nil {
		t.Fatal(err)
	}
	if !room.availablePublicRoom() {
		t.Fatal("host's public room was not listed")
	}
	if err := room.setPublicVisibility(room.hostID, false); err != nil {
		t.Fatal(err)
	}
	if room.availablePublicRoom() {
		t.Fatal("private toggle did not remove room")
	}
	room.gameState = &game.GameState{}
	if err := room.setPublicVisibility(room.hostID, true); err == nil {
		t.Fatal("visibility changed after start")
	}
}

func TestPublicRoomCapacityForEveryGame(t *testing.T) {
	for gameType, capacity := range map[string]int{"coup": 6, "uno": 6, "poker": 8, "ludo": 4, "twentynine": 4, "hearts": 4, "commune": 10, "nquestions": 10} {
		t.Run(gameType, func(t *testing.T) {
			room := discoveryRoom("ABCDE", gameType, true, capacity-1)
			if roomCapacity(gameType) != capacity || !room.availablePublicRoom() {
				t.Fatal("room below capacity was excluded")
			}
			room.players["last"] = &PlayerConn{ID: "last", Name: "last"}
			if room.availablePublicRoom() {
				t.Fatal("full room was listed")
			}
		})
	}
}

func TestStartedRoomsAreHiddenForEveryGame(t *testing.T) {
	states := map[string]func(*Room){
		"coup":       func(room *Room) { room.gameState = &game.GameState{} },
		"poker":      func(room *Room) { room.pokerState = &game.PokerState{} },
		"ludo":       func(room *Room) { room.ludoState = &game.LudoState{} },
		"nquestions": func(room *Room) { room.nqState = &game.NQState{} },
		"commune":    func(room *Room) { room.communeState = &game.CommuneState{} },
		"twentynine": func(room *Room) { room.tnState = &game.TwentyNineState{} },
		"hearts":     func(room *Room) { room.heartsState = &game.HeartsState{} },
		"uno":        func(room *Room) { room.unoState = &game.UNOState{} },
	}
	for gameType, start := range states {
		t.Run(gameType, func(t *testing.T) {
			room := discoveryRoom("ABCDE", gameType, true, 2)
			start(room)
			if room.availablePublicRoom() {
				t.Fatal("started room was available")
			}
		})
	}
}

func TestDiscoveryJoinRechecksAvailabilityAndPreservesCodeJoin(t *testing.T) {
	room := discoveryRoom("ABCDE", "ludo", false, 1)
	publicPayload, _ := json.Marshal(map[string]interface{}{"playerName": "Guest", "publicRoom": true})
	handleMessage(room, "no-connection", "guest-player-id", InMessage{Type: "join", Payload: publicPayload})
	if room.players["guest-player-id"] != nil {
		t.Fatal("private room admitted a discovery join")
	}
	codePayload, _ := json.Marshal(map[string]interface{}{"playerName": "Guest"})
	handleMessage(room, "no-connection", "guest-player-id", InMessage{Type: "join", Payload: codePayload})
	if room.players["guest-player-id"] == nil {
		t.Fatal("manual code joining private room stopped working")
	}
	room.isPublic = true
	room.players["third"] = &PlayerConn{ID: "third"}
	room.players["fourth"] = &PlayerConn{ID: "fourth"}
	handleMessage(room, "no-connection", "extra-player-id", InMessage{Type: "join", Payload: publicPayload})
	if room.players["extra-player-id"] != nil {
		t.Fatal("stale discovery admitted player over cap")
	}
}

func TestPublicRoomWebSocketLifecycle(t *testing.T) {
	server := httptest.NewServer(httpHandlerForRooms())
	defer server.Close()
	code := "PUBTEST"
	t.Cleanup(func() { roomsMu.Lock(); delete(rooms, code); roomsMu.Unlock() })
	connect := func(playerID, action string) *websocket.Conn {
		address := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?room=" + code + "&gameType=uno&playerId=" + playerID + "&action=" + action
		connection, _, err := websocket.DefaultDialer.Dial(address, nil)
		if err != nil {
			t.Fatal(err)
		}
		return connection
	}
	readUntil := func(connection *websocket.Conn, kind string) OutMessage {
		t.Helper()
		connection.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var message OutMessage
			if err := connection.ReadJSON(&message); err != nil {
				t.Fatal(err)
			}
			if message.Type == kind {
				return message
			}
		}
	}
	send := func(connection *websocket.Conn, kind string, payload interface{}) {
		t.Helper()
		if err := connection.WriteJSON(map[string]interface{}{"type": kind, "payload": payload}); err != nil {
			t.Fatal(err)
		}
	}
	host := connect("public-host-id", "create")
	defer host.Close()
	readUntil(host, "waiting")
	send(host, "join", map[string]interface{}{"playerName": "Host"})
	readUntil(host, "players-updated")
	if len(listPublicRooms("uno")) != 0 {
		t.Fatal("new room was not private")
	}
	send(host, "set-room-visibility", map[string]interface{}{"public": true})
	readUntil(host, "room-visibility")
	if len(listPublicRooms("uno")) != 1 {
		t.Fatal("public lobby missing")
	}
	if len(listPublicRooms("ludo")) != 0 {
		t.Fatal("UNO room appeared in Ludo")
	}
	guest := connect("public-guest-id", "")
	defer guest.Close()
	readUntil(guest, "waiting")
	send(guest, "join", map[string]interface{}{"playerName": "Guest", "publicRoom": true})
	readUntil(guest, "players-updated")
	if listPublicRooms("uno")[0].Members != 2 {
		t.Fatal("member count did not update")
	}
	send(guest, "set-room-visibility", map[string]interface{}{"public": false})
	readUntil(guest, "error")
	if len(listPublicRooms("uno")) != 1 {
		t.Fatal("guest hid host's room")
	}
	send(host, "start-game", nil)
	readUntil(host, "uno-state")
	if len(listPublicRooms("uno")) != 0 {
		t.Fatal("started game remained public in discovery")
	}
}

func httpHandlerForRooms() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWS)
	return mux
}
