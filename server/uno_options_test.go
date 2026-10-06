package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestUNORoomHouseRulesAreIndependentAndBroadcast(t *testing.T) {
	for _, stacking := range []bool{false, true} {
		for _, multiSkip := range []bool{false, true} {
			t.Run(fmt.Sprintf("stacking_%t_multiSkip_%t", stacking, multiSkip), func(t *testing.T) {
				room := discoveryRoom("UNORULES", "uno", false, 2)
				room.unoStackingEnabled = stacking
				room.unoMultiSkipEnabled = multiSkip
				handleMessage(room, "no-connection", room.hostID, InMessage{Type: "start-game"})
				if room.unoState == nil || room.unoState.StackingEnabled != stacking || room.unoState.MultiSkipEnabled != multiSkip {
					t.Fatal("room options were not applied independently to the game")
				}
				encoded, err := json.Marshal(room.visibilityMessage())
				if err != nil {
					t.Fatal(err)
				}
				var message struct {
					Payload struct {
						Stacking  bool `json:"unoStackingEnabled"`
						MultiSkip bool `json:"unoMultiSkipEnabled"`
					}
				}
				if err := json.Unmarshal(encoded, &message); err != nil {
					t.Fatal(err)
				}
				if message.Payload.Stacking != stacking || message.Payload.MultiSkip != multiSkip {
					t.Fatal("reconnecting clients would receive incorrect house rules")
				}
			})
		}
	}
}
