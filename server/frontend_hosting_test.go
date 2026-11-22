package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestFrontendCrossOriginPublicAPIs(t *testing.T) {
	for _, api := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/generate-code", handleGenerateCode},
		{"/api/variant-config?variant=standard", handleVariantConfig},
		{"/api/public-rooms?gameType=tictactoe", handlePublicRooms},
		{"/api/room-stats", handleRoomStats},
	} {
		t.Run(api.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, api.path, nil)
			request.Header.Set("Origin", "https://timecrushers.pages.dev")
			response := httptest.NewRecorder()
			api.handler(response, request)
			if response.Code != http.StatusOK ||
				response.Header().Get("Access-Control-Allow-Origin") != "*" ||
				response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("cross-origin API response: status=%d headers=%v", response.Code, response.Header())
			}
			if response.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("public APIs must not enable credentialed cross-origin reads")
			}
		})
	}
	for _, handler := range []http.HandlerFunc{handlePublicRooms, handleRoomStats} {
		response := httptest.NewRecorder()
		handler(response, httptest.NewRequest(http.MethodPost, "/", nil))
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("cross-origin method error: status=%d headers=%v", response.Code, response.Header())
		}
	}
}

func TestFrontendCrossOriginWebSocketMatchChatAndReconnect(t *testing.T) {
	h := newAdditionalWSTest(t, "tictactoe")
	connect := func(id, action string) *websocket.Conn {
		t.Helper()
		query := url.Values{"room": {h.code}, "gameType": {h.gameType}, "playerId": {id}, "action": {action}}
		connection, _, err := websocket.DefaultDialer.Dial(
			"ws"+strings.TrimPrefix(h.server.URL, "http")+"/ws?"+query.Encode(),
			http.Header{"Origin": {"https://timecrushers.pages.dev"}},
		)
		if err != nil {
			t.Fatal(err)
		}
		h.connections = append(h.connections, connection)
		return connection
	}
	x := connect("cloudflare-host", "create")
	h.join(x, "Host", false)
	o := connect("cloudflare-guest", "")
	h.join(o, "Guest", false)
	h.send(x, "start-game", nil)
	readTicTacToe(h, x, o)
	h.send(x, "chat", map[string]string{"message": "Hello from Cloudflare"})
	for _, client := range []*websocket.Conn{x, o} {
		message := decodeAdditional[map[string]interface{}](t, h.read(client, "chat"))
		if message["message"] != "Hello from Cloudflare" || message["senderId"] != "cloudflare-host" {
			t.Fatalf("incorrect cross-origin chat: %v", message)
		}
	}
	h.send(x, "tictactoe-move", map[string]int{"cell": 0})
	readTicTacToe(h, x, o)
	o.Close()
	room := getRoom(h.code)
	h.await(func() bool { return room.disconnectTimers["cloudflare-guest"] != nil })
	o = connect("cloudflare-guest", "")
	if state := readTicTacToe(h, o); state.Board[0] != "X" || *state.CurrentPlayerIdx != 1 {
		t.Fatalf("reconnect lost the game: %+v", state)
	}
	for index, cell := range []int{3, 1, 4, 2} {
		client := o
		if index%2 == 1 {
			client = x
		}
		h.send(client, "tictactoe-move", map[string]int{"cell": cell})
		state := readTicTacToe(h, x, o)
		if index == 3 && (state.Phase != "game_over" || state.WinnerID == nil || *state.WinnerID != "cloudflare-host") {
			t.Fatalf("incorrect completed game: %+v", state)
		}
	}
}
