package utils

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	rand2 "math/rand"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString2 --
func GenerateRandomString2(s int) string {
	b, _ := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

/*
	GenerateRandomStringWithLetters - gen string with element belongs string defined
	params:
   - n: len of string want gen
   - letterStr:
  return:
   - string you want gen
*/
func GenerateRandomStringWithLetters(n int, lettersStr string) string {
	letters := []rune(lettersStr)
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand2.Intn(len(letters))]
	}
	return string(b)
}

// GenerateRandomStringWithoutPadding --
func GenerateRandomStringWithoutPadding(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), err
}

// StringToInt64 -- convert string to int64
func StringToInt64(input string) (output int64, err error) {
	if input == "" {
		output = 0
		err = nil
	} else {
		output, err = strconv.ParseInt(input, 0, 64)
	}
	return
}

// Int64ToString -- convert int64 to string
func Int64ToString(input int64) string {
	return strconv.FormatInt(input, 10)
}

// BoolToString -- convert bool to string
func BoolToString(input bool) string {
	return strconv.FormatBool(input)
}

// BytesToString -- convert bytes array to string
func BytesToString(data []byte) string {
	return string(data[:])
}

// GenerateCSRFSecret -- generate random CSRF secret
func GenerateCSRFSecret() (string, error) {
	return GenerateRandomString(32)
}

// IsStringSliceContains -- check slice contain string
func IsStringSliceContains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if value == searchString {
			return true
		}
	}
	return false
}

// IsStringSliceContains -- check slice contain string
func IsStringSliceCaseInsensitiveContains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if strings.EqualFold(value, searchString) {
			return true
		}
	}
	return false
}

func IsStringContainsAnyKeywords(s string, keywords []string) bool {
	contain := false
	for i := range keywords {
		if strings.Contains(s, keywords[i]) {
			contain = true
			break
		}
	}
	return contain
}

// RemoveStringSliceContains -- check slice contain string and remove
func RemoveStringSliceContains(stringSlice []string, searchString string) []string {
	newData := []string{}
	for _, value := range stringSlice {
		if value != searchString {
			newData = append(newData, value)
		}
	}
	return newData
}

// IsIntSliceContains -- check slice contain string
func IsIntSliceContains(intSlice []int64, searchInt int64) bool {
	for _, value := range intSlice {
		if value == searchInt {
			return true
		}
	}
	return false
}

// StringTrimSpace -- trim space of string
func StringTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

// IsStringEmpty -- check if string is empty
func IsStringEmpty(s string) bool {
	return s == ""
}

// IsStringNotEmpty -- check if string is not empty
func IsStringNotEmpty(s string) bool {
	return s != ""
}

// StringSlice -- slice string by seperate
func StringSlice(s, sep string) []string {
	var sl []string

	for _, p := range strings.Split(s, sep) {
		if str := StringTrimSpace(p); len(str) > 0 {
			sl = append(sl, StringTrimSpace(p))
		}
	}

	return sl
}

// JoinInt64Array -- join int64 array to string with separate
func JoinInt64Array(ns []int64, sep string) string {
	var nstr []string
	for _, n := range ns {
		nstr = append(nstr, Int64ToString(n))
	}
	return strings.Join(nstr, sep)
}

// JoinStrings -- join strings together
func JoinStrings(strs ...string) string {
	//?? need to be removed
	return strings.Join(strs, "")
}

func RemoveAccents(str string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	kq, _, _ := transform.String(t, str)
	return kq
}

var numberSequence = regexp.MustCompile(`([a-zA-Z])(\d+)([a-zA-Z]?)`)
var numberReplacement = []byte(`$1 $2 $3`)

func addWordBoundariesToNumbers(s string) string {
	b := []byte(s)
	b = numberSequence.ReplaceAll(b, numberReplacement)
	return string(b)
}

// Converts a string to snake_case
func ToSnake(s string) string {
	return ToDelimited(s, '_')
}

