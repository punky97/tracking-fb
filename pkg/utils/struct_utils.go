package utils

import (
	"encoding/json"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"reflect"
)

func Struct2Map(data interface{}) (mapValue map[string]interface{}) {
	rv := reflect.TypeOf(data)
	mapValue = make(map[string]interface{})

	if rv.Kind() == reflect.Struct || (rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct) {
		b, err := json.Marshal(data)
		if err != nil {
			logger.BkLog.Errorf("Convert struct to map error, %v", err)
			return
		}

		err = json.Unmarshal(b, &mapValue)
		if err != nil {
			logger.BkLog.Errorf("Unmarshal to map error, %v", err)
			return
		}

		return mapValue
	}

	return
}

func GetUpdatedMapByUpdatedFields(object interface{}, updatedFields []string) map[string]interface{} {
	objectMap := Struct2Map(object)
	for key := range objectMap {
		if !utils.SliceContain(updatedFields, key) {
			delete(objectMap, key)
		}
	}
	return objectMap
}
