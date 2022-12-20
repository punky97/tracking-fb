package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//URLDecode - get origin text of url(`%7B%22email_tracking_url%22%3A%7B%22email_id%22%3A%222157082318%22%2C%22campaign_id%22%3A%22945080%22%7D%7D` -> `{"email_tracking_url":{"email_id":"2157082318","campaign_id":"945080"}}`)
func URLDecode(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return u.Path
}

// IsDomain -- check if domain is valid
func IsDomain(value string) bool {
	if len(value) > 255 {
		return false
	}

	// Domain regex source: https://stackoverflow.com/a/7933253
	// Slightly modified: Removed 255 max length validation since Go regex does not
	// support lookarounds. More info: https://stackoverflow.com/a/38935027
	reDomain := regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+(?:[a-z]{1,63}| xn--[a-z0-9]{1,59})$`)

	return reDomain.MatchString(value)
}

func BuilQuery(params map[string]string) string {
	query := ""
	if len(params) == 0 {
		return query
	}
	for key, value := range params {
		query += key + "=" + value + "&"
	}
	query = strings.TrimSuffix(query, "&")

	return query
}

func BuilQueryAbs(params map[string]interface{}) string {
	query := ""
	if len(params) == 0 {
		return query
	}
	for key, value := range params {
		query += fmt.Sprintf("%v=%v&", key, value)
	}
	query = strings.TrimSuffix(query, "&")

	return query
}

var SKIP = []*unicode.RangeTable{
	unicode.Mark,
	unicode.Sk,
	unicode.Lm,
}

var SAFE = []*unicode.RangeTable{
	unicode.Letter,
	unicode.Number,
}

// Slugify a string. The result will only contain lowercase letters,
// digits and dashes. It will not begin or end with a dash, and it
// will not contain runs of multiple dashes.
//
// It is NOT forced into being ASCII, but may contain any Unicode
// characters, with the above restrictions.
func Slugify(text string) string {
	buf := make([]rune, 0, len(text))
	dash := false
	for _, r := range norm.NFKD.String(text) {
		switch {
		case unicode.IsOneOf(SAFE, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case unicode.IsOneOf(SKIP, r):
		case dash:
			buf = append(buf, '-')
			dash = false
		}
	}
	if i := len(buf) - 1; i >= 0 && buf[i] == '-' {
		buf = buf[:i]
	}
	return string(buf)
}

// Slugifyf slugfy a formated string
func Slugifyf(format string, a ...interface{}) string {
	return Slugify(fmt.Sprintf(format, a...))
}

func RemoveQueryString(url string) string {
	s := StringSlice(url, "?")
	if len(s) > 1 {
		url = s[0]
	}
	return url
}
