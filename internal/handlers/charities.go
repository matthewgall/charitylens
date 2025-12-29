package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"charitylens/internal/config"
	apperrors "charitylens/internal/errors"
	"charitylens/internal/models"
	"charitylens/internal/scoring"
	"charitylens/internal/sync"

	"github.com/go-chi/chi/v5"
)

type CharityHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

func NewCharityHandler(db *sql.DB, cfg *config.Config) *CharityHandler {
	return &CharityHandler{DB: db, Cfg: cfg}
}

// debugLog logs a message only if debug mode is enabled
func (h *CharityHandler) debugLog(format string, args ...any) {
	if h.Cfg.Debug {
		log.Printf(format, args...)
	}
}

// writeJSON is a helper to write JSON responses
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError is a helper to write error responses
func writeError(w http.ResponseWriter, err error) {
	var status int
	var message string

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		status = http.StatusNotFound
		message = "Resource not found"
	case errors.Is(err, apperrors.ErrInvalidInput):
		status = http.StatusBadRequest
		message = "Invalid input"
	case errors.Is(err, apperrors.ErrUnauthorized):
		status = http.StatusUnauthorized
		message = "Unauthorized"
	default:
		status = http.StatusInternalServerError
		message = "Internal server error"
		log.Printf("Internal error: %v", err)
	}

	writeJSON(w, status, map[string]string{"error": message})
}

