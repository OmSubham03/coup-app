package game

import (
	"errors"
	"math/rand"

	"github.com/google/uuid"
)

const (
	KaboInitial   = "initial"
	KaboDraw      = "draw"
	KaboResolve   = "resolve"
	KaboPower     = "power"
	KaboPeek      = "peek"
	KaboDecision  = "decision"
	KaboRoundOver = "round_over"
	KaboGameOver  = "game_over"
)

type KaboCard struct {
	Rank int    `json:"rank"`
	Suit string `json:"suit"`
}

func (c KaboCard) Value() int {
	if c.Rank == 13 && (c.Suit == "hearts" || c.Suit == "diamonds") {
		return -1
	}
	return c.Rank
}

type KaboSlot struct {
	ID   string `json:"id"`
	card KaboCard
}

type KaboPlayer struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Active     bool       `json:"active"`
	Ready      bool       `json:"ready"`
	TotalScore int        `json:"totalScore"`
	RoundScore int        `json:"roundScore"`
	Slots      []KaboSlot `json:"-"`
	initial    []string
}

type KaboReveal struct {
	PlayerID string   `json:"playerId"`
	SlotID   string   `json:"slotId"`
	Card     KaboCard `json:"card"`
}

type KaboState struct {
	ID               string       `json:"id"`
	Revision         int          `json:"revision"`
	Round            int          `json:"round"`
	Phase            string       `json:"phase"`
	Players          []KaboPlayer `json:"-"`
	CurrentPlayerIdx int          `json:"currentPlayerIdx"`
	CallerID         string       `json:"callerId"`
	FinalTurns       int          `json:"finalTurns"`
	Power            string       `json:"power"`
	MatchOpen        bool         `json:"matchOpen"`
	DiscardID        string       `json:"discardId"`
	LastAction       string       `json:"lastAction"`
	Winners          []string     `json:"winners"`
	RoundWinners     []string     `json:"roundWinners"`
	deck             []KaboCard
	discards         []KaboCard
	drawn            *KaboCard
	drawnFromDeck    bool
	reveals          []KaboReveal
	powerTargets     []KaboReveal
	failedMatches    map[string]bool
	finalPending     map[string]bool
}

type KaboAction struct {
	Revision  int      `json:"revision"`
	SlotIDs   []string `json:"slotIds"`
	SlotID    string   `json:"slotId"`
	TargetID  string   `json:"targetId"`
	Source    string   `json:"source"`
	DiscardID string   `json:"discardId"`
	Swap      bool     `json:"swap"`
}

func NewKabo(players []KaboPlayer) (*KaboState, error) {
	if len(players) < 2 || len(players) > 6 {
		return nil, errors.New("Kabo requires 2 to 6 players")
	}
	s := &KaboState{ID: uuid.NewString(), Players: append([]KaboPlayer{}, players...)}
	for i := range s.Players {
		p := &s.Players[i]
		if p.ID == "" {
			return nil, ErrInvalidPlayer
		}
		for j := 0; j < i; j++ {
			if s.Players[j].ID == p.ID {
				return nil, ErrInvalidPlayer
			}
		}
		p.Active = true
		p.TotalScore = 0
	}
	s.deal()
	return s, nil
}

