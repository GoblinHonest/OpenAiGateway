package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Provider{},
		&domain.Token{},
		&domain.Model{},
		&domain.ModelProviderBinding{},
		&domain.Group{},
		&domain.GroupModel{},
		&domain.RequestLog{},
		&domain.APIKey{},
		&domain.ProviderHealthCheck{},
		&domain.CircuitBreakerState{},
		&domain.AdminAuditLog{},
		&domain.ReconciliationRecord{},
	)
}

// RunMigrations executes SQL migration files from the migrations directory
func RunMigrations(db *gorm.DB, migrationsDir string, driver string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create migrations tracking table if not exists
	if err := createMigrationsTable(sqlDB); err != nil {
		return err
	}

	files, err := getMigrationFiles(migrationsDir, driver, "up")
	if err != nil {
		return err
	}

	for _, file := range files {
		version := extractVersion(file)
		if isApplied(sqlDB, version) {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		if err := executeMigration(sqlDB, string(content), version); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		fmt.Printf("Applied migration: %s\n", file)
	}

	return nil
}

// RollbackMigrations rolls back the last N migration steps
func RollbackMigrations(db *gorm.DB, migrationsDir string, driver string, steps int) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if err := createMigrationsTable(sqlDB); err != nil {
		return err
	}

	if steps == 0 {
		steps = 1
	}

	appliedVersions := getAppliedVersions(sqlDB)
	if len(appliedVersions) == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	// Rollback in reverse order
	rolledBack := 0
	for i := len(appliedVersions) - 1; i >= 0 && rolledBack < steps; i-- {
		version := appliedVersions[i]
		downFile := findDownMigration(migrationsDir, driver, version)
		if downFile == "" {
			fmt.Printf("No down migration found for version %s, skipping\n", version)
			continue
		}

		content, err := os.ReadFile(downFile)
		if err != nil {
			return fmt.Errorf("failed to read down migration %s: %w", downFile, err)
		}

		if err := executeRollback(sqlDB, string(content), version); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", version, err)
		}

		fmt.Printf("Rolled back migration: %s\n", version)
		rolledBack++
	}

	if rolledBack == 0 {
		fmt.Println("No migrations were rolled back")
	}

	return nil
}

func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) PRIMARY KEY,
		description VARCHAR(255),
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64)
	)`)
	return err
}

func getMigrationFiles(dir, driver, direction string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	suffix := fmt.Sprintf("_%s.sql", driver)
	if direction == "down" {
		suffix = fmt.Sprintf("_down_%s.sql", driver)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, suffix) && !strings.Contains(name, "_down_") {
			files = append(files, name)
		}
		// Also include generic .sql files (not driver-specific)
		if direction == "up" && strings.HasSuffix(name, ".sql") && !strings.Contains(name, "_sqlite.sql") && !strings.Contains(name, "_mysql.sql") && !strings.Contains(name, "_postgres.sql") && !strings.Contains(name, "_down_") {
			// Skip generic files if driver-specific exists
		}
	}

	sort.Strings(files)
	return files, nil
}

func findDownMigration(dir, driver, version string) string {
	// Look for down migration files
	patterns := []string{
		fmt.Sprintf("%s_down_%s.sql", version, driver),
		fmt.Sprintf("%s_down.sql", version),
	}

	for _, pattern := range patterns {
		path := filepath.Join(dir, pattern)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func extractVersion(filename string) string {
	// Extract version from filename like "001_init_schema.sql" -> "001"
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return filename
}

func isApplied(db *sql.DB, version string) bool {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	return count > 0
}

func getAppliedVersions(db *sql.DB) []string {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var v string
		rows.Scan(&v)
		versions = append(versions, v)
	}
	return versions
}

func executeMigration(db *sql.DB, content, version string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(content); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func executeRollback(db *sql.DB, content, version string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(content); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
