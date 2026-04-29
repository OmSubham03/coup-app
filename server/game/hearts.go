package game

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// Hearts phases
type HeartsPhase string

const (
	HT_PhaseWaiting   HeartsPhase = "waiting"
	HT_PhasePassing   HeartsPhase = "passing"
	HT_PhasePlaying   HeartsPhase = "playing"
	HT_PhaseHandOver  HeartsPhase = "hand_over"
	HT_PhaseGameOver  HeartsPhase = "game_over"
)

// Pass directions cycle
type HeartsPassDir int

const (
	HT_PassLeft   HeartsPassDir = 0
	HT_PassRight  HeartsPassDir = 1
	HT_PassAcross HeartsPassDir = 2
	HT_PassNone   HeartsPassDir = 3
)

var htPassDirNames = []string{"Left", "Right", "Across", "No Pass"}

type HTCard struct {
	ID   string `json:"id"`
	Rank int    `json:"rank"` // 2-14 (14=Ace)
	Suit string `json:"suit"` // hearts, diamonds, clubs, spades
}

func (c HTCard) Display() string {
	suits := map[string]string{"hearts": "♥", "diamonds": "♦", "clubs": "♣", "spades": "♠"}
	ranks := map[int]string{2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "10", 11: "J", 12: "Q", 13: "K", 14: "A"}
	return ranks[c.Rank] + suits[c.Suit]
}

func (c HTCard) IsHeart() bool    { return c.Suit == "hearts" }
func (c HTCard) IsQueenSpades() bool { return c.Suit == "spades" && c.Rank == 12 }
func (c HTCard) Points() int {
	if c.IsHeart() {
		return 1
	}
	if c.IsQueenSpades() {
		return 13
	}
	return 0
}

type HTPlayer struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Cards      []HTCard `json:"cards"`
	PassCards  []string `json:"passCards"`  // card IDs selected for passing
	HasPassed  bool     `json:"hasPassed"`
	HandPoints int      `json:"handPoints"` // points for current hand
	TotalScore int      `json:"totalScore"` // cumulative score
	TricksWon  int      `json:"tricksWon"`
}

type HTTrick struct {
	Cards     []HTCard `json:"cards"`
	PlayerIDs []string `json:"playerIds"`
	LeadSuit  string   `json:"leadSuit"`
	WinnerID  string   `json:"winnerId"`
}

type HeartsState struct {
	ID               string        `json:"id"`
	Players          []HTPlayer    `json:"players"`
	Phase            HeartsPhase   `json:"phase"`
	CurrentPlayerIdx int           `json:"currentPlayerIdx"`
	DealerIdx        int           `json:"dealerIdx"`
	HandNumber       int           `json:"handNumber"`
	PassDirection    HeartsPassDir `json:"passDirection"`
	PassDirName      string        `json:"passDirName"`

	CurrentTrick    *HTTrick  `json:"currentTrick"`
	CompletedTricks []HTTrick `json:"completedTricks"`
	TricksPlayed    int       `json:"tricksPlayed"`

	HeartsBroken bool   `json:"heartsBroken"`
	LastAction   string `json:"lastAction"`
	Winner       string `json:"winner"` // player name with lowest score
	GameOverScores []HTScoreEntry `json:"gameOverScores"`
}

type HTScoreEntry struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Score      int    `json:"score"`
}

// Build standard 52-card deck
func htBuildDeck() []HTCard {
	suits := []string{"hearts", "diamonds", "clubs", "spades"}
	var deck []HTCard
	id := 0
	for _, suit := range suits {
		for rank := 2; rank <= 14; rank++ {
			deck = append(deck, HTCard{
				ID:   fmt.Sprintf("ht_%d", id),
				Rank: rank,
				Suit: suit,
			})
			id++
		}
	}
	return deck
}

func htShuffleDeck(deck []HTCard) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
}

func InitializeHeartsGame(players []struct{ ID, Name string }) *HeartsState {
	htPlayers := make([]HTPlayer, 4)
	for i := 0; i < 4; i++ {
		htPlayers[i] = HTPlayer{
			ID:   players[i].ID,
			Name: players[i].Name,
		}
	}

	state := &HeartsState{
		ID:        fmt.Sprintf("ht_%d", time.Now().UnixNano()),
		Players:   htPlayers,
		DealerIdx: 0,
		HandNumber: 0,
	}

	htStartNewHand(state)
	return state
}

