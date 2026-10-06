package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"coup-server/game"
)

func TestRoomStatsIncludesPrivatePublicAndStartedRooms(t *testing.T) {
	original := rooms
	rooms = map[string]*Room{
		"private":   discoveryRoom("private", "uno", false, 2),
		"public":    discoveryRoom("public", "ludo", true, 3),
		"started":   discoveryRoom("started", "coup", false, 4),
		"empty":     discoveryRoom("empty", "poker", true, 0),
		"uncreated": discoveryRoom("uncreated", "hearts", false, 1),
	}
	t.Cleanup(func() { rooms = original })
	rooms["uncreated"].created = false
	rooms["started"].gameState = &game.GameState{}
	rooms["private"].connPlayer = map[string]string{"tab1": "host-player-id", "tab2": "host-player-id"}
	response := httptest.NewRecorder()
	handleRoomStats(response, httptest.NewRequest("GET", "/api/room-stats", nil))
	var stats struct {
		Players int `json:"players"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Players != 9 {
		t.Fatalf("got %d players, want 9", stats.Players)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("room stats should not be cached")
	}
	delete(rooms["public"].players, "player-id-1")
	if countRoomPlayers() != 8 {
		t.Fatal("count did not reflect membership changes")
	}
	bad := httptest.NewRecorder()
	handleRoomStats(bad, httptest.NewRequest("POST", "/api/room-stats", nil))
	if bad.Code != 405 {
		t.Fatalf("got status %d, want 405", bad.Code)
	}
}
