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
go run .
```

The server starts on port **8080** and provides:
- `ws://localhost:8080/ws` — WebSocket game endpoint
- `GET /api/generate-code` — Generate room codes
- `GET /api/variant-config?variant=standard` — Get variant config
- `GET /health` — Health check

### Web game hub

Blackjack uses a green felt table with larger dealer cards above the players.
The dealer's total appears only after the hidden card is revealed. Player score
badges sit beside each wager chip, with Double, Hit and Stand below the curved
rail. Wagers are shown in game chips; the table does not offer insurance.

**Tic-Tac-Toe** is a two-player room game under **Board games**. Create a
room, share its code, and start once both players have joined. X moves first;
three marks in a row win, and a full board without a line is a draw.

**Scrabble** is a 2–4 player board game with private seven-tile racks. Place
tiles in a line, use premium squares, and build off previous words. The
server checks every new word against the bundled, public-domain
[ENABLE lexicon (172,823 entries)](https://github.com/dolph/dictionary/blob/233d25a56b9ac8bef906916ccf091a976995cce0/enable1.txt);
it is not an official tournament dictionary. Unknown words go to a house
vote: the proposer counts as approving, every other active player must
approve, and one rejection returns the tiles without consuming the turn.
Placement rules cannot be overridden by a vote. No external dictionary
service or API key is required.

Open `http://localhost:8080`. The responsive TimeCrusher homepage includes all twelve
games, category filters, game search, and a white / dark-blue theme switch.
The theme preference is saved in browser local storage. Homepage styles are
shared by the game hub, room setup, and waiting lobby. The lobby includes a
copyable invite code and player roster. Texas Hold'em also follows the saved
theme, with a scrollable player row, bet chips aligned beneath each seat on a straight table rail, five community-card slots on green felt,
and personal cards above the betting controls. Other games keep their own styling.
Illustrations are original local SVG assets; the homepage does not require
third-party fonts or image services.

At widths of 640px or less, the homepage hero and room setup banners hide
their decorative illustrations. Game cards keep their artwork.

Use the same sun/moon button on the homepage, beside **Theme** in the waiting
lobby, or under **Settings → Theme** during any game. This preference is saved in your
browser and does not change the theme for other players. All active games use
the same page background and a single narrow outer gutter on phones. Boards,
cards, and felt keep their game-specific colors.

Hearts uses a poker-style player rail above the green felt. Each player's chip
shows penalty points for the current hand; their total score is listed separately.
Trick cards stay upright in their seat positions. Your private hand sits along
the bottom edge of the felt during both passing and play. On phones, the
**Play Card** button spans the width of the table.

The Coup table puts opponents' coins and public influence in a horizontal row.
Your private role portraits appear below it, above the action buttons. Spectators
see only public influence; revealed roles are marked **Lost**.

Commune displays remaining tokens as numbered poker chips on the straight
table border, aligned with each player's seat. The final standings use the
same numbered chips.
The latest declaration appears in a speech bubble beside the player's
avatar and name. A call adds the challenger's name below the bubble.
Earlier declarations can be expanded separately. The shared
cards stay on the felt, with your private
hand below. When a declaration is called, the table reveals the hands.
To declare, choose a hand by its full name from the strength-ordered picker,
then select any required ranks. Selecting a hand does not submit it.

Open your profile on the homepage to choose one of nine animal avatars or
select **Initials**. Your choice is saved in this browser and shared with
other players in the waiting lobby and all twelve web games.
The in-game Chat tab uses the same avatar or initials. Your messages appear
on the right with a **You** label; other players' messages appear on the left.

Run the web frontend regression tests with `node --test frontend-tests` from
`server`.

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
