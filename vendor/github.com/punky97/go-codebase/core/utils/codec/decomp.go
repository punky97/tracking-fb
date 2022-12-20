package codec

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"io/ioutil"
)

// Compress -
func Compress(input []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(input); err != nil {
		return input
	}
	if err := gz.Flush(); err != nil {
		return input
	}
	if err := gz.Close(); err != nil {
		return input
	}
	return b.Bytes()
}

// Decompress -
func Decompress(input []byte) []byte {
	if input == nil {
		return []byte{}
	}
	br := bytes.NewReader(input)
	gz, err := gzip.NewReader(br)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when decompress byte array, details: %v", err))
		return []byte{}
	}
	out, err := ioutil.ReadAll(gz)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when decompress byte array, details: %v", err))
		return []byte{}
	}
	return out
}
