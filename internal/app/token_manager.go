package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

const (
	timeFormat       = "2006-01-02 15:04:05"
	authKeyTpl       = "auth:%s:%s" // auth:${course}:${student}
	lookupKeyTpl     = "lookup:%s"  // lookup:${course}
	chatCourseKeyTpl = "chat:%d"    // chat:${chatID}
	tokenPrefix      = "sk-knlbll-"
)

type TokenManager struct {
	redis *redis.Client
}

func NewTokenManager(redis *redis.Client) *TokenManager {
	return &TokenManager{redis: redis}
}

func generateToken() (string, error) {
	randomBytes := make([]byte, 12)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed ot generate random bytes: %w", err)
	}

	return tokenPrefix + hex.EncodeToString(randomBytes), nil
}

func (tm *TokenManager) FetchOrCreateStudentToken(ctx context.Context, course, student string) (*models.TokenInfo, bool, error) {
	key := fmt.Sprintf(authKeyTpl, course, student)

	token, err := tm.redis.HGet(ctx, key, "token").Result()
	if err != nil && err != redis.Nil {
		return nil, false, fmt.Errorf("failed to check token: %w", err)
	}

	now := time.Now().UTC()
	isNewToken := false

	if err == redis.Nil {
		token, err = generateToken()
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate token: %w", err)
		}

		pipe := tm.redis.Pipeline()
		pipe.HSet(ctx, key, map[string]interface{}{
			"token":                 token,
			"request_count":         1,
			"last_request_dttm_utc": now.Format(timeFormat),
			"created_dttm_utc":      now.Format(timeFormat),
		})

		if _, err := pipe.Exec(ctx); err != nil {
			return nil, false, fmt.Errorf("failed to create token: %w", err)
		}

		isNewToken = true
	} else {
		pipe := tm.redis.Pipeline()
		pipe.HIncrBy(ctx, key, "request_count", 1)
		pipe.HSet(ctx, key, "last_request_dttm_utc", now.Format(timeFormat))

		if _, err := pipe.Exec(ctx); err != nil {
			return nil, false, fmt.Errorf("failed to update token stats: %w", err)
		}
	}

	values, err := tm.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get token info: %w", err)
	}

	lastReqTime, _ := time.Parse(timeFormat, values["last_request_dttm_utc"])
	createdTime, _ := time.Parse(timeFormat, values["created_dttm_utc"])
	reqCount, _ := strconv.Atoi(values["request_count"])

	return &models.TokenInfo{
		Token:           values["token"],
		RequestCount:    reqCount,
		LastRequestTime: lastReqTime,
		CreatedTime:     createdTime,
	}, isNewToken, nil
}

func (tm *TokenManager) SaveStudentTelegramMapping(ctx context.Context, course, tgUsername, studentID string) error {
	key := fmt.Sprintf(lookupKeyTpl, course)
	return tm.redis.HSet(ctx, key, tgUsername, studentID).Err()
}

func (tm *TokenManager) FetchStudentIDByTelegram(ctx context.Context, course, tgUsername string) (string, error) {
	key := fmt.Sprintf(lookupKeyTpl, course)
	studentID, err := tm.redis.HGet(ctx, key, tgUsername).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("no mapping found for telegram user %s in course %s", tgUsername, course)
	}
	return studentID, err
}

func (tm *TokenManager) FetchCourseMappings(ctx context.Context, course string) (map[string]string, error) {
	key := fmt.Sprintf(lookupKeyTpl, course)
	return tm.redis.HGetAll(ctx, key).Result()
}

func (tm *TokenManager) AssociateChatWithCourse(ctx context.Context, chatID int64, mapping *models.ChatCourseMapping) error {
	key := fmt.Sprintf(chatCourseKeyTpl, chatID)
	return tm.redis.HSet(ctx, key, map[string]interface{}{
		"course":              mapping.Course,
		"name":                mapping.Name,
		"comment":             mapping.Comment,
		"associated_dttm_utc": mapping.AssociationTime.Format(timeFormat),
		"registered_by":       mapping.RegisteredBy,
	}).Err()
}

func (tm *TokenManager) FetchCourseMappingByChatID(ctx context.Context, chatID int64) (*models.ChatCourseMapping, error) {
	key := fmt.Sprintf(chatCourseKeyTpl, chatID)

	values, err := tm.redis.HGetAll(ctx, key).Result()
	if err == redis.Nil || len(values) == 0 {
		return nil, fmt.Errorf("no course mapping found for chat %d", chatID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat course mapping found for chat %d", chatID)
	}

	associationTime, _ := time.Parse(timeFormat, values["association_dttm_utc"])
	registeredBy, _ := strconv.ParseInt(values["registered_by"], 10, 64)

	return &models.ChatCourseMapping{
		Course:          values["course"],
		Name:            values["name"],
		Comment:         values["comment"],
		AssociationTime: associationTime,
		RegisteredBy:    registeredBy,
	}, nil
}

func (tm *TokenManager) Close() error {
	if tm.redis != nil {
		return tm.redis.Close()
	}
	return nil
}

func (tm *TokenManager) FetchAllChatMappings(ctx context.Context) (map[string]*models.ChatCourseMapping, error) {
	// FIXME: scans are expensive
	pattern := fmt.Sprintf("%s*", chatCourseKeyTpl)

	iter := tm.redis.Scan(ctx, 0, pattern, 0).Iterator()

	mappings := make(map[string]*models.ChatCourseMapping)

	for iter.Next(ctx) {
		key := iter.Val()
		chatID := strings.TrimPrefix(key, chatCourseKeyTpl)

		values, err := tm.redis.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}

		associationTime, _ := time.Parse(timeFormat, values["association_dttm_utc"])
		registeredBy, _ := strconv.ParseInt(values["registered_by"], 10, 64)

		mappings[chatID] = &models.ChatCourseMapping{
			Course:          values["course"],
			Name:            values["name"],
			Comment:         values["comment"],
			AssociationTime: associationTime,
			RegisteredBy:    registeredBy,
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to fetch chat mappings: %w", err)
	}

	return mappings, nil

}
