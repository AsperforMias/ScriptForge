export type CharacterRole = "protagonist" | "supporting" | "antagonist" | "narrator" | "other";

export type BeatType = "action" | "dialogue" | "transition" | "note";

export type InteriorExterior = "INT" | "EXT" | "INT/EXT";

export interface SourceChapter {
  index: number;
  title: string;
  summary?: string;
}

export interface ScreenplayCharacter {
  id: string;
  name: string;
  aliases?: string[];
  role: CharacterRole;
  description?: string;
}

export interface ScreenplayLocation {
  id: string;
  name: string;
  description?: string;
}

export interface ScreenplayBeat {
  type: BeatType;
  content: string;
  character_id?: string;
  emotion?: string;
}

export interface ScreenplayScene {
  id: string;
  title: string;
  source_chapters: number[];
  slugline: {
    interior_exterior: InteriorExterior;
    location_id: string;
    time: string;
  };
  summary: string;
  objective?: string;
  beats: ScreenplayBeat[];
  notes?: {
    adaptation_reason?: string;
    open_questions?: string[];
  };
}

export interface ScreenplayDocument {
  version: "1.0";
  source: {
    title: string;
    author?: string;
    language: "zh-CN";
    chapter_count: number;
    chapters: SourceChapter[];
  };
  adaptation: {
    style: string;
    audience?: string;
    notes?: string[];
  };
  characters: ScreenplayCharacter[];
  locations: ScreenplayLocation[];
  scenes: ScreenplayScene[];
  validation: {
    status: "passed" | "failed";
    warnings: string[];
  };
}
