package models

import (
	"time"
)

type TokenInfo struct {
	Token           string    `json:"token"`
	RequestCount    int       `json:"request_count"`
	LastRequestTime time.Time `json:"last_request_dttm_utc"`
	CreatedTime     time.Time `json:"created_dttm_utc"`
}
