package database

import (
	"context"
	"time"

	"alfa-credit/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedDatabase добавляет демо-набор атомарно, только если все четыре таблицы пусты.
// Повторный запуск и параллельный старт не создают дубликаты.
func SeedDatabase(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", 2026102102).Error; err != nil {
			return err
		}
		for _, model := range []any{&models.Company{}, &models.User{}, &models.Application{}, &models.NewsArticle{}} {
			var count int64
			if err := tx.Model(model).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return nil
			}
		}
		date := func(s string) *time.Time { t, _ := time.Parse("2006-01-02", s); return &t }
		companies := []models.Company{
			{INN: "7701000001", Name: `ООО "Стабильный Бизнес"`, OKVED: "62.01", Revenue: 24000000, RegistrationDate: date("2018-04-10")},
			{INN: "770100000002", Name: `ИП "Ритейл"`, OKVED: "47.11", Revenue: 12000000, RegistrationDate: date("2021-08-15")},
			{INN: "770100000003", Name: `ИП "Новый"`, OKVED: "62.01", Revenue: 600000, RegistrationDate: date("2026-09-01")},
		}
		if err := tx.Create(&companies).Error; err != nil {
			return err
		}
		for i, login := range []string{"client1", "client2", "client3", "officer1"} {
			hash, err := bcrypt.GenerateFromPassword([]byte("123"), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user := models.User{Email: login, PasswordHash: string(hash), Role: models.RoleClient}
			if i < 3 {
				user.INN = &companies[i].INN
			} else {
				user.Role = models.RoleOfficer
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			if i == 0 {
				application := models.Application{UserID: user.ID, CompanyID: companies[0].ID, Status: models.StatusScoring, Amount: 1500000, TermMonths: 12}
				if err := tx.Create(&application).Error; err != nil {
					return err
				}
			}
		}
		it, retail := "62.01", "47.11"
		news := []models.NewsArticle{
			{Title: "Как подготовиться к кредитной заявке", Content: "Демонстрационная новость: проверьте сведения о компании и финансовые показатели."},
			{Title: "Финансирование IT-бизнеса", Content: "Демонстрационная новость для ОКВЭД 62.01: планирование расходов на разработку.", TargetOKVED: &it},
			{Title: "Оборотные средства для ритейла", Content: "Демонстрационная новость для ОКВЭД 47.11: планирование закупок и сезонных запасов.", TargetOKVED: &retail},
		}
		return tx.Create(&news).Error
	})
}
