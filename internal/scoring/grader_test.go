package scoring

import (
	"testing"

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
	testCases := []struct {
		name          string
		baseScore     int
		deadline      int64
		submitTime    int64
		expectedScore int
	}{
		{
			name:          "On-time submission",
			baseScore:     10,
			deadline:      1680288000, // April 1, 2023
			submitTime:    1680288000,
			expectedScore: 10,
		},
		{
			name:          "Late submission with modifier",
			baseScore:     10,
			deadline:      1680288000, // April 1, 2023
			submitTime:    1680374400, // April 2, 2023
			expectedScore: 9,
		},
		{
			name:          "Late submission with default penalty",
			baseScore:     10,
			deadline:      1680288000, // April 1, 2023
			submitTime:    1680547200, // April 4, 2023
			expectedScore: 7,
		},
		{
			name:          "Late submission with extra penalty",
			baseScore:     10,
			deadline:      1680288000, // April 1, 2023
			submitTime:    1680806400, // April 8, 2023
			expectedScore: 0,
		},
		{
			name:          "Negative score capped at 0",
			baseScore:     10,
			deadline:      1680288000, // April 1, 2023
			submitTime:    1681411200, // April 15, 2023
			expectedScore: 0,
		},
	}

	grader := NewGrader(
		&MockStore{},
		map[int]int{1: -1, 2: -2, 3: -3},
		0.7,
		5,
		5,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := grader.CalculateScore(tc.baseScore, tc.deadline, tc.submitTime)
			assert.Equal(t, tc.expectedScore, score)
		})
	}
}
