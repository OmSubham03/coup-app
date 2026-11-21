package game

import "errors"

const (
	TicTacToePlaying  = "playing"
	TicTacToeGameOver = "game_over"
)

type TicTacToePlayer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TicTacToeState struct {
	Players          [2]TicTacToePlayer `json:"players"`
	Board            [9]string          `json:"board"`
	CurrentPlayerIdx int                `json:"currentPlayerIdx"`
	Phase            string             `json:"phase"`
	WinnerID         string             `json:"winnerId,omitempty"`
	WinningLine      []int              `json:"winningLine,omitempty"`
}

func (s *TicTacToeState) PlayerIndex(id string) int {
	for i, player := range s.Players {
		if player.ID == id {
			return i
		}
	}
	return -1
}

func (s *TicTacToeState) Move(id string, cell int) error {
	if s.Phase != TicTacToePlaying {
		return ErrWrongPhase
	}
	if s.Players[s.CurrentPlayerIdx].ID != id {
		return ErrNotYourTurn
	}
	if cell < 0 || cell >= len(s.Board) {
		return errors.New("cell must be between 0 and 8")
	}
	if s.Board[cell] != "" {
		return errors.New("cell already occupied")
	}

	mark := [2]string{"X", "O"}[s.CurrentPlayerIdx]
	s.Board[cell] = mark
	for _, line := range [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	} {
		if s.Board[line[0]] == mark && s.Board[line[1]] == mark && s.Board[line[2]] == mark {
			s.Phase, s.WinnerID, s.WinningLine = TicTacToeGameOver, id, line[:]
			return nil
		}
	}
	for _, value := range s.Board {
		if value == "" {
			s.CurrentPlayerIdx = 1 - s.CurrentPlayerIdx
			return nil
		}
	}
	s.Phase = TicTacToeGameOver
	return nil
}
