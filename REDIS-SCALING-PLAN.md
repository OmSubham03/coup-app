# Redis-Backed Horizontal Scaling Plan

## Overview

Move all game room state from in-memory Go maps to Azure Cache for Redis, enabling multiple server instances to serve the same rooms. WebSocket connections remain local to each instance; Redis Pub/Sub ensures broadcasts reach all clients regardless of which instance they're connected to.

---

## Current Architecture

```
[Client A] --ws--> [Single Go Instance] --in-memory--> rooms map
[Client B] --ws--> [Same Instance]
```

- **State**: Global `rooms` map (`map[string]*Room`) protected by `sync.RWMutex`
- **Room struct** holds everything: game state, player list, WebSocket connections, disconnect timers
- **Broadcasts**: Direct iteration over `room.connections` map
- **Cleanup**: Goroutine every 1 min deletes rooms with 0 connections + 10 min inactive

## Target Architecture

```
[Client A] --ws--> [Instance 1] --read/write--> [Azure Redis]
[Client B] --ws--> [Instance 2] --read/write--> [Azure Redis]
                        |                            |
                        +--- Redis Pub/Sub ----------+
```

---

## Files to Create

### 1. `server/store.go` — Redis State Store

**Purpose**: Abstract all room state persistence behind an interface, with Redis and in-memory implementations.

**Interface**:
```go
type RoomStore interface {
    GetRoom(code string) (*RoomData, error)
    SaveRoom(code string, data *RoomData) error
    DeleteRoom(code string) error
    ListRoomCodes() ([]string, error)
    RoomExists(code string) (bool, error)
}
```

**RoomData struct** (serializable room state — everything EXCEPT WebSocket connections and timers):
```go
type RoomData struct {
    Code         string                  `json:"code"`
    GameType     string                  `json:"gameType"`
    Variant      game.VariantKey         `json:"variant"`
    Players      map[string]*PlayerConn  `json:"players"`
    HostID       string                  `json:"hostId"`
    Created      bool                    `json:"created"`
    PokerConfig  *PokerConfig            `json:"pokerConfig,omitempty"`
    LudoColors   map[string]string       `json:"ludoColors,omitempty"`
    LastActivity time.Time               `json:"lastActivity"`
    // Game states (only one active at a time)
    GameState    *game.GameState         `json:"gameState,omitempty"`
    PokerState   *game.PokerState        `json:"pokerState,omitempty"`
    LudoState    *game.LudoState         `json:"ludoState,omitempty"`
}
```

**Redis implementation**:
- Key format: `room:{code}` — stores gob-encoded `RoomData`
- TTL: 30 minutes (auto-expire abandoned rooms, refreshed on every `SaveRoom`)
- Uses `encoding/gob` (NOT `encoding/json`) because 4 fields have `json:"-"` tags that must still be persisted:
  - `PokerState.Deck` — the shuffled deck (hidden from client but needed server-side)
  - `PokerState.LastRaiserIdx` — tracks betting round logic
  - `PokerPlayer.RoundActed` — whether player acted this betting round
  - `EvaluatedHand.Kickers` — tie-breaking data
- `gob` serializes all exported fields regardless of json tags

**In-memory fallback implementation**:
- Same as current behavior (map + mutex)
- Used when `REDIS_URL` env var is not set
- Allows local development without Redis

### 2. `server/pubsub.go` — Redis Pub/Sub for Cross-Instance Broadcasts

**Purpose**: When instance 1 needs to broadcast a game state update, it publishes to Redis. Instance 2, which also has players in that room, receives the message and sends it to its local WebSocket connections.

**Channel format**: `room:{code}:broadcast`

**Message types published**:
```go
type PubSubMessage struct {
    Type       string          `json:"type"`       // "broadcast", "send-to-player", "send-to-conn"
    RoomCode   string          `json:"roomCode"`
    TargetID   string          `json:"targetId,omitempty"`  // playerID or connID
    OutMessage json.RawMessage `json:"message"`
    SenderInstance string      `json:"senderInstance"`      // Instance ID to avoid echo
}
```

**Flow**:
1. `room.broadcast(msg)` → sends to local connections + publishes to Redis channel
2. All instances subscribed to `room:{code}:broadcast` receive the message
3. Each instance checks if it has local connections for that room and sends to them
4. `SenderInstance` field prevents double-delivery to local connections

