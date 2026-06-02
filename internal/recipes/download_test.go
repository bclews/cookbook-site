package recipes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsValidImageURL(t *testing.T) {
	// Create a downloader without test URL allowance for security tests
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 1)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid HTTPS URL",
			url:      "https://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "valid HTTP URL",
			url:      "http://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "localhost blocked",
			url:      "http://localhost/image.jpg",
			expected: false,
		},
		{
			name:     "127.0.0.1 blocked",
			url:      "http://127.0.0.1/image.jpg",
			expected: false,
		},
		{
			name:     "IPv6 loopback blocked",
			url:      "http://[::1]/image.jpg",
			expected: false,
		},
		{
			name:     "private IP 10.x blocked",
			url:      "http://10.0.0.1/image.jpg",
			expected: false,
		},
		{
			name:     "private IP 192.168.x blocked",
			url:      "http://192.168.1.1/image.jpg",
			expected: false,
		},
		{
			name:     "private IP 172.16.x blocked",
			url:      "http://172.16.0.1/image.jpg",
			expected: false,
		},
		{
			name:     "file scheme blocked",
			url:      "file:///etc/passwd",
			expected: false,
		},
		{
			name:     "ftp scheme blocked",
			url:      "ftp://example.com/image.jpg",
			expected: false,
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downloader.isValidImageURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidImageURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestNewImageDownloader(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5)

	if downloader == nil {
		t.Fatal("NewImageDownloader returned nil")
	}

	if downloader.imagesDir != tempDir {
		t.Errorf("imagesDir = %q, want %q", downloader.imagesDir, tempDir)
	}

	if downloader.maxWorkers != 5 {
		t.Errorf("maxWorkers = %d, want 5", downloader.maxWorkers)
	}

	if downloader.client == nil {
		t.Error("HTTP client is nil")
	}

	if downloader.limiter == nil {
		t.Error("Rate limiter is nil")
	}

	// Verify HTTP client has proper timeout
	if downloader.client.Timeout != 30*time.Second {
		t.Errorf("client.Timeout = %v, want 30s", downloader.client.Timeout)
	}
}

func TestImageDownloader_DownloadImagesWithContext_EmptyList(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5)

	result := downloader.DownloadImagesWithContext(context.Background(), []RecipeFile{})

	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(result))
	}

	stats := downloader.GetStats()
	if stats.Downloaded != 0 || stats.Failed != 0 || stats.Skipped != 0 {
		t.Errorf("Expected zero stats, got Downloaded=%d, Failed=%d, Skipped=%d",
			stats.Downloaded, stats.Failed, stats.Skipped)
	}
}

func TestImageDownloader_DownloadImagesWithContext_Success(t *testing.T) {
	// Create test server that returns a simple image
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		// Write minimal JPEG data
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: server.URL + "/image.jpg",
			},
		},
	}

	result := downloader.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	stats := downloader.GetStats()
	if stats.Downloaded != 1 {
		t.Errorf("Expected 1 downloaded, got %d", stats.Downloaded)
	}
}

func TestImageDownloader_DownloadImagesWithContext_CacheHit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: server.URL + "/image.jpg",
			},
		},
	}

	// First download
	downloader.DownloadImagesWithContext(context.Background(), recipes)

	// Second download should be cached
	downloader2 := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))
	result := downloader2.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 1 {
		t.Errorf("Expected 1 result from cache, got %d", len(result))
	}

	stats := downloader2.GetStats()
	if stats.Skipped != 1 {
		t.Errorf("Expected 1 skipped (cached), got %d", stats.Skipped)
	}
	if stats.Downloaded != 0 {
		t.Errorf("Expected 0 downloaded (should use cache), got %d", stats.Downloaded)
	}
}

func TestImageDownloader_DownloadImagesWithContext_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: server.URL + "/notfound.jpg",
			},
		},
	}

	result := downloader.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 0 {
		t.Errorf("Expected 0 results for failed download, got %d", len(result))
	}

	stats := downloader.GetStats()
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", stats.Failed)
	}
}

func TestImageDownloader_DownloadImagesWithContext_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: server.URL + "/slow.jpg",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := downloader.DownloadImagesWithContext(ctx, recipes)

	// Should return empty due to timeout
	if len(result) != 0 {
		t.Errorf("Expected 0 results due to cancellation, got %d", len(result))
	}
}

func TestImageDownloader_InvalidURL(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5)

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: "http://localhost/image.jpg", // Should be blocked
			},
		},
	}

	result := downloader.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 0 {
		t.Errorf("Expected 0 results for invalid URL, got %d", len(result))
	}

	stats := downloader.GetStats()
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed (invalid URL), got %d", stats.Failed)
	}
}

