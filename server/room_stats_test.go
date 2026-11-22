package main

import (
	"encoding/json"
	"fmt"
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

func TestRoomStatsPerGameCountsAllNonemptyPublicRoomsWithoutDiscoveryLimit(t *testing.T) {
	original := rooms
	rooms = make(map[string]*Room)
	t.Cleanup(func() { rooms = original })
	for i := 0; i < 7; i++ {
		code := fmt.Sprintf("PUBLIC-%d", i)
		rooms[code] = discoveryRoom(code, "uno", true, 1)
	}
	rooms["full"] = discoveryRoom("full", "uno", true, 6)
	rooms["started"] = discoveryRoom("started", "uno", true, 2)
	rooms["started"].unoState = &game.UNOState{}
	rooms["empty"] = discoveryRoom("empty", "uno", true, 0)
	rooms["private"] = discoveryRoom("private", "uno", false, 3)
	rooms["other"] = discoveryRoom("other", "ludo", false, 4)
	rooms["uncreated"] = discoveryRoom("uncreated", "uno", true, 5)
	rooms["uncreated"].created = false
	rooms["private"].connPlayer = map[string]string{
		"tab1": rooms["private"].hostID, "tab2": rooms["private"].hostID, "watcher": "spectator",
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/room-stats", nil)
	request.Header.Set("Origin", "https://timecrushers.pages.dev")
	handleRoomStats(response, request)
	var stats roomStats
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Players != 22 || stats.Games["uno"] != (gameRoomStats{PublicRooms: 9, Players: 18}) ||
		stats.Games["ludo"] != (gameRoomStats{Players: 4}) {
		t.Fatalf("unexpected counts: %+v", stats)
	}
	if len(stats.Games) != 13 {
		t.Fatalf("got %d game entries, want 13", len(stats.Games))
	}
	total := 0
	for gameType, gameStats := range stats.Games {
		total += gameStats.Players
		if gameType != "uno" && gameType != "ludo" && gameStats != (gameRoomStats{}) {
			t.Fatalf("unused game is not zero: %s %+v", gameType, gameStats)
		}
	}
	if total != stats.Players {
		t.Fatal("per-game counts do not sum to the total")
	}
	if len(listPublicRooms("uno")) != 5 {
		t.Fatal("discovery display limit changed")
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("cross-origin frontend cannot read stats")
	}
}

func TestRoomStatsReflectsVisibilityAndMembershipChanges(t *testing.T) {
	original := rooms
	room := discoveryRoom("ABCDE", "coup", false, 2)
	rooms = map[string]*Room{room.code: room}
	t.Cleanup(func() { rooms = original })
	if collectRoomStats().Games["coup"] != (gameRoomStats{Players: 2}) {
		t.Fatal("private members missing")
	}
	room.isPublic = true
	if collectRoomStats().Games["coup"] != (gameRoomStats{Players: 2, PublicRooms: 1}) {
		t.Fatal("public toggle not reflected")
	}
	room.players = map[string]*PlayerConn{}
	if collectRoomStats().Games["coup"] != (gameRoomStats{}) {
		t.Fatal("empty public room counted")
	}
}
