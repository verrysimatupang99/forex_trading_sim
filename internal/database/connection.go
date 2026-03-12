package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"forex-trading-sim/config"
	"forex-trading-sim/internal/models"
)

// Connect establishes database connection
func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return db, nil
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.CurrencyPair{},
		&models.HistoricalPrice{},
		&models.Trade{},
		&models.Position{},
		&models.MLModel{},
		&models.Prediction{},
		&models.Backtest{},
		&models.WalkForwardAnalysis{},
		&models.PendingOrder{},
		&models.OCOOrder{},
		&models.OTOOrder{},
		&models.CurrencyRate{},
		&models.MultiCurrencyAccount{},
		&models.CurrencyBalance{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
