package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"coup-server/game"
	"github.com/gorilla/websocket"
)

type kaboSnapshot struct {
	Revision         int    `json:"revision"`
	Phase            string `json:"phase"`
	Round            int    `json:"round"`
	CurrentPlayerIdx int    `json:"currentPlayerIdx"`
	Players          []struct {
		ID         string `json:"id"`
		Ready      bool   `json:"ready"`
		TotalScore int    `json:"totalScore"`
		RoundScore *int   `json:"roundScore"`
		Cards      []struct {
			ID   string `json:"id"`
			Rank int    `json:"rank"`
			Suit string `json:"suit"`
		} `json:"cards"`
	} `json:"players"`
	Drawn      *game.KaboCard    `json:"drawn"`
	Reveals    []game.KaboReveal `json:"reveals"`
	CallerID   string            `json:"callerId"`
	FinalTurns int               `json:"finalTurns"`
	Winners    []string          `json:"winners"`
}

func TestKaboWebSocketPrivatePeeksRoundAndReconnect(t *testing.T) {
	h := newAdditionalWSTest(t, "kabo")
	host := h.connect("host", "create", "")
	h.join(host, "Host", false)
	guest := h.connect("guest", "", "")
	h.join(guest, "Guest", false)
	h.send(host, "start-game", nil)
	clients := map[string]*websocket.Conn{"host": host, "guest": guest}
	states := map[string]kaboSnapshot{}
	readAll := func() {
		t.Helper()
		for id, c := range clients {
			states[id] = decodeAdditional[kaboSnapshot](t, h.read(c, "kabo-state"))
		}
	}
	readAll()
	for _, state := range states {
		if state.Phase != game.KaboInitial || len(state.Players) != 2 || len(state.Reveals) != 0 {
			t.Fatal("bad initial state")
		}
		for _, p := range state.Players {
			for _, c := range p.Cards {
				if c.Rank != 0 || c.Suit != "" {
					t.Fatal("hidden card exposed")
				}
			}
		}
	}
	for _, id := range []string{"host", "guest"} {
		s := states[id]
		var slots []string
		for _, p := range s.Players {
			if p.ID == id {
				slots = []string{p.Cards[0].ID, p.Cards[1].ID}
			}
		}
		h.send(clients[id], "kabo-initial", map[string]any{"slotIds": slots})
		readAll()
		if len(states[id].Reveals) != 2 {
			t.Fatal("peek missing")
		}
		for other, s := range states {
			if other != id && len(s.Reveals) > 0 {
				t.Fatal("peek leaked to opponent")
			}
		}
		h.send(clients[id], "kabo-ready", map[string]any{})
		readAll()
		if len(states[id].Reveals) != 0 {
			t.Fatal("ready retained reveal")
		}
	}
	state := states["host"]
	if state.Phase != game.KaboDraw {
		t.Fatal("game not started")
	}
	watcher := h.connect("watcher", "", "")
	watched := decodeAdditional[kaboSnapshot](t, h.read(watcher, "kabo-state"))
	h.send(watcher, "kabo-draw", game.KaboAction{Revision: state.Revision, Source: "deck"})
	h.read(watcher, "error")
	if watched.Drawn != nil || len(watched.Reveals) != 0 {
		t.Fatal("spectator private view")
	}
	for _, p := range watched.Players {
		for _, c := range p.Cards {
			if c.Rank != 0 {
				t.Fatal("spectator card leak")
			}
		}
	}
	actor := state.Players[state.CurrentPlayerIdx].ID
	other := "host"
	if actor == "host" {
		other = "guest"
	}
	h.send(clients[actor], "kabo-draw", game.KaboAction{Revision: state.Revision, Source: "deck"})
	readAll()
	if states[actor].Drawn == nil || states[other].Drawn != nil {
		t.Fatal("draw privacy")
	}
	h.send(clients[other], "kabo-discard", game.KaboAction{Revision: states[actor].Revision})
	h.read(clients[other], "error")
	for _, raw := range []string{`null`, `{}`, `{"revision":"bad"}`, `{"slotIds":[1,2]}`} {
		h.send(clients[actor], "kabo-replace", json.RawMessage(raw))
		h.read(clients[actor], "error")
	}
	// Reconnect during a private drawn-card decision.
	before := states[actor]
	clients[actor].Close()
	room := getRoom(h.code)
	h.await(func() bool { return room.disconnectTimers[actor] != nil })
	clients[actor] = h.connect(actor, "", "")
	after := decodeAdditional[kaboSnapshot](t, h.read(clients[actor], "kabo-state"))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("reconnect lost private decision")
	}
	states[actor] = after
	h.send(clients[actor], "kabo-call", game.KaboAction{Revision: after.Revision})
	readAll()
	if states[actor].FinalTurns != 1 || states[actor].Phase != game.KaboResolve {
		t.Fatal("call ended turn early")
	}
	h.send(clients[actor], "kabo-discard", game.KaboAction{Revision: states[actor].Revision})
	readAll()
	if states[actor].Phase == game.KaboPower {
		h.send(clients[actor], "kabo-skip", game.KaboAction{Revision: states[actor].Revision})
		readAll()
	}
	state = states["host"]
	if state.Phase != game.KaboDraw || state.Players[state.CurrentPlayerIdx].ID != other {
		t.Fatal("final turn wrong")
	}
	h.send(clients[other], "kabo-draw", game.KaboAction{Revision: state.Revision, Source: "deck"})
	readAll()
	h.send(clients[other], "kabo-discard", game.KaboAction{Revision: states[other].Revision})
	readAll()
	if states[other].Phase == game.KaboPower {
		h.send(clients[other], "kabo-skip", game.KaboAction{Revision: states[other].Revision})
		readAll()
	}
	state = states["host"]
	if state.Phase != game.KaboRoundOver {
		t.Fatal("round not complete")
	}
	for _, p := range state.Players {
		if p.RoundScore == nil || p.TotalScore != *p.RoundScore {
			t.Fatal("score mismatch")
		}
		for _, c := range p.Cards {
			if c.Rank == 0 {
				t.Fatal("result cards not revealed")
			}
		}
	}
	h.send(clients["guest"], "kabo-next-round", game.KaboAction{Revision: state.Revision})
	h.read(clients["guest"], "error")
	h.send(clients["host"], "kabo-next-round", game.KaboAction{Revision: state.Revision})
	readAll()
	if states["host"].Phase != game.KaboInitial || states["host"].Round != 2 {
		t.Fatal("redeal failed")
	}
	for _, p := range states["host"].Players {
		for _, c := range p.Cards {
			if c.Rank != 0 {
				t.Fatal("redeal cards exposed")
			}
		}
	}
	h.send(clients["host"], "return-to-lobby", nil)
	h.read(clients["host"], "error")
	h.send(clients["host"], "leave-room", nil)
	h.read(clients["host"], "room-left")
	terminal := decodeAdditional[kaboSnapshot](t, h.read(clients["guest"], "kabo-state"))
	if terminal.Phase != game.KaboGameOver {
		t.Fatal("departure did not end two-player match")
	}
	h.send(clients["guest"], "return-to-lobby", nil)
	if string(h.read(clients["guest"], "state")) != "null" {
		t.Fatal("lobby return did not clear state")
	}
}

