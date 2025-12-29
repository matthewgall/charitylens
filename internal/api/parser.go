package api

import (
	"charitylens/internal/models"
	"strconv"
	"strings"
	"time"
)

// ParseCharityData parses charity information from Charity Commission API response.
func ParseCharityData(data map[string]any, charityNum string) (models.Charity, error) {
	charity := models.Charity{
		LastUpdated: time.Now(),
	}

	// Parse registered number - prioritize charity number over company number
	possibleRegNumFields := []string{"reg_charity_number", "organisation_number", "registered_charity_number"}
	for _, field := range possibleRegNumFields {
		if rn, ok := data[field]; ok && rn != nil {
			switch v := rn.(type) {
			case float64:
				charity.RegisteredNumber = int(v)
			case int:
				charity.RegisteredNumber = v
			case string:
				if parsed, err := strconv.Atoi(v); err == nil {
					charity.RegisteredNumber = parsed
				}
			}
			break // Use the first field that works
		}
	}

	// Also capture company number if available (indicates trading subsidiary)
	if companyNum, ok := data["charity_company_registration_number"]; ok && companyNum != nil {
		switch v := companyNum.(type) {
		case string:
			charity.CompanyNumber = v
		case float64:
			charity.CompanyNumber = strconv.Itoa(int(v))
		case int:
			charity.CompanyNumber = strconv.Itoa(v)
		}
	}

	// If still 0, use the requested charity number
	if charity.RegisteredNumber == 0 {
		charity.RegisteredNumber, _ = strconv.Atoi(charityNum)
	}

	// Parse name (charity_name)
	if name, ok := data["charity_name"].(string); ok {
		charity.Name = name
	}

	// Parse status (reg_status)
	if status, ok := data["reg_status"].(string); ok {
		charity.Status = status
	}

	// Parse address (combine address lines)
	var addressParts []string
	if line1, ok := data["address_line_one"].(string); ok && line1 != "" {
		addressParts = append(addressParts, line1)
	}
	if line2, ok := data["address_line_two"].(string); ok && line2 != "" {
		addressParts = append(addressParts, line2)
	}
	if line3, ok := data["address_line_three"].(string); ok && line3 != "" {
		addressParts = append(addressParts, line3)
	}
	if line4, ok := data["address_line_four"].(string); ok && line4 != "" {
		addressParts = append(addressParts, line4)
	}
	if line5, ok := data["address_line_five"].(string); ok && line5 != "" {
		addressParts = append(addressParts, line5)
	}
	if postcode, ok := data["address_post_code"].(string); ok && postcode != "" {
		addressParts = append(addressParts, postcode)
	}
	if len(addressParts) > 0 {
		charity.Address = strings.Join(addressParts, ", ")
	}

	// Parse website
	if web, ok := data["web"].(string); ok {
		charity.Website = web
	}

	// Parse email
	if email, ok := data["email"].(string); ok {
		charity.Email = email
	}

	// Parse phone
	if phone, ok := data["phone"].(string); ok {
		charity.Phone = phone
	}

	// Parse what the charity does (who_what_where might contain this info)
	if what, ok := data["who_what_where"].(string); ok {
		charity.WhatTheCharityDoes = what
	}

	// Parse registration date
	if regDate, ok := data["date_of_registration"].(string); ok && regDate != "" {
		if parsed, err := time.Parse("2006-01-02T15:04:05", regDate); err == nil {
			charity.DateRegistered = parsed
		}
	}

	return charity, nil
}

// ParseFinancialData parses financial information from Charity Commission API response.
func ParseFinancialData(data map[string]any, charityNum int) (models.Financial, error) {
	fin := models.Financial{
		CharityNumber: charityNum,
		LastUpdated:   time.Now(),
	}

	// Parse financial data from API response
	if income, ok := data["latest_income"].(float64); ok {
		fin.TotalIncome = income
	}
	if expenditure, ok := data["latest_expenditure"].(float64); ok {
		fin.TotalSpending = expenditure
	}

	// Parse financial year end date
	if yearEndStr, ok := data["latest_acc_fin_year_end_date"].(string); ok && yearEndStr != "" {
		if parsed, err := time.Parse("2006-01-02T15:04:05", yearEndStr); err == nil {
			fin.FinancialYearEnd = parsed
		} else {
			// Try different date format
			if parsed, err := time.Parse("2006-01-02", yearEndStr); err == nil {
				fin.FinancialYearEnd = parsed
			}
		}
	}

	// Note: The basic API doesn't provide spending breakdown (charitable vs fundraising vs admin)
	// CharitableActivitiesSpend, RaisingFundsSpend, OtherSpend remain at 0 unless populated
	// from detailed financial history API

	return fin, nil
}

// ParseDetailedFinancialData parses detailed financial breakdown from financial history API response.
func ParseDetailedFinancialData(history []map[string]any) *models.Financial {
	if len(history) == 0 {
		return nil
	}

	// Use the most recent financial data (first entry)
	latest := history[0]

	fin := &models.Financial{}

	// Parse detailed spending breakdown
	if expCharitable, ok := latest["exp_charitable_activities"].(float64); ok {
		fin.CharitableActivitiesSpend = expCharitable
	}
	if expFundraising, ok := latest["exp_raising_funds"].(float64); ok {
		fin.RaisingFundsSpend = expFundraising
	}
	if expGovernance, ok := latest["exp_governance"].(float64); ok {
		fin.OtherSpend = expGovernance // Store governance in OtherSpend for now
	}

	return fin
}

// ParseTrusteesData parses trustee information from Charity Commission API response.
func ParseTrusteesData(data map[string]any, charityNum int) []models.Trustee {
	var trustees []models.Trustee

	if trusteeNames, ok := data["trustee_names"]; ok && trusteeNames != nil {
		// Handle different possible formats
		switch v := trusteeNames.(type) {
		case string:
			if v != "" {
				// Trustee names might be comma-separated, semicolon-separated, or pipe-separated
				separators := []string{",", ";", "|", "\n"}
				var names []string

				for _, sep := range separators {
					if strings.Contains(v, sep) {
						names = strings.Split(v, sep)
						break
					}
				}

				if len(names) == 0 {
					// No separator found, treat as single name
					names = []string{v}
				}

				for _, name := range names {
					name = strings.TrimSpace(name)
					if name != "" && !strings.Contains(strings.ToLower(name), "trustee") { // Filter out generic entries
						trustees = append(trustees, models.Trustee{
							CharityNumber: charityNum,
							Name:          name,
							LastUpdated:   time.Now(),
						})
					}
				}
			}
		case []any:
			// If it's an array of objects (V2 API format)
			for _, item := range v {
				if trusteeMap, ok := item.(map[string]any); ok {
					if name, ok := trusteeMap["trustee_name"].(string); ok && name != "" {
						trustees = append(trustees, models.Trustee{
							CharityNumber: charityNum,
							Name:          name,
							LastUpdated:   time.Now(),
						})
					}
				}
			}
		}
	}

	return trustees
}