func htStartNewHand(state *HeartsState) {
	state.HandNumber++
	state.PassDirection = HeartsPassDir((state.HandNumber - 1) % 4)
	state.PassDirName = htPassDirNames[state.PassDirection]

	deck := htBuildDeck()
	htShuffleDeck(deck)

	// Deal 13 cards to each player
	for i := 0; i < 4; i++ {
		cards := make([]HTCard, 13)
		copy(cards, deck[i*13:(i+1)*13])
		state.Players[i].Cards = cards
		state.Players[i].PassCards = nil
		state.Players[i].HasPassed = false
		state.Players[i].HandPoints = 0
		state.Players[i].TricksWon = 0
	}

	// Sort each player's hand
	for i := range state.Players {
		htSortHand(&state.Players[i])
	}

	state.CurrentTrick = nil
	state.CompletedTricks = nil
	state.TricksPlayed = 0
	state.HeartsBroken = false
	state.LastAction = fmt.Sprintf("Hand %d — Pass %s", state.HandNumber, state.PassDirName)

	if state.PassDirection == HT_PassNone {
		state.Phase = HT_PhasePlaying
		htStartPlaying(state)
	} else {
		state.Phase = HT_PhasePassing
	}
}

func htSortHand(p *HTPlayer) {
	suitOrder := map[string]int{"clubs": 0, "diamonds": 1, "spades": 2, "hearts": 3}
	sort.Slice(p.Cards, func(i, j int) bool {
		if suitOrder[p.Cards[i].Suit] != suitOrder[p.Cards[j].Suit] {
			return suitOrder[p.Cards[i].Suit] < suitOrder[p.Cards[j].Suit]
		}
		return p.Cards[i].Rank < p.Cards[j].Rank
	})
}

func htStartPlaying(state *HeartsState) {
	// Find player with 2 of clubs
	for i, p := range state.Players {
		for _, c := range p.Cards {
			if c.Suit == "clubs" && c.Rank == 2 {
				state.CurrentPlayerIdx = i
				state.CurrentTrick = &HTTrick{
					Cards:     []HTCard{},
					PlayerIDs: []string{},
				}
				state.LastAction = fmt.Sprintf("%s leads (has 2♣)", p.Name)
				return
			}
		}
	}
	// Fallback: player 0 leads
	state.CurrentPlayerIdx = 0
	state.CurrentTrick = &HTTrick{
		Cards:     []HTCard{},
		PlayerIDs: []string{},
	}
}

