package utils

func CensorString(str string) string {
	if len(str) <= 6 {
		return "***"
	}

	return str[:2] + "***" + str[len(str)-2:]
}
