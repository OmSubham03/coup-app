package game

import (
	"fmt"
	"math/rand"
	"time"
)

// Phases
type TwentyNinePhase string

const (
	TN_PhaseWaiting      TwentyNinePhase = "waiting"
	TN_PhaseBidding      TwentyNinePhase = "bidding"
	TN_PhaseTrumpSelect  TwentyNinePhase = "trump_select"
	TN_PhasePlaying      TwentyNinePhase = "playing"
	TN_PhaseTrumpReveal  TwentyNinePhase = "trump_reveal"
	TN_PhaseRoundOver    TwentyNinePhase = "round_over"
	TN_PhaseGameOver     TwentyNinePhase = "game_over"
)

// Card suits
const (
	TN_Hearts   = "hearts"
	TN_Diamonds = "diamonds"
	TN_Clubs    = "clubs"
	TN_Spades   = "spades"
)

// Card ranks - index in ranking order (0=highest)
// J(3), 9(2), A(1), 10(1), K(0), Q(0), 8(0), 7(0)
var tnRankOrder = map[string]int{
	"J": 0, "9": 1, "A": 2, "10": 3, "K": 4, "Q": 5, "8": 6, "7": 7,
}

var tnRankPoints = map[string]int{
	"J": 3, "9": 2, "A": 1, "10": 1, "K": 0, "Q": 0, "8": 0, "7": 0,
}

var tnAllRanks = []string{"J", "9", "A", "10", "K", "Q", "8", "7"}
var tnAllSuits = []string{TN_Hearts, TN_Diamonds, TN_Clubs, TN_Spades}

type TNCard struct {
	ID   string `json:"id"`
	Rank string `json:"rank"`
	Suit string `json:"suit"`
}

func (c TNCard) Points() int {
	return tnRankPoints[c.Rank]
}

func (c TNCard) RankOrder() int {
	return tnRankOrder[c.Rank]
}

func (c TNCard) Display() string {
	suits := map[string]string{TN_Hearts: "♥", TN_Diamonds: "♦", TN_Clubs: "♣", TN_Spades: "♠"}
	return c.Rank + suits[c.Suit]
}

type TNPlayer struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Cards       []TNCard `json:"cards"`
	Team        int      `json:"team"`        // 0 or 1 (players 0,2 = team 0; players 1,3 = team 1)
	TricksWon   int      `json:"tricksWon"`   // tricks won this round
	PointsWon   int      `json:"pointsWon"`   // card points won this round
}

type TNTrick struct {
	Cards     []TNCard `json:"cards"`     // cards played (indexed by seat position in play order)
	PlayerIDs []string `json:"playerIds"` // who played which card
	LeadSuit  string   `json:"leadSuit"`
	WinnerID  string   `json:"winnerId"`
}

type TwentyNineState struct {
	ID               string          `json:"id"`
	Players          []TNPlayer      `json:"players"`
	Phase            TwentyNinePhase `json:"phase"`
	DealerIdx        int             `json:"dealerIdx"`
	CurrentPlayerIdx int             `json:"currentPlayerIdx"`

	// Bidding
	CurrentBid    int    `json:"currentBid"`
	BidderIdx     int    `json:"bidderIdx"`
	BidPassCount  int    `json:"bidPassCount"`    // consecutive passes
	BidderPlayerID string `json:"bidderPlayerId"` // who won the bid
	MinBid        int    `json:"minBid"`

	// Trump
	TrumpSuit     string `json:"trumpSuit"`     // hidden until revealed
	TrumpRevealed bool   `json:"trumpRevealed"`
	TrumpCardID   string `json:"trumpCardId"`   // the hidden trump indicator card

	// Tricks
	CurrentTrick  *TNTrick   `json:"currentTrick"`
	TricksPlayed  int        `json:"tricksPlayed"`
	CompletedTricks []TNTrick `json:"completedTricks"`

	// Scoring
	TeamPoints    [2]int `json:"teamPoints"`    // card points won per team this round
	TeamScore     [2]int `json:"teamScore"`     // game points (-6 to +6)
	BiddingTeam   int    `json:"biddingTeam"`   // which team bid
	TargetPoints  int    `json:"targetPoints"`  // points needed by bidding team

	// Pair (K+Q of trump)
	PairDeclared    bool   `json:"pairDeclared"`
	PairDeclaredBy  int    `json:"pairDeclaredBy"` // team that declared
	PairPlayerName  string `json:"pairPlayerName"`

	// Round info
	Round         int    `json:"round"`
	Winner        string `json:"winner"` // team name
	LastAction    string `json:"lastAction"`
	RoundResult   string `json:"roundResult"`

	// Hands won display
	TeamHands     [2]int `json:"teamHands"` // tricks won per team this round
}

