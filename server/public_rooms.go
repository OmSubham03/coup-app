package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type publicRoomSummary struct {
	Code     string `json:"code"`
	GameType string `json:"gameType"`
	Variant  string `json:"variant"`
	Members  int    `json:"members"`
	Capacity int    `json:"capacity"`
}

func roomCapacity(gameType string) int {
	switch gameType {
	case "coup", "uno":
		return 6
	case "poker":
		return 8
	case "ludo", "twentynine", "hearts":
		return 4
	case "commune", "nquestions":
		return 10
	default:
		return 0
	}
}

func (room *Room) hasStartedGame() bool {
	return room.gameState != nil || room.pokerState != nil || room.ludoState != nil ||
		room.nqState != nil || room.communeState != nil || room.tnState != nil ||
		room.heartsState != nil || room.unoState != nil
}

func (room *Room) availablePublicRoom() bool {
	capacity := roomCapacity(room.gameType)
	return room.created && room.isPublic && room.hostID != "" && len(room.players) > 0 &&
		capacity > 0 && len(room.players) < capacity && !room.hasStartedGame()
}

func (room *Room) setPublicVisibility(playerID string, public bool) error {
	if playerID == "" || room.hostID != playerID || room.players[playerID] == nil {
		return fmt.Errorf("Only the host can change room visibility")
	}
	if room.hasStartedGame() {
		return fmt.Errorf("Room visibility can only be changed in the lobby")
	}
	room.isPublic = public
	return nil
}

func (room *Room) visibilityMessage() OutMessage {
	return OutMessage{Type: "room-visibility", Payload: map[string]interface{}{
		"public": room.isPublic, "hostId": room.hostID,
	}}
}

func (room *Room) sendRoomVisibility(connID string) {
	room.sendTo(connID, room.visibilityMessage())
}

func (room *Room) broadcastVisibility() {
	room.broadcast(room.visibilityMessage())
}

func listPublicRooms(gameType string) []publicRoomSummary {
	roomsMu.RLock()
	snapshot := make([]*Room, 0, len(rooms))
	for _, room := range rooms {
		snapshot = append(snapshot, room)
	}
	roomsMu.RUnlock()
	sort.Slice(snapshot, func(first, second int) bool { return snapshot[first].code < snapshot[second].code })
	result := make([]publicRoomSummary, 0, 5)
	for _, room := range snapshot {
		room.mu.Lock()
		if room.gameType == gameType && room.availablePublicRoom() {
			result = append(result, publicRoomSummary{
				Code: room.code, GameType: room.gameType, Variant: string(room.variant),
				Members: len(room.players), Capacity: roomCapacity(room.gameType),
			})
		}
		room.mu.Unlock()
		if len(result) == 5 {
			break
		}
	}
	return result
}

func handlePublicRooms(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "application/json")
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gameType := request.URL.Query().Get("gameType")
	if roomCapacity(gameType) == 0 {
		http.Error(writer, "invalid game type", http.StatusBadRequest)
		return
	}
	json.NewEncoder(writer).Encode(listPublicRooms(gameType))
}
