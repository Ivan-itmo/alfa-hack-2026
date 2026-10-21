package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"alfa-credit/internal/auth"
	"alfa-credit/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const UserContextKey = "user"

// AuthMiddleware проверяет подпись, срок JWT и актуальность пользователя в БД.
func AuthMiddleware(db *gorm.DB, tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Требуется Authorization: Bearer <token>"})
			return
		}
		claims, err := tokens.Parse(parts[1])
		if err != nil || claims.UserID == 0 || claims.Subject != strconv.FormatUint(uint64(claims.UserID), 10) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Недействительный или просроченный токен"})
			return
		}
		var user models.User
		err = db.WithContext(c.Request.Context()).First(&user, claims.UserID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не найден"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки пользователя"})
			return
		}
		// Роль и компания берутся из актуальной записи, а не из устаревших клеймов.
		c.Set(UserContextKey, user)
		c.Next()
	}
}
