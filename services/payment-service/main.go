package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hotspotos/packages/common"
	"hotspotos/packages/database"
	"hotspotos/packages/logger"
)

type STKPushRequest struct {
	PhoneNumber string  `json:"phone_number"`
	Amount      float64 `json:"amount"`
	UserID      *uint   `json:"user_id"`
	PlanID      uint    `json:"plan_id"`
	MacAddress  string  `json:"mac_address"`
	IpAddress   string  `json:"ip_address"`
}

// Safaricom Callback Structs
type CallbackMetadataItem struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value,omitempty"`
}

type CallbackMetadata struct {
	Item []CallbackMetadataItem `json:"Item"`
}

type StkCallback struct {
	MerchantRequestID string           `json:"MerchantRequestID"`
	CheckoutRequestID string           `json:"CheckoutRequestID"`
	ResultCode        int              `json:"ResultCode"`
	ResultDesc        string           `json:"ResultDesc"`
	CallbackMetadata  CallbackMetadata `json:"CallbackMetadata"`
}

type Body struct {
	StkCallback StkCallback `json:"stkCallback"`
}

type MpesaCallbackRequest struct {
	Body Body `json:"Body"`
}

func main() {
	loadEnvFile(".env")
	loadEnvFile("../.env")
	loadEnvFile("../../.env")

	logger.Init("development")
	logger.Info("Starting HotspotOS Payment Service...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=hotspotos port=5432 sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Connect to shared databases
	var err error
	_, err = database.ConnectPostgres(dsn)
	if err != nil {
		logger.Error("Payment service DB connection failed", "error", err)
		os.Exit(1)
	}

	_, err = database.ConnectRedis(redisAddr, "", 0)
	if err != nil {
		logger.Error("Payment service Redis connection failed", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(func(c *fiber.Ctx) error {
		logger.Info("Payment Service Request", "method", c.Method(), "path", c.Path())
		return c.Next()
	})

	app.Post("/payments/stk", handleSTKPush)
	app.Post("/payments/callback", handleMpesaCallback)

	logger.Info("Payment Service listening", "port", port)
	if err := app.Listen(":" + port); err != nil {
		logger.Error("Payment Service stopped", "error", err)
	}
}

func handleSTKPush(c *fiber.Ctx) error {
	var req STKPushRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if req.PhoneNumber == "" || req.Amount <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "phone_number and positive amount are required"})
	}

	phone := sanitizePhoneNumber(req.PhoneNumber)

	mpesaBaseURL := os.Getenv("MPESA_BASE_URL")
	mpesaConsumerKey := os.Getenv("MPESA_CONSUMER_KEY")
	mpesaConsumerSecret := os.Getenv("MPESA_CONSUMER_SECRET")
	mpesaShortcode := os.Getenv("MPESA_BUSINESS_SHORTCODE")
	mpesaPasskey := os.Getenv("MPESA_PASSKEY")
	mpesaCallbackURL := os.Getenv("MPESA_CALLBACK_URL")
	mpesaPartyA := os.Getenv("MPESA_PARTY_A")
	mpesaPartyB := os.Getenv("MPESA_PARTY_B")

	var checkoutRequestID string
	var realSTKSent bool

	if mpesaConsumerKey != "" && mpesaConsumerSecret != "" {
		logger.Info("Attempting real Safaricom M-Pesa STK Push...")
		accessToken, err := getMpesaAccessToken(mpesaBaseURL, mpesaConsumerKey, mpesaConsumerSecret)
		if err != nil {
			logger.Error("Failed to get Safaricom access token", "error", err)
			return c.Status(500).JSON(fiber.Map{"error": "M-Pesa authorization failed: " + err.Error()})
		}

		loc, err := time.LoadLocation("Africa/Nairobi")
		var now time.Time
		if err == nil {
			now = time.Now().In(loc)
		} else {
			now = time.Now().UTC().Add(3 * time.Hour)
		}
		timestamp := now.Format("20060102150405")
		password := generateMpesaPassword(mpesaShortcode, mpesaPasskey, timestamp)

		pA := phone
		if mpesaPartyA != "" {
			pA = mpesaPartyA
		}
		pB := mpesaShortcode
		if mpesaPartyB != "" {
			pB = mpesaPartyB
		}

		stkURL := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", mpesaBaseURL)
		stkBody := map[string]interface{}{
			"BusinessShortCode": mpesaShortcode,
			"Password":          password,
			"Timestamp":         timestamp,
			"TransactionType":   "CustomerPayBillOnline",
			"Amount":            int(req.Amount),
			"PartyA":            pA,
			"PartyB":            pB,
			"PhoneNumber":       phone,
			"CallBackURL":       mpesaCallbackURL,
			"AccountReference":  "HotspotOS",
			"TransactionDesc":   "Internet Access Plan",
		}

		bodyBytes, _ := json.Marshal(stkBody)
		stkReq, err := http.NewRequest("POST", stkURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to build STK Push request: " + err.Error()})
		}

		stkReq.Header.Set("Authorization", "Bearer "+accessToken)
		stkReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 15 * time.Second}
		stkResp, err := client.Do(stkReq)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Safaricom API request failed: " + err.Error()})
		}
		defer stkResp.Body.Close()

		var stkRes struct {
			MerchantRequestID   string `json:"MerchantRequestID"`
			CheckoutRequestID   string `json:"CheckoutRequestID"`
			ResponseCode        string `json:"ResponseCode"`
			ResponseDescription string `json:"ResponseDescription"`
			CustomerMessage     string `json:"CustomerMessage"`
		}

		if err := json.NewDecoder(stkResp.Body).Decode(&stkRes); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to decode Safaricom response: " + err.Error()})
		}

		if stkRes.ResponseCode == "0" {
			checkoutRequestID = stkRes.CheckoutRequestID
			realSTKSent = true
			logger.Info("Real M-Pesa STK Push initiated successfully", "checkout_request_id", checkoutRequestID)
		} else {
			logger.Warn("Safaricom STK Push rejected", "code", stkRes.ResponseCode, "desc", stkRes.ResponseDescription)
			return c.Status(400).JSON(fiber.Map{
				"error":   fmt.Sprintf("Safaricom rejected request: %s", stkRes.ResponseDescription),
				"details": stkRes,
			})
		}
	} else {
		// Mock Flow
		checkoutRequestID = "ws_CO_" + uuid.New().String()
		logger.Info("Using simulated M-Pesa STK Push flow", "checkout_request_id", checkoutRequestID)
	}

	// 1. Save payment as pending in the DB
	payment := common.Payment{
		CheckoutRequestID: checkoutRequestID,
		UserID:            req.UserID,
		AmountKes:         req.Amount,
		PhoneNumber:       phone,
		Status:            "pending",
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to record payment: " + err.Error()})
	}

	// 2. Start session (inactive status, waiting for payment confirmation)
	var plan common.Plan
	if err := database.DB.First(&plan, req.PlanID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid plan id"})
	}

	// Find or Create Device
	var device common.Device
	if err := database.DB.Where("mac_address = ?", req.MacAddress).First(&device).Error; err != nil {
		device = common.Device{
			MacAddress:   req.MacAddress,
			IpAddress:    req.IpAddress,
			UserID:       req.UserID,
			Manufacturer: "Unknown",
			DeviceType:   "Mobile",
		}
		if err := database.DB.Create(&device).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to record device"})
		}
	}

	session := common.Session{
		DeviceID:  device.ID,
		UserID:    req.UserID,
		PlanID:    plan.ID,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Duration(plan.DurationMinutes) * time.Minute),
		Status:    "inactive", // Inactive until payment callback confirms it
		IpAddress: req.IpAddress,
	}
	if err := database.DB.Create(&session).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create session: " + err.Error()})
	}

	// Link session to payment
	payment.SessionID = &session.ID
	database.DB.Save(&payment)

	if !realSTKSent {
		// Simulate async STK Callback only for mock runs
		go triggerMockCallback(checkoutRequestID, phone, req.Amount)
	}

	return c.JSON(fiber.Map{
		"checkout_request_id": checkoutRequestID,
		"status":              "pending",
		"message":             "STK push sent. Please check your phone.",
	})
}

