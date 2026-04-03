package game

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Suit and Rank for standard 52-card deck
type Suit string
type Rank int

const (
	SuitHearts   Suit = "hearts"
	SuitDiamonds Suit = "diamonds"
	SuitClubs    Suit = "clubs"
	SuitSpades   Suit = "spades"
)

const (
	RankTwo   Rank = 2
	RankThree Rank = 3
	RankFour  Rank = 4
	RankFive  Rank = 5
	RankSix   Rank = 6
	RankSeven Rank = 7
	RankEight Rank = 8
	RankNine  Rank = 9
	RankTen   Rank = 10
	RankJack  Rank = 11
	RankQueen Rank = 12
	RankKing  Rank = 13
	RankAce   Rank = 14
)

func (r Rank) String() string {
	switch r {
	case RankTwo:
		return "2"
	case RankThree:
		return "3"
	case RankFour:
		return "4"
	case RankFive:
		return "5"
	case RankSix:
		return "6"
	case RankSeven:
		return "7"
	case RankEight:
		return "8"
	case RankNine:
		return "9"
	case RankTen:
		return "10"
	case RankJack:
		return "J"
	case RankQueen:
		return "Q"
	case RankKing:
		return "K"
	case RankAce:
		return "A"
	}
	return "?"
}

func (s Suit) Symbol() string {
	switch s {
	case SuitHearts:
		return "♥"
	case SuitDiamonds:
		return "♦"
	case SuitClubs:
		return "♣"
	case SuitSpades:
		return "♠"
	}
	return "?"
}

// PokerCard represents a playing card
type PokerCard struct {
	ID   string `json:"id"`
	Suit Suit   `json:"suit"`
	Rank Rank   `json:"rank"`
}

func (c PokerCard) String() string {
	return c.Rank.String() + c.Suit.Symbol()
}

// PokerPhase defines game phases
type PokerPhase string

const (
	PokerPhaseWaiting  PokerPhase = "waiting"
	PokerPhasePreflop  PokerPhase = "preflop"
	PokerPhaseFlop     PokerPhase = "flop"
	PokerPhaseTurn     PokerPhase = "turn"
	PokerPhaseRiver    PokerPhase = "river"
	PokerPhaseShowdown PokerPhase = "showdown"
	PokerPhaseGameOver PokerPhase = "game_over"
)

// HandRank represents poker hand rankings
type HandRank int

const (
	HandHighCard      HandRank = 0
	HandOnePair       HandRank = 1
	HandTwoPair       HandRank = 2
	HandThreeOfAKind  HandRank = 3
	HandStraight      HandRank = 4
	HandFlush         HandRank = 5
	HandFullHouse     HandRank = 6
	HandFourOfAKind   HandRank = 7
	HandStraightFlush HandRank = 8
	HandRoyalFlush    HandRank = 9
)

func (h HandRank) String() string {
	names := []string{"High Card", "One Pair", "Two Pair", "Three of a Kind", "Straight", "Flush", "Full House", "Four of a Kind", "Straight Flush", "Royal Flush"}
	if int(h) < len(names) {
		return names[h]
	}
	return "Unknown"
}

// EvaluatedHand represents the result of hand evaluation
type EvaluatedHand struct {
	Rank     HandRank `json:"rank"`
	RankName string   `json:"rankName"`
	Kickers  []Rank   `json:"-"`       // For tie-breaking
	BestFive []PokerCard `json:"bestFive"` // The 5 cards forming the hand
}

// PokerPlayer represents a player in the poker game
type PokerPlayer struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Chips    int         `json:"chips"`
	HoleCards []PokerCard `json:"holeCards"`
	Folded   bool        `json:"folded"`
	AllIn    bool        `json:"allIn"`
	CurrentBet int       `json:"currentBet"` // Bet in current round
	TotalBet   int       `json:"totalBet"`  // Total bet this hand
	IsActive bool        `json:"isActive"`   // Still in the game (has chips or is all-in in current hand)
	Hand     *EvaluatedHand `json:"hand,omitempty"` // Filled at showdown
	RoundActed bool      `json:"-"`          // Has acted in current betting round
}

// Pot for side-pot calculation
type Pot struct {
	Amount    int      `json:"amount"`
	Eligible  []string `json:"eligible"` // Player IDs eligible
}

// PokerLogEntry is a log entry
type PokerLogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Hand      int    `json:"hand"`
}

