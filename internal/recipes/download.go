package recipes

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// ImageDownloader handles parallel image downloads.
type ImageDownloader struct {
	imagesDir     string
	maxWorkers    int
	stats         DownloadStats
	mu            sync.Mutex
	client        *http.Client
	limiter       *rate.Limiter
	allowTestURLs bool // Allow localhost URLs for testing
}

// ImageDownloaderOption configures an ImageDownloader.
type ImageDownloaderOption func(*ImageDownloader)

// WithAllowTestURLs enables localhost URLs for testing purposes.
func WithAllowTestURLs(allow bool) ImageDownloaderOption {
	return func(d *ImageDownloader) {
		d.allowTestURLs = allow
	}
}

// NewImageDownloader creates a new downloader with the specified concurrency.
// Options can be provided to customize behavior (e.g., WithAllowTestURLs for testing).
func NewImageDownloader(imagesDir string, maxWorkers int, opts ...ImageDownloaderOption) *ImageDownloader {
	d := &ImageDownloader{
		imagesDir:  imagesDir,
		maxWorkers: maxWorkers,
		limiter:    rate.NewLimiter(rate.Limit(RateLimitPerSecond), RateLimitBurst),
	}

	// Create HTTP transport with security hardening. The dialer's Control hook
	// rejects connections to non-public addresses at dial time, which closes
	// the SSRF gaps that a URL-only check leaves open: redirects to internal
	// hosts and hostnames that resolve to private IPs (DNS rebinding).
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:        MaxIdleConns,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		IdleConnTimeout:     IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout:   DialTimeout,
			KeepAlive: KeepAliveInterval,
			Control:   d.dialControl,
		}).DialContext,
	}

	d.client = &http.Client{
		Timeout:   DownloadTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("too many redirects (max %d)", MaxRedirects)
			}
			return nil
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(d)
	}

	return d
}

// dialControl runs after DNS resolution, just before the socket connects. It
// blocks loopback, private, link-local (including the 169.254.169.254 cloud
// metadata endpoint), and unspecified addresses. Because every dial — initial
// request or redirect — passes through here, it guards against redirect-based
// SSRF and DNS rebinding that the URL-level check cannot catch.
func (d *ImageDownloader) dialControl(network, address string, _ syscall.RawConn) error {
	if d.allowTestURLs {
		return nil
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("parsing dial address %q: %w", address, err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("refusing to connect to unresolved address %q", host)
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("refusing to connect to non-public address %s", ip)
	}

	return nil
}

// downloadTask represents a single image download job.
type downloadTask struct {
	URL        string
	RecipeName string
}

// downloadResult holds the result of a download attempt.
type downloadResult struct {
	URL       string
	LocalPath string
	Success   bool
}

// isValidImageURL validates that an image URL is safe to download.
func (d *ImageDownloader) isValidImageURL(imageURL string) bool {
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return false
	}

	// Only allow HTTPS and HTTP schemes
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}

	host := parsed.Hostname()

	// Allow test URLs (localhost/loopback) when testing
	if d.allowTestURLs {
		return true
	}

	// Block localhost and loopback addresses
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}

	// Block private and loopback IP ranges
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() {
			return false
		}
	}

	return true
}

// DownloadImages downloads all images in parallel and returns a URL-to-local-path map.
// Uses a background context. For cancellation support, use DownloadImagesWithContext.
func (d *ImageDownloader) DownloadImages(recipes []RecipeFile) map[string]string {
	return d.DownloadImagesWithContext(context.Background(), recipes)
}

// DownloadImagesWithContext downloads all images in parallel with context support for cancellation.
func (d *ImageDownloader) DownloadImagesWithContext(ctx context.Context, recipes []RecipeFile) map[string]string {
	imageMap := make(map[string]string)
	var mapMu sync.Mutex

	// Collect download tasks
	var tasks []downloadTask
	for _, rf := range recipes {
		if rf.Recipe.Image != "" && strings.HasPrefix(rf.Recipe.Image, "http") {
			tasks = append(tasks, downloadTask{
				URL:        rf.Recipe.Image,
				RecipeName: rf.Recipe.Name,
			})
		}
	}

	if len(tasks) == 0 {
		Logger.Info("No images to download")
		return imageMap
	}

	Logger.Info("Starting image downloads", "count", len(tasks), "maxWorkers", d.maxWorkers)

	// Wrap context so the circuit breaker can cancel in-flight downloads
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create worker pool
	taskChan := make(chan downloadTask, len(tasks))
	resultChan := make(chan downloadResult, len(tasks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Check for cancellation before processing
				select {
				case <-ctx.Done():
					resultChan <- downloadResult{
						URL:     task.URL,
						Success: false,
					}
					continue
				default:
				}

				localPath, success := d.downloadImageWithContext(ctx, task.URL, task.RecipeName)
				resultChan <- downloadResult{
					URL:       task.URL,
					LocalPath: localPath,
					Success:   success,
				}
			}
		}()
	}

	// Send tasks (check for cancellation)
	go func() {
		for _, task := range tasks {
			select {
			case <-ctx.Done():
				// Context cancelled, stop sending tasks
				close(taskChan)
				return
			case taskChan <- task:
			}
		}
		close(taskChan)
	}()

	// Wait for completion in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results with circuit breaker
	processed := 0
	failed := 0
	circuitBroken := false

	for result := range resultChan {
		processed++
		if result.Success && result.LocalPath != "" {
			mapMu.Lock()
			imageMap[result.URL] = result.LocalPath
			mapMu.Unlock()
		} else {
			failed++
		}

		// Check circuit breaker: stop if failure rate exceeds threshold
		if !circuitBroken && processed >= CircuitBreakerMinSamples {
			failureRate := float64(failed) / float64(processed)
			if failureRate > CircuitBreakerThreshold {
				circuitBroken = true
				Logger.Warn("Circuit breaker triggered: high failure rate",
					"failed", failed,
					"processed", processed,
					"rate", fmt.Sprintf("%.1f%%", failureRate*100))
				// Cancel the context to stop in-flight and queued downloads
				cancel()
			}
		}
	}

	if circuitBroken {
		Logger.Info("Download completed with circuit breaker active",
			"successful", len(imageMap),
			"failed", failed,
			"total_processed", processed)
	}

	return imageMap
}

