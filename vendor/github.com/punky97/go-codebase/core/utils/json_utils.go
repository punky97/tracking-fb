package utils

import (
	"github.com/punky97/go-codebase/core/logger"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"reflect"
	"runtime"
)

// ReadJSON -- read json
func ReadJSON(rc io.ReadCloser, v interface{}) error {
	defer rc.Close()
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	return nil
}

// ToJSONString -- convert data to json string
func ToJSONString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		logger.BkLog.Warnf("Error When convert interface : %v to json string, detail: %v", v, err)
		return ""
	}
	return string(b)
}

// ToJSONByte -- convert data to json string
func ToJSONbyte(v interface{}) []byte {
	b, _ := json.Marshal(v)

	return b
}

// ToJSONByte -- convert data to json string
func ToJSONbyteIndent(v interface{}, indent string) []byte {
	b, _ := json.MarshalIndent(v, "", indent)

	return b
}

// UnmarshalEscapedJson - Unmarshal escaped-json like this `"{\"foo\": \"bar\"}"`
func UnmarshalEscapedJson(escapedJson string, destination interface{}) error {
	var err error

	if len(escapedJson) < 1 {
		return err
	}

	if string(escapedJson[0]) == `"` {
		// Buffer data is a escaped json so must decode to json before
		err = json.Unmarshal([]byte(escapedJson), &escapedJson)
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal([]byte(escapedJson), &destination)
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		logger.BkLog.Errorw(fmt.Sprintf("Error while parse json, detail %v", err), "file", path.Base(file), "line", line)
	}
	return err
}

// Unmarshal.
func Unmarshal(s string) []string {
	var decoded []string

	json.Unmarshal([]byte(s), &decoded)

	return decoded
}

func JsonMatched(a []byte, b []byte) (bool, error) {
	var decodedA interface{}
	err := json.Unmarshal(a, &decodedA)
	if err != nil {
		return false, err
	}

	var decodedB interface{}
	err = json.Unmarshal(b, &decodedB)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(decodedA, decodedB), nil
}
