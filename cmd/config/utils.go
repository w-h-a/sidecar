package config

import "strings"

func Split(commaString string) []string {
	s := []string{}

	s = append(s, strings.Split(commaString, ",")...)

	return s
}
