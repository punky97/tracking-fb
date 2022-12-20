package utils

import (
	"fmt"
	"github.com/punky97/go-codebase/core/utils"
	"reflect"
	"strconv"
	"time"
)

// Deref is Indirect for reflect.Types
func Deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func GetListJSONFieldObject(obj interface{}) []string {
	results := make([]string, 0)
	v := reflect.ValueOf(obj)
	base := Deref(v.Type())
	vp := reflect.New(base)
	if vp.Kind() == reflect.Ptr && !vp.IsNil() {
		val := vp.Elem()
		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			switch typeField.Type.Kind() {
			case reflect.Struct:
				results = append(results, GetListJSONFieldObject(val.Field(i).Interface())...)
			default:
				jsonTags := utils.StringSlice(typeField.Tag.Get("json"), ",")
				if len(jsonTags) > 0 {
					if name := jsonTags[0]; name != "-" {
						results = append(results, name)
					}
				}
			}
		}
	}
	return results
}

func GetListXMLFieldObject(obj interface{}) []string {
	results := make([]string, 0)
	v := reflect.ValueOf(obj)
	base := Deref(v.Type())
	vp := reflect.New(base)
	if vp.Kind() == reflect.Ptr && !vp.IsNil() {
		val := vp.Elem()
		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			if _, ok := typeField.Tag.Lookup("xml"); ok {
				jsonFieldName := utils.StringSlice(typeField.Tag.Get("xml"), ",")[0]
				if jsonFieldName == "-" {
					continue
				}
				results = append(results, jsonFieldName)
			}
		}
	}
	return results
}

func ParseXMLStructValues(obj interface{}) map[string]interface{} {
	results := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	base := Deref(v.Type())
	vp := reflect.New(base)
	if vp.Kind() == reflect.Ptr && !vp.IsNil() {
		val := vp.Elem()
		values := reflect.ValueOf(obj).Elem()
		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			if _, ok := typeField.Tag.Lookup("xml"); ok {
				jsonFieldName := utils.StringSlice(typeField.Tag.Get("xml"), ",")[0]
				if jsonFieldName == "-" {
					continue
				}
				valueField := values.Field(i)
				results[jsonFieldName] = valueField.Interface()
			}
		}
	}
	return results
}

func SetZeroValueBeforeCreate(zeroColumns []string, setDateNowColumns []string, obj interface{}) {
	if zeroColumns != nil && len(zeroColumns) > 0 {
		s := reflect.ValueOf(obj).Elem()
		for i := 0; i < s.NumField(); i++ {
			typeField := s.Type().Field(i)
			jsonFieldName := utils.StringSlice(typeField.Tag.Get("json"), ",")[0]
			if utils.IsStringSliceContains(zeroColumns, jsonFieldName) {
				f := s.Field(i)
				switch f.Kind() {
				case reflect.String:
					f.SetString("")
				case reflect.Bool:
					f.SetBool(false)
				case reflect.Float64:
					f.SetFloat(0)
				case reflect.Int64:
					if setDateNowColumns != nil && len(setDateNowColumns) > 0 &&
						utils.IsStringSliceContains(setDateNowColumns, jsonFieldName) {
						f.SetInt(time.Now().Unix())
					} else {
						f.SetInt(0)
					}
				default:
					f.Set(reflect.Zero(f.Type()))
				}
			}
		}
	}
}

func FillValue2ObjString(columns []string, values []string, obj interface{}, mapColumn2Field map[string]string) {
	if len(columns) != len(values) {
		return
	}
	s := reflect.ValueOf(obj).Elem()
	for i := range columns {
		f := s.FieldByName(mapColumn2Field[columns[i]])
		switch f.Kind() {
		case reflect.String:
			f.SetString(values[i])
		case reflect.Bool:
			vBool, _ := strconv.ParseBool(values[i])
			f.SetBool(vBool)
		case reflect.Float64:
			vFloat, _ := strconv.ParseFloat(values[i], 64)
			f.SetFloat(vFloat)
		case reflect.Int64:
			vInt, _ := strconv.ParseInt(values[i], 10, 64)
			f.SetInt(vInt)
		}
	}
}

func MakeColumn2Field(obj interface{}) map[string]string {
	column2FieldName := make(map[string]string, 0)
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		typeField := s.Type().Field(i)
		jsonFieldName := utils.StringSlice(typeField.Tag.Get("json"), ",")[0]
		if jsonFieldName == "-" {
			continue
		}
		column2FieldName[jsonFieldName] = typeField.Name
	}
	return column2FieldName
}

func MakeObj2Map(obj interface{}) map[string]interface{} {
	mapResult := make(map[string]interface{}, 0)
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		typeField := s.Type().Field(i)
		jsonFieldName := utils.StringSlice(typeField.Tag.Get("json"), ",")[0]
		if jsonFieldName == "-" {
			continue
		}
		f := s.Field(i)
		mapResult[jsonFieldName] = f.Interface()
	}
	return mapResult
}

func MakeObj2Array(jsonFieldNames []string, maps map[string]string, obj interface{}) []string {
	results := []string{}
	if len(jsonFieldNames) != len(jsonFieldNames) {
		return results
	}
	s := reflect.ValueOf(obj).Elem()
	for i := range jsonFieldNames {
		jsonTag := utils.StringSlice(s.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		fieldName, ok := maps[jsonFieldNames[i]]
		if !ok {
			continue
		}
		f := s.FieldByName(fieldName)
		results = append(results, fmt.Sprintf("%v", f.Interface()))
	}
	return results
}

func GetIgnoreFieldsFromEnbleFields(obj interface{}, enbleFields []string) []string {
	allField := GetListJSONFieldObject(obj)
	return utils.SubStringArrays(allField, enbleFields)
}

func CompareObjectByField(obj interface{}, objCompare interface{}, fields []string) bool {
	if fields != nil && len(fields) > 0 {
		s := reflect.ValueOf(obj).Elem()
		sCompare := reflect.ValueOf(objCompare).Elem()
		for i := 0; i < s.NumField(); i++ {
			typeField := s.Type().Field(i)
			jsonFieldName := utils.StringSlice(typeField.Tag.Get("json"), ",")[0]
			if utils.IsStringSliceContains(fields, jsonFieldName) {
				if s.Field(i) != sCompare.Field(i) {
					return false
				}
			}
		}
	}
	return true
}
