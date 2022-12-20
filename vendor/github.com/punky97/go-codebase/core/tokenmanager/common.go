package tokenmanager

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/punky97/go-codebase/core/utils"
)

// ParseToken -- parse token
func ParseToken(tokenString string) (authToken *jwt.Token, err error) {
	return utils.ParseToken(tokenString, &TokenClaims{})
}
