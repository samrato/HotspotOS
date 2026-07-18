package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"hotspotos/packages/common"
	"hotspotos/packages/database"
)

// setupTestDb initializes an in-memory SQLite database for hermetic testing
func setupTestDb(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// AutoMigrate all models
	err = db.AutoMigrate(
		&common.Admin{},
		&common.User{},
		&common.Device{},
		&common.Plan{},
		&common.Session{},
		&common.Payment{},
		&common.Router{},
		&common.Voucher{},
		&common.AuditLog{},
		&common.Notification{},
	)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	database.DB = db
	return db
}

func TestAdminLogin(t *testing.T) {
	db := setupTestDb(t)

	// Seed admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	admin := common.Admin{
		Username:     "testadmin",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	db.Create(&admin)

	app := fiber.New()
	app.Post("/auth/login", handleLogin)

	// 1. Test valid credentials
	loginReq := LoginReq{
		Username: "testadmin",
		Password: "testpassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var loginResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if loginResp["token"] == "" {
		t.Error("expected token in response, got empty string")
	}

	// 2. Test invalid credentials
	loginReq.Password = "wrongpassword"
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestGetPlans(t *testing.T) {
	db := setupTestDb(t)

	// Seed some plans
	plans := []common.Plan{
		{Name: "1 Hour", DurationMinutes: 60, PriceKes: 20.0},
		{Name: "24 Hours", DurationMinutes: 1440, PriceKes: 100.0},
	}
	db.Create(&plans)

	app := fiber.New()
	app.Get("/plans", handleGetPlans)

	req := httptest.NewRequest("GET", "/plans", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to execute test request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var returnedPlans []common.Plan
	json.NewDecoder(resp.Body).Decode(&returnedPlans)

	if len(returnedPlans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(returnedPlans))
	}
	if returnedPlans[0].Name != "1 Hour" {
		t.Errorf("expected name to be '1 Hour', got %s", returnedPlans[0].Name)
	}
}

func TestInternalAuthorize(t *testing.T) {
	db := setupTestDb(t)

	// Seed plan & session
	plan := common.Plan{Name: "1 Hour", DurationMinutes: 60, PriceKes: 20.0}
	db.Create(&plan)

	device := common.Device{MacAddress: "11:22:33:44:55:66", IpAddress: "10.0.0.10"}
	db.Create(&device)

	session := common.Session{
		DeviceID:  device.ID,
		PlanID:    plan.ID,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
		Status:    "inactive",
	}
	db.Create(&session)

	app := fiber.New()
	// Register route with mock network-manager response logic (handled in simulation fallback)
	app.Post("/internal/sessions/authorize", handleInternalAuthorize)

	reqPayload := InternalAuthorizeReq{
		SessionID: session.ID,
		Mac:       device.MacAddress,
		Ip:        device.IpAddress,
		RateDown:  2048,
		RateUp:    1024,
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest("POST", "/internal/sessions/authorize", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Verify that mock simulation executes fine even without network manager active
	resp, err := app.Test(req, 10000) // increase timeout
	if err != nil {
		t.Fatalf("request execution failed: %v", err)
	}

	// We expect 502 Bad Gateway because the real network-manager is not running on localhost:8081,
	// which tests that the proxy logic is working and attempting HTTP requests correctly.
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected proxy 502 bad gateway, got %d", resp.StatusCode)
	}
}
