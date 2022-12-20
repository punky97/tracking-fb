package tokenmanager

import "github.com/dgrijalva/jwt-go"

type TokenDefinition struct {
	TokenType    string `json:"token_type"`
	TokenPrefix  string `json:"token_prefix"`
	TTL          int64  `json:"ttl"`
	SaveToCookie bool   `json:"save_to_cookie"`
}

type TokenClaims struct {
	jwt.StandardClaims
	TokenType    string   `json:"token_type,omitempty"`
	UserID       int64    `json:"user_id,omitempty"`
	Username     string   `json:"username,omitempty"`
	Email        string   `json:"email,omitempty"`
	IsPrivate    bool     `json:"is_private,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Roles        []string `json:"roles"`
	CustomPrefix string   `json:"custom_prefix"` // will add after token type prefix
}

type AccessTokenRequest struct {
	TokenType    string   `json:"token_type,omitempty"`
	UserID       int64    `json:"user_id,omitempty"`
	Username     string   `json:"username,omitempty"`
	Email        string   `json:"email,omitempty"`
	IsPrivate    bool     `json:"is_private,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	TTL          *int64   `json:"ttl,omitempty"`
	CustomPrefix string   `json:"custom_prefix,omitempty"`
}
