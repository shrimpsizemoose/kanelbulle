package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

type ScoreStore interface {
	Close() error
	ApplyMigrations(dir string) error

	CreateEntry(entry *models.Entry) error
	GetStudentFinishEvent(course, lab, student string) (*models.Entry, error)
	ListEntries(course string) ([]models.Entry, error)

	GetScoreOverride(course, lab, student string) (*models.ScoreOverride, error)
	CreateScoreOverride(override models.ScoreOverride) error
	ListCourseScoreOverrides(course string) ([]models.ScoreOverride, error)

	CreateLabScore(labScore models.LabScore) error
	GetLabScore(course, lab string) (*models.LabScore, error)
	ListLabScores(course string) ([]models.LabScore, error)
	GetCourseEventsByType(course, eventType string) ([]models.Entry, error)
	GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]StatResult, error)
}

// BaseStore provides common functionality for different DB implementations
type BaseStore struct {
	DB        *sqlx.DB
	Converter func(string) string
}

func (s *BaseStore) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

// ApplyMigrations applies SQL migrations from a directory, translating dialect if needed
func (s *BaseStore) ApplyMigrations(dir string, translateSQL func(string) string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file.Name(), err)
		}

		sql := string(content)
		if translateSQL != nil {
			sql = translateSQL(sql)
		}

		if _, err := s.DB.Exec(sql); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (s *BaseStore) CreateEntry(entry *models.Entry) error {
	_, err := s.DB.NamedExec(`
		INSERT INTO entries (timestamp, event_type, lab, student, course, comment)
		VALUES (:timestamp, :event_type, :lab, :student, :course, :comment)
	`, entry)
	if err != nil {
		return fmt.Errorf("failed to create entry: %w", err)
	}
	return nil
}

func (s *BaseStore) GetStudentFinishEvent(course, lab, student string) (*models.Entry, error) {
	var entry models.Entry
	query := s.Converter(`
        SELECT timestamp, event_type, lab, student, course, comment
        FROM entries
        WHERE course = ?
	        AND lab = ?
	        AND student = ?
        AND event_type = '100_lab_finish'
        ORDER BY timestamp ASC
        LIMIT 1
    `)

	err := s.DB.Get(&entry, query, course, lab, student)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get finish event: %w", err)
	}
	return &entry, nil
}

func (s *BaseStore) ListEntries(course string) ([]models.Entry, error) {
	var entries []models.Entry
	query := s.Converter(`
		SELECT
			timestamp,
			event_type,
			lab,
			student,
			course,
			comment
		FROM entries
		WHERE course = ?
		ORDER BY student, course, lab, timestamp ASC
	`)

	err := s.DB.Select(&entries, query, course)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	return entries, nil
}

func (s *BaseStore) CreateScoreOverride(override models.ScoreOverride) error {
	_, err := s.DB.NamedExec(`
		INSERT INTO score_overrides (student, lab, score, course, reason)
		VALUES (:student, :lab, :score, :course, :reason)
		ON CONFLICT(course, lab, student) DO UPDATE SET
		score = :score,
		reason = :reason
	`, override)
	if err != nil {
		return fmt.Errorf("failed to create score override: %w", err)
	}
	return nil
}

func (s *BaseStore) GetScoreOverride(course, lab, student string) (*models.ScoreOverride, error) {
	var override models.ScoreOverride
	query := s.Converter(`
		SELECT student, lab, score, course, reason
		FROM score_overrides
		WHERE course = ?
			AND lab = ?
			AND student = ?
	`)

	err := s.DB.Get(&override, query, course, lab, student)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to get score override: %w", err)
	}
	return &override, nil
}

func (s *BaseStore) ListCourseScoreOverrides(course string) ([]models.ScoreOverride, error) {
	var overrides []models.ScoreOverride
	query := s.Converter(`
		SELECT student, lab, score, course, reason 
		FROM score_overrides 
		WHERE course = ?
		ORDER BY course, lab, student
	`)
	err := s.DB.Select(&overrides, query, course)
	if err != nil {
		return nil, fmt.Errorf("failed to list score overrides: %w", err)
	}
	return overrides, nil
}

func (s *BaseStore) CreateLabScore(labScore models.LabScore) error {
	_, err := s.DB.NamedExec(`
		INSERT INTO lab_scores (deadline, lab, base_score, course)
		VALUES (:deadline, :lab, :base_score, :course)
		ON CONFLICT(course, lab) DO UPDATE SET
		base_score = :base_score,
		deadline = :deadline
	`, labScore)
	if err != nil {
		return fmt.Errorf("failed to register lab score: %w", err)
	}
	return nil
}

func (s *BaseStore) GetLabScore(course, lab string) (*models.LabScore, error) {
	var score models.LabScore
	query := s.Converter(`
			SELECT deadline, lab, base_score, course
			FROM lab_scores
			WHERE course = ? AND lab = ?
	`)
	err := s.DB.Get(&score, query, course, lab)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lab score: %w", err)
	}
	return &score, nil
}

func (s *BaseStore) ListLabScores(course string) ([]models.LabScore, error) {
	var labScores []models.LabScore
	query := s.Converter(`
		SELECT
			deadline,
			lab,
			base_score,
			course
		FROM lab_scores
		WHERE course = ?
		ORDER BY lab ASC
	`)

	err := s.DB.Select(&labScores, query, course)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lab scores: %w", err)
	}

	return labScores, nil
}

func (s *BaseStore) GetCourseEventsByType(course, eventType string) ([]models.Entry, error) {
	var entries []models.Entry
	query := s.Converter(`
		SELECT timestamp, event_type, lab, student, course, comment
		FROM entries
		WHERE course = ? AND event_type = ?
		ORDER BY student, lab, timestamp ASC
	`)

	err := s.DB.Select(&entries, query, course, eventType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get course events: %w", err)
	}

	return entries, nil
}
