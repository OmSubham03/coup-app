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
	mu               sync.Mutex
	code             string
	gameType         string // "coup", "poker", "ludo", "nquestions", "commune", "twentynine", or "hearts"
	variant          game.VariantKey
	gameState        *game.GameState
	pokerState       *game.PokerState
	pokerConfig      *PokerConfig
	ludoState        *game.LudoState
	ludoColors       map[string]string // playerID -> color choice
	nqState          *game.NQState
	nqConfig         *NQConfig
	communeState     *game.CommuneState
	tnState          *game.TwentyNineState
	heartsState      *game.HeartsState
	players          map[string]*PlayerConn // playerID -> PlayerConn
	hostID           string
	created          bool
	connections      map[string]*websocket.Conn // connID -> ws conn
	connPlayer       map[string]string          // connID -> playerID
	disconnectTimers map[string]*time.Timer     // playerID -> pending elimination timer
	lastActivity     time.Time
}

type PokerConfig struct {
	BuyIn           int  `json:"buyIn"`
	SmallBlind      int  `json:"smallBlind"`
	BigBlindEnabled bool `json:"bigBlindEnabled"`
}

type NQConfig struct {
	MaxQuestions int `json:"maxQuestions"`
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

func getOrCreateRoom(code string, gameType string, variant game.VariantKey) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	if room, ok := rooms[code]; ok {
		return room
	}

	room := &Room{
		code:             code,
		gameType:         gameType,
		variant:          variant,
		players:          make(map[string]*PlayerConn),
		ludoColors:       make(map[string]string),
		connections:      make(map[string]*websocket.Conn),
		connPlayer:       make(map[string]string),
		disconnectTimers: make(map[string]*time.Timer),
		lastActivity:     time.Now(),
	}
	rooms[code] = room
	return room
}

func getRoom(code string) *Room {
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	return rooms[code]
}

func findPlayerActiveRoom(playerID string) *Room {
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	for _, room := range rooms {
		room.mu.Lock()
		// Check if player is in this room
		if _, exists := room.players[playerID]; exists {
			// Check if game is active
			gameActive := (room.gameState != nil && room.gameState.Phase != game.PhaseWaiting && room.gameState.Phase != game.PhaseGameOver) ||
				(room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseGameOver) ||
				(room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseFinished) ||
				(room.nqState != nil && room.nqState.Phase != game.NQPhaseFinished) ||
				(room.communeState != nil && room.communeState.Phase != game.CommunePhaseFinished) ||
				(room.tnState != nil && room.tnState.Phase != game.TN_PhaseGameOver) ||
				(room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseGameOver)
			room.mu.Unlock()
			if gameActive {
				return room
			}
		} else {
			room.mu.Unlock()
		}
	}
	return nil
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
	if r.gameType == "poker" {
		r.broadcastPokerState()
		return
	}
	if r.gameType == "ludo" {
		r.broadcastLudoState()
		return
	}
	if r.gameType == "nquestions" {
		r.broadcastNQState()
		return
	}
	if r.gameType == "commune" {
		r.broadcastCommuneState()
		return
	}
	if r.gameType == "twentynine" {
		r.broadcastTNState()
		return
	}
	if r.gameType == "hearts" {
		r.broadcastHTState()
		return
	}
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

func logPokerHands(r *Room) {
	if r.pokerState == nil {
		return
	}
	log.Printf("[CARDS] room=%s Poker hand #%d dealt", r.code, r.pokerState.HandNumber)
	for _, p := range r.pokerState.Players {
		if p.IsActive && len(p.HoleCards) == 2 {
			log.Printf("[CARDS] room=%s %s: %s %s", r.code, p.Name, p.HoleCards[0], p.HoleCards[1])
		}
	}
}

func (r *Room) broadcastPokerState() {
	if r.pokerState == nil {
		return
	}
	// Find which playerIDs are in the active game
	gamePlayers := make(map[string]bool)
	for _, p := range r.pokerState.Players {
		gamePlayers[p.ID] = true
	}
	// Send personalized state to each connection
	spectatorState, _ := json.Marshal(OutMessage{Type: "poker-spectate", Payload: r.createPersonalizedPokerState("")})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			personalized := r.createPersonalizedPokerState(pID)
			data, _ := json.Marshal(OutMessage{Type: "poker-state", Payload: personalized})
			conn.WriteMessage(websocket.TextMessage, data)
		} else {
			conn.WriteMessage(websocket.TextMessage, spectatorState)
		}
	}
}

func (r *Room) createPersonalizedPokerState(viewerID string) *game.PokerState {
	ps := r.pokerState
	// Deep copy players with hidden hole cards
	players := make([]game.PokerPlayer, len(ps.Players))
	for i, p := range ps.Players {
		players[i] = p
		if p.ID != viewerID && ps.Phase != game.PokerPhaseShowdown && ps.Phase != game.PokerPhaseGameOver {
			players[i].HoleCards = nil // Hide other players' cards
		}
		// At showdown, show cards of active non-folded players
		if (ps.Phase == game.PokerPhaseShowdown || ps.Phase == game.PokerPhaseGameOver) && !p.Folded {
			players[i].HoleCards = p.HoleCards
		}
	}

	return &game.PokerState{
		ID:               ps.ID,
		Players:          players,
		CommunityCards:   ps.CommunityCards,
		Phase:            ps.Phase,
		Pots:             ps.Pots,
		CurrentBet:       ps.CurrentBet,
		MinRaise:         ps.MinRaise,
		DealerIndex:      ps.DealerIndex,
		CurrentPlayerIdx: ps.CurrentPlayerIdx,
		SmallBlind:       ps.SmallBlind,
		BigBlind:         ps.BigBlind,
		BuyIn:            ps.BuyIn,
		HandNumber:       ps.HandNumber,
		Log:              ps.Log,
		LastAction:       ps.LastAction,
		Scoreboard:       ps.Scoreboard,
		Winner:           ps.Winner,
		PendingRebuys:    ps.PendingRebuys,
		PendingJoins:     ps.PendingJoins,
	}
}

