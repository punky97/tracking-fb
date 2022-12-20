package tokenmanager

import (
	"github.com/dgrijalva/jwt-go/request"
	"github.com/punky97/go-codebase/core/utils"
	"strings"
)

const (
	XAuthorizationHeader = "X-Access-Token"
	// XAuthorizationArgument -- argument for access token
	XAuthorizationArgument = "access_token"
)

// TokenClaims -- base on https://tools.ietf.org/html/rfc7519

// OAuth2Extractor -- Extractor for OAuth2 access tokens.
// Looks in 'Authorization' header then 'access_token' argument for a token.
var OAuth2Extractor = &request.MultiExtractor{
	request.HeaderExtractor{XAuthorizationHeader},
	request.ArgumentExtractor{XAuthorizationArgument},
}

func IsGrantedTokenType(requiredTokenType string, tokenType string) bool {
	return strings.EqualFold(tokenType, requiredTokenType)
}

func IsGrantedScope(requiredScope string, scope string) bool {
	if requiredScope == scope {
		return true
	} else if IsParentScope(scope, requiredScope) {
		return true
	}

	return false
}

var mappingScopes = map[string]*[]string{}

func AddParentScopes(parentScope string, scope ...string) {
	var scopes *[]string
	var ok bool
	if scopes, ok = mappingScopes[parentScope]; !ok {
		scopeList := make([]string, 0)
		mappingScopes[parentScope] = &scopeList
		scopes = &scopeList
	}

	*scopes = append(*scopes, scope...)
}

func IsParentScope(scope string, checkScope string) bool {
	// scope always in patterns: read_<> / write_<>
	// write scope always cover read scope
	if strings.HasPrefix(scope, "write_") && strings.HasPrefix(checkScope, "read_") &&
		scope[6:] == checkScope[5:] {
		return true
	}

	if scopes, ok := mappingScopes[scope]; ok {
		if utils.SliceContain(*scopes, checkScope) {
			return true
		}
	}

	return false
}
