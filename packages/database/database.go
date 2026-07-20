package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"hotspotos/packages/common"
	"hotspotos/packages/logger"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
)

// ConnectPostgres initializes PostgreSQL database connection
func ConnectPostgres(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Retry database connection
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			logger.Info("Successfully connected to PostgreSQL")
			DB = db
			return db, nil
		}
		logger.Warn(fmt.Sprintf("Failed to connect to PostgreSQL (attempt %d/10): %v", i+1, err))
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to postgres: %w", err)
}

// ConnectRedis initializes Redis connection
func ConnectRedis(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Info("Successfully connected to Redis")
	Redis = client
	return client, nil
}

// Migrate performs database migrations and seeds initial data
func Migrate(db *gorm.DB) error {
	logger.Info("Starting database migration...")
	err := db.AutoMigrate(
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
		return fmt.Errorf("migration failed: %w", err)
	}
	logger.Info("Database migration completed successfully")

	// Seed default admin
	var adminCount int64
	db.Model(&common.Admin{}).Count(&adminCount)
	if adminCount == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin := common.Admin{
			Username:     "admin",
			PasswordHash: string(hashedPassword),
			Role:         "superadmin",
		}
		if err := db.Create(&admin).Error; err != nil {
			logger.Error("Failed to seed default admin", "error", err)
		} else {
			logger.Info("Default admin seeded: admin/admin123")
		}
	}

	// Seed default plans
	var planCount int64
	db.Model(&common.Plan{}).Count(&planCount)
	if planCount == 0 {
		plans := []common.Plan{
			{Name: "1 Hour Plan", DurationMinutes: 60, PriceKes: 20.00, BandwidthLimitDown: 2048, BandwidthLimitUp: 1024},
			{Name: "3 Hours Plan", DurationMinutes: 180, PriceKes: 50.00, BandwidthLimitDown: 3072, BandwidthLimitUp: 1536},
			{Name: "24 Hours Plan", DurationMinutes: 1440, PriceKes: 100.00, BandwidthLimitDown: 5120, BandwidthLimitUp: 2048},
			{Name: "THEGOAT", DurationMinutes: 10080, PriceKes: 500.00, BandwidthLimitDown: 10240, BandwidthLimitUp: 5120},
			{Name: "1 KES Test Plan", DurationMinutes: 60, PriceKes: 1.00, BandwidthLimitDown: 2048, BandwidthLimitUp: 1024},
			{Name: "2 KES Test Plan", DurationMinutes: 120, PriceKes: 2.00, BandwidthLimitDown: 3072, BandwidthLimitUp: 1536},
			{Name: "5 KES Test Plan", DurationMinutes: 300, PriceKes: 5.00, BandwidthLimitDown: 5120, BandwidthLimitUp: 2048},
		}
		for _, plan := range plans {
			if err := db.Create(&plan).Error; err != nil {
				logger.Error("Failed to seed plan", "name", plan.Name, "error", err)
			}
		}
		logger.Info("Default plans seeded")
	} else {
		// Ensure THEGOAT and the test plans are created even if plans already exist
		additionalPlans := []common.Plan{
			{Name: "THEGOAT", DurationMinutes: 10080, PriceKes: 500.00, BandwidthLimitDown: 10240, BandwidthLimitUp: 5120},
			{Name: "1 KES Test Plan", DurationMinutes: 60, PriceKes: 1.00, BandwidthLimitDown: 2048, BandwidthLimitUp: 1024},
			{Name: "2 KES Test Plan", DurationMinutes: 120, PriceKes: 2.00, BandwidthLimitDown: 3072, BandwidthLimitUp: 1536},
			{Name: "5 KES Test Plan", DurationMinutes: 300, PriceKes: 5.00, BandwidthLimitDown: 5120, BandwidthLimitUp: 2048},
		}
		for _, plan := range additionalPlans {
			var existing common.Plan
			if err := db.Where("name = ?", plan.Name).First(&existing).Error; err != nil {
				if err := db.Create(&plan).Error; err != nil {
					logger.Error("Failed to seed additional plan", "name", plan.Name, "error", err)
				} else {
					logger.Info("Additional plan seeded successfully", "name", plan.Name)
				}
			}
		}
	}

	return nil
}
