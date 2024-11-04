package models

type ScoreOverride struct {
	Student string `db:"student" json:"student" validate:"required,regexp=^[\\w-]+\\..+$"`
	Lab     string `db:"lab" json:"lab" validate:"required,max=3"`
	Score   int    `db:"score" json:"score"`
	Course  string `db:"course" json:"course" validate:"required,max=6"`
	Reason  string `db:"reason" json:"reason"`
}

// unique_together should be handled on DB level:
/*
CREATE TABLE score_overrides (
    student TEXT NOT NULL,
    lab VARCHAR(3) NOT NULL,
    score INTEGER NOT NULL,
    course VARCHAR(6) NOT NULL,
    reason TEXT,
    CONSTRAINT score_overrides_pkey PRIMARY KEY (course, lab, student)
);
*/
