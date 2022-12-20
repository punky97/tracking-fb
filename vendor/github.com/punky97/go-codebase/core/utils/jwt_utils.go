package utils

import (
	"github.com/punky97/go-codebase/core/encryption"
	"crypto/aes"
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

// keys are held in global variables
var (
	verifyKey  *rsa.PublicKey
	signKey    *rsa.PrivateKey
	encryptKey []byte
)

// InitJWT -- read the key files before starting http handlers
func InitJWT(signBytes []byte, verifyBytes []byte, encryptKeyBytes []byte) (err error) {

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		return err
	}

	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		return err
	}

	err = validateEncryptKey(encryptKeyBytes)
	if err != nil {
		return err
	}
	encryptKey = encryptKeyBytes

	return nil
}

func validateEncryptKey(key []byte) error {
	k := len(key)
	switch k {
	default:
		return aes.KeySizeError(k)
	case 16, 24, 32:
		break
	}
	return nil
}

// GetVerifyKey -- return verify key
func GetVerifyKey() *rsa.PublicKey {
	return verifyKey
}

// CreateToken -- create token with custom claims
func CreateToken(claims jwt.Claims) (token string, err error) {
	// create a signer for rsa 256
	j := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// generate the auth token string
	token, err = j.SignedString(signKey)
	return
}

// ParseToken -- parse token
func ParseToken(tokenString string, claimProto jwt.Claims) (authToken *jwt.Token, err error) {
	authToken, err = jwt.ParseWithClaims(tokenString, claimProto,
		func(token *jwt.Token) (interface{}, error) {
			// validate that the signig method is correct
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return GetVerifyKey(), nil
		})
	return
}

// EncryptAccessToken -- encrypt access token
func EncryptAccessToken(accessToken string) (string, error) {
	return encryption.AesEncrypt(encryptKey, accessToken)
}

// DecryptAccessToken -- decrypt encrypted access token
func DecryptAccessToken(encrypted string) (string, error) {
	return encryption.AesDecrypt(encryptKey, encrypted)
}
