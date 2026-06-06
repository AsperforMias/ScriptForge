package screenplay

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	validRoles     = []string{"protagonist", "supporting", "antagonist", "narrator", "other"}
	validBeatTypes = []string{"action", "dialogue", "transition", "note"}
	validIntExt    = []string{"INT", "EXT", "INT/EXT"}
)

type ValidatedDocument struct {
	Document Document
	YAMLText string
	Warnings []string
}

func ValidateAndSerialize(doc Document) (ValidatedDocument, error) {
	if err := Validate(doc); err != nil {
		return ValidatedDocument{}, err
	}

	if strings.TrimSpace(doc.Validation.Status) == "" {
		doc.Validation.Status = "passed"
	}
	if doc.Validation.Warnings == nil {
		doc.Validation.Warnings = []string{}
	}

	payload, err := yaml.Marshal(doc)
	if err != nil {
		return ValidatedDocument{}, fmt.Errorf("marshal yaml: %w", err)
	}

	return ValidatedDocument{
		Document: doc,
		YAMLText: string(payload),
		Warnings: doc.Validation.Warnings,
	}, nil
}

func Validate(doc Document) error {
	if strings.TrimSpace(doc.Version) != "1.0" {
		return fmt.Errorf("version must be 1.0")
	}

	if err := validateSource(doc.Source); err != nil {
		return err
	}
	if err := validateAdaptation(doc.Adaptation); err != nil {
		return err
	}
	if err := validateCharacters(doc.Characters); err != nil {
		return err
	}
	if err := validateLocations(doc.Locations); err != nil {
		return err
	}
	if err := validateScenes(doc.Scenes, doc.Source, doc.Characters, doc.Locations); err != nil {
		return err
	}
	if err := validateValidation(doc.Validation); err != nil {
		return err
	}

	return nil
}

func validateSource(source Source) error {
	if strings.TrimSpace(source.Title) == "" {
		return fmt.Errorf("source.title is required")
	}
	if source.Language != "zh-CN" {
		return fmt.Errorf("source.language must be zh-CN")
	}
	if source.ChapterCount < 3 {
		return fmt.Errorf("source.chapter_count must be at least 3")
	}
	if len(source.Chapters) != source.ChapterCount {
		return fmt.Errorf("source.chapters length must equal chapter_count")
	}
	for idx, chapter := range source.Chapters {
		expected := idx + 1
		if chapter.Index != expected {
			return fmt.Errorf("source.chapters index must be continuous from 1")
		}
		if strings.TrimSpace(chapter.Title) == "" {
			return fmt.Errorf("source.chapters title is required")
		}
	}

	return nil
}

func validateAdaptation(adaptation Adaptation) error {
	if strings.TrimSpace(adaptation.Style) == "" {
		return fmt.Errorf("adaptation.style is required")
	}
	return nil
}

func validateCharacters(characters []Character) error {
	seen := make(map[string]struct{}, len(characters))
	for _, character := range characters {
		if strings.TrimSpace(character.ID) == "" {
			return fmt.Errorf("character.id is required")
		}
		if strings.TrimSpace(character.Name) == "" {
			return fmt.Errorf("character.name is required")
		}
		if !slices.Contains(validRoles, character.Role) {
			return fmt.Errorf("character.role must be one of %v", validRoles)
		}
		if _, ok := seen[character.ID]; ok {
			return fmt.Errorf("character.id must be unique")
		}
		seen[character.ID] = struct{}{}
	}

	return nil
}

func validateLocations(locations []Location) error {
	seen := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		if strings.TrimSpace(location.ID) == "" {
			return fmt.Errorf("location.id is required")
		}
		if strings.TrimSpace(location.Name) == "" {
			return fmt.Errorf("location.name is required")
		}
		if _, ok := seen[location.ID]; ok {
			return fmt.Errorf("location.id must be unique")
		}
		seen[location.ID] = struct{}{}
	}

	return nil
}

func validateScenes(scenes []Scene, source Source, characters []Character, locations []Location) error {
	if len(scenes) == 0 {
		return fmt.Errorf("scenes must not be empty")
	}

	validChapters := make(map[int]struct{}, len(source.Chapters))
	for _, chapter := range source.Chapters {
		validChapters[chapter.Index] = struct{}{}
	}

	validCharacterIDs := make(map[string]struct{}, len(characters))
	for _, character := range characters {
		validCharacterIDs[character.ID] = struct{}{}
	}

	validLocationIDs := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		validLocationIDs[location.ID] = struct{}{}
	}

	sceneIDs := make(map[string]struct{}, len(scenes))
	for _, scene := range scenes {
		if strings.TrimSpace(scene.ID) == "" {
			return fmt.Errorf("scene.id is required")
		}
		if _, ok := sceneIDs[scene.ID]; ok {
			return fmt.Errorf("scene.id must be unique")
		}
		sceneIDs[scene.ID] = struct{}{}

		if strings.TrimSpace(scene.Title) == "" {
			return fmt.Errorf("scene.title is required")
		}
		if len(scene.SourceChapters) == 0 {
			return fmt.Errorf("scene.source_chapters must not be empty")
		}
		for _, chapterIndex := range scene.SourceChapters {
			if _, ok := validChapters[chapterIndex]; !ok {
				return fmt.Errorf("scene.source_chapters contains undefined chapter index")
			}
		}
		if !slices.Contains(validIntExt, scene.Slugline.InteriorExterior) {
			return fmt.Errorf("scene.slugline.interior_exterior must be one of %v", validIntExt)
		}
		if _, ok := validLocationIDs[scene.Slugline.LocationID]; !ok {
			return fmt.Errorf("scene.slugline.location_id must reference a defined location")
		}
		if strings.TrimSpace(scene.Slugline.Time) == "" {
			return fmt.Errorf("scene.slugline.time is required")
		}
		if strings.TrimSpace(scene.Summary) == "" {
			return fmt.Errorf("scene.summary is required")
		}
		if len(scene.Beats) == 0 {
			return fmt.Errorf("scene.beats must not be empty")
		}

		for _, beat := range scene.Beats {
			if !slices.Contains(validBeatTypes, beat.Type) {
				return fmt.Errorf("scene.beat.type must be one of %v", validBeatTypes)
			}
			if strings.TrimSpace(beat.Content) == "" {
				return fmt.Errorf("scene.beat.content is required")
			}
			if beat.Type == "dialogue" {
				if _, ok := validCharacterIDs[beat.CharacterID]; !ok {
					return fmt.Errorf("scene.dialogue.character_id must reference a defined character")
				}
			}
		}
	}

	return nil
}

func validateValidation(validation Validation) error {
	switch validation.Status {
	case "passed", "failed":
		return nil
	default:
		return fmt.Errorf("validation.status must be passed or failed")
	}
}
