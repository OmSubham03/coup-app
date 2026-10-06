package game

import (
	"fmt"
	"math/rand"
	"time"
)

type UNOPhase string

const (
	UNOPhasePlaying   UNOPhase = "playing"
	UNOPhaseRoundOver UNOPhase = "round_over"
	UNOPhaseGameOver  UNOPhase = "game_over"
)

type UNOCard struct {
	ID    string `json:"id"`
	Color string `json:"color"`
	Value string `json:"value"`
	Kind  string `json:"kind"`
}

type UNOPlayer struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Cards       []UNOCard `json:"cards"`
	SaidUNO     bool      `json:"saidUno"`
	HasDrawn    bool      `json:"hasDrawn"`
	DrawnCardID string    `json:"drawnCardId,omitempty"`
	Score       int       `json:"score"`
}

type UNOState struct {
	ID                          string      `json:"id"`
	Players                     []UNOPlayer `json:"players"`
	Phase                       UNOPhase    `json:"phase"`
	CurrentPlayerIdx            int         `json:"currentPlayerIdx"`
	Direction                   int         `json:"direction"`
	DrawPile                    []UNOCard   `json:"drawPile,omitempty"`
	DiscardPile                 []UNOCard   `json:"discardPile"`
	CurrentColor                string      `json:"currentColor"`
	PendingDraw                 int         `json:"pendingDraw"`
	StackingEnabled             bool        `json:"stackingEnabled"`
	MultiSkipEnabled            bool        `json:"multiSkipEnabled"`
	PendingSkips                int         `json:"pendingSkips"`
	PendingWinnerID             string      `json:"pendingWinnerId,omitempty"`
	UncalledUNOPlayerID         string      `json:"uncalledUnoPlayerId,omitempty"`
	ChallengeAvailable          bool        `json:"challengeAvailable"`
	PendingWildDrawFourPlayerID string      `json:"-"`
	PendingWildDrawFourIllegal  bool        `json:"-"`
	AwaitingInitialColor        bool        `json:"awaitingInitialColor"`
	Round                       int         `json:"round"`
	LastAction                  string      `json:"lastAction"`
	WinnerID                    string      `json:"winnerId,omitempty"`
	WinnerName                  string      `json:"winnerName,omitempty"`
	TurnNumber                  int         `json:"turnNumber"`
}

// UNO stack types are house rules; the base game does not allow stacking draw cards.
func UNOCanStack(previous, next UNOCard, currentColor string) bool {
	if next.Kind != "draw_two" && next.Kind != "wild_draw_four" {
		return false
	}
	if previous.Kind == "draw_two" {
		return next.Kind == "draw_two" || next.Kind == "wild_draw_four"
	}
	if previous.Kind == "wild_draw_four" {
		return next.Kind == "wild_draw_four" || (next.Kind == "draw_two" && next.Color == currentColor)
	}
	return false
}

func UNOCreateDeck() []UNOCard {
	colors := []string{"red", "yellow", "green", "blue"}
	deck := make([]UNOCard, 0, 108)
	id := 0
	add := func(color, value, kind string) {
		deck = append(deck, UNOCard{ID: fmt.Sprintf("uno_%d", id), Color: color, Value: value, Kind: kind})
		id++
	}
	for _, color := range colors {
		add(color, "0", "number")
		for value := 1; value <= 9; value++ {
			add(color, fmt.Sprint(value), "number")
			add(color, fmt.Sprint(value), "number")
		}
		for i := 0; i < 2; i++ {
			add(color, "skip", "skip")
			add(color, "reverse", "reverse")
			add(color, "+2", "draw_two")
		}
	}
	for i := 0; i < 4; i++ {
		add("wild", "wild", "wild")
		add("wild", "+4", "wild_draw_four")
	}
	return deck
}

