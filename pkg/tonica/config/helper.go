package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvStringSlice(key, fallback string) []string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.Split(",", value)
	}
	return strings.Split(",", fallback)
}

func GetEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		return convertStringToInt(value)
	}
	return fallback
}

func GetEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return convertStringToBool(value)
	}
	return fallback
}

func convertStringToBool(value string) bool {
	switch value {
	case "true":
		return true
	case "false":
		return false
	default:
		return false
	}
}

func convertStringToInt(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		slog.Error(err.Error())
		return 0
	}

	return result
}
