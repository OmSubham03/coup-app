package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"coup-server/game"
	"github.com/gorilla/websocket"
)

type additionalWSTest struct {
	t           *testing.T
	server      *httptest.Server
	code        string
	gameType    string
	connections []*websocket.Conn
}

func newAdditionalWSTest(t *testing.T, gameType string) *additionalWSTest {
	t.Helper()
	harness := &additionalWSTest{t: t, server: httptest.NewServer(httpHandlerForRooms()), code: "NEW-" + generateCode(), gameType: gameType}
	t.Cleanup(func() {
		for _, connection := range harness.connections {
			connection.Close()
		}
		if room := getRoom(harness.code); room != nil {
			harness.await(func() bool { return len(room.connections) == 0 })
			room.mu.Lock()
			for _, timer := range room.disconnectTimers {
				timer.Stop()
			}
			room.mu.Unlock()
		}
		harness.server.Close()
		roomsMu.Lock()
		delete(rooms, harness.code)
		roomsMu.Unlock()
	})
	return harness
}

func (harness *additionalWSTest) connect(id, action, decks string) *websocket.Conn {
	harness.t.Helper()
	query := url.Values{"room": {harness.code}, "gameType": {harness.gameType}, "playerId": {id}, "action": {action}}
	if decks != "" {
		query.Set("twoDecks", decks)
	}
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(harness.server.URL, "http")+"/ws?"+query.Encode(), nil)
	if err != nil {
		harness.t.Fatal(err)
	}
	harness.connections = append(harness.connections, connection)
	return connection
}

func (harness *additionalWSTest) send(connection *websocket.Conn, kind string, payload interface{}) {
	harness.t.Helper()
	if err := connection.WriteJSON(OutMessage{Type: kind, Payload: payload}); err != nil {
		harness.t.Fatal(err)
	}
}

func (harness *additionalWSTest) read(connection *websocket.Conn, kind string) json.RawMessage {
	harness.t.Helper()
	connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var message struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := connection.ReadJSON(&message); err != nil {
			harness.t.Fatalf("waiting for %s: %v", kind, err)
		}
		if message.Type == kind {
			return message.Payload
		}
	}
}

func (harness *additionalWSTest) sync(connection *websocket.Conn) {
	harness.send(connection, "ping", nil)
	harness.read(connection, "pong")
}

func (harness *additionalWSTest) join(connection *websocket.Conn, name string, public bool) {
	harness.send(connection, "join", map[string]interface{}{"playerName": name, "publicRoom": public})
	harness.read(connection, "players-updated")
}

func (harness *additionalWSTest) await(predicate func() bool) {
	harness.t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		room := getRoom(harness.code)
		room.mu.Lock()
		ready := predicate()
		room.mu.Unlock()
		if ready {
			return
		}
		select {
		case <-deadline.C:
			harness.t.Fatal("room lifecycle transition timed out")
		case <-ticker.C:
		}
	}
}

