package bot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shrimpsizemoose/kanelbulle/internal/scoring"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
	"github.com/shrimpsizemoose/trekker/logger"
)

type Bot struct {
	config *Config
	store  store.ScoreStore
	grader *scoring.Grader
	api    *tgbotapi.BotAPI
	admins map[int64]bool
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

	return &Bot{
		config: config,
		store:  store,
		grader: grader,
		api:    api,
		admins: admins,
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			go b.handleMessage(update.Message)

		case <-sigChan:
			logger.Info.Println("Shutting down bot...")
			return nil
		}
	}
}
