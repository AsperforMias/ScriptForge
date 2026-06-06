package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsRepoRootDotEnvLocalFromBackendSubdir(t *testing.T) {
	rootDir := t.TempDir()
	backendDir := filepath.Join(rootDir, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir backend dir: %v", err)
	}

	envFile := filepath.Join(rootDir, ".env.local")
	envText := "LLM_PROVIDER=openai_compatible\nLLM_MODEL=deepseek-v4-flash\nLLM_API_KEY=test-key\n"
	if err := os.WriteFile(envFile, []byte(envText), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(backendDir); err != nil {
		t.Fatalf("chdir backend dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	unsetEnvForTest(t, "LLM_PROVIDER")
	unsetEnvForTest(t, "LLM_MODEL")
	unsetEnvForTest(t, "LLM_API_KEY")

	cfg := Load()

	if cfg.LLMProvider != "openai_compatible" {
		t.Fatalf("expected LLM provider from .env.local, got %s", cfg.LLMProvider)
	}
	if cfg.LLMModel != "deepseek-v4-flash" {
		t.Fatalf("expected LLM model from .env.local, got %s", cfg.LLMModel)
	}
	if cfg.LLMAPIKey != "test-key" {
		t.Fatalf("expected api key from .env.local, got %s", cfg.LLMAPIKey)
	}
}

func TestLoadDoesNotOverrideExistingEnvironment(t *testing.T) {
	rootDir := t.TempDir()
	envFile := filepath.Join(rootDir, ".env.local")
	envText := "LLM_PROVIDER=openai_compatible\n"
	if err := os.WriteFile(envFile, []byte(envText), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir root dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	t.Setenv("LLM_PROVIDER", "mock")

	cfg := Load()

	if cfg.LLMProvider != "mock" {
		t.Fatalf("expected existing env var to win, got %s", cfg.LLMProvider)
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	previousValue, hadValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}

	t.Cleanup(func() {
		if !hadValue {
			_ = os.Unsetenv(key)
			return
		}
		_ = os.Setenv(key, previousValue)
	})
}
