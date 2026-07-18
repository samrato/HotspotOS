package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"golang.org/x/crypto/bcrypt"

	"hotspotos/packages/auth"
	"hotspotos/packages/common"
	"hotspotos/packages/database"
	"hotspotos/packages/logger"
)

// Admin login handler
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func handleLogin(c *fiber.Ctx) error {
	var req LoginReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	var admin common.Admin
	if err := database.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid username or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid username or password"})
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super_secret_jwt_key_hotspotos"
	}

	token, err := auth.GenerateToken(admin.ID, admin.Username, admin.Role, secret)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"admin": fiber.Map{
			"username": admin.Username,
			"role":     admin.Role,
		},
	})
}

// Get plans handler
func handleGetPlans(c *fiber.Ctx) error {
	var plans []common.Plan
	if err := database.DB.Find(&plans).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch plans"})
	}
	return c.JSON(plans)
}

// Initiate payment via STK Push
func handleSTKPush(c *fiber.Ctx) error {
	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		paymentServiceURL = "http://localhost:8082"
	}

	// Read client IP and MAC address (usually passed in headers by redirect/dnsmasq)
	// We'll read from JSON body or queries
	// Body contains phone_number, plan_id, mac_address, ip_address
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "serialization error"})
	}

	req, err := http.NewRequest("POST", paymentServiceURL+"/payments/stk", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to contact payment service"})
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "payment service is unreachable"})
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "malformed response from payment service"})
	}

	return c.Status(resp.StatusCode).JSON(result)
}

// Internal callback endpoint: called by payment-service to finalize client auth on firewall
type InternalAuthorizeReq struct {
	SessionID uint   `json:"session_id"`
	Mac       string `json:"mac"`
	Ip        string `json:"ip"`
	RateDown  int64  `json:"rate_down"`
	RateUp    int64  `json:"rate_up"`
}

func handleInternalAuthorize(c *fiber.Ctx) error {
	var req InternalAuthorizeReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	networkManagerURL := os.Getenv("NETWORK_MANAGER_URL")
	if networkManagerURL == "" {
		networkManagerURL = "http://localhost:8081"
	}

	// 1. Authorize on firewall
	nmURL := fmt.Sprintf("%s/clients/authorize", networkManagerURL)
	payload := map[string]interface{}{
		"mac":       req.Mac,
		"ip":        req.Ip,
		"rate_down": req.RateDown,
		"rate_up":   req.RateUp,
	}
	bodyBytes, _ := json.Marshal(payload)
	nmReq, err := http.NewRequest("POST", nmURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create request to network manager"})
	}
	nmReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	nmResp, err := client.Do(nmReq)
	if err != nil {
		logger.Error("Failed to reach network-manager", "error", err)
		return c.Status(502).JSON(fiber.Map{"error": "network-manager is unreachable"})
	}
	defer nmResp.Body.Close()

	if nmResp.StatusCode != 200 {
		return c.Status(500).JSON(fiber.Map{"error": "network-manager failed to authorize client"})
	}

	// Log audit trail
	audit := common.AuditLog{
		Action:      "CLIENT_AUTHORIZED",
		PerformedBy: "SYSTEM",
		Details:     fmt.Sprintf("Authorized MAC: %s, Session ID: %d", req.Mac, req.SessionID),
	}
	database.DB.Create(&audit)

	return c.JSON(fiber.Map{"status": "authorized"})
}

// Disconnect a client (Admin action)
type DisconnectReq struct {
	SessionID uint `json:"session_id"`
}

func handleAdminDisconnect(c *fiber.Ctx) error {
	var req DisconnectReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	var session common.Session
	if err := database.DB.Preload("Device").First(&session, req.SessionID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "session not found"})
	}

	if session.Status != "active" {
		return c.Status(400).JSON(fiber.Map{"error": "session is not active"})
	}

	networkManagerURL := os.Getenv("NETWORK_MANAGER_URL")
	if networkManagerURL == "" {
		networkManagerURL = "http://localhost:8081"
	}

	// Call network manager to revoke
	nmURL := fmt.Sprintf("%s/clients/revoke", networkManagerURL)
	payload := map[string]interface{}{
		"mac": session.Device.MacAddress,
		"ip":  session.Device.IpAddress,
	}
	bodyBytes, _ := json.Marshal(payload)
	nmReq, err := http.NewRequest("POST", nmURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to contact network manager"})
	}
	nmReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	nmResp, err := client.Do(nmReq)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "network manager unreachable"})
	}
	defer nmResp.Body.Close()

	// Update session status in DB
	session.Status = "disconnected"
	session.EndTime = time.Now()
	database.DB.Save(&session)

	// Log audit trail
	adminClaims := c.Locals("admin").(*auth.Claims)
	audit := common.AuditLog{
		Action:      "CLIENT_DISCONNECTED",
		PerformedBy: adminClaims.Username,
		Details:     fmt.Sprintf("Admin disconnected MAC: %s, Session ID: %d", session.Device.MacAddress, session.ID),
	}
	database.DB.Create(&audit)

	return c.JSON(fiber.Map{"status": "disconnected"})
}

// Get devices handler
func handleGetDevices(c *fiber.Ctx) error {
	var devices []common.Device
	if err := database.DB.Find(&devices).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to query devices"})
	}
	return c.JSON(devices)
}