// Build a 32-card deck
func tnBuildDeck() []TNCard {
	var deck []TNCard
	id := 0
	for _, suit := range tnAllSuits {
		for _, rank := range tnAllRanks {
			deck = append(deck, TNCard{
				ID:   fmt.Sprintf("tn_%d", id),
				Rank: rank,
				Suit: suit,
			})
			id++
		}
	}
	return deck
}

func tnShuffleDeck(deck []TNCard) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
}

// Initialize game with exactly 4 players
func InitializeTwentyNineGame(players []struct{ ID, Name string }) *TwentyNineState {
	tnPlayers := make([]TNPlayer, 4)
	for i := 0; i < 4; i++ {
		tnPlayers[i] = TNPlayer{
			ID:   players[i].ID,
			Name: players[i].Name,
			Team: i % 2, // 0,1,0,1 -> players 0,2 are team 0; 1,3 are team 1
		}
	}

	state := &TwentyNineState{
		ID:        fmt.Sprintf("tn_%d", time.Now().UnixNano()),
		Players:   tnPlayers,
		Phase:     TN_PhaseBidding,
		DealerIdx: 0,
		Round:     1,
		MinBid:    15,
	}

	// Deal first 4 cards to each player
	tnDealFirstFour(state)

	// Bidding starts from player to dealer's left (clockwise)
	state.CurrentPlayerIdx = (state.DealerIdx + 1) % 4
	state.BidderIdx = state.CurrentPlayerIdx
	state.CurrentBid = 0
	state.BidPassCount = 0

	return state
}

func tnDealFirstFour(state *TwentyNineState) {
	deck := tnBuildDeck()
	tnShuffleDeck(deck)

	// Deal 4 cards to each player (copy, not slice reference)
	for i := 0; i < 4; i++ {
		cards := make([]TNCard, 4)
		copy(cards, deck[i*4:(i+1)*4])
		state.Players[i].Cards = cards
		state.Players[i].TricksWon = 0
		state.Players[i].PointsWon = 0
	}

	// Store remaining 16 cards for second deal (copy)
	remaining := make([]TNCard, 16)
	copy(remaining, deck[16:])
	state.CurrentTrick = &TNTrick{
		Cards: remaining,
	}
	state.TeamPoints = [2]int{0, 0}
	state.TeamHands = [2]int{0, 0}
	state.TricksPlayed = 0
	state.CompletedTricks = nil
	state.TrumpRevealed = false
	state.TrumpSuit = ""
	state.TrumpCardID = ""
	state.PairDeclared = false
	state.PairDeclaredBy = 0
	state.PairPlayerName = ""
	state.RoundResult = ""
}

// TNBid handles a player making a bid
func TNBid(state *TwentyNineState, playerID string, bidValue int) error {
	if state.Phase != TN_PhaseBidding {
		return fmt.Errorf("not in bidding phase")
	}
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if pIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn to bid")
	}
	if bidValue < state.MinBid || bidValue > 28 {
		return fmt.Errorf("bid must be between %d and 28", state.MinBid)
	}
	if state.CurrentBid > 0 && bidValue <= state.CurrentBid {
		return fmt.Errorf("bid must be higher than %d", state.CurrentBid)
	}

	state.CurrentBid = bidValue
	state.BidderIdx = pIdx
	state.BidderPlayerID = playerID
	state.BidPassCount = 0
	state.LastAction = fmt.Sprintf("%s bid %d", state.Players[pIdx].Name, bidValue)

	// If bid is 28, bidding ends immediately
	if bidValue == 28 {
		state.MinBid = bidValue
		tnEndBidding(state)
		return nil
	}

	state.MinBid = bidValue + 1
	state.CurrentPlayerIdx = (pIdx + 1) % 4
	return nil
}

