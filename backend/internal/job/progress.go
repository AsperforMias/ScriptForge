package job

var stageProgress = map[string]int{
	"ingest":                5,
	"outline":               20,
	"entities":              35,
	"scene_planning":        55,
	"screenplay_generation": 75,
	"validation":            90,
	"persistence":           95,
}

func ProgressPercentForStage(stageName string) int {
	progress, ok := stageProgress[stageName]
	if !ok {
		return 0
	}
	return progress
}