func handleMpesaCallback(c *fiber.Ctx) error {
	var req MpesaCallbackRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error("Failed to parse M-Pesa callback", "error", err)
		return c.Status(400).SendString("Invalid Callback Format")
	}

	callback := req.Body.StkCallback
	logger.Info("Received M-Pesa callback", "checkout_request_id", callback.CheckoutRequestID, "result_code", callback.ResultCode)

	var payment common.Payment
	if err := database.DB.Where("checkout_request_id = ?", callback.CheckoutRequestID).First(&payment).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payment not found"})
	}

	rawJSON, _ := json.Marshal(req)
	payment.RawCallback = string(rawJSON)

	if callback.ResultCode == 0 {
		payment.Status = "completed"

		// Extract Receipt Number
		var receiptCode string
		for _, item := range callback.CallbackMetadata.Item {
			if item.Name == "MpesaReceiptNumber" {
				if valStr, ok := item.Value.(string); ok {
					receiptCode = valStr
				}
			}
		}
		if receiptCode == "" {
			receiptCode = "MPESA" + fmt.Sprintf("%d", rand.Intn(10000000))
		}
		payment.TransactionID = &receiptCode

		// Activate the linked session
		if payment.SessionID != nil {
			var session common.Session
			if err := database.DB.First(&session, *payment.SessionID).Error; err == nil {
				session.Status = "active"
				session.StartTime = time.Now()
				// Query Plan to get duration
				var plan common.Plan
				if err := database.DB.First(&plan, session.PlanID).Error; err == nil {
					session.EndTime = time.Now().Add(time.Duration(plan.DurationMinutes) * time.Minute)
				}
				database.DB.Save(&session)

				// Fetch Device info to pass to Network Manager
				var device common.Device
				if err := database.DB.First(&device, session.DeviceID).Error; err == nil {
					// Notify main API server to authorize client in the network manager
					go notifyApiToAuthorize(session.ID, device.MacAddress, device.IpAddress, plan.BandwidthLimitDown, plan.BandwidthLimitUp)
				}
			}
		}

		logger.Info("Payment completed successfully", "receipt", receiptCode)
	} else {
		payment.Status = "failed"
		if payment.SessionID != nil {
			database.DB.Model(&common.Session{}).Where("id = ?", *payment.SessionID).Update("status", "failed")
		}
		logger.Warn("Payment failed", "desc", callback.ResultDesc)
	}

	database.DB.Save(&payment)

	// Publish to Redis Pub/Sub to inform WebSockets of payment status
	pubData := map[string]interface{}{
		"checkout_request_id": payment.CheckoutRequestID,
		"status":              payment.Status,
		"session_id":          payment.SessionID,
	}
	pubJSON, _ := json.Marshal(pubData)
	database.Redis.Publish(context.Background(), "payment_updates", string(pubJSON))

	return c.JSON(fiber.Map{"ResultCode": 0, "ResultDesc": "Success"})
}

