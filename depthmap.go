package farkle

import (
	"encoding/binary"
	"os"

	"golang.org/x/sys/unix"
)

// TODO: Figure out how to generalize the FileDB struct
// without incurring allocations.
type depthMap struct {
	f         *os.File
	valueSize int
	mmap      []byte
}

func newDepthMap(path string, numStates int) (*depthMap, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	valueSize := 8
	fileSize := valueSize * numStates
	if err := f.Truncate(int64(fileSize)); err != nil {
		_ = f.Close()
		return nil, err
	}

	flags := unix.MAP_SHARED
	prot := unix.PROT_READ | unix.PROT_WRITE
	mmap, err := unix.Mmap(int(f.Fd()), 0, fileSize, prot, flags)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return &depthMap{
		f:         f,
		mmap:      mmap,
		valueSize: valueSize,
	}, nil
}

func (dm *depthMap) Set(id int, depth int) {
	idx := dm.valueSize * id
	buf := dm.mmap[idx : idx+dm.valueSize]
	binary.LittleEndian.PutUint64(buf, uint64(depth))
}

func (dm *depthMap) Get(id int) int {
	idx := dm.valueSize * id
	buf := dm.mmap[idx : idx+dm.valueSize]
	return int(binary.LittleEndian.Uint64(buf))
}

func (dm *depthMap) Close() error {
	defer dm.f.Close()

	if err := unix.Msync(dm.mmap, unix.MS_SYNC); err != nil {
		return err
	}
	if err := unix.Munmap(dm.mmap); err != nil {
		return err
	}

	return dm.f.Close()
}
