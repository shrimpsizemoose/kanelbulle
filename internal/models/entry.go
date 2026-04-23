package models

import (
	"encoding/json"
	"regexp"
)

var studentRegex = regexp.MustCompile(`^[\w-]+\..+$`)

type Entry struct {
	Timestamp int64  `db:"timestamp" json:"timestamp"`
	EventType string `db:"event_type" json:"event_type"`
	Lab       string `db:"lab" json:"lab" validate:"required,max=3"`
	Student   string `db:"student" json:"student" validate:"required,regexp=^[\\w-]+\\..+$"`
	Course    string `db:"course" json:"course" validate:"required,max=32"`
	Comment   string `db:"comment" json:"comment"`
}

type LabScore struct {
	Deadline  int64  `db:"deadline" json:"deadline"`
	Lab       string `db":lab" json:"lab" validate:"required,max=3"`
	BaseScore int    `db:"base_score" json:"base_score"`
	Course    string `db:"course" validate:"required,max=32"`
}

func (e *Entry) Validate() error {
	validate := newValidator()
	return validate.Struct(e)
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{
		e.Timestamp,
		e.EventType,
		e.Lab,
		e.Student,
		e.Course,
		e.Comment,
	})
}

func (l *LabScore) Validate() error {
	validate := newValidator()
	return validate.Struct(l)
}
