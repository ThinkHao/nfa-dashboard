package settlement95

import "math"

func DescendingIndex(totalPoints int) int {
	if totalPoints <= 0 {
		return -1
	}
	index := totalPoints - int(math.Ceil(float64(totalPoints)*0.95))
	if index < 0 {
		return 0
	}
	if index >= totalPoints {
		return totalPoints - 1
	}
	return index
}
