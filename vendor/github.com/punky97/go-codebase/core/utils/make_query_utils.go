package utils

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"reflect"
	"strings"
)

const FieldSelectHeader = "x-field-select-dms"
const DefaultLimitHeader = "x-default-limit-dms"
const DefaultSelectHeader = "x-default-select-dms"
const UseAllFieldHeader = "x-use-all-field-dms"
const LimitHeader = "x-limit-dms"
const OffsetHeader = "x-offset-dms"
const UseAllWithoutFieldHeader = "x-use-all-without-field-dms"

/**
There are functions make proto message to mysql query
*/
const (
	FieldConditionEq      = "condition"
	FieldConditionGt      = "greater"
	FieldConditionLt      = "less"
	FieldConditionNeq     = "neq"
	FieldConditionLike    = "like"
	FieldOrConditionLike  = "or_like"
	FieldInCondition      = "in"
	FieldNotInCondition   = "not_in"
	FieldNotNullCondition = "not_null"
	FieldNullCondition    = "is_null"
	FieldLimitCondition   = "limit"
	FieldPageCondition    = "page"
	FieldSortCondition    = "sort"
	FieldDisableLimit     = "disable_limit"
	FieldArrayConditionEq		= "array_condition"
	FieldArrayConditionGt		= "array_greater"
	FieldArrayConditionLt		= "array_less"
	FieldArrayConditionNeq		= "array_neq"
	FieldArrayConditionLike		= "array_like"
	FieldArrayConditionOrLike	= "array_or_like"
)

/**
	GetSortQuery - Make sort query from proto and sort string request
  params:
		- sort: value list sort field - "created_at:desc"
    - input: proto from mysql table
    - pre: You want to make query with pre. Example `select a.id .. from shops as a where a.id = ...". It effective for join table
*/
func GetSortQuery(sort string, input interface{}, pre string) string {
	query := ""
	listField := GetListJSONFieldObject(input)
	orders := []string{"asc", "desc"}
	sortFieldOrders := strings.Split(sort, ",")
	for _, sortFieldOrder := range sortFieldOrders {
		fieldOrder := strings.Split(sortFieldOrder, ":")
		field := fieldOrder[0]
		order := "asc"
		if len(fieldOrder) > 1 && (fieldOrder[1] == "asc" || fieldOrder[1] == "desc") {
			order = fieldOrder[1]
		}
		if IsStringSliceContains(listField, field) && IsStringSliceContains(orders, order) {
			if pre != "" {
				query = fmt.Sprintf("%v,%v.%v %v", query, pre, field, order)
			} else {
				query = fmt.Sprintf("%v,%v %v", query, field, order)
			}
		}
	}
	if len(query) > 0 {
		query = query[1:]
	}
	return query
}

/**
	GetOffsetQuery - Make limit, offset query from proto and sort string request
  params:
		- page (int64)
    - pageSize (int64)
*/
func GetOffsetQuery(page int64, pageSize int64) string {
	limit := int64(50)
	offset := int64(0)
	if pageSize > 0 {
		limit = pageSize
	}
	if page > 1 {
		offset = (page - 1) * limit
	}
	result := fmt.Sprintf(" limit %v offset %v", limit, offset)
	return result
}

