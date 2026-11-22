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

Open `http://localhost:8080`. The responsive TimeCrusher homepage includes all thirteen
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
other players in the waiting lobby and all thirteen web games.
The in-game Chat tab uses the same avatar or initials. Your messages appear
on the right with a **You** label; other players' messages appear on the left.

Activity labels on the homepage use **x live** for the total room membership
across all games. Each game card shows **n rooms** for all nonempty public
rooms (including full rooms and games in progress) and **x live** for members
in that game's public and private rooms. Each zero-valued card label is hidden;
the homepage total and desktop game setup's **x live** label still show zero.
Setup live counts appear to the left of **Game guide** with the same green
indicator dot as the homepage total. Setup live counts are hidden at widths
of 640px or less; homepage totals and card activity badges are unchanged.
All game Rules panels, including UNO house-rule explanations, use shared
panel colors, typography, borders, and close controls. They follow the saved
white/dark-blue theme on room setup screens and during games, including
when the theme changes while a Rules panel is open.
The membership definition is unchanged: reconnect-grace members remain counted,
while extra connections and spectators do not increase the member count.

`GET /api/room-stats` returns `players` plus a `games` object containing
`players` and `publicRooms` for every game, including zero entries.
It exposes only aggregate counts, not private room codes or player identities.
Activity counts refresh on page load and when opening a game setup, sharing
one in-flight request. No periodic activity refresh is added. Existing available
room discovery continues to refresh separately on setup screens. When counts
cannot be loaded, cards hide their badges and the totals show **-- live** with
an unavailable notice rather than claiming there are zero players.

Run the web frontend regression tests with `node --test frontend-tests` from
`server`.

