package model

import (
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Country   string    `json:"country" db:"country"`
	City      *string   `json:"city,omitempty" db:"city"`
	FlagEmoji string    `json:"flag_emoji" db:"flag_emoji"`

	// XUI Panel connection (hidden from regular users)
	XUIBaseURL   string `json:"-" db:"xui_base_url"`
	XUIUsername  string `json:"-" db:"xui_username"`
	XUIPassword  string `json:"-" db:"xui_password"`
	XUIInboundID int    `json:"-" db:"xui_inbound_id"`

	// Server connection details
	ServerAddress string `json:"server_address" db:"server_address"`
	ServerPort    int    `json:"server_port" db:"server_port"`
	PublicKey     string `json:"-" db:"public_key"`
	ShortID       string `json:"-" db:"short_id"`
	ServerName    string `json:"-" db:"server_name"`

	// Status
	IsActive  bool `json:"is_active" db:"is_active"`
	SortOrder int  `json:"sort_order" db:"sort_order"`

	// Capacity and health
	Capacity    int        `json:"capacity" db:"capacity"`
	CurrentLoad int        `json:"current_load" db:"current_load"`
	PingMs      *int       `json:"ping_ms,omitempty" db:"ping_ms"`
	Status      string     `json:"status" db:"status"` // online, offline, unknown
	LastCheckAt *time.Time `json:"last_check_at,omitempty" db:"last_check_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LoadPercent returns server load as percentage
func (s *Server) LoadPercent() float64 {
	if s.Capacity <= 0 {
		return 0
	}
	return float64(s.CurrentLoad) / float64(s.Capacity) * 100
}

// IsOnline returns true if server is online and active
func (s *Server) IsOnline() bool {
	return s.IsActive && s.Status == "online"
}

// ServerPublic is the public view of server for users (without sensitive data)
type ServerPublic struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Country     string    `json:"country"`
	City        *string   `json:"city,omitempty"`
	FlagEmoji   string    `json:"flag_emoji"`
	IsActive    bool      `json:"is_active"`
	PingMs      *int      `json:"ping_ms,omitempty"`
	Status      string    `json:"status"`
	LoadPercent float64   `json:"load_percent"`
}

// ToPublic converts Server to ServerPublic
func (s *Server) ToPublic() ServerPublic {
	return ServerPublic{
		ID:          s.ID,
		Name:        s.Name,
		Country:     s.Country,
		City:        s.City,
		FlagEmoji:   s.FlagEmoji,
		IsActive:    s.IsActive,
		PingMs:      s.PingMs,
		Status:      s.Status,
		LoadPercent: s.LoadPercent(),
	}
}

// ServerAdmin is the admin view of server (with sensitive data)
type ServerAdmin struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Country       string     `json:"country"`
	City          *string    `json:"city,omitempty"`
	FlagEmoji     string     `json:"flag_emoji"`
	XUIBaseURL    string     `json:"xui_base_url"`
	XUIUsername   string     `json:"xui_username"`
	XUIPassword   string     `json:"xui_password"`
	XUIInboundID  int        `json:"xui_inbound_id"`
	ServerAddress string     `json:"server_address"`
	ServerPort    int        `json:"server_port"`
	PublicKey     string     `json:"public_key"`
	ShortID       string     `json:"short_id"`
	ServerName    string     `json:"server_name"`
	IsActive      bool       `json:"is_active"`
	SortOrder     int        `json:"sort_order"`
	Capacity      int        `json:"capacity"`
	CurrentLoad   int        `json:"current_load"`
	PingMs        *int       `json:"ping_ms,omitempty"`
	Status        string     `json:"status"`
	LastCheckAt   *time.Time `json:"last_check_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToAdmin converts Server to ServerAdmin
func (s *Server) ToAdmin() ServerAdmin {
	return ServerAdmin{
		ID:            s.ID,
		Name:          s.Name,
		Country:       s.Country,
		City:          s.City,
		FlagEmoji:     s.FlagEmoji,
		XUIBaseURL:    s.XUIBaseURL,
		XUIUsername:   s.XUIUsername,
		XUIPassword:   s.XUIPassword,
		XUIInboundID:  s.XUIInboundID,
		ServerAddress: s.ServerAddress,
		ServerPort:    s.ServerPort,
		PublicKey:     s.PublicKey,
		ShortID:       s.ShortID,
		ServerName:    s.ServerName,
		IsActive:      s.IsActive,
		SortOrder:     s.SortOrder,
		Capacity:      s.Capacity,
		CurrentLoad:   s.CurrentLoad,
		PingMs:        s.PingMs,
		Status:        s.Status,
		LastCheckAt:   s.LastCheckAt,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
