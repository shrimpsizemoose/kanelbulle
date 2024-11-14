package sqlite

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shrimpsizemoose/kanelbulle/internal/models"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type SQLiteStore struct {
	store.BaseStore
}

func NewSQLiteStore(dsn, migrationsDir string) (*SQLiteStore, error) {
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	s := &SQLiteStore{BaseStore: store.BaseStore{
		DB: db,
		Converter: func(query string) string {
			return query
		},
	}}

	if err := s.ApplyMigrations(migrationsDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) ApplyMigrations(dir string) error {
	// translateToSQLite converts Postgres SQL to SQLite dialect
	translateToSQLite := func(sql string) string {
		replacements := map[string]string{
			"BIGSERIAL":                        "INTEGER PRIMARY KEY AUTOINCREMENT",
			"SERIAL":                           "INTEGER PRIMARY KEY AUTOINCREMENT",
			"BIGINT":                           "INTEGER",
			"UUID":                             "TEXT",
			"TRUE":                             "1",
			"FALSE":                            "0",
			"RETURNING":                        "",
			"to_timestamp":                     "datetime",
			"now()":                            "CURRENT_TIMESTAMP",
			"VARCHAR(3)":                       "TEXT",
			"VARCHAR(6)":                       "TEXT",
			`CHECK (student ~ '^[\w-]+\..+$')`: "",
			"::text":                           "",
		}
		result := sql
		for from, to := range replacements {
			result = strings.ReplaceAll(result, from, to)
		}
		return result
	}

	return s.BaseStore.ApplyMigrations(dir, translateToSQLite)
}

// Fetch Scoring stats for given event type
func (s *SQLiteStore) FetchScoringStats(course, eventType string) ([]models.ScoringResult, error) {
	query := `
		WITH lab_finishes AS (
			SELECT student, lab, MIN(timestamp) as first_finish
			FROM entries
			WHERE event_type = ?
			AND course = ?
			GROUP BY student, lab
		)
		SELECT 
			f.student,
			f.lab,
			s.base_score as score
		FROM lab_finishes f
		JOIN lab_scores s ON s.lab = f.lab AND s.course = ?
		ORDER BY f.student, f.lab
	`

	var results []models.ScoringResult
	err := s.DB.Select(&results, query, eventType, course, course)
	if err != nil {
		return nil, fmt.Errorf("failed to get scoring stats: %w", err)
	}

	return results, nil
}

func (s *SQLiteStore) GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]store.StatResult, error) {

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
                AND course = ?
                AND event_type = ?
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
                AND course = ?
                AND event_type = ?
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
            CASE WHEN ? THEN
                datetime(se.first_run, 'unixepoch', 'localtime')
            ELSE NULL
            END as human_first_run,
            CASE WHEN ? AND fe.first_finish IS NOT NULL THEN
                datetime(fe.first_finish, 'unixepoch', 'localtime')
            ELSE NULL
            END as human_first_finish
        FROM start_events se
        LEFT JOIN finish_events fe
            ON se.student = fe.student
            AND se.lab = fe.lab
            AND se.course = fe.course
        ORDER BY se.student, se.lab
    `

	var results []store.StatResult
	err := s.DB.Select(&results, query,
		course,
		startEventType,
		course,
		finishEventType,
		includeHumanDttm,
		includeHumanDttm,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}

	return results, nil
}
