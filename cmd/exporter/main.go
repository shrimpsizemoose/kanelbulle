package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/export"
)

func main() {
	var configPath = flag.String("config", "config.toml", "Path to config file")
	flag.Parse()

	service, err := app.NewService(*configPath)
	if err != nil {
		logger.Error.Fatalf("Failed to load config: %v", err)
	}
	defer service.Close()

	if _, err := export.NewGSheetExporter(service.Config, service.Store); err != nil {
		logger.Error.Fatalf("Failed to initialize Google Sheets exporter: %v", err)
	}

	logger.Info.Println("Садимся экспортить")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info.Println("Закончили экспортить")

}
