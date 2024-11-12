// internal/scoring/grader.go
package scoring

import (
	"fmt"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type Grader struct {
	LateDaysModifiers  map[int]int
	DefaultLatePenalty float64
	MaxLateDays       int
	ExtraLatePenalty  int
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

	score := int(float64(baseScore) * g.DefaultLatePenalty) - g.ExtraLatePenalty
	if score < 0 {
		return 0
	}
	return score
}

func ScoreForStudentLab(store store.ScoreStore, student, lab, course string) (int, error) {
	// First check for manual override
	override, err := store.GetScoreOverride(student, lab, course)
	if err != nil {
		return 0, fmt.Errorf("failed to check score override: %w", err)
	}
	if override != nil {
		return override.Score, nil
	}

	// Get finish events
	finishEvents, err := store.GetEventsByType("finish")
	if err != nil {
		return 0, fmt.Errorf("failed to get finish events: %w", err)
	}

	var studentFinish *models.Entry
	for _, event := range finishEvents {
		if event.Student == student && event.Lab == lab && event.Course == course {
			studentFinish = &event
			break
		}
	}

	if studentFinish == nil {
		return 0, nil // No finish event found
	}

	// Get lab score configuration
	labScore, err := store.GetLabScore(lab, course)
	if err != nil {
		return 0, fmt.Errorf("failed to get lab score: %w", err)
	}
	if labScore == nil {
		return 0, nil
	}

	// Create grader with default configuration
	// Note: In production, this should come from configuration
	grader := &Grader{
		LateDaysModifiers: map[int]int{
			1: -1,
			2: -2,
			3: -3,
		},
		DefaultLatePenalty: 0.7,
		MaxLateDays:       5,
		ExtraLatePenalty:  5,
	}

	return grader.CalculateScore(labScore.BaseScore, labScore.Deadline, studentFinish.Timestamp), nil
}