// Converts a string to SCREAMING_SNAKE_CASE
func ToScreamingSnake(s string) string {
	return ToScreamingDelimited(s, '_', true)
}

// Converts a string to kebab-case
func ToKebab(s string) string {
	return ToDelimited(s, '-')
}

// Converts a string to SCREAMING-KEBAB-CASE
func ToScreamingKebab(s string) string {
	return ToScreamingDelimited(s, '-', true)
}

// Converts a string to delimited.snake.case (in this case `del = '.'`)
func ToDelimited(s string, del uint8) string {
	return ToScreamingDelimited(s, del, false)
}

// Converts a string to SCREAMING.DELIMITED.SNAKE.CASE (in this case `del = '.'; screaming = true`) or delimited.snake.case (in this case `del = '.'; screaming = false`)
func ToScreamingDelimited(s string, del uint8, screaming bool) string {
	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	n := ""
	for i, v := range s {
		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		nextCaseIsChanged := false
		if i+1 < len(s) {
			next := s[i+1]
			if (v >= 'A' && v <= 'Z' && next >= 'a' && next <= 'z') || (v >= 'a' && v <= 'z' && next >= 'A' && next <= 'Z') {
				nextCaseIsChanged = true
			}
		}

		if i > 0 && n[len(n)-1] != del && nextCaseIsChanged {
			// add underscore if next letter case type is changed
			if v >= 'A' && v <= 'Z' {
				n += string(del) + string(v)
			} else if v >= 'a' && v <= 'z' {
				n += string(v) + string(del)
			}
		} else if v == ' ' || v == '_' || v == '-' {
			// replace spaces/underscores with delimiters
			n += string(del)
		} else {
			n = n + string(v)
		}
	}

	if screaming {
		n = strings.ToUpper(n)
	} else {
		n = strings.ToLower(n)
	}
	return n
}

// Upper case first character
func UcFirst(str string) string {
	r := []rune(str)
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

// Lower case first char
func LcFirst(str string) string {
	r := []rune(str)
	return string(unicode.ToLower(r[0])) + string(r[1:])
}

// Converts a string to CamelCase
func toCamelInitCase(s string, initCase bool) string {
	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	n := ""
	capNext := initCase
	for _, v := range s {
		if v >= 'A' && v <= 'Z' {
			n += string(v)
		}
		if v >= '0' && v <= '9' {
			n += string(v)
		}
		if v >= 'a' && v <= 'z' {
			if capNext {
				n += strings.ToUpper(string(v))
			} else {
				n += string(v)
			}
		}
		if v == '_' || v == ' ' || v == '-' {
			capNext = true
		} else {
			capNext = false
		}
	}
	return n
}

// Converts a string to CamelCase
func ToCamel(s string) string {
	return toCamelInitCase(s, true)
}

// Converts a string to lowerCamelCase
func ToLowerCamel(s string) string {
	if s == "" {
		return s
	}
	if r := rune(s[0]); r >= 'A' && r <= 'Z' {
		s = strings.ToLower(string(r)) + s[1:]
	}
	return toCamelInitCase(s, false)
}


// Converts a string to CamelCase
func ToCamelInitCaseKeepAll(s string, initCase bool) string {
	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	n := ""
	capNext := initCase
	for _, v := range s {
		if v >= 'A' && v <= 'Z' {
			n += string(v)
		}
		if v >= '0' && v <= '9' {
			n += string(v)
		}

		if v == '_' || v == ' ' || v == '-' {
			capNext = true
		} else {
			if capNext {
				n += strings.ToUpper(string(v))
			} else {
				n += string(v)
			}
			capNext = false
		}
	}
	return n
}

func GetStringBetween(value string, a string, b string) string {
	// Get substring between two strings.
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	newValue := value[posFirstAdjusted:]
	posLast := strings.Index(newValue, b)
	if posLast == -1 {
		return ""
	}
	posLastAdjusted := posFirstAdjusted + posLast
	return value[posFirstAdjusted:posLastAdjusted]
}