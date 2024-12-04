package farkle

import (
	"fmt"
	"math"
)

// Action is the choice made by a player after rolling.
// A zero Action is a Farkle.
type Action struct {
	HeldDiceID      int
	ContinueRolling bool
}

func ApplyAction(state GameState, action Action) GameState {
	newScore := state.ScoreThisRound + scoreCache[action.HeldDiceID]
	if newScore < state.ScoreThisRound {
		newScore = math.MaxUint8 // Overflow
	}
	state.ScoreThisRound = newScore

	state.NumDiceToRoll -= rollNumDice[action.HeldDiceID]
	if state.NumDiceToRoll > maxNumDice {
		panic(fmt.Errorf("illegal action %+v applied to state %+v: "+
			"%d dice remain after removing %d",
			action, state, state.NumDiceToRoll, rollNumDice[action.HeldDiceID]))
	}
	if state.NumDiceToRoll == 0 {
		state.NumDiceToRoll = maxNumDice
	}

	if !action.ContinueRolling {
		currentScore := state.PlayerScores[state.CurrentPlayer]
		newScore := currentScore + state.ScoreThisRound
		if newScore < currentScore {
			newScore = math.MaxUint8 // Overflow
		}
		state.PlayerScores[state.CurrentPlayer] = newScore
		state.CurrentPlayer = (state.CurrentPlayer + 1) % state.NumPlayers
		state.ScoreThisRound = 0
		state.NumDiceToRoll = maxNumDice
	}

	return state
}

var rollToPotentialActions = func() [][]Action {
	result := make([][]Action, len(rollToPotentialHolds))
	for rollID, holds := range rollToPotentialHolds {
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

var farkleProbs = func() [maxNumDice + 1]float64 {
	var pFarkle [maxNumDice + 1]float64
	for numDice := 1; numDice <= maxNumDice; numDice++ {
		for _, wRoll := range allRolls[numDice] {
			potentialActions := rollToPotentialActions[wRoll.ID]
			if len(potentialActions) == 0 { // Farkle!
				pFarkle[numDice] += wRoll.Prob
			}
		}
	}
	return pFarkle
}()

func CalculateWinProb(state GameState) [maxNumPlayers]float64 {
	if state.IsGameOver() || state.ScoreThisRound == math.MaxUint8 {
		var result [maxNumPlayers]float64
		result[state.CurrentPlayer] = 1.0
		return result
	}

	var pWin [maxNumPlayers]float64
	for _, wRoll := range allRolls[state.NumDiceToRoll] {
		// Find the action that maximize current player win probability.
		var bestSubtreeProb [maxNumPlayers]float64
		potentialActions := rollToPotentialActions[wRoll.ID]
		for _, action := range potentialActions {
			if state.CurrentPlayerScore() == 0 && state.ScoreThisRound < 500/incr && !action.ContinueRolling {
				continue // Must get at least 500 to get on the board.
			}

			newState := ApplyAction(state, action)
			pSubtree := CalculateWinProb(newState)
			if pSubtree[state.CurrentPlayer] > bestSubtreeProb[state.CurrentPlayer] {
				bestSubtreeProb = pSubtree
			}
		}

		for i := uint8(0); i < state.NumPlayers; i++ {
			pWin[i] += wRoll.Prob * bestSubtreeProb[i]
		}
	}

	newState := ApplyAction(state, Action{})
	pSubtree := CalculateWinProb(newState)
	for i := uint8(0); i < state.NumPlayers; i++ {
		pWin[i] += farkleProbs[state.NumDiceToRoll] * pSubtree[i]
	}

	return pWin
}
