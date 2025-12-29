package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"charitylens/internal/config"
	"charitylens/internal/models"
	"charitylens/internal/scoring"
	"charitylens/internal/sync"
	"charitylens/web/templates"

	"github.com/go-chi/chi/v5"
)

type WebHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

func NewWebHandler(db *sql.DB, cfg *config.Config) *WebHandler {
	return &WebHandler{DB: db, Cfg: cfg}
}

func (h *WebHandler) SearchPage(w http.ResponseWriter, r *http.Request) {
	if err := templates.Templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *WebHandler) CharityPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	number, err := strconv.Atoi(idStr)
	if err != nil || number < 1 || number > 9999999999 {
		http.Error(w, "Invalid charity ID", http.StatusBadRequest)
		return
	}

	// Check if we have basic charity info (main charity only, linked_charity_number = 0)
	var charity models.Charity
	var website, email, address, whatTheCharityDoes sql.NullString
	err = h.DB.QueryRow(`
		SELECT registered_number, name, status, date_registered, address, website, email, what_the_charity_does
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

	// If charity not found, try to sync it first (unless in offline mode)
	if err == sql.ErrNoRows {
		if h.Cfg.OfflineMode {
			// In offline mode, just show not found error
			errorData := struct {
				Code      int
				Title     string
				Message   string
				IsLoading bool
				RetryURL  string
			}{
				Code:    404,
				Title:   "Charity Not Found",
				Message: "We couldn't find this charity in our database. Please check the charity number is correct.",
			}

			if err := templates.Templates.ExecuteTemplate(w, "error.html", errorData); err != nil {
				http.Error(w, "Error rendering page", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Charity %d not found in database, showing loading page", number)

		// Show loading page
		errorData := struct {
			Code      int
			Title     string
			Message   string
			IsLoading bool
			RetryURL  string
		}{
			IsLoading: true,
		}

		if err := templates.Templates.ExecuteTemplate(w, "error.html", errorData); err != nil {
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}

		// Trigger background sync
		go func() {
			log.Printf("Starting background sync for charity %d", number)
			if syncErr := sync.FetchAndStoreCharity(h.Cfg, h.DB, strconv.Itoa(number)); syncErr != nil {
				log.Printf("Failed to sync charity %d: %v", number, syncErr)
			} else {
				log.Printf("Successfully synced charity %d", number)
			}
		}()

		return
	} else if err != nil {
		// Database error
		log.Printf("Database error fetching charity %d: %v", number, err)
		errorData := struct {
			Code      int
			Title     string
			Message   string
			IsLoading bool
			RetryURL  string
		}{
			Code:     500,
			Title:    "Database Error",
			Message:  "We're having trouble accessing our database. Please try again later.",
			RetryURL: r.URL.Path,
		}

		if err := templates.Templates.ExecuteTemplate(w, "error.html", errorData); err != nil {
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	// Check if charity is removed
	if charity.Status == "Removed" || charity.Status == "RM" {
		errorData := struct {
			Code      int
			Title     string
			Message   string
			IsLoading bool
			RetryURL  string
		}{
			Code:    404,
			Title:   "Charity Removed",
			Message: "This charity has been removed from the register and is no longer active.",
		}

		if err := templates.Templates.ExecuteTemplate(w, "error.html", errorData); err != nil {
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	// Calculate score (don't cache in offline mode - always fresh)
	score, err := scoring.CalculateScore(h.DB, number, !h.Cfg.OfflineMode)
	if err != nil {
		log.Printf("Failed to calculate score for charity %d: %v", number, err)
		score = models.CharityScore{}
	}

	// Get trustees
	trusteeRows, err := h.DB.Query(`
		SELECT name FROM trustees WHERE charity_number = ? ORDER BY name
	`, number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer trusteeRows.Close()

	var trustees []models.Trustee
	for trusteeRows.Next() {
		var trustee models.Trustee
		err := trusteeRows.Scan(&trustee.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		trustees = append(trustees, trustee)
	}

	// Get financial data
	var financial models.Financial
	var trusteeCount sql.NullInt64
	err = h.DB.QueryRow(`
		SELECT financial_year_end, total_income, total_spending, charitable_activities_spend, 
		       raising_funds_spend, other_spend, reserves, assets, trustees
		FROM financials WHERE charity_number = ?
		ORDER BY financial_year_end DESC LIMIT 1
	`, number).Scan(
		&financial.FinancialYearEnd, &financial.TotalIncome, &financial.TotalSpending,
		&financial.CharitableActivitiesSpend, &financial.RaisingFundsSpend,
		&financial.OtherSpend, &financial.Reserves, &financial.Assets, &trusteeCount,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Failed to get financial data for charity %d: %v", number, err)
	}
	if trusteeCount.Valid {
		financial.Trustees = int(trusteeCount.Int64)
	}

	// Get activities
	activityRows, err := h.DB.Query(`
		SELECT description FROM activities WHERE charity_number = ? ORDER BY description
	`, number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer activityRows.Close()

	var activities []models.Activity
	for activityRows.Next() {
		var activity models.Activity
		err := activityRows.Scan(&activity.Description)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		activities = append(activities, activity)
	}

	data := struct {
		Charity    models.Charity
		Score      models.CharityScore
		Financial  models.Financial
		Trustees   []models.Trustee
		Activities []models.Activity
	}{
		Charity:    charity,
		Score:      score,
		Financial:  financial,
		Trustees:   trustees,
		Activities: activities,
	}

	if err := templates.Templates.ExecuteTemplate(w, "charity.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *WebHandler) ComparePage(w http.ResponseWriter, r *http.Request) {
	if err := templates.Templates.ExecuteTemplate(w, "compare.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *WebHandler) LicensePage(w http.ResponseWriter, r *http.Request) {
	if err := templates.Templates.ExecuteTemplate(w, "license.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *WebHandler) MethodologyPage(w http.ResponseWriter, r *http.Request) {
	if err := templates.Templates.ExecuteTemplate(w, "methodology.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
