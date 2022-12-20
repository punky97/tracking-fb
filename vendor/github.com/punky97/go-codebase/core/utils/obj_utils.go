package utils

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Deref is Indirect for reflect.Types
func Deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

//CloneObjectSrcToDesc - copy value of object src to des
func CloneObjectSrcToDesc(src interface{}, des interface{}) {
	sSrc := reflect.ValueOf(src).Elem()
	mapValue := make(map[string]reflect.Value)
	for i := 0; i < sSrc.NumField(); i++ {
		mapValue[sSrc.Type().Field(i).Name] = sSrc.Field(i)
	}
	sDes := reflect.ValueOf(des).Elem()
	for i := 0; i < sDes.NumField(); i++ {
		if value, ok := mapValue[sDes.Type().Field(i).Name]; ok && sDes.Field(i).CanSet() {
			sDes.Field(i).Set(value)
		}
	}
}

func CorrectDataType(v interface{}, dataType reflect.Kind) interface{} {
	if v == nil {
		return nil
	}
	switch dataType {
	case reflect.String:
		return cast.ToString(v)
	case reflect.Int64:
		switch reflect.ValueOf(v).Kind() {
		case reflect.Ptr:
			if vi, ok := v.(*time.Time); ok {
				return vi.Unix()
			}
		case reflect.Struct:
			if vi, ok := v.(time.Time); ok {
				return vi.Unix()
			}
		}
		return cast.ToInt64(v)
	case reflect.Int32:
		return cast.ToInt32(v)
	case reflect.Float64:
		return cast.ToFloat64(v)
	case reflect.Float32:
		return cast.ToFloat32(v)
	case reflect.Uint64:
		return cast.ToUint64(v)
	case reflect.Uint32:
		return cast.ToUint32(v)
	case reflect.Bool:
		return cast.ToBool(v)
	}
	return nil
}

// CopyObjMapper - copy value of object src to des with mapping field
func CopyObjMapper(src interface{}, des interface{}, mapper map[string]string) {
	sSrc := reflect.ValueOf(src).Elem()
	mapValue := make(map[string]reflect.Value)
	for i := 0; i < sSrc.NumField(); i++ {
		jsonTagRaw := sSrc.Type().Field(i).Tag.Get("json")
		if jsonTagRaw == "" {
			continue
		}
		jsonTag := StringSlice(jsonTagRaw, ",")[0]
		mapValue[jsonTag] = sSrc.Field(i)
	}
	mapperReverse := map[string]string{}
	for k, v := range mapper {
		mapperReverse[v] = k
	}
	sDes := reflect.ValueOf(des).Elem()
	for i := 0; i < sDes.NumField(); i++ {
		jsonTagRaw := sDes.Type().Field(i).Tag.Get("json")
		if jsonTagRaw == "" {
			continue
		}
		jsonTag := StringSlice(jsonTagRaw, ",")[0]
		if field, ok := mapperReverse[jsonTag]; ok {
			if value, ok := mapValue[field]; ok && sDes.Field(i).CanSet() {
				correctData := CorrectDataType(value.Interface(), sDes.Field(i).Kind())
				if correctData != nil {
					sDes.Field(i).Set(reflect.ValueOf(correctData))
				}
			}
		}
	}
}

// ConvertObjToMapStringMapper - copy value of object src to des with mapping field
func ConvertObjToMapStringMapper(src interface{}, mapper map[string]string) map[string]string {
	sSrc := reflect.ValueOf(src).Elem()
	mapValue := make(map[string]interface{})
	for i := 0; i < sSrc.NumField(); i++ {
		jsonTag := StringSlice(sSrc.Type().Field(i).Tag.Get("json"), ",")[0]
		mapValue[jsonTag] = sSrc.Field(i).Interface()
	}
	mapperReverse := map[string]string{}
	for k, v := range mapper {
		mapperReverse[v] = k
	}
	des := make(map[string]string)
	for keyDes, keySrc := range mapperReverse {
		if value, ok := mapValue[keySrc]; ok {
			des[keyDes] = cast.ToString(value)
		}
	}
	return des
}

func JoinArrInterface(inputs ...[]interface{}) []interface{} {
	results := make([]interface{}, 0)
	for i := range inputs {
		for j := range inputs[i] {
			results = append(results, inputs[i][j])
		}
	}
	return results
}

