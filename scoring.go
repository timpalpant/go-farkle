package farkle

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

const incr = 50

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

type Trick struct {
	Type TrickType
	Dice Roll
}

func (t Trick) Score() uint8 {
	return trickScores[t.Type]
}

func enumeratePossibleTricks(roll Roll) [][]Trick {
	var dieCounts [numSides + 1]uint8
	for _, die := range roll.Dice() {
		dieCounts[die]++
	}

	var result [][]TrickType
	return result
}

func calculateScore(held Roll) uint8 {
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
		allDice := make([]uint8, 0, maxNumDice)
		for _, trick := range tricks {
			allDice = append(allDice, trick.Dice.Dice()...)
		}

		roll := NewRoll(allDice...)
		roll.Sort()
		result = append(result, roll)
	}

	return result
}

var rollToPotentialHolds = func() map[Roll][]Roll {
	result := make(map[Roll][]Roll)
	for _, rolls := range allRolls {
		for _, weightedRoll := range rolls {
			result[weightedRoll.Roll] = potentialHolds(weightedRoll.Roll)
		}
	}
	return result
}()

// For each set of held dice, the total score.
var heldToScore = func() map[Roll]uint8 {
	result := make(map[Roll]uint8)
	for _, holds := range rollToPotentialHolds {
		for _, hold := range holds {
			if _, ok := result[hold]; ok {
				continue
			}

			result[hold] = calculateScore(hold)
		}
	}
	return result
}()
