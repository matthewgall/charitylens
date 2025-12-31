package importer

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"charitylens/internal/scoring"
)

// CharityRecord represents a charity record from the JSON dump
type CharityRecord struct {
	DateOfExtract                    string   `json:"date_of_extract"`
	OrganisationNumber               int      `json:"organisation_number"`
	RegisteredCharityNumber          int      `json:"registered_charity_number"`
	LinkedCharityNumber              int      `json:"linked_charity_number"`
	CharityName                      string   `json:"charity_name"`
	CharityType                      *string  `json:"charity_type"`
	CharityRegistrationStatus        string   `json:"charity_registration_status"`
	DateOfRegistration               string   `json:"date_of_registration"`
	DateOfRemoval                    *string  `json:"date_of_removal"`
	CharityReportingStatus           *string  `json:"charity_reporting_status"`
	LatestAccFinPeriodStartDate      *string  `json:"latest_acc_fin_period_start_date"`
	LatestAccFinPeriodEndDate        *string  `json:"latest_acc_fin_period_end_date"`
	LatestIncome                     *float64 `json:"latest_income"`
	LatestExpenditure                *float64 `json:"latest_expenditure"`
	CharityContactAddress1           *string  `json:"charity_contact_address1"`
	CharityContactAddress2           *string  `json:"charity_contact_address2"`
	CharityContactAddress3           *string  `json:"charity_contact_address3"`
	CharityContactAddress4           *string  `json:"charity_contact_address4"`
	CharityContactAddress5           *string  `json:"charity_contact_address5"`
	CharityContactPostcode           *string  `json:"charity_contact_postcode"`
	CharityContactPhone              *string  `json:"charity_contact_phone"`
	CharityContactEmail              *string  `json:"charity_contact_email"`
	CharityContactWeb                *string  `json:"charity_contact_web"`
	CharityCompanyRegistrationNumber *string  `json:"charity_company_registration_number"`
	CharityInsolvent                 bool     `json:"charity_insolvent"`
	CharityInAdministration          bool     `json:"charity_in_administration"`
	CharityPreviouslyExcepted        *bool    `json:"charity_previously_excepted"`
	CharityIsCDFOrCIF                *bool    `json:"charity_is_cdf_or_cif"`
	CharityIsCIO                     *bool    `json:"charity_is_cio"`
	CIOIsDissolved                   *bool    `json:"cio_is_dissolved"`
	DateCIODissolutionNotice         *string  `json:"date_cio_dissolution_notice"`
	CharityActivities                *string  `json:"charity_activities"`
	CharityGiftAid                   *bool    `json:"charity_gift_aid"`
	CharityHasLand                   *bool    `json:"charity_has_land"`
}

// TrusteeRecord represents a trustee record from the JSON dump
type TrusteeRecord struct {
	DateOfExtract            string  `json:"date_of_extract"`
	OrganisationNumber       int     `json:"organisation_number"`
	RegisteredCharityNumber  int     `json:"registered_charity_number"`
	LinkedCharityNumber      int     `json:"linked_charity_number"`
	TrusteeID                int     `json:"trustee_id"`
	TrusteeName              string  `json:"trustee_name"`
	TrusteeIsChair           bool    `json:"trustee_is_chair"`
	IndividualOrOrganisation string  `json:"individual_or_organisation"`
	TrusteeDateOfAppointment *string `json:"trustee_date_of_appointment"`
}

