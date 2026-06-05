package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir sqlite dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db}
	if err := store.configureConnection(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateJob(ctx context.Context, record job.Job, stages []job.Stage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create job tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO jobs (id, source_title, status, current_stage, progress_percent, generation_mode, warning_count, warnings_json, error_message, input_snapshot_path, result_yaml_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.SourceTitle, record.Status, record.CurrentStage, record.ProgressPercent, record.GenerationMode, len(record.Warnings), warningsJSON(record.Warnings), record.ErrorMessage, record.InputSnapshotPath, record.ResultYAMLPath, record.CreatedAt, record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	for _, stage := range stages {
		if err := upsertStage(ctx, tx, record.ID, stage); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create job tx: %w", err)
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, jobID string) (job.Job, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, source_title, status, current_stage, progress_percent, generation_mode, warning_count, warnings_json, error_message, input_snapshot_path, result_yaml_path, created_at, updated_at
		FROM jobs WHERE id = ?
	`, jobID)

	var record job.Job
	var warningCount int
	var warningsRaw string
	if err := row.Scan(&record.ID, &record.SourceTitle, &record.Status, &record.CurrentStage, &record.ProgressPercent, &record.GenerationMode, &warningCount, &warningsRaw, &record.ErrorMessage, &record.InputSnapshotPath, &record.ResultYAMLPath, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return job.Job{}, job.ErrJobNotFound
		}
		return job.Job{}, fmt.Errorf("select job: %w", err)
	}
	record.Warnings = decodeWarnings(warningsRaw, warningCount)

	return record, nil
}

func (s *Store) GetStages(ctx context.Context, jobID string) ([]job.Stage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT stage_name, status, warning_count, error_message, started_at, finished_at
		FROM job_stages WHERE job_id = ?
		ORDER BY rowid ASC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("select stages: %w", err)
	}
	defer rows.Close()

	stages := []job.Stage{}
	for rows.Next() {
		var stage job.Stage
		if err := rows.Scan(&stage.Name, &stage.Status, &stage.WarningCount, &stage.ErrorMessage, &stage.StartedAt, &stage.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		stages = append(stages, stage)
	}

	return stages, nil
}

func (s *Store) UpdateJob(ctx context.Context, record job.Job) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE jobs
		SET status = ?, current_stage = ?, progress_percent = ?, generation_mode = ?, warning_count = ?, warnings_json = ?, error_message = ?, input_snapshot_path = ?, result_yaml_path = ?, updated_at = ?
		WHERE id = ?
	`, record.Status, record.CurrentStage, record.ProgressPercent, record.GenerationMode, len(record.Warnings), warningsJSON(record.Warnings), record.ErrorMessage, record.InputSnapshotPath, record.ResultYAMLPath, record.UpdatedAt, record.ID)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return nil
}

func (s *Store) UpdateStages(ctx context.Context, jobID string, stages []job.Stage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update stages tx: %w", err)
	}
	defer tx.Rollback()

	for _, stage := range stages {
		if err := upsertStage(ctx, tx, jobID, stage); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update stages tx: %w", err)
	}

	return nil
}

func (s *Store) SaveArtifact(ctx context.Context, artifact job.Artifact) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO artifacts (job_id, yaml_path, yaml_size_bytes, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			yaml_path = excluded.yaml_path,
			yaml_size_bytes = excluded.yaml_size_bytes,
			created_at = excluded.created_at
	`, artifact.JobID, artifact.YAMLPath, artifact.YAMLSizeBytes, artifact.CreatedAt)
	if err != nil {
		return fmt.Errorf("save artifact: %w", err)
	}
	return nil
}

func (s *Store) GetArtifact(ctx context.Context, jobID string) (job.Artifact, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT job_id, yaml_path, yaml_size_bytes, created_at
		FROM artifacts WHERE job_id = ?
	`, jobID)

	var artifact job.Artifact
	if err := row.Scan(&artifact.JobID, &artifact.YAMLPath, &artifact.YAMLSizeBytes, &artifact.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return job.Artifact{}, job.ErrJobNotReady
		}
		return job.Artifact{}, fmt.Errorf("select artifact: %w", err)
	}
	return artifact, nil
}

func (s *Store) initSchema() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("init sqlite schema: %w", err)
	}
	return nil
}

func (s *Store) configureConnection() error {
	if _, err := s.db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return fmt.Errorf("set sqlite busy_timeout: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return fmt.Errorf("set sqlite journal_mode: %w", err)
	}
	return nil
}

func upsertStage(ctx context.Context, tx *sql.Tx, jobID string, stage job.Stage) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO job_stages (job_id, stage_name, status, warning_count, error_message, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id, stage_name) DO UPDATE SET
			status = excluded.status,
			warning_count = excluded.warning_count,
			error_message = excluded.error_message,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at
	`, jobID, stage.Name, stage.Status, stage.WarningCount, stage.ErrorMessage, stage.StartedAt, stage.FinishedAt)
	if err != nil {
		return fmt.Errorf("upsert stage %s: %w", stage.Name, err)
	}
	return nil
}

func warningsJSON(warnings []string) string {
	if warnings == nil {
		warnings = []string{}
	}

	payload, err := json.Marshal(warnings)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeWarnings(raw string, fallbackCount int) []string {
	if raw == "" {
		return make([]string, fallbackCount)
	}

	var warnings []string
	if err := json.Unmarshal([]byte(raw), &warnings); err != nil {
		return make([]string, fallbackCount)
	}
	if warnings == nil {
		return []string{}
	}
	return warnings
}
