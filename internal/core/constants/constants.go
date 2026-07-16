package constants

const (
	// HTTP Client Configuration
	DefaultHTTPTimeout        = 12 // seconds — scrape + redirect from Render needs headroom
	DefaultHTTPMaxIdleConns   = 100
	DefaultHTTPMaxIdlePerHost = 10
	DefaultHTTPIdleTimeout    = 90 // seconds

	// Database Configuration
	DefaultDBPort              = 5432
	DefaultDBMaxOpenConns      = 25
	DefaultDBMaxIdleConns      = 10
	DefaultDBConnMaxLifetime   = 5  // minutes
	DefaultDBConnMaxIdleTime   = 10 // minutes

	// Search Service Configuration
	DefaultSearchTimeout   = 2 // seconds — drop slow providers, don't stall /search
	DefaultMaxSearchResults = 20
	DefaultMaxPageNumber    = 100
	MaxQueryLength          = 200
	MinQueryLength          = 1

	// Server Configuration
	DefaultServerPort  = "8080"
	DefaultReadTimeout = 10  // seconds
	DefaultWriteTimeout = 10 // seconds
	DefaultIdleTimeout  = 120 // seconds

	// Request Limits
	MaxRequestSize = 1024 * 1024 // 1MB

	// Validation
	MinUsernameLength = 1
	MaxUsernameLength = 100
)
