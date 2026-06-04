package job

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
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	CurrentStage    string   `json:"current_stage"`
	ProgressPercent int      `json:"progress_percent"`
	SourceTitle     string   `json:"source_title,omitempty"`
	GenerationMode  string   `json:"generation_mode,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	ErrorMessage    string   `json:"error_message,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type Stage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Details struct {
	Job    Job
	Stages []Stage
}
