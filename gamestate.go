package farkle

import "fmt"

const maxNumPlayers = 4
const sizeOfGameState = maxNumPlayers + 2

const scoreToWin = 10000 / incr

// State of the game. The current player is always player 0.
type GameState struct {
	ScoreThisRound uint8
	NumDiceToRoll  uint8
	NumPlayers     uint8
	PlayerScores   [maxNumPlayers]uint8
}

func NewGameState(numPlayers int) GameState {
	if numPlayers > maxNumPlayers {
		panic(fmt.Errorf("too many players: %d > maximum %d",
			numPlayers, maxNumPlayers))
	}

	return GameState{
		NumDiceToRoll: maxNumDice,
		NumPlayers:    uint8(numPlayers),
	}
}

func (gs GameState) String() string {
	var scores [maxNumPlayers]int
	for i := 0; i < int(gs.NumPlayers); i++ {
		scores[i] = incr * int(gs.PlayerScores[i])
	}
	return fmt.Sprintf(
		"NumDiceToRoll=%d, ScoreThisRound=%d, Scores: %v",
		gs.NumDiceToRoll, incr*int(gs.ScoreThisRound), scores[:gs.NumPlayers])
}

func (gs GameState) IsGameOver() bool {
	return gs.CurrentPlayerScore() >= scoreToWin
}

func (gs GameState) CurrentPlayerScore() uint8 {
	return gs.PlayerScores[0]
}

func GameStateFromBytes(buf []byte) GameState {
	gs := GameState{
		ScoreThisRound: buf[0],
		NumDiceToRoll:  buf[1],
		NumPlayers:     buf[2],
	}

	copy(gs.PlayerScores[:], buf[3:])
	return gs
}

func (gs GameState) ToBytes() []byte {
	nBytes := gs.NumPlayers + 2
	buf := make([]byte, nBytes)
	n := gs.SerializeTo(buf)
	return buf[:n]
}

func (gs GameState) SerializeTo(buf []byte) int {
	nBytes := int(gs.NumPlayers + 2)
	if len(buf) < nBytes {
		panic(fmt.Errorf(
			"cannot serialize GameState: "+
				"buffer has %d bytes but need %d",
			len(buf), nBytes))
	}

	buf[0] = gs.ScoreThisRound
	buf[1] = gs.NumDiceToRoll
	copy(buf[2:], gs.PlayerScores[:])
	return nBytes
}
