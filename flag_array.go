package main

import (
	"strings"
)

type stringArray []string

func (s stringArray) String() string {
	return strings.Join(s, ", ")
}

func (s *stringArray) Set(v string) error {
	*s = append(*s, v)
	return nil
}