func UNOInitializeGame(players []struct{ ID, Name string }, stackingEnabled bool) (*UNOState, error) {
	if len(players) < 2 || len(players) > 6 {
		return nil, fmt.Errorf("UNO requires 2 to 6 players")
	}
	state := &UNOState{
		ID:              fmt.Sprintf("uno_%d", time.Now().UnixNano()),
		Players:         make([]UNOPlayer, len(players)),
		Phase:           UNOPhasePlaying,
		Direction:       1,
		StackingEnabled: stackingEnabled,
		DiscardPile:     make([]UNOCard, 0),
		Round:           1,
		LastAction:      "Game started",
	}
	for i, player := range players {
		state.Players[i] = UNOPlayer{ID: player.ID, Name: player.Name, Cards: make([]UNOCard, 0, 8)}
	}
	deck := UNOCreateDeck()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for round := 0; round < 7; round++ {
		for i := range state.Players {
			state.Players[i].Cards = append(state.Players[i].Cards, deck[0])
			deck = deck[1:]
		}
	}
	state.DrawPile = deck
	unoStartWithDiscard(state)
	return state, nil
}

func unoStartWithDiscard(state *UNOState) {
	for len(state.DrawPile) > 0 {
		card := state.DrawPile[0]
		state.DrawPile = state.DrawPile[1:]
		if card.Kind == "wild_draw_four" {
			state.DrawPile = append(state.DrawPile, card)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			r.Shuffle(len(state.DrawPile), func(i, j int) { state.DrawPile[i], state.DrawPile[j] = state.DrawPile[j], state.DrawPile[i] })
			continue
		}
		state.DiscardPile = append(state.DiscardPile, card)
		state.CurrentColor = card.Color
		switch card.Kind {
		case "wild":
			state.CurrentColor = ""
			state.AwaitingInitialColor = true
		case "skip":
			state.CurrentPlayerIdx = 1 % len(state.Players)
		case "reverse":
			state.Direction = -1
			if len(state.Players) == 2 {
				state.CurrentPlayerIdx = 1
			} else {
				state.CurrentPlayerIdx = len(state.Players) - 1
			}
		case "draw_two":
			state.PendingDraw = 2
			state.CurrentPlayerIdx = 1 % len(state.Players)
		}
		return
	}
}

func UNOTopCard(state *UNOState) UNOCard {
	if len(state.DiscardPile) == 0 {
		return UNOCard{}
	}
	return state.DiscardPile[len(state.DiscardPile)-1]
}

func UNOPlayable(state *UNOState, card UNOCard) bool {
	if state.PendingSkips > 0 {
		return state.MultiSkipEnabled && card.Kind == "skip"
	}
	if state.PendingDraw > 0 {
		return state.StackingEnabled && UNOCanStack(UNOTopCard(state), card, state.CurrentColor)
	}
	return card.Kind == "wild" || card.Kind == "wild_draw_four" || card.Color == state.CurrentColor || card.Value == UNOTopCard(state).Value
}

func UNOFindPlayer(state *UNOState, playerID string) int {
	for i := range state.Players {
		if state.Players[i].ID == playerID {
			return i
		}
	}
	return -1
}

func UNOSanitizeState(state *UNOState, playerID string) *UNOState {
	if state == nil {
		return nil
	}
	copyState := *state
	copyState.DrawPile = nil
	copyState.Players = make([]UNOPlayer, len(state.Players))
	for i, player := range state.Players {
		copyState.Players[i] = player
		if player.ID != playerID {
			copyState.Players[i].DrawnCardID = ""
			copyState.Players[i].Cards = make([]UNOCard, len(player.Cards))
			for j := range player.Cards {
				copyState.Players[i].Cards[j] = UNOCard{ID: fmt.Sprintf("hidden_%d_%d", i, j)}
			}
		}
	}
	copyState.DiscardPile = append([]UNOCard(nil), state.DiscardPile...)
	return &copyState
}

func unoPlayerCanDraw(state *UNOState, playerIdx int) error {
	if state.Phase != UNOPhasePlaying {
		return fmt.Errorf("game is not in progress")
	}
	if playerIdx < 0 || playerIdx != state.CurrentPlayerIdx {
		return fmt.Errorf("not your turn")
	}
	return nil
}