func (r *Room) broadcastLudoState() {
	if r.ludoState == nil {
		return
	}
	// Find which playerIDs are in the active game
	gamePlayers := make(map[string]bool)
	for _, p := range r.ludoState.Players {
		gamePlayers[p.ID] = true
	}
	stateData, _ := json.Marshal(OutMessage{Type: "ludo-state", Payload: r.ludoState})
	spectateData, _ := json.Marshal(OutMessage{Type: "ludo-spectate", Payload: r.ludoState})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			conn.WriteMessage(websocket.TextMessage, stateData)
		} else {
			conn.WriteMessage(websocket.TextMessage, spectateData)
		}
	}
}

func (r *Room) broadcastNQState() {
	if r.nqState == nil {
		return
	}
	gamePlayers := make(map[string]bool)
	for _, p := range r.nqState.Players {
		gamePlayers[p.ID] = true
	}
	// Send giver the full state, guessers get sanitized (no secret word)
	sanitized := game.SanitizeNQStateForGuessers(r.nqState)
	sanitizedData, _ := json.Marshal(OutMessage{Type: "nq-state", Payload: sanitized})
	spectateData, _ := json.Marshal(OutMessage{Type: "nq-spectate", Payload: sanitized})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			// Check if this player is the giver
			isGiver := false
			for _, p := range r.nqState.Players {
				if p.ID == pID && p.IsGiver {
					isGiver = true
					break
				}
			}
			if isGiver {
				giverData, _ := json.Marshal(OutMessage{Type: "nq-state", Payload: r.nqState})
				conn.WriteMessage(websocket.TextMessage, giverData)
			} else {
				conn.WriteMessage(websocket.TextMessage, sanitizedData)
			}
		} else {
			conn.WriteMessage(websocket.TextMessage, spectateData)
		}
	}
}

func (r *Room) broadcastCommuneState() {
	if r.communeState == nil {
		return
	}
	gamePlayers := make(map[string]bool)
	for _, p := range r.communeState.Players {
		gamePlayers[p.ID] = true
	}
	spectatorState := game.SanitizeCommuneStateForPlayer(r.communeState, "")
	spectateData, _ := json.Marshal(OutMessage{Type: "commune-spectate", Payload: spectatorState})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			personalized := game.SanitizeCommuneStateForPlayer(r.communeState, pID)
			data, _ := json.Marshal(OutMessage{Type: "commune-state", Payload: personalized})
			conn.WriteMessage(websocket.TextMessage, data)
		} else {
			conn.WriteMessage(websocket.TextMessage, spectateData)
		}
	}
}

func (r *Room) broadcastTNState() {
	if r.tnState == nil {
		return
	}
	gamePlayers := make(map[string]bool)
	for _, p := range r.tnState.Players {
		gamePlayers[p.ID] = true
	}
	spectatorState := game.SanitizeTNStateForPlayer(r.tnState, "")
	spectateData, _ := json.Marshal(OutMessage{Type: "tn-spectate", Payload: spectatorState})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			personalized := game.SanitizeTNStateForPlayer(r.tnState, pID)
			data, _ := json.Marshal(OutMessage{Type: "tn-state", Payload: personalized})
			conn.WriteMessage(websocket.TextMessage, data)
		} else {
			conn.WriteMessage(websocket.TextMessage, spectateData)
		}
	}
}