func (h *CharityHandler) SearchCharities(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // Default page size
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Basic input validation
	if len(query) > 200 {
		http.Error(w, "Query too long (max 200 characters)", http.StatusBadRequest)
		return
	}

	log.Printf("Search request for query: '%s' (length: %d, limit: %d, offset: %d)", query, len(query), limit, offset)

	// Try searching by number first if query looks like a number
	if charityNum, err := strconv.Atoi(query); err == nil {
		h.debugLog("Searching by number: %d", charityNum)
		charities := h.searchByNumber(charityNum, limit)
		h.debugLog("Number search returned %d results", len(charities))

		response := map[string]any{
			"results": charities,
			"total":   len(charities),
			"limit":   limit,
			"offset":  offset,
		}
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Search by name
	h.debugLog("Searching by name: %s", query)
	charities, total := h.searchByName(query, limit, offset)
	h.debugLog("Name search returned %d results (out of %d total)", len(charities), total)

	response := map[string]any{
		"results":  charities,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"has_more": offset+len(charities) < total,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *CharityHandler) searchByNumber(charityNum int, limit int) []models.Charity {
	h.debugLog("Searching for charity number: %d", charityNum)

	// First check if we already have this charity in the database with score (main charity only, exclude removed)
	var existing models.Charity
	var overallScore float64
	var address, website, email, whatTheCharityDoes sql.NullString
	err := h.DB.QueryRow(`
		SELECT c.registered_number, c.name, c.status, c.address, c.website, c.email, 
		       c.what_the_charity_does, COALESCE(s.overall_score, 0) as overall_score
		FROM charities c
		LEFT JOIN charity_scores s ON c.registered_number = s.charity_number
		WHERE c.registered_number = ? 
		  AND c.linked_charity_number = 0
		  AND c.status NOT IN ('Removed', 'RM')
	`, charityNum).Scan(
		&existing.RegisteredNumber, &existing.Name, &existing.Status,
		&address, &website, &email, &whatTheCharityDoes,
		&overallScore,
	)

	if err == nil {
		// Convert NullString to string
		if address.Valid {
			existing.Address = address.String
		}
		if website.Valid {
			existing.Website = website.String
		}
		if email.Valid {
			existing.Email = email.String
		}
		if whatTheCharityDoes.Valid {
			existing.WhatTheCharityDoes = whatTheCharityDoes.String
		}

		h.debugLog("Found charity %d in database: %s (score: %.1f)", charityNum, existing.Name, overallScore)
		existing.OverallScore = overallScore
		// Charity exists in database
		return []models.Charity{existing}
	}

	// In offline mode, don't try to search API
	if h.Cfg.OfflineMode {
		h.debugLog("Charity %d not in database (offline mode - no API search)", charityNum)
		return []models.Charity{}
	}

	h.debugLog("Charity %d not in database, searching API", charityNum)

	// Charity not in database, search via API
	results, err := sync.SearchCharitiesByNumber(h.Cfg, strconv.Itoa(charityNum))
	if err != nil {
		log.Printf("Error searching by number: %v", err)
		return []models.Charity{}
	}

	h.debugLog("API search returned %d results for number %d", len(results), charityNum)
	return h.processSearchResults(results, limit)
}

func (h *CharityHandler) searchByName(query string, limit int, offset int) ([]models.Charity, int) {
	h.debugLog("Searching for charity name: %s (limit=%d, offset=%d)", query, limit, offset)

	// First, get total count of matching charities in database (main charities only, exclude removed)
	var totalInDB int
	h.DB.QueryRow(`
		SELECT COUNT(*) FROM charities
		WHERE (LOWER(name) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?))
		  AND linked_charity_number = 0
		  AND status NOT IN ('Removed', 'RM')
	`, "%"+query+"%", query+"%").Scan(&totalInDB)

	h.debugLog("Total charities in database matching '%s': %d", query, totalInDB)

	// Decide whether to search the API to discover new charities:
	// 1. If we have < 10 results (need more data)
	// 2. Or periodically for popular searches (7+ days old or 10% random)
	// But skip API search entirely if in offline mode
	shouldSearchAPI := !h.Cfg.OfflineMode && totalInDB < 10 && len(query) >= 3
	searchInBackground := false

	// For popular searches (10+ results), periodically refresh to find newly registered charities
	if !h.Cfg.OfflineMode && !shouldSearchAPI && totalInDB >= 10 && len(query) >= 3 {
		var lastRefresh time.Time
		err := h.DB.QueryRow(`
			SELECT last_searched FROM search_cache 
			WHERE query = ? AND search_type = 'name'
		`, query).Scan(&lastRefresh)

		hoursSinceRefresh := time.Since(lastRefresh).Hours()
		randomRefresh := rand.Float64() < 0.10 // 10% chance

		if err == sql.ErrNoRows || hoursSinceRefresh > 168 || randomRefresh {
			shouldSearchAPI = true
			searchInBackground = true // Don't block user response for refreshes
			if randomRefresh {
				log.Printf("Random refresh triggered for popular search '%s' (10%% chance)", query)
			} else if err == sql.ErrNoRows {
				log.Printf("First-time API search for '%s'", query)
				searchInBackground = false // Wait for first search to complete
			} else {
				log.Printf("Periodic refresh for '%s' (last searched %.1f hours ago)", query, hoursSinceRefresh)
			}
		}
	}

	// If we should search API, fetch and store ALL results
	if shouldSearchAPI {
		h.debugLog("Searching API for '%s' (totalInDB=%d, query_length=%d, background=%v)", query, totalInDB, len(query), searchInBackground)

		var apiCharities []models.Charity

		syncFunc := func() []models.Charity {
			results, err := sync.SearchCharitiesByName(h.Cfg, query)
			if err != nil {
				log.Printf("API search error for '%s': %v", query, err)
				return nil
			}

			log.Printf("API search returned %d results for '%s'", len(results), query)

			// Update search cache
			h.DB.Exec(`
				INSERT INTO search_cache (query, search_type, last_searched, result_count)
				VALUES (?, 'name', ?, ?)
				ON CONFLICT(query, search_type) DO UPDATE SET
					last_searched = excluded.last_searched,
					result_count = excluded.result_count
			`, query, time.Now(), len(results))

			// Process ALL results to get charity objects
			// Note: processSearchResults handles background sync and score calculation internally
			allCharities := h.processSearchResults(results, len(results))
			log.Printf("Processed %d charities from API (out of %d total)", len(allCharities), len(results))

			return allCharities
		}

		if searchInBackground {
			// Background refresh for popular searches - don't wait
			h.debugLog("Running API search in background")
			go syncFunc()
		} else {
			// Synchronous for first-time searches - wait and use results
			apiCharities = syncFunc()
			if apiCharities != nil && len(apiCharities) > 0 {
				// Return paginated slice of API results
				start := offset
				end := offset + limit
				if start > len(apiCharities) {
					start = len(apiCharities)
				}
				if end > len(apiCharities) {
					end = len(apiCharities)
				}

				paginatedResults := apiCharities[start:end]
				h.debugLog("Returning %d charities from API results (offset=%d, total=%d)", len(paginatedResults), offset, len(apiCharities))
				return paginatedResults, len(apiCharities)
			}
		}
	}

	// Return paginated results from database (for existing data or if API failed, main charities only, exclude removed)
	rows, err := h.DB.Query(`
		SELECT c.registered_number, c.name, c.status, c.address, c.website, c.email, 
		       c.what_the_charity_does, COALESCE(s.overall_score, 0) as overall_score
		FROM charities c
		LEFT JOIN charity_scores s ON c.registered_number = s.charity_number
		WHERE (LOWER(c.name) LIKE LOWER(?) OR LOWER(c.name) LIKE LOWER(?))
		  AND c.linked_charity_number = 0
		  AND c.status NOT IN ('Removed', 'RM')
		ORDER BY c.name
		LIMIT ? OFFSET ?
	`, "%"+query+"%", query+"%", limit, offset)

	var charities []models.Charity
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var charity models.Charity
			var overallScore float64
			var address, website, email, whatTheCharityDoes sql.NullString
			err := rows.Scan(
				&charity.RegisteredNumber, &charity.Name, &charity.Status,
				&address, &website, &email, &whatTheCharityDoes,
				&overallScore,
			)
			if err == nil {
				// Convert NullString to string
				if address.Valid {
					charity.Address = address.String
				}
				if website.Valid {
					charity.Website = website.String
				}
				if email.Valid {
					charity.Email = email.String
				}
				if whatTheCharityDoes.Valid {
					charity.WhatTheCharityDoes = whatTheCharityDoes.String
				}

				charity.OverallScore = overallScore
				charities = append(charities, charity)
			}
		}
	}

	// Recalculate total (main charities only, exclude removed)
	h.DB.QueryRow(`
		SELECT COUNT(*) FROM charities
		WHERE (LOWER(name) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?))
		  AND linked_charity_number = 0
		  AND status NOT IN ('Removed', 'RM')
	`, "%"+query+"%", query+"%").Scan(&totalInDB)

	h.debugLog("Returning %d charities from database (offset=%d, total=%d)", len(charities), offset, totalInDB)
	return charities, totalInDB
}

