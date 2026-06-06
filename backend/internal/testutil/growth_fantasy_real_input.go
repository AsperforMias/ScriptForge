package testutil

import (
	_ "embed"
	"regexp"
	"strings"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

//go:embed testdata/growth-fantasy-real-input.txt
var growthFantasyRealInput string

var growthFantasyChapterHeaderPattern = regexp.MustCompile(`(?m)^第[0-9一二三四五六七八九十]+章[^\r\n]*$`)

func GrowthFantasyRealInputRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "厄洛斯的转生见闻"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "异世界转生 / 贵族成长"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{
		"优先保证 deterministic 输出可信，而不是补足完整题材能力",
		"把叙述段落压成可拍场景初稿，避免长从句直接落字段",
	}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = growthFantasyRealInputChapters()
	return req
}

func GrowthFantasyRealInputExpectedNames() []string {
	return []string{"厄洛斯", "温蒂尼", "艾丝黛儿"}
}

func GrowthFantasyRealInputForbiddenFragments() []string {
	return []string{"脑子里", "三岁时", "因为听"}
}

func growthFantasyRealInputChapters() []job.ChapterBody {
	matches := growthFantasyChapterHeaderPattern.FindAllStringIndex(growthFantasyRealInput, -1)
	if len(matches) < 3 {
		panic("growth fantasy real input: expected at least 3 chapter headers")
	}

	chapters := make([]job.ChapterBody, 0, len(matches))
	for idx, match := range matches {
		title := strings.TrimSpace(growthFantasyRealInput[match[0]:match[1]])
		contentStart := match[1]
		contentEnd := len(growthFantasyRealInput)
		if idx+1 < len(matches) {
			contentEnd = matches[idx+1][0]
		}

		content := strings.TrimSpace(growthFantasyRealInput[contentStart:contentEnd])
		content = strings.TrimSpace(strings.Trim(content, "-\r\n "))
		chapters = append(chapters, job.ChapterBody{
			Index:   idx + 1,
			Title:   title,
			Content: content,
		})
	}
	return chapters
}
