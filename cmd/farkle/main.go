package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/golang/glog"
	"github.com/timpalpant/go-farkle"
)

const gb = 1024 * 1024 * 1024

type Params struct {
	NumPlayers int
	CacheGB    int64
	DBPath     string
}

func main() {
	var params Params
	flag.IntVar(&params.NumPlayers, "num_players", 4, "Number of players")
	flag.Int64Var(&params.CacheGB, "cache_gb", 8, "Databse cache size")
	flag.StringVar(&params.DBPath, "db", "scoredb", "Path to database directory")
	flag.Parse()

	go http.ListenAndServe(":6069", nil)

	db, err := farkle.NewDB(params.DBPath, params.CacheGB*gb)
	if err != nil {
		glog.Errorf("Unable to open database: %v", err)
		os.Exit(1)
	}

	initialState := farkle.NewGameState(params.NumPlayers)
	glog.Infof("Initial state: %v", initialState)

	winProb := farkle.CalculateWinProb(initialState, db)
	glog.Infof("Probability of winning: %v", winProb)
}
