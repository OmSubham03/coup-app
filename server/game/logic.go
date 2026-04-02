package game

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// Character types
type CharacterType string

const (
	Duke        CharacterType = "Duke"
	Assassin    CharacterType = "Assassin"
	Captain     CharacterType = "Captain"
	Ambassador  CharacterType = "Ambassador"
	Contessa    CharacterType = "Contessa"
	Inquisitor  CharacterType = "Inquisitor"
)

// Action types
type ActionType string

const (
	ActionIncome      ActionType = "income"
	ActionForeignAid  ActionType = "foreign_aid"
	ActionCoup        ActionType = "coup"
	ActionTax         ActionType = "tax"
	ActionAssassinate ActionType = "assassinate"
	ActionSteal       ActionType = "steal"
	ActionExchange    ActionType = "exchange"
	ActionInterrogate ActionType = "interrogate"
	ActionInquire     ActionType = "inquire"
)

// Block types
type BlockType string

const (
	BlockForeignAid  BlockType = "block_foreign_aid"
	BlockAssassinate BlockType = "block_assassinate"
	BlockSteal       BlockType = "block_steal"
)

// Game phases
type GamePhase string

const (
	PhaseWaiting             GamePhase = "waiting"
	PhaseAction              GamePhase = "action"
	PhaseBlockWindow         GamePhase = "block_window"
	PhaseChallengeWindow     GamePhase = "challenge_window"
	PhaseResolving           GamePhase = "resolving"
	PhaseExchange            GamePhase = "exchange"
	PhaseInterrogateSelect   GamePhase = "interrogate_select"
	PhaseInterrogateDecision GamePhase = "interrogate_decision"
	PhaseLoseInfluence       GamePhase = "lose_influence"
	PhaseGameOver            GamePhase = "game_over"
)

// Variant types
type VariantKey string

const (
	VariantStandard   VariantKey = "standard"
	VariantInquisitor VariantKey = "inquisitor"
)

// Card represents a character card
type Card struct {
	ID        string        `json:"id"`
	Character CharacterType `json:"character"`
	Revealed  bool          `json:"revealed"`
}

// Player represents a game player
type Player struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Coins   int    `json:"coins"`
	Cards   []Card `json:"cards"`
	IsAlive bool   `json:"isAlive"`
}

// ActionRequest represents a player action
type ActionRequest struct {
	Type             ActionType    `json:"type"`
	ActorID          string        `json:"actorId"`
	TargetID         string        `json:"targetId,omitempty"`
	ClaimedCharacter CharacterType `json:"claimedCharacter,omitempty"`
}

// BlockRequest represents a block attempt
type BlockRequest struct {
	Type             BlockType     `json:"type"`
	BlockerID        string        `json:"blockerId"`
	ClaimedCharacter CharacterType `json:"claimedCharacter"`
	TargetActionID   string        `json:"targetActionId"`
}

// ChallengeRequest represents a challenge
type ChallengeRequest struct {
	ChallengerID     string        `json:"challengerId"`
	TargetPlayerID   string        `json:"targetPlayerId"`
	ClaimedCharacter CharacterType `json:"claimedCharacter"`
	IsBlockChallenge bool          `json:"isBlockChallenge"`
}

// PendingInterrogate holds interrogation state
type PendingInterrogate struct {
	TargetID       string `json:"targetId"`
	SelectedCardID string `json:"selectedCardId,omitempty"`
	ActorDecision  string `json:"actorDecision,omitempty"`
}

// GameLogEntry is a log entry
type GameLogEntry struct {
	Timestamp  int64  `json:"timestamp"`
	Message    string `json:"message"`
	PlayerID   string `json:"playerId,omitempty"`
	ActionType string `json:"actionType,omitempty"`
	TargetID   string `json:"targetId,omitempty"`
	Turn       int    `json:"turn"`
}

