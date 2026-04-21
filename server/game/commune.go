package game

import (
	"fmt"
	"math/rand"
	"time"
)

// Commune phases
type CommunePhase string

const (
	CommunePhaseWaiting  CommunePhase = "waiting"
	CommunePhasePlaying  CommunePhase = "playing"
	CommunePhaseCalled   CommunePhase = "called"
	CommunePhaseFinished CommunePhase = "finished"
)

// Hand type constants are reused from poker.go:
// HandHighCard(0), HandOnePair(1), HandTwoPair(2), HandThreeOfAKind(3),
// HandStraight(4), HandFlush(5), HandFullHouse(6), HandFourOfAKind(7)
// Additional Commune-specific hand types:
const (
	CommuneTwoTrips      HandRank = 8  // Two sets of three of a kind
	CommuneUltaStraight  HandRank = 9  // Full straight 3 through A (12 cards)
	CommuneFourPlusThree HandRank = 10 // Four of a kind + three of a kind
	CommuneFiveOfAKind   HandRank = 11 // Five of the same rank (using wilds)
)

type CommuneCard struct {
	ID     string `json:"id"`
	Rank   int    `json:"rank"` // 3-14 for normal, 2 for twos, 0 for joker
	Suit   string `json:"suit"` // hearts,diamonds,clubs,spades or "" for joker
	IsWild bool   `json:"isWild"`
}

func (c CommuneCard) Display() string {
	if c.Rank == 0 {
		return "🃏"
	}
	suits := map[string]string{"hearts": "♥", "diamonds": "♦", "clubs": "♣", "spades": "♠"}
	ranks := map[int]string{2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "10", 11: "J", 12: "Q", 13: "K", 14: "A"}
	return ranks[c.Rank] + suits[c.Suit]
}

type CommunePlayer struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Tokens  int            `json:"tokens"`
	Cards   []CommuneCard  `json:"cards"`
	IsAlive bool           `json:"isAlive"`
}

type CommuneDeclaration struct {
	HandType      HandRank `json:"handType"`      // 0-9
	PrimaryRank   int      `json:"primaryRank"`   // 3-14
	SecondaryRank int      `json:"secondaryRank"` // for two pair, full house
}

type CommuneDeclEntry struct {
	PlayerIdx   int                 `json:"playerIdx"`
	PlayerName  string              `json:"playerName"`
	Declaration CommuneDeclaration  `json:"declaration"`
	DisplayText string              `json:"displayText"`
}

type CommuneState struct {
	ID               string              `json:"id"`
	Players          []CommunePlayer     `json:"players"`
	Phase            CommunePhase        `json:"phase"`
	DealerIdx        int                 `json:"dealerIdx"`
	CurrentPlayerIdx int                 `json:"currentPlayerIdx"`
	LastDeclaration  *CommuneDeclaration `json:"lastDeclaration"`
	LastDeclarerIdx  int                 `json:"lastDeclarerIdx"`
	Declarations     []CommuneDeclEntry  `json:"declarations"`
	CallerIdx        int                 `json:"callerIdx"`
	CallResult       string              `json:"callResult"` // "caller_loses" or "declarer_loses"
	LoserIdx         int                 `json:"loserIdx"`
	CommunityCards   []CommuneCard       `json:"communityCards"` // 2 face-up community cards
	AllCards         []CommuneCard       `json:"allCards"`       // revealed on call
	Round            int                 `json:"round"`
	Log              []CommuneLogEntry   `json:"log"`
	LastAction       string              `json:"lastAction"`
	Winner           string              `json:"winner"`
}

type CommuneLogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func addCommuneLog(state *CommuneState, msg string) {
	state.Log = append(state.Log, CommuneLogEntry{
		Timestamp: time.Now().UnixMilli(),
		Message:   msg,
	})
	state.LastAction = msg
}

// --- Deck & Dealing ---

