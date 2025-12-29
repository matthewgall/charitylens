package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"charitylens/internal/api"
	"charitylens/internal/config"
)

// debugLog logs a message only if debug mode is enabled
func debugLog(cfg *config.Config, format string, args ...any) {
	if cfg.Debug {
		log.Printf(format, args...)
	}
}

func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func StartSyncWorker(cfg *config.Config, db *sql.DB) {
	ticker := time.NewTicker(time.Duration(cfg.SyncIntervalHours) * time.Hour)
	defer ticker.Stop()

	log.Println("Starting sync worker...")

	for range ticker.C {
		if err := SyncCharities(cfg, db); err != nil {
			log.Printf("Sync failed: %v", err)
		}
	}
}

func SyncCharities(cfg *config.Config, db *sql.DB) error {
	// This function is called by the sync worker and admin endpoint
	// Since we use sync-on-demand when users access charities,
	// this is now a no-op to avoid unnecessary API calls
	debugLog(cfg, "SyncCharities called - no action taken (sync-on-demand is enabled)")
	return nil
}

func FetchAndStoreCharity(cfg *config.Config, db *sql.DB, charityNum string) error {
	debugLog(cfg, "Fetching charity %s from Charity Commission API", charityNum)

	// Create API client with rate limiter
	rateLimiter := api.NewRateLimiter(10.0) // 10 req/s rate limit
	client := api.NewClient(api.ClientConfig{
		APIKey:      cfg.CharityAPIKey,
		RateLimiter: rateLimiter,
		Verbose:     cfg.Debug,
	})
	ctx := context.Background()

	// Convert charity number to int
	charityNumInt, err := strconv.Atoi(charityNum)
	if err != nil {
		return fmt.Errorf("invalid charity number %s: %w", charityNum, err)
	}

	// Fetch charity details
	data, err := client.FetchCharityDetails(ctx, charityNumInt)
	if err != nil {
		log.Printf("Failed to fetch charity %s: %v", charityNum, err)
		return err
	}

	debugLog(cfg, "Successfully received and parsed API data for charity %s", charityNum)

	// Parse and store charity data
	debugLog(cfg, "Parsing charity data for %s", charityNum)
	charity, err := api.ParseCharityData(data, charityNum)
	if err != nil {
		log.Printf("Failed to parse charity data for %s: %v", charityNum, err)
		return err
	}
	debugLog(cfg, "Parsed charity: registered_number=%d, name=%s", charity.RegisteredNumber, charity.Name)

	// Insert charity
	debugLog(cfg, "Storing charity data for %s in database", charityNum)
	_, err = db.Exec(`
		INSERT OR REPLACE INTO charities
		(registered_number, company_number, name, status, date_registered, address, website, email, what_the_charity_does)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		charity.RegisteredNumber, charity.CompanyNumber, charity.Name, charity.Status, charity.DateRegistered,
		charity.Address, charity.Website, charity.Email, charity.WhatTheCharityDoes)
	if err != nil {
		log.Printf("Failed to store charity data for %s: %v", charityNum, err)
		return err
	}
	debugLog(cfg, "Successfully stored charity data for %s", charityNum)

	// Parse and store financial data
	debugLog(cfg, "Processing financial data for charity %s", charityNum)
	if fin, err := api.ParseFinancialData(data, charity.RegisteredNumber); err == nil {
		// Fetch detailed financial breakdown from financial history endpoint
		if detailedFin, err := client.FetchFinancialHistory(ctx, charityNumInt); err == nil && len(detailedFin) > 0 {
			// Parse detailed financials from the history
			if parsed := api.ParseDetailedFinancialData(detailedFin); parsed != nil {
				// Use real data from financial history if available
				if parsed.CharitableActivitiesSpend > 0 {
					fin.CharitableActivitiesSpend = parsed.CharitableActivitiesSpend
				}
				if parsed.RaisingFundsSpend > 0 {
					fin.RaisingFundsSpend = parsed.RaisingFundsSpend
				}
				if parsed.OtherSpend > 0 {
					fin.OtherSpend = parsed.OtherSpend
				}
				debugLog(cfg, "Using detailed financials: charitable=%.2f, fundraising=%.2f", fin.CharitableActivitiesSpend, fin.RaisingFundsSpend)
			}
		}

		_, err := db.Exec(`
			INSERT OR REPLACE INTO financials
			(charity_number, financial_year_end, total_income, total_spending, charitable_activities_spend, raising_funds_spend, other_spend, reserves, assets, trustees, last_updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fin.CharityNumber, fin.FinancialYearEnd, fin.TotalIncome, fin.TotalSpending,
			fin.CharitableActivitiesSpend, fin.RaisingFundsSpend, fin.OtherSpend,
			fin.Reserves, fin.Assets, fin.Trustees, fin.LastUpdated)
		if err != nil {
			log.Printf("Failed to store financial data for charity %s: %v", charityNum, err)
		} else {
			debugLog(cfg, "Stored financial data for charity %s (income: %.2f, spending: %.2f, charitable: %.2f)", charityNum, fin.TotalIncome, fin.TotalSpending, fin.CharitableActivitiesSpend)
		}
	} else {
		debugLog(cfg, "Failed to parse financial data for charity %s: %v", charityNum, err)
	}

	// Parse and store trustees
	trustees := api.ParseTrusteesData(data, charity.RegisteredNumber)
	if len(trustees) > 0 {
		debugLog(cfg, "Processing %d trustee records for charity %s", len(trustees), charityNum)
		for i, trustee := range trustees {
			debugLog(cfg, "Processing trustee record %d for charity %s: %s", i+1, charityNum, trustee.Name)
			_, err := db.Exec(`
				INSERT OR REPLACE INTO trustees
				(charity_number, name, last_updated)
				VALUES (?, ?, ?)`,
				trustee.CharityNumber, trustee.Name, trustee.LastUpdated)
			if err != nil {
				log.Printf("Failed to store trustee data for charity %s: %v", charityNum, err)
			} else {
				debugLog(cfg, "Stored trustee data for charity %s: %s", charityNum, trustee.Name)
			}
		}
	} else {
		debugLog(cfg, "No trustee data available for charity %s", charityNum)
	}

	debugLog(cfg, "Completed data storage for charity %s", charityNum)
	return nil
}

