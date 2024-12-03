package farkle

import (
	"fmt"
	"slices"
)

const maxNumDice = 6
const numSides = 6

// Roll represents an ordered roll of N dice.
// A roll can hold 1 - maxNumDice dice. Extra entries are
// at the end of the roll with the value -1.
type Roll [maxNumDice]uint8

func NewRoll(dice ...uint8) Roll {
	if len(dice) > maxNumDice {
		panic(fmt.Errorf("cannot create Roll with %d > max %d dice",
			len(dice), maxNumDice))
	}

	for _, die := range dice {
		if die < 1 || die > numSides {
			panic(fmt.Errorf("cannot create Roll with die = %d", die))
		}
	}

	r := Roll{}
	copy(r[:], dice)
	return r
}

func (r *Roll) Sort() {
	slices.Sort(r.Dice())
}

// The dice in this roll, excluding unused slots.
func (r Roll) Dice() []uint8 {
	for i, die := range r {
		if die <= 0 {
			return r[:i]
		}
	}

	return nil
}

// The number of dice in this roll, in the range 0 - maxNumDice.
func (r Roll) NumDice() int {
	n := 0
	for _, die := range r {
		if die > 0 {
			n++
		}
	}
	return n
}

// Make all distinct permutations of N dice.
func makeRolls(nDice int) []Roll {
	if nDice <= 0 {
		return nil
	} else if nDice == 1 {
		result := make([]Roll, 0, numSides)
		for i := 1; i <= numSides; i++ {
			result = append(result, NewRoll(i))
		}
		return result
	}

	subResult := makeRolls(nDice - 1)
	result := make([]Roll, 0, numSides*len(subResult))
	for _, roll := range subResult {
		for i := uint8(1); i <= numSides; i++ {
			roll[nDice-1] = i
			result = append(result, roll)
		}
	}
	return result
}

// WeightedRoll represents an unordered set of rolled dice,
// and the probability of realizing that combination.
type WeightedRoll struct {
	Roll
	Prob float64
}

// Make all distinct combinations of N dice.
func makeWeightedRolls(nDice int) []WeightedRoll {
	rollToFreq := make(map[Roll]int)
	totalCount := 0
	for _, roll := range makeRolls(nDice) {
		roll.Sort()
		rollToFreq[roll]++
		totalCount++
	}

	result := make([]WeightedRoll, 0, len(rollToFreq))
	for roll, count := range rollToFreq {
		result = append(result, WeightedRoll{
			Roll: roll,
			Prob: float64(count) / float64(totalCount),
		})
	}

	return result
}

var allRolls = func() [maxNumDice + 1][]WeightedRoll {
	var result [maxNumDice + 1][]WeightedRoll
	for nDice := 1; nDice <= maxNumDice; nDice++ {
		result[nDice] = makeWeightedRolls(nDice)
	}

	return result
}()