// AnnualReturnPartBRecord represents detailed financial data from annual returns
type AnnualReturnPartBRecord struct {
	DateOfExtract                string   `json:"date_of_extract"`
	OrganisationNumber           int      `json:"organisation_number"`
	RegisteredCharityNumber      int      `json:"registered_charity_number"`
	LatestFinPeriodSubmittedInd  bool     `json:"latest_fin_period_submitted_ind"`
	FinPeriodOrderNumber         int      `json:"fin_period_order_number"`
	FinPeriodStartDate           string   `json:"fin_period_start_date"`
	FinPeriodEndDate             string   `json:"fin_period_end_date"`
	ARReceivedDate               *string  `json:"ar_received_date"`
	IncomeTotalIncomeEndowments  *float64 `json:"income_total_income_and_endowments"`
	ExpenditureCharitableExpend  *float64 `json:"expenditure_charitable_expenditure"`
	ExpenditureRaisingFunds      *float64 `json:"expenditure_raising_funds"`
	ExpenditureGovernance        *float64 `json:"expenditure_governance"`
	ExpenditureTotal             *float64 `json:"expenditure_total"`
	Reserves                     *float64 `json:"reserves"`
	AssetsTotalAssetsLiabilities *float64 `json:"assets_total_assets_and_liabilities"`
	CountEmployees               *int     `json:"count_employees"`
}

type AnnualReturnHistoryRecord struct {
	DateOfExtract            string   `json:"date_of_extract"`
	OrganisationNumber       int      `json:"organisation_number"`
	RegisteredCharityNumber  int      `json:"registered_charity_number"`
	FinPeriodStartDate       *string  `json:"fin_period_start_date"`
	FinPeriodEndDate         *string  `json:"fin_period_end_date"`
	ARCycleReference         string   `json:"ar_cycle_reference"`
	ReportingDueDate         *string  `json:"reporting_due_date"`
	DateAnnualReturnReceived *string  `json:"date_annual_return_received"`
	DateAccountsReceived     *string  `json:"date_accounts_received"`
	TotalGrossIncome         *float64 `json:"total_gross_income"`
	TotalGrossExpenditure    *float64 `json:"total_gross_expenditure"`
	AccountsQualified        *bool    `json:"accounts_qualified"`
	SuppressionInd           bool     `json:"suppression_ind"`
	SuppressionType          *string  `json:"suppression_type"`
}

// ImportProgress tracks import progress
type ImportProgress struct {
	TotalRecords     int
	ProcessedRecords int
	SuccessRecords   int
	SkippedRecords   int
	FailedRecords    int
	StartTime        time.Time
	LastUpdate       time.Time
}

// ImportConfig holds configuration for the import process
type ImportConfig struct {
	CharityFile             string
	TrusteeFile             string
	FinancialFile           string // Annual return partb file
	AnnualReturnHistoryFile string // Annual return history file
	BatchSize               int
	ProgressInterval        int // Log progress every N records
	Verbose                 bool
}

// Importer handles importing charity data from JSON files
type Importer struct {
	db       *sql.DB
	config   ImportConfig
	progress ImportProgress
}

// NewImporter creates a new importer
func NewImporter(db *sql.DB, config ImportConfig) *Importer {
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.ProgressInterval == 0 {
		config.ProgressInterval = 5000
	}
	return &Importer{
		db:     db,
		config: config,
	}
}

// stripBOM removes UTF-8 BOM if present and returns a reader
func stripBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	// Peek at the first 3 bytes to check for UTF-8 BOM (EF BB BF)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peek, err := br.Peek(3)
	if err == nil && bytes.Equal(peek, bom) {
		// Discard the BOM
		br.Discard(3)
	}
	return br
}

// ImportCharities imports charities from a JSON file
func (i *Importer) ImportCharities() error {
	log.Printf("Starting charity import from: %s", i.config.CharityFile)
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	file, err := os.Open(i.config.CharityFile)
	if err != nil {
		return fmt.Errorf("failed to open charity file: %w", err)
	}
	defer file.Close()

	// Strip BOM if present
	reader := stripBOM(file)
	return i.importCharitiesFromReader(reader)
}

// ImportCharitiesFromReader imports charities from an io.Reader (for in-memory data)
func (i *Importer) ImportCharitiesFromReader(r io.Reader) error {
	log.Println("Starting charity import from in-memory data")
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	reader := stripBOM(r)
	return i.importCharitiesFromReader(reader)
}

