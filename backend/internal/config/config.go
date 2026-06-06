package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                string
	HTTPAddr              string
	SQLitePath            string
	ArtifactDir           string
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	RequestBodyLimitBytes int64
	JobMaxConcurrency     int
	CORSAllowOrigin       string
	GenerationModeDefault string
	LLMProvider           string
	LLMModel              string
	LLMBaseURL            string
	LLMAPIKey             string
	LLMRequestTimeout     time.Duration
}

func Load() Config {
	loadLocalEnvFile(".env.local")

	return Config{
		AppEnv:                envString("APP_ENV", "development"),
		HTTPAddr:              envString("HTTP_ADDR", ":8080"),
		SQLitePath:            envString("SQLITE_PATH", "./tmp/scriptforge.db"),
		ArtifactDir:           envString("ARTIFACT_DIR", "./tmp/artifacts"),
		HTTPReadTimeout:       envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:      envDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		RequestBodyLimitBytes: envInt64("REQUEST_BODY_LIMIT_BYTES", 4*1024*1024),
		JobMaxConcurrency:     envInt("JOB_MAX_CONCURRENCY", 2),
		CORSAllowOrigin:       envString("CORS_ALLOW_ORIGIN", "*"),
		GenerationModeDefault: envString("GENERATION_MODE_DEFAULT", "llm"),
		LLMProvider:           envString("LLM_PROVIDER", "disabled"),
		LLMModel:              envString("LLM_MODEL", ""),
		LLMBaseURL:            envString("LLM_BASE_URL", ""),
		LLMAPIKey:             envString("LLM_API_KEY", ""),
		LLMRequestTimeout:     envDuration("LLM_REQUEST_TIMEOUT", 45*time.Second),
	}
}

func loadLocalEnvFile(filename string) {
	path, ok := findEnvFile(filename)
	if !ok {
		return
	}

	file, err := os.Open(path)
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
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value := normalizeEnvValue(rawValue)
		_ = os.Setenv(key, value)
	}
}

func findEnvFile(filename string) (string, bool) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		candidate := filepath.Join(currentDir, filename)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", false
		}
		currentDir = parentDir
	}
}

func normalizeEnvValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}

	if commentIdx := strings.Index(value, " #"); commentIdx >= 0 {
		value = strings.TrimSpace(value[:commentIdx])
	}

	return value
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envInt64(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
