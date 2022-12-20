package utils

import (
	"github.com/punky97/go-codebase/core/bkgrpc"
	"google.golang.org/grpc/status"
)

// IsErrNotFound - check is error not found
func IsErrNotFound(err error) bool {
	if status.Convert(err).Message() == bkgrpc.ErrNotFoundData.Error() {
		return true
	}

	return false
}