// importCharitiesFromReader is the internal implementation that works with any reader
func (i *Importer) importCharitiesFromReader(reader io.Reader) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read opening bracket: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected array opening bracket, got: %v", token)
	}

	batch := make([]CharityRecord, 0, i.config.BatchSize)
	recordNum := 0

	// Process array elements
	for decoder.More() {
		var record CharityRecord
		if err := decoder.Decode(&record); err != nil {
			log.Printf("Failed to decode record %d: %v", recordNum, err)
			i.progress.FailedRecords++
			continue
		}

		batch = append(batch, record)
		recordNum++
		i.progress.TotalRecords = recordNum

		// Process batch when full
		if len(batch) >= i.config.BatchSize {
			if err := i.insertCharityBatch(batch); err != nil {
				log.Printf("Failed to insert batch: %v", err)
			}
			batch = batch[:0] // Reset batch
		}

		// Log progress
		if recordNum%i.config.ProgressInterval == 0 {
			i.logProgress()
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := i.insertCharityBatch(batch); err != nil {
			log.Printf("Failed to insert final batch: %v", err)
		}
	}

	i.logFinalStats("Charity import")
	return nil
}

// ImportTrustees imports trustees from a JSON file
func (i *Importer) ImportTrustees() error {
	log.Printf("Starting trustee import from: %s", i.config.TrusteeFile)
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	file, err := os.Open(i.config.TrusteeFile)
	if err != nil {
		return fmt.Errorf("failed to open trustee file: %w", err)
	}
	defer file.Close()

	// Strip BOM if present
	reader := stripBOM(file)
	return i.importTrusteesFromReader(reader)
}

// ImportTrusteesFromReader imports trustees from an io.Reader (for in-memory data)
func (i *Importer) ImportTrusteesFromReader(r io.Reader) error {
	log.Println("Starting trustee import from in-memory data")
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	reader := stripBOM(r)
	return i.importTrusteesFromReader(reader)
}

// importTrusteesFromReader is the internal implementation that works with any reader
func (i *Importer) importTrusteesFromReader(reader io.Reader) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read opening bracket: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected array opening bracket, got: %v", token)
	}

	batch := make([]TrusteeRecord, 0, i.config.BatchSize)
	recordNum := 0

	// Process array elements
	for decoder.More() {
		var record TrusteeRecord
		if err := decoder.Decode(&record); err != nil {
			log.Printf("Failed to decode trustee record %d: %v", recordNum, err)
			i.progress.FailedRecords++
			continue
		}

		batch = append(batch, record)
		recordNum++
		i.progress.TotalRecords = recordNum

		// Process batch when full
		if len(batch) >= i.config.BatchSize {
			if err := i.insertTrusteeBatch(batch); err != nil {
				log.Printf("Failed to insert trustee batch: %v", err)
			}
			batch = batch[:0] // Reset batch
		}

		// Log progress
		if recordNum%i.config.ProgressInterval == 0 {
			i.logProgress()
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := i.insertTrusteeBatch(batch); err != nil {
			log.Printf("Failed to insert final trustee batch: %v", err)
		}
	}

	i.logFinalStats("Trustee import")
	return nil
}

// ImportFinancials imports financial data from annual return partb JSON file
func (i *Importer) ImportFinancials() error {
	if i.config.FinancialFile == "" {
		log.Println("No financial file specified, skipping financial data import")
		return nil
	}

	log.Printf("Starting financial data import from: %s", i.config.FinancialFile)
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	file, err := os.Open(i.config.FinancialFile)
	if err != nil {
		return fmt.Errorf("failed to open financial file: %w", err)
	}
	defer file.Close()

	// Strip BOM if present
	reader := stripBOM(file)
	return i.importFinancialsFromReader(reader)
}

// ImportFinancialsFromReader imports financial data from an io.Reader (for in-memory data)
func (i *Importer) ImportFinancialsFromReader(r io.Reader) error {
	log.Println("Starting financial data import from in-memory data")
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	reader := stripBOM(r)
	return i.importFinancialsFromReader(reader)
}

