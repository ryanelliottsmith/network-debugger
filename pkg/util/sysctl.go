package util

import (
	"os"
	"strings"
)

// ReadSysctl reads a sysctl value from /proc filesystem
func ReadSysctl(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
