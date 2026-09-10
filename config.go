package main

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GroupCount  int
	VecSize     int
	MemoryLimit int
	Mode        Op
}

func loadConfig(path string) Config {
	values := readEnvFile(path)

	return Config{
		GroupCount:  envInt(values, "groupCount", 8192),
		VecSize:     envInt(values, "vecSize", 1024),
		MemoryLimit: envMemory(values, "memoryLimit", 4*1024*1024*1024),
		Mode:        envMode(values, "MODE", LOAD),
	}
}

func readEnvFile(path string) map[string]string {
	values := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.ToUpper(strings.TrimSpace(key))] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

func envValue(values map[string]string, key string) (string, bool) {
	if value := os.Getenv(key); value != "" {
		return value, true
	}
	value, ok := values[strings.ToUpper(key)]
	return value, ok
}

func envInt(values map[string]string, key string, def int) int {
	value, ok := envValue(values, key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return def
	}
	return n
}

func envMemory(values map[string]string, key string, def int) int {
	value, ok := envValue(values, key)
	if !ok {
		return def
	}
	n, err := parseMemory(value)
	if err != nil {
		return def
	}
	return n
}

func envMode(values map[string]string, key string, def Op) Op {
	value, ok := envValue(values, key)
	if !ok {
		return def
	}
	return parseMode(value)
}

func parseMode(s string) Op {
	if strings.EqualFold(strings.TrimSpace(s), "SERVE") {
		return SERVE
	}
	return LOAD
}

func parseMemory(s string) (int, error) {
	value := strings.ToLower(strings.TrimSpace(s))
	if value == "" {
		return 0, strconv.ErrSyntax
	}

	units := []struct {
		suffix string
		scale  float64
	}{
		{"kb", 1 << 10},
		{"mb", 1 << 20},
		{"gb", 1 << 30},
		{"tb", 1 << 40},
		{"k", 1 << 10},
		{"m", 1 << 20},
		{"g", 1 << 30},
		{"t", 1 << 40},
		{"b", 1},
	}

	scale := float64(1)
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			scale = unit.scale
			value = strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			break
		}
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number < 0 {
		return 0, strconv.ErrSyntax
	}
	return int(number * scale), nil
}
