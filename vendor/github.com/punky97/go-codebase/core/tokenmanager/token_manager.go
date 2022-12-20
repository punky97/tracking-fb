package tokenmanager

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/logger"
	"time"
)

func NewTokenManager(rd *bkredis.RedisClient, tokenDefinition []*TokenDefinition) *TokenManager {
	m := &TokenManager{
		rd:  rd,
		def: map[string]*TokenDefinition{},
	}

	for i, d := range tokenDefinition {
		m.def[d.TokenType] = tokenDefinition[i]
	}

	return m
}

// TokenManager --
type TokenManager struct {
	rd  *bkredis.RedisClient
	def map[string]*TokenDefinition
}

// GetRedisClient -- get redis client
func (m *TokenManager) GetRedisClient() redis.UniversalClient {
	return m.rd.UniversalClient
}

// SetToken --
func (m *TokenManager) SetToken(ctx context.Context, tokenType string, hashedToken, encryptedToken string, ttl *time.Duration) (err error) {
	accessToken := m.GetRedisTokenKey(tokenType, hashedToken)

	expiredIn := m.GetExpirationByTokenType(tokenType)
	if ttl != nil {
		expiredIn = *ttl
	}

	res := m.rd.Set(ctx, accessToken,
		encryptedToken,
		expiredIn)

	if res.Err() != nil && res.Err() != redis.Nil {
		logger.BkLog.Warn("Error when save token to redis: ", err)
		return err
	}

	return nil
}

// GetToken --
func (m *TokenManager) GetToken(ctx context.Context, tokenType string, hashedToken string) (string, error) {
	accessToken := m.GetRedisTokenKey(tokenType, hashedToken)
	res := m.rd.Get(ctx, accessToken)

	if res.Err() == nil {
		return res.Val(), nil
	} else {
		if res.Err().Error() != redis.Nil.Error() {
			logger.BkLog.Warnf("Could not get data from redis due to %v", res.Err())
			return "", res.Err()
		}
		return "", res.Err()
	}
}

// RemoveToken --
func (m *TokenManager) RemoveToken(ctx context.Context, tokenType string, hashedToken string) (err error) {
	accessToken := m.GetRedisTokenKey(tokenType, hashedToken)

	res := m.rd.Del(ctx, accessToken)
	if res.Err() == redis.Nil || res.Err() == nil {
	} else {
		logger.BkLog.Errorf("Error when remove token %v in redis: %v", hashedToken, res.Err())
		return res.Err()
	}

	return nil
}

func (m *TokenManager) GetRedisTokenKey(tokenType string, hashedToken string) string {
	tokenDef := m.GetTokenDefinitionByTokenType(tokenType)
	return fmt.Sprintf("%s:%s", tokenDef.TokenPrefix, hashedToken)

}

func (m *TokenManager) GetTokenDefinitionByTokenType(tokenType string) *TokenDefinition {
	detail, ok := m.def[tokenType]
	if !ok {
		return &TokenDefinition{}
	}

	return detail
}

func (m *TokenManager) GetExpirationByTokenType(tokenType string) time.Duration {
	detail, ok := m.def[tokenType]
	if !ok {
		return 1 * time.Hour // default
	}

	return time.Duration(detail.TTL) * time.Second
}
func (m *TokenManager) IsValidTokenType(tokenType string) bool {
	_, ok := m.def[tokenType]
	return ok
}

func (m *TokenManager) NeedToSaveToCookie(tokenType string) bool {
	detail, ok := m.def[tokenType]
	if ok {
		return detail.SaveToCookie
	}
	return false
}

func (m *TokenManager) NewAccessTokenClaims(req *AccessTokenRequest) TokenClaims {
	detail := m.GetTokenDefinitionByTokenType(req.TokenType)

	ttl := time.Duration(detail.TTL) * time.Second
	if req.TTL != nil {
		ttl = time.Duration(*req.TTL) * time.Second
	}

	exp := time.Now().Add(ttl).Unix()
	iat := time.Now().Unix()

	return TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
			IssuedAt:  iat,
		},
		TokenType:    req.TokenType,
		UserID:       req.UserID,
		Username:     req.Username,
		Email:        req.Email,
		IsPrivate:    req.IsPrivate,
		Scopes:       req.Scopes,
		Roles:        req.Roles,
		CustomPrefix: req.CustomPrefix,
	}
}
