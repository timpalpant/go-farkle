package farkle

import "fmt"

const maxNumPlayers = 4

type GameState struct {
	CurrentScore      uint8
	ScoreThisRound    uint8
	NumDiceToRoll     uint8
	OtherPlayerScores [maxNumPlayers]uint8
}

func GameStateFromBytes(buf []byte) GameState {
	gs := GameState{
		CurrentScore:   buf[0],
		ScoreThisRound: buf[1],
		NumDiceToRoll:  buf[2],
	}

	copy(gs.OtherPlayerScores[:], buf[3:])
	return gs
}

func (gs GameState) NumPlayers() int {
	return len(gs.OtherPlayerScores) + 1
}

func (gs GameState) ToBytes() []byte {
	nBytes := len(gs.OtherPlayerScores) + 3
	buf := make([]byte, nBytes)
	gs.SerializeTo(buf)
	return buf
}

func (gs GameState) SerializeTo(buf []byte) int {
	nBytes := len(gs.OtherPlayerScores) + 3
	if len(buf) < nBytes {
		panic(fmt.Errorf(
			"cannot serialize GameState: "+
				"buffer has %d bytes but need %d",
			len(buf), nBytes))
	}

	buf[0] = gs.CurrentScore
	buf[1] = gs.ScoreThisRound
	buf[2] = gs.NumDiceToRoll
	copy(buf[3:], gs.OtherPlayerScores[:])
	return nBytes
}