func newCommuneDeck() []CommuneCard {
	cards := make([]CommuneCard, 0, 52)
	suits := []string{"hearts", "diamonds", "clubs", "spades"}
	id := 0
	for _, suit := range suits {
		for rank := 2; rank <= 14; rank++ {
			cards = append(cards, CommuneCard{
				ID:     fmt.Sprintf("c%d", id),
				Rank:   rank,
				Suit:   suit,
				IsWild: rank == 2,
			})
			id++
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	return cards
}

func communeDeal(state *CommuneState) {
	deck := newCommuneDeck()
	idx := 0

	// Deal 2 community cards first
	state.CommunityCards = deck[idx : idx+2]
	idx += 2

	for i := range state.Players {
		if !state.Players[i].IsAlive {
			state.Players[i].Cards = nil
			continue
		}
		numCards := 6 - state.Players[i].Tokens
		if numCards < 1 {
			numCards = 1
		}
		state.Players[i].Cards = deck[idx : idx+numCards]
		idx += numCards
	}
	state.AllCards = deck[:idx] // community + all player cards
}

// --- Initialization ---

func InitializeCommuneGame(players []struct{ ID, Name string }) *CommuneState {
	communePlayers := make([]CommunePlayer, len(players))
	for i, p := range players {
		communePlayers[i] = CommunePlayer{
			ID:      p.ID,
			Name:    p.Name,
			Tokens:  5,
			IsAlive: true,
		}
	}

	state := &CommuneState{
		ID:           fmt.Sprintf("commune-%d", time.Now().UnixNano()),
		Players:      communePlayers,
		Phase:        CommunePhasePlaying,
		DealerIdx:    0,
		Round:        1,
		Log:          []CommuneLogEntry{},
		Declarations: []CommuneDeclEntry{},
	}

	communeDeal(state)
	state.CurrentPlayerIdx = communeNextAlive(state, state.DealerIdx)
	addCommuneLog(state, fmt.Sprintf("Round %d — Cards dealt! %s declares first.", state.Round, state.Players[state.CurrentPlayerIdx].Name))
	return state
}

// --- Declaration Logic ---

func CommuneDeclare(state *CommuneState, playerID string, decl CommuneDeclaration) error {
	if state.Phase != CommunePhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	if state.Players[state.CurrentPlayerIdx].ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if err := validateCommuneDeclaration(decl); err != nil {
		return err
	}
	if state.LastDeclaration != nil && !isHigherDeclaration(decl, *state.LastDeclaration) {
		return fmt.Errorf("declaration must be higher than the previous one")
	}

	text := communeDeclText(decl)
	state.Declarations = append(state.Declarations, CommuneDeclEntry{
		PlayerIdx:   state.CurrentPlayerIdx,
		PlayerName:  state.Players[state.CurrentPlayerIdx].Name,
		Declaration: decl,
		DisplayText: text,
	})
	state.LastDeclaration = &decl
	state.LastDeclarerIdx = state.CurrentPlayerIdx
	addCommuneLog(state, fmt.Sprintf("%s declares: %s", state.Players[state.CurrentPlayerIdx].Name, text))
	state.CurrentPlayerIdx = communeNextAlive(state, state.CurrentPlayerIdx)
	return nil
}

func CommuneCall(state *CommuneState, playerID string) error {
	if state.Phase != CommunePhasePlaying {
		return fmt.Errorf("not in playing phase")
	}
	if state.Players[state.CurrentPlayerIdx].ID != playerID {
		return fmt.Errorf("not your turn")
	}
	if state.LastDeclaration == nil {
		return fmt.Errorf("nothing to call — you must declare first")
	}

	state.CallerIdx = state.CurrentPlayerIdx
	callerName := state.Players[state.CallerIdx].Name
	declarerName := state.Players[state.LastDeclarerIdx].Name
	addCommuneLog(state, fmt.Sprintf("📣 %s calls %s's declaration: %s!", callerName, declarerName, communeDeclText(*state.LastDeclaration)))

	// Check if hand exists across ALL dealt cards (community + all player hands)
	present := isCommuneHandPresent(state.AllCards, *state.LastDeclaration)

	if present {
		state.CallResult = "caller_loses"
		state.LoserIdx = state.CallerIdx
		state.Players[state.CallerIdx].Tokens--
		addCommuneLog(state, fmt.Sprintf("✅ The hand IS present! %s loses a token. (%d remaining)", callerName, state.Players[state.CallerIdx].Tokens))
	} else {
		state.CallResult = "declarer_loses"
		state.LoserIdx = state.LastDeclarerIdx
		state.Players[state.LastDeclarerIdx].Tokens--
		addCommuneLog(state, fmt.Sprintf("❌ The hand is NOT present! %s loses a token. (%d remaining)", declarerName, state.Players[state.LastDeclarerIdx].Tokens))
	}

	// Check eliminations
	for i := range state.Players {
		if state.Players[i].Tokens <= 0 && state.Players[i].IsAlive {
			state.Players[i].IsAlive = false
			state.Players[i].Tokens = 0
			addCommuneLog(state, fmt.Sprintf("💀 %s is eliminated!", state.Players[i].Name))
		}
	}

	// Check if game over
	aliveCount := 0
	lastAlive := -1
	for i, p := range state.Players {
		if p.IsAlive {
			aliveCount++
			lastAlive = i
		}
	}

	if aliveCount <= 1 {
		state.Phase = CommunePhaseFinished
		if lastAlive >= 0 {
			state.Winner = state.Players[lastAlive].Name
			addCommuneLog(state, fmt.Sprintf("🏆 %s wins the game!", state.Winner))
		}
	} else {
		state.Phase = CommunePhaseCalled
	}
	return nil
}

func CommuneNextHand(state *CommuneState) {
	if state.Phase != CommunePhaseCalled {
		return
	}
	state.Round++
	state.DealerIdx = communeNextAlive(state, state.DealerIdx)
	state.LastDeclaration = nil
	state.Declarations = []CommuneDeclEntry{}
	state.CallResult = ""
	state.AllCards = nil
	state.CommunityCards = nil

	communeDeal(state)
	state.CurrentPlayerIdx = communeNextAlive(state, state.DealerIdx)
	state.Phase = CommunePhasePlaying
	addCommuneLog(state, fmt.Sprintf("— Round %d — Cards dealt! %s declares first.", state.Round, state.Players[state.CurrentPlayerIdx].Name))
}

func CommuneVoluntaryExit(state *CommuneState, playerID string) {
	if state.Phase == CommunePhaseFinished {
		return
	}
	for i, p := range state.Players {
		if p.ID == playerID {
			state.Players[i].IsAlive = false
			state.Players[i].Tokens = 0
			addCommuneLog(state, fmt.Sprintf("%s left the game.", p.Name))

			aliveCount := 0
			lastAlive := -1
			for j, pl := range state.Players {
				if pl.IsAlive {
					aliveCount++
					lastAlive = j
				}
			}

			if aliveCount <= 1 {
				state.Phase = CommunePhaseFinished
				if lastAlive >= 0 {
					state.Winner = state.Players[lastAlive].Name
					addCommuneLog(state, fmt.Sprintf("🏆 %s wins!", state.Winner))
				}
				return
			}

			if state.CurrentPlayerIdx == i && state.Phase == CommunePhasePlaying {
				state.CurrentPlayerIdx = communeNextAlive(state, i)
			}
			return
		}
	}
}

// --- Helpers ---

func communeNextAlive(state *CommuneState, fromIdx int) int {
	n := len(state.Players)
	for i := 1; i <= n; i++ {
		idx := (fromIdx + i) % n
		if state.Players[idx].IsAlive {
			return idx
		}
	}
	return fromIdx
}

func isHigherDeclaration(a, b CommuneDeclaration) bool {
	if a.HandType != b.HandType {
		return a.HandType > b.HandType
	}
	if a.PrimaryRank != b.PrimaryRank {
		return a.PrimaryRank > b.PrimaryRank
	}
	return a.SecondaryRank > b.SecondaryRank
}

func validateCommuneDeclaration(decl CommuneDeclaration) error {
	validTypes := map[HandRank]bool{0: true, 1: true, 2: true, 3: true, 4: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true}
	if !validTypes[decl.HandType] {
		return fmt.Errorf("invalid hand type")
	}
	// Ulta straight has no rank selection
	if decl.HandType == CommuneUltaStraight {
		return nil
	}
	if decl.PrimaryRank < 3 || decl.PrimaryRank > 14 {
		return fmt.Errorf("invalid rank")
	}
	switch decl.HandType {
	case HandTwoPair:
		if decl.SecondaryRank < 3 || decl.SecondaryRank > 14 {
			return fmt.Errorf("invalid secondary rank")
		}
		if decl.PrimaryRank <= decl.SecondaryRank {
			return fmt.Errorf("first pair must be higher rank than second")
		}
	case HandStraight:
		if decl.PrimaryRank < 5 {
			return fmt.Errorf("straight high card must be at least 5")
		}
	case HandFullHouse:
		if decl.SecondaryRank < 3 || decl.SecondaryRank > 14 {
			return fmt.Errorf("invalid secondary rank")
		}
		if decl.PrimaryRank == decl.SecondaryRank {
			return fmt.Errorf("full house ranks must differ")
		}
	case CommuneTwoTrips:
		if decl.SecondaryRank < 3 || decl.SecondaryRank > 14 {
			return fmt.Errorf("invalid secondary rank")
		}
		if decl.PrimaryRank <= decl.SecondaryRank {
			return fmt.Errorf("first trips must be higher rank than second")
		}
	case CommuneFourPlusThree:
		if decl.SecondaryRank < 3 || decl.SecondaryRank > 14 {
			return fmt.Errorf("invalid secondary rank")
		}
		if decl.PrimaryRank == decl.SecondaryRank {
			return fmt.Errorf("four+three ranks must differ")
		}
	}
	return nil
}

func communeRankName(rank int) string {
	names := map[int]string{3: "3s", 4: "4s", 5: "5s", 6: "6s", 7: "7s", 8: "8s", 9: "9s", 10: "10s", 11: "Jacks", 12: "Queens", 13: "Kings", 14: "Aces"}
	return names[rank]
}

func communeRankSingle(rank int) string {
	names := map[int]string{3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "10", 11: "Jack", 12: "Queen", 13: "King", 14: "Ace"}
	return names[rank]
}

func communeDeclText(d CommuneDeclaration) string {
	switch d.HandType {
	case HandHighCard:
		return "High Card — " + communeRankSingle(d.PrimaryRank)
	case HandOnePair:
		return "Pair of " + communeRankName(d.PrimaryRank)
	case HandTwoPair:
		return "Two Pair — " + communeRankName(d.PrimaryRank) + " and " + communeRankName(d.SecondaryRank)
	case HandThreeOfAKind:
		return "Three " + communeRankName(d.PrimaryRank)
	case HandStraight:
		return "Straight — " + communeRankSingle(d.PrimaryRank) + " high"
	case HandFullHouse:
		return "Full House — " + communeRankName(d.PrimaryRank) + " full of " + communeRankName(d.SecondaryRank)
	case HandFourOfAKind:
		return "Four " + communeRankName(d.PrimaryRank)
	case CommuneTwoTrips:
		return "Two Trips — " + communeRankName(d.PrimaryRank) + " and " + communeRankName(d.SecondaryRank)
	case CommuneUltaStraight:
		return "Ulta Straight (3 to A)"
	case CommuneFourPlusThree:
		return "Four " + communeRankName(d.PrimaryRank) + " + Three " + communeRankName(d.SecondaryRank)
	case CommuneFiveOfAKind:
		return "Five " + communeRankName(d.PrimaryRank)
	}
	return "Unknown"
}

// --- Hand Verification ---

func isCommuneHandPresent(cards []CommuneCard, decl CommuneDeclaration) bool {
	wildCount := 0
	rankCount := make(map[int]int)

	for _, c := range cards {
		if c.IsWild {
			wildCount++
		} else {
			rankCount[c.Rank]++
		}
	}

	switch decl.HandType {
	case HandHighCard:
		return rankCount[decl.PrimaryRank] >= 1 || wildCount >= 1
	case HandOnePair:
		return rankCount[decl.PrimaryRank]+wildCount >= 2
	case HandTwoPair:
		n1, n2 := rankCount[decl.PrimaryRank], rankCount[decl.SecondaryRank]
		need1, need2 := 0, 0
		if 2-n1 > 0 {
			need1 = 2 - n1
		}
		if 2-n2 > 0 {
			need2 = 2 - n2
		}
		return need1+need2 <= wildCount
	case HandThreeOfAKind:
		return rankCount[decl.PrimaryRank]+wildCount >= 3
	case HandStraight:
		return canCommuneStraight(rankCount, wildCount, decl.PrimaryRank)
	case HandFullHouse:
		n1, n2 := rankCount[decl.PrimaryRank], rankCount[decl.SecondaryRank]
		need1, need2 := 0, 0
		if 3-n1 > 0 {
			need1 = 3 - n1
		}
		if 2-n2 > 0 {
			need2 = 2 - n2
		}
		return need1+need2 <= wildCount
	case HandFourOfAKind:
		return rankCount[decl.PrimaryRank]+wildCount >= 4
	case CommuneTwoTrips:
		n1, n2 := rankCount[decl.PrimaryRank], rankCount[decl.SecondaryRank]
		need1, need2 := 0, 0
		if 3-n1 > 0 {
			need1 = 3 - n1
		}
		if 3-n2 > 0 {
			need2 = 3 - n2
		}
		return need1+need2 <= wildCount
	case CommuneUltaStraight:
		// Need all ranks 3 through 14 present
		needed := 0
		for r := 3; r <= 14; r++ {
			if rankCount[r] == 0 {
				needed++
			}
		}
		return needed <= wildCount
	case CommuneFourPlusThree:
		n1, n2 := rankCount[decl.PrimaryRank], rankCount[decl.SecondaryRank]
		need1, need2 := 0, 0
		if 4-n1 > 0 {
			need1 = 4 - n1
		}
		if 3-n2 > 0 {
			need2 = 3 - n2
		}
		return need1+need2 <= wildCount
	case CommuneFiveOfAKind:
		return rankCount[decl.PrimaryRank]+wildCount >= 5
	}
	return false
}

func communeStraightRanks(highCard int) []int {
	if highCard < 5 || highCard > 14 {
		return nil
	}
	if highCard == 5 {
		return []int{14, 2, 3, 4, 5} // Ace-low
	}
	ranks := make([]int, 5)
	for i := 0; i < 5; i++ {
		ranks[i] = highCard - 4 + i
	}
	return ranks
}

func canCommuneStraight(rankCount map[int]int, wildCount int, highCard int) bool {
	ranks := communeStraightRanks(highCard)
	if ranks == nil {
		return false
	}
	needed := 0
	for _, r := range ranks {
		if r == 2 {
			needed++ // 2s are wild, need a wild for this slot
		} else if rankCount[r] == 0 {
			needed++
		}
	}
	return needed <= wildCount
}

// --- Personalized State ---

func SanitizeCommuneStateForPlayer(state *CommuneState, viewerID string) *CommuneState {
	s := *state
	playersCopy := make([]CommunePlayer, len(state.Players))
	for i, p := range state.Players {
		playersCopy[i] = p
		if p.ID != viewerID && state.Phase != CommunePhaseCalled && state.Phase != CommunePhaseFinished {
			// Hide other players' cards — send count only
			playersCopy[i].Cards = make([]CommuneCard, len(p.Cards)) // empty cards with correct length
			for j := range playersCopy[i].Cards {
				playersCopy[i].Cards[j] = CommuneCard{ID: "hidden"}
			}
		}
	}
	s.Players = playersCopy

	// Community cards are always visible
	commCopy := make([]CommuneCard, len(state.CommunityCards))
	copy(commCopy, state.CommunityCards)
	s.CommunityCards = commCopy

	if state.Phase != CommunePhaseCalled && state.Phase != CommunePhaseFinished {
		s.AllCards = nil
	}

	declCopy := make([]CommuneDeclEntry, len(state.Declarations))
	copy(declCopy, state.Declarations)
	s.Declarations = declCopy

	logCopy := make([]CommuneLogEntry, len(state.Log))
	copy(logCopy, state.Log)
	s.Log = logCopy

	return &s
}
