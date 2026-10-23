package handlers

import (
	"errors"
	"net/http"

	"biz-credit-ai/internal/middleware"
	"biz-credit-ai/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) Dashboard(c *gin.Context) {
	user, ok := c.MustGet(middleware.UserContextKey).(models.User)
	if !ok {
		fail(c, http.StatusUnauthorized, "Пользователь не найден")
		return
	}
	if user.Role != models.RoleClient {
		fail(c, http.StatusForbidden, "Дашборд доступен только клиентам")
		return
	}
	if user.INN == nil {
		fail(c, http.StatusNotFound, "Компания не найдена")
		return
	}
	var company models.Company
	err := h.DB.WithContext(c.Request.Context()).Where("inn = ?", *user.INN).First(&company).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "Компания не найдена")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "Не удалось получить компанию")
		return
	}
	var application models.Application
	var active *models.Application
	err = h.DB.WithContext(c.Request.Context()).Where("user_id = ? AND company_id = ? AND status IN ?", user.ID, company.ID,
		[]models.ApplicationStatus{models.StatusDraft, models.StatusSubmitted, models.StatusScoring}).Order("id DESC").First(&application).Error
	if err == nil {
		active = &application
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusInternalServerError, "Не удалось получить заявку")
		return
	}
	news := make([]models.NewsArticle, 0)
	// Чужие отраслевые новости не попадают в ленту; NULL идет после отраслевых.
	query := h.DB.WithContext(c.Request.Context()).Where("target_okved IS NULL")
	if company.OKVED != "" {
		query = h.DB.WithContext(c.Request.Context()).Where("target_okved = ? OR target_okved IS NULL", company.OKVED)
	}
	if err := query.Order("target_okved IS NULL ASC").Order("id DESC").Find(&news).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Не удалось получить новости")
		return
	}
	c.JSON(http.StatusOK, gin.H{"active_application": active, "news_feed": news, "company_info": company})
}