func UNODrawCard(state *UNOState, playerID string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	if state.PendingDraw > 0 {
		return fmt.Errorf("accept the pending draw penalty")
	}
	if state.PendingSkips > 0 {
		return fmt.Errorf("play a Skip to continue the multi-skip chain")
	}
	if state.Players[idx].HasDrawn {
		return fmt.Errorf("you already drew this turn")
	}
	unoCloseCallWindow(state)
	card, err := unoTakeDrawCard(state)
	if err != nil {
		return err
	}
	state.Players[idx].Cards = append(state.Players[idx].Cards, card)
	state.Players[idx].HasDrawn = true
	state.Players[idx].DrawnCardID = card.ID
	state.Players[idx].SaidUNO = false
	state.LastAction = fmt.Sprintf("%s drew a card", state.Players[idx].Name)
	return nil
}

func UNOPlayCard(state *UNOState, playerID, cardID, chosenColor string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	player := &state.Players[idx]
	cardIdx := -1
	for i, card := range player.Cards {
		if card.ID == cardID {
			cardIdx = i
			break
		}
	}
	if cardIdx < 0 {
		return fmt.Errorf("card not found in your hand")
	}
	card := player.Cards[cardIdx]
	previousPending := state.PendingDraw
	previousColor := state.CurrentColor
	if !UNOPlayable(state, card) {
		return fmt.Errorf("card cannot be played on the current discard")
	}
	if (card.Kind == "wild" || card.Kind == "wild_draw_four") && !unoValidColor(chosenColor) {
		return fmt.Errorf("choose a valid color for the wild card")
	}
	wildDrawFourIllegal := false
	if card.Kind == "wild_draw_four" {
		for _, held := range player.Cards {
			if held.ID != card.ID && held.Color == previousColor {
				wildDrawFourIllegal = true
			}
		}
		if previousPending == 0 && wildDrawFourIllegal {
			return fmt.Errorf("Wild Draw Four can only be played when you have no card matching the current color")
		}
	}
	if state.PendingDraw > 0 && (!state.StackingEnabled || !UNOCanStack(UNOTopCard(state), card, state.CurrentColor)) {
		return fmt.Errorf("that draw card cannot be stacked")
	}
	if player.HasDrawn && state.PendingDraw == 0 {
		// The rules permit playing only the just-drawn card after drawing.
		if player.DrawnCardID == "" || card.ID != player.DrawnCardID {
			return fmt.Errorf("after drawing, you may play only the drawn card")
		}
	}
	unoCloseCallWindow(state)

	player.Cards = append(player.Cards[:cardIdx], player.Cards[cardIdx+1:]...)
	state.DiscardPile = append(state.DiscardPile, card)
	state.ChallengeAvailable = card.Kind == "wild_draw_four" && previousPending == 0
	state.PendingWildDrawFourPlayerID = ""
	state.PendingWildDrawFourIllegal = false
	if state.ChallengeAvailable {
		state.PendingWildDrawFourPlayerID = player.ID
		state.PendingWildDrawFourIllegal = wildDrawFourIllegal
	}
	if card.Color == "wild" {
		state.CurrentColor = chosenColor
	} else {
		state.CurrentColor = card.Color
	}
	player.HasDrawn = false
	player.DrawnCardID = ""
	if len(player.Cards) == 1 {
		player.SaidUNO = false
		state.UncalledUNOPlayerID = player.ID
	} else if len(player.Cards) == 0 {
		if card.Kind == "draw_two" || card.Kind == "wild_draw_four" {
			state.PendingWinnerID = player.ID
		} else {
			unoFinishRound(state, player.ID)
		}
	}
	state.LastAction = fmt.Sprintf("%s played %s", player.Name, card.Value)

	if state.Phase == UNOPhaseGameOver {
		return nil
	}
	next := idx
	switch card.Kind {
	case "skip":
		if state.MultiSkipEnabled && state.Phase == UNOPhasePlaying {
			state.PendingSkips++
			next = unoAdvance(state, idx, 1)
		} else {
			next = unoAdvance(state, idx, 2)
		}
	case "reverse":
		if len(state.Players) == 2 {
			// In a two-player game Reverse acts like Skip.
			next = idx
		} else {
			state.Direction *= -1
			next = unoAdvance(state, idx, 1)
		}
	case "draw_two":
		state.PendingDraw += 2
		state.ChallengeAvailable = false
		next = unoAdvance(state, idx, 1)
	case "wild_draw_four":
		state.PendingDraw += 4
		if previousPending > 0 {
			state.ChallengeAvailable = false
		}
		next = unoAdvance(state, idx, 1)
	default:
		next = unoAdvance(state, idx, 1)
	}
	state.CurrentPlayerIdx = next
	state.TurnNumber++
	unoResolveSkipChain(state)
	return nil
}

