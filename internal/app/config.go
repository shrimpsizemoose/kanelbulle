package app

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"

	"github.com/shrimpsizemoose/trekker/logger"
)

type HeaderConfig struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

type GSheetConfig struct {
	Schedule        string   `toml:"schedule"`
	Course          string   `toml:"course"`
	SheetID         string   `toml:"sheet_id"`
	CredentialsPath string   `toml:"credentials_path"`
	SheetName       string   `toml:"sheet_name"`
	StudentsRange   string   `toml:"students_range"`
	LabsRange       string   `toml:"labs_range"`
	TimestampRange  string   `toml:"timestamp_range"`
	Scoring         bool     `toml:"scoring"`
	LabsList        []string `toml:"labs_list"`
}

type Config struct {
	Server struct {
		Port string `toml:"port"`
	} `toml:"server"`
	EmojiVariants []string                    `toml:"emoji_variants"`
	GSheet        map[string][]GSheetConfig   `toml:"gsheet"`

	Auth struct {
		Enabled          bool   `toml:"enabled"`
		RedisURL         string `toml:"redis_url"`
		TokenHeader      string `toml:"token_header"`
		TokenKeyTemplate string `toml:"token_key_template"`
	} `toml:"auth"`

	API struct {
		StudentIDHeader string         `toml:"student_id_header"`
		RequiredHeaders []HeaderConfig `toml:"required_headers"`
	} `toml:"api"`

	Database struct {
		DSN string `toml:"dsn"`
	} `toml:"database"`

	Display struct {
		TimestampFormat string `toml:"timestamp_format"`
	} `toml:"display"`

	Scoring struct {
		LateDaysModifiers  map[int]int `toml:"late_days_modifiers"`
		DefaultLatePenalty float64     `toml:"default_late_penalty"`
		MaxLateDays        int         `toml:"max_late_days"`
		ExtraLatePenalty   int         `toml:"extra_late_penalty"`
	} `toml:"scoring"`

	Events struct {
		Start  string `toml:"start"`
		Finish string `toml:"finish"`
		Almost string `toml:"almost"`
	} `toml:"events"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"error reading config file %s\n> Error: %w\n> Content:\n%s",
			path,
			err,
			string(data),
		)
	}

	if config.Server.Port == "" {
		return nil, fmt.Errorf("Server port is not specified in config, use a value like :9999")
	}

	logger.Debug.Printf("Loaded scoring config: %+v", config.Scoring)

	return &config, nil
}
