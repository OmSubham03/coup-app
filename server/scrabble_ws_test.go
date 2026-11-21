package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/gorilla/websocket"
)

type scrabbleWire struct {
	ID, Phase, LastAction      string
	CurrentPlayerIdx, BagCount int
	Board, Premiums, Rack      []string
	Players                    []struct {
		ID, Name string
		Active   bool
		Score    int
	}
	Pending *struct {
		ID         uint64
		Placements []struct {
			Cell   int
			Letter string
			Blank  bool
		}
		Approvals, Words []string
	}
}

func readScrabble(h *additionalWSTest, c *websocket.Conn) scrabbleWire {
	h.t.Helper()
	raw := h.read(c, "scrabble-state")
	fields := decodeAdditional[map[string]json.RawMessage](h.t, raw)
	for k := range fields {
		if !slices.Contains([]string{"id", "phase", "lastAction", "currentPlayerIdx", "bagCount", "board", "premiums", "rack", "players", "pending"}, k) {
			h.t.Fatalf("unexpected snapshot field %q", k)
		}
	}
	for _, p := range decodeAdditional[[]map[string]json.RawMessage](h.t, fields["players"]) {
		if len(p) != 4 || p["id"] == nil || p["name"] == nil || p["active"] == nil || p["score"] == nil {
			h.t.Fatalf("private or missing player field: %v", p)
		}
	}
	if fields["pending"] != nil {
		p := decodeAdditional[map[string]json.RawMessage](h.t, fields["pending"])
		if len(p) != 4 || p["placements"] == nil || p["words"] == nil || p["approvals"] == nil || p["id"] == nil {
			h.t.Fatal("private or missing proposal field", p)
		}
	}
	s := decodeAdditional[scrabbleWire](h.t, raw)
	if s.Rack == nil || len(s.Board) != 225 || len(s.Premiums) != 225 || s.ID == "" || s.Phase == "" || s.LastAction == "" || fields["currentPlayerIdx"] == nil || fields["bagCount"] == nil {
		h.t.Fatal("invalid snapshot shape")
	}
	return s
}

