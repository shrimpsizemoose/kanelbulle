package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/scoring"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
	"github.com/shrimpsizemoose/trekker/logger"
)

type Bot struct {
	config       *Config
	store        store.ScoreStore
	grader       *scoring.Grader
	api          *tgbotapi.BotAPI
	admins       map[int64]bool
	tokenManager *app.TokenManager
}

func New(config *Config, store store.ScoreStore, grader *scoring.Grader) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(config.Bot.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	admins := make(map[int64]bool)
	for _, id := range config.Bot.AdminIDs {
		admins[id] = true
	}

	redisOpt, err := redis.ParseURL(config.Auth.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	redisClient := redis.NewClient(redisOpt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	tokenManager := app.NewTokenManager(redisClient)

	return &Bot{
		config:       config,
		store:        store,
		grader:       grader,
		api:          api,
		admins:       admins,
		tokenManager: tokenManager,
	}, nil
}

func (b *Bot) Start() error {
	logger.Info.Println("Starting bot...")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	defer func() {
		logger.Info.Println("Shutting down bot...")
		if err := b.Close(); err != nil {
			logger.Error.Printf("Error during shutdown: %v", err)
		}
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			go b.handleMessage(update.Message)

		case <-stop:
			logger.Info.Println("Received shutdown signal")
			return nil
		}
	}
}

func (b *Bot) Close() error {
	var errs []error

	if b.tokenManager != nil {
		if err := b.tokenManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("token manager: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors while closing bot: %v", errs)
	}

	return nil
}
