package screenplay

import (
	"fmt"
	"strings"
)

func applyQualityAudit(doc Document) Document {
	warnings := append([]string{}, doc.Validation.Warnings...)
	objectiveOwners := map[string]string{}
	questionOwners := map[string]string{}
	lowConfidenceScenes := 0
	newIssueCount := 0

	for idx := range doc.Scenes {
		scene := &doc.Scenes[idx]
		sceneIssues := []string{}

		if objective := strings.TrimSpace(scene.Objective); objective != "" {
			normalized := normalizeAuditText(objective)
			if previousSceneID, ok := objectiveOwners[normalized]; ok {
				sceneIssues = append(sceneIssues, fmt.Sprintf("objective duplicates %s; rewrite it from this scene's own evidence.", previousSceneID))
			} else {
				objectiveOwners[normalized] = scene.ID
			}
			if looksLikeObjectivePlaceholder(objective) {
				sceneIssues = append(sceneIssues, "objective still reads like template or placeholder copy; tighten it to a scene-specific dramatic task.")
			}
		}

		if question := firstMeaningfulQuestion(scene.Notes.OpenQuestions); question != "" {
			normalized := normalizeAuditText(question)
			if previousSceneID, ok := questionOwners[normalized]; ok {
				sceneIssues = append(sceneIssues, fmt.Sprintf("open_questions duplicates %s; keep only unresolved questions unique to this scene.", previousSceneID))
			} else {
				questionOwners[normalized] = scene.ID
			}
			if looksLikeQuestionPlaceholder(question) {
				sceneIssues = append(sceneIssues, "open_questions is still generic; keep it empty or rewrite it from explicit unresolved evidence.")
			}
		}

		sceneIssues = append(sceneIssues, auditBeats(*scene)...)
		sceneIssues = uniqueQualityStrings(sceneIssues)
		if len(sceneIssues) > 0 {
			newIssueCount += len(sceneIssues)
			for _, issue := range sceneIssues {
				warnings = append(warnings, fmt.Sprintf("%s: %s", scene.ID, issue))
			}
		}

		scene.Review = mergeSceneReview(scene.Review, sceneIssues)
		if scene.Review != nil && scene.Review.Confidence == "low" {
			lowConfidenceScenes++
		}
	}

	doc.Validation.Warnings = uniqueQualityStrings(warnings)
	if doc.Validation.Status != "failed" && shouldDowngradeValidation(lowConfidenceScenes, newIssueCount) {
		doc.Validation.Status = "failed"
	}
	return doc
}

func auditBeats(scene Scene) []string {
	issues := []string{}
	seen := map[string]string{}

	for _, beat := range scene.Beats {
		content := strings.TrimSpace(beat.Content)
		if content == "" {
			continue
		}
		key := beat.Type + ":" + normalizeAuditText(content)
		if previousType, ok := seen[key]; ok {
			issues = append(issues, fmt.Sprintf("%s beat repeats the same content inside the scene; keep one version only.", previousType))
			continue
		}
		seen[key] = beat.Type

		if beat.Type == "dialogue" && looksLikeDialoguePlaceholder(content) {
			issues = append(issues, "dialogue beat still sounds like placeholder copy; keep only explicit spoken lines from the chapter.")
		}
	}

	return issues
}

func mergeSceneReview(existing *Review, sceneIssues []string) *Review {
	review := &Review{}
	if existing != nil {
		*review = *existing
		review.Issues = append([]string{}, existing.Issues...)
	}

	review.Issues = uniqueQualityStrings(append(review.Issues, sceneIssues...))
	review.Confidence = lowerConfidence(review.Confidence, deriveSceneConfidence(review.Issues))

	if review.Confidence == "" {
		review.Confidence = "high"
	}

	if len(review.Issues) == 0 && review.Confidence == "high" && existing == nil {
		return nil
	}
	return review
}

func deriveSceneConfidence(issues []string) string {
	if len(issues) == 0 {
		return "high"
	}
	if len(issues) >= 3 || containsCriticalIssue(issues) {
		return "low"
	}
	return "medium"
}

func containsCriticalIssue(issues []string) bool {
	for _, issue := range issues {
		switch {
		case strings.Contains(issue, "duplicates"),
			strings.Contains(issue, "placeholder"),
			strings.Contains(issue, "repeats the same content"):
			return true
		}
	}
	return false
}

func shouldDowngradeValidation(lowConfidenceScenes, newIssueCount int) bool {
	if lowConfidenceScenes >= 2 {
		return true
	}
	return lowConfidenceScenes >= 1 && newIssueCount >= 3
}

func firstMeaningfulQuestion(questions []string) string {
	for _, question := range questions {
		trimmed := strings.TrimSpace(question)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func looksLikeObjectivePlaceholder(input string) bool {
	normalized := normalizeAuditText(input)
	switch {
	case normalized == "", normalized == "...", normalized == "待补充":
		return true
	case strings.Contains(normalized, "drive the chapter conflict into a filmable dramatic action"),
		strings.Contains(normalized, "推动剧情继续"),
		strings.Contains(normalized, "建立悬疑氛围"),
		strings.Contains(normalized, "建立悬疑基调"),
		strings.Contains(normalized, "先稳住") && strings.Contains(normalized, "下一步"):
		return true
	default:
		return false
	}
}

func looksLikeQuestionPlaceholder(input string) bool {
	normalized := normalizeAuditText(input)
	switch {
	case normalized == "", normalized == "...", normalized == "待补充":
		return true
	case strings.Contains(normalized, "接下来会发生什么"),
		strings.Contains(normalized, "谁会出现"),
		strings.Contains(normalized, "下一步会怎样"):
		return true
	default:
		return false
	}
}

func looksLikeDialoguePlaceholder(input string) bool {
	normalized := normalizeAuditText(input)
	return strings.Contains(normalized, "我得继续查下去") ||
		strings.Contains(normalized, "先看看情况")
}

func normalizeAuditText(input string) string {
	replacer := strings.NewReplacer(
		"\n", " ",
		"\r", " ",
		"\t", " ",
		"。", "",
		"，", "",
		"？", "",
		"！", "",
		"“", "",
		"”", "",
		"‘", "",
		"’", "",
		"\"", "",
		"'", "",
		" ", "",
	)
	return strings.ToLower(strings.TrimSpace(replacer.Replace(input)))
}

func lowerConfidence(current, candidate string) string {
	if current == "" {
		return candidate
	}
	if confidenceRank(candidate) > confidenceRank(current) {
		return candidate
	}
	return current
}

func confidenceRank(value string) int {
	switch value {
	case "low":
		return 3
	case "medium":
		return 2
	case "high":
		return 1
	default:
		return 0
	}
}

func uniqueQualityStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
