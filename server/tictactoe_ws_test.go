package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"github.com/gorilla/websocket"
)

// Decode the wire format independently of the server's fixed-size arrays.
type ticTacToeSnapshot struct {
	Players          []map[string]string `json:"players"`
	Board            []string            `json:"board"`
	CurrentPlayerIdx *int                `json:"currentPlayerIdx"`
	Phase            string              `json:"phase"`
	WinnerID         *string             `json:"winnerId"`
	WinningLine      []int               `json:"winningLine"`
}

func readTicTacToe(h *additionalWSTest, clients ...*websocket.Conn) ticTacToeSnapshot {
	h.t.Helper()
	var state ticTacToeSnapshot
	for i, client := range clients {
		got := decodeAdditional[ticTacToeSnapshot](h.t, h.read(client, "tictactoe-state"))
		if len(got.Board) != 9 || len(got.Players) != 2 || got.CurrentPlayerIdx == nil {
			h.t.Fatalf("bad state shape: %+v", got)
		}
		if i > 0 && !reflect.DeepEqual(state, got) {
			h.t.Fatalf("clients disagree: %+v vs %+v", state, got)
		}
		state = got
	}
	return state
}

func TestTicTacToeWebSocketMatchAndReplay(t *testing.T) {
	h := newAdditionalWSTest(t, "tictactoe")
	x := h.connect("host", "create", "")
	h.join(x, "Alice", false)
	o := h.connect("guest", "", "")
	h.join(o, "Bob", false)
	clients := []*websocket.Conn{x, o}
	start := func() {
		t.Helper()
		h.send(x, "start-game", nil)
		s := readTicTacToe(h, clients...)
		wantPlayers := []map[string]string{{"id": "host", "name": "Alice"}, {"id": "guest", "name": "Bob"}}
		if !reflect.DeepEqual(s.Players, wantPlayers) || !slices.Equal(s.Board, make([]string, 9)) ||
			*s.CurrentPlayerIdx != 0 || s.Phase != "playing" || s.WinnerID != nil || s.WinningLine != nil {
			t.Fatalf("bad initial state: %+v", s)
		}
	}
	reject := func(client *websocket.Conn, kind string, payload any) {
		t.Helper()
		h.send(client, kind, payload)
		h.read(client, "error")
	}
	move := func(i, cell int) ticTacToeSnapshot {
		t.Helper()
		h.send(clients[i], "tictactoe-move", map[string]int{"cell": cell})
		s := readTicTacToe(h, clients...)
		if s.Board[cell] != []string{"X", "O"}[i] || (s.Phase == "playing" && *s.CurrentPlayerIdx != 1-i) {
			t.Fatalf("incorrect move/turn: %+v", s)
		}
		return s
	}
	start()
	watcher := h.connect("watcher", "", "")
	readTicTacToe(h, watcher)
	reject(watcher, "tictactoe-move", map[string]int{"cell": 0})
	reject(x, "return-to-lobby", nil)
	reject(o, "tictactoe-move", map[string]int{"cell": 0})
	for _, raw := range []string{`{}`, `null`, `{"cell":null}`, `{"cell":"0"}`, `{"cell":1.5}`, `{"cell":-1}`, `{"cell":9}`} {
		reject(x, "tictactoe-move", json.RawMessage(raw))
	}
	h.send(x, "get-state", nil)
	if s := readTicTacToe(h, x); !slices.Equal(s.Board, make([]string, 9)) || *s.CurrentPlayerIdx != 0 {
		t.Fatalf("rejected moves changed state: %+v", s)
	}
	s := move(0, 0)
	reject(o, "tictactoe-move", map[string]int{"cell": 0})
	o.Close()
	room := getRoom(h.code)
	h.await(func() bool { return room.disconnectTimers["guest"] != nil })
	o = h.connect("guest", "", "")
	clients[1] = o
	if got := readTicTacToe(h, o); !reflect.DeepEqual(s, got) {
		t.Fatalf("reconnect changed state: %+v", got)
	}
	move(1, 3)
	move(0, 1)
	move(1, 4)
	s = move(0, 2)
	if s.Phase != "game_over" || s.WinnerID == nil || *s.WinnerID != "host" || !slices.Equal(s.WinningLine, []int{0, 1, 2}) {
		t.Fatalf("incorrect win: %+v", s)
	}
	reject(o, "tictactoe-move", map[string]int{"cell": 5})
	reject(x, "start-game", nil)
	reject(watcher, "return-to-lobby", nil)
	h.send(o, "return-to-lobby", nil)
	for _, client := range clients {
		if string(h.read(client, "state")) != "null" {
			t.Fatal("return-to-lobby did not clear state")
		}
	}
	start()
	for i, cell := range []int{0, 1, 2, 4, 3, 5, 7, 6, 8} {
		s = move(i%2, cell)
	}
	if s.Phase != "game_over" || s.WinnerID != nil || s.WinningLine != nil || slices.Contains(s.Board, "") {
		t.Fatalf("incorrect draw: %+v", s)
	}
}

