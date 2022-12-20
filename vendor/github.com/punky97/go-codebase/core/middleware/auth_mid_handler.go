package middleware

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/punky97/go-codebase/core/bkerr"
	"github.com/punky97/go-codebase/core/common"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/tokenmanager"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils"
	"io/ioutil"
	"net/http"
	"time"
)

type AuthMiddleware struct {
	authInfo transhttp.AuthInfo
	rc       *bkredis.RedisClient
	tokenDef []*tokenmanager.TokenDefinition
}

func NewAuthMiddleware(auth transhttp.AuthInfo, rc *bkredis.RedisClient, tokenDef []*tokenmanager.TokenDefinition) *AuthMiddleware {
	return &AuthMiddleware{
		authInfo: auth,
		rc:       rc,
		tokenDef: tokenDef,
	}
}

func (h *AuthMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !h.authInfo.Enable {
		next(rw, r)
		return
	}

	claims := &tokenmanager.TokenClaims{}
	token, err := h.parseToken(r, h.authInfo.TokenType)
	if err != nil {
		logger.BkLog.Warnf("Token is expired: %v", err)
		transhttp.RespondError(rw, bkerr.TokenExpiredCode, "Token expired")
		return
	}
	var ok bool
	// try to extract our custom claims from the token
	claims, ok = token.Claims.(*tokenmanager.TokenClaims)

	// check if the token is valid
	if !ok || claims == nil || token.Valid == false {
		logger.BkLog.Warnf("Verify auth failed, details: %v", err)
		transhttp.RespondError(rw, http.StatusUnauthorized, "Token is not valid")
		return
	}

	if !tokenmanager.IsGrantedTokenType(h.authInfo.TokenType, claims.TokenType) {
		logger.BkLog.Warnf("Token type not match: %v - %v", h.authInfo.TokenType, claims.TokenType)
		transhttp.RespondError(rw, http.StatusUnauthorized, "Token is not valid")
		return
	}

	if claims.ExpiresAt < time.Now().Unix() {
		tokenManager := tokenmanager.NewTokenManager(h.rc, h.tokenDef)
		// Expire
		// Delete expired token
		// extract hash token from request
		hashedToken, err := h.getHashedJWTToken(r)
		if err != nil {
			logger.BkLog.Warnf("[Authentication] Error when ExtractToken: %v", err)
			transhttp.RespondError(rw, bkerr.TokenExpiredCode, "Token expired")
			return
		}
		h.rc.Del(context.Background(), tokenManager.GetRedisTokenKey(h.authInfo.TokenType, hashedToken))
		transhttp.RespondError(rw, bkerr.TokenExpiredCode, "Token expired")
		return
	}

	// match token
	h.addInfoFromToken(r, claims)
	next(rw, r)
}

func (h *AuthMiddleware) parseToken(r *http.Request, tokenType string) (token *jwt.Token, err error) {
	tokenManager := tokenmanager.NewTokenManager(h.rc, h.tokenDef)
	// extract hash token from request
	hashedToken, err := h.getHashedJWTToken(r)
	if err != nil {
		return nil, fmt.Errorf("[parseToken-1] error when parse token %v, %v %v, details: %v", tokenType, r.Method, r.URL.Path, err)
	}

	encryptedTok, err := tokenManager.GetToken(r.Context(), tokenType, hashedToken)
	if err != nil {
		return nil, fmt.Errorf("[parseToken-2] error when GetToken %v, %v %v, details: %v", tokenType, r.Method, r.URL.Path, err)
	}

	// decrypted token
	rawToken, err := utils.DecryptAccessToken(encryptedTok)
	if err != nil {
		return nil, fmt.Errorf("[parseToken-3] error when DecryptAccessToken, details: %v", err)
	}

	// parse raw token to dgrijalva/jwt
	token, err = tokenmanager.ParseToken(rawToken)
	if err != nil {
		return nil, fmt.Errorf("[parseToken-4] error when ParseToken, rawToken: %v details: %v", rawToken, err)
	}

	return
}

func (p *AuthMiddleware) getHashedJWTToken(r *http.Request) (token string, err error) {
	buf, _ := ioutil.ReadAll(r.Body)
	token, err = tokenmanager.OAuth2Extractor.ExtractToken(r)
	r.MultipartForm = nil
	r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return
}

func (p *AuthMiddleware) addInfoFromToken(r *http.Request, token *tokenmanager.TokenClaims) {
	r.Header.Set(common.XUserIdHeader, fmt.Sprintf("%v", token.UserID))
	if len(token.Scopes) > 0 {
		r.Header.Set(common.XUserTypeHeader, token.Scopes[0])
	}
	r.Header.Set(common.XUserEmail, token.Email)
}
