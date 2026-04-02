# Coup Game - Mobile App

A mobile implementation of the Coup card game using **Go** (WebSocket server) and **React Native** (Android app).

## Project Structure

```
coup-app-2/
├── server/          # Go WebSocket game server
│   ├── main.go      # Server entry point, WebSocket handling
│   └── game/        # Game logic package
│       ├── logic.go      # Full game logic (actions, challenges, blocking)
│       ├── errors.go     # Error definitions
│       └── variants.go   # Standard & Inquisitor variant configs
│
└── mobile/          # React Native Android app
    ├── src/
    │   ├── App.tsx              # Root app with navigation
    │   ├── types.ts             # TypeScript game types
    │   ├── hooks/
    │   │   └── useGameConnection.ts  # WebSocket connection hook
    │   ├── screens/
    │   │   ├── MainMenu.tsx     # Main menu
    │   │   ├── CreateGame.tsx   # Create game flow
    │   │   ├── JoinGame.tsx     # Join game flow
    │   │   └── GameRoom.tsx     # In-game screen (lobby + gameplay)
    │   └── components/
    │       ├── GameBoard.tsx        # Player cards display
    │       ├── ActionPanel.tsx      # Action selection
    │       ├── BlockChallengePanel.tsx  # Block/challenge UI
    │       ├── CardSelector.tsx     # Card picker for exchange/influence loss
    │       └── GameLog.tsx          # Game event log
    └── android/     # Android native project
```

## Prerequisites

- **Go** 1.21+ (for the server)
- **Node.js** 18+ and npm
- **React Native CLI** (`npm install -g react-native`)
- **Android Studio** with:
  - Android SDK 35
  - NDK 27.1.12297006
  - JDK 17
- **Android device or emulator**

## Setup & Run

### 1. Start the Go Server

```bash
cd server
go mod download
go run main.go
```

The server starts on port **8080** and provides:
- `ws://localhost:8080/ws` — WebSocket game endpoint
- `GET /api/generate-code` — Generate room codes
- `GET /api/variant-config?variant=standard` — Get variant config
- `GET /health` — Health check

### 2. Configure Server Address

In `mobile/src/hooks/useGameConnection.ts` and `mobile/src/screens/CreateGame.tsx`, update `SERVER_HOST`:

```typescript
// For Android emulator connecting to host machine:
const SERVER_HOST = '10.0.2.2:8080';

// For physical device on same network:
const SERVER_HOST = '192.168.x.x:8080';
```

### 3. Install Mobile Dependencies

```bash
cd mobile
npm install
```

### 4. Run on Android

```bash
# Start Metro bundler
npx react-native start

# In another terminal
npx react-native run-android
```

### 5. Build Release APK

```bash
cd mobile/android
./gradlew assembleRelease
```

The APK will be at: `mobile/android/app/build/outputs/apk/release/app-release.apk`

## Game Features

- **Full Coup card game** with all rules implemented
- **2-6 players** multiplayer via WebSocket
- **Two variants**: Standard (Ambassador) and Inquisitor
- **Real-time gameplay**: Actions, blocking, challenges
- **Game lobby**: Create/join rooms with 5-character codes
- **Game log**: Full event history
- Host controls: Start game, kick players, return to lobby

## Google Play Deployment

To publish to Google Play:

1. **Generate a release keystore**:
   ```bash
   keytool -genkeypair -v -storetype PKCS12 -keystore release.keystore \
     -alias coup-key -keyalg RSA -keysize 2048 -validity 10000
   ```

2. **Update** `android/app/build.gradle.kts` signing config with your keystore

3. **Build the release bundle**:
   ```bash
   cd mobile/android
   ./gradlew bundleRelease
   ```

4. Upload the `.aab` from `app/build/outputs/bundle/release/` to Google Play Console

## Architecture

The Go server handles all game logic server-side — the client simply sends actions and renders state. This prevents cheating and ensures game integrity.

Communication flow:
1. Client opens WebSocket to server with room code + player ID
2. Client sends JSON messages (join, action, block, challenge, etc.)
3. Server validates, processes game logic, broadcasts updated state to all players
4. All clients re-render based on the authoritative server state