// importFinancialsFromReader is the internal implementation that works with any reader
func (i *Importer) importFinancialsFromReader(reader io.Reader) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read opening bracket: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected array opening bracket, got: %v", token)
	}

	batch := make([]AnnualReturnPartBRecord, 0, i.config.BatchSize)
	recordNum := 0

	// Process array elements
	for decoder.More() {
		var record AnnualReturnPartBRecord
		if err := decoder.Decode(&record); err != nil {
			log.Printf("Failed to decode financial record %d: %v", recordNum, err)
			i.progress.FailedRecords++
			continue
		}

		// Only process latest period for each charity to avoid duplicates
		if record.LatestFinPeriodSubmittedInd {
			batch = append(batch, record)
		} else {
			i.progress.SkippedRecords++
		}

		recordNum++
		i.progress.TotalRecords = recordNum

		// Process batch when full
		if len(batch) >= i.config.BatchSize {
			if err := i.insertFinancialBatch(batch); err != nil {
				log.Printf("Failed to insert financial batch: %v", err)
			}
			batch = batch[:0] // Reset batch
		}

		// Log progress
		if recordNum%i.config.ProgressInterval == 0 {
			i.logProgress()
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := i.insertFinancialBatch(batch); err != nil {
			log.Printf("Failed to insert final financial batch: %v", err)
		}
	}

	i.logFinalStats("Financial data import")
	return nil
}

