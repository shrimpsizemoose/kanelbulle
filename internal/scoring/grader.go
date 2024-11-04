// internal/scoring/scoring.go
package scoring

import (
	"github.com/shrimpsizemoose/kanelbulle/internal/models"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type Grader struct {
	LateDaysModifiers  map[int]int `toml:"late_days_modifiers"`
	DefaultLatePenalty float64     `toml:"default_late_penalty"`
	MaxLateDays        int         `toml:"max_late_days"`
	ExtraLatePenalty   int         `toml:"extra_late_penalty"`
}

func (g *Grader) CalculateScore(baseScore int, deadline, submitTime int64) int {
	deltaDays := int((submitTime - deadline) / (24 * 60 * 60))

	if deltaDays <= 0 {
		return baseScore
	}

	if modifier, exists := g.LateDaysModifiers[deltaDays]; exists {
		score := baseScore + modifier
		if score < 0 {
			return 0
		}
		return score
	}

	if deltaDays <= g.MaxLateDays {
		return int(float64(baseScore) * g.DefaultLatePenalty)
	}

	return int(float64(baseScore)*g.DefaultLatePenalty) - g.ExtraLatePenalty

}

func ScoreForStudentLab(store *store.Store, student, lab string) (int, error) {
	override, err := store.GetScoreOverride(student, lab)
	if err == nil && override != nil {
		return override.Score, nil
	}

	finish, err := store.GetFirstFinish(student, lab)
	if err != nil {
		return 0, err
	}
	if finish == nil {
		return 0, nil
	}

	labScore, err := store.GetLabScore(lab, finish.Course)
	if err != nil {
		return 0, err
	}
	if labScore == nil {
		return 0, nil
	}

	score := calculateScoreWithDayMismatchModifier(
		labScore.BaseScore,
		labScore.Deadline,
		finish.Timestamp,
	)

	return score, nil
}
