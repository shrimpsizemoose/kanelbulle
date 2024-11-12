package postgres

import (
	// "database/sql"
	"fmt"
	// "os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	// "github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type PostgresStore struct {
	store.BaseStore
}

func NewPostgresStore(config *store.DBConfig) (*PostgresStore, error) {
	db, err := sqlx.Connect("postgres", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &PostgresStore{BaseStore: store.BaseStore{
		DB: db,
		Converter: func(query string) string {
			out := query
			for i := 1; strings.Contains(out, "?"); i++ {
				out = strings.Replace(out, "?", fmt.Sprintf("$%d", i), 1)
			}
			return out
		},
	}}

	I := func(s string) string { return s }
	if err := s.ApplyMigrations("../../../migrations", I); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return s, nil
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

func (s *PostgresStore) GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]StatResult, error) {
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
	err := s.DB.Select(&results, query,
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

// func (s *PostgresStore) GetEventsByType(eventType string) ([]models.Entry, error) {
// 	var entries []models.Entry
// 	err := s.db.Select(&entries, `
// 		SELECT timestamp, event_type, lab, student, course, comment
// 		FROM entries
// 		WHERE event_type = $1
// 		ORDER BY timestamp ASC
// 	`, eventType)
// 	if err != nil {
// 		return nil, fmt.Errorf("faild to get events by type: %w", err)
// 	}

// 	return entries, nil
// }

func (s *PostgresStore) FetchScoringStats(course, eventType string) ([]models.ScoringResult, error) {
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

	var results []models.ScoringResult
	err := s.DB.Select(&results, query, course, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get scoring stats: %w", err)
	}

	return results, nil
}
