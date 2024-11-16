package models

import "time"

type ChatCourseMapping struct {
	Course          string    `json:"course"`
	Name            string    `json:"name"`
	Comment         string    `json:"comment"`
	AssociationTime time.Time `json:"association_time"`
	RegisteredBy    int64     `json:"registered_by"`
}