// GameState holds the full game state
type GameState struct {
	ID                   string              `json:"id"`
	Variant              VariantKey          `json:"variant"`
	Players              []Player            `json:"players"`
	CurrentPlayerIndex   int                 `json:"currentPlayerIndex"`
	CourtDeck            []Card              `json:"courtDeck"`
	DiscardPile          []Card              `json:"discardPile"`
	Phase                GamePhase           `json:"phase"`
	PendingAction        *ActionRequest      `json:"pendingAction"`
	PendingBlock         *BlockRequest       `json:"pendingBlock"`
	PendingChallenge     *ChallengeRequest   `json:"pendingChallenge"`
	PendingExchangeCards []Card              `json:"pendingExchangeCards"`
	PendingInterrogate   *PendingInterrogate `json:"pendingInterrogate"`
	PendingInfluenceLoss string              `json:"pendingInfluenceLoss"`
	PassedPlayers        []string            `json:"passedPlayers"`
	Winner               string              `json:"winner"`
	Turn                 int                 `json:"turn"`
	Log                  []GameLogEntry      `json:"log"`
}

// ActionRequirement defines action properties
type ActionRequirement struct {
	Character          CharacterType
	Cost               int
	NeedsTarget        bool
	CanBeBlocked       bool
	BlockingCharacters []CharacterType
}

func getCharacters(variant VariantKey) []CharacterType {
	if variant == VariantInquisitor {
		return []CharacterType{Duke, Assassin, Captain, Inquisitor, Contessa}
	}
	return []CharacterType{Duke, Assassin, Captain, Ambassador, Contessa}
}

func getAvailableActions(variant VariantKey) []ActionType {
	if variant == VariantInquisitor {
		return []ActionType{ActionIncome, ActionForeignAid, ActionCoup, ActionTax, ActionAssassinate, ActionSteal, ActionInterrogate, ActionInquire}
	}
	return []ActionType{ActionIncome, ActionForeignAid, ActionCoup, ActionTax, ActionAssassinate, ActionSteal, ActionExchange}
}

func getActionRequirement(variant VariantKey, action ActionType) ActionRequirement {
	stealBlockers := []CharacterType{Captain, Ambassador}
	if variant == VariantInquisitor {
		stealBlockers = []CharacterType{Captain, Inquisitor}
	}

	requirements := map[ActionType]ActionRequirement{
		ActionIncome:     {},
		ActionForeignAid: {CanBeBlocked: true, BlockingCharacters: []CharacterType{Duke}},
		ActionCoup:       {Cost: 7, NeedsTarget: true},
		ActionTax:        {Character: Duke},
		ActionAssassinate: {Character: Assassin, Cost: 3, NeedsTarget: true,
			CanBeBlocked: true, BlockingCharacters: []CharacterType{Contessa}},
		ActionSteal: {Character: Captain, NeedsTarget: true,
			CanBeBlocked: true, BlockingCharacters: stealBlockers},
		ActionExchange:    {Character: Ambassador},
		ActionInterrogate: {Character: Inquisitor, NeedsTarget: true},
		ActionInquire:     {Character: Inquisitor},
	}

	if req, ok := requirements[action]; ok {
		return req
	}
	return ActionRequirement{}
}

func createDeck(variant VariantKey) []Card {
	characters := getCharacters(variant)
	var deck []Card

	for _, char := range characters {
		for i := 0; i < 3; i++ {
			deck = append(deck, Card{
				ID:        string(char) + "-" + uuid.New().String()[:8],
				Character: char,
				Revealed:  false,
			})
		}
	}

	return shuffleDeck(deck)
}

