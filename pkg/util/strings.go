package util

import "strings"

func StringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if strings.ToLower(str) == strings.ToLower(s) {
			return true
		}
	}
	return false
}