func MakeSQLQuery(protoObj proto.Message, pre string, protoOrigin proto.Message,
	enableLimitAndSort bool, more string, moreArgs []interface{}) (string, []interface{}) {
	conditionQuery := ""
	var enableLimit = true
	valueArgs := []interface{}{}
	reflectValue := reflect.ValueOf(protoObj).Elem()
	var limit int64
	var page int64
	var sort string
	for i := 0; i < reflectValue.NumField(); i++ {
		jsonTag := StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		reflectCurrentFieldValue := reflectValue.Field(i)
		if IsZero(reflectCurrentFieldValue) {
			continue
		}
		value := reflectValue.Field(i)
		kind := value.Kind()
		switch jsonTag {
		case FieldConditionEq:
			strQuery, argsEl := MakeMysqlQueryCondition(value.Interface(), pre, "=")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionNeq:
			strQuery, argsEl := MakeMysqlQueryCondition(value.Interface(), pre, "!=")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionGt:
			strQuery, argsEl := MakeMysqlQueryCondition(value.Interface(), pre, ">")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionLt:
			strQuery, argsEl := MakeMysqlQueryCondition(value.Interface(), pre, "<")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionLike:
			strQuery, argsEl := MakeMysqlQueryCondition(value.Interface(), pre, "LIKE")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldOrConditionLike:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, "LIKE", "OR")
			if strQuery != "" {
				conditionQuery = AddConditionQueryOperator(conditionQuery, strQuery, "AND", true)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldInCondition:
			strQuery, argsEl := MakeMysqlQueryInStruct(value.Interface(), pre, "IN")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldNotInCondition:
			strQuery, argsEl := MakeMysqlQueryInStruct(value.Interface(), pre, "NOT IN")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldNotNullCondition:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					if pre == "" {
						conditionQuery = addConditionQuery(conditionQuery,
							fmt.Sprintf("%v IS NOT NULL", value.Index(i).Interface()))
					} else {
						conditionQuery = addConditionQuery(conditionQuery,
							fmt.Sprintf("%v.%v IS NOT NULL", pre, value.Index(i).Interface()))
					}

				}
			}
		case FieldNullCondition:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					if pre == "" {
						conditionQuery = addConditionQuery(conditionQuery,
							fmt.Sprintf("%v IS NULL", value.Index(i).Interface()))
					} else {
						conditionQuery = addConditionQuery(conditionQuery,
							fmt.Sprintf("%v.%v IS NULL", pre, value.Index(i).Interface()))
					}

				}
			}
		case FieldLimitCondition:
			limit = getInt64FromValue(value)
		case FieldPageCondition:
			page = getInt64FromValue(value)
		case FieldSortCondition:
			if kind == reflect.String {
				sort = value.Interface().(string)
			}
		case FieldDisableLimit:
			if !IsZero(value) {
				enableLimit = false
			}
		case FieldArrayConditionEq:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryCondition(value.Index(i).Interface(), pre, "=")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionGt:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryCondition(value.Index(i).Interface(), pre, ">")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionLt:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryCondition(value.Index(i).Interface(), pre, "<")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionNeq:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryCondition(value.Index(i).Interface(), pre, "!=")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionLike:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryCondition(value.Index(i).Interface(), pre, "LIKE")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		}

	}
	if more != "" {
		conditionQuery = AddConditionQueryOperator(conditionQuery, more, "AND", false)
	}
	if len(moreArgs) > 0 {
		valueArgs = append(valueArgs, moreArgs...)
	}
	if enableLimitAndSort {
		if sort != "" {
			sortQueryStr := GetSortQuery(sort, protoOrigin, pre)
			if sortQueryStr != "" {
				conditionQuery = conditionQuery + " ORDER BY " + sortQueryStr
			}
		}
		if enableLimit {
			conditionQuery = conditionQuery + " " + GetOffsetQuery(page, limit)
		}
	}

	return conditionQuery, valueArgs
}

/**
	MakeMysqQueryAll - This is func create all query for each field of proto
  params:
		- protoObj (proto.Message): proto you want to make query. Is has field: "condition", "greater" ...
    - pre (string): you want to query with multi tables. This field very effective. Its make build query with "pre. ...".
        Example `select a.id .. from shops as a where a.id = ...". It effective for join table
    - protoOrigin (proto.Message): This is origin proto from table. Example: core_models.Shop
    - enableLimit (bool): if value is false, this func doesn't make limit, offset query
*/
func MakeMysqQueryAll(protoObj proto.Message, pre string, protoOrigin proto.Message,
	enableLimit bool) (string, []interface{}) {
	return MakeSQLQuery(protoObj, pre, protoOrigin, enableLimit, "", nil)
}

/**
	MakeMysqlQueryCondition - This is func create query for each object, add to origin query with and
  params:
		- protoObj (proto.Message): proto you want to make query.
    - pre (string): you want to query with multi tables. This field very effective. Its make build query with "pre. ...".
        Example `select a.id .. from shops as a where a.id = ...". It effective for join table
    - condition (proto.Message): = != < >
*/
func MakeMysqlQueryCondition(protoObj interface{}, pre string, condition string) (string, []interface{}) {
	return MakeMysqlQueryConditionOperator(protoObj, pre, condition, "AND")
}

