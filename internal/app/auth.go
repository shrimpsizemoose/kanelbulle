// internal/app/auth.go
package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/shrimpsizemoose/trekker/logger"
)

// type TokenInfo struct {
// 	Token           string `json:"token"`
// 	RequestCount    int    `json:"request_count"`
// 	LastRequestDttm string `json:"last_request_dttm_utc"`
// 	CreatedDttm     string `json:"created_dttm_utc"`
// }

type Auth struct {
	enabled     bool
	redis       *redis.Client
	keyTemplate string
	tokenHeader string
}

func NewAuth(config *Config) (*Auth, error) {
	if !config.Server.EnableAuth {
		return &Auth{enabled: false}, nil
	}

	opt, err := redis.ParseURL(config.Auth.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Auth{
		enabled:     true,
		redis:       client,
		keyTemplate: config.Auth.TokenKeyTemplate,
		tokenHeader: config.Auth.TokenHeader,
	}, nil
}

func (a *Auth) Close() error {
	if a.redis != nil {
		return a.redis.Close()
	}
	return nil
}

func (a *Auth) ValidateToken(ctx context.Context, course, student, token string) error {
	if !a.enabled {
		return nil
	}

	key := strings.NewReplacer(
		"{course}", course,
		"{student}", student,
	).Replace(a.keyTemplate)

	fields, err := a.redis.HGetAll(ctx, key).Result()
	if err == redis.Nil {
		logger.Debug.Printf("Token not found for key: %s", key)
		return fmt.Errorf("token not found")
	}
	if err != nil {
		logger.Debug.Printf("Redis error: %v", err)
		return fmt.Errorf("redis error: %w", err)
	}

	if fields["token"] != token {
		logger.Debug.Printf(
			"Token mismatch for course/student/token=%s/%s/%s and what's found in %s",
			course,
			student,
			token,
			key,
		)
		return fmt.Errorf("invalid token")
	}

	return nil
}