func unoResolveSkipChain(state *UNOState) {
	if state.PendingSkips == 0 || state.Phase != UNOPhasePlaying {
		return
	}
	idx := state.CurrentPlayerIdx
	if unoHasSkip(state.Players[idx]) {
		return
	}
	amount := state.PendingSkips
	state.CurrentPlayerIdx = unoAdvance(state, idx, amount)
	state.PendingSkips = 0
	state.LastAction += fmt.Sprintf("; %d turns skipped starting with %s", amount, state.Players[idx].Name)
}

func UNOSayUNO(state *UNOState, playerID string) error {
	idx := UNOFindPlayer(state, playerID)
	if idx < 0 {
		return fmt.Errorf("player not found")
	}
	if len(state.Players[idx].Cards) != 1 || state.UncalledUNOPlayerID != playerID {
		return fmt.Errorf("you can call UNO only immediately after playing down to one card")
	}
	state.Players[idx].SaidUNO = true
	state.UncalledUNOPlayerID = ""
	state.LastAction = fmt.Sprintf("%s called UNO", state.Players[idx].Name)
	return nil
}

func UNOCatchUNO(state *UNOState, playerID, targetPlayerID string) error {
	callerIdx := UNOFindPlayer(state, playerID)
	targetIdx := UNOFindPlayer(state, targetPlayerID)
	if callerIdx < 0 || targetIdx < 0 || callerIdx == targetIdx {
		return fmt.Errorf("player not found")
	}
	target := &state.Players[targetIdx]
	if len(target.Cards) != 1 || target.SaidUNO || state.UncalledUNOPlayerID != targetPlayerID {
		return fmt.Errorf("that player cannot be caught")
	}
	target.Cards = append(target.Cards, unoTakeTwo(state)...)
	target.SaidUNO = true
	state.UncalledUNOPlayerID = ""
	state.LastAction = fmt.Sprintf("%s caught %s without calling UNO", state.Players[callerIdx].Name, target.Name)
	return nil
}

func UNOAcceptDrawPenalty(state *UNOState, playerID string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	if state.PendingDraw == 0 {
		return fmt.Errorf("there is no pending draw penalty")
	}
	unoCloseCallWindow(state)
	return unoAcceptPenalty(state, idx)
}

func UNOEndTurn(state *UNOState, playerID string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	if state.PendingDraw > 0 {
		return fmt.Errorf("accept the pending draw penalty first")
	}
	if state.PendingSkips > 0 {
		return fmt.Errorf("play a Skip to continue the multi-skip chain")
	}
	if !state.Players[idx].HasDrawn {
		return fmt.Errorf("draw a card before ending your turn")
	}
	unoCloseCallWindow(state)
	state.Players[idx].HasDrawn = false
	state.Players[idx].DrawnCardID = ""
	state.CurrentPlayerIdx = unoAdvance(state, idx, 1)
	state.TurnNumber++
	state.LastAction = fmt.Sprintf("%s ended their turn", state.Players[idx].Name)
	return nil
}

func unoAcceptPenalty(state *UNOState, idx int) error {
	amount := state.PendingDraw
	if amount <= 0 {
		return fmt.Errorf("there is no pending draw penalty")
	}
	cards, err := unoTakeCards(state, amount)
	if err != nil {
		return err
	}
	state.Players[idx].Cards = append(state.Players[idx].Cards, cards...)
	state.Players[idx].HasDrawn = false
	state.Players[idx].DrawnCardID = ""
	state.PendingDraw = 0
	state.ChallengeAvailable = false
	state.PendingWildDrawFourPlayerID = ""
	state.PendingWildDrawFourIllegal = false
	if state.PendingWinnerID != "" {
		winnerID := state.PendingWinnerID
		state.PendingWinnerID = ""
		unoFinishRound(state, winnerID)
		return nil
	}
	state.CurrentPlayerIdx = unoAdvance(state, idx, 1)
	state.TurnNumber++
	state.LastAction = fmt.Sprintf("%s drew %d cards", state.Players[idx].Name, amount)
	return nil
}

