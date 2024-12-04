package farkle

import "fmt"

const maxNumDice = 6
const numSides = 6

// Roll represents an unordered roll of N dice.
// A roll can hold 1 - maxNumDice dice. Extra entries are
// at the end of the roll with the value -1.
type Roll [numSides+1]uint8

func NewRoll(dice ...uint8) Roll {
	if len(dice) > maxNumDice {
		panic(fmt.Errorf("cannot create Roll with %d > max %d dice",
			len(dice), maxNumDice))
	}

	var roll Roll
	for _, die := range dice {
		if die < 1 || die > numSides {
			panic(fmt.Errorf("cannot create Roll with die = %d", die))
		}

		roll[die]++
	}

	return roll
}

func NewRollFromCounts(counts [numSides + 1]uint8) Roll {
	return Roll(counts)
}

func RepeatedRoll(die uint8, n uint8) Roll {
	var roll Roll
	roll[die] = n
	return roll
}

func CombineRolls(rolls ...Roll) Roll {
	var roll Roll
	for _, roll := range rolls {
		for die, c := range roll {
			roll[die] += c
		}
	}

	return roll
}

func SubtractRolls(a, b Roll) Roll {
	result := a
	for die := range a {
		if b[die] > a[die] {
			panic(fmt.Errorf("cannot remove %d %ds from roll with only %d",
				b[die], die, a[die]))
		}

		result[die] -= b[die]
	}

	return result
}

func (r Roll) String() string {
	return fmt.Sprintf("%v", r.Dice())
}

// The dice in this roll, sorted in ascending order.
func (r Roll) Dice() []uint8 {
	result := make([]uint8, 0, r.NumDice())
	for die, count := range r {
		for i := uint8(0); i < count; i++ {
			result = append(result, uint8(die))
		}
	}

	return result
}

// The number of dice in this roll, in the range 0 - maxNumDice.
func (r Roll) NumDice() uint8 {
	n := uint8(0)
	for _, c := range r {
		n += c
	}
	return n
}

// Make all distinct permutations of N dice.
func makeRolls(nDice int) []Roll {
	if nDice <= 0 {
		return nil
	} else if nDice == 1 {
		result := make([]Roll, 0, numSides)
		for i := uint8(1); i <= numSides; i++ {
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
	ID int
	Prob float64
}

// Make all distinct combinations of N dice.
func makeWeightedRolls(nDice int) []WeightedRoll {
	rollToFreq := make(map[Roll]int)
	totalCount := 0
	for _, roll := range makeRolls(nDice) {
		rollToFreq[roll]++
		totalCount++
	}

	result := make([]WeightedRoll, 0, len(rollToFreq))
	rollID := 0
	for roll, count := range rollToFreq {
		result = append(result, WeightedRoll{
			Roll: roll,
			ID: rollID,
			Prob: float64(count) / float64(totalCount),
		})
		rollID++
	}

	return result
}

var allRolls = func() [maxNumDice + 1][]WeightedRoll {
	var result [maxNumDice + 1][]WeightedRoll
	for nDice := 1; nDice <= maxNumDice; nDice++ {
		result[nDice] = makeWeightedRolls(nDice)
	}

	// Renumber all rolls with a distinct, sequential ID
	// that can be used to look up other properties.
	nextRollID := 0
	for _, rolls := range result {
		for i := range rolls {
			rolls[i].ID = nextRollID
			nextRollID++
		}
	}

	return result
}()

var nDistinctRolls = func() int {
	n := 0
	for _, rolls := range allRolls {
		n += len(rolls)
	}
	return n
}()

var rollToID = func() map[Roll]int {
	result := make(map[Roll]int, nDistinctRolls)
	for _, rolls := range allRolls {
		for _, wRoll := range rolls {
			result[wRoll.Roll] = wRoll.ID
		}
	}
	return result
}()

var rollNumDice = func() []uint8 {
	result := make([]uint8, nDistinctRolls)
	for _, rolls := range allRolls {
		for _, wRoll := range rolls {
			result[wRoll.ID] = wRoll.NumDice()
		}
	}
	return result
}()
