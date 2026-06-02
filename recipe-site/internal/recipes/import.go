package recipes

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// MaxZIPFileSize is the maximum allowed size for a single file within a ZIP (100MB)
	MaxZIPFileSize = 100 * 1024 * 1024
	// MaxTotalExtractSize is the maximum total size for all extracted files (1GB)
	MaxTotalExtractSize = 1024 * 1024 * 1024
)

// ImportResult contains the result of importing a ZIP file.
type ImportResult struct {
	ZIPPath       string
	ExtractedPath string
	RecipeCount   int
}

// FindImportZIP searches for ZIP files in the imports directory.
// Returns the path to the most recent ZIP file (by modification time),
// or an error if no ZIP files are found.
func FindImportZIP(importsDir string) (string, error) {
	pattern := filepath.Join(importsDir, "*.zip")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for ZIP files: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no ZIP files found in %s", importsDir)
	}

	// Sort by modification time (most recent first)
	type fileWithTime struct {
		path    string
		modTime int64
	}
	var files []fileWithTime
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		files = append(files, fileWithTime{path: match, modTime: info.ModTime().UnixNano()})
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no accessible ZIP files found in %s", importsDir)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	// Warn if multiple ZIP files exist
	if len(files) > 1 {
		Logger.Warn("Multiple ZIP files found, using most recent",
			"selected", filepath.Base(files[0].path),
			"total", len(files))
	}

	return files[0].path, nil
}

// ValidateZIPStructure checks if a ZIP file contains YAML recipe files.
// Returns an error if the ZIP is invalid or doesn't contain recipe files.
// This is a convenience wrapper that uses context.Background().
func ValidateZIPStructure(zipPath string) error {
	return ValidateZIPStructureWithContext(context.Background(), zipPath)
}

// ValidateZIPStructureWithContext checks if a ZIP file contains YAML recipe files.
// Returns an error if the ZIP is invalid, doesn't contain recipe files, or context is cancelled.
func ValidateZIPStructureWithContext(ctx context.Context, zipPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	ymlCount := 0
	for _, file := range reader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".yml") ||
			strings.HasSuffix(strings.ToLower(file.Name), ".yaml") {
			ymlCount++
		}
	}

	if ymlCount == 0 {
		return fmt.Errorf("ZIP file does not contain any YAML recipe files")
	}

	Logger.Info("Validated ZIP structure", "recipes", ymlCount)
	return nil
}

// ExtractZIP extracts a ZIP file to the specified destination directory.
// Creates a subdirectory named after the ZIP file (without .zip extension).
// Returns the path to the extracted directory.
// This is a convenience wrapper that uses context.Background().
func ExtractZIP(zipPath, destDir string) (string, error) {
	return ExtractZIPWithContext(context.Background(), zipPath, destDir)
}

