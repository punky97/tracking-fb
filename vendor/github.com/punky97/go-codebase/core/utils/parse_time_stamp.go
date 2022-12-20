package utils

import (
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"strconv"
	"strings"
	"time"
)

// ParseTimeStamp -- parse time stamp to timeDay, timeWeek, timeMonth, timeYear
func ParseTimeStamp(timeStamp int64) (timeDay, timeWeek, timeMonth, timeYear int64, err error) {
	t := time.Unix(timeStamp, 0)

	// get day
	timeDay = int64(t.Day())

	// get timeWeek and time Year
	var strTimeWeek string
	thisYear, thisWeek := t.ISOWeek()
	strThisYear := strconv.Itoa(thisYear)
	strThisWeek := strconv.Itoa(thisWeek)
	if thisWeek < 10 {
		strTimeWeek = strThisYear + "0" + strThisWeek
	} else {
		strTimeWeek = strThisYear + strThisWeek
	}

	timeWeek, err = StringToInt64(strTimeWeek)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when convert timeWeek from string to int64, err: %v", err))
		return
	}
	timeYear = int64(thisYear)

	// get timeMonth
	arrayTime := strings.Split(t.String(), "-")
	if len(arrayTime) == 0 {
		logger.BkLog.Errorw(fmt.Sprintf("Len of array time is 0, err: ", err))
		return
	}
	timeMonth, err = StringToInt64(arrayTime[1])
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when convert timeMonth from string to int64, err ", err))
		return
	}
	return
}
