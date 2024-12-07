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

type Params struct {
	NumPlayers int
	DBPath     string
}

func main() {
	var params Params
	flag.IntVar(&params.NumPlayers, "num_players", 2, "Number of players")
	flag.StringVar(&params.DBPath, "db", "2player.db", "Path to solution database")
	flag.Parse()

	go http.ListenAndServe(":6069", nil)

	initialState := farkle.NewGameState(params.NumPlayers)
	glog.Infof("Initial state: %v", initialState)

	db, err := farkle.NewFileDB(params.DBPath, params.NumPlayers)
	if err != nil {
		glog.Errorf("Unable to open database: %v", err)
		os.Exit(1)
	}

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
	if _, err := db.WriteTo(w); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return f.Close()
}
