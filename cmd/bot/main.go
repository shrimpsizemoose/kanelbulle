package main

import (
	"flag"

	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/bot"
)

func main() {
	var configPath = flag.String("config", "config.toml", "Path to config file")
	flag.Parse()

	cfg, err := bot.ReadConfig(*configPath)
	if err != nil {
		logger.Error.Fatalf("Failed to create store: %v", err)
	}

	store, err := app.NewStore(cfg.Database.DSN)
	if err != nil {
		logger.Error.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	b, err := bot.New(cfg, store)
	if err != nil {
		logger.Error.Fatalf("Failed to create bot: %v", err)
	}

	logger.Info.Println("Bot intialized succesfully")
	if err := b.Start(); err != nil {
		logger.Error.Fatalf("Bot error: %v", err)
	}
}
