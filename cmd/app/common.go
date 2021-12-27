package main

import (
	"strings"
)

func parseVars(vars map[string]string, param string) ([]string, bool) {
	str, ok := vars[param]
	if !ok {
		return nil, ok
	}
	if out := strings.Split(str, ","); len(out) != 0 {
		return out, true
	}
	return nil, false
}
