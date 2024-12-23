
package export

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-co-op/gocron"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	
	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type GSheetExporter struct {
	config        *app.Config
	store         store.ScoreStore
	scheduler     *gocron.Scheduler
	sheetsService *sheets.Service
}

func NewGSheetExporter(config *app.Config, store store.ScoreStore) (*GSheetExporter, error) {
	ctx := context.Background()
	scheduler := gocron.NewScheduler(time.UTC)
	
	for courseName, configs := range config.GSheet {
		for _, cfg := range configs {
			svc, err := sheets.NewService(ctx, option.WithCredentialsFile(cfg.CredentialsPath))
			if err != nil {
				return nil, fmt.Errorf("failed to create sheets service: %w", err)
			}
			
			exporter := &GSheetExporter{
				config:        config,
				store:         store,
				scheduler:     scheduler,
				sheetsService: svc,
			}
			
			_, err = scheduler.Cron(cfg.Schedule).Do(func() {
				if err := exporter.Export(courseName, &cfg); err != nil {
					fmt.Printf("Export failed: %v\n", err)
				}
			})
			if err != nil {
				return nil, fmt.Errorf("failed to schedule export: %w", err)
			}
		}
	}
	
	scheduler.StartAsync()
	return nil, nil
}

func (e *GSheetExporter) Export(courseName string, cfg *app.GSheetConfig) error {
	// Read students first
	readRange := fmt.Sprintf("%s!%s", cfg.SheetName, cfg.StudentsRange)
	resp, err := e.sheetsService.Spreadsheets.Values.Get(cfg.SheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("failed to read students: %w", err)
	}
	
	studentRows := make(map[string]int)
	for i, row := range resp.Values {
		if len(row) > 0 {
			student := row[0].(string)
			studentRows[student] = i + 4 // Assuming start from row 4
		}
	}
	
	// Update completion status for each lab
	for _, lab := range cfg.LabsList {
		for student, row := range studentRows {
			event, err := e.store.GetStudentFinishEvent(courseName, lab, student)
			if err != nil {
				continue
			}
			
			var value interface{} = "✓"
			if cfg.Scoring {
				score, err := e.store.GetLabScore(courseName, lab)
				if err == nil && score != nil {
					value = score.BaseScore
				}
			}
			
			updateRange := fmt.Sprintf("%s!%s", cfg.SheetName, fmt.Sprintf("D%d", row))
			_, err = e.sheetsService.Spreadsheets.Values.Update(cfg.SheetID, updateRange, 
				&sheets.ValueRange{Values: [][]interface{}{{value}}}).ValueInputOption("RAW").Do()
			if err != nil {
				return fmt.Errorf("failed to update cell: %w", err)
			}
		}
	}
	
	// Update timestamp
	emoji := e.config.EmojiVariants[rand.Intn(len(e.config.EmojiVariants))]
	timestamp := fmt.Sprintf("UPD: %s %s", time.Now().Format("2 January 15:04"), emoji)
	
	updateRange := fmt.Sprintf("%s!%s", cfg.SheetName, cfg.TimestampRange)
	_, err = e.sheetsService.Spreadsheets.Values.Update(cfg.SheetID, updateRange,
		&sheets.ValueRange{Values: [][]interface{}{{timestamp}}}).ValueInputOption("RAW").Do()
	
	return err
}
