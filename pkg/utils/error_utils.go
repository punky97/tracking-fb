package utils

import (
	"github.com/punky97/go-codebase/core/bkgrpc"
	"google.golang.org/grpc/status"
	"io"
)

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return status.Convert(err).Message() == bkgrpc.ErrNotFoundData.Error() || err.Error() == io.EOF.Error()

}

func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return status.Convert(err).Message() == bkgrpc.ErrDuplicateData.Error()
}

func IsDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	return status.Convert(err).Message() == bkgrpc.ErrDeadlock.Error()
}
