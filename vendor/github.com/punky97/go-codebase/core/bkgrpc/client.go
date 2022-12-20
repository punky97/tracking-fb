package bkgrpc

import (
	"errors"
)

// ErrNotFoundData -- not found data in result set
var ErrNotFoundData = errors.New("gclient: not found data in result")
var ErrRedisNil = errors.New("redis: nil")
var ErrDuplicateData = errors.New("gclient: data duplicate")
var ErrModelNotValid = errors.New("gclient: not valid data")
var ErrNotFoundDataAfterTransactions = errors.New("gclient: not found data in result after commit transactions query")
var ErrIndexNotFound = errors.New("gclient: index not found")
var ErrDeadlock = errors.New("gcclient: deadlock")

var ErrAIColReachLimit = errors.New("gclient: ai columns reach limit")
