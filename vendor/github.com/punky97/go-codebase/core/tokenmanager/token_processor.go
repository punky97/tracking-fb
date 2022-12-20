package tokenmanager

import (
	"context"
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"github.com/punky97/go-codebase/core/utils/cookie"
	"net/http"
	"time"
)

// TokenProcessor --
type TokenProcessor struct {
	TokenManager   *TokenManager
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	CookieClient   *cookie.Client

	TokenType   string
	TTL         *time.Duration // expired time redis key
	TokenSource string         // token creating source (eg: custom_login (hive))
}

// Process --
func (p *TokenProcessor) Process(ctx context.Context, req *AccessTokenRequest) (hashToken string, err error) {
	var encryptedToken string

	// generate token
	ok := p.TokenManager.IsValidTokenType(req.TokenType)
	if !ok {
		return hashToken, NewError("token not supported", http.StatusBadRequest)
	}

	claims := p.TokenManager.NewAccessTokenClaims(req)

	hashToken, encryptedToken, err = p.GenerateAccessToken(claims)
	if err != nil {
		logger.BkLog.Error("TokenProcessor::Process error 1, details: ", err)
		return hashToken, NewError(err.Error(), http.StatusConflict)
	}

	// save to database, redis
	err = p.saveTokenToDB(hashToken, encryptedToken)
	if err != nil {
		logger.BkLog.Error("TokenProcessor::Process error 2, details: ", err)
		return hashToken, NewError(err.Error(), http.StatusInternalServerError)
	}

	err = p.saveTokenToCookie(hashToken)
	if err != nil {
		logger.BkLog.Error("TokenProcessor::Process error 3, details: ", err)
		return hashToken, NewError(err.Error(), http.StatusInternalServerError)
	}

	return
}

func (p *TokenProcessor) GenerateAccessToken(claims TokenClaims) (hashToken, encryptedToken string, err error) {
	rawToken, err := utils.CreateToken(claims)
	if err != nil {
		logger.BkLog.Errorf("Error when create %v, details: %v", p.TokenType, err)
		return
	}

	// encrypt token
	encryptedToken, err = utils.EncryptAccessToken(rawToken)
	if err != nil {
		logger.BkLog.Errorf("Error when encrypt %v, details: %v", p.TokenType, err)
		return
	}

	// hash token
	hashToken = utils.GetSHA256Hash(encryptedToken)

	if len(claims.CustomPrefix) > 0 {
		hashToken = fmt.Sprintf("%s:%s", claims.CustomPrefix, hashToken)
	}

	return
}

func (p *TokenProcessor) saveTokenToDB(hashedToken, encryptedToken string) (err error) {
	err = p.TokenManager.SetToken(context.Background(), p.TokenType, hashedToken, encryptedToken, p.TTL)
	if err != nil {
		logger.BkLog.Errorf("Error when save token %v to redis, details: %v", p.TokenType, err)
	}
	return
}

func (p *TokenProcessor) saveTokenToCookie(hashToken string) (err error) {
	if !p.TokenManager.NeedToSaveToCookie(p.TokenType) {
		return
	}

	if p.CookieClient == nil {
		return
	}

	err = p.CookieClient.SaveToken(p.TokenType, hashToken, p.ResponseWriter, p.Request)
	if err != nil {
		logger.BkLog.Errorf("Error when save %v to cookie, details: %v", p.TokenType, err)
	}

	return
}
