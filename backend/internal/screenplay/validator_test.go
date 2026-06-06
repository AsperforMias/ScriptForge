package screenplay

import (
	"strings"
	"testing"
)

func TestValidateAndSerialize(t *testing.T) {
	doc := validDocument()

	validated, err := ValidateAndSerialize(doc)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if !strings.Contains(validated.YAMLText, "version: \"1.0\"") {
		t.Fatalf("expected yaml output to contain version, got %q", validated.YAMLText)
	}
	if validated.Document.Validation.Status != "passed" {
		t.Fatalf("expected validation status passed, got %s", validated.Document.Validation.Status)
	}
}

func TestValidateAndSerializePreservesFailedStatus(t *testing.T) {
	doc := validDocument()
	doc.Validation.Status = "failed"
	doc.Validation.Warnings = []string{"scene_001: objective is still derived from long narrative phrasing"}

	validated, err := ValidateAndSerialize(doc)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if validated.Document.Validation.Status != "failed" {
		t.Fatalf("expected validation status failed to be preserved, got %s", validated.Document.Validation.Status)
	}
}

func TestValidateRejectsInvalidSceneLocation(t *testing.T) {
	doc := validDocument()
	doc.Scenes[0].Slugline.LocationID = "missing"

	err := Validate(doc)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsDialogueWithoutCharacterReference(t *testing.T) {
	doc := validDocument()
	doc.Scenes[0].Beats = append(doc.Scenes[0].Beats, Beat{
		Type:    "dialogue",
		Content: "Who is there?",
	})

	err := Validate(doc)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsInvalidEvidenceChapterReference(t *testing.T) {
	doc := validDocument()
	doc.Scenes[0].Evidence = &Evidence{
		ChapterIndexes: []int{99},
		Excerpt:        "Suspicious lock detail.",
	}

	err := Validate(doc)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAndSerializeDowngradesDuplicateTemplateScenes(t *testing.T) {
	doc := validDocument()
	doc.Scenes = append(doc.Scenes, Scene{
		ID:             "scene_002",
		Title:          "Return Home Again",
		SourceChapters: []int{2},
		Slugline: Slugline{
			InteriorExterior: "INT",
			LocationID:       "loc_old_apartment",
			Time:             "NIGHT",
		},
		Summary:   "The same scene content repeats.",
		Objective: doc.Scenes[0].Objective,
		Beats: []Beat{
			{
				Type:    "action",
				Content: "Lin Qi freezes in front of the apartment door.",
			},
			{
				Type:    "action",
				Content: "Lin Qi freezes in front of the apartment door.",
			},
			{
				Type:        "dialogue",
				CharacterID: "char_lin_qi",
				Content:     "I know I locked this this morning.",
			},
			{
				Type:        "dialogue",
				CharacterID: "char_lin_qi",
				Content:     "I know I locked this this morning.",
			},
		},
		Notes: SceneNotes{
			OpenQuestions: []string{"What happens next?"},
		},
	})
	doc.Validation.Status = "passed"

	validated, err := ValidateAndSerialize(doc)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if validated.Document.Validation.Status != "failed" {
		t.Fatalf("expected validation status failed after quality audit, got %s", validated.Document.Validation.Status)
	}
	if len(validated.Document.Validation.Warnings) == 0 {
		t.Fatal("expected quality warnings to be added")
	}
}

func validDocument() Document {
	return Document{
		Version: "1.0",
		Source: Source{
			Title:        "Night Rain",
			Author:       "Demo Author",
			Language:     "zh-CN",
			ChapterCount: 3,
			Chapters: []SourceChapter{
				{Index: 1, Title: "Chapter 1", Summary: "The protagonist returns home."},
				{Index: 2, Title: "Chapter 2", Summary: "A suspicious clue appears."},
				{Index: 3, Title: "Chapter 3", Summary: "The mystery escalates."},
			},
		},
		Adaptation: Adaptation{
			Style:    "Suspense Drama",
			Audience: "General",
			Notes:    []string{"Keep a strong hook in each scene"},
		},
		Characters: []Character{
			{
				ID:          "char_lin_qi",
				Name:        "Lin Qi",
				Role:        "protagonist",
				Description: "A young writer with sharp instincts.",
			},
		},
		Locations: []Location{
			{
				ID:          "loc_old_apartment",
				Name:        "Old Apartment",
				Description: "A dimly lit apartment corridor.",
			},
		},
		Scenes: []Scene{
			{
				ID:             "scene_001",
				Title:          "Return Home",
				SourceChapters: []int{1},
				Slugline: Slugline{
					InteriorExterior: "INT",
					LocationID:       "loc_old_apartment",
					Time:             "NIGHT",
				},
				Summary:   "Lin Qi notices that the lock has been disturbed.",
				Objective: "Establish mystery and tension.",
				Beats: []Beat{
					{
						Type:    "action",
						Content: "Lin Qi freezes in front of the apartment door.",
					},
					{
						Type:        "dialogue",
						CharacterID: "char_lin_qi",
						Content:     "I know I locked this this morning.",
						Emotion:     "uneasy",
					},
				},
				Notes: SceneNotes{
					AdaptationReason: "Turn internal monologue into a visible action and short dialogue.",
					OpenQuestions:    []string{},
				},
				Evidence: &Evidence{
					ChapterIndexes: []int{1},
					Excerpt:        "Lin Qi notices that the lock has been disturbed.",
					Cues:           []string{"disturbed lock", "return home"},
				},
			},
		},
		Validation: Validation{
			Status:   "passed",
			Warnings: []string{},
		},
	}
}
