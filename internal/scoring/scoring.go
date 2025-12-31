package scoring

import (
	"database/sql"
	"log"
	"math"
	"time"

	"charitylens/internal/models"
)

func CalculateScore(db *sql.DB, charityNumber int, cacheScore ...bool) (models.CharityScore, error) {
	// cacheScore is optional - defaults to true for backwards compatibility
	shouldCache := true
	if len(cacheScore) > 0 {
		shouldCache = cacheScore[0]
	}
	score := models.CharityScore{
		CharityNumber:  charityNumber,
		LastCalculated: time.Now(),
	}

	// Get charity info (main charity only)
	var charity models.Charity
	var website sql.NullString
	var lastUpdated sql.NullTime
	err := db.QueryRow(`
		SELECT registered_number, name, website, last_updated
		FROM charities WHERE registered_number = ? AND linked_charity_number = 0
	`, charityNumber).Scan(&charity.RegisteredNumber, &charity.Name, &website, &lastUpdated)
	if err != nil {
		return score, err
	}

	// Convert NullString to string
	if website.Valid {
		charity.Website = website.String
	}
	if lastUpdated.Valid {
		charity.LastUpdated = lastUpdated.Time
	}

	// Get latest financial data
	var fin models.Financial
	err = db.QueryRow(`
		SELECT total_income, total_spending, charitable_activities_spend, reserves, assets, COALESCE(trustees, 0)
		FROM financials WHERE charity_number = ?
		ORDER BY financial_year_end DESC LIMIT 1
	`, charityNumber).Scan(&fin.TotalIncome, &fin.TotalSpending, &fin.CharitableActivitiesSpend, &fin.Reserves, &fin.Assets, &fin.Trustees)
	hasFinancial := err == nil

	// Get trustee count
	var trusteeCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM trustees WHERE charity_number = ?
	`, charityNumber).Scan(&trusteeCount)

	// Calculate Efficiency Score (40%)
	var efficiencyScore float64
	hasSpendingBreakdown := hasFinancial && fin.CharitableActivitiesSpend > 0
	if hasSpendingBreakdown && fin.TotalSpending > 0 {
		ratio := fin.CharitableActivitiesSpend / fin.TotalSpending
		efficiencyScore = math.Min(100, ratio*100)
	} else if hasFinancial && fin.TotalSpending > 0 {
		// No spending breakdown available - use neutral score
		// Don't penalize charities for missing data
		efficiencyScore = 60 // Neutral/average score when data unavailable
	}
	score.EfficiencyScore = efficiencyScore

	// Calculate Financial Health Score (30%)
	var financialHealthScore float64
	if hasFinancial && fin.TotalSpending > 0 {
		monthlySpending := fin.TotalSpending / 12

		// Check if we have valid reserves data
		if fin.Reserves > 0 || fin.Assets > 0 {
			// Use reserves if available, otherwise use assets as proxy
			reserves := fin.Reserves
			if reserves == 0 && fin.Assets > 0 {
				reserves = fin.Assets
			}

			reserveMonths := reserves / monthlySpending
			if reserveMonths >= 3 && reserveMonths <= 12 {
				// Optimal range: 3-12 months of reserves
				financialHealthScore = 100
			} else if reserveMonths < 3 {
				// Too few reserves: scale from 0-100
				financialHealthScore = (reserveMonths / 3) * 100
			} else {
				// More than 12 months: still good, just cap the penalty
				// Having extra reserves isn't as bad as having too few
				// Gentle penalty: 100 at 12mo, 90 at 24mo, 80 at 36mo, floor at 70
				excessMonths := reserveMonths - 12
				penalty := math.Min(30, (excessMonths/12)*5) // Max 30 point penalty
				financialHealthScore = math.Max(70, 100-penalty)
			}
		} else {
			// No reserves/assets data available - use neutral score
			// Don't penalize charities for missing financial data
			// New or small charities may not have detailed reserves reporting
			financialHealthScore = 50 // Neutral score when reserves data unavailable
		}
	}
	score.FinancialHealthScore = financialHealthScore

	// Calculate Transparency Score (20%) - Enhanced with filing history
	transparencyScore := 0.0

	// Website presence (30 points)
	if charity.Website != "" {
		transparencyScore += 30
	}

	// Has current financial data (20 points)
	if hasFinancial {
		transparencyScore += 20
	}

	// Has trustees listed (10 points)
	if trusteeCount > 0 {
		transparencyScore += 10
	}

	// Filing timeliness - last 3 years (25 points)
	// Check if annual returns were filed on time
	filingScore := calculateFilingTimeliness(db, charityNumber)
	transparencyScore += filingScore * 0.25 // Scale 0-100 to 0-25

	// Filing consistency - no gaps in last 5 years (10 points)
	consistencyScore := calculateFilingConsistency(db, charityNumber)
	transparencyScore += consistencyScore * 0.10 // Scale 0-100 to 0-10

	// Accounts quality - no qualified accounts (5 points)
	qualityScore := calculateAccountsQuality(db, charityNumber)
	transparencyScore += qualityScore * 0.05 // Scale 0-100 to 0-5

	score.TransparencyScore = transparencyScore

	// Calculate Governance Score (10%)
	governanceScore := 0.0
	if trusteeCount >= 3 {
		governanceScore = 100
	} else if trusteeCount > 0 {
		governanceScore = float64(trusteeCount) / 3 * 100
	}
	score.GovernanceScore = governanceScore

	// Overall Score
	score.OverallScore = (efficiencyScore*0.4 + financialHealthScore*0.3 + transparencyScore*0.2 + governanceScore*0.1)

	// Confidence Level
	confidence := "high"
	dataCompleteness := 0
	if hasFinancial {
		dataCompleteness += 1
	}
	if charity.Website != "" {
		dataCompleteness += 1
	}
	if trusteeCount > 0 {
		dataCompleteness += 1
	}
	if time.Since(charity.LastUpdated) > 365*24*time.Hour {
		dataCompleteness -= 1
	}
	if dataCompleteness >= 2 {
		confidence = "high"
	} else if dataCompleteness == 1 {
		confidence = "medium"
	} else {
		confidence = "low"
	}
	score.ConfidenceLevel = confidence

	// Store the score in the database (unless caching is disabled)
	if shouldCache {
		_, err = db.Exec(`
			INSERT OR REPLACE INTO charity_scores
			(charity_number, overall_score, efficiency_score, financial_health_score, transparency_score, governance_score, confidence_level, last_calculated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			score.CharityNumber, score.OverallScore, score.EfficiencyScore, score.FinancialHealthScore,
			score.TransparencyScore, score.GovernanceScore, score.ConfidenceLevel, score.LastCalculated)
		if err != nil {
			log.Printf("Failed to store score for charity %d: %v", charityNumber, err)
			return score, err
		}
	}

	return score, nil
}

