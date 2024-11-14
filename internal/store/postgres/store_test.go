package postgres

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

// setupTestDB creates an in-memory Postgres database and initializes schema
func setupTestDB(t *testing.T) (*PostgresStore, func()) {
	ctx := context.Background()

	postgres, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	dsn, err := postgres.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	s, err := NewPostgresStore(dsn, "../../../migrations")
	require.NoError(t, err, "Failed to create store")

	cleanup := func() {
		s.Close()
		postgres.Terminate(ctx)
	}

	return s, cleanup
}

type testData struct {
	store *PostgresStore
	now   time.Time
}

func setupTestData(t *testing.T) (*testData, func()) {
	s, cleanup := setupTestDB(t)
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// Lab scores setup
	_, err := s.DB.Exec(`
		INSERT INTO lab_scores (lab, course, deadline, base_score) VALUES 
		('l1', 'cs101', $1, 10),
		('l2', 'cs101', $2, 15),
		('l3', 'cs101', $3, 20)`,
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
	flag.Parse()
	if testing.Short() {
		log.Println("Skipping Postgres integration tests. Use -short=false to run them.")
		os.Exit(0)
	}
	log.Println("Starting Postgres store tests...")
	code := m.Run()
	log.Println("Finished Postgres store tests")
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
		got, err := td.store.GetStudentFinishEvent(entry.Student, entry.Lab, entry.Course)
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
		got, err := td.store.GetStudentFinishEvent("john.doe", "l1", "cs101")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, entries[1].Timestamp, got.Timestamp)
		assert.Equal(t, "100_lab_finish", got.EventType)
	})

	t.Run("get non-existent event", func(t *testing.T) {
		got, err := td.store.GetStudentFinishEvent("not.exists", "l1", "cs101")
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
		got, err := td.store.GetScoreOverride(override.Student, override.Lab, override.Course)
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
		score, err := td.store.GetLabScore("l1", "cs101")
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
