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

func (db *DB) Put(gs GameState, score float64) {
	key := make([]byte, maxNumPlayers+3)
	n := gs.SerializeTo(key)
	key = key[:n]
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, math.Float64bits(score))
	db.db.Set(key, value, &pebble.WriteOptions{})
}

func (db *DB) Get(gs GameState) (float64, bool) {
	key := make([]byte, maxNumPlayers+3)
	n := gs.SerializeTo(key)
	key = key[:n]
	value, closer, err := db.db.Get(key)
	if err == pebble.ErrNotFound {
		return -1.0, false
	} else if err != nil {
		panic(fmt.Errorf("unable to get score from DB: %w", err))
	}
	defer closer.Close()
	score := math.Float64frombits(binary.LittleEndian.Uint64(value))
	return score, true
}
