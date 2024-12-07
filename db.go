package farkle

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/edsrzf/mmap-go"
	"github.com/golang/glog"
)

type DB interface {
	// Store the result for a game state in the database.
	Put(state GameState, pWin [maxNumPlayers]float64)
	// Retrieve a stored result for the given game state.
	// Returns the result (if found), and a bool indicating whether or not it was found.
	Get(state GameState) ([maxNumPlayers]float64, bool)
	io.Closer
}

// DB that stores results in a memory-mapped flat file.
type FileDB struct {
	f          *os.File
	mmap       mmap.MMap
	numPlayers int

	nPuts int
}

func NewFileDB(path string, numPlayers int) (*FileDB, error) {
	numStates := calcNumDistinctStates(numPlayers)
	numEntries := numPlayers * numStates
	fileSize := int64(8 * numEntries)

	var f *os.File
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		// Initialize a new empty database with all NaN values.
		f, err = os.Create(path)
		if err != nil {
			return nil, err
		}
		glog.Infof("Initializing new database at %s with %d entries", path, numEntries)
		w := bufio.NewWriterSize(f, 4*1024*1024)
		nanBits := make([]byte, 8)
		binary.LittleEndian.PutUint64(nanBits, math.Float64bits(math.NaN()))
		for i := 0; i < numEntries; i++ {
			w.Write(nanBits)
		}
		if err := w.Flush(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if stat.Size() != fileSize {
		return nil, fmt.Errorf(
			"%s is not the correct size for %d-player database: "+
				"got %d, expected %d", path, numPlayers, stat.Size(), fileSize)
	} else {
		f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return nil, err
		}
	}

	mmap, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return &FileDB{
		f:          f,
		mmap:       mmap,
		numPlayers: numPlayers,
	}, nil
}

func (db *FileDB) Put(gs GameState, pWin [maxNumPlayers]float64) {
	idx := 8 * calcOffset(gs)
	buf := db.mmap[idx : idx+8*db.numPlayers]
	nonZero := false
	for i, p := range pWin[:gs.NumPlayers] {
		nonZero = nonZero || (p > 0)
		value := math.Float64bits(p)
		binary.LittleEndian.PutUint64(buf[8*i:8*(i+1)], value)
	}

	if nonZero {
		db.nPuts++
	}

	if nonZero && db.nPuts%100000 == 0 {
		glog.Infof(
			"Database has %d entries. Last put: %s -> %v",
			db.nPuts, gs, pWin[:gs.NumPlayers])
	}
}

func (db *FileDB) Get(gs GameState) ([maxNumPlayers]float64, bool) {
	idx := 8 * calcOffset(gs)
	buf := db.mmap[idx : idx+8*db.numPlayers]

	var result [maxNumPlayers]float64
	for i := 0; i < db.numPlayers; i++ {
		value := binary.LittleEndian.Uint64(buf[8*i : 8*(i+1)])
		result[i] = math.Float64frombits(value)
	}

	return result, !math.IsNaN(result[0])
}

func (db *FileDB) Close() error {
	defer db.f.Close()

	// FileDB is already on disk.
	if err := db.mmap.Unmap(); err != nil {
		return err
	}

	err := db.f.Close()
	return err
}

func calcNumDistinctStates(numPlayers int) int {
	return MaxNumDice << ((numPlayers + 1) * numScoreBits)
}

func calcOffset(gs GameState) int {
	// First dimension is number of dice to roll.
	idx := int(gs.NumDiceToRoll-1) << ((gs.NumPlayers + 1) * numScoreBits)
	// Second dimension is current player score this round.
	idx += int(gs.ScoreThisRound) << (gs.NumPlayers * numScoreBits)
	// Remaining dimensions are player scores.
	for i, score := range gs.PlayerScores[:gs.NumPlayers] {
		idx += int(score) << (i * numScoreBits)
	}
	return int(gs.NumPlayers) * idx
}
