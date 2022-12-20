package utils

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Base64ToFile -
func Base64ToFile(base64Encoded string) (file io.Reader, contentType string, err error) {
	strs := strings.Split(base64Encoded, ";base64,")
	if len(strs) < 2 {
		return nil, "", errors.New("invalid file data")
	}

	fileContent, err := base64.StdEncoding.DecodeString(strs[1])
	if err != nil {
		return nil, "", err
	}

	return bytes.NewReader(fileContent), http.DetectContentType(fileContent), nil
}
