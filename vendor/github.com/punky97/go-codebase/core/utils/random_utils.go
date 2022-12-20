package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func RandIntRange(from int64, to int64) int64 {
	if from >= to {
		return from
	}

	return from + int64(rand.Uint64()%(uint64(to - from)))
}
