package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"coup-server/game"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Room represents a game room
type Room struct {
	mu           sync.Mutex
	code         string
	variant      game.VariantKey
	gameState    *game.GameState
	players      map[string]*PlayerConn // playerID -> PlayerConn
	hostID       string
	created      bool
	connections  map[string]*websocket.Conn // connID -> ws conn
	connPlayer   map[string]string          // connID -> playerID
	lastActivity time.Time
}

type PlayerConn struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Message types
type InMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type OutMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Room manager
var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
)

func getOrCreateRoom(code string, variant game.VariantKey) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	if room, ok := rooms[code]; ok {
		return room
	}

	room := &Room{
		code:         code,
		variant:      variant,
		players:      make(map[string]*PlayerConn),
		connections:  make(map[string]*websocket.Conn),
		connPlayer:   make(map[string]string),
		lastActivity: time.Now(),
	}
	rooms[code] = room
	return room
}

func getRoom(code string) *Room {
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	return rooms[code]
}

func (r *Room) broadcast(msg OutMessage) {
	data, _ := json.Marshal(msg)
	for connID, conn := range r.connections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("broadcast error to %s: %v", connID, err)
		}
	}
}

func (r *Room) sendTo(connID string, msg OutMessage) {
	data, _ := json.Marshal(msg)
	if conn, ok := r.connections[connID]; ok {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (r *Room) sendToPlayer(playerID string, msg OutMessage) {
	for connID, pID := range r.connPlayer {
		if pID == playerID {
			r.sendTo(connID, msg)
		}
	}
}

func (r *Room) playerList() []PlayerConn {
	list := make([]PlayerConn, 0, len(r.players))
	for _, p := range r.players {
		list = append(list, *p)
	}
	return list
}

func (r *Room) broadcastState() {
	if r.gameState == nil {
		return
	}
	// Find which playerIDs are in the active game
	gamePlayers := make(map[string]bool)
	for _, p := range r.gameState.Players {
		gamePlayers[p.ID] = true
	}
	stateData, _ := json.Marshal(OutMessage{Type: "state", Payload: r.gameState})
	spectateData, _ := json.Marshal(OutMessage{Type: "spectate-state", Payload: r.gameState})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			conn.WriteMessage(websocket.TextMessage, stateData)
		} else {
			conn.WriteMessage(websocket.TextMessage, spectateData)
		}
	}
}

func handleWS(w http.ResponseWriter, req *http.Request) {
	roomCode := req.URL.Query().Get("room")
	action := req.URL.Query().Get("action")
	playerID := req.URL.Query().Get("playerId")
	variantStr := req.URL.Query().Get("variant")

	if roomCode == "" {
		http.Error(w, "room required", http.StatusBadRequest)
		return
	}

	variant := game.NormalizeVariant(variantStr)

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	connID := uuid.New().String()
	log.Printf("[WS] New connection: connID=%s playerID=%s room=%s action=%s variant=%s", connID[:8], playerID[:8], roomCode, action, variant)

	room := getOrCreateRoom(roomCode, variant)
	room.mu.Lock()

	if action == "create" {
		room.created = true
		log.Printf("[ROOM] Created room %s by player %s", roomCode, playerID[:8])
	}

	if !room.created && action != "create" {
		data, _ := json.Marshal(OutMessage{Type: "error", Payload: map[string]string{"message": "Incorrect Game Code or No Session Found"}})
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		room.mu.Unlock()
		return
	}

	// Store connection
	room.connections[connID] = conn
	room.connPlayer[connID] = playerID
	log.Printf("[ROOM] %s: stored conn %s -> player %s (total conns: %d, players: %d)", roomCode, connID[:8], playerID[:8], len(room.connections), len(room.players))

	// If game already started, check reconnection
	if room.gameState != nil {
		isReconnecting := false
		for _, p := range room.gameState.Players {
			if p.ID == playerID {
				isReconnecting = true
				break
			}
		}

		if !isReconnecting {
			data, _ := json.Marshal(OutMessage{Type: "error", Payload: map[string]string{"message": "The Game Already Started"}})
			conn.WriteMessage(websocket.TextMessage, data)
			conn.Close()
			delete(room.connections, connID)
			delete(room.connPlayer, connID)
			room.mu.Unlock()
			return
		}

		room.sendTo(connID, OutMessage{Type: "state", Payload: room.gameState})
		room.mu.Unlock()
	} else {
		room.sendTo(connID, OutMessage{Type: "waiting", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": false,
		}})
		room.mu.Unlock()
	}

	// Read loop
	defer func() {
		room.mu.Lock()
		delete(room.connections, connID)
		pID := room.connPlayer[connID]
		delete(room.connPlayer, connID)

		// Handle disconnect
		if room.gameState != nil && room.gameState.Phase != game.PhaseWaiting && room.gameState.Phase != game.PhaseGameOver && pID != "" {
			for _, p := range room.gameState.Players {
				if p.ID == pID && p.IsAlive {
					game.EliminatePlayer(room.gameState, pID)
					room.broadcastState()
					break
				}
			}
		}

		if (room.gameState == nil || room.gameState.Phase == game.PhaseWaiting) && pID != "" {
			delete(room.players, pID)
			if pID == room.hostID {
				room.hostID = ""
				for id := range room.players {
					room.hostID = id
					break
				}
			}
			room.broadcast(OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
			}})
		}
		room.mu.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg InMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[MSG] parse error from conn %s: %v", connID[:8], err)
			continue
		}

		log.Printf("[MSG] room=%s player=%s type=%s payload=%s", roomCode, playerID[:8], msg.Type, string(msg.Payload))
		room.mu.Lock()
		room.lastActivity = time.Now()
		handleMessage(room, connID, playerID, msg)
		room.mu.Unlock()
	}
}