func decodeAdditional[T any](t *testing.T, payload json.RawMessage) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAdditionalWebSocketMinimumAndDecks(t *testing.T) {
	for _, gameType := range []string{"bluff", "blackjack"} {
		for _, decks := range []string{"", "true"} {
			t.Run(fmt.Sprintf("%s/twoDecks=%s", gameType, decks), func(t *testing.T) {
				harness := newAdditionalWSTest(t, gameType)
				host := harness.connect("minimum-host-id", "create", decks)
				visibility := decodeAdditional[map[string]interface{}](t, harness.read(host, "room-visibility"))
				if visibility["twoDecks"] != (decks == "true") || visibility["public"] != false {
					t.Fatal("incorrect creation defaults")
				}
				harness.read(host, "waiting")
				harness.join(host, "Host", false)
				if gameType == "bluff" {
					harness.send(host, "start-game", nil)
					harness.read(host, "error")
					guest := harness.connect("minimum-guest-id", "", "")
					harness.read(guest, "waiting")
					harness.join(guest, "Guest", false)
				}
				harness.send(host, "start-game", nil)
				payload := harness.read(host, gameType+"-state")
				if gameType == "bluff" {
					state := decodeAdditional[game.BluffState](t, payload)
					count := 0
					for _, player := range state.Players {
						count += len(player.Cards)
					}
					want := 52
					if decks == "true" {
						want = 104
					}
					if len(state.Players) != 2 || count != want || state.TwoDecks != (decks == "true") {
						t.Fatal("incorrect Bluff minimum/deal")
					}
				} else {
					state := decodeAdditional[game.BlackjackState](t, payload)
					if len(state.Players) != 1 || state.Round != 1 || state.TwoDecks != (decks == "true") {
						t.Fatal("incorrect Blackjack solo start")
					}
					if state.Phase == game.BlackjackPhasePlaying && state.DealerCards[1].ID != "" {
						t.Fatal("dealer hole card leaked")
					}
					fields := decodeAdditional[map[string]json.RawMessage](t, payload)
					if fields["shoe"] != nil {
						t.Fatal("shoe leaked")
					}
				}
				room := getRoom(harness.code)
				room.mu.Lock()
				if room.gameState != nil || !room.hasStartedGame() {
					t.Fatal("new game fell through to Coup")
				}
				room.mu.Unlock()
			})
		}
	}
}

func TestAdditionalWebSocketCapacityDiscoveryAndHostSettings(t *testing.T) {
	for _, gameType := range []string{"bluff", "blackjack"} {
		t.Run(gameType, func(t *testing.T) {
			harness := newAdditionalWSTest(t, gameType)
			host := harness.connect("capacity-host-id", "create", "true")
			harness.read(host, "waiting")
			harness.join(host, "Host", false)
			harness.send(host, "set-room-visibility", map[string]bool{"public": true})
			harness.read(host, "room-visibility")
			listed := listPublicRooms(gameType)
			found := false
			for _, room := range listed {
				if room.Code == harness.code && room.Capacity == roomCapacity(gameType) {
					found = true
				}
			}
			if !found {
				t.Fatal("new game missing from discovery")
			}
			guest := harness.connect("capacity-guest-id", "create", "false")
			visibility := decodeAdditional[map[string]interface{}](t, harness.read(guest, "room-visibility"))
			if visibility["twoDecks"] != true {
				t.Fatal("second create overwrote deck setting")
			}
			harness.read(guest, "waiting")
			harness.join(guest, "Guest", true)
			harness.send(guest, "set-room-visibility", map[string]bool{"public": false})
			harness.read(guest, "error")
			harness.send(guest, "start-game", nil)
			harness.read(guest, "error")
			for index := 2; index < roomCapacity(gameType); index++ {
				connection := harness.connect(fmt.Sprintf("capacity-player-%d", index), "", "")
				harness.read(connection, "waiting")
				harness.join(connection, fmt.Sprintf("Player%d", index), false)
			}
			for _, room := range listPublicRooms(gameType) {
				if room.Code == harness.code {
					t.Fatal("full room still discoverable")
				}
			}
			extra := harness.connect("capacity-extra-id", "", "")
			harness.read(extra, "waiting")
			for _, public := range []bool{true, false} {
				harness.send(extra, "join", map[string]interface{}{"playerName": "Extra", "publicRoom": public})
				harness.read(extra, "error")
			}
			harness.sync(host)
			harness.send(host, "start-game", nil)
			payload := decodeAdditional[map[string]interface{}](t, harness.read(host, gameType+"-state"))
			if len(payload["players"].([]interface{})) != roomCapacity(gameType) {
				t.Fatal("wrong maximum start size")
			}
			harness.send(host, "set-room-visibility", map[string]bool{"public": false})
			harness.read(host, "error")
			harness.sync(extra)
			harness.send(extra, "join", map[string]interface{}{"playerName": "Extra", "publicRoom": true})
			harness.read(extra, "error")
		})
	}
}

