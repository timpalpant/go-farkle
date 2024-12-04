package farkle

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/cockroachdb/pebble"
	"github.com/golang/glog"
)

type DB interface {
	// Store the result for a game state in the database.
	Put(state GameState, pWin [maxNumPlayers]float64)
	// Retrieve a stored result for the given game state.
	// Returns the result (if found), and a bool indicating whether or not it was found.
	Get(state GameState) ([maxNumPlayers]float64, bool)
	// Serialize the database to the given io.Writer.
	WriteTo(io.Writer) error
}

type InMemoryDB struct {
	// A slab of the pWin for all possible game states in a game of numPlayers.
	// The slab is indexed with the following dimensions:
	//   - 0: Number of dice to roll
	//   - 1: Current player score this round
	//   - 2: Current total score of player 0
	//   - 3: Current total score of player 1
	//   - N+2: Current total score of player N
	table      []float64
	numPlayers int

	nPuts, nHits, nMisses int
}

func NewInMemoryDB(numPlayers int) *InMemoryDB {
	numStates := calcNumDistinctStates(numPlayers)
	numEntries := numPlayers * numStates
	table := make([]float64, numEntries)
	for i := range table {
		table[i] = math.NaN()
	}

	return &InMemoryDB{
		table:      table,
		numPlayers: numPlayers,
	}
}

func LoadInMemoryDB(r io.Reader) (*InMemoryDB, error) {
	var numPlayers int64
	err := binary.Read(r, binary.LittleEndian, &numPlayers)
	if err != nil {
		return nil, err
	}

	db := NewInMemoryDB(int(numPlayers))
	binary.Read(r, binary.LittleEndian, &db.table)
	return db, nil
}

func calcNumDistinctStates(numPlayers int) int {
	return maxNumDice << ((numPlayers + 1) * numDistinctScoreBits)
}

func (db *InMemoryDB) Put(gs GameState, pWin [maxNumPlayers]float64) {
	idx := db.calcOffset(gs)
	copy(db.table[idx:], pWin[:gs.NumPlayers])
	db.nPuts++
}

func (db *InMemoryDB) Get(gs GameState) ([maxNumPlayers]float64, bool) {
	if (db.nHits+db.nMisses)%10000000 == 0 {
		pctComplete := float64(db.nPuts) / float64(len(db.table)/int(gs.NumPlayers))
		hitRate := float64(db.nHits) / float64(db.nHits+db.nMisses)
		glog.Infof("Database has %d entries (%.1f%% complete). Hit rate: %d hits, %d misses (%.1f%%)",
			db.nPuts, 100*pctComplete, db.nHits, db.nMisses, 100*hitRate)
	}

	idx := db.calcOffset(gs)
	var result [maxNumPlayers]float64
	if math.IsNaN(db.table[idx]) {
		db.nMisses++
		return result, false
	}

	copy(result[:], db.table[idx:idx+int(gs.NumPlayers)])
	db.nHits++
	return result, true
}

func (db *InMemoryDB) WriteTo(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, int64(db.numPlayers)); err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, db.table)
}

func (db *InMemoryDB) calcOffset(gs GameState) int {
	// First dimension is number of dice to roll.
	idx := int(gs.NumDiceToRoll) << ((gs.NumPlayers + 1) * numDistinctScoreBits)
	// Second dimension is current player score this round.
	idx += int(gs.ScoreThisRound) << (gs.NumPlayers * numDistinctScoreBits)
	// Remaining dimesions are player scores.
	for i := uint8(0); i < gs.NumPlayers; i++ {
		idx += int(gs.PlayerScores[i]) << (i * numDistinctScoreBits)
	}
	return idx
}

// DB that stores results in a Pebble (RocksDB) database.
type PebbleDB struct {
	db *pebble.DB
}

func NewDB(dirName string, cacheSizeBytes int64) (*PebbleDB, error) {
	cache := pebble.NewCache(cacheSizeBytes)
	defer cache.Unref()
	db, err := pebble.Open(dirName, &pebble.Options{
		BytesPerSync: 10 * 1024 * 1024,
		Cache:        cache,
	})
	if err != nil {
		return nil, err
	}

	return &PebbleDB{
		db: db,
	}, nil
}

func (db *PebbleDB) Put(gs GameState, pWin [maxNumPlayers]float64) {
	key := make([]byte, sizeOfGameState)
	n := gs.SerializeTo(key)
	key = key[:n]

	value := make([]byte, 8*gs.NumPlayers)
	for i := uint8(0); i < gs.NumPlayers; i++ {
		buf := value[8*i : 8*(i+1)]
		p := pWin[i]
		binary.LittleEndian.PutUint64(buf, math.Float64bits(p))
	}

	db.db.Set(key, value, &pebble.WriteOptions{})
}

func (db *PebbleDB) Get(gs GameState) ([maxNumPlayers]float64, bool) {
	key := make([]byte, sizeOfGameState)
	n := gs.SerializeTo(key)
	key = key[:n]

	value, closer, err := db.db.Get(key)
	if err == pebble.ErrNotFound {
		return [maxNumPlayers]float64{}, false
	} else if err != nil {
		panic(fmt.Errorf("unable to get score from DB: %w", err))
	}
	defer closer.Close()

	var pWin [maxNumPlayers]float64
	for i := 0; i < len(value); i += 8 {
		buf := value[i : i+8]
		pWin[i/8] = math.Float64frombits(binary.LittleEndian.Uint64(buf))
	}
	return pWin, true
}

func (db *PebbleDB) WriteTo(w io.Writer) error {
	return nil // PebbleDB is already on disk.
}
