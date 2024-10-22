package config

import "strings"

func Split(str string) []string {
	s := []string{}

	s = append(s, strings.Split(str, ",")...)

	return s
}
