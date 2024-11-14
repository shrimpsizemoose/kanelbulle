package app

import (
	"fmt"
	"strings"

	"github.com/shrimpsizemoose/kanelbulle/internal/store"
	"github.com/shrimpsizemoose/kanelbulle/internal/store/postgres"
	"github.com/shrimpsizemoose/kanelbulle/internal/store/sqlite"
)

func NewStore(dsn string) (store.ScoreStore, error) {
	dbType := store.DBTypeSQLite
	if strings.HasPrefix(dsn, "postgres") {
		dbType = store.DBTypePostgres
	}

	switch dbType {
	case store.DBTypePostgres:
		return postgres.NewPostgresStore(dsn, "./migrations")
	case store.DBTypeSQLite:
		return sqlite.NewSQLiteStore(dsn, "./migrations")
	default:
		return nil, fmt.Errorf("unable to determine database type from DSN: %s", dsn)
	}
}
