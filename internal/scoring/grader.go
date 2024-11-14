// internal/scoring/grader.go
package scoring

import (
	"fmt"
	"math"

	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type Grader struct {
	store              store.ScoreStore
	lateDaysModifiers  map[int]int
	defaultLatePenalty float64
	maxLateDays        int
	extraLatePenalty   int
}

func NewGrader(store store.ScoreStore, lateDaysModifiers map[int]int, defaultPenalty float64, maxLateDays, extraPenalty int) *Grader {
	return &Grader{
		store:              store,
		lateDaysModifiers:  lateDaysModifiers,
		defaultLatePenalty: defaultPenalty,
		maxLateDays:        maxLateDays,
		extraLatePenalty:   extraPenalty,
	}
}

func (g *Grader) CalculateScore(baseScore int, deadline, submitTime int64) int {
	if submitTime <= deadline {
		return baseScore
	}

	deltaDays := int(math.Ceil(float64(submitTime-deadline) / float64(24*60*60)))

	clamp := func(score int) int {
		if score < 0 {
			return 0
		}
		return score
	}

	if modifier, exists := g.lateDaysModifiers[deltaDays]; exists {
		return clamp(baseScore + modifier)
	}

	if deltaDays <= g.maxLateDays {
		return clamp(int(float64(baseScore) * g.defaultLatePenalty))
	}

	return clamp(int(float64(baseScore)*g.defaultLatePenalty) - g.extraLatePenalty)
}

func (g *Grader) ScoreForStudent(course, lab, student string) (int, error) {
	override, err := g.store.GetScoreOverride(course, lab, student)
	if err != nil {
		return 0, fmt.Errorf("failed to check score override: %w", err)
	}
	if override != nil {
		return override.Score, nil
	}

	finishEvent, err := g.store.GetStudentFinishEvent(course, lab, student)

	if err != nil {
		return 0, fmt.Errorf("failed to get finish events: %w", err)
	}
	if finishEvent == nil {
		return 0, err
	}

	labScore, err := g.store.GetLabScore(course, lab)
	if err != nil {
		return 0, err
	}
	if labScore == nil {
		return 0, nil
	}

	return g.CalculateScore(labScore.BaseScore, labScore.Deadline, finishEvent.Timestamp), nil
}
