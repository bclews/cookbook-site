package recipes

import "time"

// Configuration constants for the recipe processing system.
// These values control various limits, timeouts, and behaviors.

// ============================================================================
// Download Configuration
// ============================================================================

// DefaultParallelDownloads is the default number of concurrent image downloads.
const DefaultParallelDownloads = 10

// RateLimitPerSecond is the maximum number of download requests per second.
const RateLimitPerSecond = 5

// RateLimitBurst is the burst capacity for rate limiting.
const RateLimitBurst = 10

// DownloadTimeout is the maximum time to wait for an image download.
const DownloadTimeout = 30 * time.Second

// MaxRetries is the number of retry attempts for failed downloads.
const MaxRetries = 3

// RetryBaseDelay is the base delay for exponential backoff (doubles each retry).
const RetryBaseDelay = 1 * time.Second

// MaxRedirects is the maximum number of HTTP redirects to follow.
const MaxRedirects = 5

// ============================================================================
// Circuit Breaker Configuration
// ============================================================================

// CircuitBreakerThreshold is the failure rate (0.0-1.0) that triggers the circuit breaker.
// If more than this percentage of downloads fail after processing MinSamples, stop downloading.
const CircuitBreakerThreshold = 0.5

// CircuitBreakerMinSamples is the minimum number of downloads before circuit breaker activates.
const CircuitBreakerMinSamples = 10

// ============================================================================
// HTTP Client Configuration
// ============================================================================

// MaxIdleConns is the maximum number of idle HTTP connections.
const MaxIdleConns = 100

// MaxIdleConnsPerHost is the maximum idle connections per host.
const MaxIdleConnsPerHost = 10

// IdleConnTimeout is how long to keep idle connections open.
const IdleConnTimeout = 90 * time.Second

// DialTimeout is the maximum time to establish a connection.
const DialTimeout = 30 * time.Second

// KeepAliveInterval is the interval for TCP keep-alive probes.
const KeepAliveInterval = 30 * time.Second

// ============================================================================
// ZIP Import Configuration
// ============================================================================
// The ZIP extraction limits MaxZIPFileSize (100MB per file) and
// MaxTotalExtractSize (1GB total) live next to the extraction code in
// import.go.

// ============================================================================
// Conversion Configuration
// ============================================================================

// ConversionProgressInterval is how often to log progress during batch conversion.
const ConversionProgressInterval = 50

// ImageURLPrefix is the URL path prefix for recipe images in the Hugo site.
const ImageURLPrefix = "/images/recipes/"
