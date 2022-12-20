package utils

import "time"

func Retry(do func() error, retryTime int, sleep int) {
	for i := 0; i < retryTime; i++ {
		err := do()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
}
