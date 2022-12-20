package utils

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/punky97/go-codebase/core/logger"
	"time"
)

func DoSomethingWithRetry(doAction func() (interface{}, error), timesRetry int, description string) interface{} {
	for i := 0; i < timesRetry; i++ {
		res, err := doAction()
		if err == nil || err == redis.Nil {
			return res
		}
		logger.BkLog.Error(fmt.Sprintf("Do %v caught err", description), "err", err)
		time.Sleep(200)
	}
	return nil
}
