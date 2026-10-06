package main

import (
	"encoding/json"
	"net/http"
)

func countRoomPlayers() int {
	roomsMu.RLock()
	snapshot := make([]*Room, 0, len(rooms))
	for _, room := range rooms {
		snapshot = append(snapshot, room)
	}
	roomsMu.RUnlock()
	total := 0
	for _, room := range snapshot {
		room.mu.Lock()
		if room.created {
			total += len(room.players)
		}
		room.mu.Unlock()
	}
	return total
}

func handleRoomStats(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(writer).Encode(map[string]int{"players": countRoomPlayers()})
}
