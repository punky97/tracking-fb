package utils

import (
	"fmt"
	"math"
	"strconv"

	"github.com/spf13/cast"
)

type numberConvertion uint8

const (
	toInt64   numberConvertion = 0
	toFloat64 numberConvertion = 1
)

func numberConverter(value interface{}, numType numberConvertion) (interface{}, bool) {
	tempValue := fmt.Sprint(value)
	switch numType {
	case toInt64:
		if number, ok := strconv.ParseInt(tempValue, 10, 64); ok == nil {
			return number, true
		}

	case toFloat64:

		if number, ok := strconv.ParseFloat(tempValue, 64); ok == nil {
			return number, true
		}
	}

	return nil, false
}

func ConvertToInt64(value interface{}) int64 {
	fmt.Sprintln("Value: ", value)
	number, _ := numberConverter(value, toInt64)

	return cast.ToInt64(number)
}

func ConvertFloat64(value interface{}) float64 {
	number, _ := numberConverter(value, toFloat64)

	return cast.ToFloat64(number)
}

func RoundFloat(x, unit float64) float64 {
	return math.Ceil(x/unit) * unit
}

func CalculateFloatAverage(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	var total float64 = 0
	n := 0
	for i := range s {
		total += s[i]
		n++
	}
	return total / float64(n)
}
