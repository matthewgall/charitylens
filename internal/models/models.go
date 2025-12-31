package models

import "time"

// Charity represents a charity record
type Charity struct {
	OrganisationNumber  int        `json:"organisation_number" db:"organisation_number"`     // Unique org ID (primary key)
	RegisteredNumber    int        `json:"registered_number" db:"registered_number"`         // Charity registration number (can be shared)
	LinkedCharityNumber int        `json:"linked_charity_number" db:"linked_charity_number"` // 0 = main, 1+ = linked entities
	CompanyNumber       string     `json:"company_number" db:"company_number"`
	Name                string     `json:"name" db:"name"`
	Status              string     `json:"status" db:"status"`
	DateRegistered      time.Time  `json:"date_registered" db:"date_registered"`
	DateRemoved         *time.Time `json:"date_removed" db:"date_removed"`
	Address             string     `json:"address" db:"address"`
	Website             string     `json:"website" db:"website"`
	Email               string     `json:"email" db:"email"`
	Phone               string     `json:"phone" db:"phone"`
	WhatTheCharityDoes  string     `json:"what_the_charity_does" db:"what_the_charity_does"`
	WhoTheCharityHelps  string     `json:"who_the_charity_helps" db:"who_the_charity_helps"`
	HowTheCharityWorks  string     `json:"how_the_charity_works" db:"how_the_charity_works"`
	LastUpdated         time.Time  `json:"last_updated" db:"last_updated"`
	OverallScore        float64    `json:"overall_score,omitempty" db:"-"`    // Not stored in charities table, joined from scores
	LinkedCharities     []Charity  `json:"linked_charities,omitempty" db:"-"` // Populated when querying with linked entities
}

// Financial represents financial data for a charity
type Financial struct {
	CharityNumber             int       `json:"charity_number" db:"charity_number"`
	FinancialYearEnd          time.Time `json:"financial_year_end" db:"financial_year_end"`
	TotalIncome               float64   `json:"total_income" db:"total_income"`
	TotalSpending             float64   `json:"total_spending" db:"total_spending"`
	CharitableActivitiesSpend float64   `json:"charitable_activities_spend" db:"charitable_activities_spend"`
	RaisingFundsSpend         float64   `json:"raising_funds_spend" db:"raising_funds_spend"`
	OtherSpend                float64   `json:"other_spend" db:"other_spend"`
	Reserves                  float64   `json:"reserves" db:"reserves"`
	Assets                    float64   `json:"assets" db:"assets"`
	Employees                 int       `json:"employees" db:"employees"`
	Volunteers                int       `json:"volunteers" db:"volunteers"`
	Trustees                  int       `json:"trustees" db:"trustees"`
	LastUpdated               time.Time `json:"last_updated" db:"last_updated"`
}

// Trustee represents a trustee of a charity
type Trustee struct {
	CharityNumber int       `json:"charity_number" db:"charity_number"`
	Name          string    `json:"name" db:"name"`
	LastUpdated   time.Time `json:"last_updated" db:"last_updated"`
}

// Activity represents an activity of a charity
type Activity struct {
	CharityNumber int       `json:"charity_number" db:"charity_number"`
	Description   string    `json:"description" db:"description"`
	LastUpdated   time.Time `json:"last_updated" db:"last_updated"`
}

// CharityScore represents the calculated score for a charity
type CharityScore struct {
	CharityNumber        int       `json:"charity_number" db:"charity_number"`
	OverallScore         float64   `json:"overall_score" db:"overall_score"`
	EfficiencyScore      float64   `json:"efficiency_score" db:"efficiency_score"`
	FinancialHealthScore float64   `json:"financial_health_score" db:"financial_health_score"`
	TransparencyScore    float64   `json:"transparency_score" db:"transparency_score"`
	GovernanceScore      float64   `json:"governance_score" db:"governance_score"`
	ConfidenceLevel      string    `json:"confidence_level" db:"confidence_level"`
	LastCalculated       time.Time `json:"last_calculated" db:"last_calculated"`
}

// AnnualReturnHistory represents the filing history for a charity
type AnnualReturnHistory struct {
	ID                       int        `json:"id" db:"id"`
	OrganisationNumber       int        `json:"organisation_number" db:"organisation_number"`
	RegisteredCharityNumber  int        `json:"registered_charity_number" db:"registered_charity_number"`
	FinPeriodStartDate       *time.Time `json:"fin_period_start_date" db:"fin_period_start_date"`
	FinPeriodEndDate         *time.Time `json:"fin_period_end_date" db:"fin_period_end_date"`
	ARCycleReference         string     `json:"ar_cycle_reference" db:"ar_cycle_reference"`
	ReportingDueDate         *time.Time `json:"reporting_due_date" db:"reporting_due_date"`
	DateAnnualReturnReceived *time.Time `json:"date_annual_return_received" db:"date_annual_return_received"`
	DateAccountsReceived     *time.Time `json:"date_accounts_received" db:"date_accounts_received"`
	TotalGrossIncome         *float64   `json:"total_gross_income" db:"total_gross_income"`
	TotalGrossExpenditure    *float64   `json:"total_gross_expenditure" db:"total_gross_expenditure"`
	AccountsQualified        *bool      `json:"accounts_qualified" db:"accounts_qualified"`
	SuppressionInd           bool       `json:"suppression_ind" db:"suppression_ind"`
	SuppressionType          *string    `json:"suppression_type" db:"suppression_type"`
	DateOfExtract            *time.Time `json:"date_of_extract" db:"date_of_extract"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
}