func (s *KaboState) PlayerIndex(id string) int {
	for i := range s.Players {
		if s.Players[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *KaboState) deal() {
	s.deck = nil
	for _, suit := range []string{"clubs", "diamonds", "hearts", "spades"} {
		for rank := 1; rank <= 13; rank++ {
			s.deck = append(s.deck, KaboCard{Rank: rank, Suit: suit})
		}
	}
	rand.Shuffle(len(s.deck), func(i, j int) { s.deck[i], s.deck[j] = s.deck[j], s.deck[i] })
	s.Round++
	s.Phase, s.CallerID, s.Power = KaboInitial, "", ""
	s.FinalTurns, s.CurrentPlayerIdx = 0, -1
	s.finalPending = nil
	s.MatchOpen = false
	s.drawn, s.reveals, s.powerTargets = nil, nil, nil
	s.Winners, s.RoundWinners = []string{}, []string{}
	for i := range s.Players {
		p := &s.Players[i]
		p.Ready, p.RoundScore, p.initial = false, 0, nil
		p.Slots = []KaboSlot{}
		if !p.Active {
			continue
		}
		for j := 0; j < 4; j++ {
			p.Slots = append(p.Slots, KaboSlot{ID: uuid.NewString(), card: s.deck[0]})
			s.deck = s.deck[1:]
		}
	}
	s.discards = []KaboCard{s.deck[0]}
	s.deck = s.deck[1:]
	s.DiscardID = uuid.NewString()
	s.failedMatches = map[string]bool{}
	s.Revision++
	s.LastAction = "Choose two cards to peek, then confirm ready."
}

func (s *KaboState) slot(id string) (*KaboPlayer, *KaboSlot) {
	for i := range s.Players {
		if !s.Players[i].Active {
			continue
		}
		for j := range s.Players[i].Slots {
			if s.Players[i].Slots[j].ID == id && s.Players[i].Slots[j].card.Rank != 0 {
				return &s.Players[i], &s.Players[i].Slots[j]
			}
		}
	}
	return nil, nil
}

func (s *KaboState) next(i int) int {
	for n := 1; n <= len(s.Players); n++ {
		j := (i + n) % len(s.Players)
		if s.Players[j].Active {
			return j
		}
	}
	return -1
}

func (s *KaboState) takeCard() (KaboCard, error) {
	if len(s.deck) == 0 {
		if len(s.discards) <= 1 {
			return KaboCard{}, errors.New("No cards remain to draw")
		}
		s.deck = append([]KaboCard{}, s.discards[:len(s.discards)-1]...)
		s.discards = s.discards[len(s.discards)-1:]
		rand.Shuffle(len(s.deck), func(i, j int) { s.deck[i], s.deck[j] = s.deck[j], s.deck[i] })
	}
	card := s.deck[0]
	s.deck = s.deck[1:]
	return card, nil
}

func (s *KaboState) putDiscard(card KaboCard) {
	s.discards = append(s.discards, card)
	s.DiscardID = uuid.NewString()
	s.failedMatches = map[string]bool{}
}

func (s *KaboState) finishTurn() {
	s.drawn, s.reveals, s.powerTargets = nil, nil, nil
	s.Power = ""
	if s.CallerID != "" && s.Players[s.CurrentPlayerIdx].ID != s.CallerID {
		delete(s.finalPending, s.Players[s.CurrentPlayerIdx].ID)
		s.FinalTurns = len(s.finalPending)
		if s.FinalTurns <= 0 {
			s.finishRound()
			return
		}
	}
	s.CurrentPlayerIdx = s.nextTurn(s.CurrentPlayerIdx)
	s.Phase, s.MatchOpen = KaboDraw, len(s.discards) > 0
}

func (s *KaboState) nextTurn(from int) int {
	for n := 1; n <= len(s.Players); n++ {
		i := (from + n) % len(s.Players)
		if s.Players[i].Active && (s.CallerID == "" || s.finalPending[s.Players[i].ID]) {
			return i
		}
	}
	return -1
}

func (s *KaboState) finishRound() {
	if s.Phase == KaboRoundOver || s.Phase == KaboGameOver {
		return
	}
	s.Phase, s.MatchOpen = KaboRoundOver, false
	s.drawn, s.reveals, s.powerTargets = nil, nil, nil
	s.Power = ""
	best, found := 0, false
	s.RoundWinners = []string{}
	for i := range s.Players {
		p := &s.Players[i]
		if !p.Active {
			continue
		}
		p.RoundScore = 0
		for _, slot := range p.Slots {
			p.RoundScore += slot.card.Value()
		}
		p.TotalScore += p.RoundScore
		if !found || p.RoundScore < best {
			best, found, s.RoundWinners = p.RoundScore, true, []string{p.ID}
		} else if p.RoundScore == best {
			s.RoundWinners = append(s.RoundWinners, p.ID)
		}
	}
	for _, p := range s.Players {
		if p.Active && p.TotalScore >= 100 {
			s.endMatch()
			break
		}
	}
	s.LastAction = "Round complete. All cards revealed."
}

func (s *KaboState) endMatch() {
	s.Phase, s.MatchOpen = KaboGameOver, false
	best, found := 0, false
	s.Winners = []string{}
	for _, p := range s.Players {
		if !p.Active {
			continue
		}
		if !found || p.TotalScore < best {
			best, found, s.Winners = p.TotalScore, true, []string{p.ID}
		} else if p.TotalScore == best {
			s.Winners = append(s.Winners, p.ID)
		}
	}
}

func (s *KaboState) InitialPeek(id string, slots []string) error {
	i := s.PlayerIndex(id)
	if s.Phase != KaboInitial || i < 0 || !s.Players[i].Active || s.Players[i].Ready {
		return ErrWrongPhase
	}
	if len(s.Players[i].initial) != 0 {
		return errors.New("Initial cards have already been peeked")
	}
	if len(slots) != 2 || slots[0] == slots[1] {
		return errors.New("Choose exactly two different cards")
	}
	for _, slotID := range slots {
		owner, card := s.slot(slotID)
		if card == nil || owner.ID != id {
			return ErrInvalidCard
		}
	}
	s.Players[i].initial = append([]string{}, slots...)
	s.Revision++
	return nil
}

func (s *KaboState) Ready(id string) error {
	i := s.PlayerIndex(id)
	if s.Phase != KaboInitial || i < 0 || s.Players[i].Ready || !s.Players[i].Active || len(s.Players[i].initial) != 2 {
		return ErrWrongPhase
	}
	s.Players[i].Ready = true
	for _, p := range s.Players {
		if p.Active && !p.Ready {
			s.Revision++
			return nil
		}
	}
	s.begin()
	s.Revision++
	return nil
}

func (s *KaboState) begin() {
	s.CurrentPlayerIdx = s.next(-1)
	s.Phase = KaboDraw
	s.LastAction = "Draw a card to begin."
}

func (s *KaboState) Apply(id, kind string, a KaboAction) error {
	i := s.PlayerIndex(id)
	if i < 0 || !s.Players[i].Active {
		return ErrInvalidPlayer
	}
	if a.Revision != s.Revision {
		return errors.New("The table changed. Choose your action again.")
	}
	if kind == "match" {
		return s.match(id, a)
	}
	if s.Phase == KaboRoundOver || s.Phase == KaboGameOver || s.Phase == KaboInitial {
		return ErrWrongPhase
	}
	if s.Players[s.CurrentPlayerIdx].ID != id {
		return ErrNotYourTurn
	}
	switch kind {
	case "draw":
		if s.Phase != KaboDraw {
			return ErrWrongPhase
		}
		var card KaboCard
		var err error
		switch a.Source {
		case "deck":
			card, err = s.takeCard()
		case "discard":
			if len(s.discards) == 0 {
				return errors.New("Discard pile is empty")
			}
			card = s.discards[len(s.discards)-1]
			s.discards = s.discards[:len(s.discards)-1]
		default:
			return errors.New("Choose the draw pile or discard pile")
		}
		if err != nil {
			return err
		}
		s.drawn, s.drawnFromDeck = &card, a.Source == "deck"
		s.Phase, s.MatchOpen = KaboResolve, false
		s.LastAction = s.Players[i].Name + " drew a card."
	case "call":
		if s.Phase != KaboResolve || s.CallerID != "" || s.drawn == nil {
			return ErrWrongPhase
		}
		s.CallerID = id
		s.FinalTurns = 0
		s.finalPending = map[string]bool{}
		for _, p := range s.Players {
			if p.Active && p.ID != id {
				s.FinalTurns++
				s.finalPending[p.ID] = true
			}
		}
		s.LastAction = s.Players[i].Name + " called Kabo. Finish the drawn card."
	case "replace":
		if s.Phase != KaboResolve || s.drawn == nil {
			return ErrWrongPhase
		}
		owner, slot := s.slot(a.SlotID)
		if slot == nil || owner.ID != id {
			return ErrInvalidCard
		}
		old := slot.card
		slot.card = *s.drawn
		s.putDiscard(old)
		s.LastAction = s.Players[i].Name + " replaced a card."
		s.finishTurn()
	case "discard":
		if s.Phase != KaboResolve || s.drawn == nil || !s.drawnFromDeck {
			return errors.New("A card taken from discard must replace one of your cards")
		}
		card := *s.drawn
		s.putDiscard(card)
		s.drawn = nil
		switch {
		case card.Rank == 7 || card.Rank == 8:
			s.Power = "own"
		case card.Rank == 9 || card.Rank == 10:
			s.Power = "spy"
		case card.Rank == 11 || card.Rank == 12:
			s.Power = "swap"
		case card.Rank == 13 && (card.Suit == "clubs" || card.Suit == "spades"):
			s.Power = "king"
		default:
			s.Power = ""
		}
		s.LastAction = s.Players[i].Name + " discarded a card."
		if s.Power == "" {
			s.finishTurn()
		} else {
			s.Phase = KaboPower
		}
	case "power":
		if s.Phase != KaboPower {
			return ErrWrongPhase
		}
		owner, own := s.slot(a.SlotID)
		targetOwner, target := s.slot(a.TargetID)
		switch s.Power {
		case "own":
			if own == nil || owner.ID != id {
				return ErrInvalidCard
			}
			s.reveals = []KaboReveal{{id, own.ID, own.card}}
		case "spy":
			if target == nil || targetOwner.ID == id {
				return ErrInvalidCard
			}
			s.reveals = []KaboReveal{{targetOwner.ID, target.ID, target.card}}
		case "swap", "king":
			if own == nil || target == nil || owner.ID != id || targetOwner.ID == id {
				return ErrInvalidCard
			}
			if s.Power == "swap" {
				own.card, target.card = target.card, own.card
				s.LastAction = s.Players[i].Name + " swapped two hidden cards."
				s.finishTurn()
			} else {
				s.reveals = []KaboReveal{{id, own.ID, own.card}, {targetOwner.ID, target.ID, target.card}}
				s.powerTargets = append([]KaboReveal{}, s.reveals...)
			}
		}
		if s.Phase == KaboPower {
			s.Phase = KaboPeek
		}
	case "ack":
		if s.Phase != KaboPeek {
			return ErrWrongPhase
		}
		s.reveals = nil
		if s.Power == "king" {
			s.Phase = KaboDecision
		} else {
			s.finishTurn()
		}
	case "decision":
		if s.Phase != KaboDecision || len(s.powerTargets) != 2 {
			return ErrWrongPhase
		}
		if a.Swap {
			_, own := s.slot(s.powerTargets[0].SlotID)
			_, target := s.slot(s.powerTargets[1].SlotID)
			if own == nil || target == nil {
				return ErrInvalidCard
			}
			own.card, target.card = target.card, own.card
		}
		s.LastAction = s.Players[i].Name + " finished the King power."
		s.finishTurn()
	case "skip":
		if s.Phase != KaboPower {
			return ErrWrongPhase
		}
		s.finishTurn()
	default:
		return errors.New("Unknown Kabo action")
	}
	s.Revision++
	return nil
}

func (s *KaboState) match(id string, a KaboAction) error {
	if s.Phase != KaboDraw || !s.MatchOpen || a.DiscardID != s.DiscardID || len(s.discards) == 0 {
		return errors.New("Matching opportunity is closed")
	}
	if s.failedMatches[id] {
		return errors.New("You already tried to match this discard")
	}
	owner, target := s.slot(a.TargetID)
	if target == nil {
		return ErrInvalidCard
	}
	var give *KaboSlot
	if owner.ID != id {
		giveOwner, slot := s.slot(a.SlotID)
		if slot == nil || giveOwner.ID != id {
			return errors.New("Choose your card to give the other player")
		}
		give = slot
	}
	if target.card.Rank != s.discards[len(s.discards)-1].Rank {
		card, err := s.takeCard()
		if err != nil {
			return err
		}
		i := s.PlayerIndex(id)
		s.Players[i].Slots = append(s.Players[i].Slots, KaboSlot{ID: uuid.NewString(), card: card})
		s.failedMatches[id] = true
		s.LastAction = s.Players[i].Name + " missed the match and received a hidden penalty card."
		s.Revision++
		return nil
	}
	discarded := target.card
	removeID := target.ID
	removeOwner := owner
	if give != nil {
		target.card = give.card
		removeID = give.ID
		removeOwner = &s.Players[s.PlayerIndex(id)]
	}
	for j, slot := range removeOwner.Slots {
		if slot.ID == removeID {
			removeOwner.Slots[j].card = KaboCard{}
			break
		}
	}
	s.putDiscard(discarded)
	s.MatchOpen = false
	s.LastAction = s.Players[s.PlayerIndex(id)].Name + " matched and discarded a card."
	if removeOwner.cardCount() == 0 {
		s.finishRound()
	}

	s.Revision++
	return nil
}

func (p *KaboPlayer) cardCount() int {
	n := 0
	for _, slot := range p.Slots {
		if slot.card.Rank != 0 {
			n++
		}
	}
	return n
}

func (s *KaboState) NextRound() error {
	if s.Phase != KaboRoundOver {
		return ErrWrongPhase
	}
	active := 0
	for _, p := range s.Players {
		if p.Active {
			active++
		}
	}
	if active < 2 {
		return errors.New("At least two active players are needed to redeal")
	}
	s.deal()
	return nil
}

func (s *KaboState) EndGame() error {
	if s.Phase != KaboRoundOver {
		return ErrWrongPhase
	}
	s.endMatch()
	s.Revision++
	return nil
}

func (s *KaboState) Leave(id string) {
	i := s.PlayerIndex(id)
	if i < 0 || !s.Players[i].Active {
		return
	}
	for _, slot := range s.Players[i].Slots {
		if slot.card.Rank != 0 {
			s.deck = append(s.deck, slot.card)
		}
	}
	s.Players[i].Active, s.Players[i].Slots = false, nil
	active := 0
	for _, p := range s.Players {
		if p.Active {
			active++
		}
	}
	if active < 2 {
		if s.Phase != KaboInitial {
			s.finishRound()
		}
		s.endMatch()
	} else if s.Phase == KaboInitial {
		ready := true
		for _, p := range s.Players {
			ready = ready && (!p.Active || p.Ready)
		}
		if ready {
			s.begin()
		}
	} else if s.Phase != KaboRoundOver && s.Phase != KaboGameOver {
		if s.CallerID != "" {
			delete(s.finalPending, id)
			s.FinalTurns = len(s.finalPending)
		}
		if s.CallerID != "" && s.FinalTurns <= 0 {
			s.finishRound()
		} else if i == s.CurrentPlayerIdx {
			if s.drawn != nil {
				s.putDiscard(*s.drawn)
			}
			s.drawn, s.reveals, s.powerTargets = nil, nil, nil
			s.Power = ""
			s.CurrentPlayerIdx = s.nextTurn(s.CurrentPlayerIdx)
			s.Phase, s.MatchOpen = KaboDraw, false
		} else if len(s.powerTargets) == 2 && s.powerTargets[1].PlayerID == id {
			s.finishTurn()
		} else if s.Phase == KaboPeek && len(s.reveals) == 1 && s.reveals[0].PlayerID == id {
			s.finishTurn()
		}
	}
	s.LastAction = s.Players[i].Name + " left the game."
	s.Revision++
}

func (s *KaboState) View(viewer string) any {
	type slotView struct {
		ID    string `json:"id"`
		Empty bool   `json:"empty"`
		KaboCard
	}
	type playerView struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		Active     bool       `json:"active"`
		Ready      bool       `json:"ready"`
		TotalScore int        `json:"totalScore"`
		RoundScore *int       `json:"roundScore,omitempty"`
		Cards      []slotView `json:"cards"`
	}
	players := []playerView{}
	over := s.Phase == KaboRoundOver || s.Phase == KaboGameOver
	reveals := []KaboReveal{}
	actor := s.CurrentPlayerIdx >= 0 && s.Players[s.CurrentPlayerIdx].ID == viewer
	for _, p := range s.Players {
		v := playerView{ID: p.ID, Name: p.Name, Active: p.Active, Ready: p.Ready, TotalScore: p.TotalScore, Cards: []slotView{}}
		if over && p.Active {
			score := p.RoundScore
			v.RoundScore = &score
		}
		for _, slot := range p.Slots {
			card := KaboCard{}
			if over {
				card = slot.card
			}
			v.Cards = append(v.Cards, slotView{ID: slot.ID, Empty: slot.card.Rank == 0, KaboCard: card})
			if s.Phase == KaboInitial && p.ID == viewer && !p.Ready {
				for _, id := range p.initial {
					if id == slot.ID {
						reveals = append(reveals, KaboReveal{p.ID, slot.ID, slot.card})
					}
				}
			}
		}
		players = append(players, v)
	}
	var drawn *KaboCard
	if actor {
		drawn = s.drawn
		if s.Phase == KaboPeek {
			reveals = append(reveals, s.reveals...)
		}
	}
	var discard *KaboCard
	if len(s.discards) > 0 {
		top := s.discards[len(s.discards)-1]
		discard = &top
	}
	return struct {
		ID               string       `json:"id"`
		Revision         int          `json:"revision"`
		Round            int          `json:"round"`
		Phase            string       `json:"phase"`
		Players          []playerView `json:"players"`
		CurrentPlayerIdx int          `json:"currentPlayerIdx"`
		CallerID         string       `json:"callerId"`
		FinalTurns       int          `json:"finalTurns"`
		Power            string       `json:"power"`
		MatchOpen        bool         `json:"matchOpen"`
		MatchAllowed     bool         `json:"matchAllowed"`
		DiscardID        string       `json:"discardId"`
		Discard          *KaboCard    `json:"discard"`
		Drawn            *KaboCard    `json:"drawn"`
		FromDeck         bool         `json:"fromDeck"`
		Reveals          []KaboReveal `json:"reveals"`
		LastAction       string       `json:"lastAction"`
		Winners          []string     `json:"winners"`
		RoundWinners     []string     `json:"roundWinners"`
	}{s.ID, s.Revision, s.Round, s.Phase, players, s.CurrentPlayerIdx, s.CallerID, s.FinalTurns, s.Power, s.MatchOpen,
		s.MatchOpen && !s.failedMatches[viewer] && s.PlayerIndex(viewer) >= 0 && s.Players[s.PlayerIndex(viewer)].Active,
		s.DiscardID, discard, drawn, s.drawnFromDeck, reveals, s.LastAction, s.Winners, s.RoundWinners}
}
