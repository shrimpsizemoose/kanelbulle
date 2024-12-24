package export

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/store"
	"github.com/shrimpsizemoose/trekker/logger"
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
			logger.Info.Printf("Registering gsheet exporter[%s] with schedule %s", cfg.Course, cfg.Schedule)

			_, err = scheduler.Cron(cfg.Schedule).Do(func() {
				if err := exporter.Export(courseName, &cfg); err != nil {
					logger.Debug.Printf("Gsheet export for %s failed: %v", cfg.Course, err)
				} else {
					logger.Debug.Printf("Gsheet export[%s] succesfull", cfg.Course)

				}
			})
			if err != nil {
				return nil, fmt.Errorf("failed to schedule export for [%s] on schedule '%s': %w", cfg.Course, cfg.Schedule, err)
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
	startRow := 1
	if parts := strings.Split(cfg.StudentsRange, ":"); len(parts) > 0 {
		if row := strings.TrimLeft(parts[0], "ABCDEFGHIJKLMNOPQRSTUVWXYZ"); row != "" {
			if num, err := strconv.Atoi(row); err == nil {
				startRow = num
			}
		}
	}
	for i, row := range resp.Values {
		if len(row) > 0 {
			student := row[0].(string)
			studentRows[student] = startRow + i // Assuming start from row 4
		}
	}
	// Prepare for batc hupdate
	var valueRanges []*sheets.ValueRange

	labRange := cfg.LabsRange
	startCol := labRange[0:1]
	labColOffset := int(byte(startCol[0]) - byte('A'))

	// Update completion status for each lab
	for labIdx, lab := range cfg.LabsList {
		col := string(byte('A' + labColOffset + labIdx))

		for student, row := range studentRows {
			event, err := e.store.GetStudentFinishEvent(courseName, lab, student)
			if err != nil {
				continue
			}

			var value interface{} = ""
			if !cfg.Scoring {
				if event == nil {
					value = ""
				} else {
					value = "✓"
				}
			} else {
				score, err := e.store.GetLabScore(courseName, lab)
				if err == nil && score != nil {
					value = score.BaseScore
				}
			}

			updateRange := fmt.Sprintf("%s!%s%d", cfg.SheetName, col, row)
			valueRanges = append(valueRanges, &sheets.ValueRange{
				Range:  updateRange,
				Values: [][]interface{}{{value}},
			})
		}
	}

	// Update timestamp
	emoji := e.config.RandomEmoji()
	timestamp := fmt.Sprintf("UPD: %s", time.Now().Format("2 January 15:04"))
	timestampRange := fmt.Sprintf("%s!%s", cfg.SheetName, cfg.TimestampRange)
	valueRanges = append(valueRanges, &sheets.ValueRange{
		Range:  timestampRange,
		Values: [][]interface{}{{emoji, timestamp}},
	})

	if len(valueRanges) > 0 {
		batchUpdate := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "RAW",
			Data:             valueRanges,
		}
		_, err = e.sheetsService.Spreadsheets.Values.BatchUpdate(cfg.SheetID, batchUpdate).Do()
		if err != nil {
			return fmt.Errorf("failed to batch update cell: %w", err)
		}
	}

	return nil
}
