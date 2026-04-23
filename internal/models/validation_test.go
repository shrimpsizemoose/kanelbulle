package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEntryValidateCourseLength(t *testing.T) {
	entry := Entry{
		Timestamp: 1710000000,
		EventType: "100_lab_finish",
		Lab:       "l1",
		Student:   "john.doe",
		Course:    "ITMOPDA26",
	}

	require.NoError(t, entry.Validate())

	entry.Course = strings.Repeat("X", 33)
	require.Error(t, entry.Validate())
}

func TestScoreOverrideValidateCourseLength(t *testing.T) {
	override := ScoreOverride{
		Student: "john.doe",
		Lab:     "l1",
		Score:   10,
		Course:  "ITMOPDA26",
	}

	require.NoError(t, override.Validate())

	override.Course = strings.Repeat("X", 33)
	require.Error(t, override.Validate())
}

func TestLabScoreValidateCourseLength(t *testing.T) {
	labScore := LabScore{
		Deadline:  1710000000,
		Lab:       "l1",
		BaseScore: 10,
		Course:    "ITMOPDA26",
	}

	require.NoError(t, labScore.Validate())

	labScore.Course = strings.Repeat("X", 33)
	require.Error(t, labScore.Validate())
}
