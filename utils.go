package caches

func ContainString(target string, slice []string) bool {
	for _, s := range slice {
		if target == s {
			return true
		}
	}
	return false
}
