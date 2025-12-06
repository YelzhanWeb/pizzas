package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	parsed, err := parseYAML(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	jsonData, err := json.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func parseYAML(data string) (map[string]any, error) {
	lines := strings.Split(data, "\n")
	result := make(map[string]any)

	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Секция: "database:"
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			currentSection = strings.TrimSuffix(line, ":")
			result[currentSection] = make(map[string]any)
			continue
		}

		// Внутренний ключ: "host: localhost"
		if currentSection != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid yaml line: %s", line)
			}

			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Попытка конвертации в число
			if n, err := strconv.Atoi(val); err == nil {
				result[currentSection].(map[string]any)[key] = n
			} else {
				result[currentSection].(map[string]any)[key] = val
			}

			continue
		}

		return nil, fmt.Errorf("unexpected yaml line: %s", line)
	}

	return result, nil
}