func TestKaboRoomCapacityStatsAndHostTransfer(t *testing.T) {
	if roomCapacity("kabo") != 6 {
		t.Fatal("capacity")
	}
	h := newAdditionalWSTest(t, "kabo")
	host := h.connect("host", "create", "")
	h.join(host, "Host", false)
	for i, id := range []string{"p1", "p2", "p3", "p4", "p5"} {
		c := h.connect(id, "", "")
		h.join(c, id, false)
		if i == 4 {
			h.send(host, "set-room-visibility", map[string]bool{"public": true})
			h.sync(host)
		}
	}
	w := httptest.NewRecorder()
	handleRoomStats(w, httptest.NewRequest("GET", "/api/room-stats", nil))
	stats := decodeAdditional[roomStats](t, w.Body.Bytes())
	if stats.Games["kabo"].Players < 6 || stats.Games["kabo"].PublicRooms < 1 {
		t.Fatal("Kabo stats missing")
	}
	extra := h.connect("extra", "", "")
	h.send(extra, "join", map[string]string{"playerName": "extra"})
	h.read(extra, "error")
	h.send(host, "leave-room", nil)
	h.read(host, "room-left")
	room := getRoom(h.code)
	room.mu.Lock()
	defer room.mu.Unlock()
	if len(room.players) != 5 || room.hostID == "" || room.hostID == "host" {
		t.Fatal("host transfer failed")
	}
}
