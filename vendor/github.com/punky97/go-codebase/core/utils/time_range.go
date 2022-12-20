package utils

import (
	"errors"
	"fmt"
	"time"
)

// TimeRange --
type TimeRange struct {
	From time.Time
	To   time.Time
}

// StrTimeRange --
type StrTimeRange struct {
	Str  string
	From time.Time
}

// GetTimeRange --
func (t *StrTimeRange) GetTimeRange() (timeRange *TimeRange, err error) {
	var timestampFrom, timestampTo int64

	now := time.Now()
	beginningOfToDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	firstDayOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	switch t.Str {
	case "today":
		timestampFrom = beginningOfToDay.Unix()
		timestampTo = beginningOfToDay.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "yesterday":
		// Back to one day
		startDate := beginningOfToDay.Add(-24 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "7_days_ago":
		// Back seven days
		startDate := beginningOfToDay.Add(-168 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "30_days_ago":
		// Back 30 days
		startDate := beginningOfToDay.Add(-720 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "90_days_ago":
		// Back 90 days
		startDate := beginningOfToDay.Add(-2160 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "weekly":
		if weekDay := int(now.Weekday()); weekDay == 1 {
			// Now is monday, get only today's statistics
			timestampFrom = beginningOfToDay.Unix()
			timestampTo = beginningOfToDay.Add(24*time.Hour - 1*time.Second).Unix()
		} else {
			var dayAgo int
			// Now is sunday
			if dayAgo = weekDay; dayAgo == 0 {
				dayAgo = 7
			}
			timestampFrom = beginningOfToDay.Add(-24 * time.Duration(dayAgo-1) * time.Hour).Unix()
			timestampTo = beginningOfToDay.Add(24*time.Hour - 1*time.Second).Unix()
		}
		break

	case "monthly":
		lastDayOfMonth := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC)
		timestampFrom = firstDayOfThisMonth.Unix()
		timestampTo = lastDayOfMonth.Unix()
		break

	case "last_month":
		lastDayOfMonth := firstDayOfThisMonth.Add(-1 * time.Second)
		firstDayOfMonth := time.Date(lastDayOfMonth.Year(), lastDayOfMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		timestampFrom = firstDayOfMonth.Unix()
		timestampTo = lastDayOfMonth.Unix()
		break

	case "last_week":
		var dayToMinus int
		today := time.Now()
		// Get same day last week
		dayLastWeek := today.AddDate(0, 0, -7)
		// If day last week is Sunday, minus 6 days to get last Monday
		if dayLastWeek.Weekday() == 0 {
			dayToMinus = 6
		} else {
			// minus (weekday - 1) to get last Monday due to Monday is 1
			dayToMinus = int(dayLastWeek.Weekday() - 1)
		}
		mondayLastWeek := dayLastWeek.AddDate(0, 0, -dayToMinus)
		sundayLastWeek := mondayLastWeek.AddDate(0, 0, 6)
		timestampFrom = time.Date(mondayLastWeek.Year(), mondayLastWeek.Month(), mondayLastWeek.Day(), 0, 0, 0, 0, time.UTC).Unix()
		timestampTo = time.Date(sundayLastWeek.Year(), sundayLastWeek.Month(), sundayLastWeek.Day(), 23, 59, 59, 0, time.UTC).Unix()
		break

	case "fort_night":
		timestampFrom = beginningOfToDay.AddDate(0, 0, -14).Unix()
		timestampTo = beginningOfToDay.Add(24*time.Hour - 1*time.Second).Unix()
		break

	case "last_7_days":
		startTime := time.Now()
		if t.From.IsZero() != true {
			startTime = t.From
		}
		startDate := startTime.Add(-168 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(168*time.Hour - 1*time.Second).Unix()
		break

	case "last_30_days":
		startTime := time.Now()
		if t.From.IsZero() != true {
			startTime = t.From
		}
		startDate := startTime.Add(-720 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(720*time.Hour - 1*time.Second).Unix()
		break

	case "last_60_days":
		startTime := time.Now()
		if t.From.IsZero() != true {
			startTime = t.From
		}
		startDate := startTime.Add(-1440 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(1440*time.Hour - 1*time.Second).Unix()
		break

	case "last_90_days":
		startTime := time.Now()
		if t.From.IsZero() != true {
			startTime = t.From
		}
		startDate := startTime.Add(-2160 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(2160*time.Hour - 1*time.Second).Unix()
		break

	case "last_1_hours":
		startDate := time.Now().Add(-1 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(1*time.Hour - 5*time.Minute).Unix()
		break

	case "last_4_hours":
		startDate := time.Now().Add(-4 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(4*time.Hour - 10*time.Minute).Unix()
		break

	case "last_24_hours":
		startDate := now.Add(-24 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = now.Unix()
		break

	case "previous_hour":
		startDate := time.Now().Truncate(time.Hour).Add(-1 * time.Hour)
		timestampFrom = startDate.Unix()
		timestampTo = startDate.Add(1 * time.Hour).Unix()
		break
	default:
		err = errors.New(fmt.Sprintf("This time range `%v` has not supported", t.Str))
	}

	timeRange = &TimeRange{
		From: time.Unix(timestampFrom, 0),
		To:   time.Unix(timestampTo, 0),
	}

	return
}

// CleanUpTimeByType --
func CleanUpTimeByType(timeType string, statisticTime int64) int64 {
	now := time.Now()
	timeClean := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var timeCleanUnix int64

	if statisticTime != 0 {
		now := time.Unix(statisticTime, 0)
		timeClean = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	switch timeType {
	case "total":
		timeCleanUnix = 0
		break

	case "day":
		timeCleanUnix = timeClean.Unix()
		break

	case "week":
		if weekDay := int(now.Weekday()); weekDay == 1 {
			// Now is monday, get only today's statistics
			timeCleanUnix = timeClean.Unix()
		} else {
			var dayAgo int
			// Now is sunday
			if dayAgo = weekDay; dayAgo == 0 {
				dayAgo = 7
			}
			timeCleanUnix = timeClean.Add(-24 * time.Duration(dayAgo-1) * time.Hour).Unix()
		}

		break

	case "month":
		currentYear, currentMonth, _ := now.Date()
		currentLocation := now.Location()

		firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

		timeCleanUnix = firstOfMonth.Unix()
		break

	default:
		timeCleanUnix = timeClean.Unix()
	}
	return timeCleanUnix
}
