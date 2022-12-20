package utils

import (
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"reflect"
	"strconv"

	"github.com/spf13/cast"
)

func InterfaceArrToStringArr(list []interface{}) []string {
	results := []string{}
	for i := range list {
		results = append(results, list[i].(string))
	}
	return results
}

func GetStringArrFromJSON(input map[string]interface{}, key string) []string {
	strs := input[key]
	if strs == nil {
		return nil
	}
	return InterfaceArrToStringArr(strs.([]interface{}))
}

func GetStringByKeyFromJSON(input map[string]interface{}, key string) string {
	strs := input[key]
	if strs == nil {
		return ""
	}
	return fmt.Sprintf("%v", strs)
}

func GetInt64ByKeyFromJSON(input map[string]interface{}, key string) int64 {
	strs := input[key]
	if strs == nil {
		return 0
	}
	switch reflect.TypeOf(strs).Kind() {
	case reflect.Int64:
		return strs.(int64)
	case reflect.Int32:
		return int64(strs.(int32))
	case reflect.Int16:
		return int64(strs.(int16))
	case reflect.Float64:
		result, err := strconv.ParseInt(fmt.Sprintf("%.0f", strs), 10, 64)
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("GetInt64ByKeyFromJSON cannot convert string to int, detail: %v", err))
		}
		return result
	case reflect.String:
		result, err := strconv.ParseInt(strs.(string), 10, 64)
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("GetInt64ByKeyFromJSON cannot convert string to int, detail: %v", err))
		}
		return result
	default:
		return 0
	}
}

func GetBoolByKeyFromJSON(input map[string]interface{}, key string) bool {
	zeroStr := "0"
	val := input[key]
	if val == nil {
		return false
	}
	valStr := cast.ToString(val)
	valueType := reflect.TypeOf(val).Kind()
	if valueType != reflect.String && valueType != reflect.Bool {
		if valStr != zeroStr {
			valStr = "1"
		}
	}

	return cast.ToBool(valStr)
}

func GetMapByKeyFromJSON(input map[string]interface{}, key string) map[string]interface{} {
	strs := input[key]
	if strs == nil {
		return nil
	}

	return strs.(map[string]interface{})
}

func GetMapStringByKeyFromJSON(input map[string]interface{}, key string) map[string]string {
	strs := input[key]
	if strs == nil {
		return nil
	}

	return strs.(map[string]string)
}

func GetValueFieldFromJSON(input map[string]interface{}) string {
	strs := input["value"]
	if strs == nil {
		return ""
	}
	return strs.(string)
}

func GetListKeyFromMap(input map[string]interface{}) []string {
	results := []string{}
	for k := range input {
		results = append(results, k)
	}
	return results
}

func Append(originalMap map[string]float64, inputMap map[string]float64) map[string]float64 {
	for k, v := range inputMap {
		value := v
		if val, ok := originalMap[k]; ok {
			value += val
		}
		originalMap[k] = value
	}
	return originalMap
}

func CopyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = CopyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}

func GetKeysString(myMap map[string]interface{}) []string {
	keys := make([]string, 0, len(myMap))
	for k := range myMap {
		keys = append(keys, k)
	}

	return keys
}
