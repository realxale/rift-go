package config

import (
	"bufio"
	"os"
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
