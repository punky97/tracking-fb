package utils

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// MaxLength max allowed password length
const MaxLength = 4096

// DefaultLoopTimes default loop times
const DefaultLoopTimes = 5000

// GeneratePasswordToStoreDB -- generate hashed password to store in database
func GeneratePasswordToStoreDB(
	rawPassword string, genSalt bool, currentSalt string) (
	salt, hashedPassword string, err error) {

	// if has request generate salt, generate new salt.
	if genSalt {
		var randStr string
		randStr, err = GenerateRandomString(32)
		if err != nil {
			return
		}
		salt = GetMD5Hash(randStr)
	} else {
		salt = currentSalt
	}

	hashedPassword, err = CreatePassword(rawPassword, salt, true)
	if err != nil {
		return
	}

	return
}

// ComparePasswords is to compare 2 passwords
// Generally, it calls create password again to create password from input and salt
// then compare the 2 values to see if they equal or not.
func ComparePasswords(inputPassword, salt, hashedPassword string, isbase64Encoded bool) bool {
	if reachLengthLimit(inputPassword) {
		return false
	}

	hashedInputPwd := create(inputPassword, salt)
	encodedHash := ""
	if isbase64Encoded {
		encodedHash = base64.StdEncoding.EncodeToString([]byte(hashedInputPwd))
	} else {
		encodedHash = hex.EncodeToString([]byte(hashedInputPwd))
	}

	if strings.Compare(encodedHash, hashedPassword) == 0 {
		return true
	}

	return false
}

// CreatePassword with salt, compatible with symfony 2 password generator
// Return:
//        - base64 encoded string if isbase64Encoded is set
//        - hex represented string otherwise
func CreatePassword(password, salt string, isbase64Encoded bool) (string, error) {
	if reachLengthLimit(password) {
		return "", fmt.Errorf("Password is too long")
	}

	result := create(password, salt)
	if isbase64Encoded {
		return base64.StdEncoding.EncodeToString([]byte(result)), nil
	}

	return hex.EncodeToString([]byte(result)), nil
}

func create(password, salt string) (result string) {
	runner := createRunner(password, salt)
	result = loop(DefaultLoopTimes, runner)
	return
}

func loop(n int, iterator func() string) (result string) {
	if n <= 0 {
		return ""
	}

	for i := 0; i < n; i++ {
		result = iterator()
	}
	return
}

func reachLengthLimit(password string) bool {
	return len(password) > MaxLength
}

func createRunner(password, salt string) func() string {
	digest := ""
	saltedPwd, _ := merge(password, salt)

	return func() string {
		salted := digest + saltedPwd
		shaHash := sha512.New()
		shaHash.Write([]byte(salted))
		digest = string(shaHash.Sum(nil))
		return digest
	}
}

func merge(password, salt string) (string, error) {
	if len(salt) == 0 {
		return password, nil
	}

	if strings.Contains(salt, "}") || strings.Contains(salt, "{") {
		return "", fmt.Errorf("Cannot contain { or } in salt")
	}
	return fmt.Sprintf("%s{%s}", password, salt), nil
}
