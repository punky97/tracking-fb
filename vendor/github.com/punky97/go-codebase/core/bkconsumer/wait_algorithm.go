package bkconsumer

import (
	"github.com/spf13/cast"
	"math"
	"math/rand"
)

const (
	LinearAlgorithm = "linear"
	JitterAlgorithm = "jitter"

	defaultJitterCap = int64(600000)
)

type waitAlgorithm interface {
	Process(ttl, retryNumber int) string
}

func GetWaitAlgorithm(algoType string) waitAlgorithm {
	switch algoType {
	case LinearAlgorithm:
		return NewLinear()
	case JitterAlgorithm:
		return NewJitter(defaultJitterCap)
	default:
		return NewLinear()
	}
}

type Linear struct {
}

func NewLinear() *Linear {
	return &Linear{}
}

func (h *Linear) Process(ttl, retryNumber int) string {
	return cast.ToString((retryNumber + 1) * ttl)
}

type Jitter struct {
	cap int64
}

func NewJitter(cap int64) *Jitter {
	return &Jitter{
		cap: cap,
	}
}

func (h *Jitter) Process(ttl, retryNumber int) string {
	return cast.ToString(rand.Int63n(int64(math.Min(float64(h.cap), float64(ttl)*math.Exp2(float64(retryNumber))))))
}