// calculateFilingTimeliness checks if annual returns were filed on time in the last 3 years
// Returns a score from 0-100
func calculateFilingTimeliness(db *sql.DB, charityNumber int) float64 {
	// Get the last 3 filing records
	rows, err := db.Query(`
		SELECT reporting_due_date, date_annual_return_received, date_accounts_received
		FROM annual_return_history
		WHERE registered_charity_number = ?
		AND reporting_due_date IS NOT NULL
		ORDER BY fin_period_end_date DESC
		LIMIT 3
	`, charityNumber)
	if err != nil {
		return 50 // Neutral score if no data
	}
	defer rows.Close()

	onTimeCount := 0
	totalCount := 0

	for rows.Next() {
		var dueDate, arReceived, accountsReceived sql.NullTime
		if err := rows.Scan(&dueDate, &arReceived, &accountsReceived); err != nil {
			continue
		}

		if !dueDate.Valid {
			continue
		}

		totalCount++

		// Check if either the annual return or accounts were received on time
		// Some charities file AR and accounts separately
		arOnTime := arReceived.Valid && !arReceived.Time.After(dueDate.Time)
		accountsOnTime := accountsReceived.Valid && !accountsReceived.Time.After(dueDate.Time)

		if arOnTime || accountsOnTime {
			onTimeCount++
		}
	}

	if totalCount == 0 {
		return 50 // Neutral score if no filing data
	}

	// Calculate percentage and scale to 0-100
	percentage := float64(onTimeCount) / float64(totalCount)
	return percentage * 100
}

// calculateFilingConsistency checks for gaps in filing history over the last 5 years
// Returns a score from 0-100
func calculateFilingConsistency(db *sql.DB, charityNumber int) float64 {
	// Get filing records from the last 5 years
	rows, err := db.Query(`
		SELECT ar_cycle_reference, date_annual_return_received
		FROM annual_return_history
		WHERE registered_charity_number = ?
		AND fin_period_end_date >= date('now', '-5 years')
		ORDER BY fin_period_end_date DESC
	`, charityNumber)
	if err != nil {
		return 50 // Neutral score if no data
	}
	defer rows.Close()

	expectedFilings := 0
	actualFilings := 0

	// Count expected vs actual filings
	for rows.Next() {
		var cycleRef string
		var received sql.NullTime
		if err := rows.Scan(&cycleRef, &received); err != nil {
			continue
		}

		expectedFilings++
		if received.Valid {
			actualFilings++
		}
	}

	if expectedFilings == 0 {
		return 50 // Neutral score if charity is too new or no data
	}

	// Calculate consistency percentage
	percentage := float64(actualFilings) / float64(expectedFilings)
	return percentage * 100
}

// calculateAccountsQuality checks for qualified accounts (audit issues) in recent years
// Returns a score from 0-100
func calculateAccountsQuality(db *sql.DB, charityNumber int) float64 {
	// Check last 3 years for qualified accounts
	var qualifiedCount int
	var totalCount int

	err := db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN accounts_qualified = 1 THEN 1 ELSE 0 END) as qualified
		FROM annual_return_history
		WHERE registered_charity_number = ?
		AND accounts_qualified IS NOT NULL
		AND fin_period_end_date >= date('now', '-3 years')
	`, charityNumber).Scan(&totalCount, &qualifiedCount)

	if err != nil || totalCount == 0 {
		return 100 // Assume good quality if no data (benefit of doubt)
	}

	// If any accounts were qualified, reduce the score
	if qualifiedCount > 0 {
		// Penalty based on how many were qualified
		penalty := float64(qualifiedCount) / float64(totalCount) * 100
		return 100 - penalty
	}

	return 100 // No qualified accounts = perfect score
}
