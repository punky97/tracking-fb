package utils

import "fmt"

// Find all ele in src but not in sub array
func SubStringArrays(src []string, sub []string) []string {
	if sub == nil || len(sub) <= 0 {
		return src
	}
	mx := map[string]bool{}
	for _, x := range sub {
		mx[x] = true
	}
	arrayResult := []string{}
	for _, x := range src {
		if _, ok := mx[x]; !ok {
			arrayResult = append(arrayResult, x)
		}
	}
	return arrayResult
}

// Find list all value of map
func GetListValueMapStringString(input map[string]string) (result []string) {
	for k := range input {
		result = append(result, input[k])
	}
	return
}

// Find Union of string array inputs (with remove duplicate)
func MereArrayDistinct(inputs ...[]string) []string {
	results := []string{}
	mapCheck := map[string]bool{}
	for i := range inputs {
		for j := range inputs[i] {
			if check, ok := mapCheck[inputs[i][j]]; !ok && !check {
				results = append(results, inputs[i][j])
				mapCheck[inputs[i][j]] = true
			}
		}
	}
	return results
}

// Find Union of int64 array inputs (with remove duplicate)
func MereArrayInt64Distinct(inputs ...[]int64) []int64 {
	results := make([]int64, 0)
	mapCheck := map[int64]bool{}
	for i := range inputs {
		for j := range inputs[i] {
			if check, ok := mapCheck[inputs[i][j]]; !ok && !check {
				results = append(results, inputs[i][j])
				mapCheck[inputs[i][j]] = true
			}
		}
	}
	return results
}

// CheckSlicesStringEqual - check all inputs same len and items
func CheckSlicesStringEqual(inputs ...[]string) bool {
	listMapCheck := make([]map[string]bool, 0)
	if len(inputs) < 2 {
		return true
	}
	lenCommon := len(inputs[0])
	for i := range inputs {
		if len(inputs[i]) != lenCommon {
			return false
		}
	}
	if len(inputs[0]) < 1 {
		return true
	}
	for i := range inputs {
		tmp := map[string]bool{}
		for j := range inputs[i] {
			tmp[inputs[i][j]] = true
		}
		listMapCheck = append(listMapCheck, tmp)
	}

	for i := 1; i < len(inputs); i++ {
		for j := range inputs[i] {
			if check, ok := listMapCheck[i][inputs[i][j]]; !ok || !check {
				return false
			}
		}
	}
	return true
}

func SliceContain(input []string, item string) bool {
	for _, str := range input {
		if str == item {
			return true
		}
	}
	return false
}

// Check if first slice has any intersect element with second slice.
func IsSliceIntersect(arr1, arr2 []string) bool {
	intersect := false

	for _, a := range arr1 {
		if intersect == true {
			break
		}
		for _, b := range arr2 {
			if a == b {
				intersect = true
				break
			}
		}
	}

	return intersect
}

func RemoveDuplicatesString(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := make([]string, 0)

	for v := range elements {
		if !encountered[elements[v]] {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func SliceStringToString(sl []string) (s string) {
	if len(sl) == 0 {
		return
	}
	for i := range sl {
		if i == 0 {
			s = sl[i]
		}
		s = fmt.Sprintf("%v, %v", s, sl[i])
	}
	return
}
