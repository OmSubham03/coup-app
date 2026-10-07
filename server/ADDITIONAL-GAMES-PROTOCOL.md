# Bluff and Blackjack Server Protocol

## Connection and Lobby

Connect to `/ws?room=CODE&playerId=STABLE_ID&gameType=bluff&action=create&twoDecks=true`
or use `gameType=blackjack`. Use the existing stable player ID format (at least
eight characters, as required by existing server logging).

`action=create` creates the session; omit it when joining/reconnecting. `twoDecks`
is true only for the exact query value `true`, defaults to false, and is stored
only on the first creation. Join/reconnect queries and subsequent create attempts
cannot change it. There is no deck-setting message.

Messages use `{"type":"NAME","payload":...}`. Send `join` with
`{"playerName":"Name"}` after connecting to a lobby. Discovery joins add
`"publicRoom":true`; availability is rechecked on receipt. A full lobby rejects
both discovery and direct-code joins. Bluff supports 2-6 players; Blackjack
supports 1-4 players.

The existing `waiting`, `players-updated`, and `room-visibility` events carry
`twoDecks` in the lobby. `room-visibility` carries `public` and `hostId` as well.
Rooms are private by default. Only the host may send `set-room-visibility` with
`{"public":true}` or `{"public":false}`, and only before a game has started.
`GET /api/public-rooms?gameType=bluff` and `gameType=blackjack` use the existing
summary shape: `code`, `gameType`, `variant`, `members`, `capacity`. Started
rooms, including terminal results awaiting lobby reset, are excluded.

## Actions and State

Only the host can send `start-game` (no payload). Initialization uses the stored
deck setting. There are no separate `bluff-started` or `blackjack-started` events.

| Message | Payload | Restriction |
| --- | --- | --- |
| `bluff-play` | `{"cardIds":["bluff_0_clubs_1"]}` | Current player; 1-4 owned card IDs |
| `bluff-challenge` | none | Engine-eligible challenger |
| `bluff-accept` | none | Next player after the pending actor |
| `blackjack-hit` | none | Current active player |
| `blackjack-stand` | none | Current active player |
| `blackjack-double` | none | Current player; engine validates cards/chips |
| `blackjack-next-round` | none | Host and active engine player; round must be over |

Initial state, updates, reconnect views, and `spectate` responses all use
`bluff-state` or `blackjack-state`, with the sanitized engine state as payload.
There are no separate spectator event names or spectator flags. Clients determine
player participation using their stable ID and the state's player/active fields;
clients requesting `spectate` already know they requested a spectator view.

Bluff reveals only the recipient's hand; other hands and the pile retain their
lengths but contain zero-valued cards (`id:""`, `suit:""`, `rank:0`). Spectator
views hide every hand. Blackjack hands are public by engine design; while playing,
only the dealer's first card is visible, and subsequent dealer cards are
zero-valued. Dealer cards become visible after settlement. Neither engine's
internal pending-card data or shoe is serialized. Actions rejected by the server
or engine produce `error` with `{"message":"..."}`.

## Lifecycle

`exit-game` forfeits the participant through the engine, broadcasts the game
state, and sends the exiting connection `state:null` and `players-updated`.
It retains lobby membership, but exited players cannot act or trigger active-room
redirects. `leave-room` forfeits if necessary, removes membership and all player
connections, stops their timer, transfers hosting if needed, and sends `room-left`
before closing. Spectator departures do not forfeit a participant.

Disconnects of active participants allow the existing 300-second reconnect grace.
Reconnect cancels the timer and restores the sanitized view. Expiry invokes the
engine exit, removes membership, transfers hosting, and broadcasts updated state
and room metadata. Blackjack's `round_over` is still an active session for this
purpose. Spectators and already-exited participants do not get a forfeit timer.
Trying another room while participating redirects with the existing `redirect`
payload containing `message`, `roomCode`, and `gameType`.

`return-to-lobby` is allowed for a room member only once the engine reaches
`game_over` (or if no state exists). It clears the game and pending timers, retains
only connected room members, and broadcasts `state:null`, `room-visibility`, and
`players-updated`. Deck settings persist for the next start. Hosts cannot reset
an active Blackjack session merely because a round has ended.

## Integration Boundary

This implementation is server-only; no browser/mobile UI wiring is included.
Blackjack card visibility, fresh shuffled shoes each round, fixed 50-chip bets,
settlement rules, and terminal conditions are those of the supplied engine.
Player identity/authentication and short-ID logging behavior remain the existing
server's responsibility and were not changed here.