// queueScoreCalculations triggers background score calculation for charities without scores
func (h *CharityHandler) queueScoreCalculations(charities []models.Charity) {
	for _, charity := range charities {
		if charity.RegisteredNumber > 0 && charity.OverallScore == 0 {
			// Check if charity has a score
			var hasScore bool
			h.DB.QueryRow("SELECT 1 FROM charity_scores WHERE charity_number = ?", charity.RegisteredNumber).Scan(&hasScore)

			if !hasScore {
				h.debugLog("Queuing score calculation for charity %d", charity.RegisteredNumber)
				go func(charityNum int) {
					if score, err := scoring.CalculateScore(h.DB, charityNum); err == nil {
						h.debugLog("Score calculated for charity %d: %.2f", charityNum, score.OverallScore)
					} else {
						log.Printf("Score calculation failed for charity %d: %v", charityNum, err)
					}
				}(charity.RegisteredNumber)
			}
		}
	}
}

// mergeCharityResults combines database and API results, deduplicating by registered number
func (h *CharityHandler) mergeCharityResults(db []models.Charity, api []models.Charity) []models.Charity {
	seen := make(map[int]bool)
	var merged []models.Charity

	// Add all database results first
	for _, charity := range db {
		if !seen[charity.RegisteredNumber] {
			merged = append(merged, charity)
			seen[charity.RegisteredNumber] = true
		}
	}

	// Add API results that aren't already in the list
	for _, charity := range api {
		if !seen[charity.RegisteredNumber] {
			merged = append(merged, charity)
			seen[charity.RegisteredNumber] = true
		}
	}

	return merged
}