// TNPass handles a player passing on the bid
func TNPass(state *TwentyNineState, playerID string) error {
	if state.Phase != TN_PhaseBidding {
		return fmt.Errorf("not in bidding phase")
	}
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if pIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn")
	}

	// First bidder (dealer's left) must bid at least 15 if no one has bid
	if state.CurrentBid == 0 && pIdx == (state.DealerIdx+1)%4 {
		return fmt.Errorf("you must bid at least 15")
	}

	state.BidPassCount++
	state.LastAction = fmt.Sprintf("%s passed", state.Players[pIdx].Name)

	// If 3 consecutive passes after a bid, bidding ends
	if state.BidPassCount >= 3 && state.CurrentBid > 0 {
		tnEndBidding(state)
		return nil
	}

	// If first 3 players pass (no bid yet), dealer must bid 15
	if state.CurrentBid == 0 {
		nextIdx := (pIdx + 1) % 4
		if nextIdx == state.DealerIdx {
			// Dealer is forced to bid 15
			state.CurrentBid = 15
			state.BidderIdx = state.DealerIdx
			state.BidderPlayerID = state.Players[state.DealerIdx].ID
			state.MinBid = 15
			state.LastAction = fmt.Sprintf("%s forced bid 15", state.Players[state.DealerIdx].Name)
			tnEndBidding(state)
			return nil
		}
	}

	state.CurrentPlayerIdx = (pIdx + 1) % 4
	// Skip the bidder (they already won)
	if state.CurrentBid > 0 && state.CurrentPlayerIdx == state.BidderIdx {
		state.CurrentPlayerIdx = (state.CurrentPlayerIdx + 1) % 4
	}
	return nil
}

func tnEndBidding(state *TwentyNineState) {
	state.Phase = TN_PhaseTrumpSelect
	state.CurrentPlayerIdx = state.BidderIdx
	state.BiddingTeam = state.Players[state.BidderIdx].Team
	state.TargetPoints = state.CurrentBid
}

// TNSelectTrump - bidder selects trump suit
func TNSelectTrump(state *TwentyNineState, playerID string, suit string) error {
	if state.Phase != TN_PhaseTrumpSelect {
		return fmt.Errorf("not in trump selection phase")
	}
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 || pIdx != state.BidderIdx {
		return fmt.Errorf("only the bidder can select trump")
	}

	validSuit := false
	for _, s := range tnAllSuits {
		if s == suit {
			validSuit = true
			break
		}
	}
	if !validSuit {
		return fmt.Errorf("invalid suit")
	}

	// Verify bidder has at least one card of this suit
	hasCard := false
	for _, c := range state.Players[pIdx].Cards {
		if c.Suit == suit {
			hasCard = true
			state.TrumpCardID = c.ID // use first card of that suit as indicator
			break
		}
	}
	if !hasCard {
		return fmt.Errorf("you must have at least one card of the chosen trump suit")
	}

	state.TrumpSuit = suit
	state.TrumpRevealed = false
	state.LastAction = fmt.Sprintf("%s selected trump (hidden)", state.Players[pIdx].Name)

	// Deal remaining 4 cards to each player
	tnDealSecondFour(state)

	// Start play: player to dealer's left leads
	state.Phase = TN_PhasePlaying
	state.CurrentPlayerIdx = (state.DealerIdx + 1) % 4
	state.CurrentTrick = &TNTrick{
		Cards:     []TNCard{},
		PlayerIDs: []string{},
	}

	return nil
}

func tnDealSecondFour(state *TwentyNineState) {
	// The remaining 16 cards were stored in CurrentTrick.Cards
	remaining := state.CurrentTrick.Cards
	for i := 0; i < 4; i++ {
		cards := make([]TNCard, 4)
		copy(cards, remaining[i*4:(i+1)*4])
		state.Players[i].Cards = append(state.Players[i].Cards, cards...)
	}
}

