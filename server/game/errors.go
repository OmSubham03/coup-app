package game

import "errors"

var (
	ErrInvalidPlayer      = errors.New("invalid player")
	ErrNotYourTurn        = errors.New("not your turn")
	ErrWrongPhase         = errors.New("wrong game phase")
	ErrActionNotAvailable = errors.New("action not available")
	ErrNotEnoughCoins     = errors.New("not enough coins")
	ErrMustCoup           = errors.New("must coup with 10+ coins")
	ErrNeedsTarget        = errors.New("action requires a target")
	ErrInvalidTarget      = errors.New("invalid target")
	ErrSelfTarget         = errors.New("cannot target yourself")
	ErrNoPendingAction    = errors.New("no pending action")
	ErrCannotBlock        = errors.New("action cannot be blocked")
	ErrInvalidBlockChar   = errors.New("character cannot block this action")
	ErrSelfBlock          = errors.New("cannot block your own action")
	ErrSelfChallenge      = errors.New("cannot challenge yourself")
	ErrWrongCardCount     = errors.New("must keep the same number of cards")
	ErrInvalidCard        = errors.New("invalid card selection")
)
