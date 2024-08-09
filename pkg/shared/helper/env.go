package helper

import (
	"os"
	"strconv"
)

func GetenvStr(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return os.Getenv(key)
}

func GetenvInt(key string) int {
	s := GetenvStr(key, "")
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func GetenvBool(key string) bool {
	s := GetenvStr(key, "true")
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return v
}
