package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/aigateway/internal/config"
	"github.com/example/aigateway/internal/repository"
	"github.com/example/aigateway/pkg/logger"
	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "config/config.yaml", "Configuration file path")
	direction  = flag.String("direction", "up", "Migration direction: up or down")
	steps      = flag.Int("steps", 0, "Number of migration steps (0 = all)")
)

func main() {
	flag.Parse()

	if flag.NArg() > 0 {
		*direction = flag.Arg(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		logger.L.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	migrationsDir := filepath.Join("migrations")

	switch *direction {
	case "up":
		logger.L.Info("Running migrations up")
		if err := repository.RunMigrations(db, migrationsDir, cfg.Database.Driver); err != nil {
			logger.L.Fatal("Migration failed", zap.Error(err))
			os.Exit(1)
		}
		logger.L.Info("Migrations completed successfully")
	case "down":
		logger.L.Info("Running migrations down")
		if err := repository.RollbackMigrations(db, migrationsDir, cfg.Database.Driver, *steps); err != nil {
			logger.L.Fatal("Rollback failed", zap.Error(err))
			os.Exit(1)
		}
		logger.L.Info("Rollback completed successfully")
	default:
		fmt.Printf("Unknown direction: %s. Use 'up' or 'down'\n", *direction)
		os.Exit(1)
	}
}
