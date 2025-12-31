package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	dbType := os.Getenv("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	offlineMode := os.Getenv("OFFLINE_MODE") == "true"

	var driverName, dataSourceName string

	switch dbType {
	case "sqlite":
		driverName = "sqlite3"
		dataSourceName = os.Getenv("DATABASE_URL")
		if dataSourceName == "" {
			dataSourceName = "charitylens.db"
		}
		// Add SQLite performance pragmas for read-heavy workload
		// cache=shared allows multiple connections to share cache
		// In offline mode, use read-only mode for maximum performance and safety
		// In online mode, use WAL for write-ahead logging (better concurrency)
		if offlineMode {
			dataSourceName += "?cache=shared&mode=ro"
		} else {
			dataSourceName += "?cache=shared&_journal_mode=WAL"
		}
	case "mysql":
		driverName = "mysql"
		dataSourceName = os.Getenv("DATABASE_URL")
	case "postgres":
		driverName = "postgres"
		dataSourceName = os.Getenv("DATABASE_URL")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for better performance
	// These settings are optimized for read-heavy workloads on fly.io
	db.SetMaxOpenConns(25)   // Limit concurrent connections
	db.SetMaxIdleConns(5)    // Keep some connections ready
	db.SetConnMaxLifetime(0) // No max lifetime (reuse connections)
	db.SetConnMaxIdleTime(0) // No max idle time for SQLite file access

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	return MigrateWithPath(db, "migrations")
}

func MigrateWithPath(db *sql.DB, migrationsPath string) error {
	dbType := os.Getenv("DATABASE_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	var driver database.Driver
	var err error

	switch dbType {
	case "sqlite":
		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
	case "mysql":
		driver, err = mysql.WithInstance(db, &mysql.Config{})
	case "postgres":
		driver, err = postgres.WithInstance(db, &postgres.Config{})
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		dbType,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %v", err)
	}

	// Get current version to check if migrations are needed
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %v", err)
	}

	// Only run migrations if needed (optimizes startup on fly.io)
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	// Log migration status for debugging
	newVersion, _, _ := m.Version()
	if dirty {
		return fmt.Errorf("database is in dirty state, manual intervention required")
	}
	if version != newVersion && newVersion > 0 {
		// Migrations were applied
		_ = version // Used for logging in production
	}

	return nil
}
