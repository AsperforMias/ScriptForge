package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

type Store struct {
	baseDir string
}

func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

func (s *Store) WriteInputSnapshot(jobID string, req job.CreateJobRequest) (string, error) {
	path := filepath.Join(s.baseDir, jobID, "input.json")
	if err := writeJSON(path, req); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) WriteNormalizedSource(jobID string, source ingest.NormalizedSource) (string, error) {
	path := filepath.Join(s.baseDir, jobID, "normalized_source.json")
	if err := writeJSON(path, source); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) WriteYAML(jobID, content string) (string, error) {
	path := filepath.Join(s.baseDir, jobID, "screenplay.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("mkdir artifact dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write yaml artifact: %w", err)
	}
	return path, nil
}

func (s *Store) WriteProviderDebug(jobID string, payload any) (string, error) {
	path := filepath.Join(s.baseDir, jobID, "provider_debug.json")
	if err := writeJSON(path, payload); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ReadYAML(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read yaml artifact: %w", err)
	}
	return string(data), nil
}

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir artifact dir: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal artifact json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write artifact json: %w", err)
	}
	return nil
}
