package bkerr

import "errors"

// BkError -- beekit error
type BkError struct {
	Code    int64  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

var (
	ErrServerError                    = errors.New("Server Error")
	ErrParseRequestBody               = errors.New("Failed to parse request body")
	ErrParseRequestParams             = errors.New("Failed to parse request params")
	ErrInvalidToken                   = errors.New("Token is not valid")
	ErrUnknownTokenType               = errors.New("Unknown token type")
	ErrRedisMissingTokenKeyPrefix     = errors.New("Missing key prefix for token")
	ErrRedisMustSetExpirationForToken = errors.New("Must set expiration time for token")
	ErrTokenExpired                   = errors.New("Token is expired")
	ErrInvalidInput                   = errors.New("Invalid input")
	ErrPermissionDenied               = errors.New("Permission denied")
	ErrInvalidRequest                 = errors.New("Invalid request")
	ErrNotFound                       = errors.New("Not found")
	ErrInvalidCredentials             = errors.New("Username or password is not valid")
	ErrAccountNotAvailable            = errors.New("Account is not available")
	ErrShopNotAvailable               = errors.New("Shop is not available")
	ErrAccountDisabled                = errors.New("Account is disabled")
	ErrInvalidAccount                 = errors.New("Invalid account")
	ErrUnknownActionType              = errors.New("Unknown action type")
	ErrUnknownAppCode                 = errors.New("Unknown app code")
	ErrInvalidShop                    = errors.New("Invalid shop")
	ErrInvalidAppShop                 = errors.New("Invalid app_shop")
	ErrDomainNotAvailable             = errors.New("Domain is not available")
	ErrPasswordIncorrect              = errors.New("Password is incorrect")

	unknownErrCode = BkError{
		Code:    999,
		Message: "Unknown error",
	}

	TokenExpiredCode = 456
)
