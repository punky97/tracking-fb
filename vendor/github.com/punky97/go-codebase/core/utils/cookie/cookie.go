package cookie

import (
	"errors"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/gorilla/sessions"
	"net/http"
)

const (
	// AuthSessionName -- name for auth session
	AuthSessionName = "auth-cookie"
)

var ErrorSessionNotFound = errors.New("session not found")

// Client -- real implementation
type Client struct {
	store *sessions.CookieStore
}

// NewClient -- init cookie store
func NewClient(keyBytes []byte) *Client {
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	cStore := sessions.NewCookieStore(keyBytes)
	return &Client{cStore}
}

// SaveToken -- save token in cookies
func (cc *Client) SaveToken(tokenType string, hashedToken string, w http.ResponseWriter, r *http.Request) error {
	session, err := cc.store.Get(r, AuthSessionName)
	if err != nil {
		logger.BkLog.Error("Error while get session ", err)
		return err
	}
	session.Values[tokenType] = hashedToken
	return session.Save(r, w)
}

// GetToken -- get token
func (cc *Client) GetToken(tokenType string, r *http.Request) (string, error) {
	ss, err := cc.store.Get(r, AuthSessionName)
	if err != nil {
		logger.BkLog.Error("Error while get session ", err)
		return "", err
	}

	if _, ok := ss.Values[tokenType]; !ok {
		return "", ErrorSessionNotFound
	}

	return ss.Values[tokenType].(string), nil
}
