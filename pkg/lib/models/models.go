package models

import "strings"

func IsValidS3Directory(path string) bool {
	if strings.HasSuffix(path, ".zip") {
		return true
	}

	return false
}