func TestAdditionalWebSocketPrivateViewsActionsAndLifecycle(t *testing.T) {
	for _, gameType := range []string{"bluff", "blackjack"} {
		t.Run(gameType, func(t *testing.T) {
			harness := newAdditionalWSTest(t, gameType)
			hostID, guestID := "a-lifecycle-host", "b-lifecycle-guest"
			host := harness.connect(hostID, "create", "true")
			harness.read(host, "waiting")
			harness.join(host, "Host", false)
			guest := harness.connect(guestID, "", "")
			harness.read(guest, "waiting")
			harness.join(guest, "Guest", false)
			harness.sync(host)
			harness.send(host, "start-game", nil)
			hostPayload := harness.read(host, gameType+"-state")
			guestPayload := harness.read(guest, gameType+"-state")
			spectator := harness.connect("lifecycle-spectator", "", "false")
			spectatorPayload := harness.read(spectator, gameType+"-state")
			room := getRoom(harness.code)
			if gameType == "bluff" {
				hostView := decodeAdditional[game.BluffState](t, hostPayload)
				guestView := decodeAdditional[game.BluffState](t, guestPayload)
				spectatorView := decodeAdditional[game.BluffState](t, spectatorPayload)
				if hostView.Players[0].Cards[0].ID == "" || hostView.Players[1].Cards[0].ID != "" || guestView.Players[1].Cards[0].ID == "" || guestView.Players[0].Cards[0].ID != "" {
					t.Fatal("incorrect private hands")
				}
				for _, player := range spectatorView.Players {
					for _, card := range player.Cards {
						if card.ID != "" {
							t.Fatal("spectator hand leak")
						}
					}
				}
				harness.send(host, "bluff-play", map[string]interface{}{"cardIds": []string{hostView.Players[0].Cards[0].ID}})
				played := decodeAdditional[game.BluffState](t, harness.read(host, "bluff-state"))
				if played.Phase != game.BluffPhaseChallenge || played.Pile[0].ID != "" {
					t.Fatal("play/pile sanitization failed")
				}
				harness.sync(guest)
				harness.send(guest, "bluff-accept", nil)
				accepted := decodeAdditional[game.BluffState](t, harness.read(guest, "bluff-state"))
				if accepted.Phase != game.BluffPhasePlaying {
					t.Fatal("accept not dispatched")
				}
				harness.send(guest, "bluff-play", map[string]interface{}{"cardIds": []string{guestView.Players[1].Cards[0].ID}})
				harness.read(guest, "bluff-state")
				harness.sync(host)
				harness.send(host, "bluff-challenge", nil)
				challenged := decodeAdditional[game.BluffState](t, harness.read(host, "bluff-state"))
				if challenged.Phase != game.BluffPhasePlaying || len(challenged.Pile) != 0 {
					t.Fatal("challenge not dispatched")
				}
			} else {
				state := decodeAdditional[game.BlackjackState](t, hostPayload)
				if state.Phase == game.BlackjackPhasePlaying && state.DealerCards[1].ID != "" {
					t.Fatal("dealer leak")
				}
				for _, payload := range []json.RawMessage{guestPayload, spectatorPayload} {
					view := decodeAdditional[game.BlackjackState](t, payload)
					if view.Phase == game.BlackjackPhasePlaying && view.DealerCards[1].ID != "" {
						t.Fatal("dealer hole card leaked to guest or spectator")
					}
					if view.Players[0].Cards[0].ID == "" || view.Players[1].Cards[0].ID == "" {
						t.Fatal("public Blackjack player hands were hidden")
					}
				}
				room.mu.Lock()
				room.blackjackState.Phase = game.BlackjackPhasePlaying
				room.blackjackState.CurrentPlayerIdx = 0
				room.blackjackState.DealerCards = []game.BlackjackCard{{ID: "dealer-visible", Rank: 10}, {ID: "dealer-hidden", Rank: 7}}
				for index := range room.blackjackState.Players {
					player := &room.blackjackState.Players[index]
					player.Status = game.BlackjackStatusActive
					player.Cards = []game.BlackjackCard{{ID: "fixture-one", Rank: 2}, {ID: "fixture-two", Rank: 2}}
				}
				room.mu.Unlock()
				harness.sync(host)
				harness.send(host, "blackjack-hit", nil)
				hit := decodeAdditional[game.BlackjackState](t, harness.read(host, "blackjack-state"))
				if len(hit.Players[0].Cards) != 3 || hit.DealerCards[1].ID != "" {
					t.Fatal("hit or hidden dealer failed")
				}
				harness.send(host, "blackjack-stand", nil)
				harness.read(host, "blackjack-state")
				harness.sync(guest)
				harness.send(guest, "blackjack-double", nil)
				doubled := decodeAdditional[game.BlackjackState](t, harness.read(guest, "blackjack-state"))
				if doubled.Players[1].Bet != 100 || doubled.Phase != game.BlackjackPhaseRoundOver || doubled.DealerCards[1].ID == "" {
					t.Fatal("double/settlement not dispatched")
				}
				harness.send(guest, "blackjack-next-round", nil)
				harness.read(guest, "error")
				harness.sync(host)
				harness.send(host, "blackjack-next-round", nil)
				next := decodeAdditional[game.BlackjackState](t, harness.read(host, "blackjack-state"))
				if next.Round != 2 || !next.TwoDecks {
					t.Fatal("host next round lost settings")
				}
			}
			harness.sync(spectator)
			harness.send(spectator, "spectate", nil)
			harness.read(spectator, gameType+"-state")
			harness.send(spectator, gameType+map[string]string{"bluff": "-accept", "blackjack": "-hit"}[gameType], nil)
			harness.read(spectator, "error")
			harness.send(host, "return-to-lobby", nil)
			harness.read(host, "error")
			if findPlayerActiveRoom(hostID) != room {
				t.Fatal("active game not found for redirect")
			}
			otherRoom := newAdditionalWSTest(t, gameType)
			redirected := otherRoom.connect(hostID, "create", "")
			redirect := decodeAdditional[map[string]string](t, otherRoom.read(redirected, "redirect"))
			if redirect["roomCode"] != harness.code || redirect["gameType"] != gameType {
				t.Fatal("incorrect active-room redirect")
			}
			harness.sync(guest)
			harness.send(guest, "exit-game", nil)
			harness.read(guest, "state")
			if findPlayerActiveRoom(guestID) != nil {
				t.Fatal("exited player remains in active game")
			}
			harness.send(guest, gameType+map[string]string{"bluff": "-accept", "blackjack": "-hit"}[gameType], nil)
			harness.read(guest, "error")
			if gameType == "blackjack" {
				harness.send(host, "exit-game", nil)
				harness.read(host, "state")
			} else {
				finishedReconnect := harness.connect(hostID, "", "false")
				finished := decodeAdditional[game.BluffState](t, harness.read(finishedReconnect, "bluff-state"))
				if finished.Phase != game.BluffPhaseGameOver || finished.Players[0].Cards[0].ID == "" {
					t.Fatal("finished reconnect lost the participant's private view")
				}
			}
			harness.sync(host)
			harness.send(host, "return-to-lobby", nil)
			harness.read(host, "players-updated")
			room.mu.Lock()
			started := room.hasStartedGame()
			decks := room.twoDecks
			room.mu.Unlock()
			if started || !decks {
				t.Fatal("return-to-lobby retained game or lost deck settings")
			}
			harness.send(guest, "leave-room", nil)
			harness.read(guest, "room-left")
			harness.await(func() bool { return room.players[guestID] == nil })
		})
	}
}

