package handlers

import (
	"bytes"
	"io"
	"time"

	"encoding/json"
	"net/http"

	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/app"
	"github.com/shrimpsizemoose/kanelbulle/internal/metrics"
	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

type EntryHandler struct {
	service *app.Service
}

func NewEntryHandler(service *app.Service) *EntryHandler {
	return &EntryHandler{
		service: service,
	}
}

func (h *EntryHandler) HandleLabEvent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.APIRequestDuration.WithLabelValues(
			r.URL.Path,
			r.Method,
			"200",
		).Observe(duration)
	}()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.service.ValidateHeaders(r.Header) {
		http.Error(w, "these are not the droids you are looking for", http.StatusForbidden)
		return
	}

	course := r.PathValue("course")
	if course == "" {
		logger.Error.Printf("Failed to extract course from path: %s", r.URL.Path)
		http.Error(w, "Invalid course", http.StatusBadRequest)
		return
	}

	student := r.Header.Get(h.service.Config.API.StudentIDHeader)
	if student == "" {
		http.Error(w, "Invalid student id specified", http.StatusUnauthorized)
		return
	}

	if err := h.service.ValidateAuthAndStudent(r, course, student); err != nil {
		logger.Error.Printf("Auth failed: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	logger.Debug.Printf("Received request body: %s", string(body))

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var entry models.Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	entry.Course = r.PathValue("course")

	if err := h.service.Store.CreateEntry(&entry); err != nil {
		http.Error(w, "Failed to save entry", http.StatusInternalServerError)
		return
	}

	metrics.EventsTotal.WithLabelValues(
		entry.Course,
		entry.Lab,
		entry.EventType,
	).Inc()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *EntryHandler) HandleLabInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.service.ValidateHeaders(r.Header) {
		http.Error(w, "these are not the droids you are looking for", http.StatusNotFound)
		return
	}

	course := r.PathValue("course")
	if course == "" {
		logger.Error.Printf("Failed to extract course from path: %s", r.URL.Path)
		http.Error(w, "Invalid course", http.StatusBadRequest)
		return
	}

	entries, err := h.service.Store.ListEntries(course)
	if err != nil {
		logger.Error.Printf("ERROR: %v", err)
		http.Error(w, "Failed to fetch entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"rows": entries,
	}); err != nil {
		logger.Debug.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *EntryHandler) HandleLabFinishInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.service.ValidateHeaders(r.Header) {
		http.Error(w, "these are not the droids you are looking for", http.StatusNotFound)
		return
	}

	includeHumanDttm := r.URL.Query().Get("human_dttm") == "true"
	course := r.PathValue("course")
	if course == "" {
		logger.Error.Printf("Failed to extract course from path: %s", r.URL.Path)
		http.Error(w, "Invalid course", http.StatusBadRequest)
		return
	}

	stats, err := h.service.GetDetailedStats(course, includeHumanDttm)
	if err != nil {
		logger.Error.Printf("Failed to fetch stats: %v", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": stats,
	}); err != nil {
		logger.Error.Printf("Failed to encode stats: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *EntryHandler) HandleScoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.service.ValidateHeaders(r.Header) {
		http.Error(w, "these are not the droids you are looking for", http.StatusNotFound)
		return
	}

	course := r.PathValue("course")
	if course == "" {
		logger.Error.Printf("Failed to extract course from path: %s", r.URL.Path)
		http.Error(w, "Invalid course", http.StatusBadRequest)
		return
	}

	scores, err := h.service.GetScoring(course)
	if err != nil {
		logger.Error.Printf("Failed to get scoring for course %s: %v", course, err)
		http.Error(w, "Failed to fetch scoring", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": scores,
	}); err != nil {
		logger.Error.Printf("Failed to encode scoring response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
