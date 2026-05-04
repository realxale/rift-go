package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

var loadEnvOnce sync.Once

func GetEnv(key, fallback string) string {
	loadEnvOnce.Do(loadEnvFile)

	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func GetEnvInt(key string, fallback int) int {
	val := GetEnv(key, "")
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}

		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}