The homepage footer links to the
[TimeCrusher Privacy Policy](https://hobbystudio.github.io/Website/Timecrushers/privacy_policy.html)
and the [Contact Us section](https://hobbystudio.github.io/Website/#contact)
on the Hobby Studio website. Those destinations use Hobby Studio's shared
theme and are maintained in the separate `HobbyStudio/Website` repository.
Contact information is centralized on the studio homepage, not duplicated
as a TimeCrusher contact page. Publish changes to those destinations before
deploying frontend changes that link to them.

### Host only the frontend on Cloudflare Pages

The frontend can run on Cloudflare Pages while the Go server stays on its
existing host. Pages does not run this game's backend. The default frontend
build connects to
`https://timecrushers.politeflower-423ed73a.eastus.azurecontainerapps.io`.

The production frontend is live at [timecrushers.pages.dev](https://timecrushers.pages.dev/).
The existing `timecrushers` Pages project uses Direct Upload, not Git integration.
To publish future frontend changes from the repository root:

```powershell
node .\server\build-frontend.cjs
if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }
npx --yes wrangler pages deploy .\server\dist --project-name timecrushers --branch main --commit-dirty=true
```

Run `npx --yes wrangler login` first if the CLI is not authenticated. Authorize
in the browser; never paste credentials into source or chat. Deploy backend
changes separately with `.\deploy.ps1`. The commands below describe creating
a new project; Git integration requires a separate project.

**Deploy the updated backend first**, using the existing `deploy.ps1` workflow.
The update enables cross-origin reads for public-room discovery and player
counts. These public, credential-free APIs allow any origin, matching the
existing code-generation and variant APIs. WebSockets already accept the
separate frontend origin. No Cloudflare-specific backend environment variable
is required.

In the [Cloudflare dashboard](https://dash.cloudflare.com/), open
**Workers & Pages → Create application → Pages** and use either method:

- **Connect to Git:** push these changes to your repository, connect
  `OmSubham03/coup-app`, and set:

  | Setting | Value |
  | --- | --- |
  | Project name | `timecrushers` (if available) |
  | Production branch | `main` |
  | Framework preset | None |
  | Root directory | `server` |
  | Build command | `node build-frontend.cjs` |
  | Build output directory | `dist` |

  Optionally set the build environment variable `BACKEND_URL` to a different
  HTTPS backend origin. Set it for both production and previews when overriding
  the default. This is a public address, not a secret.

- **Upload assets:** run the following locally, then upload the contents of
  `server/dist` (or a ZIP containing those contents at its root):

  ```powershell
  Set-Location server
  node build-frontend.cjs
  ```

  To target a different backend before building:

  ```powershell
  $env:BACKEND_URL = 'https://your-backend.example.com'
  node build-frontend.cjs
  ```

Choose the deployment method when creating the project; a Direct Upload
project cannot later be converted to Git integration. The requested address
is `https://timecrushers.pages.dev`, subject to name availability.

The build publishes only `client.html` as `index.html` and the contents of
`public`; it does not upload Go source, binaries, or the mobile app.
Generated `dist/js/config.js` selects the remote HTTP and WebSocket backend
for all games, including reconnects. Source `public/js/config.js` keeps
the existing same-host/local-development behavior. Do not put API tokens
in either config file. HTML, scripts, styles, and the service worker use
no-store response headers on Pages; the service worker never caches
cross-origin backend requests.

After deployment, check the player count, public-room discovery, room creation,
joining from another browser, a complete game, chat, and reconnects.
Browser-local profiles, theme preferences, and saved sessions do not transfer
automatically from the old Azure address to the new Pages address.
Cloudflare hosting does not enable AdSense by itself.

The homepage includes the public AdSense ownership-verification meta tag for
`ca-pub-2847092601328775`. It does not load Google's advertising script or
enable ads. After publishing, select **Meta tag** in AdSense, confirm the tag
is placed, and click **Verify**. Site review, consent setup, and an updated
privacy policy are separate requirements before activating advertising.

### Google Search indexing

The homepage includes a separate `google-site-verification` tag for the
Search Console URL-prefix property `https://timecrushers.pages.dev/`.
It also has a descriptive title, meta description, and canonical URL pointing
to that production homepage. The verification tags are public, not credentials,
and neither tag enables ads.

After deployment, verify ownership in [Google Search Console](https://search.google.com/search-console/).
Inspect `https://timecrushers.pages.dev/`, run **Test live URL**, then select
**Request indexing**. Submit `sitemap.xml` under **Sitemaps**.
The frontend build publishes `public/robots.txt` and `public/sitemap.xml` at
the site root. The sitemap lists the homepage and all thirteen public game guides.
Hobby Studio
policy/contact links belong to a different website and are not in this sitemap.
This does not update the currently deployed Azure homepage; its next backend
deployment will include the same canonical tag from `client.html`.

Search Console verification and crawl requests do not guarantee indexing,
ranking, or a particular search-result title. Google may choose different
titles or descriptions based on the query and page content.

### Public game guides

All thirteen guides are static HTML in `server/public/games`, with original
explanations, implemented rules, room instructions, examples, and tips.
Cloudflare Pages serves each folder's `index.html` at its trailing-slash URL.
The Go static-file routes also serve guides and crawl metadata for local
development and future same-host deployments. This release deploys only the
Cloudflare frontend; the currently running Azure container is unchanged.
Each game's room-setup toolbar has a separate **Game guide** link styled as a
button beside the unchanged **Rules** button. The guide opens in a new tab
so the room settings and code entry remain intact. There is no guide grid
on the main homepage. The links are regular HTML anchors, and all guide URLs
remain in the sitemap. Each guide links to related guides, the game collection,
and the game's existing setup using a `game` query parameter.

| Guide URL | Game query value |
| --- | --- |
| `/games/coup/` | `coup` |
| `/games/ludo/` | `ludo` |
| `/games/tic-tac-toe/` | `tictactoe` |
| `/games/poker/` | `poker` |
| `/games/20-questions/` | `nquestions` |
| `/games/commune/` | `commune` |
| `/games/twenty-nine/` | `twentynine` |
| `/games/hearts/` | `hearts` |
| `/games/uno/` | `uno` |
| `/games/bluff/` | `bluff` |
| `/games/blackjack/` | `blackjack` |
| `/games/scrabble/` | `scrabble` |
| `/games/kabo/` | `kabo` |

Guides share the app's white/dark-blue colors and the same saved theme preference
through `public/js/theme.js`. They load no game scripts, service worker, APIs,
room discovery, or WebSockets, so reading them does not wake the Azure backend.
Their content and links remain available without JavaScript; only the saved
theme and theme switch require it. They do not create additional containers.
Guide HTML uses no-store headers, and the service worker bypasses guide
pages, the sitemap, and robots.txt rather than retaining stale HTML or crawl
metadata across releases. Artwork and icons remain cached as before.

The app recognizes `game` query values using its existing game-capacity keys.
A saved room and joined name take priority and reconnect as before; a query
does not create a room or discard a session. Invalid, empty, or duplicate
`game` parameters leave the collection available with a visible notification.
When adding another guide, verify its rules against the corresponding game
implementation, add room-setup and related-page links, and update the sitemap.

### Kabo rules and implementation

Kabo is a 2-6 player, single-deck memory game. Four hidden cards are dealt to
each player; initial two-card peeks are private and hidden again after ready.
Ranks score their number, J 11, Q 12, black K 13, red K -1; no jokers.
Deck-drawn discards may activate 7/8 own peek, 9/10 opponent peek, J/Q blind
swap, or black K two-card peek followed by an optional swap. Discard-pile draws
must replace a player's card and do not activate a power.

Matching opens after an action and power resolve, before the next draw.
One attempt per player per discard avoids repeated blind penalty attempts.
The first successful rank match closes the opportunity (no chains);
wrong attempts leave the target and add a hidden penalty card. Opponent
matches discard their card and transfer one of the actor's cards.
The server validates opaque slot IDs, discard IDs and revisions under the
room lock, so stale/racing commands cannot move the same card twice.

Kabo is called after drawing and before resolving the draw. The caller must
finish that card and power, then other active players get one final turn.
Empty hands end the round immediately. Actual card totals, including negatives,
are added without caller bonuses or penalties. The host may redeal or end after
a round; 100 cumulative points automatically ends the match. Ties share wins.
Leaving forfeits participation; remaining players continue, or the last player
wins when fewer than two remain. Standard five-minute reconnect grace applies.

Private ranks live only in server memory. Kabo state views hide all cards until
round results, apart from active authorized peeks and the current actor's draw.
Acknowledging a peek removes it from the next state, including reconnect views.
Matched cards leave empty positions instead of shifting the other hidden cards;
penalty cards add new positions.
Kabo message logs omit payloads. The UI has larger tappable opponent cards,
two-column private hands (including penalty cards), confirmation for transfers,
shared chat/avatars/settings/themes/rules, and a static guide at `/games/kabo/`.
No third-party service or additional container is required.

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
