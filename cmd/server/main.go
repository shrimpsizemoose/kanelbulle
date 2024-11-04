package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/handlers"
)

func main() {
	service, err := app.NewService("config.toml")
	if err != nil {
		logger.Error.Fatalf("Failed to load config: %v", err)
	}
	defer service.Close()

	if err := service.Store.ApplyMigrations("./migrations"); err != nil {
		logger.Error.Fatalf("Failed to apply migrations: %v", err)
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
