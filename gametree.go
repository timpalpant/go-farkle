package farkle

import (
	"fmt"
	"math"
)

const maxValueIter = 20
const valueTol = 1e-12

// Action is the choice made by a player after rolling.
// A zero Action is a Farkle.
type Action struct {
	HeldDiceID      uint16
	ContinueRolling bool
}

func (a Action) String() string {
	if a.HeldDiceID == 0 && !a.ContinueRolling {
		return "FARKLE!"
	}

	roll := rollsByID[a.HeldDiceID]
	contStr := "Stop"
	if a.ContinueRolling {
		contStr = "Continue"
	}
	return fmt.Sprintf("{Held: %s, %s}", roll, contStr)
}

func ApplyAction(state GameState, action Action) GameState {
	trickScore := scoreCache[action.HeldDiceID]
	newScore := state.ScoreThisRound + trickScore
	if newScore < state.ScoreThisRound {
		newScore = math.MaxUint8 // Overflow
	}
	state.ScoreThisRound = newScore
	if trickScore == 0 { // Farkle
		state.ScoreThisRound = 0
	}

	numDiceHeld := rollNumDice[action.HeldDiceID]
	if numDiceHeld > state.NumDiceToRoll {
		panic(fmt.Errorf("illegal action %+v applied to state %+v: "+
			"held %d dice but only had %d to roll",
			action, state, numDiceHeld, state.NumDiceToRoll))
	}
	state.NumDiceToRoll -= numDiceHeld
	if state.NumDiceToRoll == 0 {
		state.NumDiceToRoll = MaxNumDice
	}

	if !action.ContinueRolling {
		currentScore := state.PlayerScores[0]
		newScore := currentScore + state.ScoreThisRound
		if newScore < currentScore {
			newScore = math.MaxUint8 // Overflow
		}
		// Advance to next player by rotating the scores.
		copy(state.PlayerScores[:state.NumPlayers], state.PlayerScores[1:state.NumPlayers])
		state.PlayerScores[state.NumPlayers-1] = newScore
		state.ScoreThisRound = 0
		state.NumDiceToRoll = MaxNumDice
	}

	return state
}

// Find the action that maximizes current player win probability.
func SelectAction(state GameState, rollID uint16, db DB) (Action, [maxNumPlayers]float64) {
	var bestWinProb [maxNumPlayers]float64
	var bestAction Action
	notYetOnBoard := (state.PlayerScores[0] == 0)
	potentialActions := rollIDToPotentialActions[rollID]
	for _, action := range potentialActions {
		if state.ScoreThisRound == math.MaxUint8 && action.ContinueRolling {
			// Overflowed score this round. Our assumption is that this is unlikely.
			// Approximate the solution using the probability as if they stopped.
			action.ContinueRolling = false
		}

		newState := ApplyAction(state, action)
		if notYetOnBoard && !action.ContinueRolling && newState.PlayerScores[state.NumPlayers-1] < 500/incr {
			// Not a valid state: You must get at least 500 to get on the board.
			continue
		}

		pSubtree := db.Get(newState)
		if !action.ContinueRolling {
			// Probabilities are rotated since we advanced to the
			// next player in next state.
			pSubtree = unrotate(pSubtree, state.NumPlayers)
		}
		if pSubtree[0] > bestWinProb[0] {
			bestWinProb = pSubtree
			bestAction = action
		}
	}

	if len(potentialActions) == 0 {
		newState := ApplyAction(state, bestAction)
		pSubtree := db.Get(newState)
		bestWinProb = unrotate(pSubtree, state.NumPlayers)
	}

	return bestAction, bestWinProb
}

var rollIDToPotentialActions = func() [][]Action {
	result := make([][]Action, len(rollIDToPotentialHolds))
	for rollID, holds := range rollIDToPotentialHolds {
		actions := make([]Action, 0, 2*len(holds))
		for _, holdOption := range holds {
			for _, continueRolling := range []bool{true, false} {
				actions = append(actions, Action{
					HeldDiceID:      rollToID[holdOption],
					ContinueRolling: continueRolling,
				})
			}
		}

		result[rollID] = actions
	}

	return result
}()

func UpdateAll(db DB) {
	numPlayers := db.NumPlayers()
	for _, playerScores := range allPossibleScores(numPlayers) {
		for scoreThisRound := uint8(math.MaxUint8); scoreThisRound >= scoreThisRound; scoreThisRound-- {
			for numDiceToRoll := uint8(1); numDiceToRoll <= MaxNumDice; numDiceToRoll++ {
				state := GameState{
					ScoreThisRound: scoreThisRound,
					NumDiceToRoll:  numDiceToRoll,
					NumPlayers:     numPlayers,
					PlayerScores:   playerScores,
				}

				pWin := [maxNumPlayers]float64{}
				for _, wRoll := range allRolls[state.NumDiceToRoll] {
					_, pSubgame := SelectAction(state, wRoll.ID, db)
					for i, p := range pSubgame[:state.NumPlayers] {
						pWin[i] += wRoll.Prob * p
					}
				}

				db.Put(state, pWin)
			}
		}
	}
}

// Enumerate all possible combinations of N-player scores, in descending order.
// We sort descending so that end game states (mostly) get processed first,
// accelerating convergence of earlier game states.
func allPossibleScores(numPlayers uint8) [][maxNumPlayers]uint8 {
	if numPlayers == 0 {
		return [][maxNumPlayers]uint8{{}}
	}

	n := pow(math.MaxUint8, numPlayers)
	result := make([][maxNumPlayers]uint8, 0, n)
	for _, scores := range allPossibleScores(numPlayers - 1) {
		for score := math.MaxUint8; score >= 0; score-- {
			scores[numPlayers-1] = uint8(score)
			result = append(result, scores)
		}
	}

	return result
}

func pow(m, n uint8) int {
	result := 1
	for i := uint8(1); i <= n; i++ {
		result *= int(m)
	}
	return result
}

func unrotate(pWin [maxNumPlayers]float64, numPlayers uint8) [maxNumPlayers]float64 {
	var result [maxNumPlayers]float64
	copy(result[1:numPlayers], pWin[:numPlayers])
	result[0] = pWin[numPlayers-1]
	return result
}

func init() {
	if scoreCache[0] != 0 {
		panic(fmt.Errorf("farkle should have zero score! got %d", scoreCache[0]))
	}
}
