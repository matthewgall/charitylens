package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"charitylens/internal/api"
	"charitylens/internal/database"
	"charitylens/internal/downloader"
	"charitylens/internal/importer"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

const (
	defaultRateLimit   = 10  // requests per second
	defaultConcurrency = 5   // concurrent workers
	defaultMaxRetries  = 5   // max retry attempts
	checkpointInterval = 100 // Save progress every N charities
)

type Config struct {
	Mode           string   // "api" or "file"
	APIKeys        []string // Multiple API keys for load balancing
	CharityFile    string   // Path to charity JSON file (for file mode)
	TrusteeFile    string   // Path to trustee JSON file (for file mode)
	FinancialFile  string   // Path to annual return partb JSON file (for file mode)
	DBPath         string
	MigrationsPath string
	RateLimit      int
	Concurrency    int
	MaxRetries     int
	StartCharity   int
	EndCharity     int
	ResumeFrom     int
	BatchSize      int // For file imports
	Verbose        bool
}

type Scraper struct {
	config      *Config
	db          *sql.DB
	apiClient   *api.Client
	stats       *Stats
	ctx         context.Context
	cancel      context.CancelFunc
	progressBar *progressbar.ProgressBar
}

type Stats struct {
	mu             sync.Mutex
	TotalProcessed int
	Successful     int
	Failed         int
	Skipped        int
	StartTime      time.Time
	LastCheckpoint time.Time
	CurrentCharity int
}

