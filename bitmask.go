package farkle

type bitMask struct {
	values []uint64
}

func newBitMask(n int) *bitMask {
	numInts := n / 64 + 1
	return &bitMask{
		values: make([]uint64, numInts),
	}
}

func (bm *bitMask) Set(i int) {
	idx := i / 64
	shift := i % 64
	bm.values[idx] |= (uint64(1) << shift)
}

func (bm *bitMask) Clear(i int) {
	idx := i / 64
	shift := i % 64
	bm.values[idx] &= ^(uint64(1) << shift)
}

func (bm *bitMask) IsSet(i int) bool {
	idx := i / 64
	shift := i % 64
	return (bm.values[idx] & (uint64(1) << shift)) != 0
}