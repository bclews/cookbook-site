package recipes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateFile validates a single YAML recipe file.
func ValidateFile(path string) ValidationResult {
	result := ValidationResult{File: filepath.Base(path)}

	data, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read file: %v", err))
		return result
	}

	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("YAML parse error: %v", err))
		return result
	}

	// Check required field: name
	if recipe.Name == "" {
		result.Errors = append(result.Errors, "Missing required field: name")
	} else if strings.TrimSpace(recipe.Name) == "" {
		result.Errors = append(result.Errors, "Required field 'name' is empty")
	}

	// Check recommended fields
	if recipe.Description == "" {
		result.Warnings = append(result.Warnings, "Missing recommended field: description")
	}
	if len(recipe.Ingredients) == 0 {
		result.Warnings = append(result.Warnings, "Missing recommended field: ingredients")
	}
	if len(recipe.Directions) == 0 {
		result.Warnings = append(result.Warnings, "Missing recommended field: directions")
	}

	// Validate time formats
	if recipe.PrepTime != "" && !strings.HasPrefix(recipe.PrepTime, "PT") {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("prep_time '%s' is not in ISO 8601 format (should start with 'PT')", recipe.PrepTime))
	}
	if recipe.CookTime != "" && !strings.HasPrefix(recipe.CookTime, "PT") {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("cook_time '%s' is not in ISO 8601 format (should start with 'PT')", recipe.CookTime))
	}

	// Validate image URL
	if recipe.Image != "" {
		if !strings.HasPrefix(recipe.Image, "http://") &&
			!strings.HasPrefix(recipe.Image, "https://") &&
			!strings.HasPrefix(recipe.Image, "/") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("image '%s' may not be a valid URL or path", recipe.Image))
		}
	}

	return result
}

// ValidateDirectory validates all YAML files in a directory.
// Directory scan errors are logged but not returned; use ValidateDirectoryWithError
// if you need to handle scan errors.
func ValidateDirectory(dir string) ([]ValidationResult, ValidationSummary) {
	results, summary, err := ValidateDirectoryWithError(dir)
	if err != nil {
		Logger.Error("Failed to scan directory for validation", "dir", dir, "error", err)
	}
	return results, summary
}

// ValidateDirectoryWithError validates all YAML files and returns detailed error information.
func ValidateDirectoryWithError(dir string) ([]ValidationResult, ValidationSummary, error) {
	pattern := filepath.Join(dir, "*.yml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, ValidationSummary{}, fmt.Errorf("failed to scan directory %s: %w", dir, err)
	}

	var results []ValidationResult
	var summary ValidationSummary
	summary.TotalFiles = len(files)

	for _, file := range files {
		result := ValidateFile(file)
		results = append(results, result)

		if len(result.Errors) > 0 {
			summary.FilesWithErrors++
			summary.TotalErrors += len(result.Errors)
		}
		if len(result.Warnings) > 0 {
			summary.FilesWithWarnings++
			summary.TotalWarnings += len(result.Warnings)
		}
	}

	return results, summary, nil
}
