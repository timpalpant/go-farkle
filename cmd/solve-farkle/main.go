package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/golang/glog"
	"github.com/timpalpant/go-farkle"
)

type Params struct {
	NumPlayers int
	DBPath     string
	NumIter    int
}

func main() {
	var params Params
	flag.IntVar(&params.NumPlayers, "num_players", 2, "Number of players")
	flag.StringVar(&params.DBPath, "db", "2player.db", "Path to solution database")
	flag.IntVar(&params.NumIter, "num_iter", 10, "Number of value iteration cycles")
	flag.Parse()

	go http.ListenAndServe(":6069", nil)

	initialState := farkle.NewGameState(params.NumPlayers)
	glog.Infof("Initial state: %v", initialState)

	db, err := farkle.NewFileDB(params.DBPath, params.NumPlayers)
	if err != nil {
		glog.Errorf("Unable to open database: %v", err)
		os.Exit(1)
	}

	for i := 0; i < params.NumIter; i++ {
		glog.Infof("Starting value iteration cycle %d", i)
		db.Train() // Clear "is calculated" bit for all states
		winProb := farkle.CalculateWinProb(initialState, db)
		glog.Infof("Probability of winning: %v", winProb)
	}

	if err := db.Close(); err != nil {
		glog.Errorf("Error closing database: %v", err)
		os.Exit(1)
	}
}
