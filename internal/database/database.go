package database

import (
	"context"
	"time"

	"biz-credit-ai/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Сериализуем миграции нескольких одновременно стартующих экземпляров.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", 2026102101).Error; err != nil {
			return err
		}
		return tx.AutoMigrate(&models.Company{}, &models.User{}, &models.Application{}, &models.NewsArticle{})
	})
}
