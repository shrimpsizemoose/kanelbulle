package models

type ScoringResult struct {
	Student string `db:"student"`
	Lab     string `db:"lab"`
	Score   int    `db:"score"`
}
