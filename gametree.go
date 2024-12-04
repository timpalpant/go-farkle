package farkle

import (
	"fmt"
	"math"
)

// Action is the choice made by a player after rolling.
// A zero Action is a Farkle.
type Action struct {
	HeldDice        Roll
	ContinueRolling bool
}

func ApplyAction(state GameState, action Action) GameState {
	newScore := state.ScoreThisRound + heldToScore[action.HeldDice]
	if newScore < state.ScoreThisRound {
		newScore = math.MaxUint8 // Overflow
	}
	state.ScoreThisRound = newScore

	state.NumDiceToRoll -= action.HeldDice.NumDice()
	if state.NumDiceToRoll > maxNumDice {
		panic(fmt.Errorf("illegal action %+v applied to state %+v: "+
			"%d dice remain after removing %d",
			action, state, state.NumDiceToRoll, action.HeldDice.NumDice()))
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

var rollToPotentialActions = func() map[Roll][]Action {
	result := make(map[Roll][]Action, len(rollToPotentialHolds))
	for roll, holds := range rollToPotentialHolds {
		actions := make([]Action, 0, 2*len(holds))
		for _, holdOption := range holds {
			for _, continueRolling := range []bool{true, false} {
				actions = append(actions, Action{
					HeldDice:        holdOption,
					ContinueRolling: continueRolling,
				})
			}
		}

		result[roll] = actions
	}

	return result
}()

func CalculateWinProb(state GameState, db *DB) [maxNumPlayers]float64 {
	if state.IsGameOver() || state.ScoreThisRound == math.MaxUint8 {
		var result [maxNumPlayers]float64
		result[state.CurrentPlayer] = 1.0
		return result
	}

	if pWin, ok := db.Get(state); ok {
		return pWin
	}

	var pWin [maxNumPlayers]float64
	var pFarkle float64
	for _, wRoll := range allRolls[state.NumDiceToRoll] {
		// Find the action that maximize current player win probability.
		var bestSubtreeProb [maxNumPlayers]float64
		potentialActions := rollToPotentialActions[wRoll.Roll]
		for _, action := range potentialActions {
			newState := ApplyAction(state, action)
			pSubtree := CalculateWinProb(newState, db)
			if pSubtree[state.CurrentPlayer] > bestSubtreeProb[state.CurrentPlayer] {
				bestSubtreeProb = pSubtree
			}
		}

		if len(potentialActions) == 0 { // Farkle!
			pFarkle += wRoll.Prob
		}

		for i := uint8(0); i < state.NumPlayers; i++ {
			pWin[i] += wRoll.Prob * bestSubtreeProb[i]
		}
	}

	if pFarkle > 0 {
		newState := ApplyAction(state, Action{})
		pSubtree := CalculateWinProb(newState, db)
		for i := uint8(0); i < state.NumPlayers; i++ {
			pWin[i] += pFarkle * pSubtree[i]
		}
	}

	db.Put(state, pWin[:state.NumPlayers])
	return pWin
}