// GetListJSONFieldObject - Get all json-tag field of object
func GetListJSONFieldObject(obj interface{}) []string {
	results := []string{}
	v := reflect.ValueOf(obj)
	base := Deref(v.Type())
	vp := reflect.New(base)
	if vp.Kind() == reflect.Ptr && !vp.IsNil() {
		val := vp.Elem()
		for i := 0; i < val.NumField(); i++ {
			typeField := val.Type().Field(i)
			results = append(results, StringSlice(typeField.Tag.Get("json"), ",")[0])
		}
	}
	return results
}

func GetFieldsDiff(obj1 interface{}, obj2 interface{}) (results []string) {
	//sSrc := reflect.ValueOf(obj1).Elem()
	//results = make([]string, 0)
	//mapValue := make(map[string]reflect.Value)
	//for i := 0; i < sSrc.NumField(); i++ {
	//	mapValue[sSrc.Type().Field(i).Name] = sSrc.Field(i)
	//}
	//sDes := reflect.ValueOf(obj2).Elem()
	//for i := 0; i < sDes.NumField(); i++ {
	//	if value, ok := mapValue[sDes.Type().Field(i).Name]; ok && !reflect.DeepEqual(value.Interface(),  sDes.Field(i).Interface()) {
	//		typeField := sDes.Type().Field(i)
	//		results = append(results, StringSlice(typeField.Tag.Get("json"), ",")[0])
	//	}
	//}
	return GetFieldsDiffWithInclude(obj1, obj2, nil, nil)
}

func GetFieldsDiffWithInclude(obj1 interface{}, obj2 interface{},
	includeFields []string, ignoreFields []string) (results []string) {
	enableInclude := false
	if len(includeFields) > 0 {
		enableInclude = true
	}
	includesMap := map[string]bool{}
	ignoreMap := map[string]bool{}
	for _, include := range includeFields {
		includesMap[include] = true
	}
	for _, ignore := range ignoreFields {
		ignoreMap[ignore] = true
	}

	sSrc := reflect.ValueOf(obj1).Elem()
	results = make([]string, 0)
	mapValue := make(map[string]reflect.Value)
	for i := 0; i < sSrc.NumField(); i++ {
		typeField := sSrc.Type().Field(i)
		jsonName := StringSlice(typeField.Tag.Get("json"), ",")[0]
		if _, ok := includesMap[jsonName]; enableInclude && !ok {
			continue
		}
		if _, ok := ignoreMap[jsonName]; ok {
			continue
		}
		mapValue[typeField.Name] = sSrc.Field(i)
	}
	sDes := reflect.ValueOf(obj2).Elem()
	for i := 0; i < sDes.NumField(); i++ {
		if value, ok := mapValue[sDes.Type().Field(i).Name]; ok && !reflect.DeepEqual(value.Interface(), sDes.Field(i).Interface()) {
			typeField := sDes.Type().Field(i)
			results = append(results, StringSlice(typeField.Tag.Get("json"), ",")[0])
		}
	}
	return
}

// IsZero - Check value is zero (pointer is nil, int is 0, ... )
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func:
		return v.IsNil()
	case reflect.Map, reflect.Slice:
		return v.Len() == 0
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && IsZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && IsZero(v.Field(i))
		}
		return z
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return IsZero(reflect.Indirect(v))
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func getInt64FromValue(v reflect.Value) int64 {
	switch v.Kind() {
	case reflect.Int64, reflect.Int, reflect.Int16, reflect.Int8:
		return int64(v.Interface().(int64))
	}
	return 0
}

// GetIgnoreColumnsFromEnableColumns --
func GetIgnoreColumnsFromEnableColumns(obj interface{}, enableFields []string) []string {
	allField := GetListJSONFieldObject(obj)
	return SubStringArrays(allField, enableFields)
}

// IncrByName --
func IncrByName(src interface{}, fieldName string, value int64) (isSet bool) {
	field := reflect.ValueOf(src).Elem().FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return false
	}
	switch field.Kind() {
	case reflect.Int64:
		field.Set(reflect.ValueOf(field.Interface().(int64) + value))
		break
	case reflect.Int32:
		field.Set(reflect.ValueOf(field.Interface().(int32) + int32(value)))
		break

	}
	return true
}

