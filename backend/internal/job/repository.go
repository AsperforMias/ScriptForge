package job

import "context"

type Repository interface {
	CreateJob(ctx context.Context, record Job, stages []Stage) error
	GetJob(ctx context.Context, jobID string) (Job, error)
	GetStages(ctx context.Context, jobID string) ([]Stage, error)
	UpdateJob(ctx context.Context, record Job) error
	UpdateStages(ctx context.Context, jobID string, stages []Stage) error
	SaveArtifact(ctx context.Context, artifact Artifact) error
	GetArtifact(ctx context.Context, jobID string) (Artifact, error)
}

type Runner interface {
	Run(ctx context.Context, jobID string, req CreateJobRequest) (ExecutionResult, error)
}
