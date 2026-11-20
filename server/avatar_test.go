package main

import "testing"

func TestAvatarSharedWithRoomAndReturningConnections(t *testing.T) {
	h := newAdditionalWSTest(t, "blackjack")
	const hostID = "avatar-host"
	host := h.connect(hostID, "create", "")
	h.read(host, "room-visibility")
	h.send(host, "join", map[string]string{"playerName": "Fox", "avatar": "unknown"})
	h.read(host, "error")
	h.send(host, "join", map[string]string{"playerName": "Fox", "avatar": "fox"})
	type metadata struct {
		Avatars map[string]string `json:"avatars"`
	}
	if got := decodeAdditional[metadata](t, h.read(host, "room-visibility")).Avatars[hostID]; got != "fox" {
		t.Fatalf("host avatar = %q", got)
	}
	guest := h.connect("avatar-guest", "", "")
	if got := decodeAdditional[metadata](t, h.read(guest, "room-visibility")).Avatars[hostID]; got != "fox" {
		t.Fatalf("new connection saw host avatar %q", got)
	}
	h.join(guest, "Initials", false)
	profiles := decodeAdditional[metadata](t, h.read(host, "room-visibility")).Avatars
	if profiles[hostID] != "fox" || profiles["avatar-guest"] != "" {
		t.Fatalf("unexpected profiles: %v", profiles)
	}
	h.send(host, "start-game", nil)
	h.read(host, "blackjack-state")
	host.Close()
	returned := h.connect(hostID, "", "")
	if got := decodeAdditional[metadata](t, h.read(returned, "room-visibility")).Avatars[hostID]; got != "fox" {
		t.Fatalf("reconnected host avatar = %q", got)
	}
}
