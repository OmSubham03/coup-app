package game

import (
	"fmt"
	"math/rand"
	"time"
)

// Ludo colors
type LudoColor string

const (
	LudoRed    LudoColor = "red"
	LudoGreen  LudoColor = "green"
	LudoYellow LudoColor = "yellow"
	LudoBlue   LudoColor = "blue"
)

// Ludo phases
type LudoPhase string

const (
	LudoPhaseWaiting  LudoPhase = "waiting"
	LudoPhaseRolling  LudoPhase = "rolling"  // Current player needs to roll
	LudoPhaseMoving   LudoPhase = "moving"   // Current player needs to pick a token to move
	LudoPhaseFinished LudoPhase = "finished" // Game over
)

// Board layout constants
// The main track has 52 squares (0-51)
// Each player has a start position on the main track and a home column of 6 squares
// Player positions (by color index 0=red, 1=green, 2=blue, 3=yellow):
//   Start squares: 40, 1, 14, 27
//   Entry to home column: 39, 0, 13, 26 (the square before their home column)

const (
	BoardSize      = 52 // Main track squares
	HomeColumnSize = 6  // Squares in each player's home column
	TokensPerPlayer = 4
)

// Start positions on the main track for each color index
var StartPositions = [4]int{40, 1, 14, 27}

// Safe squares (star squares) — these positions are safe from capture
var SafeSquares = map[int]bool{
	1: true, 9: true, 14: true, 22: true, 27: true, 35: true, 40: true, 48: true,
}

// LudoToken represents a single token
type LudoToken struct {
	ID       int    `json:"id"`       // 0-3 within player
	State    string `json:"state"`    // "yard", "track", "home_col", "finished"
	Position int    `json:"position"` // track position (0-51) or home column position (0-5)
}

// LudoPlayer represents a player in the game
type LudoPlayer struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Color      LudoColor   `json:"color"`
	ColorIndex int         `json:"colorIndex"` // 0=red, 1=green, 2=blue, 3=yellow
	Tokens     [4]LudoToken `json:"tokens"`
	FinishOrder int        `json:"finishOrder"` // 0 = not finished, 1 = first, etc.
}

// LudoState is the full game state
type LudoState struct {
	ID                string       `json:"id"`
	Players           []LudoPlayer `json:"players"`
	Phase             LudoPhase    `json:"phase"`
	CurrentPlayerIdx  int          `json:"currentPlayerIndex"`
	DiceValue         int          `json:"diceValue"`
	ConsecutiveSixes  int          `json:"consecutiveSixes"`
	MovableTokens     []int        `json:"movableTokens"`     // Token IDs that can move
	LastAction        string       `json:"lastAction"`
	Log               []LudoLogEntry `json:"log"`
	Winner            string       `json:"winner"`
	FinishedCount     int          `json:"finishedCount"`
	TurnNumber        int          `json:"turnNumber"`
}

type LudoLogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Turn      int    `json:"turn"`
}

func addLudoLog(state *LudoState, msg string) {
	state.Log = append(state.Log, LudoLogEntry{
		Timestamp: time.Now().UnixMilli(),
		Message:   msg,
		Turn:      state.TurnNumber,
	})
	state.LastAction = msg
}

// InitializeLudoGame creates a new Ludo game
func InitializeLudoGame(players []struct{ ID, Name, Color string }) *LudoState {
	ludoPlayers := make([]LudoPlayer, len(players))

	for i, p := range players {
		colorIdx := colorToIndex(LudoColor(p.Color))
		ludoPlayers[i] = LudoPlayer{
			ID:         p.ID,
			Name:       p.Name,
			Color:      LudoColor(p.Color),
			ColorIndex: colorIdx,
		}
		for t := 0; t < TokensPerPlayer; t++ {
			ludoPlayers[i].Tokens[t] = LudoToken{
				ID:    t,
				State: "yard",
			}
		}
	}

	// Random first player
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstPlayer := r.Intn(len(players))

	state := &LudoState{
		ID:               fmt.Sprintf("ludo-%d", time.Now().UnixMilli()),
		Players:          ludoPlayers,
		Phase:            LudoPhaseRolling,
		CurrentPlayerIdx: firstPlayer,
		TurnNumber:       1,
		Log:              []LudoLogEntry{},
	}
	addLudoLog(state, fmt.Sprintf("Game started! %s goes first.", ludoPlayers[firstPlayer].Name))
	return state
}

