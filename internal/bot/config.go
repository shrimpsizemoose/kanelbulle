package bot

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Auth struct {
		Enabled          bool   `toml:"enabled"`
		RedisURL         string `toml:"redis_url"`
		TokenHeader      string `toml:"token_header"`
		TokenKeyTemplate string `toml:"token_key_template"`
	} `toml:"auth"`
	Bot struct {
		Token    string  `toml:"token"`
		AdminIDs []int64 `toml:"admin_ids"`
	} `toml:"bot"`
	Database struct {
		DSN string `toml:"dsn"`
	} `toml:"database"`
	Scoring struct {
		LateDaysModifiers  map[int]int `toml:"late_days_modifiers"`
		DefaultLatePenalty float64     `toml:"default_late_penalty"`
		MaxLateDays        int         `toml:"max_late_days"`
		ExtraLatePenalty   int         `toml:"extra_late_penalty"`
	} `toml:"scoring"`
}

func ReadConfig(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load config: %v", err)
	}

	return &cfg, nil
}
