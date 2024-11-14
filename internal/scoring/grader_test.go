package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) Close() error {
	return nil
}

func (m *MockStore) ApplyMigrations(dir string) error {
	return nil
}

func (m *MockStore) CreateEntry(entry *models.Entry) error {
	return nil
}

func (m *MockStore) GetStudentFinishEvent(course, lab, student string) (*models.Entry, error) {
	args := m.Called(course, lab, student)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Entry), args.Error(1)
}

func (m *MockStore) ListEntries(course string) ([]models.Entry, error) {
	return nil, nil
}

func (m *MockStore) GetScoreOverride(course, lab, student string) (*models.ScoreOverride, error) {
	args := m.Called(course, lab, student)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ScoreOverride), args.Error(1)
}

func (m *MockStore) CreateScoreOverride(override models.ScoreOverride) error {
	return nil
}

func (m *MockStore) ListScoreOverrides() ([]models.ScoreOverride, error) {
	return nil, nil
}

func (m *MockStore) GetLabScore(course, lab string) (*models.LabScore, error) {
	args := m.Called(course, lab)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LabScore), args.Error(1)
}

func (m *MockStore) FetchScoringStats(course, eventFinishType string) ([]models.ScoringResult, error) {
	return nil, nil
}

func (m *MockStore) GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]store.StatResult, error) {
	return nil, nil
}

func TestGrader_CalculateScore(t *testing.T) {

	deadline := time.Date(2024, 4, 1, 23, 59, 59, 0, time.UTC)

	testCases := []struct {
		name          string
		baseScore     int
		deadline      time.Time
		submitTime    time.Time
		expectedScore int
	}{
		{
			name:          "Early submission",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(-6 * time.Hour),
			expectedScore: 10,
		},
		{
			name:          "Last minute submission, but still on time",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(-1 * time.Minute),
			expectedScore: 10,
		},
		{
			name:          "One second late counts as one day late",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(1 * time.Second),
			expectedScore: 9,
		},
		{
			name:          "Late submission: 23 hours late treated as 1 day late",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(23 * time.Hour),
			expectedScore: 9,
		},
		{
			name:          "Late submission: 24h1m late treated as 2 days late",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(24*time.Hour + 1*time.Minute),
			expectedScore: 8,
		},
		{
			name:          "Late submission: 25 hours late treated as 2 days late",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(25 * time.Hour),
			expectedScore: 8,
		},
		{
			name:          "Late submission: 49 hours late means 3 days late",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(49 * time.Hour),
			expectedScore: 7,
		},
		{
			name:          "Late submission: 6 days late (baseScore times modifier)",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(6 * 24 * time.Hour),
			expectedScore: 5,
		},
		{
			name:          "Late submission: 10 days late: baseScore times modifier minus extra penalty",
			baseScore:     10,
			deadline:      deadline,
			submitTime:    deadline.Add(10 * 24 * time.Hour),
			expectedScore: 4,
		},
	}

	grader := NewGrader(
		&MockStore{},
		map[int]int{1: -1, 2: -2, 3: -3},
		0.5,
		7,
		1,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := grader.CalculateScore(
				tc.baseScore,
				tc.deadline.Unix(),
				tc.submitTime.Unix(),
			)
			assert.Equal(t, tc.expectedScore, score)
		})
	}
}
