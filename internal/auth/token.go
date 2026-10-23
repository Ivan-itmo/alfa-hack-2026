package auth

import (
	"strconv"
	"time"

	"biz-credit-ai/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "alfa-credit"
const TokenTTL = 24 * time.Hour

type Claims struct {
	UserID uint        `json:"user_id"`
	Role   models.Role `json:"role"`
	INN    *string     `json:"inn"`
	jwt.RegisteredClaims
}

type TokenManager struct{ secret []byte }

func NewTokenManager(secret string) *TokenManager { return &TokenManager{secret: []byte(secret)} }

func (m *TokenManager) Generate(user models.User) (string, error) {
	now := time.Now()
	claims := Claims{UserID: user.ID, Role: user.Role, INN: user.INN, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: Issuer, Subject: strconv.FormatUint(uint64(user.ID), 10), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(Issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	return claims, nil
}
