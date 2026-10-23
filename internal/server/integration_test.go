package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"biz-credit-ai/internal/auth"
	"biz-credit-ai/internal/config"
	"biz-credit-ai/internal/database"
	"biz-credit-ai/internal/models"
	"biz-credit-ai/internal/scoring"
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

	// Full pipeline and authorization checks run in the same isolated PostgreSQL schema.
	client1, client2, client3, officer := login("client1"), login("client2"), login("client3"), login("officer1")
	request("GET", "/api/products", "", "", 401)
	request("GET", "/api/products", "", officer, 403)
	request("GET", "/api/products", "", client1, 200)
	request("GET", "/api/officer/applications", "", client1, 403)
	create := func(token string, amount int64) uint {
		t.Helper()
		res := request("POST", "/api/applications", fmt.Sprintf(`{"product_id":2,"purpose":"Оборотные средства","amount":%d,"term_months":12}`, amount), token, 201)
		var id uint
		json.Unmarshal(res["id"], &id)
		return id
	}
	path := func(id uint, action string) string { return fmt.Sprintf("/api/applications/%d%s", id, action) }
	id := create(client1, 1500000)
	request("GET", path(id, ""), "", client2, 404)
	request("POST", path(id, "/submit"), "", client2, 404)
	request("POST", path(id, "/sign"), `{"code":"1234"}`, client1, 409)
	res := request("POST", path(id, "/submit"), "", client1, 200)
	if string(res["status"]) != `"APPROVED"` {
		t.Fatalf("submit: %s", res["status"])
	}
	request("POST", path(id, "/submit"), "", client1, 409)
	request("POST", path(id, "/sign"), `{"code":"0000"}`, client1, 400)
	var before, after models.Company
	db.First(&before, 1)
	// Racing signatures must result in exactly one credit to the company balance.
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			req := httptest.NewRequest("POST", path(id, "/sign"), strings.NewReader(`{"code":"1234"}`))
			req.Header.Set("Authorization", "Bearer "+client1)
			out := httptest.NewRecorder()
			router.ServeHTTP(out, req)
			codes <- out.Code
		}()
	}
	a, b := <-codes, <-codes
	if !((a == 200 && b == 409) || (a == 409 && b == 200)) {
		t.Fatalf("signature codes %d %d", a, b)
	}
	db.First(&after, 1)
	if after.Balance-before.Balance != 1500000 {
		t.Fatal("duplicate or missing disbursement")
	}
	request("POST", fmt.Sprintf("/api/officer/applications/%d/decision", id), `{"decision":"REJECTED","comment":"Поздний отказ"}`, officer, 409)
	counter := create(client1, 10000000)
	res = request("POST", path(counter, "/submit"), "", client1, 200)
	if string(res["status"]) != `"COUNTER_OFFER"` {
		t.Fatal("expected counter offer")
	}
	request("POST", path(counter, "/counter-offer"), `{"action":"invalid"}`, client1, 400)
	request("POST", path(counter, "/counter-offer"), `{"action":"accept"}`, client1, 200)
	request("POST", path(counter, "/sign"), `{"code":"1234"}`, client1, 200)
	cancelled := create(client1, 10000000)
	request("POST", path(cancelled, "/submit"), "", client1, 200)
	request("POST", path(cancelled, "/counter-offer"), `{"action":"reject"}`, client1, 200)
	request("POST", path(cancelled, "/sign"), `{"code":"1234"}`, client1, 409)
	young := create(client3, 100000)
	res = request("POST", path(young, "/submit"), "", client3, 200)
	if string(res["status"]) != `"DOCS_REQUIRED"` {
		t.Fatal("expected documents")
	}
	upload := func(want int) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		f, _ := writer.CreateFormFile("file", "statement.txt")
		f.Write([]byte("Synthetic statement"))
		writer.Close()
		req := httptest.NewRequest("POST", path(young, "/documents"), &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+client3)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("upload: %d %s", out.Code, out.Body.String())
		}
	}
	upload(200)
	upload(409)
	var document models.ApplicationDocument
	if err := db.Where("application_id = ?", young).First(&document).Error; err != nil || string(document.Data) != "Synthetic statement" {
		t.Fatal("attachment not stored")
	}
	request("GET", "/api/officer/applications?status=MANUAL_REVIEW,SCORING_IN_PROGRESS", "", officer, 200)
	request("GET", "/api/officer/applications?status=UNKNOWN", "", officer, 400)
	decisionPath := fmt.Sprintf("/api/officer/applications/%d/decision", young)
	request("POST", decisionPath, `{"decision":"APPROVED","amount":-1,"comment":"Проверено"}`, officer, 400)
	request("POST", decisionPath, `{"decision":"APPROVED","amount":100000,"term_months":12,"comment":"Выписка проверена"}`, officer, 200)
	res = request("GET", path(young, ""), "", client3, 200)
	if string(res["status"]) != `"APPROVED"` {
		t.Fatal("officer decision invisible")
	}
	request("POST", path(young, "/sign"), `{"code":"1234"}`, client3, 200)
	request("POST", "/api/applications", `{"product_id":999,"purpose":"test","amount":100000,"term_months":12}`, client1, 400)

	// A late ML result must never overwrite a decision made by the officer.
	started, release := make(chan struct{}), make(chan struct{})
	ml := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input scoring.Request
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if input.CompanyINN != "7701000001" || input.ProductRate != 20 || input.MonthlyNetProfit != 600000 {
			t.Error("incorrect ML features")
		}
		if input.RequestedAmount == 2000000 {
			close(started)
			<-release
		}
		json.NewEncoder(w).Encode(scoring.Fallback(input, false))
	}))
	defer ml.Close()
	router = server.NewRouter(db, tokens, &config.Config{ScoringBaseURL: ml.URL})
	mlID := create(client1, 1500000)
	res = request("POST", path(mlID, "/submit"), "", client1, 200)
	if string(res["scoring_source"]) != `"ml"` {
		t.Fatal("ML result was not used")
	}
	racing := create(client1, 2000000)
	resultCode := make(chan int, 1)
	go func() {
		req := httptest.NewRequest("POST", path(racing, "/submit"), nil)
		req.Header.Set("Authorization", "Bearer "+client1)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		resultCode <- out.Code
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("ML did not start")
	}
	request("POST", fmt.Sprintf("/api/officer/applications/%d/decision", racing), `{"decision":"REJECTED","comment":"Ручная проверка выявила риск"}`, officer, 200)
	close(release)
	if code := <-resultCode; code != 409 {
		t.Fatalf("late ML response status=%d", code)
	}
	res = request("GET", path(racing, ""), "", client1, 200)
	if string(res["status"]) != `"REJECTED"` {
		t.Fatal("ML overwrote officer")
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
