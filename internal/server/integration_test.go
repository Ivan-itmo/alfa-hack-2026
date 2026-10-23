package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"biz-credit-ai/internal/auth"
	"biz-credit-ai/internal/database"
	"biz-credit-ai/internal/models"
	"biz-credit-ai/internal/server"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Интеграционный тест работает с настоящим PostgreSQL в собственной временной схеме.
func TestAPIWithPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("задайте TEST_DATABASE_URL для интеграционного теста PostgreSQL")
	}
	admin, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := admin.DB()
	defer pool.Close()
	schema := fmt.Sprintf("test_api_%d", time.Now().UnixNano())
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
	testPool, _ := db.DB()
	defer testPool.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("повторная миграция: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := database.SeedDatabase(db); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for _, item := range []struct {
		model any
		want  int64
	}{{&models.User{}, 4}, {&models.Company{}, 3}, {&models.Application{}, 1}, {&models.NewsArticle{}, 3}} {
		var n int64
		if err := db.Model(item.model).Count(&n).Error; err != nil || n != item.want {
			t.Fatalf("сидинг %T: count=%d, err=%v", item.model, n, err)
		}
	}
	var seeded models.User
	db.Where("email = ?", "client1").First(&seeded)
	if err := bcrypt.CompareHashAndPassword([]byte(seeded.PasswordHash), []byte("123")); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	tokens := auth.NewTokenManager("test-secret-is-at-least-32-bytes-long")
	router := server.NewRouter(db, tokens)
	request := func(method, path, body, token string, want int) map[string]json.RawMessage {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, out.Code, want, out.Body.String())
		}
		if bytes.Contains(out.Body.Bytes(), []byte("password_hash")) || bytes.Contains(out.Body.Bytes(), []byte("$2a$")) {
			t.Fatal("хеш пароля попал в ответ")
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	login := func(name string) string {
		t.Helper()
		response := request(http.MethodPost, "/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"123"}`, name), "", 200)
		var token string
		json.Unmarshal(response["token"], &token)
		if token == "" {
			t.Fatal("пустой токен")
		}
		return token
	}
	request("GET", "/healthz", "", "", 200)
	request("GET", "/api/dashboard/", "", "", 401)
	request("GET", "/api/dashboard/", "", "invalid", 401)
	request("POST", "/api/auth/login", `{"email":"client1","password":"wrong"}`, "", 401)
	request("POST", "/api/auth/login", `{"email":"absent","password":"123"}`, "", 401)
	request("POST", "/api/auth/login", `{"email":"client1"}`, "", 400)
	request("POST", "/api/auth/login", `{"email":"client1","password":"123"}{}`, "", 400)
	request("POST", "/api/auth/login", `{"email":"client1","password":"`+strings.Repeat("x", 1<<20)+`"}`, "", 413)
	for _, item := range []struct {
		login, okved string
		active       bool
	}{{"client1", "62.01", true}, {"client2", "47.11", false}, {"client3", "62.01", false}} {
		token := login(item.login)
		response := request("GET", "/api/dashboard/", "", token, 200)
		var news []models.NewsArticle
		json.Unmarshal(response["news_feed"], &news)
		if len(news) != 2 || news[0].TargetOKVED == nil || *news[0].TargetOKVED != item.okved || news[1].TargetOKVED != nil {
			t.Fatalf("неверная персонализация: %s", response["news_feed"])
		}
		if item.active {
			var app models.Application
			json.Unmarshal(response["active_application"], &app)
			if app.Status != models.StatusScoring || app.UserID != seeded.ID {
				t.Fatalf("неверная активная заявка: %s", response["active_application"])
			}
		} else if string(response["active_application"]) != "null" {
			t.Fatal("заявка другого клиента попала в ответ")
		}
	}
	request("GET", "/api/dashboard/", "", login("officer1"), 403)
	request("POST", "/api/auth/register", `{"email":"bad","password":"123","inn":"abc"}`, "", 400)
	request("POST", "/api/auth/register", `{"email":"bad","password":"12","inn":"7701000001"}`, "", 400)
	// Попытка задать роль игнорируется: публичная регистрация создает только клиента.
	response := request("POST", "/api/auth/register", `{"email":"NEW@example.test","password":"secret123","inn":"7701000001","role":"officer"}`, "", 201)
	var token string
	json.Unmarshal(response["token"], &token)
	dashboard := request("GET", "/api/dashboard", "", token, 200)
	var company models.Company
	json.Unmarshal(dashboard["company_info"], &company)
	if company.INN != "7701000001" || company.OKVED != "62.01" {
		t.Fatal("не выполнена привязка к существующей компании")
	}
	var registered models.User
	json.Unmarshal(response["user"], &registered)
	if registered.Role != models.RoleClient || registered.Email != "new@example.test" {
		t.Fatalf("неверный пользователь: %+v", registered)
	}
	// При конфликте логина создание компании откатывается той же транзакцией.
	request("POST", "/api/auth/register", `{"email":"new@example.test","password":"secret123","inn":"9999999999"}`, "", 409)
	var count int64
	db.Model(&models.Company{}).Where("inn = ?", "9999999999").Count(&count)
	if count != 0 {
		t.Fatal("при конфликте осталась лишняя компания")
	}
	response = request("POST", "/api/auth/register", `{"email":"new-company","password":"123","inn":"9999999999"}`, "", 201)
	json.Unmarshal(response["token"], &token)
	dashboard = request("GET", "/api/dashboard/", "", token, 200)
	var news []models.NewsArticle
	json.Unmarshal(dashboard["news_feed"], &news)
	if len(news) != 1 || news[0].TargetOKVED != nil {
		t.Fatal("для компании без ОКВЭД ожидаются только общие новости")
	}
	db.Delete(&models.User{}, registered.ID)
	deletedToken, _ := tokens.Generate(registered)
	request("GET", "/api/dashboard/", "", deletedToken, 401)
	if err := database.SeedDatabase(db); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.Company{}).Count(&count)
	if count != 4 {
		t.Fatal("повторный сидинг изменил пользовательские данные")
	}
}