func TestTicTacToeWebSocketSeatsDiscoveryAndLeave(t *testing.T) {
	h := newAdditionalWSTest(t, "tictactoe")
	listed := func(want bool) {
		t.Helper()
		w := httptest.NewRecorder()
		handlePublicRooms(w, httptest.NewRequest("GET", "/api/public-rooms?gameType=tictactoe", nil))
		rooms := decodeAdditional[[]publicRoomSummary](t, w.Body.Bytes())
		found := false
		for _, r := range rooms {
			if r.Code == h.code {
				found = true
				if r.GameType != "tictactoe" || r.Members != 1 || r.Capacity != 2 {
					t.Fatalf("unexpected public room: %+v", r)
				}
			}
		}
		if found != want || w.Code != 200 {
			t.Fatalf("public listing: %s, want listed=%t", w.Body.String(), want)
		}
	}
	x := h.connect("host", "create", "")
	h.join(x, "X", false)
	h.send(x, "start-game", nil)
	h.read(x, "error")
	listed(false)
	h.send(x, "set-room-visibility", map[string]bool{"public": true})
	h.sync(x)
	listed(true)
	o := h.connect("guest", "", "")
	h.join(o, "O", true)
	listed(false)
	third := h.connect("third", "", "")
	for _, public := range []bool{true, false} {
		h.send(third, "join", map[string]any{"playerName": "Third", "publicRoom": public})
		h.read(third, "error")
	}
	h.send(o, "start-game", nil)
	h.read(o, "error")
	h.send(x, "start-game", nil)
	readTicTacToe(h, x, o)
	otherRoom := newAdditionalWSTest(t, "tictactoe")
	redirect := otherRoom.connect("host", "create", "")
	target := decodeAdditional[map[string]string](t, h.read(redirect, "redirect"))
	if target["roomCode"] != h.code || target["gameType"] != "tictactoe" {
		t.Fatalf("wrong redirect: %+v", target)
	}
	h.send(x, "leave-room", nil)
	h.read(x, "room-left")
	s := readTicTacToe(h, o)
	if s.Phase != "game_over" || s.WinnerID == nil || *s.WinnerID != "guest" || s.WinningLine != nil {
		t.Fatalf("incorrect forfeit: %+v", s)
	}
	lobby := decodeAdditional[map[string]any](t, h.read(o, "players-updated"))
	if lobby["hostId"] != "guest" || lobby["gameActive"] != true {
		t.Fatalf("bad host transfer: %+v", lobby)
	}
	listed(false)
	h.send(o, "return-to-lobby", nil)
	h.read(o, "players-updated")
	listed(true)
	h.sync(third) // Drain leave/return broadcasts before waiting for the join acknowledgement.
	h.join(third, "Third", true)
	h.send(o, "start-game", nil)
	s = readTicTacToe(h, o)
	if s.Phase != "playing" || s.Players[0]["id"] != "guest" || s.Players[1]["id"] != "third" {
		t.Fatalf("bad replacement match: %+v", s)
	}

	// Exercise the production grace-period callback without a five-minute test.
	h.sync(third)
	o.Close()
	room := getRoom(h.code)
	h.await(func() bool { return room.disconnectTimers["guest"] != nil })
	room.mu.Lock()
	room.disconnectTimers["guest"].Reset(0)
	room.mu.Unlock()
	s = readTicTacToe(h, third)
	if s.Phase != "game_over" || s.WinnerID == nil || *s.WinnerID != "third" {
		t.Fatalf("grace expiry didn't forfeit: %+v", s)
	}
}
