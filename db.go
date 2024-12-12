package farkle

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/golang/glog"
	"golang.org/x/sys/unix"
)

type DB interface {
	NumPlayers() uint8
	// Store the result for a game state in the database.
	Put(state GameState, pWin [maxNumPlayers]float64)
	// Retrieve a stored result for the given game state.
	Get(state GameState) [maxNumPlayers]float64
	io.Closer
}

// DB that stores results in a memory-mapped flat file.
type FileDB struct {
	numPlayers int
	f          *os.File

	mmap  []byte
	nPuts int64
}

func NewFileDB(path string, numPlayers int) (*FileDB, error) {
	numStates := calcNumDistinctStates(numPlayers)
	numEntries := numPlayers * numStates
	fileSize := int64(8 * numEntries)

	var f *os.File
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		glog.Infof("Initializing new database at %s with %d states", path, numStates)
		f, err = os.Create(path)
		if err != nil {
			return nil, err
		}
		if err := initDB(f, numStates, numPlayers); err != nil {
			_ = f.Close()
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if stat.Size() != fileSize {
		return nil, fmt.Errorf(
			"%s is not the correct size for %d-player database: "+
				"got %d, expected %d", path, numPlayers, stat.Size(), fileSize)
	} else {
		f, err = os.OpenFile(path, os.O_RDWR, 0755)
		if err != nil {
			return nil, err
		}
	}

	flags := unix.MAP_SHARED
	prot := unix.PROT_READ | unix.PROT_WRITE
	mmap, err := unix.Mmap(int(f.Fd()), 0, int(fileSize), prot, flags)
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

func initDB(w io.Writer, numStates, numPlayers int) error {
	bufW := bufio.NewWriterSize(w, 4*1024*1024)

	defaultValue := make([]byte, 8*numPlayers)
	bits := math.Float64bits(1.0 / float64(numPlayers))
	for i := 0; i < numPlayers; i++ {
		buf := defaultValue[8*i : 8*(i+1)]
		binary.LittleEndian.PutUint64(buf, bits)
	}

	for i := 0; i < numStates; i++ {
		if i%100000000 == 0 {
			glog.Infof("...%d", i)
		}
		bufW.Write(defaultValue)
	}
	return bufW.Flush()
}

func (db *FileDB) NumPlayers() uint8 {
	return uint8(db.numPlayers)
}

func (db *FileDB) Put(gs GameState, pWin [maxNumPlayers]float64) {
	gsID := calcOffset(gs)
	idx := 8 * db.numPlayers * gsID

	buf := db.mmap[idx : idx+8*db.numPlayers]
	for i, p := range pWin[:gs.NumPlayers] {
		value := math.Float64bits(p)
		binary.LittleEndian.PutUint64(buf[8*i:8*(i+1)], value)
	}

	db.nPuts++
	if db.nPuts%100000 == 0 {
		glog.Infof(
			"%d puts into database. Last put: %s -> %v",
			db.nPuts, gs, pWin[:gs.NumPlayers])
	}
}

func (db *FileDB) Get(gs GameState) [maxNumPlayers]float64 {
	gsID := calcOffset(gs)
	idx := 8 * db.numPlayers * gsID

	buf := db.mmap[idx : idx+8*db.numPlayers]
	var result [maxNumPlayers]float64

	for i := 0; i < db.numPlayers; i++ {
		value := binary.LittleEndian.Uint64(buf[8*i : 8*(i+1)])
		result[i] = math.Float64frombits(value)
	}

	return result
}

func (db *FileDB) Close() error {
	defer db.f.Close()

	if err := unix.Msync(db.mmap, unix.MS_SYNC); err != nil {
		return err
	}
	if err := unix.Munmap(db.mmap); err != nil {
		return err
	}

	return db.f.Close()
}

func calcNumDistinctStates(numPlayers int) int {
	return MaxNumDice << ((numPlayers + 1) * numScoreBits)
}

func calcOffset(gs GameState) int {
	// The array must be arranged so that there is locality in the
	// mmapped pages as process all states.
	// First the number of dice to roll.
	idx := int(gs.NumDiceToRoll-1) << ((gs.NumPlayers + 1) * numScoreBits)
	// First dimensions are player scores.
	numPlayers := int(gs.NumPlayers)
	for i, score := range gs.PlayerScores[:numPlayers] {
		idx += int(score) << ((numPlayers-i) * numScoreBits)
	}
	// Then current player score this round.
	idx += int(gs.ScoreThisRound)
	return idx
}
