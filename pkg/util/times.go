package util

import "time"

func DurationSliceContains(slice []time.Duration, d time.Duration) bool {
	for _, j := range slice {
		if d == j {
			return true
		}
	}
	return false
}