// insertCharityBatch inserts a batch of charity records
func (i *Importer) insertCharityBatch(records []CharityRecord) error {
	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO charities
		(organisation_number, registered_number, linked_charity_number, company_number, 
		 name, status, date_registered, date_removed, 
		 address, website, email, phone, what_the_charity_does, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// Only import active or registered charities (optional filter)
		// Skip if already registered charity number is 0 (invalid)
		if record.RegisteredCharityNumber == 0 {
			i.progress.SkippedRecords++
			continue
		}

		// Build address string
		address := buildAddress(
			record.CharityContactAddress1,
			record.CharityContactAddress2,
			record.CharityContactAddress3,
			record.CharityContactAddress4,
			record.CharityContactAddress5,
			record.CharityContactPostcode,
		)

		// Parse dates
		dateRegistered := parseDate(record.DateOfRegistration)
		var dateRemoved *time.Time
		if record.DateOfRemoval != nil {
			dr := parseDate(*record.DateOfRemoval)
			dateRemoved = &dr
		}

		// Execute insert
		_, err := stmt.Exec(
			record.OrganisationNumber,
			record.RegisteredCharityNumber,
			record.LinkedCharityNumber,
			record.CharityCompanyRegistrationNumber,
			record.CharityName,
			record.CharityRegistrationStatus,
			dateRegistered,
			dateRemoved,
			address,
			record.CharityContactWeb,
			record.CharityContactEmail,
			record.CharityContactPhone,
			record.CharityActivities,
			time.Now(),
		)
		if err != nil {
			if i.config.Verbose {
				log.Printf("Failed to insert charity %d: %v", record.RegisteredCharityNumber, err)
			}
			i.progress.FailedRecords++
			continue
		}

		i.progress.SuccessRecords++

		// Also insert financial data if available
		if record.LatestIncome != nil && record.LatestExpenditure != nil {
			i.insertFinancialData(tx, record)
		}
	}

	i.progress.ProcessedRecords += len(records)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// insertTrusteeBatch inserts a batch of trustee records
func (i *Importer) insertTrusteeBatch(records []TrusteeRecord) error {
	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO trustees
		(charity_number, name, last_updated)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// Skip invalid records
		if record.RegisteredCharityNumber == 0 || record.TrusteeName == "" {
			i.progress.SkippedRecords++
			continue
		}

		_, err := stmt.Exec(
			record.RegisteredCharityNumber,
			record.TrusteeName,
			time.Now(),
		)
		if err != nil {
			if i.config.Verbose {
				log.Printf("Failed to insert trustee for charity %d: %v", record.RegisteredCharityNumber, err)
			}
			i.progress.FailedRecords++
			continue
		}

		i.progress.SuccessRecords++
	}

	i.progress.ProcessedRecords += len(records)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// insertFinancialBatch inserts a batch of financial records from annual return partb
func (i *Importer) insertFinancialBatch(records []AnnualReturnPartBRecord) error {
	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO financials
		(charity_number, financial_year_end, total_income, total_spending,
		 charitable_activities_spend, raising_funds_spend, other_spend,
		 reserves, assets, employees, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		// Skip invalid records
		if record.RegisteredCharityNumber == 0 {
			i.progress.SkippedRecords++
			continue
		}

		// Parse financial year end date
		yearEnd := parseDate(record.FinPeriodEndDate)
		if yearEnd.IsZero() {
			i.progress.SkippedRecords++
			continue
		}

		// Calculate other spend (governance + any other expenditure)
		otherSpend := 0.0
		if record.ExpenditureGovernance != nil {
			otherSpend = *record.ExpenditureGovernance
		}

		_, err := stmt.Exec(
			record.RegisteredCharityNumber,
			yearEnd,
			orDefaultPtr(record.IncomeTotalIncomeEndowments, 0),
			orDefaultPtr(record.ExpenditureTotal, 0),
			orDefaultPtr(record.ExpenditureCharitableExpend, 0),
			orDefaultPtr(record.ExpenditureRaisingFunds, 0),
			otherSpend,
			orDefaultPtr(record.Reserves, 0),
			orDefaultPtr(record.AssetsTotalAssetsLiabilities, 0),
			orDefaultPtrInt(record.CountEmployees, 0),
			time.Now(),
		)
		if err != nil {
			if i.config.Verbose {
				log.Printf("Failed to insert financial data for charity %d: %v", record.RegisteredCharityNumber, err)
			}
			i.progress.FailedRecords++
			continue
		}

		i.progress.SuccessRecords++
	}

	i.progress.ProcessedRecords += len(records)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ImportAnnualReturnHistory imports annual return history data from a file
func (i *Importer) ImportAnnualReturnHistory() error {
	if i.config.AnnualReturnHistoryFile == "" {
		log.Println("No annual return history file specified, skipping")
		return nil
	}

	log.Printf("Starting annual return history import from: %s", i.config.AnnualReturnHistoryFile)
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	file, err := os.Open(i.config.AnnualReturnHistoryFile)
	if err != nil {
		return fmt.Errorf("failed to open annual return history file: %w", err)
	}
	defer file.Close()

	reader := stripBOM(file)
	return i.importAnnualReturnHistoryFromReader(reader)
}

// ImportAnnualReturnHistoryFromReader imports annual return history data from an io.Reader
func (i *Importer) ImportAnnualReturnHistoryFromReader(r io.Reader) error {
	log.Println("Starting annual return history import from in-memory data")
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	reader := stripBOM(r)
	return i.importAnnualReturnHistoryFromReader(reader)
}

// importAnnualReturnHistoryFromReader is the internal implementation that works with any reader
func (i *Importer) importAnnualReturnHistoryFromReader(reader io.Reader) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read opening bracket: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected array opening bracket, got: %v", token)
	}

	batch := make([]AnnualReturnHistoryRecord, 0, i.config.BatchSize)
	recordNum := 0

	// Process array elements
	for decoder.More() {
		var record AnnualReturnHistoryRecord
		if err := decoder.Decode(&record); err != nil {
			log.Printf("Failed to decode annual return history record %d: %v", recordNum, err)
			i.progress.FailedRecords++
			continue
		}

		batch = append(batch, record)
		recordNum++
		i.progress.TotalRecords = recordNum

		// Process batch when full
		if len(batch) >= i.config.BatchSize {
			if err := i.insertAnnualReturnHistoryBatch(batch); err != nil {
				log.Printf("Failed to insert annual return history batch: %v", err)
			}
			batch = batch[:0] // Reset batch
		}

		// Log progress
		if recordNum%i.config.ProgressInterval == 0 {
			i.logProgress()
		}
	}

	// Process remaining records
	if len(batch) > 0 {
		if err := i.insertAnnualReturnHistoryBatch(batch); err != nil {
			log.Printf("Failed to insert final annual return history batch: %v", err)
		}
	}

	i.logFinalStats("Annual return history import")
	return nil
}

