package ingest

import (
	"strings"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

type NormalizedSource struct {
	Title    string              `json:"title"`
	Author   string              `json:"author,omitempty"`
	Language string              `json:"language"`
	Chapters []NormalizedChapter `json:"chapters"`
}

type NormalizedChapter struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func Normalize(req job.CreateJobRequest) NormalizedSource {
	chapters := make([]NormalizedChapter, 0, len(req.Source.Chapters))
	for _, chapter := range req.Source.Chapters {
		chapters = append(chapters, NormalizedChapter{
			Index:   chapter.Index,
			Title:   strings.TrimSpace(chapter.Title),
			Content: normalizeWhitespace(chapter.Content),
		})
	}

	return NormalizedSource{
		Title:    strings.TrimSpace(req.Source.Title),
		Author:   strings.TrimSpace(req.Source.Author),
		Language: "zh-CN",
		Chapters: chapters,
	}
}

func normalizeWhitespace(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}
