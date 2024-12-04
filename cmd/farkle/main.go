package main

import (
	"bufio"
	"flag"
	"fmt"
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
	flag.IntVar(&params.NumPlayers, "num_players", 2, "Number of players")
	flag.Int64Var(&params.CacheGB, "cache_gb", 8, "Databse cache size")
	flag.StringVar(&params.DBPath, "db", "scoredb", "Path to database")
	flag.Parse()

	go http.ListenAndServe(":6069", nil)

	initialState := farkle.NewGameState(params.NumPlayers)
	glog.Infof("Initial state: %v", initialState)

	db := farkle.NewInMemoryDB(params.NumPlayers)

	winProb := farkle.CalculateWinProb(initialState, db)
	glog.Infof("Probability of winning: %v", winProb)

	if err := saveDB(db, params.DBPath); err != nil {
		glog.Errorf("Error saving database: %v", err)
		os.Exit(1)
	}
}

func saveDB(db farkle.DB, dbPath string) error {
	f, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 4*1024*1024)
	if err := db.WriteTo(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return f.Close()
}
