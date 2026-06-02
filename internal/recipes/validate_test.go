package recipes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "recipe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name         string
		content      string
		wantErrors   int
		wantWarnings int
	}{
		{
			name: "valid recipe with all fields",
			content: `name: Test Recipe
description: A test recipe
ingredients:
  - 1 cup flour
  - 2 eggs
directions:
  - Mix ingredients
  - Bake
prep_time: PT15M
cook_time: PT30M
`,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "missing name",
			content: `description: A test recipe
ingredients:
  - 1 cup flour
`,
			wantErrors:   1,
			wantWarnings: 1, // missing directions
		},
		{
			name:         "empty name",
			content:      `name: ""`,
			wantErrors:   1,
			wantWarnings: 3, // missing description, ingredients, directions
		},
		{
			name:         "whitespace only name",
			content:      `name: "   "`,
			wantErrors:   1,
			wantWarnings: 3,
		},
		{
			name: "missing recommended fields",
			content: `name: Test Recipe
`,
			wantErrors:   0,
			wantWarnings: 3, // missing description, ingredients, directions
		},
		{
			name: "invalid prep_time format",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
prep_time: 15 minutes
`,
			wantErrors:   0,
			wantWarnings: 1, // invalid prep_time format
		},
		{
			name: "invalid cook_time format",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
cook_time: 30m
`,
			wantErrors:   0,
			wantWarnings: 1, // invalid cook_time format
		},
		{
			name: "valid ISO 8601 times",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
prep_time: PT15M
cook_time: PT1H30M
`,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "invalid image URL",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
image: not-a-valid-url
`,
			wantErrors:   0,
			wantWarnings: 1, // invalid image URL
		},
		{
			name: "valid http image URL",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
image: https://example.com/image.jpg
`,
			wantErrors:   0,
			wantWarnings: 0,
		},
		{
			name: "valid local image path",
			content: `name: Test Recipe
description: A test
ingredients:
  - 1 cup flour
directions:
  - Mix
image: /images/recipes/test.jpg
`,
			wantErrors:   0,
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			testFile := filepath.Join(tmpDir, "test-recipe.yml")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			result := ValidateFile(testFile)

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("ValidateFile() errors = %d, want %d; errors: %v",
					len(result.Errors), tt.wantErrors, result.Errors)
			}

			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("ValidateFile() warnings = %d, want %d; warnings: %v",
					len(result.Warnings), tt.wantWarnings, result.Warnings)
			}
		})
	}
}

func TestValidateFile_InvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "recipe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "invalid.yml")
	invalidYAML := `name: Test
  invalid indentation: here
another: field`

	if err := os.WriteFile(testFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result := ValidateFile(testFile)

	if len(result.Errors) == 0 {
		t.Error("ValidateFile() expected YAML parse error, got none")
	}
}

func TestValidateFile_NonexistentFile(t *testing.T) {
	result := ValidateFile("/nonexistent/path/to/file.yml")

	if len(result.Errors) == 0 {
		t.Error("ValidateFile() expected file read error, got none")
	}
}

func TestValidateDirectoryWithError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "recipe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create some test recipe files
	validRecipe := `name: Valid Recipe
description: A valid recipe
ingredients:
  - 1 cup flour
directions:
  - Mix
`
	invalidRecipe := `name: ""
`

	if err := os.WriteFile(filepath.Join(tmpDir, "valid.yml"), []byte(validRecipe), 0644); err != nil {
		t.Fatalf("Failed to write valid recipe: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte(invalidRecipe), 0644); err != nil {
		t.Fatalf("Failed to write invalid recipe: %v", err)
	}

	results, summary, err := ValidateDirectoryWithError(tmpDir)

	if err != nil {
		t.Errorf("ValidateDirectoryWithError() error = %v", err)
	}

	if summary.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", summary.TotalFiles)
	}

	if summary.FilesWithErrors != 1 {
		t.Errorf("FilesWithErrors = %d, want 1", summary.FilesWithErrors)
	}

	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}

func TestValidateDirectory_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "recipe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	results, summary := ValidateDirectory(tmpDir)

	if summary.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", summary.TotalFiles)
	}

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}