func handleMessage(room *Room, connID, playerID string, msg InMessage) {
	switch msg.Type {
	case "join":
		var payload struct {
			PlayerName string `json:"playerName"`
		}
		json.Unmarshal(msg.Payload, &payload)

		if room.gameState != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "The Game Already Started"}})
			return
		}

		// Check name taken
		name := strings.TrimSpace(payload.PlayerName)
		for _, p := range room.players {
			if strings.EqualFold(strings.TrimSpace(p.Name), name) && p.ID != playerID {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Name is already taken"}})
				return
			}
		}

		if len(room.players) == 0 && room.hostID == "" {
			room.hostID = playerID
		}

		room.players[playerID] = &PlayerConn{ID: playerID, Name: name}
		log.Printf("[JOIN] room=%s player=%s name=%q (total players: %d)", room.code, playerID[:8], name, len(room.players))

		room.broadcast(OutMessage{Type: "players-updated", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
		}})

	case "start-game":
		if playerID != room.hostID {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can start the game"}})
			return
		}
		if len(room.players) < 2 {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Need at least 2 players to start"}})
			return
		}

		playerList := make([]struct{ ID, Name string }, 0, len(room.players))
		for _, p := range room.players {
			playerList = append(playerList, struct{ ID, Name string }{p.ID, p.Name})
		}

		room.gameState = game.InitializeGame(playerList, room.variant)
		room.broadcast(OutMessage{Type: "game-started", Payload: map[string]interface{}{"gameState": room.gameState}})

	case "return-to-lobby":
		hostShort := "(none)"
		if len(room.hostID) >= 8 { hostShort = room.hostID[:8] }
		log.Printf("[RTL] player=%s host=%s gameState=%v phase=%v", playerID[:8], hostShort, room.gameState != nil, func() string { if room.gameState != nil { return string(room.gameState.Phase) }; return "nil" }())
		if room.gameState == nil {
			log.Printf("[RTL] gameState is nil, sending lobby state to player")
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
			}})
			return
		}
		if playerID != room.hostID && room.gameState.Phase != game.PhaseGameOver {
			log.Printf("[RTL] blocked: not host and not game_over")
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can return to lobby"}})
			return
		}
		log.Printf("[RTL] returning to lobby, clearing game state")
		room.gameState = nil

		// Only keep connected players
		activeIDs := make(map[string]bool)
		for _, pid := range room.connPlayer {
			activeIDs[pid] = true
		}
		for pid := range room.players {
			if !activeIDs[pid] {
				delete(room.players, pid)
			}
		}
		if room.hostID != "" {
			if _, ok := room.players[room.hostID]; !ok {
				room.hostID = ""
				for id := range room.players {
					room.hostID = id
					break
				}
			}
		}

		room.broadcast(OutMessage{Type: "state", Payload: nil})
		room.broadcast(OutMessage{Type: "players-updated", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": false,
		}})

	case "exit-game":
		if room.gameState == nil || room.gameState.Phase == game.PhaseGameOver {
			return
		}
		// Eliminate this player (reveal all cards)
		game.VoluntaryExit(room.gameState, playerID)
		room.broadcastState()

		// Send this player back to lobby
		room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
		room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
		}})

	case "spectate":
		if room.gameState != nil {
			room.sendTo(connID, OutMessage{Type: "spectate-state", Payload: room.gameState})
		}

	case "kick-player":
		if playerID != room.hostID {
			return
		}
		if room.gameState != nil && room.gameState.Phase != game.PhaseWaiting {
			return
		}

		var payload struct {
			PlayerID string `json:"playerId"`
		}
		json.Unmarshal(msg.Payload, &payload)

		delete(room.players, payload.PlayerID)

		room.broadcast(OutMessage{Type: "players-updated", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
		}})

		room.sendToPlayer(payload.PlayerID, OutMessage{Type: "kicked", Payload: map[string]string{"message": "You have been kicked from the game"}})

	case "action":
		if room.gameState == nil {
			return
		}
		var action game.ActionRequest
		json.Unmarshal(msg.Payload, &action)

		if err := game.PerformAction(room.gameState, action); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "block":
		if room.gameState == nil {
			return
		}
		var block game.BlockRequest
		json.Unmarshal(msg.Payload, &block)

		if err := game.BlockAction(room.gameState, block); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "pass-block":
		if room.gameState == nil || playerID == "" {
			return
		}
		game.PassBlock(room.gameState, playerID)
		room.broadcastState()

	case "challenge":
		if room.gameState == nil {
			return
		}
		var challenge game.ChallengeRequest
		json.Unmarshal(msg.Payload, &challenge)

		if err := game.ChallengeAction(room.gameState, challenge); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "pass-challenge":
		if room.gameState == nil || playerID == "" {
			return
		}
		game.PassChallenge(room.gameState, playerID)
		room.broadcastState()

	case "exchange":
		if room.gameState == nil || playerID == "" {
			return
		}
		var payload struct {
			KeptCardIDs []string `json:"keptCardIds"`
		}
		json.Unmarshal(msg.Payload, &payload)

		if err := game.ExchangeCards(room.gameState, playerID, payload.KeptCardIDs); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "interrogate-select":
		if room.gameState == nil || playerID == "" {
			return
		}
		var payload struct {
			CardID string `json:"cardId"`
		}
		json.Unmarshal(msg.Payload, &payload)

		if err := game.SelectInterrogateCard(room.gameState, playerID, payload.CardID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "interrogate-decision":
		if room.gameState == nil || playerID == "" {
			return
		}
		var payload struct {
			Decision string `json:"decision"`
		}
		json.Unmarshal(msg.Payload, &payload)

		if err := game.DecideInterrogate(room.gameState, playerID, payload.Decision); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastState()

	case "lose-influence":
		if room.gameState == nil || playerID == "" {
			return
		}
		var payload struct {
			CardID string `json:"cardId"`
		}
		json.Unmarshal(msg.Payload, &payload)

		game.LoseInfluence(room.gameState, playerID, payload.CardID)
		room.broadcastState()

	case "get-state":
		if room.gameState != nil {
			room.sendTo(connID, OutMessage{Type: "state", Payload: room.gameState})
		}

	case "ping":
		room.sendTo(connID, OutMessage{Type: "pong"})
	}
}

func generateCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, 5)
	for i := range code {
		code[i] = letters[r.Intn(len(letters))]
	}
	return string(code)
}

func handleGenerateCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	code := generateCode()
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}

func handleVariantConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	variant := game.NormalizeVariant(r.URL.Query().Get("variant"))
	config := game.GetVariantConfig(variant)
	json.NewEncoder(w).Encode(config)
}

func main() {
	// Room cleanup goroutine - remove rooms inactive for 10 minutes
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			roomsMu.Lock()
			for code, room := range rooms {
				room.mu.Lock()
				if len(room.connections) == 0 && time.Since(room.lastActivity) > 10*time.Minute {
					log.Printf("[ROOM] Cleaning up inactive room %s", code)
					delete(rooms, code)
				}
				room.mu.Unlock()
			}
			roomsMu.Unlock()
		}
	}()

	http.HandleFunc("/ws", handleWS)
	http.HandleFunc("/api/generate-code", handleGenerateCode)
	http.HandleFunc("/api/variant-config", handleVariantConfig)

	// Serve static files (textures, icons, etc)
	http.Handle("/textures/", http.StripPrefix("/textures/", http.FileServer(http.Dir("public/textures"))))
	http.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir("public/icons"))))
	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFile(w, r, "public/manifest.json")
	})
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "public/sw.js")
	})

	// Serve test client
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, "client.html")
			return
		}
		http.NotFound(w, r)
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "ok")
	})

	port := "8080"
	log.Printf("Coup server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