// TNPlayCard - player plays a card to the current trick
func TNPlayCard(state *TwentyNineState, playerID string, cardID string) error {
	if state.Phase != TN_PhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if pIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn")
	}

	// Find card in hand
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

	// Validate play
	if len(state.CurrentTrick.Cards) > 0 {
		leadSuit := state.CurrentTrick.LeadSuit
		// Must follow suit if possible
		hasSuit := false
		for _, c := range state.Players[pIdx].Cards {
			if c.Suit == leadSuit {
				hasSuit = true
				break
			}
		}
		if hasSuit && card.Suit != leadSuit {
			return fmt.Errorf("you must follow suit (%s)", leadSuit)
		}

		// If trump is revealed and player can't follow suit but plays a lower trump
		// when a higher trump is already played, check overtrump rule
		if state.TrumpRevealed && !hasSuit && card.Suit == state.TrumpSuit {
			// Find highest trump in current trick
			highestTrumpOrder := 8 // worse than any card
			for _, tc := range state.CurrentTrick.Cards {
				if tc.Suit == state.TrumpSuit && tc.RankOrder() < highestTrumpOrder {
					highestTrumpOrder = tc.RankOrder()
				}
			}
			if highestTrumpOrder < 8 && card.RankOrder() > highestTrumpOrder {
				// Check if player has a higher trump
				hasHigherTrump := false
				for _, c := range state.Players[pIdx].Cards {
					if c.Suit == state.TrumpSuit && c.RankOrder() < highestTrumpOrder {
						hasHigherTrump = true
						break
					}
				}
				if hasHigherTrump {
					return fmt.Errorf("you must play a higher trump or a non-trump card")
				}
			}
		}
	} else {
		// Leading the trick
		state.CurrentTrick.LeadSuit = card.Suit
	}

	// Remove card from hand
	state.Players[pIdx].Cards = append(state.Players[pIdx].Cards[:cardIdx], state.Players[pIdx].Cards[cardIdx+1:]...)

	// Add to trick
	state.CurrentTrick.Cards = append(state.CurrentTrick.Cards, card)
	state.CurrentTrick.PlayerIDs = append(state.CurrentTrick.PlayerIDs, playerID)

	state.LastAction = fmt.Sprintf("%s played %s", state.Players[pIdx].Name, card.Display())

	// If 4 cards played, evaluate trick
	if len(state.CurrentTrick.Cards) == 4 {
		tnEvaluateTrick(state)
		return nil
	}

	state.CurrentPlayerIdx = (pIdx + 1) % 4
	return nil
}

// TNRevealTrump - called when a player can't follow suit and asks for trump reveal
func TNRevealTrump(state *TwentyNineState, playerID string) error {
	if state.Phase != TN_PhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	if state.TrumpRevealed {
		return fmt.Errorf("trump already revealed")
	}
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}
	if pIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn")
	}

	// Player must be unable to follow suit
	if len(state.CurrentTrick.Cards) > 0 {
		leadSuit := state.CurrentTrick.LeadSuit
		for _, c := range state.Players[pIdx].Cards {
			if c.Suit == leadSuit {
				return fmt.Errorf("you have cards of the lead suit, cannot ask for trump reveal")
			}
		}
	}

	state.TrumpRevealed = true
	state.LastAction = fmt.Sprintf("%s asked for trump reveal: %s!", state.Players[pIdx].Name, tnSuitSymbol(state.TrumpSuit))

	return nil
}

func tnSuitSymbol(suit string) string {
	symbols := map[string]string{TN_Hearts: "♥ Hearts", TN_Diamonds: "♦ Diamonds", TN_Clubs: "♣ Clubs", TN_Spades: "♠ Spades"}
	return symbols[suit]
}

func tnEvaluateTrick(state *TwentyNineState) {
	trick := state.CurrentTrick
	leadSuit := trick.LeadSuit

	winnerIdx := 0
	winnerCard := trick.Cards[0]

	for i := 1; i < 4; i++ {
		card := trick.Cards[i]
		if tnCardBeats(card, winnerCard, leadSuit, state.TrumpSuit, state.TrumpRevealed) {
			winnerIdx = i
			winnerCard = card
		}
	}

	winnerPlayerID := trick.PlayerIDs[winnerIdx]
	trick.WinnerID = winnerPlayerID

	// Calculate points in this trick
	trickPoints := 0
	for _, c := range trick.Cards {
		trickPoints += c.Points()
	}

	// Find winner player index
	winnerPIdx := tnFindPlayer(state, winnerPlayerID)
	state.Players[winnerPIdx].TricksWon++
	state.Players[winnerPIdx].PointsWon += trickPoints
	winnerTeam := state.Players[winnerPIdx].Team
	state.TeamPoints[winnerTeam] += trickPoints
	state.TeamHands[winnerTeam]++

	state.CompletedTricks = append(state.CompletedTricks, *trick)
	state.TricksPlayed++

	state.LastAction = fmt.Sprintf("%s won the trick (+%d pts)", state.Players[winnerPIdx].Name, trickPoints)

	// Last trick bonus point
	if state.TricksPlayed == 8 {
		state.TeamPoints[winnerTeam]++ // +1 for last trick (total 29)
		tnEndRound(state)
		return
	}

	// If trump not revealed by trick 7, the hand is annulled
	if state.TricksPlayed == 7 && !state.TrumpRevealed {
		state.RoundResult = "Hand annulled - trump was never revealed"
		state.Phase = TN_PhaseRoundOver
		state.LastAction = "Hand annulled - trump was never revealed"
		return
	}

	// Start new trick - winner leads
	state.CurrentPlayerIdx = winnerPIdx
	state.CurrentTrick = &TNTrick{
		Cards:     []TNCard{},
		PlayerIDs: []string{},
	}
}

