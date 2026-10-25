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
		// Drop the old constraint before widening the column and translating legacy statuses.
		if tx.Migrator().HasTable(&models.Application{}) {
			if err := tx.Exec("ALTER TABLE applications DROP CONSTRAINT IF EXISTS chk_applications_status").Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE applications ALTER COLUMN status TYPE varchar(32)").Error; err != nil {
				return err
			}
			if err := tx.Exec("UPDATE applications SET status = CASE status WHEN 'SUBMITTED' THEN 'DATA_ENRICHMENT' WHEN 'SCORING' THEN 'SCORING_IN_PROGRESS' ELSE status END").Error; err != nil {
				return err
			}
		}
		if err := tx.AutoMigrate(&models.Company{}, &models.User{}, &models.Application{}, &models.NewsArticle{}, &models.ApplicationDocument{}, &models.ApplicationStatusLog{}); err != nil {
			return err
		}
		if err := tx.Exec("UPDATE applications SET requested_amount = amount, requested_term_months = term_months WHERE requested_amount = 0").Error; err != nil {
			return err
		}
		// Upgrade known demo fixtures even when the database was seeded before this release.
		return tx.Exec(`UPDATE companies SET monthly_revenue = revenue / 12,
   monthly_net_profit = CASE inn WHEN '7701000001' THEN 600000 WHEN '770100000002' THEN 180000 ELSE 10000 END,
   age_months = CASE inn WHEN '7701000001' THEN 102 WHEN '770100000002' THEN 62 ELSE 1 END,
   bki_monthly_payment = CASE inn WHEN '7701000001' THEN 50000 WHEN '770100000002' THEN 40000 ELSE 0 END
   WHERE inn IN ('7701000001','770100000002','770100000003') AND monthly_revenue = 0`).Error
	})
}
