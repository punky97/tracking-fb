package utils

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
)

// HandleSigterm -- Handles Ctrl+C or most other means of "controlled" shutdown gracefully.
// Invokes the supplied func before exiting.
func HandleSigterm(handleExit func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		handleExit()
		os.Exit(1)
	}()
}

func ConvertInterfaceToMap(params interface{}) map[string]interface{} {
	paramsConverted := map[string]interface{}{}
	value := reflect.ValueOf(params)

	if value.Kind() == reflect.Map && value.Len() > 0 {
		for _, key := range value.MapKeys() {
			strct := value.MapIndex(key)

			paramskey := fmt.Sprint(key.Interface())
			paramsConverted[paramskey] = strct.Interface()
		}
	}

	return paramsConverted
}

// result: map[string]interface{}, and result convert with bool type
func ConvertInterfaceToMapStringInterface(params interface{}) (map[string]interface{}, bool) {
	s := reflect.ValueOf(params)
	if s.Kind() != reflect.Map {
		return map[string]interface{}{}, false
	}

	value := reflect.ValueOf(params)
	paramsConverted := map[string]interface{}{}
	if value.Len() > 0 {
		for _, key := range value.MapKeys() {
			strct := value.MapIndex(key)

			paramskey := fmt.Sprint(key.Interface())
			paramsConverted[paramskey] = strct.Interface()
		}
	}

	return paramsConverted, true
}

func ConvertInterfaceSliceToMapString(slice interface{}) map[string]interface{} {
	paramsConverted := map[string]interface{}{}
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return paramsConverted
	}

	for i := 0; i < s.Len(); i++ {
		paramsConverted[fmt.Sprint(i)] = s.Index(i).Interface()
	}

	return paramsConverted
}

func ConvertInterfaceToSlice(slice interface{}) ([]interface{}, bool) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil, false
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret, true
}
