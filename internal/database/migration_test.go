package database_test

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"biz-credit-ai/internal/database"
	"biz-credit-ai/internal/models"
)

func TestLegacyMigration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	admin, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := admin.DB()
	defer pool.Close()
	schema := fmt.Sprintf("test_legacy_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP SCHEMA " + schema + " CASCADE")
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := database.Open(u.String())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := db.DB()
	defer p.Close()
	statements := []string{
		`CREATE TABLE companies (id bigserial PRIMARY KEY, inn varchar(12) NOT NULL, name text NOT NULL, okved varchar(16) NOT NULL, revenue bigint NOT NULL CHECK (revenue >= 0), registration_date date)`,
		`CREATE TABLE users (id bigserial PRIMARY KEY, email varchar(254) NOT NULL, password_hash text NOT NULL, role varchar(16) NOT NULL CHECK (role IN ('client','officer')), inn varchar(12))`,
		`CREATE TABLE applications (id bigserial PRIMARY KEY, user_id bigint NOT NULL, company_id bigint NOT NULL, status varchar(16) NOT NULL CONSTRAINT chk_applications_status CHECK (status IN ('DRAFT','SUBMITTED','SCORING','APPROVED','REJECTED')), amount bigint NOT NULL CHECK (amount > 0), term_months bigint NOT NULL CHECK (term_months > 0))`,
		`CREATE UNIQUE INDEX idx_companies_inn ON companies (inn)`,
		`CREATE UNIQUE INDEX idx_users_email ON users (email)`,
		`INSERT INTO companies (inn,name,okved,revenue) VALUES ('7701000001','Legacy demo','62.01',24000000)`,
		`INSERT INTO users (email,password_hash,role,inn) VALUES ('legacy','test','client','7701000001')`,
		`INSERT INTO applications (user_id,company_id,status,amount,term_months) VALUES (1,1,'SCORING',1500000,12),(1,1,'SUBMITTED',100000,6)`,
	}
	for _, sql := range statements {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := database.Migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	var apps []models.Application
	if err := db.Order("id").Find(&apps).Error; err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].Status != models.StatusScoring || apps[1].Status != models.StatusDataEnrichment || apps[0].RequestedAmount != 1500000 || apps[0].RequestedTermMonths != 12 {
		t.Fatalf("migration lost data: %+v", apps)
	}
	var company models.Company
	db.First(&company, 1)
	if company.MonthlyNetProfit != 600000 || company.MonthlyRevenue != 2000000 || company.BkiMonthlyPayment != 50000 {
		t.Fatalf("demo not upgraded: %+v", company)
	}
	if err := db.Model(&company).Update("balance", 123456).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	db.First(&company, 1)
	if company.Balance != 123456 {
		t.Fatal("migration reset balance")
	}
	if err := db.Exec("UPDATE applications SET status = 'INVALID' WHERE id = 1").Error; err == nil {
		t.Fatal("missing status constraint")
	}
}
