package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"biz-credit-ai/internal/auth"
	"biz-credit-ai/internal/scoring"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

type Handler struct {
	Scoring *scoring.Client
	DB      *gorm.DB
	Tokens  *auth.TokenManager
}

func fail(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"error": message}) }

func bindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(c, http.StatusRequestEntityTooLarge, "Тело запроса превышает 1 МиБ")
		} else {
			fail(c, http.StatusBadRequest, "Некорректный JSON")
		}
		return false
	}
	if err := binding.Validator.ValidateStruct(dst); err != nil {
		fail(c, http.StatusBadRequest, "Не заполнены обязательные поля")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(c, http.StatusRequestEntityTooLarge, "Тело запроса превышает 1 МиБ")
		} else {
			fail(c, http.StatusBadRequest, "Ожидается один JSON-объект")
		}
		return false
	}
	return true
}
