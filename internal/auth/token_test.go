package auth

import (
	"testing"
	"time"

	"alfa-credit/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

func TestTokens(t *testing.T) {
	manager := NewTokenManager("test-secret-at-least-32-bytes-long")
	inn := "7701000001"
	raw, err := manager.Generate(models.User{ID: 42, Role: models.RoleClient, INN: &inn})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(raw)
	if err != nil || claims.UserID != 42 || claims.Role != models.RoleClient || claims.INN == nil || *claims.INN != inn {
		t.Fatalf("неверные claims: %+v, %v", claims, err)
	}
	if _, err := NewTokenManager("another-secret-at-least-32-bytes").Parse(raw); err == nil {
		t.Fatal("принят токен с чужой подписью")
	}
	for _, item := range []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.RegisteredClaims
	}{
		{"expired", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: Issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}},
		{"missing_exp", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: Issuer}},
		{"wrong_issuer", jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "other", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
		{"wrong_algorithm", jwt.SigningMethodHS384, jwt.RegisteredClaims{Issuer: Issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}},
	} {
		t.Run(item.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(item.method, Claims{RegisteredClaims: item.claims}).SignedString(manager.secret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Parse(raw); err == nil {
				t.Fatal("принят недействительный токен")
			}
		})
	}
}
