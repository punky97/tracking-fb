package tokenmanager

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils"
	"net/http"
	"strings"
)

// JWTAuthMid --
type JWTAuthMid struct {
	TokenType    string
	TokenManager TokenManager
	// RedisDBClient redisstorage.IClient
}

func (m *JWTAuthMid) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	// parse token
	token, err := m.parseToken(r)
	if err != nil {
		transhttp.RespondError(w, http.StatusInternalServerError, "Error when parse token")
		return
	}

	// try to extract our custom claims from the token
	claims, ok := token.Claims.(*TokenClaims)

	// check if the token is valid
	if !ok || token.Valid == false {
		logger.BkLog.Error("Token is not valid")
		transhttp.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if !strings.EqualFold(m.TokenType, claims.TokenType) {
		logger.BkLog.Errorf("Not match token type, token type in claims %v, expected %v", claims.TokenType, m.TokenType)
		transhttp.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	next(w, r)
}

func (m *JWTAuthMid) parseToken(r *http.Request) (token *jwt.Token, err error) {
	// extract hash token from request
	hashedToken, err := OAuth2Extractor.ExtractToken(r)
	if err != nil {
		return
	}

	// get encrypted token from redis
	encryptedTok, err := m.TokenManager.GetToken(r.Context(), m.TokenType, hashedToken)
	if err != nil {
		return
	}

	// decrypted token
	rawToken, err := utils.DecryptAccessToken(encryptedTok)
	if err != nil {
		return
	}

	// parse raw token to dgrijalva/jwt
	token, err = ParseToken(rawToken)
	if err != nil {
		return
	}
	return
}
