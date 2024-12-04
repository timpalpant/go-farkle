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

func init() {
	if scoreCache[0] != 0 {
		panic(fmt.Errorf("farkle should have zero score! got %d", scoreCache[0]))
	}
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
		currentScore := state.CurrentPlayerScore()
		newScore := currentScore + state.ScoreThisRound
		if newScore < currentScore {
			newScore = math.MaxUint8 // Overflow
		}
		// Advance to next player by rotating the scores.
		for i := uint8(0); i < state.NumPlayers-1; i++ {
			state.PlayerScores[i] = state.PlayerScores[i+1]
		}
		state.PlayerScores[state.NumPlayers-1] = newScore
		state.ScoreThisRound = 0
		state.NumDiceToRoll = maxNumDice
	}

	return state
}

var rollIDToPotentialActions = func() [][]Action {
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
			potentialActions := rollIDToPotentialActions[wRoll.ID]
			if len(potentialActions) == 0 { // Farkle!
				pFarkle[numDice] += wRoll.Prob
			}
		}
	}
	return pFarkle
}()

// Recursive implementation that forward propagates all possible actions from the current
// game state and stores results in `db`.
func CalculateWinProb(state GameState, db DB) [maxNumPlayers]float64 {
	if state.IsGameOver() || state.ScoreThisRound == math.MaxUint8 {
		winningScore := state.HighestScore()
		winners := make([]int, 0, maxNumPlayers)
		for player, score := range state.PlayerScores {
			if score >= winningScore {
				winners = append(winners, player)
			}
		}

		// Not clear how ties should be considered in terms of "win probability".
		// We split the win amongst all players with the same score.
		var result [maxNumPlayers]float64
		p := 1.0 / float64(len(winners))
		for _, winner := range winners {
			result[winner] = p
		}
		return result
	}

	if pWin, ok := db.Get(state); ok {
		return pWin
	}

	var pWin [maxNumPlayers]float64
	for _, wRoll := range allRolls[state.NumDiceToRoll] {
		// Find the action that maximize current player win probability.
		var bestSubtreeProb [maxNumPlayers]float64
		potentialActions := rollIDToPotentialActions[wRoll.ID]
		for _, action := range potentialActions {
			if state.CurrentPlayerScore() == 0 && state.ScoreThisRound < 500/incr && !action.ContinueRolling {
				continue // Must get at least 500 to get on the board.
			}

			newState := ApplyAction(state, action)
			pSubtree := CalculateWinProb(newState, db)
			if pSubtree[0] > bestSubtreeProb[0] {
				bestSubtreeProb = pSubtree
			}
		}

		for i := uint8(0); i < state.NumPlayers; i++ {
			pWin[i] += wRoll.Prob * bestSubtreeProb[i]
		}
	}

	// With non-zero probability, all players will Farkle in a row, resulting
	// in recursing to the same game state. At this state, the win probabilities
	// must be the same as the one we are currently calculating.
	db.Put(state, pWin)

	newState := ApplyAction(state, Action{})
	pSubtree := CalculateWinProb(newState, db)
	for i := uint8(0); i < state.NumPlayers; i++ {
		pWin[i] += farkleProbs[state.NumDiceToRoll] * pSubtree[i]
	}

	db.Put(state, pWin)
	return pWin
}
