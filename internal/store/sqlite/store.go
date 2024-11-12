// internal/store/sqlite/store.go
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

func NewSQLiteStore(config *store.DBConfig) (*SQLiteStore, error) {
	db, err := sqlx.Connect("sqlite3", config.DSN)
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

	if err := s.ApplyMigrations("../../../migrations", translateToSQLite); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return s, nil
}

// translateToSQLite converts Postgres SQL to SQLite dialect
func translateToSQLite(sql string) string {
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
