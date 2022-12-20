package transhttp

import (
	"errors"
	"net/http"
)

var ErrorNotFound = errors.New("not found")
var ErrorTooManyRequest = errors.New("too many request")

func GetStatusCode(err error) int {
	switch err.Error() {
	case ErrorNotFound.Error():
		return http.StatusNotFound
	case ErrorTooManyRequest.Error():
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}