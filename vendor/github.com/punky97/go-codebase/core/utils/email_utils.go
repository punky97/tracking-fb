package utils

import (
	"golang.org/x/text/unicode/norm"
	"regexp"
	"strings"
)

func IsEmail(email string) (bool) {
	return matchesRegex(emailRegex, email)
}

func NormalEmailAddress(email string) string {
	email = RemoveAccents(email)
	email = strings.ToLower(string(norm.NFKC.Bytes([]byte(email))))
	var re2 = regexp.MustCompile("[^/a-zA-Z0-9.!#$%&'*+/=?^_@{|}~-]")
	return strings.ToLower(re2.ReplaceAllString(email, ""))
}

func EmailValidate(email string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(email)
}