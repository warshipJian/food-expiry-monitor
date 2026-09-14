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

func TestFamilyMembersShareFoodsAndDashboard(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, Config{TokenSecret: "test-secret", ReminderCronHour: 9})
	for _, id := range []string{"owner", "member"} {
		if _, err := db.Exec("INSERT INTO users (id,openid,nickname) VALUES (?,?,?)", id, id+"-openid", id); err != nil {
			t.Fatal(err)
		}
	}
	ownerToken, err := svc.signToken("owner")
	if err != nil {
		t.Fatal(err)
	}
	memberToken, err := svc.signToken("member")
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		svc.Router().ServeHTTP(res, req)
		return res
	}
	if res := call(http.MethodPost, "/v1/foods", `{"name":"牛奶","expiryDate":"2030-01-02"}`, ownerToken); res.Code != http.StatusCreated {
		t.Fatalf("owner food: %d %s", res.Code, res.Body.String())
	}
	if res := call(http.MethodPost, "/v1/family", `{"name":"幸福小家"}`, ownerToken); res.Code != http.StatusCreated {
		t.Fatalf("create family: %d %s", res.Code, res.Body.String())
	}
	var inviteCode string
	if err := db.QueryRow("SELECT invite_code FROM families").Scan(&inviteCode); err != nil {
		t.Fatal(err)
	}
	if res := call(http.MethodPost, "/v1/foods", `{"name":"鸡蛋","expiryDate":"2030-01-03"}`, memberToken); res.Code != http.StatusCreated {
		t.Fatalf("member food: %d %s", res.Code, res.Body.String())
	}
	if res := call(http.MethodPost, "/v1/family/join", `{"inviteCode":"`+inviteCode+`"}`, memberToken); res.Code != http.StatusOK {
		t.Fatalf("join family: %d %s", res.Code, res.Body.String())
	}
	if res := call(http.MethodGet, "/v1/dashboard", "", memberToken); res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"active":2`) {
		t.Fatalf("shared dashboard: %d %s", res.Code, res.Body.String())
	}
	var foodID string
	if err := db.QueryRow("SELECT id FROM foods WHERE name='牛奶'").Scan(&foodID); err != nil {
		t.Fatal(err)
	}
	if res := call(http.MethodPatch, "/v1/foods/"+foodID, `{"name":"鲜奶","expiryDate":"2030-01-02"}`, memberToken); res.Code != http.StatusOK {
		t.Fatalf("member edit: %d %s", res.Code, res.Body.String())
	}
}