/**
	MakeMysqlQueryConditionOperator - This is func same MakeMysqlQueryCondition but this can custom operator (and, or) for add to origin query
  params:
		- protoObj (proto.Message): proto you want to make query.
    - pre (string): you want to query with multi tables. This field very effective. Its make build query with "pre. ...".
        Example `select a.id .. from shops as a where a.id = ...". It effective for join table
    - condition (string): = != < >
    - operator (string): and, or
*/
func MakeMysqlQueryConditionOperator(protoObj interface{}, pre string, condition string, operator string) (string, []interface{}) {
	conditionQuery := ""
	valueArgs := make([]interface{}, 0)
	reflectValue := reflect.ValueOf(protoObj).Elem()
	for i := 0; i < reflectValue.NumField(); i++ {
		jsonTag := StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		reflectCurrentFieldValue := reflectValue.Field(i)
		if IsZero(reflectCurrentFieldValue) {
			continue
		}
		var fieldName string
		if pre != "" {
			fieldName = fmt.Sprintf("%v.%v", pre, jsonTag)
		} else {
			fieldName = jsonTag
		}

		if condition == "!=" {
			conditionQuery = AddConditionQueryOperator(conditionQuery, fmt.Sprintf("(%v %s ? OR %v IS NULL)", fieldName, condition, fieldName), operator, false)
		} else {
			conditionQuery = AddConditionQueryOperator(conditionQuery, fmt.Sprintf("%v %s ?", fieldName, condition), operator, false)
		}
		valueArgs = append(valueArgs, reflectCurrentFieldValue.Interface())
	}
	return conditionQuery, valueArgs
}

func MakeMysqlQueryInStruct(obj interface{}, pre string, condition string) (string, []interface{}) {
	reflectValue := reflect.ValueOf(obj).Elem()
	conditionQuery := ""
	args := make([]interface{}, 0)
	for i := 0; i < reflectValue.NumField(); i++ {
		jsonTag := StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		queryStr, argsEl := MakeMysqlQueryIn(reflectValue.Field(i).Interface(), pre, jsonTag, condition)
		conditionQuery = addConditionQuery(conditionQuery, queryStr)
		args = JoinArrInterface(args, argsEl)
	}
	return conditionQuery, args
}

func MakeMysqlQueryIn(obj interface{}, pre string, fieldName string, condition string) (string, []interface{}) {
	value := reflect.ValueOf(obj)
	kind := value.Kind()
	if kind != reflect.Slice {
		return "", nil
	}
	valueArgs := make([]interface{}, 0)
	questionMarksIn := make([]string, 0)
	if value.Len() < 1 {
		return "", nil
	}
	for i := 0; i < value.Len(); i++ {
		questionMarksIn = append(questionMarksIn, "?")
		valueArgs = append(valueArgs, value.Index(i).Interface())
	}

	if pre == "" {
		return fmt.Sprintf("%v %v (%v)", fieldName, condition, strings.Join(questionMarksIn, ",")), valueArgs
	}
	return fmt.Sprintf("%v.%v %v (%v)", pre, fieldName, condition, strings.Join(questionMarksIn, ",")), valueArgs
}

func addConditionQuery(origin string, more string) string {
	return AddConditionQueryOperator(origin, more, "AND", false)
}

func AddConditionQueryOperator(origin string, more string, operator string, private bool) string {
	if more == "" {
		return origin
	}
	if private {
		more = fmt.Sprintf("(%v)", more)
	}
	if origin == "" && more == "" {
		return ""
	}
	if origin != "" {
		return fmt.Sprintf("%v %v %v", origin, operator, more)
	}
	return more
}

// Use to get updated fields bit mask from reflectValue, which was gotten from protobuf
// bit mask will let we know that which field was updated, which fields wasn't.
func GetUpdatedFieldsBitMask(reflectValue reflect.Value) ([]bool, bool) {
	numField := reflectValue.NumField()
	bitMask := make([]bool, numField)
	isDefault := true
	for i := 0; i < numField; i++ {
		if reflectValue.Type().Field(i).Name != "UpdatedField" {
			continue
		}
		isDefault = false
		tempBitMask, _ := reflectValue.Field(i).Interface().([]bool)
		if len(tempBitMask) != numField {
			isDefault = true
			break
		}

		var countDiff = 0
		for _, i := range tempBitMask {
			if i {
				countDiff++
			}
		}

		if countDiff == 0 {
			isDefault = true
			break
		}

		bitMask = tempBitMask
		break
	}
	// if have no info of updatedFields/no different fields => default update all field
	if isDefault {
		for i := range bitMask {
			bitMask[i] = true
		}
	}
	return bitMask, isDefault
}

