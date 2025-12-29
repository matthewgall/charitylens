package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseType      string
	DatabaseURL       string
	Port              string
	BindIP            string
	CharityAPIKey     string
	AdminAPIKey       string
	SyncIntervalHours int
	EnableSyncWorker  bool
	OfflineMode       bool
	Debug             bool
}

func Load() *Config {
	cfg := &Config{
		DatabaseType:      getEnv("DATABASE_TYPE", "sqlite"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		Port:              getEnv("PORT", "8080"),
		BindIP:            getEnv("IP", "0.0.0.0"),
		CharityAPIKey:     getEnv("CHARITY_API_KEY", ""),
		AdminAPIKey:       getEnv("ADMIN_API_KEY", ""),
		SyncIntervalHours: getEnvInt("SYNC_INTERVAL_HOURS", 24),
		EnableSyncWorker:  getEnvBool("ENABLE_SYNC_WORKER", false),
		OfflineMode:       getEnvBool("OFFLINE_MODE", false),
		Debug:             getEnvBool("DEBUG", false),
	}

	// Set defaults for database
	if cfg.DatabaseURL == "" {
		if cfg.DatabaseType == "sqlite" {
			cfg.DatabaseURL = "charitylens.db"
		}
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