// insertAnnualReturnHistoryBatch inserts a batch of annual return history records
func (i *Importer) insertAnnualReturnHistoryBatch(records []AnnualReturnHistoryRecord) error {
	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO annual_return_history
		(organisation_number, registered_charity_number, fin_period_start_date, 
		 fin_period_end_date, ar_cycle_reference, reporting_due_date,
		 date_annual_return_received, date_accounts_received, total_gross_income,
		 total_gross_expenditure, accounts_qualified, suppression_ind, 
		 suppression_type, date_of_extract)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		var finStartDate, finEndDate, dueDate, arReceivedDate, accountsReceivedDate, extractDate interface{}

		if record.FinPeriodStartDate != nil {
			finStartDate = parseDate(*record.FinPeriodStartDate)
		}
		if record.FinPeriodEndDate != nil {
			finEndDate = parseDate(*record.FinPeriodEndDate)
		}
		if record.ReportingDueDate != nil {
			dueDate = parseDate(*record.ReportingDueDate)
		}
		if record.DateAnnualReturnReceived != nil {
			arReceivedDate = parseDate(*record.DateAnnualReturnReceived)
		}
		if record.DateAccountsReceived != nil {
			accountsReceivedDate = parseDate(*record.DateAccountsReceived)
		}
		extractDate = parseDate(record.DateOfExtract)

		_, err := stmt.Exec(
			record.OrganisationNumber,
			record.RegisteredCharityNumber,
			finStartDate,
			finEndDate,
			record.ARCycleReference,
			dueDate,
			arReceivedDate,
			accountsReceivedDate,
			record.TotalGrossIncome,
			record.TotalGrossExpenditure,
			record.AccountsQualified,
			record.SuppressionInd,
			record.SuppressionType,
			extractDate,
		)

		if err != nil {
			if i.config.Verbose {
				log.Printf("Failed to insert annual return history for charity %d: %v",
					record.RegisteredCharityNumber, err)
			}
			i.progress.FailedRecords++
			continue
		}

		i.progress.SuccessRecords++
	}

	i.progress.ProcessedRecords += len(records)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// insertFinancialData inserts financial data for a charity
