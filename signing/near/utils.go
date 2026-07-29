package near

import "regexp"

var nearAccountIDRegex = regexp.MustCompile(`^(([a-z\d]+[-_])*[a-z\d]+\.)*([a-z\d]+[-_])*[a-z\d]+$`)

func ValidAddress(address string) bool {
	if len(address) < 2 || len(address) > 64 {
		return false
	}
	return nearAccountIDRegex.MatchString(address)
}
