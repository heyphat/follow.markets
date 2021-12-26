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
	out := strings.Split(str, ",")
	if len(out) == 0 {
		return out, false
	}
	return out, true
}
