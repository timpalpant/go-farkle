package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/timpalpant/go-farkle"
)

const gb = 1024 * 1024 * 1024

type Params struct {
	NumPlayers int
	DBPath     string
	CacheGB    int64
}

func main() {
	var params Params
	flag.IntVar(&params.NumPlayers, "num_players", 2, "Number of players")
	flag.StringVar(&params.DBPath, "db", "2player.db", "Path to solution database")
	flag.Int64Var(&params.CacheGB, "cache_gb", 8, "Databse cache size, if using RocksDB")
	flag.Parse()

	go http.ListenAndServe(":6069", nil)

	stat, err := os.Stat(params.DBPath)
	if err != nil {
		glog.Errorf("Unable to open database: %v", err)
		os.Exit(1)
	}

	var db farkle.DB
	if stat.IsDir() {
		db, err = farkle.NewPebbleDB(params.DBPath, params.CacheGB*gb)
	} else {
		db, err = loadDB(params.DBPath)
	}

	if err != nil {
		glog.Errorf("Unable to initialize database: %v", err)
		os.Exit(1)
	}

	playGame(db, params.NumPlayers)
}

func playGame(db farkle.DB, numPlayers int) {
	state := farkle.NewGameState(numPlayers)
	playerID := 0

	for {
		roll := farkle.NewRandomRoll(farkle.MaxNumDice)
		fmt.Printf("Rolled: %s\n", roll)
		rollID := farkle.GetRollID(roll)

		var action farkle.Action
		if farkle.IsFarkle(roll) {
			fmt.Println("...farkle!")
		} else if playerID == 0 {
			held := promptUserForDiceToKeep(roll)
			score := state.ScoreThisRound + farkle.CalculateScore(held)
			continueRolling := true
			if state.CurrentPlayerScore() > 0 || score >= 500/50 {
				continueRolling = promptUserToContinue()
			}
			action = farkle.Action{
				HeldDiceID:      rollID,
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
				fmt.Printf("...optimal action was %s with pWin = %.1f%%\n",
					optAction, 100*pOpt)
				fmt.Printf("...selected action has pWin = %.1f%% (%.1f%%)\n",
					100*pAction, 100*(pAction-pOpt))
			}
		} else { // CP
			action, _ = farkle.SelectAction(state, rollID, db)
			fmt.Printf("...selected action: %s\n", action)
		}

		state = farkle.ApplyAction(state, action)
		if !action.ContinueRolling {
			playerID--
			if playerID < 0 {
				playerID = numPlayers - 1
			}
		}
	}
}

func promptUserForDiceToKeep(roll farkle.Roll) farkle.Roll {
	var held farkle.Roll
	var err error
	for {
		fmt.Printf("...enter dice to keep: ")
		var toKeepStr string
		fmt.Scanln(&toKeepStr)

		held, err = parseHeld(toKeepStr)
		if err == nil {
			for die, numHeld := range held {
				if numHeld > roll[die] {
					err = fmt.Errorf("can't hold %d %ds, only have %d",
						numHeld, die, roll[die])
					break
				}
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
	toKeepStr = strings.ReplaceAll(strings.ReplaceAll(toKeepStr, " ", ""), ",", "")
	var held farkle.Roll
	for _, c := range toKeepStr {
		die, ok := charToDie[c]
		if !ok {
			return farkle.Roll{}, fmt.Errorf("not a valid die: %c", c)
		}

		held[die]++
	}

	return held, nil
}

func loadDB(dbPath string) (*farkle.InMemoryDB, error) {
	f, err := os.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 4*1024*1024)
	return farkle.LoadInMemoryDB(r)
}
