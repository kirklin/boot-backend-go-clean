package entity

import "time"

// LoginActivity records a single user login event along with
// the resolved IP geolocation. This is a common requirement in
// Chinese internet applications for security auditing and analytics.
type LoginActivity struct {
	ID        int64     `json:"id,string"`
	UserID    int64     `json:"user_id,string"`
	LoginAt   time.Time `json:"login_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`

	// GeoIP fields resolved at login time
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	ISP      string `json:"isp"`

	CreatedAt time.Time `json:"created_at"`
}