**Subscription management**:
- Each instance subscribes to room channels when a WebSocket connects to that room
- Unsubscribes when last local connection to that room disconnects
- Uses `redis.PubSub` with a goroutine per subscription

---

## Files to Modify

### 3. `server/main.go` — Major Refactoring

**Room struct split** — separate local connection tracking from persisted state:

```go
// LocalRoom holds only instance-local data (connections, timers)
type LocalRoom struct {
    mu               sync.Mutex
    code             string
    connections      map[string]*websocket.Conn // connID -> ws conn (LOCAL to this instance)
    connPlayer       map[string]string          // connID -> playerID (LOCAL)
    disconnectTimers map[string]*time.Timer     // LOCAL timers
}
```

**Changes to global state**:
```go
// BEFORE (current)
var rooms = make(map[string]*Room)
var roomsMu sync.RWMutex

// AFTER
var localRooms = make(map[string]*LocalRoom)  // Only local connections
var localRoomsMu sync.RWMutex
var store RoomStore                            // Redis or in-memory
var pubsub *PubSubManager                      // Redis pub/sub (nil if in-memory)
var instanceID = uuid.New().String()           // Unique per instance
```

**Function-by-function changes**:

| Function | Current Behavior | New Behavior |
|----------|-----------------|-------------|
| `getOrCreateRoom()` | Creates Room in memory map | Creates `LocalRoom` locally + `RoomData` in Redis via `store.GetRoom()`/`store.SaveRoom()` |
| `getRoom()` | Reads from memory map | Reads from `store.GetRoom()` |
| `broadcast()` | Iterates `room.connections` directly | Sends to local connections + publishes to Redis pub/sub |
| `sendTo()` | Writes to local conn map | Tries local first; if not found, publishes targeted message via pub/sub |
| `sendToPlayer()` | Iterates local connPlayer map | Same pattern: local first + pub/sub |
| `broadcastState()` | Builds personalized state, sends to each local conn | Builds state, sends to local conns, publishes generic broadcast for other instances |
| `handleWS()` | Reads/writes Room directly | Loads `RoomData` from store, processes, saves back. Subscribes to pub/sub channel |
| `handleMessage()` | Mutates room state directly | Pattern: `load from store → mutate → save to store → broadcast via pub/sub` |
| `defer` cleanup (disconnect) | Modifies room directly | Load → modify → save, then publish player-left event |
| Room cleanup goroutine | Iterates rooms map, deletes stale | For Redis: relies on TTL auto-expiry. For local rooms: cleanup when last local conn disconnects |

