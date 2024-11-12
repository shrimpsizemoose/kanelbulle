package store

type DatabaseType string

const (
	DBTypePostgres DatabaseType = "postgres"
	DBTypeSQLite   DatabaseType = "sqlite"
)

type DBConfig struct {
	DSN  string
	Type DatabaseType
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
