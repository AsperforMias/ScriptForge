package job

import "github.com/AsperforMias/ScriptForge/backend/internal/screenplay"

type CreateJobRequest struct {
	Source struct {
		Title    string        `json:"title"`
		Author   string        `json:"author"`
		Chapters []ChapterBody `json:"chapters"`
	} `json:"source"`
	Adaptation struct {
		Style    string   `json:"style"`
		Audience string   `json:"audience"`
		Notes    []string `json:"notes"`
	} `json:"adaptation"`
	Generation struct {
		Mode string `json:"mode"`
	} `json:"generation"`
}

type ChapterBody struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Job struct {
	ID                string   `json:"id"`
	Status            string   `json:"status"`
	CurrentStage      string   `json:"current_stage"`
	ProgressPercent   int      `json:"progress_percent"`
	SourceTitle       string   `json:"source_title,omitempty"`
	GenerationMode    string   `json:"generation_mode,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	ErrorMessage      string   `json:"error_message,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	InputSnapshotPath string   `json:"-"`
	ResultYAMLPath    string   `json:"-"`
}

type Stage struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	WarningCount int    `json:"warning_count,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

type Details struct {
	Job    Job
	Stages []Stage
}

type Artifact struct {
	JobID         string
	YAMLPath      string
	YAMLSizeBytes int
	CreatedAt     string
}

type ResultPayload struct {
	JobID      string              `json:"job_id"`
	Screenplay screenplay.Document `json:"screenplay"`
	YAMLText   string              `json:"yaml_text"`
}

type ExecutionResult struct {
	Document             screenplay.Document
	YAMLText             string
	InputSnapshotPath    string
	NormalizedSourcePath string
	ProviderDebugPath    string
	YAMLPath             string
	Warnings             []string
	Stages               []Stage
	CurrentStage         string
}
