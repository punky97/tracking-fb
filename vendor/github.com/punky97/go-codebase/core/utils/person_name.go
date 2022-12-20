package utils

import "strings"

type PersonName struct {
	FirstName string
	LastName  string
}

func ParsePersonName(fullName string) (personName *PersonName) {
	personName = &PersonName{}
	strs := make([]string, 2)
	strs = strings.Split(fullName, " ")

	personName.FirstName = strs[0]

	if len(strs) < 2 {
		personName.LastName = strs[0]
	} else {
		personName.LastName = strs[1]
	}

	return
}