func unoCloseCallWindow(state *UNOState) {
	state.UncalledUNOPlayerID = ""
}

func UNOChallengeWildDrawFour(state *UNOState, playerID string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	if !state.ChallengeAvailable || state.PendingDraw != 4 {
		return fmt.Errorf("there is no Wild Draw Four available to challenge")
	}
	unoCloseCallWindow(state)
	challengedPlayerIdx := UNOFindPlayer(state, state.PendingWildDrawFourPlayerID)
	if challengedPlayerIdx < 0 {
		return fmt.Errorf("the Wild Draw Four player was not found")
	}
	state.ChallengeAvailable = false
	state.PendingDraw = 0
	state.PendingWildDrawFourPlayerID = ""
	if state.PendingWildDrawFourIllegal {
		cards, err := unoTakeCards(state, 4)
		if err != nil {
			return err
		}
		state.Players[challengedPlayerIdx].Cards = append(state.Players[challengedPlayerIdx].Cards, cards...)
		state.PendingWildDrawFourIllegal = false
		if state.PendingWinnerID == state.Players[challengedPlayerIdx].ID {
			state.PendingWinnerID = ""
		}
		state.LastAction = fmt.Sprintf("%s successfully challenged the Wild Draw Four", state.Players[idx].Name)
		return nil
	}
	cards, err := unoTakeCards(state, 6)
	if err != nil {
		return err
	}
	state.Players[idx].Cards = append(state.Players[idx].Cards, cards...)
	state.PendingWildDrawFourIllegal = false
	state.CurrentPlayerIdx = unoAdvance(state, idx, 1)
	state.TurnNumber++
	state.LastAction = fmt.Sprintf("%s failed the Wild Draw Four challenge and drew 6 cards", state.Players[idx].Name)
	return nil
}

func UNOChooseInitialColor(state *UNOState, playerID, color string) error {
	idx := UNOFindPlayer(state, playerID)
	if err := unoPlayerCanDraw(state, idx); err != nil {
		return err
	}
	if !state.AwaitingInitialColor {
		return fmt.Errorf("the opening card does not need a color choice")
	}
	if !unoValidColor(color) {
		return fmt.Errorf("choose a valid color")
	}
	state.CurrentColor = color
	state.AwaitingInitialColor = false
	state.LastAction = fmt.Sprintf("%s chose %s", state.Players[idx].Name, color)
	return nil
}

func UNOStartNextRound(state *UNOState) error {
	if state.Phase != UNOPhaseRoundOver {
		return fmt.Errorf("the round is not over")
	}
	scores := make([]int, len(state.Players))
	players := make([]struct{ ID, Name string }, len(state.Players))
	for i, player := range state.Players {
		scores[i] = player.Score
		players[i] = struct{ ID, Name string }{player.ID, player.Name}
	}
	stackingEnabled := state.StackingEnabled
	next, err := UNOInitializeGame(players, stackingEnabled)
	if err != nil {
		return err
	}
	next.ID = state.ID
	next.MultiSkipEnabled = state.MultiSkipEnabled
	next.Round = state.Round + 1
	for i := range next.Players {
		next.Players[i].Score = scores[i]
	}
	*state = *next
	return nil
}

