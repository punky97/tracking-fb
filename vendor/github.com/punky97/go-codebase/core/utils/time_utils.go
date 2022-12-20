package utils

import (
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"strings"
	"time"
)

type Day struct {
	Start time.Time
	End   time.Time
}

const TIMEOUT_MINUTE = "m"
const TIMEOUT_HOUR = "h"
const TIMEOUT_DAY = "d"
const TIMEOUT_WEEK = "w"
const TIMEOUT_MONTH = "mo"
const TIMEOUT_YEAR = "y"

// Get timestamp of weekday by current timestamp
// @daySelected monday, tuesday, ...
// @time time.Now.Unix()
// @timeReturn time.Time

func GetDaysOfWeekByTimestamp(daySelected string, time time.Time) (timeReturn time.Time) {
	currentWeekday := time.Weekday()
	var weekdaySelected int

	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

	for key, day := range days {
		if day == strings.ToLower(daySelected) {
			weekdaySelected = key
		}
	}

	compareDay := int(weekdaySelected) - int(currentWeekday)
	timeReturn = time.AddDate(0, 0, compareDay)

	return
}

func GetNearestNextWeekday(daySelected string, time time.Time) (timeReturn time.Time) {
	for strings.ToLower(time.Weekday().String()) != daySelected {
		time = time.AddDate(0, 0, 1)
	}
	return time
}

func GetTimeoutInSecond(timeout int64, timeoutUnit string) int64 {
	var multiply int64
	switch timeoutUnit {
	case TIMEOUT_WEEK:
		multiply = 604800
	case TIMEOUT_DAY:
		multiply = 86400
	case TIMEOUT_HOUR:
		multiply = 3600
	case TIMEOUT_MONTH:
		multiply = 2592000
	case TIMEOUT_YEAR:
		multiply = 31536000
	case TIMEOUT_MINUTE:
		multiply = 60
	default:
		multiply = 60
	}

	return timeout * multiply
}

// return the datetime with format mm dd, yyyy
func FormatDateTimeMDY(t time.Time) string {
	return fmt.Sprintf("%d %d, %d", t.Month(), t.Day(), t.Year())
}

// return the datetime with format mm/dd/yyyy
func FormatDateTimeMDY2(t time.Time) string {
	return fmt.Sprintf("%d/%d/%d", t.Month(), t.Day(), t.Year())
}

// GetBeginDayTimestampFromTimestamp - get begin day timestamp from timestamp
func GetBeginDayTimestampFromTimestamp(timestamp int64) int64 {
	currentDate := time.Unix(timestamp, 0)
	beginningOfDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
	return beginningOfDate.Unix()
}

// return the datetime with format yyyy-mm-dd h:i:s
func FormatDateTimeYMDHSI(t time.Time) string {
	return fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func ParseTime(str string) (time.Time, error) {
	layout := "2006-01-02"
	return time.Parse(layout, str)
}

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		logger.BkLog.Infof("%v took too much time %v", name, elapsed)
	}
}

// FormatDate -- return the datetime with format yyyy-mm-dd h:i:s
func FormatDate(t time.Time) string {
	return fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
}

// GetTimeNowMilisecond --
func GetTimeNowMilisecond() int64 {
	now := time.Now()
	nanos := now.UnixNano() / 1000000
	return nanos
}

// GetTimeStampsFromTimeRange --
func GetTimeStampsFromTimeRange(timeRangeStr string, fromTime int64, toTime int64) (timestamps []int64) {
	switch timeRangeStr {
	case "custom":
		timestamps = GetTimeStamps(fromTime, toTime)
	default:
		timeRange, err := (&StrTimeRange{Str: timeRangeStr}).GetTimeRange()
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error GetTimeRange: %v", err), "time_range", timeRange)
			return
		}
		timestamps = GetTimeStamps(timeRange.From.Unix(), timeRange.To.Unix())
	}

	return
}

// GetTimeStamps --
func GetTimeStamps(from int64, to int64) []int64 {

	fromTime := time.Unix(from, 0)
	beginningOfFromTime := time.Date(fromTime.Year(), fromTime.Month(), fromTime.Day(), 0, 0, 0, 0, time.UTC)

	toTime := time.Unix(to, 0)
	beginningOfToTime := time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 0, 0, 0, 0, time.UTC)

	timeStamps := []int64{}
	tempTime := beginningOfFromTime

	for tempTime.Before(beginningOfToTime) || tempTime.Equal(beginningOfToTime) {
		timeStamps = append(timeStamps, tempTime.Unix())
		tempTime = tempTime.Add(24 * time.Hour)
	}

	return timeStamps
}