func shuffleDeck(deck []Card) []Card {
	shuffled := make([]Card, len(deck))
	copy(shuffled, deck)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func NormalizeVariant(v string) VariantKey {
	if v == string(VariantInquisitor) {
		return VariantInquisitor
	}
	return VariantStandard
}

// InitializeGame creates a new game state
func InitializeGame(playersList []struct{ ID, Name string }, variant VariantKey) *GameState {
	deck := createDeck(variant)
	players := make([]Player, len(playersList))

	for i, p := range playersList {
		card1 := deck[len(deck)-1]
		deck = deck[:len(deck)-1]
		card2 := deck[len(deck)-1]
		deck = deck[:len(deck)-1]

		players[i] = Player{
			ID:      p.ID,
			Name:    p.Name,
			Coins:   2,
			Cards:   []Card{card1, card2},
			IsAlive: true,
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	startIdx := r.Intn(len(players))

	state := &GameState{
		ID:                 "game-" + uuid.New().String()[:8],
		Variant:            variant,
		Players:            players,
		CurrentPlayerIndex: startIdx,
		CourtDeck:          deck,
		Phase:              PhaseAction,
		PassedPlayers:      []string{},
		Turn:               1,
		Log: []GameLogEntry{{
			Timestamp: time.Now().UnixMilli(),
			Message:   "Game started",
			Turn:      1,
		}},
	}

	return state
}

func getCurrentPlayer(state *GameState) *Player {
	return &state.Players[state.CurrentPlayerIndex]
}

func getPlayer(state *GameState, playerID string) *Player {
	for i := range state.Players {
		if state.Players[i].ID == playerID {
			return &state.Players[i]
		}
	}
	return nil
}

func getAlivePlayers(state *GameState) []*Player {
	var alive []*Player
	for i := range state.Players {
		if state.Players[i].IsAlive {
			alive = append(alive, &state.Players[i])
		}
	}
	return alive
}

func getPlayerInfluence(player *Player) int {
	count := 0
	for _, c := range player.Cards {
		if !c.Revealed {
			count++
		}
	}
	return count
}

func addLog(state *GameState, message string, playerID, actionType, targetID string) {
	state.Log = append(state.Log, GameLogEntry{
		Timestamp:  time.Now().UnixMilli(),
		Message:    message,
		PlayerID:   playerID,
		ActionType: actionType,
		TargetID:   targetID,
		Turn:       state.Turn,
	})
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsAction(slice []ActionType, item ActionType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsChar(slice []CharacterType, item CharacterType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PerformAction executes a player action
func PerformAction(state *GameState, action ActionRequest) error {
	actor := getPlayer(state, action.ActorID)
	if actor == nil || !actor.IsAlive {
		return ErrInvalidPlayer
	}
	cp := getCurrentPlayer(state)
	if cp.ID != action.ActorID {
		return ErrNotYourTurn
	}
	if state.Phase != PhaseAction {
		return ErrWrongPhase
	}

	available := getAvailableActions(state.Variant)
	if !containsAction(available, action.Type) {
		return ErrActionNotAvailable
	}

	req := getActionRequirement(state.Variant, action.Type)

	if req.Cost > 0 && actor.Coins < req.Cost {
		return ErrNotEnoughCoins
	}

	if actor.Coins >= 10 && action.Type != ActionCoup {
		return ErrMustCoup
	}

	if req.NeedsTarget && action.TargetID == "" {
		return ErrNeedsTarget
	}

	if action.TargetID != "" {
		target := getPlayer(state, action.TargetID)
		if target == nil || !target.IsAlive {
			return ErrInvalidTarget
		}
		if target.ID == action.ActorID {
			return ErrSelfTarget
		}
	}

	// Deduct cost
	if req.Cost > 0 {
		actor.Coins -= req.Cost
	}

	// Set claimed character
	if req.Character != "" {
		action.ClaimedCharacter = req.Character
	}

	state.PendingAction = &action
	state.PassedPlayers = []string{}

	if req.Character != "" {
		state.Phase = PhaseChallengeWindow
		msg := actor.Name + " claims " + string(req.Character) + " to " + string(action.Type)
		if action.TargetID != "" {
			target := getPlayer(state, action.TargetID)
			if target != nil {
				if action.Type == ActionSteal {
					msg += " from " + target.Name
				} else if action.Type == ActionAssassinate || action.Type == ActionInterrogate {
					msg += " " + target.Name
				}
			}
		}
		addLog(state, msg, action.ActorID, string(action.Type), action.TargetID)
	} else if req.CanBeBlocked {
		state.Phase = PhaseBlockWindow
		addLog(state, actor.Name+" attempts "+string(action.Type), action.ActorID, string(action.Type), action.TargetID)
	} else {
		resolveAction(state)
	}

	return nil
}

func resolveAction(state *GameState) {
	if state.PendingAction == nil {
		return
	}

	action := state.PendingAction
	actor := getPlayer(state, action.ActorID)
	if actor == nil {
		return
	}

	switch action.Type {
	case ActionIncome:
		actor.Coins++
		addLog(state, actor.Name+" takes 1 coin (Income)", actor.ID, string(action.Type), "")

	case ActionForeignAid:
		actor.Coins += 2
		addLog(state, actor.Name+" takes 2 coins (Foreign Aid)", actor.ID, string(action.Type), "")

	case ActionCoup:
		if action.TargetID != "" {
			target := getPlayer(state, action.TargetID)
			if target != nil {
				addLog(state, actor.Name+" coups "+target.Name, actor.ID, string(action.Type), target.ID)
				state.PendingInfluenceLoss = target.ID
				state.Phase = PhaseLoseInfluence
				return
			}
		}

	case ActionTax:
		actor.Coins += 3
		addLog(state, actor.Name+" takes 3 coins (Tax)", actor.ID, string(action.Type), "")

	case ActionAssassinate:
		if action.TargetID != "" {
			target := getPlayer(state, action.TargetID)
			if target != nil {
				addLog(state, actor.Name+" assassinates "+target.Name, actor.ID, string(action.Type), target.ID)
				state.PendingInfluenceLoss = target.ID
				state.Phase = PhaseLoseInfluence
				return
			}
		}

	case ActionSteal:
		if action.TargetID != "" {
			target := getPlayer(state, action.TargetID)
			if target != nil {
				stolen := 2
				if target.Coins < stolen {
					stolen = target.Coins
				}
				target.Coins -= stolen
				actor.Coins += stolen
				addLog(state, actor.Name+" steals coins from "+target.Name, actor.ID, string(action.Type), target.ID)
			}
		}

	case ActionExchange:
		var drawn []Card
		for i := 0; i < 2 && len(state.CourtDeck) > 0; i++ {
			drawn = append(drawn, state.CourtDeck[len(state.CourtDeck)-1])
			state.CourtDeck = state.CourtDeck[:len(state.CourtDeck)-1]
		}
		state.PendingExchangeCards = drawn
		state.Phase = PhaseExchange
		addLog(state, actor.Name+" exchanges cards", actor.ID, string(action.Type), "")
		return

	case ActionInquire:
		var drawn []Card
		if len(state.CourtDeck) > 0 {
			drawn = append(drawn, state.CourtDeck[len(state.CourtDeck)-1])
			state.CourtDeck = state.CourtDeck[:len(state.CourtDeck)-1]
		}
		state.PendingExchangeCards = drawn
		state.Phase = PhaseExchange
		addLog(state, actor.Name+" inquires from the court", actor.ID, string(action.Type), "")
		return

	case ActionInterrogate:
		if action.TargetID != "" {
			target := getPlayer(state, action.TargetID)
			if target != nil {
				state.PendingInterrogate = &PendingInterrogate{TargetID: target.ID}
				state.Phase = PhaseInterrogateSelect
				addLog(state, actor.Name+" interrogates "+target.Name, actor.ID, string(action.Type), target.ID)
				return
			}
		}
	}

	state.PendingAction = nil
	endTurn(state)
}

// BlockAction processes a block
func BlockAction(state *GameState, block BlockRequest) error {
	if state.Phase != PhaseBlockWindow {
		return ErrWrongPhase
	}
	if state.PendingAction == nil {
		return ErrNoPendingAction
	}

	blocker := getPlayer(state, block.BlockerID)
	if blocker == nil || !blocker.IsAlive {
		return ErrInvalidPlayer
	}

	req := getActionRequirement(state.Variant, state.PendingAction.Type)
	if !req.CanBeBlocked {
		return ErrCannotBlock
	}
	if !containsChar(req.BlockingCharacters, block.ClaimedCharacter) {
		return ErrInvalidBlockChar
	}
	if state.PendingAction.ActorID == block.BlockerID {
		return ErrSelfBlock
	}

	state.PendingBlock = &block
	state.Phase = PhaseChallengeWindow
	state.PassedPlayers = []string{}
	addLog(state, blocker.Name+" claims "+string(block.ClaimedCharacter)+" to block", block.BlockerID, "", "")

	return nil
}

// PassBlock processes a pass on blocking
func PassBlock(state *GameState, playerID string) {
	if state.Phase != PhaseBlockWindow || state.PendingAction == nil {
		return
	}

	if !contains(state.PassedPlayers, playerID) {
		state.PassedPlayers = append(state.PassedPlayers, playerID)
	}

	allPassed := false
	alive := getAlivePlayers(state)

	if state.PendingAction.Type == ActionForeignAid {
		eligible := 0
		passed := 0
		for _, p := range alive {
			if p.ID != state.PendingAction.ActorID {
				eligible++
				if contains(state.PassedPlayers, p.ID) {
					passed++
				}
			}
		}
		allPassed = passed >= eligible
	} else {
		if state.PendingAction.TargetID != "" && contains(state.PassedPlayers, state.PendingAction.TargetID) {
			allPassed = true
		}
	}

	if allPassed {
		state.Phase = PhaseResolving
		state.PassedPlayers = []string{}
		resolveAction(state)
	}
}

// ChallengeAction processes a challenge
func ChallengeAction(state *GameState, challenge ChallengeRequest) error {
	challenger := getPlayer(state, challenge.ChallengerID)
	if challenger == nil || !challenger.IsAlive {
		return ErrInvalidPlayer
	}
	if challenge.ChallengerID == challenge.TargetPlayerID {
		return ErrSelfChallenge
	}
	if state.Phase != PhaseChallengeWindow {
		return ErrWrongPhase
	}

	state.PendingChallenge = &challenge
	target := getPlayer(state, challenge.TargetPlayerID)
	if target == nil {
		return ErrInvalidTarget
	}

	addLog(state, challenger.Name+" challenges "+target.Name+"'s "+string(challenge.ClaimedCharacter), challenge.ChallengerID, "", "")

	// Check if target has the claimed character
	hasChar := false
	cardIdx := -1
	for i, card := range target.Cards {
		if !card.Revealed && card.Character == challenge.ClaimedCharacter {
			hasChar = true
			cardIdx = i
			break
		}
	}

	if hasChar {
		// Challenge failed - challenger loses influence
		addLog(state, target.Name+" reveals "+string(challenge.ClaimedCharacter)+"! Challenge failed.", target.ID, "", "")
		state.PendingInfluenceLoss = challenger.ID
		state.Phase = PhaseLoseInfluence

		// Target reveals and reshuffles card
		if cardIdx >= 0 {
			revealedCard := target.Cards[cardIdx]
			target.Cards = append(target.Cards[:cardIdx], target.Cards[cardIdx+1:]...)
			state.CourtDeck = append(state.CourtDeck, revealedCard)
			state.CourtDeck = shuffleDeck(state.CourtDeck)

			if len(state.CourtDeck) > 0 {
				newCard := state.CourtDeck[len(state.CourtDeck)-1]
				state.CourtDeck = state.CourtDeck[:len(state.CourtDeck)-1]
				target.Cards = append(target.Cards, newCard)
			}
		}
	} else {
		// Challenge succeeded - target loses influence
		addLog(state, target.Name+" doesn't have "+string(challenge.ClaimedCharacter)+"! Challenge succeeded.", target.ID, "", "")
		state.PendingInfluenceLoss = target.ID
		state.Phase = PhaseLoseInfluence
	}

	return nil
}

// PassChallenge handles a player passing on challenging
func PassChallenge(state *GameState, playerID string) {
	if state.Phase != PhaseChallengeWindow {
		return
	}

	if !contains(state.PassedPlayers, playerID) {
		state.PassedPlayers = append(state.PassedPlayers, playerID)
	}

	alive := getAlivePlayers(state)
	var subjectID string

	if state.PendingBlock != nil {
		subjectID = state.PendingBlock.BlockerID
	} else if state.PendingAction != nil {
		subjectID = state.PendingAction.ActorID
	} else {
		return
	}

	eligible := 0
	passed := 0
	for _, p := range alive {
		if p.ID != subjectID {
			eligible++
			if contains(state.PassedPlayers, p.ID) {
				passed++
			}
		}
	}

	if passed >= eligible {
		state.PassedPlayers = []string{}
		if state.PendingBlock != nil {
			blocker := getPlayer(state, state.PendingBlock.BlockerID)
			if blocker != nil {
				addLog(state, blocker.Name+"'s block succeeds", state.PendingBlock.BlockerID, "", "")
			}
			state.PendingBlock = nil
			state.PendingAction = nil
			endTurn(state)
		} else {
			action := state.PendingAction
			req := getActionRequirement(state.Variant, action.Type)

			if req.CanBeBlocked {
				state.Phase = PhaseBlockWindow
			} else {
				resolveAction(state)
			}
		}
	}
}

// LoseInfluence handles a player losing a card
func LoseInfluence(state *GameState, playerID string, cardID string) {
	player := getPlayer(state, playerID)
	if player == nil {
		return
	}

	var card *Card
	for i := range player.Cards {
		if player.Cards[i].ID == cardID && !player.Cards[i].Revealed {
			card = &player.Cards[i]
			break
		}
	}
	if card == nil {
		// fallback: reveal first unrevealed
		for i := range player.Cards {
			if !player.Cards[i].Revealed {
				card = &player.Cards[i]
				break
			}
		}
	}

	if card == nil {
		return
	}

	card.Revealed = true
	addLog(state, player.Name+" loses influence ("+string(card.Character)+")", playerID, "", "")
	state.PendingInfluenceLoss = ""

	if getPlayerInfluence(player) == 0 {
		player.IsAlive = false
		addLog(state, player.Name+" is eliminated", playerID, "", "")

		alive := getAlivePlayers(state)
		if len(alive) == 1 {
			state.Winner = alive[0].ID
			state.Phase = PhaseGameOver
			addLog(state, alive[0].Name+" wins!", alive[0].ID, "", "")
			return
		}
	}

	// Continue based on context
	if state.PendingChallenge != nil {
		wasChallengeSuccessful := state.PendingChallenge.TargetPlayerID == playerID

		if state.PendingChallenge.IsBlockChallenge {
			if wasChallengeSuccessful {
				state.PendingBlock = nil
				state.PendingChallenge = nil
				resolveAction(state)
			} else {
				state.PendingBlock = nil
				state.PendingAction = nil
				state.PendingChallenge = nil
				endTurn(state)
			}
		} else {
			if wasChallengeSuccessful {
				state.PendingAction = nil
				state.PendingChallenge = nil
				endTurn(state)
			} else {
				state.PendingChallenge = nil
				actionType := state.PendingAction.Type
				req := getActionRequirement(state.Variant, actionType)

				if req.CanBeBlocked {
					state.Phase = PhaseBlockWindow
					state.PassedPlayers = []string{}
				} else {
					resolveAction(state)
				}
			}
		}
	} else if state.PendingAction != nil {
		state.PendingAction = nil
		endTurn(state)
	} else {
		endTurn(state)
	}
}

// ExchangeCards handles the exchange phase
func ExchangeCards(state *GameState, playerID string, keptCardIDs []string) error {
	if state.Phase != PhaseExchange {
		return ErrWrongPhase
	}

	player := getPlayer(state, playerID)
	cp := getCurrentPlayer(state)
	if player == nil || player.ID != cp.ID {
		return ErrInvalidPlayer
	}

	drawn := state.PendingExchangeCards

	// Combine unrevealed player cards with drawn cards
	var allCards []Card
	var revealedCards []Card
	for _, c := range player.Cards {
		if !c.Revealed {
			allCards = append(allCards, c)
		} else {
			revealedCards = append(revealedCards, c)
		}
	}
	allCards = append(allCards, drawn...)

	// Keep specified cards
	var kept []Card
	var returned []Card
	for _, c := range allCards {
		found := false
		for _, id := range keptCardIDs {
			if c.ID == id {
				found = true
				break
			}
		}
		if found {
			kept = append(kept, c)
		} else {
			returned = append(returned, c)
		}
	}

	influence := getPlayerInfluence(player)
	if len(kept) != influence {
		return ErrWrongCardCount
	}

	player.Cards = append(kept, revealedCards...)
	state.CourtDeck = append(state.CourtDeck, returned...)
	state.CourtDeck = shuffleDeck(state.CourtDeck)

	state.PendingAction = nil
	state.PendingExchangeCards = nil
	endTurn(state)

	return nil
}

// SelectInterrogateCard handles the target selecting a card for interrogation
func SelectInterrogateCard(state *GameState, playerID string, cardID string) error {
	if state.Phase != PhaseInterrogateSelect || state.PendingInterrogate == nil {
		return ErrWrongPhase
	}
	if state.PendingInterrogate.TargetID != playerID {
		return ErrInvalidPlayer
	}

	target := getPlayer(state, playerID)
	if target == nil {
		return ErrInvalidPlayer
	}

	found := false
	for _, c := range target.Cards {
		if c.ID == cardID && !c.Revealed {
			found = true
			break
		}
	}
	if !found {
		return ErrInvalidCard
	}

	state.PendingInterrogate.SelectedCardID = cardID
	state.Phase = PhaseInterrogateDecision
	addLog(state, target.Name+" selects a card for interrogation", target.ID, "interrogate", target.ID)

	return nil
}

// DecideInterrogate handles the inquisitor's keep/replace decision
func DecideInterrogate(state *GameState, playerID string, decision string) error {
	if state.Phase != PhaseInterrogateDecision || state.PendingInterrogate == nil || state.PendingAction == nil {
		return ErrWrongPhase
	}
	if state.PendingAction.ActorID != playerID {
		return ErrInvalidPlayer
	}

	actor := getPlayer(state, playerID)
	target := getPlayer(state, state.PendingInterrogate.TargetID)
	selectedCardID := state.PendingInterrogate.SelectedCardID

	if actor == nil || target == nil || selectedCardID == "" {
		state.PendingInterrogate = nil
		state.PendingAction = nil
		endTurn(state)
		return nil
	}

	selectedIdx := -1
	for i, c := range target.Cards {
		if c.ID == selectedCardID && !c.Revealed {
			selectedIdx = i
			break
		}
	}

	if selectedIdx == -1 {
		state.PendingInterrogate = nil
		state.PendingAction = nil
		endTurn(state)
		return nil
	}

	if decision == "replace" && len(state.CourtDeck) > 0 {
		removed := target.Cards[selectedIdx]
		target.Cards = append(target.Cards[:selectedIdx], target.Cards[selectedIdx+1:]...)
		state.CourtDeck = append(state.CourtDeck, removed)
		state.CourtDeck = shuffleDeck(state.CourtDeck)

		newCard := state.CourtDeck[len(state.CourtDeck)-1]
		state.CourtDeck = state.CourtDeck[:len(state.CourtDeck)-1]
		target.Cards = append(target.Cards, newCard)

		addLog(state, actor.Name+" replaces "+target.Name+"'s card", actor.ID, "interrogate", target.ID)
	} else {
		addLog(state, actor.Name+" allows "+target.Name+" to keep the card", actor.ID, "interrogate", target.ID)
	}

	state.PendingInterrogate = nil
	state.PendingAction = nil
	endTurn(state)

	return nil
}

// VoluntaryExit handles a player choosing to leave mid-game
func VoluntaryExit(state *GameState, playerID string) {
	player := getPlayer(state, playerID)
	if player == nil || !player.IsAlive {
		return
	}

	for i := range player.Cards {
		player.Cards[i].Revealed = true
	}
	player.IsAlive = false
	addLog(state, player.Name+" left the game", playerID, "", "")

	alive := getAlivePlayers(state)
	if len(alive) <= 1 {
		if len(alive) == 1 {
			state.Winner = alive[0].ID
			addLog(state, alive[0].Name+" wins!", alive[0].ID, "", "")
		}
		state.Phase = PhaseGameOver
		return
	}

	// Clean up pending state if exiting player was involved
	if getCurrentPlayer(state).ID == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		state.PendingExchangeCards = nil
		state.PendingInfluenceLoss = ""
		endTurn(state)
		return
	}

	if state.PendingAction != nil && state.PendingAction.TargetID == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		endTurn(state)
		return
	}

	if state.PendingBlock != nil && state.PendingBlock.BlockerID == playerID {
		state.PendingBlock = nil
		state.Phase = PhaseResolving
		resolveAction(state)
		return
	}

	if state.PendingInfluenceLoss == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		state.PendingInfluenceLoss = ""
		endTurn(state)
		return
	}
}

// EliminatePlayer removes a player (disconnect)
func EliminatePlayer(state *GameState, playerID string) {
	player := getPlayer(state, playerID)
	if player == nil || !player.IsAlive {
		return
	}

	for i := range player.Cards {
		player.Cards[i].Revealed = true
	}
	player.IsAlive = false
	addLog(state, player.Name+" disconnected and was eliminated", playerID, "", "")

	alive := getAlivePlayers(state)
	if len(alive) <= 1 {
		if len(alive) == 1 {
			state.Winner = alive[0].ID
			addLog(state, alive[0].Name+" wins!", alive[0].ID, "", "")
		}
		state.Phase = PhaseGameOver
		return
	}

	// Clean up pending state if disconnected player was involved
	if getCurrentPlayer(state).ID == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		state.PendingExchangeCards = nil
		state.PendingInfluenceLoss = ""
		endTurn(state)
		return
	}

	if state.PendingAction != nil && state.PendingAction.TargetID == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		endTurn(state)
		return
	}

	if state.PendingBlock != nil && state.PendingBlock.BlockerID == playerID {
		state.PendingBlock = nil
		state.Phase = PhaseResolving
		resolveAction(state)
		return
	}

	if state.PendingInfluenceLoss == playerID {
		state.PendingAction = nil
		state.PendingBlock = nil
		state.PendingChallenge = nil
		state.PendingInfluenceLoss = ""
		endTurn(state)
		return
	}
}

func endTurn(state *GameState) {
	alive := getAlivePlayers(state)
	if len(alive) <= 1 {
		state.Phase = PhaseGameOver
		return
	}

	nextIdx := (state.CurrentPlayerIndex + 1) % len(state.Players)
	for !state.Players[nextIdx].IsAlive {
		nextIdx = (nextIdx + 1) % len(state.Players)
	}

	state.CurrentPlayerIndex = nextIdx
	state.Turn++
	state.Phase = PhaseAction
	state.PendingAction = nil
	state.PendingBlock = nil
	state.PendingChallenge = nil
	state.PassedPlayers = []string{}
	state.PendingInterrogate = nil

	if len(state.CourtDeck) == 0 && len(state.DiscardPile) > 0 {
		state.CourtDeck = shuffleDeck(state.DiscardPile)
		state.DiscardPile = nil
	}
}
