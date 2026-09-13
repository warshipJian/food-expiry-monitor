package app

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/james/food-expiry-monitor/internal/store"
	_ "modernc.org/sqlite"
)

func TestCreateFoodAndDashboard(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, Config{TokenSecret: "test-secret", ReminderCronHour: 9})
	if _, err := db.Exec("INSERT INTO users (id, openid) VALUES ('user-1', 'openid-1')"); err != nil {
		t.Fatal(err)
	}
	token, err := svc.signToken("user-1")
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/foods", strings.NewReader(`{"name":"牛奶","expiryDate":"2030-01-02"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	svc.Router().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create food: status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	svc.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard: status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"active":1`) {
		t.Fatalf("unexpected dashboard: %s", response.Body.String())
	}
}

func TestDevelopmentLoginIsDisabledInProduction(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, Config{TokenSecret: "test-secret", Environment: "production"})
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/dev/login", strings.NewReader(`{"deviceId":"local"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	svc.Router().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("development login should not exist in production: %d", response.Code)
	}
}

func TestDevelopmentLoginIssuesToken(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, Config{TokenSecret: "test-secret", Environment: "development"})
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/dev/login", strings.NewReader(`{"deviceId":"local"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	svc.Router().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"token"`) {
		t.Fatalf("development login failed: status=%d body=%s", response.Code, response.Body.String())
	}
}
