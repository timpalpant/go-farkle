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
	newScore := state.ScoreThisRound + scoreCache[action.HeldDiceID]
	if newScore < state.ScoreThisRound {
		newScore = math.MaxUint8 // Overflow
	}
	state.ScoreThisRound = newScore

	state.NumDiceToRoll -= rollNumDice[action.HeldDiceID]
	if state.NumDiceToRoll > MaxNumDice {
		panic(fmt.Errorf("illegal action %+v applied to state %+v: "+
			"%d dice remain after removing %d",
			action, state, state.NumDiceToRoll, rollNumDice[action.HeldDiceID]))
	}
	if state.NumDiceToRoll == 0 {
		state.NumDiceToRoll = MaxNumDice
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
		state.NumDiceToRoll = MaxNumDice
	}

	return state
}

func SelectAction(state GameState, rollID int, db DB) (Action, [maxNumPlayers]float64) {
	var bestWinProb [maxNumPlayers]float64
	var bestAction Action
	notYetOnBoard := state.CurrentPlayerScore() == 0
	for _, action := range rollIDToPotentialActions[rollID] {
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

		pSubtree := CalculateWinProb(newState, db)
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

	return bestAction, bestWinProb
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

func IsFarkle(roll Roll) bool {
	rollID := rollToID[roll]
	return len(rollIDToPotentialActions[rollID]) == 0
}

var farkleProbs = func() [MaxNumDice + 1]float64 {
	var pFarkle [MaxNumDice + 1]float64
	for numDice := 1; numDice <= MaxNumDice; numDice++ {
		for _, wRoll := range allRolls[numDice] {
			if IsFarkle(wRoll.Roll) {
				pFarkle[numDice] += wRoll.Prob
			}
		}
	}
	return pFarkle
}()

// Recursive implementation that forward propagates all possible actions from the current
// game state and stores results in `db`.
func CalculateWinProb(state GameState, db DB) [maxNumPlayers]float64 {
	if state.IsGameOver() {
		winningScore := state.HighestScore()
		winners := make([]int, 0, maxNumPlayers)
		for player, score := range state.PlayerScores[:state.NumPlayers] {
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
		_, pSubgame := SelectAction(state, wRoll.ID, db)
		for i := uint8(0); i < state.NumPlayers; i++ {
			pWin[i] += wRoll.Prob * pSubgame[i]
		}
	}

	// With non-zero probability, all players will Farkle in a row, resulting
	// in recursing to the same game state. At this state, the win probabilities
	// must be the same as the one we are currently calculating. This is true if:
	//
	//  p_win = \sum (p_roll * p_action) + p_f * p_win
	//  (1 - p_f ^ N) * p_win = \sum (p_roll * p_action)
	//  p_win = \sum (p_roll * p_action) / (1 - p_f)
	//
	// Therefore we put zeros into the database now. After computing all other
	// subtrees (that do not end up in the same state), scale the final result.
	db.Put(state, [maxNumPlayers]float64{})

	newState := ApplyAction(state, Action{})
	pSubtree := unrotate(CalculateWinProb(newState, db), state.NumPlayers)
	for i := uint8(0); i < state.NumPlayers; i++ {
		pWin[i] += farkleProbs[state.NumDiceToRoll] * pSubtree[i]
	}

	// Rescale probabilities to account for recursive farkle into this same state.
	pTotal := 0.0
	for _, p := range pWin[:state.NumPlayers] {
		pTotal += p
	}
	for i, p := range pWin[:state.NumPlayers] {
		pWin[i] = p / pTotal
	}

	db.Put(state, pWin)
	return pWin
}

func unrotate(pWin [maxNumPlayers]float64, numPlayers uint8) [maxNumPlayers]float64 {
	var result [maxNumPlayers]float64
	result[0] = pWin[numPlayers-1]
	for i := uint8(1); i < numPlayers; i++ {
		result[i] = pWin[i-1]
	}
	return result
}

func init() {
	if scoreCache[0] != 0 {
		panic(fmt.Errorf("farkle should have zero score! got %d", scoreCache[0]))
	}
}