func (i *Importer) insertFinancialData(tx *sql.Tx, record CharityRecord) {
	if record.LatestAccFinPeriodEndDate == nil {
		return
	}

	yearEnd := parseDate(*record.LatestAccFinPeriodEndDate)

	_, err := tx.Exec(`
		INSERT OR REPLACE INTO financials
		(charity_number, financial_year_end, total_income, total_spending, 
		 charitable_activities_spend, raising_funds_spend, other_spend, 
		 reserves, assets, trustees, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.RegisteredCharityNumber,
		yearEnd,
		orDefault(record.LatestIncome, 0),
		orDefault(record.LatestExpenditure, 0),
		0, // Not in the data dump - will be estimated or fetched from API
		0, // Not in the data dump
		0, // Not in the data dump
		0, // Not in the data dump
		0, // Not in the data dump
		0, // Not in the data dump
		time.Now(),
	)
	if err != nil && i.config.Verbose {
		log.Printf("Failed to insert financial data for charity %d: %v", record.RegisteredCharityNumber, err)
	}
}

// Helper functions

func buildAddress(parts ...*string) string {
	var address string
	for _, part := range parts {
		if part != nil && *part != "" {
			if address != "" {
				address += ", "
			}
			address += *part
		}
	}
	return address
}

func parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Try multiple date formats
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

func orDefault(val *float64, def float64) float64 {
	if val == nil {
		return def
	}
	return *val
}

func orDefaultPtr(val *float64, def float64) float64 {
	if val == nil {
		return def
	}
	return *val
}

func orDefaultPtrInt(val *int, def int) int {
	if val == nil {
		return def
	}
	return *val
}

func (i *Importer) logProgress() {
	elapsed := time.Since(i.progress.StartTime)
	rate := float64(i.progress.ProcessedRecords) / elapsed.Seconds()

	log.Printf("Progress: %d processed (%d success, %d failed, %d skipped) | Rate: %.2f/sec",
		i.progress.ProcessedRecords,
		i.progress.SuccessRecords,
		i.progress.FailedRecords,
		i.progress.SkippedRecords,
		rate,
	)

	i.progress.LastUpdate = time.Now()
}

func (i *Importer) logFinalStats(label string) {
	elapsed := time.Since(i.progress.StartTime)
	rate := float64(i.progress.ProcessedRecords) / elapsed.Seconds()

	log.Printf("\n=== %s Complete ===", label)
	log.Printf("Total Records: %d", i.progress.TotalRecords)
	log.Printf("Processed: %d", i.progress.ProcessedRecords)
	log.Printf("Successful: %d", i.progress.SuccessRecords)
	log.Printf("Failed: %d", i.progress.FailedRecords)
	log.Printf("Skipped: %d", i.progress.SkippedRecords)
	log.Printf("Time Elapsed: %v", elapsed)
	log.Printf("Average Rate: %.2f records/second\n", rate)
}

// StreamingImportCharities imports charities using a streaming approach for very large files
func (i *Importer) StreamingImportCharities() error {
	// This is the same as ImportCharities but documented as the streaming approach
	// The json.Decoder already streams, so we're good
	return i.ImportCharities()
}

// GetProgress returns the current import progress
func (i *Importer) GetProgress() ImportProgress {
	return i.progress
}

// CalculateAllScores calculates transparency scores for all charities in the database
func (i *Importer) CalculateAllScores() error {
	log.Println("Starting score calculation for all charities...")

	// Get count of charities that need scores (main charities only, exclude removed)
	var totalCharities int
	err := i.db.QueryRow(`
		SELECT COUNT(*) FROM charities c
		WHERE c.linked_charity_number = 0
		  AND c.status NOT IN ('Removed', 'RM')
		  AND NOT EXISTS (
			SELECT 1 FROM charity_scores s 
			WHERE s.charity_number = c.registered_number
		  )
	`).Scan(&totalCharities)
	if err != nil {
		return fmt.Errorf("failed to count charities: %w", err)
	}

	log.Printf("Found %d charities needing score calculation", totalCharities)

	if totalCharities == 0 {
		log.Println("All charities already have scores")
		return nil
	}

	// Reset progress tracker
	i.progress = ImportProgress{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	// Fetch ALL charity numbers upfront (before we start modifying the database)
	// This prevents issues with OFFSET pagination as we add scores
	log.Println("Fetching all charity numbers that need scoring...")
	rows, err := i.db.Query(`
		SELECT c.registered_number 
		FROM charities c
		WHERE c.linked_charity_number = 0
		  AND c.status NOT IN ('Removed', 'RM')
		  AND NOT EXISTS (
			SELECT 1 FROM charity_scores s 
			WHERE s.charity_number = c.registered_number
		  )
		ORDER BY c.registered_number
	`)
	if err != nil {
		return fmt.Errorf("failed to fetch charity numbers: %w", err)
	}

	var allCharityNumbers []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			log.Printf("Error scanning charity number: %v", err)
			continue
		}
		allCharityNumbers = append(allCharityNumbers, num)
	}
	rows.Close()

	log.Printf("Fetched %d charity numbers, starting score calculation...", len(allCharityNumbers))

	// Now process all the charity numbers we fetched
	for _, charityNum := range allCharityNumbers {
		// Use the shared scoring.CalculateScore function to ensure consistency
		_, err := scoring.CalculateScore(i.db, charityNum)
		if err != nil {
			if i.config.Verbose {
				log.Printf("Failed to calculate score for charity %d: %v", charityNum, err)
			}
			i.progress.FailedRecords++
		} else {
			i.progress.SuccessRecords++
		}

		i.progress.ProcessedRecords++

		// Log progress periodically
		if i.progress.ProcessedRecords%i.config.ProgressInterval == 0 {
			i.logProgress()
		}
	}

	i.logFinalStats("Score calculation")
	return nil
}