// PokerScoreEntry tracks cumulative results
type PokerScoreEntry struct {
	PlayerID string `json:"playerId"`
	Name     string `json:"name"`
	NetGain  int    `json:"netGain"` // chips gained/lost relative to buy-in
	FinalChips int  `json:"finalChips"`
}

// PokerState holds the full poker game state
type PokerState struct {
	ID               string         `json:"id"`
	Players          []PokerPlayer  `json:"players"`
	CommunityCards   []PokerCard    `json:"communityCards"`
	Deck             []PokerCard    `json:"-"` // Hidden from client
	Phase            PokerPhase     `json:"phase"`
	Pots             []Pot          `json:"pots"`
	CurrentBet       int            `json:"currentBet"`    // Highest bet in current round
	MinRaise         int            `json:"minRaise"`      // Minimum raise amount
	DealerIndex      int            `json:"dealerIndex"`
	CurrentPlayerIdx int            `json:"currentPlayerIndex"`
	LastRaiserIdx    int            `json:"-"`             // Who last raised (betting ends when it comes back to them)
	SmallBlind       int            `json:"smallBlind"`
	BigBlind         int            `json:"bigBlind"`
	BuyIn            int            `json:"buyIn"`
	HandNumber       int            `json:"handNumber"`
	Log              []PokerLogEntry `json:"log"`
	LastAction       string         `json:"lastAction"`
	Scoreboard       []PokerScoreEntry `json:"scoreboard,omitempty"`
	Winner           string         `json:"winner,omitempty"` // Overall game winner
}

// InitializePokerGame creates a new poker game
func InitializePokerGame(players []struct{ ID, Name string }, buyIn, smallBlind int) *PokerState {
	bigBlind := smallBlind * 2

	pokerPlayers := make([]PokerPlayer, len(players))
	for i, p := range players {
		pokerPlayers[i] = PokerPlayer{
			ID:       p.ID,
			Name:     p.Name,
			Chips:    buyIn,
			IsActive: true,
		}
	}

	// Shuffle player order
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(pokerPlayers), func(i, j int) {
		pokerPlayers[i], pokerPlayers[j] = pokerPlayers[j], pokerPlayers[i]
	})

	state := &PokerState{
		ID:          uuid.New().String(),
		Players:     pokerPlayers,
		Phase:       PokerPhaseWaiting,
		SmallBlind:  smallBlind,
		BigBlind:    bigBlind,
		BuyIn:       buyIn,
		HandNumber:  0,
		DealerIndex: 0,
	}

	startNewHand(state)
	return state
}

