package middleware

import (
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/urfave/negroni"
)

// NewBkRecovery --
func NewBkRecovery() *negroni.Recovery {
	recovery := negroni.NewRecovery()
	recovery.ErrorHandlerFunc = recoveryErrorHandlerFunc
	recovery.PrintStack = false
	return recovery
}

func recoveryErrorHandlerFunc(error interface{}) {
	logger.BkLog.Errorw(fmt.Sprintf("Recovery catch panic: %v", error))
}
