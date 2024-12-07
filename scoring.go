package farkle

const numDistinctScoreBits = 8
const incr = 50
const scoreToWin = 10000 / incr

type TrickType int

const (
	Single1 TrickType = iota
	Single5
	Three1s
	Three2s
	Three3s
	Three4s
	Three5s
	Three6s
	FourOfAKind
	FiveOfAKind
	SixOfAKind
	Straight
	ThreePairs
	FourOfAKindPlusPair
	TwoTriplets
)

var trickScores = map[TrickType]uint8{
	Single1:             100 / incr,
	Single5:             50 / incr,
	Three1s:             300 / incr,
	Three2s:             200 / incr,
	Three3s:             300 / incr,
	Three4s:             400 / incr,
	Three5s:             500 / incr,
	Three6s:             600 / incr,
	FourOfAKind:         1000 / incr,
	FiveOfAKind:         2000 / incr,
	SixOfAKind:          3000 / incr,
	Straight:            1500 / incr,
	ThreePairs:          1500 / incr,
	FourOfAKindPlusPair: 1500 / incr,
	TwoTriplets:         2500 / incr,
}

var threeOfAKind = map[int]TrickType{
	1: Three1s,
	2: Three2s,
	3: Three3s,
	4: Three4s,
	5: Three5s,
	6: Three6s,
}

var singles = map[int]TrickType{
	1: Single1,
	5: Single5,
}

type Trick struct {
	Type TrickType
	Dice Roll
}

func (t Trick) Score() uint8 {
	return trickScores[t.Type]
}

func remainingTricks(roll Roll, trick Trick) [][]Trick {
	result := [][]Trick{{trick}}
	remainingDice := SubtractRolls(roll, trick.Dice)
	for _, addlTricks := range enumeratePossibleTricks(remainingDice) {
		result = append(result, append([]Trick{trick}, addlTricks...))
	}
	return result
}

func enumeratePossibleTricks(roll Roll) [][]Trick {
	var result [][]Trick
	for die, count := range roll {
		if count >= 1 && (die == 1 || die == 5) {
			trick := Trick{
				Type: singles[die],
				Dice: NewRoll(uint8(die)),
			}

			result = append(result, remainingTricks(roll, trick)...)
		}

		if count >= 3 {
			trick := Trick{
				Type: threeOfAKind[die],
				Dice: RepeatedRoll(uint8(die), count),
			}

			result = append(result, remainingTricks(roll, trick)...)
		}

		if count >= 4 {
			trick := Trick{
				Type: FourOfAKind,
				Dice: RepeatedRoll(uint8(die), count),
			}

			result = append(result, remainingTricks(roll, trick)...)
		}

		if count >= 5 {
			trick := Trick{
				Type: FiveOfAKind,
				Dice: RepeatedRoll(uint8(die), count),
			}

			result = append(result, remainingTricks(roll, trick)...)
		}

		if count >= 6 {
			trick := Trick{
				Type: SixOfAKind,
				Dice: roll,
			}

			result = append(result, []Trick{trick})
		}
	}

	if isStraight(roll) {
		trick := Trick{
			Type: Straight,
			Dice: roll,
		}

		result = append(result, []Trick{trick})
	} else if isThreePairs(roll) {
		trick := Trick{
			Type: ThreePairs,
			Dice: roll,
		}

		result = append(result, []Trick{trick})
	} else if isFourOfAKindPlusPair(roll) {
		trick := Trick{
			Type: FourOfAKindPlusPair,
			Dice: roll,
		}

		result = append(result, []Trick{trick})
	} else if isTwoTriplets(roll) {
		trick := Trick{
			Type: TwoTriplets,
			Dice: roll,
		}

		result = append(result, []Trick{trick})
	}

	return result
}

func isStraight(roll Roll) bool {
	for die := 1; die <= numSides; die++ {
		if roll[die] != 1 {
			return false
		}
	}

	return true
}

func isThreePairs(roll Roll) bool {
	numPairs := 0
	for _, count := range roll {
		if count == 2 {
			numPairs++
		}
	}

	return numPairs >= 3
}

func isFourOfAKindPlusPair(roll Roll) bool {
	fourOfAKind := false
	pair := false
	for _, count := range roll {
		if count == 2 {
			pair = true
		}
		if count == 4 {
			fourOfAKind = true
		}
	}

	return fourOfAKind && pair
}

func isTwoTriplets(roll Roll) bool {
	numTriplets := 0
	for _, count := range roll {
		if count == 3 {
			numTriplets++
		}
	}

	return numTriplets >= 2
}

func CalculateScore(held Roll) uint8 {
	result := uint8(0)
	for _, tricks := range enumeratePossibleTricks(held) {
		score := uint8(0)
		for _, trick := range tricks {
			score += trick.Score()
		}

		result = max(result, score)
	}

	return result
}

func potentialHolds(roll Roll) []Roll {
	trickSets := enumeratePossibleTricks(roll)
	result := make([]Roll, 0, len(trickSets))
	for _, tricks := range trickSets {
		allRolls := make([]Roll, len(tricks))
		for i := range allRolls {
			allRolls[i] = tricks[i].Dice
		}

		roll := CombineRolls(allRolls...)
		result = append(result, roll)
	}

	return result
}

var rollIDToPotentialHolds = func() [][]Roll {
	var result [][]Roll
	for _, rolls := range allRolls {
		for _, weightedRoll := range rolls {
			result = append(result, potentialHolds(weightedRoll.Roll))
		}
	}
	return result
}()

func IsFarkle(roll Roll) bool {
	rollID := rollToID[roll]
	return len(rollIDToPotentialHolds[rollID]) == 0
}

func IsValidHold(roll, held Roll) bool {
	rollID := GetRollID(roll)
	potentialHolds := rollIDToPotentialHolds[rollID]
	potentialHoldsSet := make(map[Roll]struct{}, len(potentialHolds))
	for _, hold := range potentialHolds {
		potentialHoldsSet[hold] = struct{}{}
	}

	_, ok := potentialHoldsSet[held]
	return ok
}

// For each set of held dice, the total score.
var scoreCache = func() []uint8 {
	result := make([]uint8, nDistinctRolls)
	for _, holds := range rollIDToPotentialHolds {
		for _, hold := range holds {
			rollID := rollToID[hold]
			result[rollID] = CalculateScore(hold)
		}
	}
	return result
}()
