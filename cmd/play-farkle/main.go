package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/timpalpant/go-farkle"
)

const gb = 1024 * 1024 * 1024

type Params struct {
	NumPlayers int
	DBPath     string
}

func main() {
	var params Params
	flag.IntVar(&params.NumPlayers, "num_players", 2, "Number of players")
	flag.StringVar(&params.DBPath, "db", "2player.db", "Path to solution database")
	flag.Parse()

	db, err := farkle.NewFileDB(params.DBPath, params.NumPlayers)
	if err != nil {
		glog.Errorf("Unable to initialize database: %v", err)
		os.Exit(1)
	}

	playGame(db, params.NumPlayers)
}

func playGame(db farkle.DB, numPlayers int) {
	state := farkle.NewGameState(numPlayers)
	humanPlayerID := 0

	for {
		roll := farkle.NewRandomRoll(int(state.NumDiceToRoll))
		fmt.Printf("Player %d rolled: %s\n", humanPlayerID, roll)
		rollID := farkle.GetRollID(roll)

		var action farkle.Action
		if farkle.IsFarkle(roll) {
			fmt.Println("...farkle!")
		} else if humanPlayerID == 0 {
			held := promptUserForDiceToKeep(roll)
			score := state.ScoreThisRound + farkle.CalculateScore(held)
			continueRolling := true
			if state.CurrentPlayerScore() > 0 || score >= 500/50 {
				fmt.Printf("...score this round = %d\n", int(score)*50)
				continueRolling = promptUserToContinue()
			}
			action = farkle.Action{
				HeldDiceID:      farkle.GetRollID(held),
				ContinueRolling: continueRolling,
			}

			optAction, pWinOpt := farkle.SelectAction(state, rollID, db)
			if action == optAction {
				fmt.Println("...selected action is optimal!")
			} else {
				pOpt := pWinOpt[0]
				if !optAction.ContinueRolling {
					pOpt = pWinOpt[numPlayers-1]
				}
				selectedState := farkle.ApplyAction(state, action)
				pWinAction := farkle.CalculateWinProb(selectedState, db)
				pAction := pWinAction[0]
				if !action.ContinueRolling {
					pAction = pWinAction[numPlayers-1]
				}
				fmt.Printf("...optimal action was %s with pWin = %f\n",
					optAction, pOpt)
				fmt.Printf("...selected action has pWin = %f (%f)\n",
					pAction, pAction-pOpt)
			}
		} else { // CP
			action, _ = farkle.SelectAction(state, rollID, db)
			fmt.Printf("...selected action %s\n", action)
		}

		state = farkle.ApplyAction(state, action)
		if !action.ContinueRolling {
			playerScore := int(state.PlayerScores[humanPlayerID]) * 50
			otherScores := make([]int, 0, state.NumPlayers-1)
			for player, score := range state.PlayerScores[:state.NumPlayers] {
				if player == humanPlayerID {
					continue
				}

				otherScores = append(otherScores, 50*int(score))
			}
			fmt.Printf("Current scores: player = %d, others: %v\n\n",
				playerScore, otherScores)
			humanPlayerID--
			if humanPlayerID < 0 {
				humanPlayerID = numPlayers - 1
			}
		}
	}
}

func promptUserForDiceToKeep(roll farkle.Roll) farkle.Roll {
	var held farkle.Roll
	for {
		fmt.Printf("...enter dice to keep: ")
		rdr := bufio.NewReader(os.Stdin)
		toKeepStr, err := rdr.ReadString('\n')
		if err != nil {
			fmt.Printf("......unable to read dice: %v\n", err)
			continue
		}

		held, err = parseHeld(toKeepStr)
		if err == nil {
			if !farkle.IsValidHold(roll, held) {
				err = fmt.Errorf("can't hold %v, not a valid trick", held)
			}

			if err == nil {
				return held
			}
		}

		fmt.Printf("......unable to parse dice: %v\n", err)
	}
}

var yesNoResponses = map[string]bool{
	"Y":   true,
	"N":   false,
	"1":   true,
	"0":   false,
	"YES": true,
	"NO":  false,
}

func promptUserToContinue() bool {
	for {
		fmt.Printf("...continue rolling (Y/N)? ")
		var yesNoStr string
		fmt.Scanln(&yesNoStr)

		yesNoStr = strings.ToUpper(strings.TrimSpace(yesNoStr))
		continueRolling, ok := yesNoResponses[yesNoStr]
		if !ok {
			fmt.Printf("......don't understand '%s'\n", yesNoStr)
			continue
		}

		return continueRolling
	}
}

var charToDie = map[rune]uint8{
	'1': 1,
	'2': 2,
	'3': 3,
	'4': 4,
	'5': 5,
	'6': 6,
}

func parseHeld(toKeepStr string) (farkle.Roll, error) {
	toKeepStr = strings.ReplaceAll(strings.Map(func(c rune) rune {
		_, ok := charToDie[c]
		if ok {
			return c
		}
		return ' '
	}, toKeepStr), " ", "")

	var held farkle.Roll
	for _, c := range toKeepStr {
		die, ok := charToDie[c]
		if !ok {
			return farkle.Roll{}, fmt.Errorf("not a valid die: '%c'", c)
		}

		held[die]++
	}

	return held, nil
}