func (h *CharityHandler) GetCharity(w http.ResponseWriter, r *http.Request) {
	numberStr := chi.URLParam(r, "number")
	number, err := strconv.Atoi(numberStr)
	if err != nil || number < 1 || number > 9999999999 {
		http.Error(w, "Invalid charity number", http.StatusBadRequest)
		return
	}

	// Get charity details (main charity only, linked_charity_number = 0)
	var charity models.Charity
	var website, email, address, whatTheCharityDoes sql.NullString
	err = h.DB.QueryRow(`
		SELECT registered_number, name, status, date_registered, address, website,
		       email, what_the_charity_does
		FROM charities WHERE registered_number = ? AND linked_charity_number = 0
	`, number).Scan(
		&charity.RegisteredNumber, &charity.Name, &charity.Status,
		&charity.DateRegistered, &address, &website,
		&email, &whatTheCharityDoes,
	)

	// Convert NullString to string
	if address.Valid {
		charity.Address = address.String
	}
	if website.Valid {
		charity.Website = website.String
	}
	if email.Valid {
		charity.Email = email.String
	}
	if whatTheCharityDoes.Valid {
		charity.WhatTheCharityDoes = whatTheCharityDoes.String
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Charity not found"})
			return
		}
		log.Printf("Database error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Calculate score (don't cache in offline mode - always fresh)
	score, err := scoring.CalculateScore(h.DB, number, !h.Cfg.OfflineMode)
	scoreError := ""
	if err != nil {
		// If error, continue without score but log it
		log.Printf("Error calculating score for charity %d: %v", number, err)
		scoreError = err.Error()
		score = models.CharityScore{
			CharityNumber: number,
		}
	}

	type Response struct {
		Charity    models.Charity      `json:"charity"`
		Score      models.CharityScore `json:"score"`
		ScoreError string              `json:"score_error,omitempty"`
	}

	response := Response{
		Charity:    charity,
		Score:      score,
		ScoreError: scoreError,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CharityHandler) CompareCharities(w http.ResponseWriter, r *http.Request) {
	numbersStr := strings.TrimSpace(r.URL.Query().Get("numbers"))
	if numbersStr == "" {
		http.Error(w, "Query parameter 'numbers' is required", http.StatusBadRequest)
		return
	}

	numberStrs := strings.Split(numbersStr, ",")
	if len(numberStrs) > 5 {
		http.Error(w, "Cannot compare more than 5 charities", http.StatusBadRequest)
		return
	}
	if len(numberStrs) < 2 {
		http.Error(w, "Please provide at least 2 charity numbers to compare", http.StatusBadRequest)
		return
	}

	var charities []models.Charity
	var scores []models.CharityScore

	for _, numStr := range numberStrs {
		numStr = strings.TrimSpace(numStr)
		number, err := strconv.Atoi(numStr)
		if err != nil {
			continue // skip invalid
		}

		var charity models.Charity
		var address, website sql.NullString
		err = h.DB.QueryRow(`
			SELECT registered_number, name, status, address, website
			FROM charities WHERE registered_number = ? AND linked_charity_number = 0
		`, number).Scan(&charity.RegisteredNumber, &charity.Name, &charity.Status, &address, &website)
		if err == nil {
			// Convert NullString to string
			if address.Valid {
				charity.Address = address.String
			}
			if website.Valid {
				charity.Website = website.String
			}

			charities = append(charities, charity)

			var score models.CharityScore
			h.DB.QueryRow(`
				SELECT overall_score, efficiency_score, financial_health_score,
				       transparency_score, governance_score
				FROM charity_scores WHERE charity_number = ?
			`, number).Scan(&score.OverallScore, &score.EfficiencyScore, &score.FinancialHealthScore,
				&score.TransparencyScore, &score.GovernanceScore)
			scores = append(scores, score)
		}
	}

	response := struct {
		Charities []models.Charity      `json:"charities"`
		Scores    []models.CharityScore `json:"scores"`
	}{
		Charities: charities,
		Scores:    scores,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *CharityHandler) SyncData(w http.ResponseWriter, r *http.Request) {
	// Reject sync requests in offline mode
	if h.Cfg.OfflineMode {
		http.Error(w, "Sync is disabled in offline mode", http.StatusForbidden)
		return
	}

	// Check for admin authentication
	if h.Cfg.AdminAPIKey != "" {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + h.Cfg.AdminAPIKey
		if authHeader != expectedAuth {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if err := sync.SyncCharities(h.Cfg, h.DB); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sync completed"})
}

func (h *CharityHandler) processSearchResults(results []map[string]any, limit int) []models.Charity {
	h.debugLog("PROCESSING SEARCH RESULTS: %d total", len(results))
	var charities []models.Charity
	rmCount := 0

	for i, result := range results {
		if i >= limit {
			break
		}

		charity := models.Charity{
			LastUpdated: time.Now(),
		}

		// Extract charity data from search result
		h.debugLog("Search result raw data: %+v", result)

		// Try different possible field names for registration number
		// Note: The search API sometimes returns organisation_number which might differ from reg_charity_number
		possibleRegFields := []string{"registered_charity_number", "reg_charity_number", "charity_registration_number"}
		for _, field := range possibleRegFields {
			if rn, ok := result[field]; ok && rn != nil {
				h.debugLog("Found reg number in field %s: %v (type: %T)", field, rn, rn)
				switch v := rn.(type) {
				case string:
					if parsed, err := strconv.Atoi(v); err == nil {
						charity.RegisteredNumber = parsed
					}
				case float64:
					charity.RegisteredNumber = int(v)
				case int:
					charity.RegisteredNumber = v
				}
				if charity.RegisteredNumber != 0 {
					break // Found a valid number
				}
			}
		}

		// If we still don't have a number, try organisation_number as fallback
		if charity.RegisteredNumber == 0 {
			if orgNum, ok := result["organisation_number"]; ok && orgNum != nil {
				h.debugLog("Warning: Using organisation_number as fallback: %v", orgNum)
				switch v := orgNum.(type) {
				case string:
					if parsed, err := strconv.Atoi(v); err == nil {
						charity.RegisteredNumber = parsed
					}
				case float64:
					charity.RegisteredNumber = int(v)
				case int:
					charity.RegisteredNumber = v
				}
			}
		}
		if name, ok := result["charity_name"].(string); ok {
			charity.Name = name
		}
		if status, ok := result["reg_status"].(string); ok {
			charity.Status = status
		}

		// Skip removed charities (status RM with non-null date_of_removal)
		if charity.Status == "RM" {
			removalDate, ok := result["date_of_removal"]
			h.debugLog("RM charity check: %s, has field: %v, value: %v, type: %T", charity.Name, ok, removalDate, removalDate)
			if ok {
				if str, isString := removalDate.(string); isString && str != "" {
					h.debugLog("Skipping removed charity: %s (removed: %s)", charity.Name, str)
					rmCount++
					continue // Skip this charity
				}
			}
		}

		h.debugLog("Processed search result: reg_num=%d, name=%s, status=%s", charity.RegisteredNumber, charity.Name, charity.Status)

		// Trigger background operations for this charity (only if not in offline mode)
		if !h.Cfg.OfflineMode && charity.RegisteredNumber > 0 {
			var exists bool
			var hasScore bool

			h.DB.QueryRow("SELECT 1 FROM charities WHERE registered_number = ?", charity.RegisteredNumber).Scan(&exists)
			h.DB.QueryRow("SELECT 1 FROM charity_scores WHERE charity_number = ?", charity.RegisteredNumber).Scan(&hasScore)

			// Sync charity data if it doesn't exist
			if !exists {
				h.debugLog("Triggering background sync for charity %d", charity.RegisteredNumber)
				go func(charityNum int, cfg *config.Config) {
					charityNumStr := strconv.Itoa(charityNum)
					if err := sync.FetchAndStoreCharity(cfg, h.DB, charityNumStr); err != nil {
						log.Printf("Background sync failed for charity %s: %v", charityNumStr, err)
					} else {
						h.debugLog("Background sync completed for charity %s", charityNumStr)

						// After sync, calculate score
						if score, err := scoring.CalculateScore(h.DB, charityNum); err == nil {
							h.debugLog("Score calculated for charity %d: %.2f", charityNum, score.OverallScore)
						}
					}
				}(charity.RegisteredNumber, h.Cfg)
			} else if !hasScore {
				// Charity exists but no score - check if it has financial data before calculating
				h.debugLog("Checking if charity %d has financial data for scoring", charity.RegisteredNumber)
				go func(charityNum int) {
					// Check if charity has financial data (required for scoring)
					var hasFinancials bool
					h.DB.QueryRow("SELECT 1 FROM financials WHERE charity_number = ?", charityNum).Scan(&hasFinancials)

					if hasFinancials {
						if score, err := scoring.CalculateScore(h.DB, charityNum); err == nil {
							h.debugLog("Score calculated for charity %d: %.2f", charityNum, score.OverallScore)
						} else {
							log.Printf("Score calculation failed for charity %d: %v", charityNum, err)
						}
					} else {
						h.debugLog("Charity %d has no financial data yet, skipping score calculation", charityNum)
					}
				}(charity.RegisteredNumber)
			} else {
				h.debugLog("Charity %d already exists with score in database", charity.RegisteredNumber)
			}
		}

		charities = append(charities, charity)
	}

	h.debugLog("Returning %d charities from search (filtered %d RM charities)", len(charities), rmCount)
	return charities
}
