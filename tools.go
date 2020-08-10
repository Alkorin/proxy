package main

import (
	"strconv"
)

// ParseUint16 call strconv.ParseUint and return an uint16
func ParseUint16(s string) (uint16, error) {
	v, err := strconv.ParseUint(s, 10, 16)
	return uint16(v), err
}