// HTPassCards - player selects 3 cards to pass
func HTPassCards(state *HeartsState, playerID string, cardIDs []string) error {
	if state.Phase != HT_PhasePassing {
		return fmt.Errorf("not in passing phase")
	}
	if len(cardIDs) != 3 {
		return fmt.Errorf("must pass exactly 3 cards")
	}

	pIdx := htFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if state.Players[pIdx].HasPassed {
		return fmt.Errorf("already passed cards")
	}

	// Validate cards exist in hand
	for _, cid := range cardIDs {
		found := false
		for _, c := range state.Players[pIdx].Cards {
			if c.ID == cid {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("card not found in your hand")
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, cid := range cardIDs {
		if seen[cid] {
			return fmt.Errorf("duplicate card selected")
		}
		seen[cid] = true
	}

	state.Players[pIdx].PassCards = cardIDs
	state.Players[pIdx].HasPassed = true
	state.LastAction = fmt.Sprintf("%s selected cards to pass", state.Players[pIdx].Name)

	// Check if all players have passed
	allPassed := true
	for _, p := range state.Players {
		if !p.HasPassed {
			allPassed = false
			break
		}
	}

	if allPassed {
		htExecutePass(state)
	}

	return nil
}

func htExecutePass(state *HeartsState) {
	// Extract cards to pass from each player
	passCards := make([][]HTCard, 4)
	for i := 0; i < 4; i++ {
		passCards[i] = make([]HTCard, 0, 3)
		remaining := make([]HTCard, 0)
		passSet := make(map[string]bool)
		for _, cid := range state.Players[i].PassCards {
			passSet[cid] = true
		}
		for _, c := range state.Players[i].Cards {
			if passSet[c.ID] {
				passCards[i] = append(passCards[i], c)
			} else {
				remaining = append(remaining, c)
			}
		}
		state.Players[i].Cards = remaining
	}

	// Execute the pass
	for i := 0; i < 4; i++ {
		var targetIdx int
		switch state.PassDirection {
		case HT_PassLeft:
			targetIdx = (i + 1) % 4
		case HT_PassRight:
			targetIdx = (i + 3) % 4
		case HT_PassAcross:
			targetIdx = (i + 2) % 4
		default:
			targetIdx = i // shouldn't happen
		}
		state.Players[targetIdx].Cards = append(state.Players[targetIdx].Cards, passCards[i]...)
	}

	// Sort hands and clear pass state
	for i := range state.Players {
		htSortHand(&state.Players[i])
		state.Players[i].PassCards = nil
		state.Players[i].HasPassed = false
	}

	state.Phase = HT_PhasePlaying
	state.LastAction = fmt.Sprintf("Cards passed %s", state.PassDirName)
	htStartPlaying(state)
}

// HTPlayCard - play a card to the current trick
func HTPlayCard(state *HeartsState, playerID string, cardID string) error {
	if state.Phase != HT_PhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	pIdx := htFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if pIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn")
	}

	// Find card
	cardIdx := -1
	for i, c := range state.Players[pIdx].Cards {
		if c.ID == cardID {
			cardIdx = i
			break
		}
	}
	if cardIdx < 0 {
		return fmt.Errorf("card not found in your hand")
	}

	card := state.Players[pIdx].Cards[cardIdx]
	isFirstTrick := state.TricksPlayed == 0

	// First card of first trick must be 2 of clubs
	if isFirstTrick && len(state.CurrentTrick.Cards) == 0 {
		if card.Suit != "clubs" || card.Rank != 2 {
			return fmt.Errorf("must lead with 2 of clubs")
		}
	}

	// Must follow suit
	if len(state.CurrentTrick.Cards) > 0 {
		leadSuit := state.CurrentTrick.LeadSuit
		hasSuit := false
		for _, c := range state.Players[pIdx].Cards {
			if c.Suit == leadSuit {
				hasSuit = true
				break
			}
		}
		if hasSuit && card.Suit != leadSuit {
			return fmt.Errorf("must follow suit (%s)", leadSuit)
		}

		// On first trick, cannot play hearts or Q♠ (unless only have penalty cards)
		if isFirstTrick && card.Points() > 0 {
			hasNonPenalty := false
			for _, c := range state.Players[pIdx].Cards {
				if c.Suit == leadSuit {
					if c.Points() == 0 {
						hasNonPenalty = true
						break
					}
				} else if !hasSuit && c.Points() == 0 {
					hasNonPenalty = true
					break
				}
			}
			if hasNonPenalty {
				return fmt.Errorf("cannot play hearts or Queen of Spades on the first trick")
			}
		}
	} else {
		// Leading
		state.CurrentTrick.LeadSuit = card.Suit

		// Cannot lead hearts until broken (unless only have hearts)
		if card.IsHeart() && !state.HeartsBroken {
			allHearts := true
			for _, c := range state.Players[pIdx].Cards {
				if !c.IsHeart() {
					allHearts = false
					break
				}
			}
			if !allHearts {
				return fmt.Errorf("hearts not broken yet")
			}
		}
	}

	// Remove card from hand
	state.Players[pIdx].Cards = append(state.Players[pIdx].Cards[:cardIdx], state.Players[pIdx].Cards[cardIdx+1:]...)

	// Add to trick
	state.CurrentTrick.Cards = append(state.CurrentTrick.Cards, card)
	state.CurrentTrick.PlayerIDs = append(state.CurrentTrick.PlayerIDs, playerID)

	// Track hearts broken
	if card.IsHeart() || card.IsQueenSpades() {
		state.HeartsBroken = true
	}

	state.LastAction = fmt.Sprintf("%s played %s", state.Players[pIdx].Name, card.Display())

	// If 4 cards played, evaluate trick
	if len(state.CurrentTrick.Cards) == 4 {
		htEvaluateTrick(state)
		return nil
	}

	state.CurrentPlayerIdx = (pIdx + 1) % 4
	return nil
}

func htEvaluateTrick(state *HeartsState) {
	trick := state.CurrentTrick
	leadSuit := trick.LeadSuit

	winnerIdx := 0
	for i := 1; i < 4; i++ {
		if trick.Cards[i].Suit == leadSuit && trick.Cards[i].Rank > trick.Cards[winnerIdx].Rank {
			winnerIdx = i
		} else if trick.Cards[i].Suit == leadSuit && trick.Cards[winnerIdx].Suit != leadSuit {
			winnerIdx = i
		}
	}

	winnerPlayerID := trick.PlayerIDs[winnerIdx]
	trick.WinnerID = winnerPlayerID

	// Count points in trick
	trickPoints := 0
	for _, c := range trick.Cards {
		trickPoints += c.Points()
	}

	winnerPIdx := htFindPlayer(state, winnerPlayerID)
	state.Players[winnerPIdx].HandPoints += trickPoints
	state.Players[winnerPIdx].TricksWon++

	state.CompletedTricks = append(state.CompletedTricks, *trick)
	state.TricksPlayed++

	state.LastAction = fmt.Sprintf("%s won the trick (+%d pts)", state.Players[winnerPIdx].Name, trickPoints)

	// All 13 tricks played?
	if state.TricksPlayed == 13 {
		htEndHand(state)
		return
	}

	// Winner leads next trick
	state.CurrentPlayerIdx = winnerPIdx
	state.CurrentTrick = &HTTrick{
		Cards:     []HTCard{},
		PlayerIDs: []string{},
	}
}

func htEndHand(state *HeartsState) {
	// Check for shooting the moon
	moonShooter := -1
	for i, p := range state.Players {
		if p.HandPoints == 26 {
			moonShooter = i
			break
		}
	}

	if moonShooter >= 0 {
		// Shooter gets 0, everyone else gets +26
		state.Players[moonShooter].HandPoints = 0
		for i := range state.Players {
			if i != moonShooter {
				state.Players[i].HandPoints = 26
			}
		}
		state.LastAction = fmt.Sprintf("🌙 %s shot the moon! Everyone else gets +26!", state.Players[moonShooter].Name)
	}

	// Add hand points to total
	for i := range state.Players {
		state.Players[i].TotalScore += state.Players[i].HandPoints
	}

	// Check for game over (anyone ≥ 100)
	gameOver := false
	for _, p := range state.Players {
		if p.TotalScore >= 100 {
			gameOver = true
			break
		}
	}

	if gameOver {
		state.Phase = HT_PhaseGameOver
		// Find winner (lowest score)
		lowestScore := state.Players[0].TotalScore
		winnerName := state.Players[0].Name
		for _, p := range state.Players {
			if p.TotalScore < lowestScore {
				lowestScore = p.TotalScore
				winnerName = p.Name
			}
		}
		state.Winner = winnerName

		scores := make([]HTScoreEntry, 4)
		for i, p := range state.Players {
			scores[i] = HTScoreEntry{PlayerID: p.ID, PlayerName: p.Name, Score: p.TotalScore}
		}
		sort.Slice(scores, func(i, j int) bool { return scores[i].Score < scores[j].Score })
		state.GameOverScores = scores
		state.LastAction = fmt.Sprintf("🏆 %s wins with %d points!", winnerName, lowestScore)
	} else {
		state.Phase = HT_PhaseHandOver
	}
}

// HTNextHand starts the next hand
func HTNextHand(state *HeartsState) {
	if state.Phase != HT_PhaseHandOver {
		return
	}
	state.DealerIdx = (state.DealerIdx + 1) % 4
	htStartNewHand(state)
}

// SanitizeHTStateForPlayer hides other players' cards
func SanitizeHTStateForPlayer(state *HeartsState, viewerID string) *HeartsState {
	sanitized := *state
	sanitized.Players = make([]HTPlayer, len(state.Players))
	for i, p := range state.Players {
		sanitized.Players[i] = p
		if p.ID != viewerID {
			hidden := make([]HTCard, len(p.Cards))
			for j := range p.Cards {
				hidden[j] = HTCard{ID: fmt.Sprintf("hidden_%d_%d", i, j)}
			}
			sanitized.Players[i].Cards = hidden
			// Hide pass card selections
			sanitized.Players[i].PassCards = nil
		}
	}
	return &sanitized
}

func htFindPlayer(state *HeartsState, playerID string) int {
	for i, p := range state.Players {
		if p.ID == playerID {
			return i
		}
	}
	return -1
}

// HTVoluntaryExit handles a player leaving
func HTVoluntaryExit(state *HeartsState, playerID string) {
	pIdx := htFindPlayer(state, playerID)
	if pIdx < 0 {
		return
	}
	state.Phase = HT_PhaseGameOver
	// Find player with lowest score as winner
	lowestScore := 999
	winnerName := ""
	for i, p := range state.Players {
		if i != pIdx && p.TotalScore < lowestScore {
			lowestScore = p.TotalScore
			winnerName = p.Name
		}
	}
	state.Winner = winnerName
	state.LastAction = fmt.Sprintf("%s left the game. %s wins!", state.Players[pIdx].Name, winnerName)
}