// downloadImageWithContext downloads a single image with context support for cancellation.
// Includes URL validation, rate limiting, and retry logic with exponential backoff.
func (d *ImageDownloader) downloadImageWithContext(ctx context.Context, imageURL, recipeName string) (string, bool) {
	// Validate URL for security
	if !d.isValidImageURL(imageURL) {
		Logger.Warn("Invalid or unsafe image URL", "url", imageURL, "recipe", recipeName)
		d.recordFailure()
		return "", false
	}

	// Determine file extension from URL path
	ext := ".jpg" // default
	if urlExt := strings.ToLower(path.Ext(imageURL)); urlExt != "" {
		switch urlExt {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			ext = urlExt
		}
	}

	// Generate unique filename
	// Note: MD5 is used here only for cache-busting, not for cryptographic purposes
	hash := fmt.Sprintf("%x", md5.Sum([]byte(imageURL)))[:8]
	safeName := SanitizeFilename(recipeName)
	filename := fmt.Sprintf("%s-%s%s", safeName, hash, ext)
	localPath := filepath.Join(d.imagesDir, filename)

	// Check if already downloaded
	if _, err := os.Stat(localPath); err == nil {
		d.recordSkipped()
		return ImageURLPrefix + filename, true
	}

	// Apply rate limiting
	if err := d.limiter.Wait(ctx); err != nil {
		d.recordFailure()
		return "", false
	}

	// Download with retry logic and exponential backoff
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		// Apply exponential backoff before each retry: RetryBaseDelay doubled
		// per attempt (with the defaults, 1s then 2s).
		if attempt > 0 {
			backoff := RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			Logger.Debug("Retrying download", "url", imageURL, "attempt", attempt+1, "backoff", backoff)
			select {
			case <-ctx.Done():
				d.recordFailure()
				return "", false
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "recipe-tool/1.0 (+https://github.com/bclews/cookbook-site)")

		resp, lastErr = d.client.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break // Success
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		// Log retry-able errors
		if lastErr != nil {
			Logger.Debug("Download attempt failed", "url", imageURL, "attempt", attempt+1, "error", lastErr)
		} else if resp != nil {
			Logger.Debug("Download attempt failed", "url", imageURL, "attempt", attempt+1, "status", resp.StatusCode)
		}
	}

	// Check if all retries failed
	if lastErr != nil || resp == nil || resp.StatusCode != http.StatusOK {
		d.recordFailure()
		if lastErr != nil {
			Logger.Warn("Failed to download image after retries", "url", imageURL, "error", lastErr)
		} else if resp != nil {
			Logger.Warn("Failed to download image after retries", "url", imageURL, "status", resp.StatusCode)
		}
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	// Write to file
	out, err := os.Create(localPath)
	if err != nil {
		d.recordFailure()
		return "", false
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		d.recordFailure()
		_ = os.Remove(localPath)
		return "", false
	}

	d.recordDownloaded()

	return ImageURLPrefix + filename, true
}

// GetStats returns download statistics.
func (d *ImageDownloader) GetStats() DownloadStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stats
}

// recordFailure increments the failed-download counter.
func (d *ImageDownloader) recordFailure() {
	d.mu.Lock()
	d.stats.Failed++
	d.mu.Unlock()
}

// recordSkipped increments the skipped (already cached) counter.
func (d *ImageDownloader) recordSkipped() {
	d.mu.Lock()
	d.stats.Skipped++
	d.mu.Unlock()
}

// recordDownloaded increments the downloaded counter and logs progress
// every 50 images.
func (d *ImageDownloader) recordDownloaded() {
	d.mu.Lock()
	d.stats.Downloaded++
	count := d.stats.Downloaded
	d.mu.Unlock()
	if count%50 == 0 {
		Logger.Info("Download progress", "downloaded", count)
	}
}