func colorToIndex(c LudoColor) int {
	switch c {
	case LudoRed:
		return 0
	case LudoGreen:
		return 1
	case LudoBlue:
		return 2
	case LudoYellow:
		return 3
	}
	return 0
}

// LudoRollDice handles a player rolling the dice
func LudoRollDice(state *LudoState, playerID string) error {
	if state.Phase != LudoPhaseRolling {
		return fmt.Errorf("not in rolling phase")
	}
	cp := &state.Players[state.CurrentPlayerIdx]
	if cp.ID != playerID {
		return fmt.Errorf("not your turn")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := r.Intn(6) + 1
	state.DiceValue = dice

	addLudoLog(state, fmt.Sprintf("%s rolled a %d", cp.Name, dice))

	if dice == 6 {
		state.ConsecutiveSixes++
		if state.ConsecutiveSixes >= 3 {
			addLudoLog(state, fmt.Sprintf("%s rolled three 6s in a row! Turn lost.", cp.Name))
			state.ConsecutiveSixes = 0
			advanceLudoTurn(state)
			return nil
		}
	}

	// Find movable tokens
	movable := findMovableTokens(state, cp, dice)
	state.MovableTokens = movable

	if len(movable) == 0 {
		addLudoLog(state, fmt.Sprintf("%s has no valid moves", cp.Name))
		if dice != 6 {
			state.ConsecutiveSixes = 0
			advanceLudoTurn(state)
		} else {
			// Got a 6 but no moves, bonus roll
			state.Phase = LudoPhaseRolling
		}
		return nil
	}

	if len(movable) == 1 {
		// Auto-move the only option
		return LudoMoveToken(state, playerID, movable[0])
	}

	state.Phase = LudoPhaseMoving
	return nil
}

// findMovableTokens returns token IDs that can be moved with the given dice value
func findMovableTokens(state *LudoState, player *LudoPlayer, dice int) []int {
	var movable []int
	for i := 0; i < TokensPerPlayer; i++ {
		token := &player.Tokens[i]
		switch token.State {
		case "yard":
			if dice == 6 {
				// Can enter the board — check if start square is not blocked by own token
				startPos := StartPositions[player.ColorIndex]
				if !isBlockedByOwn(state, player, startPos, -1) {
					movable = append(movable, i)
				}
			}
		case "track":
			// Calculate destination
			dest := (token.Position + dice) % BoardSize
			stepsToHome := stepsToHomeEntry(player.ColorIndex, token.Position)

			if stepsToHome < dice {
				// Might enter home column
				homeColPos := dice - stepsToHome - 1
				if homeColPos < HomeColumnSize {
					// Can enter home column — check not blocked by own in home col
					if !isOwnInHomeCol(player, homeColPos, i) {
						movable = append(movable, i)
					}
				}
				// else: overshot, can't move
			} else if stepsToHome == dice {
				// Enters home column position 0
				if !isOwnInHomeCol(player, 0, i) {
					movable = append(movable, i)
				}
			} else {
				// Normal track move
				if !isBlockedByOwn(state, player, dest, i) {
					movable = append(movable, i)
				}
			}
		case "home_col":
			newPos := token.Position + dice
			if newPos == HomeColumnSize {
				// Exact roll to finish!
				movable = append(movable, i)
			} else if newPos < HomeColumnSize {
				if !isOwnInHomeCol(player, newPos, i) {
					movable = append(movable, i)
				}
			}
			// else: overshot, can't move
		}
	}
	return movable
}

// stepsToHomeEntry calculates how many steps from current position to the home entry square
func stepsToHomeEntry(colorIndex int, currentPos int) int {
	// Home entry square is the square just before the home column
	// For colorIndex 0 (red): start=40, so home entry = 39
	// For colorIndex 1 (green): start=1, home entry = 0
	// For colorIndex 2 (blue): start=14, home entry = 13
	// For colorIndex 3 (yellow): start=27, home entry = 26
	homeEntry := (StartPositions[colorIndex] + BoardSize - 1) % BoardSize
	if currentPos <= homeEntry {
		return homeEntry - currentPos
	}
	return BoardSize - currentPos + homeEntry
}

func isBlockedByOwn(state *LudoState, player *LudoPlayer, pos int, excludeToken int) bool {
	// In standard Ludo, multiple own tokens can occupy the same square
	return false
}

func isOwnInHomeCol(player *LudoPlayer, pos int, excludeToken int) bool {
	for i := 0; i < TokensPerPlayer; i++ {
		if i == excludeToken {
			continue
		}
		if player.Tokens[i].State == "home_col" && player.Tokens[i].Position == pos {
			return true
		}
	}
	return false
}

// LudoMoveToken moves a specific token
func LudoMoveToken(state *LudoState, playerID string, tokenID int) error {
	if state.Phase != LudoPhaseMoving && state.Phase != LudoPhaseRolling {
		return fmt.Errorf("not in moving phase")
	}
	cp := &state.Players[state.CurrentPlayerIdx]
	if cp.ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if tokenID < 0 || tokenID >= TokensPerPlayer {
		return fmt.Errorf("invalid token")
	}

	// Verify this token is in the movable list
	found := false
	for _, m := range state.MovableTokens {
		if m == tokenID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("token cannot be moved")
	}

	dice := state.DiceValue
	token := &cp.Tokens[tokenID]
	gotBonus := dice == 6
	captured := false

	switch token.State {
	case "yard":
		// Enter the board at start position
		startPos := StartPositions[cp.ColorIndex]
		token.State = "track"
		token.Position = startPos
		addLudoLog(state, fmt.Sprintf("%s brings a token onto the board", cp.Name))
		// Check for capture at start
		captured = checkCapture(state, cp, startPos)

	case "track":
		stepsToHome := stepsToHomeEntry(cp.ColorIndex, token.Position)

		if stepsToHome < dice {
			// Enter home column
			homeColPos := dice - stepsToHome - 1
			token.State = "home_col"
			token.Position = homeColPos
			addLudoLog(state, fmt.Sprintf("%s moves a token into the home column", cp.Name))
		} else if stepsToHome == dice {
			token.State = "home_col"
			token.Position = 0
			addLudoLog(state, fmt.Sprintf("%s moves a token into the home column", cp.Name))
		} else {
			// Normal track move
			oldPos := token.Position
			newPos := (oldPos + dice) % BoardSize
			token.Position = newPos
			addLudoLog(state, fmt.Sprintf("%s moves a token %d spaces", cp.Name, dice))
			captured = checkCapture(state, cp, newPos)
		}

	case "home_col":
		newPos := token.Position + dice
		if newPos == HomeColumnSize {
			// Token reaches home!
			token.State = "finished"
			token.Position = 0
			addLudoLog(state, fmt.Sprintf("%s gets a token home! 🏠", cp.Name))
			gotBonus = true // Bonus roll for getting home

			// Check if all tokens are finished
			allFinished := true
			for i := 0; i < TokensPerPlayer; i++ {
				if cp.Tokens[i].State != "finished" {
					allFinished = false
					break
				}
			}
			if allFinished {
				state.FinishedCount++
				cp.FinishOrder = state.FinishedCount
				if state.FinishedCount == 1 {
					state.Winner = cp.ID
				}
				addLudoLog(state, fmt.Sprintf("🏆 %s finishes in position %d!", cp.Name, cp.FinishOrder))

				// Check if game is over (only 1 player left)
				playersLeft := 0
				for _, p := range state.Players {
					if p.FinishOrder == 0 {
						playersLeft++
					}
				}
				if playersLeft <= 1 {
					// Assign remaining player's finish order
					for i := range state.Players {
						if state.Players[i].FinishOrder == 0 {
							state.FinishedCount++
							state.Players[i].FinishOrder = state.FinishedCount
						}
					}
					state.Phase = LudoPhaseFinished
					addLudoLog(state, "Game Over!")
					return nil
				}
			}
		} else {
			token.Position = newPos
			addLudoLog(state, fmt.Sprintf("%s moves a token in the home column", cp.Name))
		}
	}

	if captured {
		gotBonus = true // Bonus roll for capture
	}

	// Determine next action
	if gotBonus && cp.FinishOrder == 0 {
		// Bonus roll
		state.Phase = LudoPhaseRolling
		state.MovableTokens = nil
	} else {
		state.ConsecutiveSixes = 0
		advanceLudoTurn(state)
	}

	return nil
}

// checkCapture checks if landing on a position captures an opponent's token
func checkCapture(state *LudoState, currentPlayer *LudoPlayer, pos int) bool {
	// Safe squares can't be captured on
	if SafeSquares[pos] {
		return false
	}

	captured := false
	for i := range state.Players {
		p := &state.Players[i]
		if p.ID == currentPlayer.ID {
			continue
		}
		for t := 0; t < TokensPerPlayer; t++ {
			if p.Tokens[t].State == "track" && p.Tokens[t].Position == pos {
				p.Tokens[t].State = "yard"
				p.Tokens[t].Position = 0
				addLudoLog(state, fmt.Sprintf("%s captures %s's token! 💥", currentPlayer.Name, p.Name))
				captured = true
			}
		}
	}
	return captured
}

func advanceLudoTurn(state *LudoState) {
	state.MovableTokens = nil
	state.TurnNumber++

	// Find next active player (who hasn't finished)
	for i := 1; i <= len(state.Players); i++ {
		nextIdx := (state.CurrentPlayerIdx + i) % len(state.Players)
		if state.Players[nextIdx].FinishOrder == 0 {
			state.CurrentPlayerIdx = nextIdx
			state.Phase = LudoPhaseRolling
			return
		}
	}
	// All finished
	state.Phase = LudoPhaseFinished
}

// LudoVoluntaryExit handles a player leaving mid-game
func LudoVoluntaryExit(state *LudoState, playerID string) {
	for i := range state.Players {
		if state.Players[i].ID == playerID && state.Players[i].FinishOrder == 0 {
			state.FinishedCount++
			state.Players[i].FinishOrder = state.FinishedCount
			// Move all tokens to yard
			for t := 0; t < TokensPerPlayer; t++ {
				state.Players[i].Tokens[t].State = "yard"
				state.Players[i].Tokens[t].Position = 0
			}
			addLudoLog(state, fmt.Sprintf("%s left the game", state.Players[i].Name))

			// If it was their turn, advance
			if state.CurrentPlayerIdx == i {
				state.ConsecutiveSixes = 0
				advanceLudoTurn(state)
			}

			// Check if game is over
			playersLeft := 0
			var lastPlayer int
			for j := range state.Players {
				if state.Players[j].FinishOrder == 0 {
					playersLeft++
					lastPlayer = j
				}
			}
			if playersLeft <= 1 {
				if playersLeft == 1 {
					state.FinishedCount++
					state.Players[lastPlayer].FinishOrder = state.FinishedCount
					if state.Winner == "" {
						state.Winner = state.Players[lastPlayer].ID
					}
				}
				state.Phase = LudoPhaseFinished
				addLudoLog(state, "Game Over!")
			}
			break
		}
	}
}

// GetLudoColorIndex returns the color index for a color string (exported for main.go)
func GetLudoColorIndex(color string) int {
	return colorToIndex(LudoColor(color))
}

// GetPlayerPositionsForColor returns the seat index for a 2-player game (opposite seats)
func GetLudoSeats(numPlayers int, colors []LudoColor) []int {
	if numPlayers == 2 {
		// Players sit opposite: indices 0 and 2
		return []int{colorToIndex(colors[0]), colorToIndex(colors[1])}
	}
	seats := make([]int, numPlayers)
	for i, c := range colors {
		seats[i] = colorToIndex(c)
	}
	return seats
}