func TestScrabbleWebSocketLifecycle(t *testing.T) {
	h := newAdditionalWSTest(t, "scrabble")
	host := h.connect("scrabble-host", "create", "")
	h.join(host, "Host", false)
	reject := func(c *websocket.Conn, kind string, p any) { t.Helper(); h.send(c, kind, p); h.read(c, "error") }
	reject(host, "start-game", nil)
	h.send(host, "set-room-visibility", map[string]bool{"public": true})
	h.sync(host)
	if !slices.ContainsFunc(listPublicRooms("scrabble"), func(r publicRoomSummary) bool { return r.Code == h.code && r.Capacity == 4 }) {
		t.Fatal("missing public lobby")
	}
	guest := h.connect("scrabble-guest", "", "")
	h.join(guest, "Guest", true)
	reject(guest, "start-game", nil)
	h.send(host, "start-game", nil)
	initial := readScrabble(h, host)
	other := readScrabble(h, guest)
	if initial.ID != other.ID || initial.Players[0].ID != "scrabble-host" || len(initial.Rack) != 7 || len(other.Rack) != 7 || initial.BagCount != 86 || initial.Phase != "playing" {
		t.Fatal("bad initial deal")
	}
	watcher := h.connect("scrabble-watcher", "", "")
	if w := readScrabble(h, watcher); len(w.Rack) != 0 {
		t.Fatal("spectator received a rack")
	}
	query := func(c *websocket.Conn) scrabbleWire {
		t.Helper()
		h.send(c, "get-state", map[string]string{"playerId": "scrabble-host"})
		return readScrabble(h, c)
	}
	if len(query(watcher).Rack) != 0 || !slices.Equal(query(guest).Rack, other.Rack) {
		t.Fatal("get-state leaked a different rack")
	}
	reject(watcher, "scrabble-pass", nil)
	reject(guest, "scrabble-exchange", map[string]any{"tiles": other.Rack})
	reject(host, "return-to-lobby", nil)
	reject(host, "scrabble-play", json.RawMessage(`{"placements":[{"letter":"A","blank":false}]}`))

	// Stage a board fixture so the challenge path is deterministic regardless of
	// the random deal. The eventual move forms a word containing four Qs.
	room := getRoom(h.code)
	room.mu.Lock()
	for i := 109; i <= 112; i++ {
		room.scrabbleState.Board[i] = "Q"
	}
	room.mu.Unlock()
	before := query(host)
	tile := initial.Rack[0]
	face := tile
	if tile == "?" {
		face = "Z"
	}
	move := map[string]any{"placements": []map[string]any{{"letter": face, "cell": 113, "blank": tile == "?"}}}
	h.send(host, "scrabble-play", move)
	pending := readScrabble(h, host)
	p := pending.Pending
	if pending.Phase != "voting" || p == nil || len(p.Placements) != 1 || !slices.Equal(p.Approvals, []string{"scrabble-host"}) || !slices.Equal(p.Words, []string{"QQQQ" + face}) || !slices.Equal(pending.Board, before.Board) || !slices.Equal(pending.Rack, initial.Rack) {
		t.Fatal("bad proposal")
	}
	if w := readScrabble(h, watcher); len(w.Rack) != 0 || !reflect.DeepEqual(w.Pending, p) {
		t.Fatal("bad public proposal")
	}
	guest.Close()
	h.await(func() bool { return room.disconnectTimers["scrabble-guest"] != nil })
	guest = h.connect("scrabble-guest", "", "")
	if g := readScrabble(h, guest); !slices.Equal(g.Rack, other.Rack) || !reflect.DeepEqual(g.Pending, p) {
		t.Fatal("reconnect lost proposal or rack")
	}
	vote := func(id uint64, yes bool) any { return map[string]any{"id": id, "approve": yes} }
	reject(watcher, "scrabble-vote", vote(p.ID, false))
	reject(host, "scrabble-vote", vote(p.ID, true))
	reject(guest, "scrabble-vote", map[string]any{"id": p.ID})
	h.send(guest, "scrabble-vote", vote(p.ID, false))
	restored := readScrabble(h, host)
	restored.LastAction = before.LastAction
	if !reflect.DeepEqual(before, restored) {
		t.Fatal("rejection changed committed state")
	}
	h.send(host, "scrabble-play", move)
	p2 := readScrabble(h, host).Pending
	reject(guest, "scrabble-vote", vote(p.ID, false))
	h.send(guest, "scrabble-vote", vote(p2.ID, true))
	accepted := readScrabble(h, host)
	if accepted.Pending != nil || accepted.CurrentPlayerIdx != 1 || accepted.Board[113] == "" || accepted.Players[0].Score <= 0 || accepted.BagCount != 85 {
		t.Fatal("approval failed")
	}
	h.send(guest, "scrabble-exchange", map[string]any{"tiles": other.Rack[:1]})
	if s := readScrabble(h, host); s.CurrentPlayerIdx != 0 || s.BagCount != 85 || !slices.Equal(s.Rack, accepted.Rack) {
		t.Fatal("exchange affected other seat")
	}
	h.send(host, "leave-room", nil)
	h.read(host, "room-left")
	h.sync(guest) // Discard previously queued game broadcasts.
	final := query(guest)
	if final.Phase != "game_over" || final.Players[0].Active || !final.Players[1].Active || !slices.Equal(final.Rack[:6], other.Rack[1:]) {
		t.Fatal("bad forfeit")
	}
	reject(watcher, "return-to-lobby", nil)
	reject(guest, "start-game", nil)
	h.send(guest, "return-to-lobby", nil)
	if string(h.read(guest, "state")) != "null" {
		t.Fatal("lobby didn't clear game")
	}
	h.sync(watcher)
	h.join(watcher, "New seat", true)
	h.send(guest, "start-game", nil)
	if replay := readScrabble(h, guest); replay.ID == initial.ID || replay.CurrentPlayerIdx != 0 || replay.Players[0].ID != "scrabble-guest" || replay.Phase != "playing" {
		t.Fatal("bad replay")
	}
}