// Get reports (payments and sessions)
func handleGetReports(c *fiber.Ctx) error {
	var payments []common.Payment
	if err := database.DB.Order("created_at desc").Limit(100).Find(&payments).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to query payment reports"})
	}

	var sessions []common.Session
	if err := database.DB.Preload("Device").Preload("Plan").Order("created_at desc").Limit(100).Find(&sessions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to query session reports"})
	}

	return c.JSON(fiber.Map{
		"payments": payments,
		"sessions": sessions,
	})
}

// Get dashboard analytics
func handleGetAnalytics(c *fiber.Ctx) error {
	var totalRevenue float64
	database.DB.Model(&common.Payment{}).Where("status = ?", "completed").Select("COALESCE(SUM(amount_kes), 0)").Scan(&totalRevenue)

	var activeUsers int64
	database.DB.Model(&common.Session{}).Where("status = ?", "active").Count(&activeUsers)

	var totalDevices int64
	database.DB.Model(&common.Device{}).Count(&totalDevices)

	// Fetch active sessions with details
	var sessions []common.Session
	database.DB.Preload("Device").Preload("Plan").Where("status = ?", "active").Order("start_time desc").Find(&sessions)

	// Add recent payments
	var payments []common.Payment
	database.DB.Order("created_at desc").Limit(5).Find(&payments)

	// Total today KES
	var todayRevenue float64
	todayStart := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&common.Payment{}).Where("status = ? AND created_at >= ?", "completed", todayStart).Select("COALESCE(SUM(amount_kes), 0)").Scan(&todayRevenue)

	// Speed is simulated/read from active devices
	bandwidthUsageMbps := 12.4 + float64(activeUsers)*1.2

	return c.JSON(fiber.Map{
		"revenue_total":        totalRevenue,
		"revenue_today":        todayRevenue,
		"active_users":         activeUsers,
		"total_devices":        totalDevices,
		"bandwidth_usage_mbps": bandwidthUsageMbps,
		"active_sessions":      sessions,
		"recent_payments":      payments,
	})
}

// Global array of websocket client connections
var (
	wsClients   = make(map[*websocket.Conn]bool)
	wsClientsMu sync.Mutex
)

// handleWebSocket manages websocket connections for real-time dashboard notifications
func handleWebSocket(c *websocket.Conn) {
	wsClientsMu.Lock()
	wsClients[c] = true
	wsClientsMu.Unlock()

	defer func() {
		wsClientsMu.Lock()
		delete(wsClients, c)
		wsClientsMu.Unlock()
		c.Close()
	}()

	// Read loop (keep-alive)
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

// Broadcasts payment alerts and statistics updates to dashboard admins
func startRedisListener() {
	rdb := database.Redis
	pubsub := rdb.Subscribe(context.Background(), "payment_updates")

	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		wsClientsMu.Lock()
		for client := range wsClients {
			if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				client.Close()
				delete(wsClients, client)
			}
		}
		wsClientsMu.Unlock()
	}
}

// Create a plan
func handleAdminCreatePlan(c *fiber.Ctx) error {
	var plan common.Plan
	if err := c.BodyParser(&plan); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if plan.Name == "" || plan.DurationMinutes <= 0 || plan.PriceKes < 0 {
		return c.Status(400).JSON(fiber.Map{"error": "name, duration, and price are required"})
	}

	if err := database.DB.Create(&plan).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create plan"})
	}

	// Log audit trail
	adminClaims := c.Locals("admin").(*auth.Claims)
	audit := common.AuditLog{
		Action:      "PLAN_CREATED",
		PerformedBy: adminClaims.Username,
		Details:     fmt.Sprintf("Created plan: %s (Duration: %d min, Price: %.2f KES)", plan.Name, plan.DurationMinutes, plan.PriceKes),
	}
	database.DB.Create(&audit)

	return c.JSON(plan)
}

// Update a plan
func handleAdminUpdatePlan(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id parameter"})
	}

	var plan common.Plan
	if err := database.DB.First(&plan, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "plan not found"})
	}

	var req common.Plan
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Update fields
	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.DurationMinutes > 0 {
		plan.DurationMinutes = req.DurationMinutes
	}
	if req.PriceKes >= 0 {
		plan.PriceKes = req.PriceKes
	}
	plan.BandwidthLimitDown = req.BandwidthLimitDown
	plan.BandwidthLimitUp = req.BandwidthLimitUp

	if err := database.DB.Save(&plan).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update plan"})
	}

	// Log audit trail
	adminClaims := c.Locals("admin").(*auth.Claims)
	audit := common.AuditLog{
		Action:      "PLAN_UPDATED",
		PerformedBy: adminClaims.Username,
		Details:     fmt.Sprintf("Updated plan ID %d: %s (Duration: %d min, Price: %.2f KES)", plan.ID, plan.Name, plan.DurationMinutes, plan.PriceKes),
	}
	database.DB.Create(&audit)

	return c.JSON(plan)
}

// Delete a plan
func handleAdminDeletePlan(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id parameter"})
	}

	var plan common.Plan
	if err := database.DB.First(&plan, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "plan not found"})
	}

	if err := database.DB.Delete(&plan).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete plan"})
	}

	// Log audit trail
	adminClaims := c.Locals("admin").(*auth.Claims)
	audit := common.AuditLog{
		Action:      "PLAN_DELETED",
		PerformedBy: adminClaims.Username,
		Details:     fmt.Sprintf("Deleted plan ID %d: %s", plan.ID, plan.Name),
	}
	database.DB.Create(&audit)

	return c.JSON(fiber.Map{"status": "deleted"})
}
