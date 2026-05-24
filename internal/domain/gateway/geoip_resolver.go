package gateway

// GeoLocation represents the geographic location information resolved from an IP address.
type GeoLocation struct {
	Country  string // Country name (e.g., "中国")
	Region   string // Region / area (often "0" meaning empty)
	Province string // Province (e.g., "北京")
	City     string // City (e.g., "北京市")
	ISP      string // Internet Service Provider (e.g., "电信")
}

// GeoIPResolver defines the interface for resolving IP addresses to geographic locations.
// This interface belongs to the domain layer, ensuring that usecases do not depend on
// specific GeoIP database implementations (ip2region, MaxMind, etc.).
type GeoIPResolver interface {
	// Resolve looks up the geographic location for the given IP address.
	// Returns an empty GeoLocation (all fields "") if the IP cannot be resolved.
	Resolve(ip string) GeoLocation

	// Close releases any resources held by the resolver (e.g., memory-mapped files).
	Close() error
}
