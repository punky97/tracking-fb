package codec

import (
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vmihailenco/msgpack/v5"
)

// EncodeProtoF --
type EncodeProtoF func(proto.Message) ([]byte, error)

// DecodeProtoF --
type DecodeProtoF func([]byte, proto.Message) error

// EncodeProto encode protobuf
func EncodeProto(v proto.Message) ([]byte, error) {
	d, err := proto.Marshal(v)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cannot encode proto message: %v", err), "message", v)
		return nil, err
	}
	return d, nil
}

// DecodeProto decode proto message
func DecodeProto(b []byte, result proto.Message) error {
	err := proto.Unmarshal(b, result)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cannot decode proto message: %v", err), "message", string(b))
		return err
	}
	return nil
}

func DecodeToStruct(data []byte, item interface{}) error {
	err := msgpack.Unmarshal(data, &item)
	if err != nil {
		return err
	}
	return nil
}

func Encode(value interface{}) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Failed Encoding message %v", err))
		return nil, err
	}

	return b, nil
}

func Decode(data []byte) (interface{}, error) {
	var item interface{}
	err := msgpack.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}
	return item, nil
}
