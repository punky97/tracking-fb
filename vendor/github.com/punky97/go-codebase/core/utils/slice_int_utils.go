package utils

import (
	"github.com/spf13/cast"
)

func ConvertStringSliceToIntSlice(in []string) []int64 {
	res := make([]int64, 0)
	for i := range in {
		res = append(res, cast.ToInt64(in[i]))
	}
	return res
}

func ConvertIntSliceToStringSlice(in []int64) []string {
	res := make([]string, 0)
	for i := range in {
		res = append(res, cast.ToString(in[i]))
	}
	return res
}

func Contains(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func IntersectionStringArrays(array1 []string, array2 []string) []string {
	arrayResult := []string{}
	mx := map[string]bool{}
	for _, x := range array2 {
		mx[x] = true
	}
	for _, x := range array1 {
		if _, ok := mx[x]; ok {
			arrayResult = append(arrayResult, x)
		}
	}
	return arrayResult
}

func IntersectionArrays(array1 []int64, array2 []int64) []int64 {
	arrayResult := []int64{}
	mx := map[int64]bool{}
	for _, x := range array2 {
		mx[x] = true
	}
	for _, x := range array1 {
		if _, ok := mx[x]; ok {
			arrayResult = append(arrayResult, x)
		}
	}
	return arrayResult
}

func SubInt64Arrays(src []int64, sub []int64) []int64 {
	if sub == nil || len(sub) <= 0 {
		return src
	}
	mx := map[int64]bool{}
	for _, x := range sub {
		mx[x] = true
	}
	arrayResult := []int64{}
	for _, x := range src {
		if _, ok := mx[x]; !ok {
			arrayResult = append(arrayResult, x)
		}
	}
	return arrayResult
}

func RemoveDuplicatesInt64(elements []int64) []int64 {
	// Use map to record duplicates as we find them.
	encountered := map[int64]bool{}
	result := make([]int64, 0)

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

func SliceIntoBatches(source []int64, batchLength int) [][]int64 {
	batches := [][]int64{}
	batch := []int64{}
	for _, data := range source {
		batch = append(batch, data)
		if len(batch) >= batchLength {
			batches = append(batches, batch)
			batch = []int64{}
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

func SumMapsArrayInt64(source map[interface{}][]int64) (result int64) {
	for _, re := range source {
		for _, count := range re {
			result += count
		}
	}
	return
}
