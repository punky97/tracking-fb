package common

import (
	"github.com/punky97/go-codebase/core/bkgrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func SplitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", fullMethodName
}

func GrpcErrorConvert(err error) codes.Code {
	st, ok := status.FromError(err)
	if !ok {
		if status.Convert(err).Message() == bkgrpc.ErrNotFoundData.Error() {
			return codes.NotFound
		}
	}

	return st.Code()
}

func IsErrorCode(code string) bool {
	if code == codes.OK.String() || code == codes.NotFound.String() {
		return false
	}

	return true
}
