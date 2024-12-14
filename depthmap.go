package farkle

import (
	"encoding/binary"
	"os"

	"golang.org/x/sys/unix"
)

// TODO: Figure out how to generalize the FileDB struct
// without incurring allocations.
type depthMap struct {
	f    *os.File
	mmap []byte
}

func newDepthMap(path string, numStates int) (*depthMap, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	fileSize := 2 * numStates
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
		f:    f,
		mmap: mmap,
	}, nil
}

func (dm *depthMap) Set(id int, depth int) {
	idx := 2 * id
	buf := dm.mmap[idx : idx+2]
	binary.LittleEndian.PutUint16(buf, uint16(depth))
}

func (dm *depthMap) Get(id int) int {
	idx := 2 * id
	buf := dm.mmap[idx : idx+2]
	return int(binary.LittleEndian.Uint16(buf))
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
