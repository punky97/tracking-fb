package tokenmanager

import "fmt"

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("Code %v. %v", e.Code, e.Message)
}

func NewError(msg string, code int) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}