// Some field doesn't need to save to db, so we ignore them.
func IsIgnoreThisField(fieldName string) bool {
	// TODO: Can we use config file instead of hard code?
	return fieldName == "UpdatedField" || fieldName == "OldAttribute"
}

func MakeSQLQueryAny(protoObj proto.Message, pre string, protoOrigin proto.Message,
	enableLimitAndSort bool, more string, moreArgs []interface{}) (string, []interface{}) {
	conditionQuery := ""
	var enableLimit = true
	valueArgs := []interface{}{}
	reflectValue := reflect.ValueOf(protoObj).Elem()
	var limit int64
	var page int64
	var sort string
	for i := 0; i < reflectValue.NumField(); i++ {
		jsonTag := StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		reflectCurrentFieldValue := reflectValue.Field(i)
		if IsZero(reflectCurrentFieldValue) {
			continue
		}
		value := reflectValue.Field(i)
		kind := value.Kind()
		switch jsonTag {
		case FieldConditionEq:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, "=","OR")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionNeq:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, "!=","OR")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionGt:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, ">","OR")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionLt:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, "<","OR")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldConditionLike:
			strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Interface(), pre, "LIKE","OR")
			if strQuery != "" {
				conditionQuery = addConditionQuery(conditionQuery, strQuery)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldInCondition:
			strQuery, argsEl := MakeMysqlQueryInStruct(value.Interface(), pre, "IN")
			if strQuery != "" {
				conditionQuery = AddConditionQueryOperator(conditionQuery, strQuery, "OR", false)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldNotInCondition:
			strQuery, argsEl := MakeMysqlQueryInStruct(value.Interface(), pre, "NOT IN")
			if strQuery != "" {
				conditionQuery = AddConditionQueryOperator(conditionQuery, strQuery, "OR", false)
				valueArgs = JoinArrInterface(valueArgs, argsEl)
			}
		case FieldNotNullCondition:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					if pre == "" {
						conditionQuery = AddConditionQueryOperator(conditionQuery,
							fmt.Sprintf("%v IS NOT NULL", value.Index(i).Interface()), "OR", false)
					} else {
						conditionQuery = AddConditionQueryOperator(conditionQuery,
							fmt.Sprintf("%v.%v IS NOT NULL", pre, value.Index(i).Interface()), "OR", false)
					}

				}
			}
		case FieldNullCondition:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					if pre == "" {
						conditionQuery = AddConditionQueryOperator(conditionQuery,
							fmt.Sprintf("%v IS NULL", value.Index(i).Interface()), "OR", false)
					} else {
						conditionQuery = AddConditionQueryOperator(conditionQuery,
							fmt.Sprintf("%v.%v IS NULL", pre, value.Index(i).Interface()), "OR", false)
					}

				}
			}
		case FieldLimitCondition:
			limit = getInt64FromValue(value)
		case FieldPageCondition:
			page = getInt64FromValue(value)
		case FieldSortCondition:
			if kind == reflect.String {
				sort = value.Interface().(string)
			}
		case FieldDisableLimit:
			if !IsZero(value) {
				enableLimit = false
			}
		case FieldArrayConditionEq:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Index(i).Interface(), pre, "=","OR")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionGt:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Index(i).Interface(), pre, ">","OR")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionLt:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Index(i).Interface(), pre, "<","OR")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionNeq:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Index(i).Interface(), pre, "!=","OR")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		case FieldArrayConditionLike:
			if kind == reflect.Slice {
				for i := 0; i < value.Len(); i++ {
					strQuery, argsEl := MakeMysqlQueryConditionOperator(value.Index(i).Interface(), pre, "LIKE","OR")
					if strQuery != "" {
						conditionQuery = addConditionQuery(conditionQuery, strQuery)
						valueArgs = JoinArrInterface(valueArgs, argsEl)
					}
				}
			}
		}
	}
	if more != "" {
		conditionQuery = AddConditionQueryOperator(conditionQuery, more, "AND", false)
	}
	if len(moreArgs) > 0 {
		valueArgs = append(valueArgs, moreArgs...)
	}
	if enableLimitAndSort {
		if sort != "" {
			sortQueryStr := GetSortQuery(sort, protoOrigin, pre)
			if sortQueryStr != "" {
				conditionQuery = conditionQuery + " ORDER BY " + sortQueryStr
			}
		}
		if enableLimit {
			conditionQuery = conditionQuery + " " + GetOffsetQuery(page, limit)
		}
	}

	return conditionQuery, valueArgs
}