func tnCardBeats(card, current TNCard, leadSuit, trumpSuit string, trumpRevealed bool) bool {
	// If trump is revealed
	if trumpRevealed {
		// Trump beats non-trump
		if card.Suit == trumpSuit && current.Suit != trumpSuit {
			return true
		}
		if card.Suit != trumpSuit && current.Suit == trumpSuit {
			return false
		}
		// Both trump or both non-trump
		if card.Suit == trumpSuit && current.Suit == trumpSuit {
			return card.RankOrder() < current.RankOrder()
		}
	}
	// Same suit: higher rank wins
	if card.Suit == current.Suit {
		return card.RankOrder() < current.RankOrder()
	}
	// Different suit (non-trump): can't beat
	// Card of lead suit beats non-lead suit
	if current.Suit == leadSuit && card.Suit != leadSuit {
		return false
	}
	if card.Suit == leadSuit && current.Suit != leadSuit {
		return true
	}
	return false
}

func tnEndRound(state *TwentyNineState) {
	state.Phase = TN_PhaseRoundOver

	biddingTeam := state.BiddingTeam
	targetPoints := state.TargetPoints

	// Adjust for pair
	if state.PairDeclared {
		if state.PairDeclaredBy == biddingTeam {
			targetPoints -= 4
			if targetPoints < 15 {
				targetPoints = 15
			}
		} else {
			targetPoints += 4
			if targetPoints > 28 {
				targetPoints = 28
			}
		}
	}

	biddingTeamPoints := state.TeamPoints[biddingTeam]

	teamNames := [2]string{
		fmt.Sprintf("%s & %s", state.Players[0].Name, state.Players[2].Name),
		fmt.Sprintf("%s & %s", state.Players[1].Name, state.Players[3].Name),
	}

	if biddingTeamPoints >= targetPoints {
		// Bidding team wins
		state.TeamScore[biddingTeam]++
		state.RoundResult = fmt.Sprintf("%s won! (%d/%d points)", teamNames[biddingTeam], biddingTeamPoints, targetPoints)
	} else {
		// Bidding team loses
		state.TeamScore[biddingTeam]--
		// Check under-half: if less than half of bid, lose 2
		if biddingTeamPoints < state.CurrentBid/2 {
			state.TeamScore[biddingTeam]-- // extra penalty
			state.RoundResult = fmt.Sprintf("%s lost badly! Only %d/%d points (under half - double penalty!)", teamNames[biddingTeam], biddingTeamPoints, targetPoints)
		} else {
			state.RoundResult = fmt.Sprintf("%s lost. Only %d/%d points", teamNames[biddingTeam], biddingTeamPoints, targetPoints)
		}
	}

	state.LastAction = state.RoundResult

	// Check for game over (±6)
	for t := 0; t < 2; t++ {
		if state.TeamScore[t] >= 6 {
			state.Phase = TN_PhaseGameOver
			state.Winner = teamNames[t]
			state.LastAction = fmt.Sprintf("%s wins the game! (Score: %d)", teamNames[t], state.TeamScore[t])
			return
		}
		if state.TeamScore[t] <= -6 {
			otherTeam := 1 - t
			state.Phase = TN_PhaseGameOver
			state.Winner = teamNames[otherTeam]
			state.LastAction = fmt.Sprintf("%s wins the game! (%s reached -6)", teamNames[otherTeam], teamNames[t])
			return
		}
	}
}

