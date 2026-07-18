package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"hotspotos/packages/logger"
)

type ClientAuthReq struct {
	Mac      string `json:"mac"`
	Ip       string `json:"ip"`
	RateDown int64  `json:"rate_down"` // kbps
	RateUp   int64  `json:"rate_up"`   // kbps
}

type ClientRevokeReq struct {
	Mac string `json:"mac"`
	Ip  string `json:"ip"`
}

type ActiveClient struct {
	Mac      string    `json:"mac"`
	Ip       string    `json:"ip"`
	RateDown int64     `json:"rate_down"`
	RateUp   int64     `json:"rate_up"`
	BytesIn  int64     `json:"bytes_in"`
	BytesOut int64     `json:"bytes_out"`
	JoinedAt time.Time `json:"joined_at"`
}

var (
	activeClients = make(map[string]*ActiveClient)
	clientsMu     sync.RWMutex
	firewall      FirewallController
)

func main() {
	logger.Init("development")
	logger.Info("Starting HotspotOS Network Manager...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	simulateStr := os.Getenv("SIMULATE")
	simulate := true
	if simulateStr == "false" {
		simulate = false
	}

	flag.BoolVar(&simulate, "simulate", simulate, "Run in firewall simulation mode")
	flag.Parse()

	firewall = NewFirewallController(simulate)
	if err := firewall.SetupPortal(); err != nil {
		logger.Error("Failed to initialize firewall", "error", err)
	}

	// Simulating client byte increments in the background
	go startUsageSimulator()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(func(c *fiber.Ctx) error {
		logger.Info("NM API Request", "method", c.Method(), "path", c.Path())
		return c.Next()
	})

	app.Post("/clients/authorize", handleAuthorize)
	app.Post("/clients/revoke", handleRevoke)
	app.Get("/clients/active", handleListActive)

	logger.Info("Network Manager listening", "port", port)
	if err := app.Listen(":" + port); err != nil {
		logger.Error("Network Manager stopped", "error", err)
	}
}

func handleAuthorize(c *fiber.Ctx) error {
	var req ClientAuthReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if req.Mac == "" || req.Ip == "" {
		return c.Status(400).JSON(fiber.Map{"error": "mac and ip are required"})
	}

	if err := firewall.AuthorizeClient(req.Mac, req.Ip, req.RateDown, req.RateUp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	clientsMu.Lock()
	activeClients[req.Mac] = &ActiveClient{
		Mac:      req.Mac,
		Ip:       req.Ip,
		RateDown: req.RateDown,
		RateUp:   req.RateUp,
		BytesIn:  0,
		BytesOut: 0,
		JoinedAt: time.Now(),
	}
	clientsMu.Unlock()

	return c.JSON(fiber.Map{"status": "authorized"})
}

func handleRevoke(c *fiber.Ctx) error {
	var req ClientRevokeReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	if req.Mac == "" {
		return c.Status(400).JSON(fiber.Map{"error": "mac is required"})
	}

	if err := firewall.RevokeClient(req.Mac, req.Ip); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	clientsMu.Lock()
	delete(activeClients, req.Mac)
	clientsMu.Unlock()

	return c.JSON(fiber.Map{"status": "revoked"})
}

func handleListActive(c *fiber.Ctx) error {
	clientsMu.RLock()
	list := make([]ActiveClient, 0, len(activeClients))
	for _, client := range activeClients {
		list = append(list, *client)
	}
	clientsMu.RUnlock()
	return c.JSON(list)
}

// Background simulation loop that generates mock network traffic
func startUsageSimulator() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		clientsMu.Lock()
		for _, client := range activeClients {
			// Simulate traffic: adding a random increment of bytes
			// Inbound/outbound traffic based on the rate limits if any
			limitDown := client.RateDown
			if limitDown == 0 {
				limitDown = 10240 // Default to 10Mbps
			}
			// simulate 5 seconds of transfer at 10-80% capacity
			incrementIn := (limitDown * 1024 / 8) * 5 / 2
			incrementOut := (client.RateUp * 1024 / 8) * 5 / 2
			if incrementOut == 0 {
				incrementOut = incrementIn / 4
			}

			client.BytesIn += incrementIn
			client.BytesOut += incrementOut
		}
		clientsMu.Unlock()
	}
}
