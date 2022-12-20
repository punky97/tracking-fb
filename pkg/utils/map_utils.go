package utils

import "reflect"

func GetMapFromSliceOfInterface(s interface{}) (isValidMap bool, sliceOfMap []map[string]interface{}) {
	rv := reflect.ValueOf(s)
	sliceOfMap = make([]map[string]interface{}, 0)

	if rv.Kind() != reflect.Slice {
		return
	}

	if sm, ok := s.([]map[string]interface{}); ok {
		return true, sm
	}

	if sm, ok := s.([]interface{}); ok {
		for _, v := range sm {
			if m, ok := v.(map[string]interface{}); ok {
				sliceOfMap = append(sliceOfMap, m)
			} else {
				return
			}
		}

		return len(sliceOfMap) > 0, sliceOfMap
	}

	return
}

func GetKeysFromInterfaceMap(v map[interface{}]interface{}) []interface{} {
	keys := make([]interface{}, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}

	return keys
}