package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode"

	"biz-credit-ai/internal/auth"
	"biz-credit-ai/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type credentials struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	INN      string `json:"inn" binding:"required"`
}

func validLogin(s string) bool {
	return s != "" && len(s) <= 254 && !strings.ContainsFunc(s, unicode.IsSpace)
}
func validINN(s string) bool {
	if len(s) != 10 && len(s) != 12 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func (h *Handler) tokenResponse(c *gin.Context, status int, user models.User) {
	token, err := h.Tokens.Generate(user)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}
	c.JSON(status, gin.H{"token": token, "token_type": "Bearer", "expires_in": int(auth.TokenTTL.Seconds()), "user": user})
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if !bindJSON(c, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.INN = strings.TrimSpace(req.INN)
	if !validLogin(req.Email) || !validINN(req.INN) || len(req.Password) < 3 || len(req.Password) > 72 {
		fail(c, http.StatusBadRequest, "Нужны логин до 254 байт, пароль от 3 до 72 байт и ИНН из 10 или 12 цифр")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Не удалось обработать пароль")
		return
	}
	user := models.User{Email: req.Email, PasswordHash: string(hash), Role: models.RoleClient, INN: &req.INN}
	err = h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		// При конкурентных регистрациях один ИНН всегда соответствует одной компании.
		company := models.Company{INN: req.INN, Name: "Компания ИНН " + req.INN, OKVED: ""}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "inn"}}, DoNothing: true}).Create(&company).Error; err != nil {
			return err
		}
		return tx.Create(&user).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		fail(c, http.StatusConflict, "Пользователь с таким логином уже существует")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "Не удалось зарегистрировать пользователя")
		return
	}
	h.tokenResponse(c, http.StatusCreated, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req credentials
	if !bindJSON(c, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !validLogin(req.Email) || len(req.Password) > 72 {
		fail(c, http.StatusBadRequest, "Некорректный логин или длина пароля")
		return
	}
	var user models.User
	err := h.DB.WithContext(c.Request.Context()).Where("email = ?", req.Email).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusInternalServerError, "Не удалось выполнить вход")
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		fail(c, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	h.tokenResponse(c, http.StatusOK, user)
}