**Disconnect timer handling**:
- Timers are instance-local (can't serialize a `*time.Timer`)
- If a player disconnects from instance 1 and reconnects to instance 2, instance 2 must cancel instance 1's timer
- Solution: Store `disconnecting:{roomCode}:{playerID}` key in Redis with 120s TTL. On reconnect (any instance), delete the key. A goroutine on each instance checks for expired keys and processes eliminations.

**Voice relay** (`voice-data`, `voice-join`, `voice-leave`):
- These are high-frequency, ephemeral messages that should NOT go through Redis state store
- Route directly through pub/sub (publish raw audio relay messages)
- No state persistence needed — just cross-instance relay

### 4. `server/go.mod` — Add Redis Dependency

```
require (
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/redis/go-redis/v9 v9.x.x
)
```

### 5. `server/Dockerfile` — No Changes Needed

The Dockerfile already does `go mod download` and builds a static binary. Redis client is a pure Go library, no system dependencies.

### 6. `server/game/poker.go` — Register Types for Gob

Add `init()` function to register types needed for gob encoding:
```go
func init() {
    gob.Register(PokerState{})
    gob.Register([]PokerPlayer{})
    gob.Register([]PokerCard{})
    // ... etc for all nested types
}
```

Same for `logic.go` (GameState types) and `ludo.go` (LudoState types).

---

## Infrastructure Changes

### 7. Provision Azure Cache for Redis

```powershell
# Create Redis instance (Basic C0 = cheapest, ~$16/month)
az redis create `
  --name coup-redis `
  --resource-group coup-rg `
  --location eastus `
  --sku Basic `
  --vm-size c0 `
  --redis-version 6

# Get connection string
az redis list-keys --name coup-redis --resource-group coup-rg
```

### 8. Update Azure Container App — Environment Variable

```powershell
az containerapp update `
  --name coup-server `
  --resource-group coup-rg `
  --set-env-vars "REDIS_URL=rediss://:ACCESS_KEY@coup-redis.redis.cache.windows.net:6380/0"
```

### 9. Update `deploy.ps1` — Scale to Multiple Instances

Add after container update:
```powershell
# Scale to 2+ replicas
az containerapp update --name coup-server --resource-group coup-rg `
  --min-replicas 2 --max-replicas 5
```

---

## Message Flow Example (After Changes)

**Player A (Instance 1) makes a poker raise:**

1. Instance 1 receives WebSocket message `{"type":"poker-action","payload":{"action":"raise","amount":50}}`
2. Instance 1 loads `RoomData` from Redis: `GET room:ABC12`
3. Instance 1 calls `game.PokerAction(roomData.PokerState, ...)`
4. Instance 1 saves mutated state: `SET room:ABC12 <gob-encoded> EX 1800`
5. Instance 1 builds personalized state for each local connection, sends via WebSocket
6. Instance 1 publishes to `room:ABC12:broadcast`: `{"type":"broadcast","roomCode":"ABC12","senderInstance":"inst-1"}`
7. Instance 2 receives pub/sub message, loads fresh state from Redis
8. Instance 2 builds personalized state for its local connections, sends via WebSocket

---

## Race Condition Handling

**Problem**: Two instances could load, mutate, and save state simultaneously.

**Solution**: Redis optimistic locking with `WATCH`/`MULTI`/`EXEC`:
```go
func (s *RedisStore) UpdateRoom(code string, fn func(*RoomData) error) error {
    key := "room:" + code
    for retries := 0; retries < 5; retries++ {
        err := s.client.Watch(ctx, func(tx *redis.Tx) error {
            data, _ := tx.Get(ctx, key).Bytes()
            room := decodeRoomData(data)
            if err := fn(room); err != nil {
                return err
            }
            _, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
                pipe.Set(ctx, key, encodeRoomData(room), 30*time.Minute)
                return nil
            })
            return err
        }, key)
        if err == nil {
            return nil
        }
        if err == redis.TxFailedErr {
            continue // Retry on conflict
        }
        return err
    }
    return fmt.Errorf("too many conflicts")
}
```

This ensures that if two instances try to modify the same room simultaneously, one will retry with the updated state.

---

## Rollback / Fallback Strategy

- When `REDIS_URL` environment variable is **not set**, the server falls back to in-memory storage (current behavior)
- This means:
  - Local development works unchanged (no Redis needed)
  - If Redis goes down, you can scale to 1 replica and remove `REDIS_URL` to restore current behavior
  - Feature flag: set `REDIS_URL` to enable, unset to disable

---

## Estimated Cost Impact

| Resource | Current | After |
|----------|---------|-------|
| Container App (1 replica) | ~$0 (free tier) | ~$0 per replica |
| Azure Cache for Redis (Basic C0) | $0 | ~$16/month |
| Container App (2 replicas) | N/A | ~$0 (consumption plan) |

---

## Testing Plan

1. **Local**: Run server without `REDIS_URL` → verify in-memory fallback works identically
2. **Local + Redis**: Run local Redis (Docker), set `REDIS_URL=redis://localhost:6379` → test single instance with Redis store
3. **Two local instances**: Run 2 server instances on different ports with same Redis → verify cross-instance broadcasts
4. **Deploy**: Deploy to Azure with Redis → verify with 2 replicas
5. **Failover**: Kill one replica mid-game → verify other replica picks up (player reconnects to surviving instance)

---

## File Summary

| File | Action | Description |
|------|--------|-------------|
| `server/store.go` | **CREATE** | `RoomStore` interface + Redis implementation + in-memory fallback |
| `server/pubsub.go` | **CREATE** | Redis Pub/Sub manager for cross-instance broadcasts |
| `server/main.go` | **MODIFY** | Split Room struct, use store/pubsub, load/save pattern for all handlers |
| `server/go.mod` | **MODIFY** | Add `github.com/redis/go-redis/v9` |
| `server/game/poker.go` | **MODIFY** | Add gob type registrations in `init()` |
| `server/game/logic.go` | **MODIFY** | Add gob type registrations in `init()` |
| `server/game/ludo.go` | **MODIFY** | Add gob type registrations in `init()` |
| `deploy.ps1` | **MODIFY** | Add Redis provisioning + multi-replica scaling |
| **Azure Redis** | **PROVISION** | `az redis create` in coup-rg |
| **Container App** | **UPDATE** | Add `REDIS_URL` env var, set min-replicas=2 |
