package screenplay

type Document struct {
	Version    string      `json:"version" yaml:"version"`
	Source     Source      `json:"source" yaml:"source"`
	Adaptation Adaptation  `json:"adaptation" yaml:"adaptation"`
	Characters []Character `json:"characters" yaml:"characters"`
	Locations  []Location  `json:"locations" yaml:"locations"`
	Scenes     []Scene     `json:"scenes" yaml:"scenes"`
	Validation Validation  `json:"validation" yaml:"validation"`
}

type Source struct {
	Title        string          `json:"title" yaml:"title"`
	Author       string          `json:"author,omitempty" yaml:"author,omitempty"`
	Language     string          `json:"language" yaml:"language"`
	ChapterCount int             `json:"chapter_count" yaml:"chapter_count"`
	Chapters     []SourceChapter `json:"chapters" yaml:"chapters"`
}

type SourceChapter struct {
	Index   int    `json:"index" yaml:"index"`
	Title   string `json:"title" yaml:"title"`
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
}

type Adaptation struct {
	Style    string   `json:"style" yaml:"style"`
	Audience string   `json:"audience,omitempty" yaml:"audience,omitempty"`
	Notes    []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type Character struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Aliases     []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Role        string   `json:"role" yaml:"role"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type Location struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Scene struct {
	ID             string     `json:"id" yaml:"id"`
	Title          string     `json:"title" yaml:"title"`
	SourceChapters []int      `json:"source_chapters" yaml:"source_chapters"`
	Slugline       Slugline   `json:"slugline" yaml:"slugline"`
	Summary        string     `json:"summary" yaml:"summary"`
	Objective      string     `json:"objective,omitempty" yaml:"objective,omitempty"`
	Beats          []Beat     `json:"beats" yaml:"beats"`
	Notes          SceneNotes `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type Slugline struct {
	InteriorExterior string `json:"interior_exterior" yaml:"interior_exterior"`
	LocationID       string `json:"location_id" yaml:"location_id"`
	Time             string `json:"time" yaml:"time"`
}

type Beat struct {
	Type        string `json:"type" yaml:"type"`
	CharacterID string `json:"character_id,omitempty" yaml:"character_id,omitempty"`
	Content     string `json:"content" yaml:"content"`
	Emotion     string `json:"emotion,omitempty" yaml:"emotion,omitempty"`
}

type SceneNotes struct {
	AdaptationReason string   `json:"adaptation_reason,omitempty" yaml:"adaptation_reason,omitempty"`
	OpenQuestions    []string `json:"open_questions,omitempty" yaml:"open_questions,omitempty"`
}

type Validation struct {
	Status   string   `json:"status" yaml:"status"`
	Warnings []string `json:"warnings" yaml:"warnings"`
}