func SwitchMapToObj(src map[string]interface{}, obj interface{}) (err error) {
	b, err := json.Marshal(src)
	if err != nil {
		return errors.New("Error when Marshal map: " + err.Error())
	}
	return json.Unmarshal(b, obj)
}

func SetZeroValue(zeroColumns []string, obj interface{}) {
	if zeroColumns != nil && len(zeroColumns) > 0 {
		s := reflect.ValueOf(obj).Elem()
		for i := 0; i < s.NumField(); i++ {
			typeField := s.Type().Field(i)
			jsonFieldName := StringSlice(typeField.Tag.Get("json"), ",")[0]
			if IsStringSliceContains(zeroColumns, jsonFieldName) {
				f := s.Field(i)
				switch f.Kind() {
				case reflect.String:
					f.SetString("")
				case reflect.Bool:
					f.SetBool(false)
				case reflect.Float64:
					f.SetFloat(0)
				case reflect.Int64:
					f.SetInt(0)
				default:
					f.Set(reflect.Zero(f.Type()))
				}
			}
		}
	}
}

func ConvertMapString2Obj(rsym interface{}, input map[string]string) {
	sDes := reflect.ValueOf(rsym).Elem()
	for i := 0; i < sDes.NumField(); i++ {
		jsonTag := StringSlice(sDes.Type().Field(i).Tag.Get("json"), ",")[0]
		if value, ok := input[jsonTag]; ok && sDes.Field(i).CanSet() {
			if jsonTag == "updated_at" {
				create, _ := time.Parse(time.RFC3339, value)
				sDes.Field(i).Set(reflect.ValueOf(create))
			} else {
				correctData := CorrectDataType(value, sDes.Field(i).Kind())
				if correctData != nil {
					sDes.Field(i).Set(reflect.ValueOf(correctData))
				}
			}
		}
	}
}

// MapArrayIntoStruct --
func MapArrayIntoStruct(rawData []interface{}, dest interface{}, skipError bool) error {
	reflectValue := reflect.ValueOf(dest).Elem()
	for i := 0; i < len(rawData); i++ {
		if rawData[i] != nil {
			destinationFName := reflectValue.Type().Field(i).Name
			reflectTypeF := reflectValue.FieldByName(destinationFName).Kind()
			switch reflectTypeF {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val, err := cast.ToUint64E(rawData[i])
				if err != nil && !skipError {
					return err
				}
				reflectValue.FieldByName(destinationFName).SetUint(val)
			case reflect.Int, reflect.Int32, reflect.Int16, reflect.Int64:
				var val int64
				var err error
				valPtr, ok := rawData[i].(time.Time)
				if ok {
					val = valPtr.Unix()
				} else {
					val, err = cast.ToInt64E(rawData[i])
					if err != nil && !skipError {
						return err
					}
				}
				reflectValue.FieldByName(destinationFName).SetInt(val)
			case reflect.String:
				reflectValue.FieldByName(destinationFName).SetString(cast.ToString(rawData[i]))
			case reflect.Float64, reflect.Float32:
				val, err := cast.ToFloat64E(rawData[i])
				if err != nil && !skipError {
					return err
				}
				reflectValue.FieldByName(destinationFName).SetFloat(val)
			case reflect.Bool:
				reflectValue.FieldByName(destinationFName).SetBool(cast.ToBool(rawData[i]))
			}
		}
	}
	return nil
}

// GetColumnNameByID --
func GetColumnNameByID(src interface{}, idx int) string {
	reflectValue := reflect.ValueOf(src).Elem()
	destinationFName := reflectValue.Type().Field(idx).Name
	return destinationFName
}

// GetColumnNamesByIndices --
func GetColumnNamesByIndices(src interface{}, indices []int) []string {
	reflectValue := reflect.ValueOf(src).Elem()
	columnNames := make([]string, len(indices))
	for i := 0; i < len(indices); i++ {
		columnNames[i] = strings.Split(reflectValue.Type().Field(indices[i]).Tag.Get("json"), ",")[0]
	}
	return columnNames
}
