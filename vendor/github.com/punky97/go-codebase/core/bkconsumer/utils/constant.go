package utils

const (
	// ProcessOKCode marks message as done
	ProcessOKCode int64 = 200
	// ProcessFailRetryCode marks message as fail and retry
	ProcessFailRetryCode = 500
	// ProcessFailDropCode marks message as fail and drop
	ProcessFailDropCode = 400
	// ProcessFailReproduceCode --
	ProcessFailReproduceCode = 302
	// ProcessFailForceReproduceCode enable reproduce without increate retry times
	ProcessFailForceReproduceCode = 301

	// Drop message reason
	DropRetryLimitExceed = "retry limit exceeded"
	DropRedeliveryFail   = "redeliver fail"
	DropFail             = "process fail"
	DropInternalError    = "internal server error"
)