func TestAdditionalWebSocketReconnectAndDisconnectForfeit(t *testing.T) {
	for _, gameType := range []string{"bluff", "blackjack"} {
		t.Run(gameType, func(t *testing.T) {
			harness := newAdditionalWSTest(t, gameType)
			hostID, guestID := "a-disconnect-host", "b-disconnect-guest"
			host := harness.connect(hostID, "create", "")
			harness.read(host, "waiting")
			harness.join(host, "Host", false)
			guest := harness.connect(guestID, "", "")
			harness.read(guest, "waiting")
			harness.join(guest, "Guest", false)
			harness.send(host, "start-game", nil)
			harness.read(host, gameType+"-state")
			harness.read(guest, gameType+"-state")
			room := getRoom(harness.code)
			host.Close()
			harness.await(func() bool { return room.disconnectTimers[hostID] != nil })
			reconnected := harness.connect(hostID, "", "true")
			view := harness.read(reconnected, gameType+"-state")
			if gameType == "bluff" && decodeAdditional[game.BluffState](t, view).Players[0].Cards[0].ID == "" {
				t.Fatal("reconnect lost private hand")
			}
			harness.await(func() bool { return room.disconnectTimers[hostID] == nil })
			harness.send(reconnected, "join", map[string]string{"playerName": "Host"})
			harness.read(reconnected, gameType+"-state")
			harness.sync(guest)
			reconnected.Close()
			harness.await(func() bool { return room.disconnectTimers[hostID] != nil })
			room.mu.Lock()
			room.disconnectTimers[hostID].Reset(0)
			room.mu.Unlock()
			harness.read(guest, gameType+"-state")
			harness.await(func() bool {
				return room.players[hostID] == nil && room.hostID == guestID && room.disconnectTimers[hostID] == nil
			})
			room.mu.Lock()
			active := room.additionalPlayerActive(hostID)
			room.mu.Unlock()
			if active {
				t.Fatal("disconnect did not forfeit engine player")
			}
			harness.send(guest, "leave-room", nil)
			harness.read(guest, "room-left")
			harness.await(func() bool { return len(room.players) == 0 })
		})
	}
}

