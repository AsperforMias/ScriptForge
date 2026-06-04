package job

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestCreateRejectsLessThanThreeChapters(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := validCreateJobRequest()
	req.Source.Chapters = req.Source.Chapters[:2]

	_, err := service.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr AppError
	if !AsAppError(err, &appErr) {
		t.Fatal("expected app error")
	}
	if appErr.Code != "invalid_input" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestGetResultReturnsNotReadyForQueuedJob(t *testing.T) {
	service := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := validCreateJobRequest()

	created, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = service.GetResult(context.Background(), created.ID)
	if err == nil {
		t.Fatal("expected job not ready error")
	}

	var appErr AppError
	if !AsAppError(err, &appErr) {
		t.Fatal("expected app error")
	}
	if appErr.Code != "job_not_ready" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func validCreateJobRequest() CreateJobRequest {
	var req CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Adaptation.Style = "Suspense Drama"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "A"},
		{Index: 2, Title: "Chapter 2", Content: "B"},
		{Index: 3, Title: "Chapter 3", Content: "C"},
	}
	return req
}