// TNNextRound starts a new round
func TNNextRound(state *TwentyNineState) {
	if state.Phase != TN_PhaseRoundOver {
		return
	}

	state.Round++
	state.DealerIdx = (state.DealerIdx + 1) % 4
	state.Phase = TN_PhaseBidding
	state.CurrentBid = 0
	state.BidPassCount = 0
	state.MinBid = 15
	state.BidderPlayerID = ""

	// Deal first 4 cards
	tnDealFirstFour(state)

	state.CurrentPlayerIdx = (state.DealerIdx + 1) % 4
	state.BidderIdx = state.CurrentPlayerIdx
	state.LastAction = fmt.Sprintf("Round %d - Dealing...", state.Round)
}

// TNDeclarePair - declare King+Queen of trump pair
func TNDeclarePair(state *TwentyNineState, playerID string) error {
	if state.Phase != TN_PhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	if !state.TrumpRevealed {
		return fmt.Errorf("trump must be revealed first")
	}
	if state.PairDeclared {
		return fmt.Errorf("pair already declared")
	}

	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return fmt.Errorf("player not found")
	}

	// Check that player holds both K and Q of trump
	hasKing := false
	hasQueen := false
	for _, c := range state.Players[pIdx].Cards {
		if c.Suit == state.TrumpSuit {
			if c.Rank == "K" {
				hasKing = true
			}
			if c.Rank == "Q" {
				hasQueen = true
			}
		}
	}

	if !hasKing || !hasQueen {
		return fmt.Errorf("you must hold both King and Queen of trump")
	}

	// Check that the declaring player's team has won at least one trick after trump was revealed
	team := state.Players[pIdx].Team
	wonTrickAfterReveal := false
	for _, t := range state.CompletedTricks {
		if t.WinnerID != "" {
			wIdx := tnFindPlayer(state, t.WinnerID)
			if wIdx >= 0 && state.Players[wIdx].Team == team {
				wonTrickAfterReveal = true
				break
			}
		}
	}
	if !wonTrickAfterReveal {
		return fmt.Errorf("your team must win a trick after trump is revealed before declaring pair")
	}

	state.PairDeclared = true
	state.PairDeclaredBy = team
	state.PairPlayerName = state.Players[pIdx].Name
	state.LastAction = fmt.Sprintf("%s declared Pair (K+Q of %s)!", state.Players[pIdx].Name, tnSuitSymbol(state.TrumpSuit))

	return nil
}

// SanitizeTNStateForPlayer hides other players' cards and trump suit
func SanitizeTNStateForPlayer(state *TwentyNineState, viewerID string) *TwentyNineState {
	// Deep copy
	sanitized := *state
	sanitized.Players = make([]TNPlayer, len(state.Players))
	for i, p := range state.Players {
		sanitized.Players[i] = p
		if p.ID != viewerID {
			// Hide other players' cards
			hidden := make([]TNCard, len(p.Cards))
			for j := range p.Cards {
				hidden[j] = TNCard{ID: fmt.Sprintf("hidden_%d_%d", i, j)}
			}
			sanitized.Players[i].Cards = hidden
		}
	}

	// Hide trump suit from non-bidder until revealed
	if !state.TrumpRevealed {
		viewerIdx := tnFindPlayer(state, viewerID)
		if viewerIdx < 0 || viewerIdx != state.BidderIdx {
			sanitized.TrumpSuit = ""
			sanitized.TrumpCardID = ""
		}
	}

	return &sanitized
}

func tnFindPlayer(state *TwentyNineState, playerID string) int {
	for i, p := range state.Players {
		if p.ID == playerID {
			return i
		}
	}
	return -1
}

// TNVoluntaryExit handles a player leaving mid-game
func TNVoluntaryExit(state *TwentyNineState, playerID string) {
	pIdx := tnFindPlayer(state, playerID)
	if pIdx < 0 {
		return
	}

	// End the game, the team with the departed player loses
	losingTeam := state.Players[pIdx].Team
	winningTeam := 1 - losingTeam
	teamNames := [2]string{
		fmt.Sprintf("%s & %s", state.Players[0].Name, state.Players[2].Name),
		fmt.Sprintf("%s & %s", state.Players[1].Name, state.Players[3].Name),
	}

	state.Phase = TN_PhaseGameOver
	state.Winner = teamNames[winningTeam]
	state.LastAction = fmt.Sprintf("%s left the game. %s wins!", state.Players[pIdx].Name, teamNames[winningTeam])
}
