package postgres

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type PostgresStore struct {
	store.BaseStore
}

func NewPostgresStore(dsn, migrationsDir string) (*PostgresStore, error) {
	db, err := sqlx.Connect("postgres", dsn)
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

	if err := s.ApplyMigrations(migrationsDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return s, nil
}

func (s *PostgresStore) ApplyMigrations(dir string) error {
	return s.BaseStore.ApplyMigrations(dir, nil)
}

func (s *PostgresStore) GetDetailedStats(course, startEventType, finishEventType string, timestampFormat string, includeHumanDttm bool) ([]store.StatResult, error) {
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
        ORDER BY se.student, se.lab
    `

	var results []store.StatResult
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
