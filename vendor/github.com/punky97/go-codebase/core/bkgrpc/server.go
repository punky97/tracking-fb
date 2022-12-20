package bkgrpc

import (
	"errors"
	"reflect"
)

// Server -- gRPC Server
type Server struct {
	Handler interface{}
}

var (
	errRequestMethodNotSupported = errors.New("Request method is not supported")
)

func isZeroOfUnderlyingType(x interface{}) bool {
	return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}