func main() {
	config := parseFlags()

	if err := run(config); err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func parseFlags() *Config {
	config := &Config{}

	var apiKeysStr string
	flag.StringVar(&config.Mode, "mode", "api", "Import mode: 'api' (scrape from API), 'file' (import from JSON files), 'download' (download and import in-memory), or 'score' (calculate scores for existing charities)")
	flag.StringVar(&apiKeysStr, "api-keys", os.Getenv("CHARITY_API_KEYS"), "Comma-separated list of API keys for load balancing (or set CHARITY_API_KEYS env var)")
	flag.StringVar(&config.CharityFile, "charity-file", "publicextract.charity.json", "Path to charity JSON file (file mode only)")
	flag.StringVar(&config.TrusteeFile, "trustee-file", "publicextract.charity_trustee.json", "Path to trustee JSON file (file mode only)")
	flag.StringVar(&config.FinancialFile, "financial-file", "publicextract.charity_annual_return_partb.json", "Path to annual return partb JSON file (file mode only)")
	flag.StringVar(&config.DBPath, "db", "seed.db", "Path to SQLite database file")
	flag.StringVar(&config.MigrationsPath, "migrations", "../../migrations", "Path to migrations directory")
	flag.IntVar(&config.RateLimit, "rate-limit", defaultRateLimit, "Maximum requests per second (API mode only)")
	flag.IntVar(&config.Concurrency, "concurrency", defaultConcurrency, "Number of concurrent workers (API mode only)")
	flag.IntVar(&config.MaxRetries, "max-retries", defaultMaxRetries, "Maximum retry attempts for failed requests (API mode only)")
	flag.IntVar(&config.StartCharity, "start", 1, "Starting charity number (API mode only)")
	flag.IntVar(&config.EndCharity, "end", 999999, "Ending charity number (API mode only)")
	flag.IntVar(&config.ResumeFrom, "resume", 0, "Resume from specific charity number (API mode only, overrides checkpoint)")
	flag.IntVar(&config.BatchSize, "batch-size", 1000, "Batch size for file imports (file mode only)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()

	// Validate mode
	if config.Mode != "api" && config.Mode != "file" && config.Mode != "download" && config.Mode != "score" {
		log.Fatalf("Invalid mode: %s (must be 'api', 'file', 'download', or 'score')", config.Mode)
	}

	// Mode-specific validation
	if config.Mode == "api" {
		// Parse API keys (comma-separated)
		if apiKeysStr != "" {
			keys := strings.Split(apiKeysStr, ",")
			for _, key := range keys {
				trimmed := strings.TrimSpace(key)
				if trimmed != "" {
					config.APIKeys = append(config.APIKeys, trimmed)
				}
			}
		}

		// Also check for single key in CHARITY_API_KEY for backwards compatibility
		if len(config.APIKeys) == 0 {
			singleKey := os.Getenv("CHARITY_API_KEY")
			if singleKey != "" {
				config.APIKeys = []string{singleKey}
			}
		}

		if len(config.APIKeys) == 0 {
			log.Fatal("At least one API key is required for API mode. Set via -api-keys flag, CHARITY_API_KEYS, or CHARITY_API_KEY environment variable")
		}

		if len(config.APIKeys) > 1 {
			log.Printf("Using %d API keys for load balancing", len(config.APIKeys))
		}
	} else if config.Mode == "file" {
		// Validate file paths (all three required for complete data)
		if _, err := os.Stat(config.CharityFile); os.IsNotExist(err) {
			log.Fatalf("Charity file not found: %s", config.CharityFile)
		}
		if _, err := os.Stat(config.TrusteeFile); os.IsNotExist(err) {
			log.Fatalf("Trustee file not found: %s", config.TrusteeFile)
		}
		// Financial file is optional but recommended
		if _, err := os.Stat(config.FinancialFile); os.IsNotExist(err) {
			log.Printf("Warning: Financial file not found: %s (detailed financial data will not be available for scoring)", config.FinancialFile)
			config.FinancialFile = "" // Clear it so importer knows to skip
		}
		log.Printf("File mode: importing from charity, trustee, and financial files")
	}

	return config
}

func run(config *Config) error {
	// Initialize database
	db, err := initDatabase(config.DBPath, config.MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Branch based on mode
	if config.Mode == "file" {
		return runFileImport(config, db)
	} else if config.Mode == "download" {
		return runDownloadImport(config, db)
	} else if config.Mode == "score" {
		return runScoreCalculation(config, db)
	}
	return runAPIScrape(config, db)
}

func runScoreCalculation(config *Config, db *sql.DB) error {
	log.Println("=== Score Calculation Mode ===")
	log.Println("Calculating scores for charities without scores...")

	// Create importer just to use its CalculateAllScores method
	imp := importer.NewImporter(db, importer.ImportConfig{
		BatchSize:        config.BatchSize,
		ProgressInterval: 5000,
		Verbose:          config.Verbose,
	})

	if err := imp.CalculateAllScores(); err != nil {
		return fmt.Errorf("failed to calculate scores: %w", err)
	}

	log.Println("\n=== Score Calculation Complete ===")
	return nil
}

func runFileImport(config *Config, db *sql.DB) error {
	log.Println("=== File Import Mode ===")
	log.Printf("Charity file: %s", config.CharityFile)
	log.Printf("Trustee file: %s", config.TrusteeFile)
	if config.FinancialFile != "" {
		log.Printf("Financial file: %s", config.FinancialFile)
	}
	log.Printf("Batch size: %d\n", config.BatchSize)

	// Create importer
	imp := importer.NewImporter(db, importer.ImportConfig{
		CharityFile:      config.CharityFile,
		TrusteeFile:      config.TrusteeFile,
		FinancialFile:    config.FinancialFile,
		BatchSize:        config.BatchSize,
		ProgressInterval: 5000,
		Verbose:          config.Verbose,
	})

	// Import charities first
	log.Println("\n[1/3] Importing charities...")
	if err := imp.ImportCharities(); err != nil {
		return fmt.Errorf("failed to import charities: %w", err)
	}

	// Then import trustees
	log.Println("\n[2/3] Importing trustees...")
	if err := imp.ImportTrustees(); err != nil {
		return fmt.Errorf("failed to import trustees: %w", err)
	}

	// Finally import detailed financials
	log.Println("\n[3/4] Importing detailed financial data...")
	if err := imp.ImportFinancials(); err != nil {
		return fmt.Errorf("failed to import financial data: %w", err)
	}

	// Calculate scores for all imported charities
	log.Println("\n[4/4] Calculating scores for all charities...")
	if err := imp.CalculateAllScores(); err != nil {
		log.Printf("Warning: Failed to calculate all scores: %v (import was successful)", err)
	}

	log.Println("\n=== File Import Complete ===")
	return nil
}

func runDownloadImport(config *Config, db *sql.DB) error {
	ctx := context.Background()

	log.Println("=== Download Import Mode ===")
	log.Println("Downloading Charity Commission data files...")

	// Create downloader with progress tracking
	dl := downloader.NewDownloader(downloader.Config{
		Timeout:    15 * time.Minute,
		MaxRetries: 3,
		RetryDelay: 10 * time.Second,
		ProgressHandler: func(fileType downloader.FileType, bytesDownloaded, totalBytes int64) {
			if totalBytes > 0 {
				pct := float64(bytesDownloaded) / float64(totalBytes) * 100
				if int(pct)%10 == 0 && bytesDownloaded < totalBytes {
					log.Printf("  %s: %.1f%% (%d/%d MB)", fileType, pct,
						bytesDownloaded/1024/1024, totalBytes/1024/1024)
				}
			}
		},
	})

	// Download all required files in parallel
	files, err := dl.DownloadFiles(ctx, downloader.DefaultFileSet())
	if err != nil {
		return fmt.Errorf("failed to download files: %w", err)
	}

	log.Printf("\nAll files downloaded successfully!")
	log.Printf("Total data size: %.2f MB\n", float64(calculateTotalSize(files))/1024.0/1024.0)

	// Create importer
	imp := importer.NewImporter(db, importer.ImportConfig{
		BatchSize:        config.BatchSize,
		ProgressInterval: 5000,
		Verbose:          config.Verbose,
	})

	// Import charities from in-memory data
	log.Println("[1/4] Importing charities from downloaded data...")
	if charityFile, ok := files[downloader.FileCharity]; ok {
		if err := imp.ImportCharitiesFromReader(charityFile.GetReader()); err != nil {
			return fmt.Errorf("failed to import charities: %w", err)
		}
	} else {
		return fmt.Errorf("charity file not downloaded")
	}

	// Import trustees from in-memory data
	log.Println("\n[2/4] Importing trustees from downloaded data...")
	if trusteeFile, ok := files[downloader.FileCharityTrustee]; ok {
		if err := imp.ImportTrusteesFromReader(trusteeFile.GetReader()); err != nil {
			return fmt.Errorf("failed to import trustees: %w", err)
		}
	} else {
		return fmt.Errorf("trustee file not downloaded")
	}

	// Import financial data from in-memory data
	log.Println("\n[3/4] Importing financial data from downloaded data...")
	if financialFile, ok := files[downloader.FileCharityAnnualReturnB]; ok {
		if err := imp.ImportFinancialsFromReader(financialFile.GetReader()); err != nil {
			return fmt.Errorf("failed to import financials: %w", err)
		}
	} else {
		log.Println("Warning: Financial file not downloaded, skipping detailed financial data")
	}

	// Calculate scores
	log.Println("\n[4/4] Calculating scores for all charities...")
	if err := imp.CalculateAllScores(); err != nil {
		log.Printf("Warning: Failed to calculate all scores: %v (import was successful)", err)
	}

	log.Println("\n=== Download Import Complete ===")
	return nil
}

func calculateTotalSize(files map[downloader.FileType]*downloader.DownloadedFile) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}

func runAPIScrape(config *Config, db *sql.DB) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

	// Create API client with multiple keys
	rateLimiter := api.NewRateLimiter(config.RateLimit)
	apiClient := api.NewClient(api.ClientConfig{
		APIKeys:     config.APIKeys,
		UserAgent:   "CharityLens-Seeder/1.0 (Charity Transparency Tool)",
		RateLimiter: rateLimiter,
		MaxRetries:  config.MaxRetries,
		Verbose:     config.Verbose,
	})

	// Determine starting point
	startCharity := config.StartCharity
	if config.ResumeFrom > 0 {
		startCharity = config.ResumeFrom
		log.Printf("Resuming from charity number: %d", config.ResumeFrom)
	} else if checkpoint, err := loadCheckpoint(db); err == nil && checkpoint > 0 {
		// Only use checkpoint if it's within the requested range
		if checkpoint >= config.StartCharity && checkpoint <= config.EndCharity {
			startCharity = checkpoint
			log.Printf("Resuming from checkpoint: %d", checkpoint)
		}
	}

	// Calculate total work for progress bar
	totalCharities := config.EndCharity - startCharity + 1

	// Create progress bar
	bar := progressbar.NewOptions(totalCharities,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("[cyan][1/3][reset] Scraping charities..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("charities"),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	// Create scraper
	scraper := &Scraper{
		config:      config,
		db:          db,
		apiClient:   apiClient,
		progressBar: bar,
		stats: &Stats{
			StartTime:      time.Now(),
			LastCheckpoint: time.Now(),
			CurrentCharity: startCharity,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Run the scraper (progress bar will show real-time updates)
	return scraper.scrape()
}

func initDatabase(dbPath, migrationsPath string) (*sql.DB, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Verify migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory not found: %s", migrationsPath)
	}

	// Save and restore original environment
	origDBType := os.Getenv("DATABASE_TYPE")
	origDBURL := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("DATABASE_TYPE", origDBType)
		os.Setenv("DATABASE_URL", origDBURL)
	}()

	// Set environment for database initialization
	os.Setenv("DATABASE_TYPE", "sqlite")
	os.Setenv("DATABASE_URL", dbPath)

	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run migrations with specified path
	if err := database.MigrateWithPath(db, migrationsPath); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func loadCheckpoint(db *sql.DB) (int, error) {
	var checkpoint int
	err := db.QueryRow("SELECT last_charity_number FROM scraper_checkpoints WHERE id = 1").Scan(&checkpoint)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return checkpoint, err
}

func saveCheckpoint(db *sql.DB, charityNumber int) error {
	_, err := db.Exec(`
		INSERT INTO scraper_checkpoints (id, last_charity_number, updated_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET 
			last_charity_number = excluded.last_charity_number,
			updated_at = CURRENT_TIMESTAMP
	`, charityNumber)
	return err
}

func (s *Scraper) scrape() error {
	fmt.Printf("\n")
	fmt.Printf("🔍 Starting scraper\n")
	fmt.Printf("   Range: %d to %d\n", s.stats.CurrentCharity, s.config.EndCharity)
	fmt.Printf("   Rate limit: %d req/s\n", s.config.RateLimit)
	fmt.Printf("   Workers: %d\n", s.config.Concurrency)
	fmt.Printf("   API keys: %d\n\n", len(s.config.APIKeys))

	// Create work queue
	workQueue := make(chan int, s.config.Concurrency*2)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.worker(workerID, workQueue)
		}(i)
	}

	// Feed work to queue
	go func() {
		defer close(workQueue)
		for charityNum := s.stats.CurrentCharity; charityNum <= s.config.EndCharity; charityNum++ {
			select {
			case <-s.ctx.Done():
				return
			case workQueue <- charityNum:
				s.stats.mu.Lock()
				s.stats.CurrentCharity = charityNum
				s.stats.mu.Unlock()

				// Save checkpoint periodically
				if charityNum%checkpointInterval == 0 {
					if err := saveCheckpoint(s.db, charityNum); err != nil {
						log.Printf("Failed to save checkpoint: %v", err)
					}
				}
			}
		}
	}()

	// Wait for all workers to finish
	wg.Wait()

	// Ensure progress bar is finished
	s.progressBar.Finish()

	// Final checkpoint
	if err := saveCheckpoint(s.db, s.stats.CurrentCharity); err != nil {
		log.Printf("Failed to save final checkpoint: %v", err)
	}

	s.printFinalStats()
	return nil
}

func (s *Scraper) worker(workerID int, workQueue <-chan int) {
	for charityNum := range workQueue {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if err := s.processCharity(charityNum); err != nil {
			if s.config.Verbose {
				log.Printf("Worker %d: Failed to process charity %d: %v", workerID, charityNum, err)
			}
			s.stats.mu.Lock()
			s.stats.Failed++
			s.stats.mu.Unlock()
		} else {
			s.stats.mu.Lock()
			s.stats.Successful++
			s.stats.mu.Unlock()
		}

		s.stats.mu.Lock()
		s.stats.TotalProcessed++
		s.stats.mu.Unlock()

		// Update progress bar
		s.progressBar.Add(1)
	}
}

func (s *Scraper) processCharity(charityNum int) error {
	// Check if charity already exists
	var exists bool
	err := s.db.QueryRow("SELECT 1 FROM charities WHERE registered_number = ?", charityNum).Scan(&exists)
	if err == nil {
		s.stats.mu.Lock()
		s.stats.Skipped++
		s.stats.mu.Unlock()
		return nil
	}

	// Fetch charity data using shared API client
	data, err := s.apiClient.FetchCharityDetails(s.ctx, charityNum)
	if err != nil {
		// 404 is expected for non-existent charity numbers
		if err.Error() == "not found (404)" {
			s.stats.mu.Lock()
			s.stats.Skipped++
			s.stats.mu.Unlock()
			return nil
		}
		return err
	}

	// Store data in database
	return s.storeCharity(data, charityNum)
}

func (s *Scraper) storeCharity(data map[string]any, charityNum int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Parse charity data using shared parser
	charity, err := api.ParseCharityData(data, fmt.Sprintf("%d", charityNum))
	if err != nil {
		return fmt.Errorf("failed to parse charity: %w", err)
	}

	// Insert charity
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO charities
		(registered_number, company_number, name, status, date_registered, address, website, email, phone, what_the_charity_does, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, charity.RegisteredNumber, charity.CompanyNumber, charity.Name, charity.Status,
		charity.DateRegistered, charity.Address, charity.Website, charity.Email, charity.Phone,
		charity.WhatTheCharityDoes, charity.LastUpdated)
	if err != nil {
		return fmt.Errorf("failed to insert charity: %w", err)
	}

	// Parse and store financials using shared parser
	financial, err := api.ParseFinancialData(data, charityNum)
	if err == nil && (financial.TotalIncome > 0 || financial.TotalSpending > 0) {
		_, err = tx.Exec(`
			INSERT OR REPLACE INTO financials
			(charity_number, financial_year_end, total_income, total_spending, charitable_activities_spend,
			 raising_funds_spend, other_spend, reserves, assets, trustees, last_updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, financial.CharityNumber, financial.FinancialYearEnd, financial.TotalIncome, financial.TotalSpending,
			financial.CharitableActivitiesSpend, financial.RaisingFundsSpend, financial.OtherSpend,
			financial.Reserves, financial.Assets, financial.Trustees, financial.LastUpdated)
		if err != nil {
			return fmt.Errorf("failed to insert financial: %w", err)
		}
	}

	// Parse and store trustees using shared parser
	trustees := api.ParseTrusteesData(data, charityNum)
	for _, trustee := range trustees {
		_, err = tx.Exec(`
			INSERT OR REPLACE INTO trustees (charity_number, name, last_updated)
			VALUES (?, ?, ?)
		`, trustee.CharityNumber, trustee.Name, trustee.LastUpdated)
		if err != nil {
			return fmt.Errorf("failed to insert trustee: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Scraper) printFinalStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	elapsed := time.Since(s.stats.StartTime)
	log.Println("\n=== Final Statistics ===")
	log.Printf("Total Processed: %d", s.stats.TotalProcessed)
	log.Printf("Successful: %d", s.stats.Successful)
	log.Printf("Failed: %d", s.stats.Failed)
	log.Printf("Skipped: %d", s.stats.Skipped)
	log.Printf("Time Elapsed: %v", elapsed.Round(time.Second))
	log.Printf("Average Rate: %.2f charities/second", float64(s.stats.TotalProcessed)/elapsed.Seconds())
	log.Printf("Last Charity: %d", s.stats.CurrentCharity)

	// Print API key stats if multiple keys were used
	keyStats := s.apiClient.GetKeyStats()
	if len(keyStats) > 1 {
		log.Println("\n=== API Key Usage ===")
		for key := range keyStats {
			log.Printf("Key %s: %d requests, %d failures",
				key, keyStats[key].TotalRequests, keyStats[key].FailedRequests)
		}
	}
}
