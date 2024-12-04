package farkle

import "fmt"

const maxNumPlayers = 4
const sizeOfGameState = maxNumPlayers + 8

const scoreToWin = 10000 / incr

type GameState struct {
	ScoreThisRound uint8
	NumDiceToRoll  uint8
	CurrentPlayer  uint8
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
		"CurrentPlayer=%d, NumDiceToRoll=%d, ScoreThisRound=%d, Scores: %v",
		gs.CurrentPlayer, gs.NumDiceToRoll,
		incr*int(gs.ScoreThisRound), scores[:gs.NumPlayers])
}

func (gs GameState) IsGameOver() bool {
	return gs.PlayerScores[gs.CurrentPlayer] >= scoreToWin
}

func GameStateFromBytes(buf []byte) GameState {
	// TODO: Use an unsafe cast from 8 bytes if
	// this is performance-limiting.
	gs := GameState{
		ScoreThisRound: buf[0],
		NumDiceToRoll:  buf[1],
		CurrentPlayer:  buf[2],
		NumPlayers:     buf[3],
	}

	copy(gs.PlayerScores[:], buf[4:])
	return gs
}

func (gs GameState) ToBytes() []byte {
	nBytes := gs.NumPlayers + 4
	buf := make([]byte, nBytes)
	n := gs.SerializeTo(buf)
	return buf[:n]
}

func (gs GameState) SerializeTo(buf []byte) int {
	nBytes := int(gs.NumPlayers + 4)
	if len(buf) < nBytes {
		panic(fmt.Errorf(
			"cannot serialize GameState: "+
				"buffer has %d bytes but need %d",
			len(buf), nBytes))
	}

	buf[0] = gs.ScoreThisRound
	buf[1] = gs.NumDiceToRoll
	buf[2] = gs.CurrentPlayer
	buf[3] = gs.NumPlayers
	copy(buf[4:], gs.PlayerScores[:])
	return nBytes
}
