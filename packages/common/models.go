package common

import (
	"time"
)

type Admin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);default:'admin'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"uniqueIndex;not null" json:"phone_number"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Devices     []Device  `gorm:"foreignKey:UserID" json:"devices,omitempty"`
	Payments    []Payment `gorm:"foreignKey:UserID" json:"payments,omitempty"`
	Sessions    []Session `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
}

type Device struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	MacAddress   string    `gorm:"uniqueIndex;not null;type:varchar(17)" json:"mac_address"`
	IpAddress    string    `gorm:"type:varchar(45)" json:"ip_address"`
	Manufacturer string    `gorm:"type:varchar(100)" json:"manufacturer"`
	DeviceType   string    `gorm:"type:varchar(50)" json:"device_type"`
	UserID       *uint     `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Plan struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `gorm:"not null" json:"name"`
	DurationMinutes    int       `gorm:"not null" json:"duration_minutes"`
	PriceKes           float64   `gorm:"type:decimal(10,2);not null" json:"price_kes"`
	BandwidthLimitDown int64     `gorm:"default:0" json:"bandwidth_limit_down"` // in kbps (0 means unlimited)
	BandwidthLimitUp   int64     `gorm:"default:0" json:"bandwidth_limit_up"`   // in kbps (0 means unlimited)
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DeviceID  uint      `gorm:"not null" json:"device_id"`
	Device    Device    `gorm:"foreignKey:DeviceID" json:"device"`
	UserID    *uint     `json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	PlanID    uint      `gorm:"not null" json:"plan_id"`
	Plan      Plan      `gorm:"foreignKey:PlanID" json:"plan"`
	StartTime time.Time `gorm:"not null" json:"start_time"`
	EndTime   time.Time `gorm:"not null;index" json:"end_time"`
	Status    string    `gorm:"type:varchar(50);default:'active';index" json:"status"` // active, expired, disconnected
	IpAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	BytesIn   int64     `gorm:"default:0" json:"bytes_in"`
	BytesOut  int64     `gorm:"default:0" json:"bytes_out"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Payment struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	TransactionID     *string    `gorm:"uniqueIndex;type:varchar(50)" json:"transaction_id"` // M-Pesa receipt code e.g. KDL39DJS8S
	CheckoutRequestID string     `gorm:"uniqueIndex;not null;type:varchar(50)" json:"checkout_request_id"`
	UserID            *uint      `json:"user_id"`
	User              *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SessionID         *uint      `json:"session_id"`
	Session           *Session   `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	AmountKes         float64    `gorm:"type:decimal(10,2);not null" json:"amount_kes"`
	PhoneNumber       string     `gorm:"not null;type:varchar(15)" json:"phone_number"`
	Status            string     `gorm:"type:varchar(50);default:'pending';index" json:"status"` // pending, completed, failed
	RawCallback       string     `gorm:"type:text" json:"raw_callback,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Router struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	IpAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	Location  string    `json:"location"`
	Status    string    `gorm:"type:varchar(50);default:'online'" json:"status"` // online, offline
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Voucher struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Code         string     `gorm:"uniqueIndex;not null;type:varchar(16)" json:"code"`
	PlanID       uint       `gorm:"not null" json:"plan_id"`
	Plan         Plan       `gorm:"foreignKey:PlanID" json:"plan"`
	Status       string     `gorm:"type:varchar(50);default:'active';index" json:"status"` // active, used, expired
	UsedByDevice *string    `gorm:"type:varchar(17)" json:"used_by_device"`
	UsedAt       *time.Time `json:"used_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Action      string    `gorm:"not null" json:"action"`
	PerformedBy string    `gorm:"type:varchar(100)" json:"performed_by"`
	Details     string    `gorm:"type:text" json:"details"`
	CreatedAt   time.Time `json:"created_at"`
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `json:"user_id"` // null for admin notification
	Title     string    `gorm:"not null" json:"title"`
	Message   string    `gorm:"type:text" json:"message"`
	Read      bool      `gorm:"default:false" json:"read"`
	CreatedAt time.Time `json:"created_at"`
}
