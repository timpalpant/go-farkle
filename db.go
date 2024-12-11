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
	mask       []uint64
	mmap       []byte
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
		mask:    make([]uint64, numStates/64+1),
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

func (db *FileDB) Put(gs GameState, pWin [maxNumPlayers]float64) {
	gsID := calcOffset(gs)
	idx := 8 * db.numPlayers * gsID
	buf := db.mmap[idx : idx+8*db.numPlayers]
	for i, p := range pWin[:gs.NumPlayers] {
		value := math.Float64bits(p)
		binary.LittleEndian.PutUint64(buf[8*i:8*(i+1)], value)
	}

	// Mark this GameState as calculated in the bitmask.
	maskIdx := gsID / 64
	bitIdx := gsID % 64
	db.mask[maskIdx] |= (1 << bitIdx)

	db.nPuts++
	if db.nPuts%100000 == 0 {
		glog.Infof(
			"%d puts into database. Last put: %s -> %v",
			db.nPuts, gs, pWin[:gs.NumPlayers])
	}
}

func (db *FileDB) Get(gs GameState) ([maxNumPlayers]float64, bool) {
	gsID := calcOffset(gs)
	idx := 8 * db.numPlayers * gsID
	buf := db.mmap[idx : idx+8*db.numPlayers]

	var result [maxNumPlayers]float64
	for i := 0; i < db.numPlayers; i++ {
		value := binary.LittleEndian.Uint64(buf[8*i : 8*(i+1)])
		result[i] = math.Float64frombits(value)
	}

	maskIdx := gsID / 64
	bitIdx := gsID % 64
	isSet := (db.mask[maskIdx] & (1 << bitIdx)) != 0

	return result, isSet
}

// Mark all states as unset in the database.
func (db *FileDB) Train() {
	for i := range db.mask {
		db.mask[i] = 0
	}
}

// Mark all states as set in the database.
func (db *FileDB) Eval() {
	for i := range db.mask {
		db.mask[i] = ^uint64(0)
	}
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
	// First dimension is number of dice to roll.
	idx := int(gs.NumDiceToRoll-1) << ((gs.NumPlayers + 1) * numScoreBits)
	// Second dimension is current player score this round.
	idx += int(gs.ScoreThisRound) << (gs.NumPlayers * numScoreBits)
	// Remaining dimensions are player scores.
	for i, score := range gs.PlayerScores[:gs.NumPlayers] {
		idx += int(score) << (i * numScoreBits)
	}
	return idx
}