// ExtractZIPWithContext extracts a ZIP file to the specified destination directory.
// Creates a subdirectory named after the ZIP file (without .zip extension).
// Returns the path to the extracted directory.
// Supports cancellation via context.
func ExtractZIPWithContext(ctx context.Context, zipPath, destDir string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Always extract to a fixed directory name
	extractPath := filepath.Join(destDir, "CookBook-Recipes-YAML")

	// Remove existing extracted directory if it exists
	if _, err := os.Stat(extractPath); err == nil {
		Logger.Info("Removing existing extracted directory", "path", extractPath)
		if err := os.RemoveAll(extractPath); err != nil {
			return "", fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Open the ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	// Create extraction directory
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create extraction directory: %w", err)
	}

	extractedCount := 0
	var totalExtracted int64 = 0

	for _, file := range reader.File {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Security: prevent path traversal
		cleanName := filepath.Clean(file.Name)
		if strings.Contains(cleanName, "..") {
			Logger.Warn("Skipping potentially unsafe path", "path", file.Name)
			continue
		}

		// Security: check file size to prevent ZIP bombs
		if file.UncompressedSize64 > MaxZIPFileSize {
			Logger.Warn("Skipping file exceeding size limit",
				"file", file.Name,
				"size", file.UncompressedSize64,
				"limit", MaxZIPFileSize)
			continue
		}

		// Security: check total extraction size
		if totalExtracted+int64(file.UncompressedSize64) > MaxTotalExtractSize {
			return "", fmt.Errorf("extraction aborted: total size would exceed %d bytes limit", MaxTotalExtractSize)
		}

		// Extract directly to the extraction directory (flatten structure)
		destPath := filepath.Join(extractPath, filepath.Base(cleanName))

		written, err := extractZIPFile(file, destPath)
		if err != nil {
			Logger.Warn("Failed to extract file", "file", file.Name, "error", err)
			continue
		}

		totalExtracted += written
		extractedCount++
	}

	Logger.Info("Extraction complete", "files", extractedCount, "destination", extractPath)
	return extractPath, nil
}

// extractZIPFile extracts a single file from a ZIP archive to destPath.
// Returns the number of bytes written.
func extractZIPFile(file *zip.File, destPath string) (int64, error) {
	src, err := file.Open()
	if err != nil {
		return 0, fmt.Errorf("opening file in ZIP: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("creating destination file: %w", err)
	}
	defer dst.Close()

	written, err := io.CopyN(dst, src, MaxZIPFileSize)
	if err != nil && err != io.EOF {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("copying contents: %w", err)
	}

	return written, nil
}

// CleanupImports removes extracted directories and optionally ZIP files from the imports directory.
func CleanupImports(importsDir string, removeZIPs bool) error {
	// Remove .extracted directory
	extractedDir := filepath.Join(importsDir, ".extracted")
	if _, err := os.Stat(extractedDir); err == nil {
		Logger.Info("Removing extracted directory", "path", extractedDir)
		if err := os.RemoveAll(extractedDir); err != nil {
			return fmt.Errorf("failed to remove extracted directory: %w", err)
		}
	}

	// Optionally remove ZIP files
	if removeZIPs {
		pattern := filepath.Join(importsDir, "*.zip")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to find ZIP files for cleanup: %w", err)
		}
		for _, zipFile := range matches {
			Logger.Info("Removing ZIP file", "path", zipFile)
			if err := os.Remove(zipFile); err != nil {
				Logger.Warn("Failed to remove ZIP file", "path", zipFile, "error", err)
			}
		}
	}

	return nil
}

// Import performs the complete import workflow:
// 1. Finds ZIP file in imports directory
// 2. Validates the ZIP structure
// 3. Extracts to imports/.extracted/
// Returns the path to the extracted YAML directory.
// This is a convenience wrapper that uses context.Background().
func Import(importsDir string) (*ImportResult, error) {
	return ImportWithContext(context.Background(), importsDir)
}

// ImportWithContext performs the complete import workflow with context support:
// 1. Finds ZIP file in imports directory
// 2. Validates the ZIP structure
// 3. Extracts to imports/.extracted/
// Returns the path to the extracted YAML directory.
// Supports cancellation via context.
func ImportWithContext(ctx context.Context, importsDir string) (*ImportResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Find ZIP file
	zipPath, err := FindImportZIP(importsDir)
	if err != nil {
		return nil, err
	}
	Logger.Info("Found ZIP file", "path", zipPath)

	// Validate ZIP structure
	if err := ValidateZIPStructureWithContext(ctx, zipPath); err != nil {
		return nil, err
	}

	// Extract to .extracted subdirectory
	extractedDir := filepath.Join(importsDir, ".extracted")
	extractedPath, err := ExtractZIPWithContext(ctx, zipPath, extractedDir)
	if err != nil {
		return nil, err
	}

	// Count YAML files in extracted directory
	ymlFiles, _ := filepath.Glob(filepath.Join(extractedPath, "*.yml"))
	yamlFiles, _ := filepath.Glob(filepath.Join(extractedPath, "*.yaml"))
	recipeCount := len(ymlFiles) + len(yamlFiles)

	return &ImportResult{
		ZIPPath:       zipPath,
		ExtractedPath: extractedPath,
		RecipeCount:   recipeCount,
	}, nil
}