func TestDialControl(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 1)

	// Addresses are post-DNS-resolution "ip:port" pairs, as the net.Dialer
	// passes them to Control. This guard is what stops redirect-based SSRF and
	// DNS rebinding from reaching internal hosts.
	blocked := []string{
		"127.0.0.1:80",       // loopback
		"10.0.0.5:443",       // private
		"192.168.1.1:80",     // private
		"169.254.169.254:80", // link-local (cloud metadata)
		"0.0.0.0:80",         // unspecified
	}
	for _, addr := range blocked {
		if err := downloader.dialControl("tcp", addr, nil); err == nil {
			t.Errorf("dialControl allowed %q, expected it to be blocked", addr)
		}
	}

	allowed := []string{
		"93.184.216.34:443", // public (example.com)
		"8.8.8.8:53",        // public
	}
	for _, addr := range allowed {
		if err := downloader.dialControl("tcp", addr, nil); err != nil {
			t.Errorf("dialControl blocked public %q: %v", addr, err)
		}
	}

	// With test URLs allowed, even loopback is permitted (used by httptest).
	testDownloader := NewImageDownloader(tempDir, 1, WithAllowTestURLs(true))
	if err := testDownloader.dialControl("tcp", "127.0.0.1:80", nil); err != nil {
		t.Errorf("dialControl with allowTestURLs blocked loopback: %v", err)
	}
}

func TestImageDownloader_FileExtensions(t *testing.T) {
	tests := []struct {
		urlPath string
		wantExt string
	}{
		{"/image.jpg", ".jpg"},
		{"/image.jpeg", ".jpeg"},
		{"/image.png", ".png"},
		{"/image.gif", ".gif"},
		{"/image.webp", ".webp"},
		{"/image.JPG", ".jpg"}, // uppercase
		{"/image", ".jpg"},     // no extension defaults to .jpg
		{"/image.bmp", ".jpg"}, // unsupported extension defaults to .jpg
	}

	for _, tt := range tests {
		t.Run(tt.urlPath, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/jpeg")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
			}))
			defer server.Close()

			tempDir := t.TempDir()
			downloader := NewImageDownloader(tempDir, 5, WithAllowTestURLs(true))

			recipes := []RecipeFile{
				{
					Recipe: &Recipe{
						Name:  "Test Recipe",
						Image: server.URL + tt.urlPath,
					},
				},
			}

			result := downloader.DownloadImagesWithContext(context.Background(), recipes)

			if len(result) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(result))
			}

			// Check that the local path has the expected extension
			for _, localPath := range result {
				ext := filepath.Ext(localPath)
				if ext != tt.wantExt {
					t.Errorf("File extension = %q, want %q", ext, tt.wantExt)
				}
			}
		})
	}
}

func TestImageDownloader_GetStats(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5)

	// Initial stats should be zero
	stats := downloader.GetStats()
	if stats.Downloaded != 0 || stats.Failed != 0 || stats.Skipped != 0 {
		t.Errorf("Initial stats not zero: Downloaded=%d, Failed=%d, Skipped=%d",
			stats.Downloaded, stats.Failed, stats.Skipped)
	}
}

func TestImageDownloader_NoHTTPPrefix(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 5)

	recipes := []RecipeFile{
		{
			Recipe: &Recipe{
				Name:  "Test Recipe",
				Image: "/local/image.jpg", // No http prefix, should be skipped
			},
		},
	}

	result := downloader.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 0 {
		t.Errorf("Expected 0 results for non-HTTP URL, got %d", len(result))
	}
}

func TestImageDownloader_ParallelDownloads(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader := NewImageDownloader(tempDir, 3, WithAllowTestURLs(true)) // 3 workers

	recipes := make([]RecipeFile, 5)
	for i := 0; i < 5; i++ {
		recipes[i] = RecipeFile{
			Recipe: &Recipe{
				Name:  "Test Recipe " + string(rune('A'+i)),
				Image: server.URL + "/image" + string(rune('A'+i)) + ".jpg",
			},
		}
	}

	result := downloader.DownloadImagesWithContext(context.Background(), recipes)

	if len(result) != 5 {
		t.Errorf("Expected 5 results, got %d", len(result))
	}

	stats := downloader.GetStats()
	if stats.Downloaded != 5 {
		t.Errorf("Expected 5 downloaded, got %d", stats.Downloaded)
	}
}