func createPokerDeck() []PokerCard {
	suits := []Suit{SuitHearts, SuitDiamonds, SuitClubs, SuitSpades}
	var deck []PokerCard
	for _, s := range suits {
		for r := RankTwo; r <= RankAce; r++ {
			deck = append(deck, PokerCard{
				ID:   uuid.New().String(),
				Suit: s,
				Rank: r,
			})
		}
	}
	// Shuffle
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func startNewHand(state *PokerState) {
	state.HandNumber++
	state.Deck = createPokerDeck()
	state.CommunityCards = []PokerCard{}
	state.CurrentBet = 0
	state.MinRaise = state.BigBlind
	state.Pots = []Pot{{Amount: 0, Eligible: nil}}
	state.LastAction = ""

	// Count active players
	activePlayers := 0
	for i := range state.Players {
		if state.Players[i].IsActive && state.Players[i].Chips > 0 {
			activePlayers++
		}
	}

	if activePlayers < 2 {
		// Game over - find winner
		state.Phase = PokerPhaseGameOver
		finalizeScoreboard(state)
		return
	}

	// Reset player hand state
	for i := range state.Players {
		state.Players[i].HoleCards = nil
		state.Players[i].Folded = false
		state.Players[i].AllIn = false
		state.Players[i].CurrentBet = 0
		state.Players[i].TotalBet = 0
		state.Players[i].Hand = nil
		state.Players[i].RoundActed = false
		if state.Players[i].Chips <= 0 {
			state.Players[i].IsActive = false
		}
	}

	// Move dealer
	state.DealerIndex = nextActivePlayer(state, state.DealerIndex)

	// Post blinds
	sbIdx := nextActivePlayer(state, state.DealerIndex)
	bbIdx := nextActivePlayer(state, sbIdx)

	// Handle heads-up (2 players): dealer posts small blind
	if activePlayers == 2 {
		sbIdx = state.DealerIndex
		bbIdx = nextActivePlayer(state, sbIdx)
	}

	postBlind(state, sbIdx, state.SmallBlind)
	postBlind(state, bbIdx, state.BigBlind)

	state.CurrentBet = state.BigBlind
	state.MinRaise = state.BigBlind

	// Deal hole cards
	for i := range state.Players {
		if state.Players[i].IsActive && state.Players[i].Chips >= 0 {
			state.Players[i].HoleCards = []PokerCard{drawCard(state), drawCard(state)}
		}
	}

	// First to act is after big blind
	state.CurrentPlayerIdx = nextActivePlayer(state, bbIdx)
	state.LastRaiserIdx = bbIdx // Big blind is initial "raiser"
	state.Phase = PokerPhasePreflop

	addPokerLog(state, fmt.Sprintf("Hand #%d starts. Dealer: %s", state.HandNumber, state.Players[state.DealerIndex].Name))
	addPokerLog(state, fmt.Sprintf("%s posts small blind (%d)", state.Players[sbIdx].Name, min(state.SmallBlind, state.Players[sbIdx].Chips+state.Players[sbIdx].CurrentBet)))
	addPokerLog(state, fmt.Sprintf("%s posts big blind (%d)", state.Players[bbIdx].Name, min(state.BigBlind, state.Players[bbIdx].Chips+state.Players[bbIdx].CurrentBet)))

	// Check if we need to skip to showdown (all but one all-in after blinds)
	checkAutoAdvance(state)
}

func postBlind(state *PokerState, playerIdx int, amount int) {
	p := &state.Players[playerIdx]
	blind := amount
	if blind > p.Chips {
		blind = p.Chips
	}
	p.Chips -= blind
	p.CurrentBet = blind
	p.TotalBet = blind
	if p.Chips == 0 {
		p.AllIn = true
	}
}

func drawCard(state *PokerState) PokerCard {
	card := state.Deck[0]
	state.Deck = state.Deck[1:]
	return card
}

func nextActivePlayer(state *PokerState, fromIdx int) int {
	n := len(state.Players)
	for i := 1; i <= n; i++ {
		idx := (fromIdx + i) % n
		p := &state.Players[idx]
		if p.IsActive && !p.Folded && p.Chips > 0 {
			return idx
		}
	}
	// If no one with chips, find any active non-folded (all-in)
	for i := 1; i <= n; i++ {
		idx := (fromIdx + i) % n
		p := &state.Players[idx]
		if p.IsActive && !p.Folded {
			return idx
		}
	}
	return fromIdx
}

func countActivePlayers(state *PokerState) int {
	count := 0
	for _, p := range state.Players {
		if p.IsActive && !p.Folded {
			count++
		}
	}
	return count
}

func countPlayersCanAct(state *PokerState) int {
	count := 0
	for _, p := range state.Players {
		if p.IsActive && !p.Folded && !p.AllIn {
			count++
		}
	}
	return count
}

// PokerAction handles a player action
func PokerAction(state *PokerState, playerID string, action string, amount int) error {
	if state.Phase == PokerPhaseShowdown || state.Phase == PokerPhaseGameOver || state.Phase == PokerPhaseWaiting {
		return fmt.Errorf("cannot act in this phase")
	}

	cp := &state.Players[state.CurrentPlayerIdx]
	if cp.ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if cp.Folded || cp.AllIn {
		return fmt.Errorf("you cannot act")
	}

	cp.RoundActed = true

	switch action {
	case "fold":
		cp.Folded = true
		addPokerLog(state, fmt.Sprintf("%s folds", cp.Name))
		state.LastAction = cp.Name + " folds"

		// Check if only one player left
		if countActivePlayers(state) == 1 {
			// Award pot to the remaining player
			awardPotToLastPlayer(state)
			return nil
		}

	case "check":
		if cp.CurrentBet < state.CurrentBet {
			return fmt.Errorf("cannot check, must call %d or fold", state.CurrentBet-cp.CurrentBet)
		}
		addPokerLog(state, fmt.Sprintf("%s checks", cp.Name))
		state.LastAction = cp.Name + " checks"

	case "call":
		callAmount := state.CurrentBet - cp.CurrentBet
		if callAmount <= 0 {
			return fmt.Errorf("nothing to call")
		}
		if callAmount > cp.Chips {
			callAmount = cp.Chips // All-in
		}
		cp.Chips -= callAmount
		cp.CurrentBet += callAmount
		cp.TotalBet += callAmount
		if cp.Chips == 0 {
			cp.AllIn = true
			addPokerLog(state, fmt.Sprintf("%s calls %d (ALL IN)", cp.Name, callAmount))
			state.LastAction = cp.Name + " ALL IN"
		} else {
			addPokerLog(state, fmt.Sprintf("%s calls %d", cp.Name, callAmount))
			state.LastAction = fmt.Sprintf("%s calls %d", cp.Name, callAmount)
		}

	case "raise":
		raiseTotal := amount // Total bet amount after raise
		raiseBy := raiseTotal - state.CurrentBet
		if raiseBy < state.MinRaise && raiseTotal < cp.Chips+cp.CurrentBet {
			return fmt.Errorf("raise must be at least %d", state.MinRaise)
		}
		callAmount := raiseTotal - cp.CurrentBet
		if callAmount > cp.Chips {
			callAmount = cp.Chips
			raiseTotal = cp.CurrentBet + callAmount
		}
		cp.Chips -= callAmount
		cp.CurrentBet = raiseTotal
		cp.TotalBet += callAmount
		state.MinRaise = raiseBy
		if raiseTotal > state.CurrentBet {
			state.CurrentBet = raiseTotal
		}
		state.LastRaiserIdx = state.CurrentPlayerIdx
		if cp.Chips == 0 {
			cp.AllIn = true
			addPokerLog(state, fmt.Sprintf("%s raises to %d (ALL IN)", cp.Name, raiseTotal))
			state.LastAction = cp.Name + " ALL IN"
		} else {
			addPokerLog(state, fmt.Sprintf("%s raises to %d", cp.Name, raiseTotal))
			state.LastAction = fmt.Sprintf("%s raises to %d", cp.Name, raiseTotal)
		}

	case "allin":
		allInAmount := cp.Chips
		cp.CurrentBet += allInAmount
		cp.TotalBet += allInAmount
		cp.Chips = 0
		cp.AllIn = true
		if cp.CurrentBet > state.CurrentBet {
			raiseBy := cp.CurrentBet - state.CurrentBet
			if raiseBy >= state.MinRaise {
				state.MinRaise = raiseBy
			}
			state.CurrentBet = cp.CurrentBet
			state.LastRaiserIdx = state.CurrentPlayerIdx
		}
		addPokerLog(state, fmt.Sprintf("%s goes ALL IN (%d)", cp.Name, allInAmount))
		state.LastAction = fmt.Sprintf("%s ALL IN (%d)", cp.Name, allInAmount)

	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	advancePlay(state)
	return nil
}

func advancePlay(state *PokerState) {
	// Check if only one player left
	if countActivePlayers(state) <= 1 {
		awardPotToLastPlayer(state)
		return
	}

	// Find next player who can act (not folded, not all-in)
	n := len(state.Players)
	for i := 1; i <= n; i++ {
		idx := (state.CurrentPlayerIdx + i) % n
		p := &state.Players[idx]
		if !p.IsActive || p.Folded || p.AllIn {
			continue
		}
		// If we've come back to the last raiser with everyone matched and they have acted, round is over
		if idx == state.LastRaiserIdx && p.CurrentBet >= state.CurrentBet && p.RoundActed {
			advancePhase(state)
			return
		}
		// If this player hasn't matched the current bet, they need to act
		if p.CurrentBet < state.CurrentBet {
			state.CurrentPlayerIdx = idx
			return
		}
		// Player has matched but hasn't acted yet this round (e.g., BB preflop)
		if !p.RoundActed {
			state.CurrentPlayerIdx = idx
			return
		}
	}

	// If we get here, everyone has acted — advance phase
	advancePhase(state)
}

func advancePhase(state *PokerState) {
	// Collect bets into pots
	collectBets(state)

	// Reset current bets for new round
	for i := range state.Players {
		state.Players[i].CurrentBet = 0
		state.Players[i].RoundActed = false
	}
	state.CurrentBet = 0
	state.MinRaise = state.BigBlind

	switch state.Phase {
	case PokerPhasePreflop:
		// Deal flop (3 community cards)
		drawCard(state) // burn
		state.CommunityCards = append(state.CommunityCards, drawCard(state), drawCard(state), drawCard(state))
		state.Phase = PokerPhaseFlop
		addPokerLog(state, fmt.Sprintf("Flop: %s %s %s", state.CommunityCards[0], state.CommunityCards[1], state.CommunityCards[2]))

	case PokerPhaseFlop:
		drawCard(state) // burn
		card := drawCard(state)
		state.CommunityCards = append(state.CommunityCards, card)
		state.Phase = PokerPhaseTurn
		addPokerLog(state, fmt.Sprintf("Turn: %s", card))

	case PokerPhaseTurn:
		drawCard(state) // burn
		card := drawCard(state)
		state.CommunityCards = append(state.CommunityCards, card)
		state.Phase = PokerPhaseRiver
		addPokerLog(state, fmt.Sprintf("River: %s", card))

	case PokerPhaseRiver:
		resolveShowdown(state)
		return
	}

	// Set first to act (left of dealer)
	state.CurrentPlayerIdx = nextActivePlayer(state, state.DealerIndex)
	state.LastRaiserIdx = state.CurrentPlayerIdx // Reset: first player is also the "last raiser" for new round
	checkAutoAdvance(state)
}

func checkAutoAdvance(state *PokerState) {
	if state.Phase == PokerPhaseShowdown || state.Phase == PokerPhaseGameOver {
		return
	}

	canAct := countPlayersCanAct(state)
	active := countActivePlayers(state)

	if active <= 1 {
		awardPotToLastPlayer(state)
		return
	}

	if canAct <= 1 {
		// Everyone is all-in or folded except maybe one
		// If there are still community cards to deal, deal them out
		collectBets(state)
		for i := range state.Players {
			state.Players[i].CurrentBet = 0
		}
		state.CurrentBet = 0

		// Deal remaining community cards
		for len(state.CommunityCards) < 5 {
			if len(state.CommunityCards) == 0 {
				drawCard(state) // burn
				state.CommunityCards = append(state.CommunityCards, drawCard(state), drawCard(state), drawCard(state))
				addPokerLog(state, fmt.Sprintf("Flop: %s %s %s", state.CommunityCards[0], state.CommunityCards[1], state.CommunityCards[2]))
			} else {
				drawCard(state) // burn
				card := drawCard(state)
				state.CommunityCards = append(state.CommunityCards, card)
				if len(state.CommunityCards) == 4 {
					addPokerLog(state, fmt.Sprintf("Turn: %s", card))
				} else {
					addPokerLog(state, fmt.Sprintf("River: %s", card))
				}
			}
		}
		resolveShowdown(state)
	}
}

func collectBets(state *PokerState) {
	// Sum all current bets into the main pot
	totalBets := 0
	var eligible []string
	for i := range state.Players {
		totalBets += state.Players[i].CurrentBet
		if state.Players[i].IsActive && !state.Players[i].Folded {
			eligible = append(eligible, state.Players[i].ID)
		}
	}

	if totalBets == 0 {
		return
	}

	if len(state.Pots) == 0 {
		state.Pots = []Pot{{Amount: totalBets, Eligible: eligible}}
	} else {
		state.Pots[0].Amount += totalBets
		state.Pots[0].Eligible = eligible
	}
}

func awardPotToLastPlayer(state *PokerState) {
	collectBets(state)
	for i := range state.Players {
		state.Players[i].CurrentBet = 0
	}

	var winner *PokerPlayer
	for i := range state.Players {
		if state.Players[i].IsActive && !state.Players[i].Folded {
			winner = &state.Players[i]
			break
		}
	}

	if winner == nil {
		return
	}

	totalPot := 0
	for _, pot := range state.Pots {
		totalPot += pot.Amount
	}

	winner.Chips += totalPot
	addPokerLog(state, fmt.Sprintf("%s wins %d chips", winner.Name, totalPot))
	state.Pots = nil

	state.Phase = PokerPhaseShowdown
	state.LastAction = fmt.Sprintf("%s wins %d", winner.Name, totalPot)
}

func resolveShowdown(state *PokerState) {
	state.Phase = PokerPhaseShowdown

	// Evaluate hands for all active non-folded players
	var contenders []int
	for i, p := range state.Players {
		if p.IsActive && !p.Folded {
			hand := evaluateHand(p.HoleCards, state.CommunityCards)
			state.Players[i].Hand = &hand
			contenders = append(contenders, i)
		}
	}

	// Award pot(s)
	totalPot := 0
	for _, pot := range state.Pots {
		totalPot += pot.Amount
	}

	if len(contenders) == 0 {
		state.Pots = nil
		return
	}

	// Find winners
	winners := findWinners(state, contenders)

	share := totalPot / len(winners)
	remainder := totalPot % len(winners)
	for i, wIdx := range winners {
		extra := 0
		if i == 0 {
			extra = remainder
		}
		state.Players[wIdx].Chips += share + extra
	}

	winnerNames := ""
	for i, wIdx := range winners {
		if i > 0 {
			winnerNames += ", "
		}
		winnerNames += state.Players[wIdx].Name
	}

	handName := ""
	if state.Players[winners[0]].Hand != nil {
		handName = " with " + state.Players[winners[0]].Hand.RankName
	}
	addPokerLog(state, fmt.Sprintf("%s wins %d chips%s", winnerNames, totalPot, handName))
	state.LastAction = fmt.Sprintf("%s wins %d%s", winnerNames, totalPot, handName)
	state.Pots = nil
}

func findWinners(state *PokerState, contenders []int) []int {
	if len(contenders) == 0 {
		return nil
	}

	best := contenders[0]
	winners := []int{best}

	for _, idx := range contenders[1:] {
		cmp := compareHands(state.Players[idx].Hand, state.Players[best].Hand)
		if cmp > 0 {
			best = idx
			winners = []int{idx}
		} else if cmp == 0 {
			winners = append(winners, idx)
		}
	}

	return winners
}

func compareHands(a, b *EvaluatedHand) int {
	if a.Rank > b.Rank {
		return 1
	}
	if a.Rank < b.Rank {
		return -1
	}
	// Same rank, compare kickers
	for i := 0; i < len(a.Kickers) && i < len(b.Kickers); i++ {
		if a.Kickers[i] > b.Kickers[i] {
			return 1
		}
		if a.Kickers[i] < b.Kickers[i] {
			return -1
		}
	}
	return 0
}

// StartNextHand starts a new hand (called by server when players click continue)
func StartNextHand(state *PokerState) {
	state.Phase = PokerPhaseWaiting
	startNewHand(state)
}

// PokerVoluntaryExit handles a player leaving mid-game
func PokerVoluntaryExit(state *PokerState, playerID string) {
	for i := range state.Players {
		if state.Players[i].ID == playerID {
			state.Players[i].Folded = true
			state.Players[i].IsActive = false
			state.Players[i].Chips = 0
			addPokerLog(state, fmt.Sprintf("%s leaves the game", state.Players[i].Name))

			// Check if game should end
			activeCount := 0
			for _, p := range state.Players {
				if p.IsActive {
					activeCount++
				}
			}
			if activeCount < 2 {
				if countActivePlayers(state) <= 1 {
					awardPotToLastPlayer(state)
				}
				state.Phase = PokerPhaseGameOver
				finalizeScoreboard(state)
			} else if state.CurrentPlayerIdx == i {
				advancePlay(state)
			}
			break
		}
	}
}

func finalizeScoreboard(state *PokerState) {
	state.Scoreboard = nil
	for _, p := range state.Players {
		state.Scoreboard = append(state.Scoreboard, PokerScoreEntry{
			PlayerID:   p.ID,
			Name:       p.Name,
			NetGain:    p.Chips - state.BuyIn,
			FinalChips: p.Chips,
		})
	}
	// Sort by final chips descending
	sort.Slice(state.Scoreboard, func(i, j int) bool {
		return state.Scoreboard[i].FinalChips > state.Scoreboard[j].FinalChips
	})

	// Find winner
	if len(state.Scoreboard) > 0 {
		state.Winner = state.Scoreboard[0].PlayerID
	}
}

func addPokerLog(state *PokerState, message string) {
	state.Log = append(state.Log, PokerLogEntry{
		Timestamp: time.Now().UnixMilli(),
		Message:   message,
		Hand:      state.HandNumber,
	})
}

// ========== HAND EVALUATION ==========

func evaluateHand(holeCards []PokerCard, communityCards []PokerCard) EvaluatedHand {
	all := append([]PokerCard{}, holeCards...)
	all = append(all, communityCards...)

	// Generate all 5-card combinations from 7 cards
	combos := combinations(all, 5)

	var bestHand EvaluatedHand
	bestHand.Rank = -1

	for _, combo := range combos {
		hand := evaluateFiveCards(combo)
		if compareHands(&hand, &bestHand) > 0 {
			bestHand = hand
		}
	}

	return bestHand
}

func combinations(cards []PokerCard, k int) [][]PokerCard {
	var result [][]PokerCard
	var helper func(start int, combo []PokerCard)
	helper = func(start int, combo []PokerCard) {
		if len(combo) == k {
			c := make([]PokerCard, k)
			copy(c, combo)
			result = append(result, c)
			return
		}
		for i := start; i < len(cards); i++ {
			helper(i+1, append(combo, cards[i]))
		}
	}
	helper(0, nil)
	return result
}

func evaluateFiveCards(cards []PokerCard) EvaluatedHand {
	// Sort by rank descending
	sorted := make([]PokerCard, len(cards))
	copy(sorted, cards)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rank > sorted[j].Rank })

	ranks := make([]Rank, 5)
	for i, c := range sorted {
		ranks[i] = c.Rank
	}

	isFlush := sorted[0].Suit == sorted[1].Suit &&
		sorted[1].Suit == sorted[2].Suit &&
		sorted[2].Suit == sorted[3].Suit &&
		sorted[3].Suit == sorted[4].Suit

	isStraight := false
	straightHigh := Rank(0)

	// Normal straight check
	if ranks[0]-ranks[4] == 4 &&
		ranks[0] != ranks[1] && ranks[1] != ranks[2] && ranks[2] != ranks[3] && ranks[3] != ranks[4] {
		isStraight = true
		straightHigh = ranks[0]
	}

	// Ace-low straight (A-2-3-4-5)
	if !isStraight && ranks[0] == RankAce && ranks[1] == RankFive && ranks[2] == RankFour && ranks[3] == RankThree && ranks[4] == RankTwo {
		isStraight = true
		straightHigh = RankFive // 5-high straight
	}

	// Count ranks
	rankCount := make(map[Rank]int)
	for _, r := range ranks {
		rankCount[r]++
	}

	// Classify
	var pairs []Rank
	var trips []Rank
	var quads []Rank
	var singles []Rank

	for r, count := range rankCount {
		switch count {
		case 4:
			quads = append(quads, r)
		case 3:
			trips = append(trips, r)
		case 2:
			pairs = append(pairs, r)
		case 1:
			singles = append(singles, r)
		}
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i] > pairs[j] })
	sort.Slice(singles, func(i, j int) bool { return singles[i] > singles[j] })

	result := EvaluatedHand{BestFive: sorted}

	if isStraight && isFlush {
		if straightHigh == RankAce {
			result.Rank = HandRoyalFlush
			result.RankName = "Royal Flush"
			result.Kickers = []Rank{RankAce}
		} else {
			result.Rank = HandStraightFlush
			result.RankName = "Straight Flush"
			result.Kickers = []Rank{straightHigh}
		}
	} else if len(quads) == 1 {
		result.Rank = HandFourOfAKind
		result.RankName = "Four of a Kind"
		result.Kickers = append([]Rank{quads[0]}, singles...)
	} else if len(trips) == 1 && len(pairs) == 1 {
		result.Rank = HandFullHouse
		result.RankName = "Full House"
		result.Kickers = []Rank{trips[0], pairs[0]}
	} else if isFlush {
		result.Rank = HandFlush
		result.RankName = "Flush"
		result.Kickers = ranks
	} else if isStraight {
		result.Rank = HandStraight
		result.RankName = "Straight"
		result.Kickers = []Rank{straightHigh}
	} else if len(trips) == 1 {
		result.Rank = HandThreeOfAKind
		result.RankName = "Three of a Kind"
		result.Kickers = append([]Rank{trips[0]}, singles...)
	} else if len(pairs) == 2 {
		result.Rank = HandTwoPair
		result.RankName = "Two Pair"
		result.Kickers = append(pairs, singles...)
	} else if len(pairs) == 1 {
		result.Rank = HandOnePair
		result.RankName = "One Pair"
		result.Kickers = append([]Rank{pairs[0]}, singles...)
	} else {
		result.Rank = HandHighCard
		result.RankName = "High Card"
		result.Kickers = ranks
	}

	return result
}