// Simulates user entering PIN and Safaricom hitting our callback URL
func triggerMockCallback(checkoutRequestID, phoneNumber string, amount float64) {
	time.Sleep(3 * time.Second)

	callbackPort := os.Getenv("PORT")
	if callbackPort == "" {
		callbackPort = "8082"
	}

	// Form Safaricom callback JSON payload
	callbackPayload := MpesaCallbackRequest{
		Body: Body{
			StkCallback: StkCallback{
				MerchantRequestID: "mock_merchant_id",
				CheckoutRequestID: checkoutRequestID,
				ResultCode:        0, // 0 = Success
				ResultDesc:        "The service request is processed successfully.",
				CallbackMetadata: CallbackMetadata{
					Item: []CallbackMetadataItem{
						{Name: "Amount", Value: amount},
						{Name: "MpesaReceiptNumber", Value: "Q" + uuid.New().String()[:9]},
						{Name: "TransactionDate", Value: 20260718210000},
						{Name: "PhoneNumber", Value: phoneNumber},
					},
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(callbackPayload)
	url := fmt.Sprintf("http://localhost:%s/payments/callback", callbackPort)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Error("Failed to create mock callback request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Mock callback network request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	logger.Info("Mock callback triggered successfully", "status", resp.Status)
}

func notifyApiToAuthorize(sessionID uint, mac, ip string, rateDown, rateUp int64) {
	apiURL := os.Getenv("API_INTERNAL_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	notifyURL := fmt.Sprintf("%s/internal/sessions/authorize", apiURL)
	payload := map[string]interface{}{
		"session_id": sessionID,
		"mac":        mac,
		"ip":         ip,
		"rate_down":  rateDown,
		"rate_up":    rateUp,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", notifyURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		logger.Error("Failed to create notify request to main API", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// We can use a secret header to verify internal requests
	req.Header.Set("X-Internal-Token", "internal_secret_token")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Internal call to API server to authorize client failed", "error", err)
		return
	}
	defer resp.Body.Close()

	logger.Info("Notified API server to authorize client", "session_id", sessionID, "response_status", resp.Status)
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func sanitizePhoneNumber(phone string) string {
	reg := regexp.MustCompile(`[^0-9]`)
	clean := reg.ReplaceAllString(phone, "")

	if len(clean) == 10 && (strings.HasPrefix(clean, "07") || strings.HasPrefix(clean, "01")) {
		return "254" + clean[1:]
	}
	if len(clean) == 12 && strings.HasPrefix(clean, "254") {
		return clean
	}
	return clean
}

func getMpesaAccessToken(baseURL, consumerKey, consumerSecret string) (string, error) {
	url := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", consumerKey, consumerSecret)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth failed with status %d", resp.StatusCode)
	}

	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

func generateMpesaPassword(shortcode, passkey, timestamp string) string {
	val := shortcode + passkey + timestamp
	return base64.StdEncoding.EncodeToString([]byte(val))
}
