package mgo_mock

import (
	"encoding/json"
	"github.com/punky97/go-codebase/core/logger"
)

type Iter struct {
	Session *Session
	Result  []interface{}
	index   int
}

func (i *Iter) Next(result interface{}) bool {
	if len(i.Result) < 1 {
		return false
	}
	if i.index >= len(i.Result) {
		return false
	}
	dataByte, err := json.Marshal(i.Result[i.index])
	if err != nil {
		logger.BkLog.Error("Error when Marshal data, err: %v", err)
		return false
	}
	err = json.Unmarshal(dataByte, result)
	if err != nil {
		logger.BkLog.Error("Error when Unmarshal data, err: %v", err)
		return false
	}
	i.index++
	return true
}

func (i *Iter) Close() error {
	return nil
}
