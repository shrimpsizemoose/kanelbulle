package main

import (
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/handlers"
)

func main() {
	var configPath = flag.String("config", "config.toml", "Path to config file")
	flag.Parse()

	service, err := app.NewService(*configPath)
	if err != nil {
		logger.Error.Fatalf("Failed to load config: %v", err)
	}
	defer service.Close()

	if _, err := export.NewGSheetExporter(cfg, store); err != nil {
		log.Fatalf("Failed to initialize Google Sheets exporter: %v", err)
	}


	entryHandler := handlers.NewEntryHandler(service)

	http.HandleFunc("POST /api/v1/{course}/analytics", entryHandler.HandleLabEvent)
	http.HandleFunc("GET /api/v1/{course}/analytics", entryHandler.HandleLabInfo)
	http.HandleFunc("GET /api/v1/{course}/analytics/finish", entryHandler.HandleLabFinishInfo)
	http.HandleFunc("GET /api/v1/{course}/scoring", entryHandler.HandleScoring)

	http.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	http.Handle("/metrics", promhttp.Handler())

	logger.Info.Printf("Starting kanelbulle server on %s", service.Config.Server.Port)
	logger.Debug.Println("Requiring headers:")
	for _, h := range service.Config.API.RequiredHeaders {
		logger.Debug.Printf("  %s: %s", h.Name, h.Value)
	}
	if err := http.ListenAndServe(service.Config.Server.Port, nil); err != nil {
		logger.Error.Fatalf("Kanelbulle server failed: %v", err)
	}
}
