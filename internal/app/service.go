package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shrimpsizemoose/kanelbulle/internal/store"
)

type Service struct {
	Config *Config
	Store  store.ScoreStore
	Auth   *Auth
}

func NewService(configPath string) (*Service, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	store, err := NewStore(config.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to init store: %w", err)
	}

	auth, err := NewAuth(config)
	if err != nil {
		return nil, fmt.Errorf("failed to init auth: %w", err)
	}

	return &Service{
		Config: config,
		Store:  store,
		Auth:   auth,
	}, nil
}

type LabStats struct {
	StartCounts  int64  `json:"start_counts"`
	FirstRun     int64  `json:"first_run"`
	FirstFinish  *int64 `json:"first_finish,omitempty"`
	DeltaSeconds *int64 `json:"delta_first_run_first_finish,omitempty"`
	HumanDttms   *struct {
		FirstRun    string  `json:"first_run"`
		FirstFinish *string `json:"first_finish,omitempty"`
		Delta       string  `json:"delta_first_run_first_finish,omitempty"`
	} `json:"human_dttms,omitempty"`
}

func (s *Service) ValidateAuthAndStudent(r *http.Request, course, student string) error {
	if !s.Config.Server.EnableAuth {
		return nil
	}

	authHeader := r.Header.Get(s.Auth.tokenHeader)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("Invalid authorization header format")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	return s.Auth.ValidateToken(r.Context(), course, student, token)
}

func (s *Service) ValidateHeaders(headers map[string][]string) bool {
	for _, required := range s.Config.API.RequiredHeaders {
		value := headers[http.CanonicalHeaderKey(required.Name)]
		if len(value) == 0 || !strings.EqualFold(value[0], required.Value) {
			return false
		}
	}
	return true
}

func (s *Service) GetScoring(course string) (map[string]map[string]int, error) {
	results, err := s.Store.FetchScoringStats(course, s.Config.Events.Finish)
	if err != nil {
		return nil, fmt.Errorf("failed to get scoring stats: %w", err)
	}

	scores := make(map[string]map[string]int)
	for _, result := range results {
		if scores[result.Student] == nil {
			scores[result.Student] = make(map[string]int)
		}
		scores[result.Student][result.Lab] = result.Score
	}

	return scores, nil
}

func (s *Service) GetDetailedStats(course string, includeHumanDttm bool) (map[string]map[string]*LabStats, error) {
	results, err := s.Store.GetDetailedStats(
		course,
		s.Config.Events.Start,
		s.Config.Events.Finish,
		s.Config.Display.TimestampFormat,
		includeHumanDttm,
	)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]map[string]*LabStats)
	for _, r := range results {
		if stats[r.Student] == nil {
			stats[r.Student] = make(map[string]*LabStats)
		}

		labKey := fmt.Sprintf("%s/%s", r.Course, r.Lab)
		stat := &LabStats{
			StartCounts:  r.StartCount,
			FirstRun:     r.FirstRun,
			FirstFinish:  r.FirstFinish,
			DeltaSeconds: r.DeltaSeconds,
		}

		if includeHumanDttm {
			stat.HumanDttms = &struct {
				FirstRun    string  `json:"first_run"`
				FirstFinish *string `json:"first_finish,omitempty"`
				Delta       string  `json:"delta_first_run_first_finish,omitempty"`
			}{
				FirstRun:    *r.HumanFirstRun,
				FirstFinish: r.HumanFirstFinish,
			}

			if r.DeltaSeconds != nil {
				duration := time.Duration(*r.DeltaSeconds) * time.Second
				stat.HumanDttms.Delta = s.formatDuration(duration)
			}
		}

		stats[r.Student][labKey] = stat
	}

	return stats, nil
}

func (s *Service) formatDuration(d time.Duration) string {
	days := d / (24 * time.Hour)
	d = d % (24 * time.Hour)
	hours := d / time.Hour
	d = d % time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func (s *Service) Close() error {
	var errs []error

	if err := s.Store.Close(); err != nil {
		errs = append(errs, fmt.Errorf("store: %w", err))
	}
	if err := s.Auth.Close(); err != nil {
		errs = append(errs, fmt.Errorf("auth: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors while closing: %v", errs)
	}
	return nil
}
