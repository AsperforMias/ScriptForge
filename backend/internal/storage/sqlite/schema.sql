CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  source_title TEXT NOT NULL,
  status TEXT NOT NULL,
  current_stage TEXT NOT NULL,
  generation_mode TEXT NOT NULL,
  warning_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  input_snapshot_path TEXT NOT NULL,
  result_yaml_path TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS job_stages (
  job_id TEXT NOT NULL,
  stage_name TEXT NOT NULL,
  status TEXT NOT NULL,
  warning_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (job_id, stage_name)
);

CREATE TABLE IF NOT EXISTS artifacts (
  job_id TEXT PRIMARY KEY,
  yaml_path TEXT NOT NULL,
  yaml_size_bytes INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