func TestAdditionalWebSocketMissingAndStaleRooms(t *testing.T) {
	for _, gameType := range []string{"bluff", "blackjack"} {
		t.Run(gameType, func(t *testing.T) {
			harness := newAdditionalWSTest(t, gameType)
			missing := harness.connect("missing-player-id", "", "")
			harness.read(missing, "error")
			host := harness.connect("stale-host-player", "create", "")
			harness.read(host, "waiting")
			harness.join(host, "Host", false)
			guest := harness.connect("stale-guest-player", "", "true")
			harness.read(guest, "waiting")
			harness.send(guest, "join", map[string]interface{}{"playerName": "Guest", "publicRoom": true})
			harness.read(guest, "error")
			harness.join(guest, "Guest", false)
			harness.sync(host)
			harness.send(host, "set-room-visibility", map[string]bool{"public": true})
			harness.read(host, "room-visibility")
			late := harness.connect("stale-late-player", "", "")
			harness.read(late, "waiting")
			harness.send(host, "set-room-visibility", map[string]bool{"public": false})
			harness.read(host, "room-visibility")
			harness.send(late, "join", map[string]interface{}{"playerName": "Late", "publicRoom": true})
			harness.read(late, "error")
			room := getRoom(harness.code)
			room.mu.Lock()
			if room.twoDecks || room.players["stale-late-player"] != nil {
				t.Fatal("join changed settings or admitted stale discovery")
			}
			room.mu.Unlock()
			harness.sync(guest)
			harness.send(host, "start-game", nil)
			harness.read(host, gameType+"-state")
			harness.read(guest, gameType+"-state")
			harness.send(guest, "leave-room", nil)
			harness.read(guest, "room-left")
			harness.await(func() bool {
				return room.players["stale-guest-player"] == nil && !room.additionalPlayerActive("stale-guest-player")
			})
		})
	}
}
