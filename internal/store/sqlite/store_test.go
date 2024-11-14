// internal/store/sqlite/store_test.go
package sqlite

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

// setupTestDB creates an in-memory SQLite database and initializes schema
func setupTestDB(t *testing.T) (*SQLiteStore, func()) {
	// Create tables directly instead of using migrations for tests
	schema := `
	CREATE TABLE IF NOT EXISTS entries (
		timestamp INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		lab TEXT NOT NULL,
		student TEXT NOT NULL,
		course TEXT NOT NULL,
		comment TEXT
	);

	CREATE TABLE IF NOT EXISTS score_overrides (
		student TEXT NOT NULL,
		lab TEXT NOT NULL,
		score INTEGER NOT NULL,
		course TEXT NOT NULL,
		reason TEXT,
		PRIMARY KEY (course, lab, student)
	);

	CREATE TABLE IF NOT EXISTS lab_scores (
		lab TEXT NOT NULL,
		course TEXT NOT NULL,
		deadline INTEGER NOT NULL,
		base_score INTEGER NOT NULL,
		PRIMARY KEY (lab, course)
	);`

	s, err := NewSQLiteStore(":memory:", "../../../migrations")
	require.NoError(t, err, "Failed to create store")

	_, err = s.DB.Exec(schema)
	require.NoError(t, err, "Failed to create schema")

	cleanup := func() {
		err := s.Close()
		require.NoError(t, err, "Failed to close database")
	}

	return s, cleanup
}

type testData struct {
	store *SQLiteStore
	now   time.Time
}

func setupTestData(t *testing.T) (*testData, func()) {
	s, cleanup := setupTestDB(t)
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// Lab scores setup
	_, err := s.DB.Exec(`
		INSERT INTO lab_scores (lab, course, deadline, base_score) VALUES 
		('l1', 'cs101', ?, 10),
		('l2', 'cs101', ?, 15),
		('l3', 'cs101', ?, 20)`,
		time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC).Unix(),
		time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC).Unix(),
		time.Date(2024, 2, 1, 23, 59, 59, 0, time.UTC).Unix(),
	)
	require.NoError(t, err, "Failed to insert test data")

	return &testData{
		store: s,
		now:   now,
	}, cleanup
}

func TestMain(m *testing.M) {
	log.Println("Starting SQLite store tests...")
	code := m.Run()
	log.Println("Finished SQLite store tests")
	os.Exit(code)
}

func TestCreateAndGetEntry(t *testing.T) {
	td, cleanup := setupTestData(t)
	defer cleanup()

	entry := models.Entry{
		Timestamp: td.now.Unix(),
		EventType: "100_lab_finish",
		Lab:       "l1",
		Student:   "john.doe",
		Course:    "cs101",
		Comment:   "test entry",
	}

	t.Run("create entry", func(t *testing.T) {
		err := td.store.CreateEntry(&entry)
		require.NoError(t, err, "Failed to create entry")
	})

	t.Run("get entry", func(t *testing.T) {
		got, err := td.store.GetStudentFinishEvent(entry.Course, entry.Lab, entry.Student)
		require.NoError(t, err, "Failed to get entry")
		require.NotNil(t, got)
		assert.Equal(t, entry.Timestamp, got.Timestamp)
		assert.Equal(t, entry.EventType, got.EventType)
		assert.Equal(t, entry.Lab, got.Lab)
		assert.Equal(t, entry.Student, got.Student)
		assert.Equal(t, entry.Course, got.Course)
		assert.Equal(t, entry.Comment, got.Comment)
	})
}

func TestGetStudentFinishEvent(t *testing.T) {
	td, cleanup := setupTestData(t)
	defer cleanup()

	// Create test entries
	entries := []models.Entry{
		{
			Timestamp: td.now.Add(-2 * time.Hour).Unix(),
			EventType: "000_lab_start",
			Lab:       "l1",
			Student:   "john.doe",
			Course:    "cs101",
		},
		{
			Timestamp: td.now.Add(-1 * time.Hour).Unix(),
			EventType: "100_lab_finish",
			Lab:       "l1",
			Student:   "john.doe",
			Course:    "cs101",
		},
	}

	for _, e := range entries {
		err := td.store.CreateEntry(&e)
		require.NoError(t, err, "Failed to create test entry")
	}

	t.Run("get existing finish event", func(t *testing.T) {
		got, err := td.store.GetStudentFinishEvent("cs101", "l1", "john.doe")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, entries[1].Timestamp, got.Timestamp)
		assert.Equal(t, "100_lab_finish", got.EventType)
	})

	t.Run("get non-existent event", func(t *testing.T) {
		got, err := td.store.GetStudentFinishEvent("cs101", "11", "not.exists")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestScoreOverrideOperations(t *testing.T) {
	td, cleanup := setupTestData(t)
	defer cleanup()

	override := models.ScoreOverride{
		Student: "john.doe",
		Lab:     "l1",
		Course:  "cs101",
		Score:   8,
		Reason:  "late submission accepted",
	}

	t.Run("create override", func(t *testing.T) {
		err := td.store.CreateScoreOverride(override)
		require.NoError(t, err)
	})

	t.Run("get override", func(t *testing.T) {
		got, err := td.store.GetScoreOverride(override.Course, override.Lab, override.Student)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, override.Score, got.Score)
		assert.Equal(t, override.Reason, got.Reason)
	})

	t.Run("list overrides", func(t *testing.T) {
		overrides, err := td.store.ListScoreOverrides()
		require.NoError(t, err)
		assert.Len(t, overrides, 1)
		assert.Equal(t, override.Student, overrides[0].Student)
	})
}

func TestLabScoreOperations(t *testing.T) {
	td, cleanup := setupTestData(t)
	defer cleanup()

	t.Run("get existing score", func(t *testing.T) {
		score, err := td.store.GetLabScore("cs101", "l1")
		require.NoError(t, err)
		require.NotNil(t, score)
		assert.Equal(t, 10, score.BaseScore)
	})

	t.Run("get non-existent score", func(t *testing.T) {
		score, err := td.store.GetLabScore("not.exists", "cs101")
		require.NoError(t, err)
		assert.Nil(t, score)
	})
}
