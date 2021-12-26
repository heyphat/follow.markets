package main

import (
	"strings"
)

func parseVars(vars map[string]string, param string) ([]string, bool) {
	var out []string
	str, ok := vars[param]
	if !ok {
		return out, ok
	}
	strs := strings.Split(str, ",")
	if len(strs) == 0 {
		return out, false
	}
	return strs, true
}
