package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
)

type OutlineChapter struct {
	Index    int
	Title    string
	Summary  string
	Conflict string
}

type OutlineBundle struct {
	Chapters []OutlineChapter
}

type EntityBundle struct {
	Characters []screenplay.Character
	Locations  []screenplay.Location
}

type ScenePlan struct {
	Scenes []screenplay.Scene
}

func BuildOutline(source ingest.NormalizedSource) OutlineBundle {
	chapters := make([]OutlineChapter, 0, len(source.Chapters))
	for _, chapter := range source.Chapters {
		summary := summarize(chapter.Content)
		chapters = append(chapters, OutlineChapter{
			Index:    chapter.Index,
			Title:    chapter.Title,
			Summary:  summary,
			Conflict: buildConflict(summary),
		})
	}

	return OutlineBundle{Chapters: chapters}
}

func ExtractEntities(source ingest.NormalizedSource) EntityBundle {
	mainCharacterName := inferCharacterName(source)
	characters := []screenplay.Character{
		{
			ID:          "char_" + slugify(mainCharacterName),
			Name:        mainCharacterName,
			Role:        "protagonist",
			Description: "Primary viewpoint character inferred from the source input.",
		},
	}

	locations := make([]screenplay.Location, 0, len(source.Chapters))
	for _, chapter := range source.Chapters {
		locationName := inferLocationName(chapter)
		locations = append(locations, screenplay.Location{
			ID:          fmt.Sprintf("loc_chapter_%02d", chapter.Index),
			Name:        locationName,
			Description: fmt.Sprintf("Primary dramatic location inferred from chapter %d.", chapter.Index),
		})
	}

	return EntityBundle{
		Characters: characters,
		Locations:  locations,
	}
}

func BuildScenePlan(source ingest.NormalizedSource, outline OutlineBundle, entities EntityBundle) ScenePlan {
	characterID := entities.Characters[0].ID
	scenes := make([]screenplay.Scene, 0, len(source.Chapters))

	for idx, chapter := range source.Chapters {
		chapterOutline := outline.Chapters[idx]
		locationID := fmt.Sprintf("loc_chapter_%02d", chapter.Index)
		scene := screenplay.Scene{
			ID:             fmt.Sprintf("scene_%03d", idx+1),
			Title:          chapter.Title,
			SourceChapters: []int{chapter.Index},
			Slugline: screenplay.Slugline{
				InteriorExterior: inferInteriorExterior(chapter.Content),
				LocationID:       locationID,
				Time:             inferTime(chapter.Content),
			},
			Summary:   chapterOutline.Summary,
			Objective: fmt.Sprintf("Adapt chapter %d into a filmable dramatic beat.", chapter.Index),
			Beats: []screenplay.Beat{
				{
					Type:    "action",
					Content: chapterOutline.Summary,
				},
				{
					Type:        "dialogue",
					CharacterID: characterID,
					Content:     buildDialogue(chapterOutline.Summary),
					Emotion:     "focused",
				},
			},
			Notes: screenplay.SceneNotes{
				AdaptationReason: "Convert chapter-level prose into one traceable scene with clear visual action.",
				OpenQuestions:    []string{},
			},
		}
		scenes = append(scenes, scene)
	}

	return ScenePlan{Scenes: scenes}
}

func BuildDocument(req job.CreateJobRequest, source ingest.NormalizedSource, outline OutlineBundle, entities EntityBundle, plan ScenePlan) screenplay.Document {
	sourceChapters := make([]screenplay.SourceChapter, 0, len(outline.Chapters))
	for _, chapter := range outline.Chapters {
		sourceChapters = append(sourceChapters, screenplay.SourceChapter{
			Index:   chapter.Index,
			Title:   chapter.Title,
			Summary: chapter.Summary,
		})
	}

	return screenplay.Document{
		Version: "1.0",
		Source: screenplay.Source{
			Title:        source.Title,
			Author:       source.Author,
			Language:     source.Language,
			ChapterCount: len(source.Chapters),
			Chapters:     sourceChapters,
		},
		Adaptation: screenplay.Adaptation{
			Style:    req.Adaptation.Style,
			Audience: req.Adaptation.Audience,
			Notes:    req.Adaptation.Notes,
		},
		Characters: entities.Characters,
		Locations:  entities.Locations,
		Scenes:     plan.Scenes,
		Validation: screenplay.Validation{
			Status:   "passed",
			Warnings: []string{},
		},
	}
}

func summarize(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "This chapter introduces a new dramatic development."
	}
	if len([]rune(content)) <= 80 {
		return content
	}
	return string([]rune(content)[:80]) + "..."
}

func buildConflict(summary string) string {
	return "The chapter introduces pressure that pushes the protagonist into the next dramatic decision."
}

func buildDialogue(summary string) string {
	return "Something here does not add up."
}

func inferCharacterName(source ingest.NormalizedSource) string {
	candidates := extractCJKPhrases(strings.Join(chapterContents(source.Chapters), " "))
	if len(candidates) > 0 {
		return candidates[0]
	}
	return "主角"
}

func inferLocationName(chapter ingest.NormalizedChapter) string {
	keywords := []string{"公寓", "房间", "走廊", "办公室", "街道", "学校", "咖啡馆", "医院", "仓库", "车站"}
	for _, keyword := range keywords {
		if strings.Contains(chapter.Content, keyword) {
			return keyword
		}
	}
	return fmt.Sprintf("Chapter %d Main Location", chapter.Index)
}

func inferInteriorExterior(content string) string {
	if strings.Contains(content, "街") || strings.Contains(content, "路") || strings.Contains(content, "广场") {
		return "EXT"
	}
	return "INT"
}

func inferTime(content string) string {
	switch {
	case strings.Contains(content, "夜"), strings.Contains(content, "凌晨"):
		return "NIGHT"
	case strings.Contains(content, "早"), strings.Contains(content, "清晨"):
		return "MORNING"
	default:
		return "DAY"
	}
}

func extractCJKPhrases(input string) []string {
	re := regexp.MustCompile(`[\p{Han}]{2,4}`)
	seen := map[string]struct{}{}
	results := []string{}
	for _, candidate := range re.FindAllString(input, -1) {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		results = append(results, candidate)
		if len(results) == 3 {
			break
		}
	}
	return results
}

func chapterContents(chapters []ingest.NormalizedChapter) []string {
	results := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		results = append(results, chapter.Content)
	}
	return results
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return "protagonist"
	}
	input = strings.ReplaceAll(input, " ", "_")
	return input
}
