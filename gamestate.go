package farkle

import "fmt"

const maxNumPlayers = 4
const sizeOfGameState = maxNumPlayers + 2

const scoreToWin = 10000 / incr

// State of the game. The current player is always player 0.
// Game states can be partially ordered since scores can only go up during game play.
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
		NumDiceToRoll: MaxNumDice,
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

// Whether the game is over, i.e. this is a terminal game state.
func (gs GameState) IsGameOver() bool {
	// After a player exceeds the score to win, other players get one more turn.
	// Therefore the game is over when we come back around such that the current player
	// has a score exceeding the threshold.
	return gs.CurrentPlayerScore() >= scoreToWin
}

// Score of the current player.
func (gs GameState) CurrentPlayerScore() uint8 {
	return gs.PlayerScores[0]
}

// Highest score of any player.
func (gs GameState) HighestScore() uint8 {
	bestScore := uint8(0)
	for _, score := range gs.PlayerScores[:gs.NumPlayers] {
		if score > bestScore {
			bestScore = score
		}
	}
	return bestScore
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
