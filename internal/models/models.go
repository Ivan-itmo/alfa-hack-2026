package models

import "time"

type Role string

const (
	RoleClient  Role = "client"
	RoleOfficer Role = "officer"
)

type ApplicationStatus string

const (
	StatusDraft     ApplicationStatus = "DRAFT"
	StatusSubmitted ApplicationStatus = "SUBMITTED"
	StatusScoring   ApplicationStatus = "SCORING"
	StatusApproved  ApplicationStatus = "APPROVED"
	StatusRejected  ApplicationStatus = "REJECTED"
)

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Email        string `gorm:"size:254;not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         Role   `gorm:"size:16;not null;check:role IN ('client','officer')" json:"role"`
	// Сотрудник банка не привязан к компании; у клиента ИНН обязателен.
	INN     *string  `gorm:"size:12;index" json:"inn"`
	Company *Company `gorm:"belongsTo:Company;foreignKey:INN;references:INN;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}

type Company struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	INN   string `gorm:"size:12;not null;uniqueIndex" json:"inn"`
	Name  string `gorm:"not null" json:"name"`
	OKVED string `gorm:"size:16;not null" json:"okved"`
	// Денежные величины демо хранятся в целых рублях, без float.
	Revenue          int64      `gorm:"not null;check:revenue >= 0" json:"revenue"`
	RegistrationDate *time.Time `gorm:"type:date" json:"registration_date"`
}

type Application struct {
	ID         uint              `gorm:"primaryKey" json:"id"`
	UserID     uint              `gorm:"not null;index" json:"user_id"`
	User       User              `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	CompanyID  uint              `gorm:"not null;index" json:"company_id"`
	Company    Company           `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	Status     ApplicationStatus `gorm:"size:16;not null;index;check:status IN ('DRAFT','SUBMITTED','SCORING','APPROVED','REJECTED')" json:"status"`
	Amount     int64             `gorm:"not null;check:amount > 0" json:"amount"`
	TermMonths int               `gorm:"not null;check:term_months > 0" json:"term_months"`
}

type NewsArticle struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Title       string  `gorm:"not null" json:"title"`
	Content     string  `gorm:"type:text;not null" json:"content"`
	TargetOKVED *string `gorm:"size:16;index" json:"target_okved"`
}
