package util

func Int64SliceContains(slice []int64, i int64) bool {
	for _, j := range slice {
		if i == j {
			return true
		}
	}
	return false
}