func (r *Room) broadcastHTState() {
	if r.heartsState == nil {
		return
	}
	gamePlayers := make(map[string]bool)
	for _, p := range r.heartsState.Players {
		gamePlayers[p.ID] = true
	}
	spectatorState := game.SanitizeHTStateForPlayer(r.heartsState, "")
	spectateData, _ := json.Marshal(OutMessage{Type: "ht-spectate", Payload: spectatorState})
	for connID, conn := range r.connections {
		pID := r.connPlayer[connID]
		if gamePlayers[pID] {
			personalized := game.SanitizeHTStateForPlayer(r.heartsState, pID)
			data, _ := json.Marshal(OutMessage{Type: "ht-state", Payload: personalized})
			conn.WriteMessage(websocket.TextMessage, data)
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
	gameType := req.URL.Query().Get("gameType")

	if roomCode == "" {
		http.Error(w, "room required", http.StatusBadRequest)
		return
	}

	if gameType == "" {
		gameType = "coup" // Default for backward compatible
	}

	variant := game.NormalizeVariant(variantStr)

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	connID := uuid.New().String()
	log.Printf("[WS] New connection: connID=%s playerID=%s room=%s action=%s variant=%s gameType=%s", connID[:8], playerID[:8], roomCode, action, variant, gameType)

	// Check if player is already in an active game in a different room
	if existingRoom := findPlayerActiveRoom(playerID); existingRoom != nil && existingRoom.code != roomCode {
		log.Printf("[REDIRECT] Player %s already in active game %s, redirecting from attempted join to %s", playerID[:8], existingRoom.code, roomCode)
		data, _ := json.Marshal(OutMessage{Type: "redirect", Payload: map[string]string{
			"message":  "You are already in an active game. Redirecting...",
			"roomCode": existingRoom.code,
			"gameType": existingRoom.gameType,
		}})
		conn.WriteMessage(websocket.TextMessage, data)
		conn.Close()
		return
	}

	room := getOrCreateRoom(roomCode, gameType, variant)
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

	// Cancel any pending disconnect timer for this player
	if timer, ok := room.disconnectTimers[playerID]; ok {
		timer.Stop()
		delete(room.disconnectTimers, playerID)
		log.Printf("[RECONNECT] room=%s player=%s — cancelled disconnect timer", roomCode, playerID[:8])
	}

	// If game already started, check reconnection
	gameInProgress := room.gameState != nil || room.pokerState != nil || room.ludoState != nil || room.nqState != nil || room.communeState != nil || room.tnState != nil || room.heartsState != nil
	if gameInProgress {
		isReconnecting := false
		if room.gameType == "poker" && room.pokerState != nil {
			for _, p := range room.pokerState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameType == "ludo" && room.ludoState != nil {
			for _, p := range room.ludoState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameType == "nquestions" && room.nqState != nil {
			for _, p := range room.nqState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameType == "commune" && room.communeState != nil {
			for _, p := range room.communeState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameType == "twentynine" && room.tnState != nil {
			for _, p := range room.tnState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameType == "hearts" && room.heartsState != nil {
			for _, p := range room.heartsState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		} else if room.gameState != nil {
			for _, p := range room.gameState.Players {
				if p.ID == playerID {
					isReconnecting = true
					break
				}
			}
		}

		if !isReconnecting {
			// Allow spectating — send current game state as spectator
			if room.gameType == "poker" {
				personalized := room.createPersonalizedPokerState("")
				room.sendTo(connID, OutMessage{Type: "poker-spectate", Payload: personalized})
			} else if room.gameType == "ludo" {
				room.sendTo(connID, OutMessage{Type: "ludo-spectate", Payload: room.ludoState})
			} else if room.gameType == "nquestions" {
				sanitized := game.SanitizeNQStateForGuessers(room.nqState)
				room.sendTo(connID, OutMessage{Type: "nq-spectate", Payload: sanitized})
			} else if room.gameType == "commune" {
				spectatorState := game.SanitizeCommuneStateForPlayer(room.communeState, "")
				room.sendTo(connID, OutMessage{Type: "commune-spectate", Payload: spectatorState})
			} else if room.gameType == "twentynine" {
				spectatorState := game.SanitizeTNStateForPlayer(room.tnState, "")
				room.sendTo(connID, OutMessage{Type: "tn-spectate", Payload: spectatorState})
			} else if room.gameType == "hearts" {
				spectatorState := game.SanitizeHTStateForPlayer(room.heartsState, "")
				room.sendTo(connID, OutMessage{Type: "ht-spectate", Payload: spectatorState})
			} else {
				room.sendTo(connID, OutMessage{Type: "spectate-state", Payload: room.gameState})
			}
			room.mu.Unlock()
		} else {
			if room.gameType == "poker" {
				personalized := room.createPersonalizedPokerState(playerID)
				room.sendTo(connID, OutMessage{Type: "poker-state", Payload: personalized})
			} else if room.gameType == "ludo" {
				room.sendTo(connID, OutMessage{Type: "ludo-state", Payload: room.ludoState})
			} else if room.gameType == "nquestions" {
				// Giver gets full state, guessers get sanitized
				isGiver := false
				for _, p := range room.nqState.Players {
					if p.ID == playerID && p.IsGiver {
						isGiver = true
						break
					}
				}
				if isGiver {
					room.sendTo(connID, OutMessage{Type: "nq-state", Payload: room.nqState})
				} else {
					sanitized := game.SanitizeNQStateForGuessers(room.nqState)
					room.sendTo(connID, OutMessage{Type: "nq-state", Payload: sanitized})
				}
			} else if room.gameType == "commune" {
				personalized := game.SanitizeCommuneStateForPlayer(room.communeState, playerID)
				room.sendTo(connID, OutMessage{Type: "commune-state", Payload: personalized})
			} else if room.gameType == "twentynine" {
				personalized := game.SanitizeTNStateForPlayer(room.tnState, playerID)
				room.sendTo(connID, OutMessage{Type: "tn-state", Payload: personalized})
			} else if room.gameType == "hearts" {
				personalized := game.SanitizeHTStateForPlayer(room.heartsState, playerID)
				room.sendTo(connID, OutMessage{Type: "ht-state", Payload: personalized})
			} else {
				room.sendTo(connID, OutMessage{Type: "state", Payload: room.gameState})
			}
			room.mu.Unlock()
		}
	} else {
		room.sendTo(connID, OutMessage{Type: "waiting", Payload: map[string]interface{}{
			"players":    room.playerList(),
			"hostId":     room.hostID,
			"gameActive": false,
			"gameType":   room.gameType,
		}})
		room.mu.Unlock()
	}

	// Read loop
	defer func() {
		room.mu.Lock()
		delete(room.connections, connID)
		pID := room.connPlayer[connID]
		delete(room.connPlayer, connID)

		// Check if this player still has another active connection
		hasOtherConn := false
		for _, pid := range room.connPlayer {
			if pid == pID {
				hasOtherConn = true
				break
			}
		}

		if !hasOtherConn && pID != "" {
			gameInProgress := false
			if room.gameType == "poker" {
				gameInProgress = room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseWaiting && room.pokerState.Phase != game.PokerPhaseGameOver
			} else if room.gameType == "ludo" {
				gameInProgress = room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseWaiting && room.ludoState.Phase != game.LudoPhaseFinished
			} else if room.gameType == "nquestions" {
				gameInProgress = room.nqState != nil && room.nqState.Phase != game.NQPhaseWaiting && room.nqState.Phase != game.NQPhaseFinished
			} else if room.gameType == "commune" {
				gameInProgress = room.communeState != nil && room.communeState.Phase != game.CommunePhaseWaiting && room.communeState.Phase != game.CommunePhaseFinished
			} else if room.gameType == "twentynine" {
				gameInProgress = room.tnState != nil && room.tnState.Phase != game.TN_PhaseWaiting && room.tnState.Phase != game.TN_PhaseGameOver
			} else if room.gameType == "hearts" {
				gameInProgress = room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseWaiting && room.heartsState.Phase != game.HT_PhaseGameOver
			} else {
				gameInProgress = room.gameState != nil && room.gameState.Phase != game.PhaseWaiting && room.gameState.Phase != game.PhaseGameOver
			}

			if gameInProgress {
				// Give 5 minutes to reconnect before eliminating
				log.Printf("[DISCONNECT] room=%s player=%s — starting 300s reconnect grace period", room.code, pID[:8])
				timer := time.AfterFunc(300*time.Second, func() {
					room.mu.Lock()
					defer room.mu.Unlock()
					// Check again if they reconnected
					for _, pid := range room.connPlayer {
						if pid == pID {
							return // They reconnected, do nothing
						}
					}
					delete(room.disconnectTimers, pID)
					log.Printf("[DISCONNECT] room=%s player=%s — grace period expired, eliminating", room.code, pID[:8])
					if room.gameType == "poker" {
						if room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseGameOver {
							game.PokerVoluntaryExit(room.pokerState, pID)
							room.broadcastState()
						}
					} else if room.gameType == "ludo" {
						if room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseFinished {
							game.LudoVoluntaryExit(room.ludoState, pID)
							room.broadcastState()
						}
					} else if room.gameType == "nquestions" {
						if room.nqState != nil && room.nqState.Phase != game.NQPhaseFinished {
							game.NQVoluntaryExit(room.nqState, pID)
							room.broadcastState()
						}
					} else if room.gameType == "commune" {
						if room.communeState != nil && room.communeState.Phase != game.CommunePhaseFinished {
							game.CommuneVoluntaryExit(room.communeState, pID)
							room.broadcastState()
						}
					} else if room.gameType == "twentynine" {
						if room.tnState != nil && room.tnState.Phase != game.TN_PhaseGameOver {
							game.TNVoluntaryExit(room.tnState, pID)
							room.broadcastState()
						}
					} else if room.gameType == "hearts" {
						if room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseGameOver {
							game.HTVoluntaryExit(room.heartsState, pID)
							room.broadcastState()
						}
					} else {
						if room.gameState != nil && room.gameState.Phase != game.PhaseGameOver {
							for _, p := range room.gameState.Players {
								if p.ID == pID && p.IsAlive {
									game.EliminatePlayer(room.gameState, pID)
									room.broadcastState()
									break
								}
							}
						}
					}
					// Transfer host if needed
					if pID == room.hostID {
						room.hostID = ""
						for id := range room.connPlayer {
							pid := room.connPlayer[id]
							if _, ok := room.players[pid]; ok {
								room.hostID = pid
								break
							}
						}
						log.Printf("[DISCONNECT] room=%s host transferred to %s", room.code, room.hostID)
					}
					// Remove from players map
					delete(room.players, pID)
				})
				room.disconnectTimers[pID] = timer
			} else {
				// In lobby — remove immediately
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
					"gameActive": false,
				}})
			}
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

		gameInProgress := room.gameState != nil || room.pokerState != nil || room.ludoState != nil || room.nqState != nil || room.communeState != nil
		if gameInProgress {
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
			"gameActive": false,
			"gameType":   room.gameType,
		}})

	case "start-game":
		if playerID != room.hostID {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can start the game"}})
			return
		}
		// Prevent re-starting an already active game (allow if game is in terminal state)
		gameAlreadyActive := (room.gameState != nil && room.gameState.Phase != game.PhaseGameOver) ||
			(room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseGameOver) ||
			(room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseFinished) ||
			(room.nqState != nil && room.nqState.Phase != game.NQPhaseFinished) ||
			(room.communeState != nil && room.communeState.Phase != game.CommunePhaseFinished) ||
			(room.tnState != nil && room.tnState.Phase != game.TN_PhaseGameOver) ||
			(room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseGameOver)
		if gameAlreadyActive {
			return
		}
		// Clear any stale terminal-state games
		room.gameState = nil
		room.pokerState = nil
		room.ludoState = nil
		room.nqState = nil
		room.communeState = nil
		room.tnState = nil
		room.heartsState = nil
		if len(room.players) < 2 {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Need at least 2 players to start"}})
			return
		}

		playerList := make([]struct{ ID, Name string }, 0, len(room.players))
		for _, p := range room.players {
			playerList = append(playerList, struct{ ID, Name string }{p.ID, p.Name})
		}

		if room.gameType == "poker" {
			if room.pokerConfig == nil {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Poker config not set"}})
				return
			}
			room.pokerState = game.InitializePokerGame(playerList, room.pokerConfig.BuyIn, room.pokerConfig.SmallBlind, room.pokerConfig.BigBlindEnabled)
			logPokerHands(room)
			room.broadcast(OutMessage{Type: "poker-started", Payload: nil})
			room.broadcastPokerState()
		} else if room.gameType == "ludo" {
			if len(room.players) > 4 {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Ludo supports max 4 players"}})
				return
			}
			// Assign colors: use player-chosen colors or default assignment
			availableColors := []string{"red", "green", "blue", "yellow"}
			usedColors := make(map[string]bool)
			ludoPlayers := make([]struct{ ID, Name, Color string }, 0, len(room.players))

			// First pass: assign chosen colors
			for _, p := range room.players {
				if c, ok := room.ludoColors[p.ID]; ok && !usedColors[c] {
					usedColors[c] = true
					ludoPlayers = append(ludoPlayers, struct{ ID, Name, Color string }{p.ID, p.Name, c})
				}
			}
			// Second pass: assign remaining players default colors
			colorIdx := 0
			for _, p := range room.players {
				alreadyAssigned := false
				for _, lp := range ludoPlayers {
					if lp.ID == p.ID {
						alreadyAssigned = true
						break
					}
				}
				if !alreadyAssigned {
					for colorIdx < len(availableColors) && usedColors[availableColors[colorIdx]] {
						colorIdx++
					}
					if colorIdx < len(availableColors) {
						c := availableColors[colorIdx]
						usedColors[c] = true
						ludoPlayers = append(ludoPlayers, struct{ ID, Name, Color string }{p.ID, p.Name, c})
						colorIdx++
					}
				}
			}

			// For 2 players, ensure they sit opposite (color indices 0,2 or 1,3)
			if len(ludoPlayers) == 2 {
				c1 := ludoPlayers[0].Color
				c2 := ludoPlayers[1].Color
				idx1 := game.GetLudoColorIndex(c1)
				idx2 := game.GetLudoColorIndex(c2)
				diff := (idx2 - idx1 + 4) % 4
				if diff != 2 {
					// Reassign second player to opposite color
					oppositeIdx := (idx1 + 2) % 4
					oppositeColors := []string{"red", "green", "blue", "yellow"}
					ludoPlayers[1] = struct{ ID, Name, Color string }{ludoPlayers[1].ID, ludoPlayers[1].Name, oppositeColors[oppositeIdx]}
				}
			}

			room.ludoState = game.InitializeLudoGame(ludoPlayers)
			log.Printf("[LUDO] room=%s Ludo game started with %d players", room.code, len(ludoPlayers))
			room.broadcast(OutMessage{Type: "ludo-started", Payload: nil})
			room.broadcastLudoState()
		} else if room.gameType == "nquestions" {
			maxQ := 20
			if room.nqConfig != nil && room.nqConfig.MaxQuestions > 0 {
				maxQ = room.nqConfig.MaxQuestions
			}
			// Ensure host is first in list (becomes giver)
			orderedList := make([]struct{ ID, Name string }, 0, len(playerList))
			for _, p := range playerList {
				if p.ID == room.hostID {
					orderedList = append([]struct{ ID, Name string }{p}, orderedList...)
				} else {
					orderedList = append(orderedList, p)
				}
			}
			room.nqState = game.InitializeNQGame(orderedList, maxQ)
			log.Printf("[NQ] room=%s 20 QUESTIONS game started with %d players, maxQ=%d", room.code, len(orderedList), maxQ)
			room.broadcast(OutMessage{Type: "nq-started", Payload: nil})
			room.broadcastNQState()
		} else if room.gameType == "commune" {
			if len(room.players) > 10 {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Commune supports max 10 players"}})
				return
			}
			room.communeState = game.InitializeCommuneGame(playerList)
			log.Printf("[COMMUNE] room=%s Commune game started with %d players", room.code, len(playerList))
			room.broadcast(OutMessage{Type: "commune-started", Payload: nil})
			room.broadcastCommuneState()
		} else if room.gameType == "twentynine" {
			if len(room.players) != 4 {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "29 requires exactly 4 players"}})
				return
			}
			room.tnState = game.InitializeTwentyNineGame(playerList)
			log.Printf("[TN] room=%s 29 game started with 4 players", room.code)
			room.broadcast(OutMessage{Type: "tn-started", Payload: nil})
			room.broadcastTNState()
		} else if room.gameType == "hearts" {
			if len(room.players) != 4 {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Hearts requires exactly 4 players"}})
				return
			}
			room.heartsState = game.InitializeHeartsGame(playerList)
			log.Printf("[HT] room=%s Hearts game started with 4 players", room.code)
			room.broadcast(OutMessage{Type: "ht-started", Payload: nil})
			room.broadcastHTState()
		} else {
			room.gameState = game.InitializeGame(playerList, room.variant)
			log.Printf("[CARDS] room=%s Coup game started", room.code)
			for _, p := range room.gameState.Players {
				log.Printf("[CARDS] room=%s %s: %s, %s", room.code, p.Name, p.Cards[0].Character, p.Cards[1].Character)
			}
			room.broadcast(OutMessage{Type: "game-started", Payload: map[string]interface{}{"gameState": room.gameState}})
		}

	case "set-poker-config":
		var config PokerConfig
		json.Unmarshal(msg.Payload, &config)
		if config.BuyIn < 100 {
			config.BuyIn = 100
		}
		if config.SmallBlind < 1 {
			config.SmallBlind = 1
		}
		room.pokerConfig = &config
		room.broadcast(OutMessage{Type: "poker-config", Payload: config})

	case "set-nq-config":
		var config NQConfig
		json.Unmarshal(msg.Payload, &config)
		if config.MaxQuestions < 5 {
			config.MaxQuestions = 5
		}
		if config.MaxQuestions > 50 {
			config.MaxQuestions = 50
		}
		room.nqConfig = &config

	case "return-to-lobby":
		if room.gameType == "poker" {
			if room.pokerState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if playerID != room.hostID && room.pokerState.Phase != game.PokerPhaseGameOver {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can return to lobby"}})
				return
			}
			room.pokerState = nil
		} else if room.gameType == "ludo" {
			if room.ludoState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if playerID != room.hostID && room.ludoState.Phase != game.LudoPhaseFinished {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can return to lobby"}})
				return
			}
			room.ludoState = nil
			room.ludoColors = make(map[string]string)
		} else if room.gameType == "nquestions" {
			if room.nqState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if playerID != room.hostID && room.nqState.Phase != game.NQPhaseFinished {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Only the host can return to lobby"}})
				return
			}
			room.nqState = nil
			room.nqConfig = nil
		} else if room.gameType == "commune" {
			if room.communeState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if room.communeState.Phase != game.CommunePhaseFinished {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Game is still in progress"}})
				return
			}
			room.communeState = nil
		} else if room.gameType == "twentynine" {
			if room.tnState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if room.tnState.Phase != game.TN_PhaseGameOver {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Game is still in progress"}})
				return
			}
			room.tnState = nil
		} else if room.gameType == "hearts" {
			if room.heartsState == nil {
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
				}})
				return
			}
			if room.heartsState.Phase != game.HT_PhaseGameOver {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Game is still in progress"}})
				return
			}
			room.heartsState = nil
		} else {
			hostShort := "(none)"
			if len(room.hostID) >= 8 { hostShort = room.hostID[:8] }
			log.Printf("[RTL] player=%s host=%s gameState=%v phase=%v", playerID[:8], hostShort, room.gameState != nil, func() string { if room.gameState != nil { return string(room.gameState.Phase) }; return "nil" }())
			if room.gameState == nil {
				log.Printf("[RTL] gameState is nil, sending lobby state to player")
				room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
				room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
					"players":    room.playerList(),
					"hostId":     room.hostID,
					"gameActive": false,
					"gameType":   room.gameType,
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
		}

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
			"gameType":   room.gameType,
		}})

	case "exit-game":
		if room.gameType == "poker" {
			if room.pokerState == nil || room.pokerState.Phase == game.PokerPhaseGameOver {
				return
			}
			game.PokerVoluntaryExit(room.pokerState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.pokerState != nil && room.pokerState.Phase != game.PokerPhaseGameOver,
				"gameType":   room.gameType,
			}})
		} else if room.gameType == "ludo" {
			if room.ludoState == nil || room.ludoState.Phase == game.LudoPhaseFinished {
				return
			}
			game.LudoVoluntaryExit(room.ludoState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.ludoState != nil && room.ludoState.Phase != game.LudoPhaseFinished,
				"gameType":   room.gameType,
			}})
		} else if room.gameType == "nquestions" {
			if room.nqState == nil || room.nqState.Phase == game.NQPhaseFinished {
				return
			}
			game.NQVoluntaryExit(room.nqState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.nqState != nil && room.nqState.Phase != game.NQPhaseFinished,
				"gameType":   room.gameType,
			}})
		} else if room.gameType == "commune" {
			if room.communeState == nil || room.communeState.Phase == game.CommunePhaseFinished {
				return
			}
			game.CommuneVoluntaryExit(room.communeState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.communeState != nil && room.communeState.Phase != game.CommunePhaseFinished,
				"gameType":   room.gameType,
			}})
		} else if room.gameType == "twentynine" {
			if room.tnState == nil || room.tnState.Phase == game.TN_PhaseGameOver {
				return
			}
			game.TNVoluntaryExit(room.tnState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.tnState != nil && room.tnState.Phase != game.TN_PhaseGameOver,
				"gameType":   room.gameType,
			}})
		} else if room.gameType == "hearts" {
			if room.heartsState == nil || room.heartsState.Phase == game.HT_PhaseGameOver {
				return
			}
			game.HTVoluntaryExit(room.heartsState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.heartsState != nil && room.heartsState.Phase != game.HT_PhaseGameOver,
				"gameType":   room.gameType,
			}})
		} else {
			if room.gameState == nil || room.gameState.Phase == game.PhaseGameOver {
				return
			}
			game.VoluntaryExit(room.gameState, playerID)
			room.broadcastState()
			room.sendTo(connID, OutMessage{Type: "state", Payload: nil})
			room.sendTo(connID, OutMessage{Type: "players-updated", Payload: map[string]interface{}{
				"players":    room.playerList(),
				"hostId":     room.hostID,
				"gameActive": room.gameState != nil && room.gameState.Phase != game.PhaseGameOver,
				"gameType":   room.gameType,
			}})
		}

	case "spectate":
		if room.gameType == "poker" && room.pokerState != nil {
			personalized := room.createPersonalizedPokerState("")
			room.sendTo(connID, OutMessage{Type: "poker-spectate", Payload: personalized})
		} else if room.gameType == "ludo" && room.ludoState != nil {
			room.sendTo(connID, OutMessage{Type: "ludo-spectate", Payload: room.ludoState})
		} else if room.gameType == "nquestions" && room.nqState != nil {
			sanitized := game.SanitizeNQStateForGuessers(room.nqState)
			room.sendTo(connID, OutMessage{Type: "nq-spectate", Payload: sanitized})
		} else if room.gameType == "commune" && room.communeState != nil {
			spectatorState := game.SanitizeCommuneStateForPlayer(room.communeState, "")
			room.sendTo(connID, OutMessage{Type: "commune-spectate", Payload: spectatorState})
		} else if room.gameState != nil {
			room.sendTo(connID, OutMessage{Type: "spectate-state", Payload: room.gameState})
		}

	case "kick-player":
		if playerID != room.hostID {
			return
		}
		gameInProgress := false
		if room.gameType == "poker" {
			gameInProgress = room.pokerState != nil
		} else if room.gameType == "ludo" {
			gameInProgress = room.ludoState != nil
		} else if room.gameType == "nquestions" {
			gameInProgress = room.nqState != nil
		} else if room.gameType == "commune" {
			gameInProgress = room.communeState != nil
		} else {
			gameInProgress = room.gameState != nil && room.gameState.Phase != game.PhaseWaiting
		}
		if gameInProgress {
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
			"gameActive": false,
			"gameType":   room.gameType,
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
		if room.gameType == "poker" && room.pokerState != nil {
			personalized := room.createPersonalizedPokerState(playerID)
			room.sendTo(connID, OutMessage{Type: "poker-state", Payload: personalized})
		} else if room.gameType == "ludo" && room.ludoState != nil {
			room.sendTo(connID, OutMessage{Type: "ludo-state", Payload: room.ludoState})
		} else if room.gameType == "nquestions" && room.nqState != nil {
			isGiver := false
			for _, p := range room.nqState.Players {
				if p.ID == playerID && p.IsGiver {
					isGiver = true
					break
				}
			}
			if isGiver {
				room.sendTo(connID, OutMessage{Type: "nq-state", Payload: room.nqState})
			} else {
				sanitized := game.SanitizeNQStateForGuessers(room.nqState)
				room.sendTo(connID, OutMessage{Type: "nq-state", Payload: sanitized})
			}
		} else if room.gameType == "commune" && room.communeState != nil {
			personalized := game.SanitizeCommuneStateForPlayer(room.communeState, playerID)
			room.sendTo(connID, OutMessage{Type: "commune-state", Payload: personalized})
		} else if room.gameState != nil {
			room.sendTo(connID, OutMessage{Type: "state", Payload: room.gameState})
		}

	// Poker actions
	case "poker-action":
		if room.pokerState == nil {
			return
		}
		var payload struct {
			Action string `json:"action"`
			Amount int    `json:"amount"`
		}
		json.Unmarshal(msg.Payload, &payload)

		if err := game.PokerAction(room.pokerState, playerID, payload.Action, payload.Amount); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastPokerState()

	case "poker-next-hand":
		if room.pokerState == nil || room.pokerState.Phase != game.PokerPhaseShowdown {
			return
		}
		if room.pokerState.PendingRebuys {
			return // Can't start next hand while waiting for rebuy decisions
		}
		game.StartNextHand(room.pokerState)
		logPokerHands(room)
		room.broadcastPokerState()

	case "poker-rebuy":
		if room.pokerState == nil {
			return
		}
		if err := game.PokerRebuy(room.pokerState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		logPokerHands(room)
		room.broadcastPokerState()

	case "poker-skip-rebuy":
		if room.pokerState == nil {
			return
		}
		if err := game.PokerSkipRebuy(room.pokerState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		logPokerHands(room)
		room.broadcastPokerState()

	case "poker-end-game":
		if room.pokerState == nil || (room.pokerState.Phase != game.PokerPhaseShowdown) {
			return
		}
		game.EndGameEarly(room.pokerState)
		room.broadcastPokerState()

	case "poker-join-game":
		if room.pokerState == nil || room.pokerState.Phase == game.PokerPhaseGameOver {
			return
		}
		// Only spectators (not already a player) can join
		for _, p := range room.pokerState.Players {
			if p.ID == playerID {
				return // Already a player
			}
		}
		name := ""
		if pc, ok := room.players[playerID]; ok {
			name = pc.Name
		}
		if name == "" {
			return
		}
		game.AddPendingPokerPlayer(room.pokerState, playerID, name)
		room.broadcastPokerState()

	// Ludo actions
	case "set-ludo-color":
		var payload struct {
			Color string `json:"color"`
		}
		json.Unmarshal(msg.Payload, &payload)
		validColors := map[string]bool{"red": true, "green": true, "blue": true, "yellow": true}
		if !validColors[payload.Color] {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Invalid color"}})
			return
		}
		// Check if color is taken by someone else
		for pid, c := range room.ludoColors {
			if c == payload.Color && pid != playerID {
				room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Color already taken"}})
				return
			}
		}
		room.ludoColors[playerID] = payload.Color
		room.broadcast(OutMessage{Type: "ludo-colors", Payload: room.ludoColors})

	case "ludo-roll":
		if room.ludoState == nil {
			return
		}
		if err := game.LudoRollDice(room.ludoState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastLudoState()

	case "ludo-move":
		if room.ludoState == nil {
			return
		}
		var payload struct {
			TokenID int `json:"tokenId"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.LudoMoveToken(room.ludoState, playerID, payload.TokenID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastLudoState()

	// 20 QUESTIONS actions
	case "nq-set-word":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Category string `json:"category"`
			Word     string `json:"word"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQSetWord(room.nqState, playerID, payload.Category, payload.Word); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-ask":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Question  string `json:"question"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQAskQuestion(room.nqState, playerID, payload.Question); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-guess":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Guess string `json:"guess"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQMakeGuess(room.nqState, playerID, payload.Guess); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-verify-guess":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Correct bool `json:"correct"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQVerifyGuess(room.nqState, playerID, payload.Correct); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-answer":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Answer string `json:"answer"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQAnswerQuestion(room.nqState, playerID, payload.Answer); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-final-guess":
		if room.nqState == nil {
			return
		}
		var payload struct {
			Guess string `json:"guess"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.NQMakeGuess(room.nqState, playerID, payload.Guess); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastNQState()

	case "nq-next-round":
		if room.nqState == nil || room.nqState.Phase != game.NQPhaseFinished {
			return
		}
		if len(room.nqState.Players) < 2 {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": "Need at least 2 players"}})
			return
		}
		game.NQNextRound(room.nqState)
		room.broadcastNQState()

	// Commune actions
	case "commune-declare":
		if room.communeState == nil {
			return
		}
		var decl game.CommuneDeclaration
		json.Unmarshal(msg.Payload, &decl)
		if err := game.CommuneDeclare(room.communeState, playerID, decl); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastCommuneState()

	case "commune-call":
		if room.communeState == nil {
			return
		}
		if err := game.CommuneCall(room.communeState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastCommuneState()

	case "commune-next-hand":
		if room.communeState == nil || room.communeState.Phase != game.CommunePhaseCalled {
			return
		}
		game.CommuneNextHand(room.communeState)
		room.broadcastCommuneState()

	// 29 Card Game actions
	case "tn-bid":
		if room.tnState == nil {
			return
		}
		var payload struct {
			Value int `json:"value"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.TNBid(room.tnState, playerID, payload.Value); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-pass":
		if room.tnState == nil {
			return
		}
		if err := game.TNPass(room.tnState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-select-trump":
		if room.tnState == nil {
			return
		}
		var payload struct {
			Suit string `json:"suit"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.TNSelectTrump(room.tnState, playerID, payload.Suit); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-play-card":
		if room.tnState == nil {
			return
		}
		var payload struct {
			CardID string `json:"cardId"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.TNPlayCard(room.tnState, playerID, payload.CardID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-reveal-trump":
		if room.tnState == nil {
			return
		}
		if err := game.TNRevealTrump(room.tnState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-declare-pair":
		if room.tnState == nil {
			return
		}
		if err := game.TNDeclarePair(room.tnState, playerID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastTNState()

	case "tn-next-round":
		if room.tnState == nil || room.tnState.Phase != game.TN_PhaseRoundOver {
			return
		}
		game.TNNextRound(room.tnState)
		room.broadcastTNState()

	// Hearts Card Game actions
	case "ht-pass-cards":
		if room.heartsState == nil {
			return
		}
		var payload struct {
			CardIDs []string `json:"cardIds"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.HTPassCards(room.heartsState, playerID, payload.CardIDs); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastHTState()

	case "ht-play-card":
		if room.heartsState == nil {
			return
		}
		var payload struct {
			CardID string `json:"cardId"`
		}
		json.Unmarshal(msg.Payload, &payload)
		if err := game.HTPlayCard(room.heartsState, playerID, payload.CardID); err != nil {
			room.sendTo(connID, OutMessage{Type: "error", Payload: map[string]string{"message": err.Error()}})
			return
		}
		room.broadcastHTState()

	case "ht-next-hand":
		if room.heartsState == nil || room.heartsState.Phase != game.HT_PhaseHandOver {
			return
		}
		game.HTNextHand(room.heartsState)
		room.broadcastHTState()

	case "chat":
		var payload struct {
			Message string `json:"message"`
		}
		json.Unmarshal(msg.Payload, &payload)
		text := strings.TrimSpace(payload.Message)
		if text == "" || len(text) > 500 {
			return
		}
		playerName := ""
		if p, ok := room.players[playerID]; ok {
			playerName = p.Name
		}
		if playerName == "" {
			return
		}
		room.broadcast(OutMessage{Type: "chat", Payload: map[string]interface{}{
			"senderId":   playerID,
			"senderName": playerName,
			"message":    text,
			"timestamp":  time.Now().UnixMilli(),
		}})

	case "ping":
		room.sendTo(connID, OutMessage{Type: "pong"})

	// ---- Voice chat (WebSocket relay) ----
	case "voice-join":
		for cID, pID := range room.connPlayer {
			if pID != playerID {
				room.sendTo(cID, OutMessage{Type: "voice-join", Payload: map[string]interface{}{
					"peerId": playerID,
				}})
			}
		}

	case "voice-leave":
		for cID, pID := range room.connPlayer {
			if pID != playerID {
				room.sendTo(cID, OutMessage{Type: "voice-leave", Payload: map[string]interface{}{
					"peerId": playerID,
				}})
			}
		}

	case "voice-data":
		// Relay audio data to all other players
		var vPayload struct {
			Audio string `json:"audio"`
			Sr    int    `json:"sr"`
		}
		json.Unmarshal(msg.Payload, &vPayload)
		for cID, pID := range room.connPlayer {
			if pID != playerID {
				room.sendTo(cID, OutMessage{Type: "voice-data", Payload: map[string]interface{}{
					"peerId": playerID,
					"audio":  vPayload.Audio,
					"sr":     vPayload.Sr,
				}})
			}
		}
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

	// Serve static files (textures, icons, css, js, etc)
	http.Handle("/textures/", http.StripPrefix("/textures/", http.FileServer(http.Dir("public/textures"))))
	http.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir("public/icons"))))
	noCacheFS := func(dir, prefix string) http.Handler {
		fs := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			fs.ServeHTTP(w, r)
		})
	}
	http.Handle("/css/", noCacheFS("public/css", "/css/"))
	http.Handle("/js/", noCacheFS("public/js", "/js/"))
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
	log.Printf("Game server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
