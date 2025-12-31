package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charitylens/internal/config"
	"charitylens/internal/database"
	"charitylens/internal/handlers"
	"charitylens/internal/logger"
	custommiddleware "charitylens/internal/middleware"
	"charitylens/internal/sync"
	"charitylens/internal/version"
	"charitylens/web/static"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Command line flags
	port := flag.String("port", "", "Port to bind to (overrides PORT env var)")
	ip := flag.String("ip", "", "IP address to bind to (overrides IP env var)")
	apiKey := flag.String("api-key", "", "Charity Commission API key (overrides CHARITY_API_KEY env var)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	offline := flag.Bool("offline", false, "Run in offline mode (no API calls, uses pre-seeded database)")
	flag.Parse()

	// Set environment variables from flags
	if *port != "" {
		os.Setenv("PORT", *port)
	}
	if *ip != "" {
		os.Setenv("IP", *ip)
	}
	if *apiKey != "" {
		os.Setenv("CHARITY_API_KEY", *apiKey)
	}
	if *debug {
		os.Setenv("DEBUG", "true")
	}
	if *offline {
		os.Setenv("OFFLINE_MODE", "true")
	}

	cfg := config.Load()

	// Initialize logger
	logger.WithDebug(cfg.Debug)

	// Log version info
	logger.Info("Starting CharityLens", "version", version.GetVersion(), "user_agent", version.UserAgent())

	// Log offline mode status
	if cfg.OfflineMode {
		logger.Info("Running in OFFLINE MODE - all data will be served from the database, no API calls will be made")
	}

	// Require API key only if not in offline mode
	if !cfg.OfflineMode && cfg.CharityAPIKey == "" {
		logger.Error("Charity Commission API key is required. Set CHARITY_API_KEY environment variable, use -api-key flag, or run with -offline flag.")
		os.Exit(1)
	}

	// Create router early for health checks
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Add a simple health check endpoint that responds immediately
	readyChan := make(chan bool, 1)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-readyChan:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Initializing..."))
		}
	})

	// Create and start server immediately
	addr := cfg.BindIP + ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Initialize database in background
	go func() {
		logger.Info("Initializing database...")
		db, err := database.InitDB()
		if err != nil {
			logger.Error("Failed to initialize database", "error", err)
			os.Exit(1)
		}

		// Run migrations only if not in offline mode
		// In offline mode, we use a pre-seeded database that already has the correct schema
		if !cfg.OfflineMode {
			logger.Info("Checking migrations...")
			if err := database.Migrate(db); err != nil {
				logger.Error("Failed to run migrations", "error", err)
				os.Exit(1)
			}
		} else {
			logger.Info("Skipping migrations (offline mode - using pre-seeded database)")
		}

		// Warm up the database with a simple query to ensure it's ready
		// This prevents the first user request from being slow
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM charities LIMIT 1").Scan(&count); err != nil {
			logger.Error("Failed to warm up database", "error", err)
			// Don't exit - this is not critical
		}

		logger.Info("Database ready")

		// Initialize handlers
		charityHandler := handlers.NewCharityHandler(db, cfg)
		webHandler := handlers.NewWebHandler(db, cfg)

		// Static files (embedded)
		staticFS := http.FS(static.FS())
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

		// Web Routes
		r.Get("/", webHandler.SearchPage)
		r.Get("/charity/{id}", webHandler.CharityPage)
		r.Get("/compare", webHandler.ComparePage)
		r.Get("/license", webHandler.LicensePage)
		r.Get("/methodology", webHandler.MethodologyPage)

		// API Routes with CORS
		r.Route("/api", func(r chi.Router) {
			// Add CORS for API routes
			r.Use(custommiddleware.CORS([]string{"*"})) // Allow all origins for API
			r.Use(custommiddleware.Timeout(30 * time.Second))

			r.Get("/charities/search", charityHandler.SearchCharities)
			r.Get("/charities/{number}", charityHandler.GetCharity)
			r.Get("/charities/compare", charityHandler.CompareCharities)
			r.Post("/admin/sync", charityHandler.SyncData)
		})

		// Start sync worker if enabled
		if cfg.EnableSyncWorker {
			logger.Info("Starting background sync worker")
			go sync.StartSyncWorker(cfg, db)
		} else {
			logger.Info("Background sync worker disabled (using sync-on-demand)")
		}

		// Signal that the app is ready
		readyChan <- true
		logger.Info("Application ready to serve requests")
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Gracefully shutdown with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server gracefully stopped")
}
