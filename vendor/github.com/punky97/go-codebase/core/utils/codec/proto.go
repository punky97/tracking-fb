package codec

import (
	"errors"
	"github.com/punky97/go-codebase/core/logger"

	"github.com/golang/protobuf/proto"
)

var (
	// ErrorNilMessage --
	ErrorNilMessage = errors.New("Nil messsage")
)

// EncodeProto encode proto message
func EncodeProto(msg proto.Message) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("Nil messsage")
	}
	var b []byte
	b, err := proto.Marshal(msg)
	if err != nil {
		logger.BkLog.Warn("Cannot encode message due to error. ", err)
	}
	return b, err
}

// DecodeProto --
func DecodeProto(b []byte, output proto.Message) error {
	if b == nil || output == nil {
		return ErrorNilMessage
	}
	err := proto.Unmarshal(b, output)
	if err != nil {
		logger.BkLog.Warn("Cannot decode message due to error. ", err)
	}
	return err
}