func SearchCharitiesByName(cfg *config.Config, query string) ([]map[string]any, error) {
	debugLog(cfg, "Searching charities by name: %s", query)

	// Create API client with rate limiter
	rateLimiter := api.NewRateLimiter(10.0) // 10 req/s rate limit
	client := api.NewClient(api.ClientConfig{
		APIKey:      cfg.CharityAPIKey,
		RateLimiter: rateLimiter,
		Verbose:     cfg.Debug,
	})
	ctx := context.Background()

	// Search using the client
	results, err := client.SearchByName(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search charities by name: %w", err)
	}

	debugLog(cfg, "Search returned %d results", len(results))
	if cfg.Debug && len(results) > 0 {
		debugLog(cfg, "First result keys: %v", getMapKeys(results[0]))
		debugLog(cfg, "First result sample: charity_name=%v, reg_status=%v",
			results[0]["charity_name"], results[0]["reg_status"])
	}

	return results, nil
}

func SearchCharitiesByNumber(cfg *config.Config, charityNum string) ([]map[string]any, error) {
	debugLog(cfg, "Searching charity by number: %s", charityNum)

	// Create API client with rate limiter
	rateLimiter := api.NewRateLimiter(10.0) // 10 req/s rate limit
	client := api.NewClient(api.ClientConfig{
		APIKey:      cfg.CharityAPIKey,
		RateLimiter: rateLimiter,
		Verbose:     cfg.Debug,
	})
	ctx := context.Background()

	// Search using the client - returns []map[string]any
	results, err := client.SearchByNumber(ctx, charityNum)
	if err != nil {
		return nil, fmt.Errorf("failed to search charity by number: %w", err)
	}

	return results, nil
}