func UNOForfeitPlayer(state *UNOState, playerID string) error {
	if state.Phase != UNOPhasePlaying && state.Phase != UNOPhaseRoundOver {
		return fmt.Errorf("game is already over")
	}
	idx := UNOFindPlayer(state, playerID)
	if idx < 0 {
		return fmt.Errorf("player not found")
	}
	name := state.Players[idx].Name
	state.Players = append(state.Players[:idx], state.Players[idx+1:]...)
	if len(state.Players) == 1 {
		state.Phase = UNOPhaseGameOver
		state.PendingSkips = 0
		state.WinnerID = state.Players[0].ID
		state.WinnerName = state.Players[0].Name
		state.LastAction = fmt.Sprintf("%s forfeited; %s wins", name, state.WinnerName)
		return nil
	}
	if len(state.Players) == 0 {
		state.Phase = UNOPhaseGameOver
		state.PendingSkips = 0
		state.LastAction = fmt.Sprintf("%s forfeited", name)
		return nil
	}
	if idx < state.CurrentPlayerIdx {
		state.CurrentPlayerIdx--
	} else if idx == state.CurrentPlayerIdx {
		if state.PendingSkips > 0 && state.Direction < 0 {
			state.CurrentPlayerIdx = (idx - 1 + len(state.Players)) % len(state.Players)
		} else {
			state.CurrentPlayerIdx %= len(state.Players)
		}
	}
	if state.UncalledUNOPlayerID == playerID {
		state.UncalledUNOPlayerID = ""
	}
	if state.PendingWildDrawFourPlayerID == playerID {
		state.ChallengeAvailable = false
		state.PendingWildDrawFourPlayerID = ""
		state.PendingWildDrawFourIllegal = false
	}
	state.LastAction = fmt.Sprintf("%s left the game", name)
	unoResolveSkipChain(state)
	return nil
}

func unoFinishRound(state *UNOState, winnerID string) {
	winnerIdx := UNOFindPlayer(state, winnerID)
	if winnerIdx < 0 {
		return
	}
	state.PendingSkips = 0
	points := 0
	for i := range state.Players {
		if i == winnerIdx {
			continue
		}
		for _, card := range state.Players[i].Cards {
			points += unoCardPoints(card)
		}
	}
	state.Players[winnerIdx].Score += points
	state.WinnerID = state.Players[winnerIdx].ID
	state.WinnerName = state.Players[winnerIdx].Name
	state.PendingWinnerID = ""
	if state.Players[winnerIdx].Score >= 500 {
		state.Phase = UNOPhaseGameOver
		state.LastAction = fmt.Sprintf("%s wins the game with %d points", state.Players[winnerIdx].Name, state.Players[winnerIdx].Score)
	} else {
		state.Phase = UNOPhaseRoundOver
		state.LastAction = fmt.Sprintf("%s wins round %d and scores %d points", state.Players[winnerIdx].Name, state.Round, points)
	}
}

func unoCardPoints(card UNOCard) int {
	if card.Kind == "number" {
		var points int
		fmt.Sscan(card.Value, &points)
		return points
	}
	if card.Kind == "wild" || card.Kind == "wild_draw_four" {
		return 50
	}
	return 20
}

func unoTakeDrawCard(state *UNOState) (UNOCard, error) {
	cards, err := unoTakeCards(state, 1)
	if err != nil {
		return UNOCard{}, err
	}
	return cards[0], nil
}

func unoTakeTwo(state *UNOState) []UNOCard {
	cards, err := unoTakeCards(state, 2)
	if err != nil {
		return nil
	}
	return cards
}

func unoTakeCards(state *UNOState, count int) ([]UNOCard, error) {
	if len(state.DrawPile) < count {
		if len(state.DiscardPile) <= 1 {
			return nil, fmt.Errorf("not enough cards to draw")
		}
		top := state.DiscardPile[len(state.DiscardPile)-1]
		state.DrawPile = append(state.DrawPile, state.DiscardPile[:len(state.DiscardPile)-1]...)
		state.DiscardPile = []UNOCard{top}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(state.DrawPile), func(i, j int) { state.DrawPile[i], state.DrawPile[j] = state.DrawPile[j], state.DrawPile[i] })
	}
	if len(state.DrawPile) < count {
		return nil, fmt.Errorf("not enough cards to draw")
	}
	cards := append([]UNOCard(nil), state.DrawPile[:count]...)
	state.DrawPile = state.DrawPile[count:]
	return cards, nil
}

func unoAdvance(state *UNOState, from, steps int) int {
	idx := from
	for i := 0; i < steps; i++ {
		idx = (idx + state.Direction + len(state.Players)) % len(state.Players)
	}
	return idx
}

func unoHasSkip(player UNOPlayer) bool {
	for _, card := range player.Cards {
		if card.Kind == "skip" {
			return true
		}
	}
	return false
}

func unoValidColor(color string) bool {
	return color == "red" || color == "yellow" || color == "green" || color == "blue"
}
