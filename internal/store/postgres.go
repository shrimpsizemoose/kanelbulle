package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(dsn string) (*Store, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) ApplyMigrations(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(fmt.Sprintf("migrations/%s", file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file.Name(), err)
		}

		logger.Info.Printf("Applying migration: %s", file.Name())
		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (s *Store) CreateEntry(entry *models.Entry) error {
	_, err := s.db.NamedExec(`
		INSERT INTO entries (timestamp, event_type, lab, student, course, comment)
		VALUES (:timestamp, :event_type, :lab, :student, :course, :comment)
	`, entry)
	if err != nil {
		return fmt.Errorf("failed to create lab log entry: %w", err)
	}
	return nil
}

func (s *Store) CreateScoreOverride(override *models.ScoreOverride) error {
	_, err := s.db.NamedExec(`
        INSERT INTO score_overrides (student, lab, score, course, reason)
        VALUES (:student, :lab, :score, :course, :reason)
    `, override)
	if err != nil {
		return fmt.Errorf("failed to create score override entry: %w", err)
	}
	return nil
}

func (s *Store) GetScoreOverride(student, lab, course string) (*models.ScoreOverride, error) {
	var override models.ScoreOverride
	err := s.db.Get(&override, `
        SELECT * FROM score_overrides 
        WHERE student = $1 AND lab = $2 AND course = $3
    `, student, lab, course)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &override, err
}

func (s *Store) ListScoreOverrides() ([]models.ScoreOverride, error) {
	var overrides []models.ScoreOverride
	err := s.db.Select(&overrides, `
        SELECT * FROM score_overrides 
        ORDER BY course, lab, student
    `)
	return overrides, err
}

func (s *Store) ListEntries(course string) ([]models.Entry, error) {
	var entries []models.Entry
	err := s.db.Select(
		&entries, `
		SELECT
			timestamp,
			event_type,
			lab,
			student,
			course,
			comment
		FROM entries
		WHERE course = course
		ORDER BY student, course, lab, timestamp ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	return entries, err
}

type StatResult struct {
	Student          string  `db:"student"`
	Lab              string  `db:"lab"`
	Course           string  `db:"course"`
	StartCount       int64   `db:"start_count"`
	FirstRun         int64   `db:"first_run"`
	FirstFinish      *int64  `db:"first_finish"`
	DeltaSeconds     *int64  `db:"delta_seconds"`
	HumanFirstRun    *string `db:"human_first_run"`
	HumanFirstFinish *string `db:"human_first_finish"`
}

func (s *Store) GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]StatResult, error) {
	query := `
		WITH start_events AS (
            SELECT 
                student,
                lab,
                course,
                COUNT(*) as start_count,
                MIN(timestamp) as first_run
            FROM entries
            WHERE 1=1
                AND course = $1
                AND event_type = $2
            GROUP BY student, lab, course
        ),
        finish_events AS (
            SELECT
                student,
                lab,
                course,
                MIN(timestamp) as first_finish
            FROM entries
            WHERE 1=1
                AND course = $1
                AND event_type = $3
            GROUP BY student, lab, course
        )
        SELECT 
            se.student,
            se.lab,
            se.course,
            se.start_count,
            se.first_run,
            fe.first_finish,
            CASE 
                WHEN fe.first_finish IS NOT NULL 
                THEN fe.first_finish - se.first_run
            END as delta_seconds,
            CASE WHEN $4 THEN
                to_char(to_timestamp(se.first_run), $5)
            ELSE NULL
            END as human_first_run,
            CASE WHEN $4 AND fe.first_finish IS NOT NULL THEN
                to_char(to_timestamp(fe.first_finish), $5)
            ELSE NULL
            END as human_first_finish
        FROM start_events se
        LEFT JOIN finish_events fe
            ON se.student = fe.student 
            AND se.lab = fe.lab 
            AND se.course = fe.course
        ORDER BY se.student, se.lab`

	var results []StatResult
	err := s.db.Select(&results, query,
		course,
		startEventType,
		finishEventType,
		includeHumanDttm,
		timestampFormat,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	return results, nil
}

func (s *Store) GetEventsByType(eventType string) ([]models.Entry, error) {
	var entries []models.Entry
	err := s.db.Select(&entries, `
		SELECT timestamp, event_type, lab, student, course, comment
		FROM entries
		WHERE event_type = $1
		ORDER BY timestamp ASC
	`, eventType)
	if err != nil {
		return nil, fmt.Errorf("faild to get events by type: %w", err)
	}

	return entries, nil
}

func (s *Store) GetLabScore(lab, course string) (*models.LabScore, error) {
	var score models.LabScore
	err := s.db.Get(&score, `
        SELECT deadline, lab, base_score, course FROM lab_scores 
        WHERE lab = $1 AND course = $2
    `, lab, course)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &score, nil
}

type ScoringResult struct {
	Student string `db:"student"`
	Lab     string `db:"lab"`
	Score   int    `db:"score"`
}

func (s *Store) FetchScoringStats(course, eventFinishType string) ([]ScoringResult, error) {
	// FIXME: use real calc
	query := `
        WITH lab_finishes AS (
            SELECT 
                student,
                lab,
                MIN(timestamp) as first_finish
            FROM entries
            WHERE event_type = $2
            AND course = $1
            GROUP BY student, lab
        ),
        lab_scores AS (
            SELECT 
                lab,
                base_score
            FROM lab_scores
            WHERE course = $1
        )
        SELECT 
            lf.student,
            lf.lab,
            ls.base_score as score
        FROM lab_finishes lf
        JOIN lab_scores ls ON ls.lab = lf.lab
        ORDER BY lf.student, lf.lab
    `

	var results []ScoringResult
	err := s.db.Select(&results, query, course, eventFinishType)
	if err != nil {
		return nil, fmt.Errorf("failed to get scoring stats: %w", err)
	}

	return results, nil
}
