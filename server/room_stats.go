package main

import (
	"encoding/json"
	"net/http"
)

type gameRoomStats struct {
	PublicRooms int `json:"publicRooms"`
	Players     int `json:"players"`
}

type roomStats struct {
	Players int                      `json:"players"`
	Games   map[string]gameRoomStats `json:"games"`
}

func collectRoomStats() roomStats {
	roomsMu.RLock()
	snapshot := make([]*Room, 0, len(rooms))
	for _, room := range rooms {
		snapshot = append(snapshot, room)
	}
	roomsMu.RUnlock()
	stats := roomStats{Games: map[string]gameRoomStats{
		"coup": {}, "poker": {}, "ludo": {}, "nquestions": {},
		"commune": {}, "twentynine": {}, "hearts": {}, "uno": {},
		"bluff": {}, "blackjack": {}, "tictactoe": {}, "scrabble": {},
		"kabo": {},
	}}
	for _, room := range snapshot {
		room.mu.Lock()
		if room.created {
			members := len(room.players)
			stats.Players += members
			gameStats := stats.Games[room.gameType]
			gameStats.Players += members
			if room.isPublic && members > 0 {
				gameStats.PublicRooms++
			}
			stats.Games[room.gameType] = gameStats
		}
		room.mu.Unlock()
	}
	return stats
}

func countRoomPlayers() int {
	return collectRoomStats().Players
}

func handleRoomStats(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(writer).Encode(collectRoomStats())
}
