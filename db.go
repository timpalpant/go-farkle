package farkle

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cockroachdb/pebble"
)

type DB struct {
	db *pebble.DB
}

func NewDB(dirName string, cacheSizeBytes int64) (*DB, error) {
	cache := pebble.NewCache(cacheSizeBytes)
	defer cache.Unref()
	db, err := pebble.Open(dirName, &pebble.Options{
		BytesPerSync: 10 * 1024 * 1024,
		Cache:        cache,
	})
	if err != nil {
		return nil, err
	}

	return &DB{
		db: db,
	}, nil
}

func (db *DB) Put(gs GameState, pWin []float64) {
	key := make([]byte, sizeOfGameState)
	n := gs.SerializeTo(key)
	key = key[:n]
	value := make([]byte, 8*len(pWin))
	for i, p := range pWin {
		buf := value[8*i : 8*(i+1)]
		binary.LittleEndian.PutUint64(buf, math.Float64bits(p))
	}
	db.db.Set(key, value, &pebble.WriteOptions{})
}

func (db *DB) Get(gs GameState) ([maxNumPlayers]float64, bool) {
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
