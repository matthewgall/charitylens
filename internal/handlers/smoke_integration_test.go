//go:build integration

package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"charitylens/internal/config"
	"charitylens/internal/database/sqlbuilder"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func TestAPIEndpointsSmoke(t *testing.T) {
	dbType := os.Getenv("SMOKE_DB")
	if dbType == "" {
		t.Skip("SMOKE_DB not set")
	}
	dsn := os.Getenv("SMOKE_DSN")
	if dsn == "" {
		t.Skip("SMOKE_DSN not set")
	}

	driver := dbType
	if dbType == "postgres" {
		driver = "postgres"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	t.Setenv("DATABASE_TYPE", dbType)

	if err := recreateSmokeSchema(t, db, dbType); err != nil {
		t.Fatalf("schema setup: %v", err)
	}
	if err := seedSmokeData(db); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	cfg := &config.Config{OfflineMode: true}
	h := NewCharityHandler(db, cfg)

	r := chi.NewRouter()
	r.Get("/api/charities/search", h.SearchCharities)
	r.Get("/api/charities/{number}", h.GetCharity)
	r.Get("/api/charities/compare", h.CompareCharities)

	assertJSONEndpoint(t, r, "/api/charities/search?q=Smoke")
	assertJSONEndpoint(t, r, "/api/charities/12345")
	assertJSONEndpoint(t, r, "/api/charities/compare?numbers=12345,12346")
}

func assertJSONEndpoint(t *testing.T, h http.Handler, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("%s status = %d body=%s", path, rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("%s invalid json: %v body=%s", path, err, rr.Body.String())
	}
}

func recreateSmokeSchema(t *testing.T, db *sql.DB, dbType string) error {
	t.Helper()

	statements := []string{
		"DROP TABLE IF EXISTS search_cache",
		"DROP TABLE IF EXISTS annual_return_history",
		"DROP TABLE IF EXISTS activities",
		"DROP TABLE IF EXISTS trustees",
		"DROP TABLE IF EXISTS charity_scores",
		"DROP TABLE IF EXISTS financials",
		"DROP TABLE IF EXISTS charities",
		`CREATE TABLE charities (
			organisation_number BIGINT PRIMARY KEY,
			registered_number BIGINT NOT NULL,
			linked_charity_number INT DEFAULT 0,
			company_number VARCHAR(64),
			name TEXT NOT NULL,
			status VARCHAR(32),
			date_registered TIMESTAMP NULL,
			date_removed TIMESTAMP NULL,
			address TEXT,
			website TEXT,
			email TEXT,
			phone TEXT,
			what_the_charity_does TEXT,
			who_the_charity_helps TEXT,
			how_the_charity_works TEXT,
			last_updated TIMESTAMP NULL
		)`,
		"CREATE INDEX idx_charities_registered_number ON charities(registered_number)",
		`CREATE TABLE financials (
			charity_number BIGINT NOT NULL,
			financial_year_end TIMESTAMP NOT NULL,
			total_income DOUBLE PRECISION,
			total_spending DOUBLE PRECISION,
			charitable_activities_spend DOUBLE PRECISION,
			raising_funds_spend DOUBLE PRECISION,
			other_spend DOUBLE PRECISION,
			reserves DOUBLE PRECISION,
			assets DOUBLE PRECISION,
			employees INT,
			volunteers INT,
			trustees INT,
			last_updated TIMESTAMP NULL,
			PRIMARY KEY (charity_number, financial_year_end)
		)`,
		`CREATE TABLE charity_scores (
			charity_number BIGINT PRIMARY KEY,
			overall_score DOUBLE PRECISION,
			efficiency_score DOUBLE PRECISION,
			financial_health_score DOUBLE PRECISION,
			transparency_score DOUBLE PRECISION,
			governance_score DOUBLE PRECISION,
			confidence_level VARCHAR(16),
			last_calculated TIMESTAMP NULL
		)`,
		`CREATE TABLE trustees (
			charity_number BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			last_updated TIMESTAMP NULL,
			PRIMARY KEY (charity_number, name)
		)`,
		`CREATE TABLE activities (
			charity_number BIGINT NOT NULL,
			description TEXT NOT NULL,
			last_updated TIMESTAMP NULL
		)`,
		`CREATE TABLE search_cache (
			query VARCHAR(255) NOT NULL,
			search_type VARCHAR(32) NOT NULL,
			last_searched TIMESTAMP NULL,
			result_count INT DEFAULT 0,
			PRIMARY KEY (query, search_type)
		)`,
	}

	annualHistory := ""
	switch dbType {
	case "postgres":
		annualHistory = `CREATE TABLE annual_return_history (
			id BIGSERIAL PRIMARY KEY,
			organisation_number BIGINT NOT NULL,
			registered_charity_number BIGINT NOT NULL,
			fin_period_start_date TIMESTAMP NULL,
			fin_period_end_date TIMESTAMP NULL,
			ar_cycle_reference VARCHAR(10),
			reporting_due_date TIMESTAMP NULL,
			date_annual_return_received TIMESTAMP NULL,
			date_accounts_received TIMESTAMP NULL,
			total_gross_income DOUBLE PRECISION,
			total_gross_expenditure DOUBLE PRECISION,
			accounts_qualified BOOLEAN,
			suppression_ind BOOLEAN,
			suppression_type VARCHAR(50),
			date_of_extract TIMESTAMP NULL,
			created_at TIMESTAMP NULL
		)`
	default:
		annualHistory = `CREATE TABLE annual_return_history (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			organisation_number BIGINT NOT NULL,
			registered_charity_number BIGINT NOT NULL,
			fin_period_start_date TIMESTAMP NULL,
			fin_period_end_date TIMESTAMP NULL,
			ar_cycle_reference VARCHAR(10),
			reporting_due_date TIMESTAMP NULL,
			date_annual_return_received TIMESTAMP NULL,
			date_accounts_received TIMESTAMP NULL,
			total_gross_income DOUBLE,
			total_gross_expenditure DOUBLE,
			accounts_qualified BOOLEAN,
			suppression_ind BOOLEAN,
			suppression_type VARCHAR(50),
			date_of_extract TIMESTAMP NULL,
			created_at TIMESTAMP NULL
		)`
	}

	statements = append(statements, annualHistory)

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func seedSmokeData(db *sql.DB) error {
	b := sqlbuilder.Builder()
	now := time.Now()

	charityInsert, charityArgs, err := b.Insert("charities").
		Columns("organisation_number", "registered_number", "linked_charity_number", "name", "status", "date_registered", "last_updated").
		Values(12345, 12345, 0, "Smoke Test Charity", "Registered", now, now).
		Values(12346, 12346, 0, "Smoke Compare Charity", "Registered", now, now).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := db.Exec(charityInsert, charityArgs...); err != nil {
		return err
	}

	financialInsert, financialArgs, err := b.Insert("financials").
		Columns("charity_number", "financial_year_end", "total_income", "total_spending", "charitable_activities_spend", "reserves", "assets", "trustees", "last_updated").
		Values(12345, now, 100000.0, 80000.0, 64000.0, 40000.0, 60000.0, 4, now).
		Values(12346, now, 120000.0, 90000.0, 72000.0, 50000.0, 70000.0, 5, now).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := db.Exec(financialInsert, financialArgs...); err != nil {
		return err
	}

	trusteeInsert, trusteeArgs, err := b.Insert("trustees").
		Columns("charity_number", "name", "last_updated").
		Values(12345, "Trustee One", now).
		Values(12346, "Trustee Two", now).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := db.Exec(trusteeInsert, trusteeArgs...); err != nil {
		return err
	}

	return nil
}
