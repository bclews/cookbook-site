package recipes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "Pasta Recipe",
			want:  "pasta-recipe",
		},
		{
			name:  "name with special characters",
			input: "Mom's Famous Pasta!",
			want:  "moms-famous-pasta",
		},
		{
			name:  "name with multiple spaces",
			input: "Easy   Quick   Pasta",
			want:  "easy-quick-pasta",
		},
		{
			name:  "name with hyphens and spaces",
			input: "Quick-and-Easy Pasta",
			want:  "quick-and-easy-pasta",
		},
		{
			name:  "name with unicode",
			input: "Crème Brûlée",
			want:  "crme-brle",
		},
		{
			name:  "name with numbers",
			input: "15-Minute Pasta Recipe",
			want:  "15-minute-pasta-recipe",
		},
		{
			name:  "name with leading/trailing hyphens",
			input: "-Pasta Recipe-",
			want:  "pasta-recipe",
		},
		{
			name:  "name with ampersand",
			input: "Mac & Cheese",
			want:  "mac-cheese",
		},
		{
			name:  "name with parentheses",
			input: "Pasta (Easy Version)",
			want:  "pasta-easy-version",
		},
		{
			name:  "name with quotes",
			input: `"Best" Pasta Ever`,
			want:  "best-pasta-ever",
		},
		{
			name:  "already lowercase with hyphens",
			input: "simple-pasta",
			want:  "simple-pasta",
		},
		{
			name:  "all caps",
			input: "PASTA RECIPE",
			want:  "pasta-recipe",
		},
		{
			name:  "mixed case",
			input: "PaStA rEcIpE",
			want:  "pasta-recipe",
		},
		{
			name:  "underscores converted",
			input: "pasta_recipe_easy",
			want:  "pasta_recipe_easy",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "!@#$%^&*()",
			want:  "",
		},
		{
			name:  "long name",
			input: "This Is A Very Long Recipe Name That Should Still Work Fine",
			want:  "this-is-a-very-long-recipe-name-that-should-still-work-fine",
		},
		{
			name:  "name with colon",
			input: "Pasta: A Simple Recipe",
			want:  "pasta-a-simple-recipe",
		},
		{
			name:  "name with forward slash",
			input: "Pasta/Noodles Recipe",
			want:  "pastanoodles-recipe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_Idempotent(t *testing.T) {
	// Running SanitizeFilename twice should produce the same result
	inputs := []string{
		"Pasta Recipe",
		"Mom's Famous Pasta!",
		"15-Minute Pasta",
	}

	for _, input := range inputs {
		first := SanitizeFilename(input)
		second := SanitizeFilename(first)

		if first != second {
			t.Errorf("SanitizeFilename is not idempotent: first=%q, second=%q", first, second)
		}
	}
}

func TestLoadRecipes(t *testing.T) {
	t.Run("loads valid recipes", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid recipe files
		recipe1 := `name: Pasta Recipe
description: A simple pasta dish
ingredients:
  - 500g pasta
  - 2 cups sauce
directions:
  - Boil pasta
  - Add sauce
`
		recipe2 := `name: Soup Recipe
description: Warm comfort food
ingredients:
  - Vegetables
  - Broth
`
		if err := os.WriteFile(filepath.Join(tmpDir, "pasta.yml"), []byte(recipe1), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "soup.yml"), []byte(recipe2), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		recipes, err := LoadRecipes(tmpDir)
		if err != nil {
			t.Fatalf("LoadRecipes() error = %v", err)
		}

		if len(recipes) != 2 {
			t.Errorf("LoadRecipes() returned %d recipes, want 2", len(recipes))
		}
	})

	t.Run("skips files with missing name", func(t *testing.T) {
		tmpDir := t.TempDir()

		validRecipe := `name: Valid Recipe
description: This is valid
`
		invalidRecipe := `description: No name field
ingredients:
  - stuff
`
		if err := os.WriteFile(filepath.Join(tmpDir, "valid.yml"), []byte(validRecipe), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte(invalidRecipe), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		recipes, err := LoadRecipes(tmpDir)
		if err != nil {
			t.Fatalf("LoadRecipes() error = %v", err)
		}

		if len(recipes) != 1 {
			t.Errorf("LoadRecipes() returned %d recipes, want 1 (valid only)", len(recipes))
		}
	})

	t.Run("handles invalid YAML gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()

		validRecipe := `name: Valid Recipe
`
		invalidYAML := `name: Invalid
  bad indentation: here
another: value`

		if err := os.WriteFile(filepath.Join(tmpDir, "valid.yml"), []byte(validRecipe), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		recipes, err := LoadRecipes(tmpDir)
		if err != nil {
			t.Fatalf("LoadRecipes() error = %v", err)
		}

		if len(recipes) != 1 {
			t.Errorf("LoadRecipes() returned %d recipes, want 1", len(recipes))
		}
	})

	t.Run("empty directory returns empty slice", func(t *testing.T) {
		tmpDir := t.TempDir()

		recipes, err := LoadRecipes(tmpDir)
		if err != nil {
			t.Fatalf("LoadRecipes() error = %v", err)
		}

		if len(recipes) != 0 {
			t.Errorf("LoadRecipes() returned %d recipes for empty dir, want 0", len(recipes))
		}
	})
}

func TestLoadRecipesWithErrors(t *testing.T) {
	t.Run("returns detailed error information", func(t *testing.T) {
		tmpDir := t.TempDir()

		validRecipe := `name: Valid Recipe
`
		missingName := `description: No name
`
		invalidYAML := `name: Test
  bad: indentation`

		if err := os.WriteFile(filepath.Join(tmpDir, "valid.yml"), []byte(validRecipe), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "missing-name.yml"), []byte(missingName), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := LoadRecipesWithErrors(tmpDir)

		if len(result.Recipes) != 1 {
			t.Errorf("Expected 1 valid recipe, got %d", len(result.Recipes))
		}

		if len(result.Errors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(result.Errors))
		}
	})
}

func TestConvertRecipe(t *testing.T) {
	t.Run("converts full recipe to markdown", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		// Create source file for mod time
		sourceFile := filepath.Join(tmpDir, "source.yml")
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		rf := RecipeFile{
			Path: sourceFile,
			Recipe: &Recipe{
				Name:        "Test Pasta",
				Description: "A delicious pasta recipe",
				Servings:    "4",
				PrepTime:    "PT10M",
				CookTime:    "PT20M",
				Source:      "grandma",
				Image:       "https://example.com/pasta.jpg",
				Tags:        []string{"italian", "pasta"},
				Keywords:    []string{"easy", "quick"},
				Ingredients: []string{"pasta", "sauce"},
				Directions:  []string{"boil pasta", "add sauce"},
				Nutrition:   "500 calories",
				Notes:       "Best served hot",
				Favorite:    true,
				CookCount:   5,
			},
		}

		err := ConvertRecipe(rf, outputDir, nil)
		if err != nil {
			t.Fatalf("ConvertRecipe() error = %v", err)
		}

		// Check output file exists
		outputFile := filepath.Join(outputDir, "test-pasta.md")
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		// Verify front matter content
		contentStr := string(content)
		if !strings.Contains(contentStr, "title: Test Pasta") {
			t.Error("Output missing title")
		}
		if !strings.Contains(contentStr, "description: A delicious pasta recipe") {
			t.Error("Output missing description")
		}
		if !strings.Contains(contentStr, "prep_time: PT10M") {
			t.Error("Output missing prep_time")
		}
		if !strings.Contains(contentStr, "cook_time: PT20M") {
			t.Error("Output missing cook_time")
		}
		if !strings.Contains(contentStr, "favorite: true") {
			t.Error("Output missing favorite")
		}
		if !strings.Contains(contentStr, "cook_count: 5") {
			t.Error("Output missing cook_count")
		}
	})

	t.Run("converts minimal recipe", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		sourceFile := filepath.Join(tmpDir, "source.yml")
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		rf := RecipeFile{
			Path: sourceFile,
			Recipe: &Recipe{
				Name: "Simple Recipe",
			},
		}

		err := ConvertRecipe(rf, outputDir, nil)
		if err != nil {
			t.Fatalf("ConvertRecipe() error = %v", err)
		}

		outputFile := filepath.Join(outputDir, "simple-recipe.md")
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Output file was not created")
		}
	})

	t.Run("uses image map for local paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		sourceFile := filepath.Join(tmpDir, "source.yml")
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		rf := RecipeFile{
			Path: sourceFile,
			Recipe: &Recipe{
				Name:  "Mapped Image Recipe",
				Image: "https://example.com/original.jpg",
			},
		}

		imageMap := map[string]string{
			"https://example.com/original.jpg": "/images/recipes/local.jpg",
		}

		err := ConvertRecipe(rf, outputDir, imageMap)
		if err != nil {
			t.Fatalf("ConvertRecipe() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(outputDir, "mapped-image-recipe.md"))
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}

		if !strings.Contains(string(content), "/images/recipes/local.jpg") {
			t.Error("Output should contain local image path")
		}
		if strings.Contains(string(content), "https://example.com/original.jpg") {
			t.Error("Output should not contain original URL when mapping exists")
		}
	})

	t.Run("filters empty directions", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		sourceFile := filepath.Join(tmpDir, "source.yml")
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		rf := RecipeFile{
			Path: sourceFile,
			Recipe: &Recipe{
				Name:       "Filtered Directions",
				Directions: []string{"Step 1", "", "  ", "Step 2", ""},
			},
		}

		err := ConvertRecipe(rf, outputDir, nil)
		if err != nil {
			t.Fatalf("ConvertRecipe() error = %v", err)
		}

		content, err := os.ReadFile(filepath.Join(outputDir, "filtered-directions.md"))
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}

		// Check that we have exactly 2 non-empty directions
		if strings.Count(string(content), "- Step 1") != 1 {
			t.Error("Should have Step 1")
		}
		if strings.Count(string(content), "- Step 2") != 1 {
			t.Error("Should have Step 2")
		}
	})
}

func TestConvertAll(t *testing.T) {
	t.Run("converts multiple recipes", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		sourceFile := filepath.Join(tmpDir, "source.yml")
		if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		recipes := []RecipeFile{
			{Path: sourceFile, Recipe: &Recipe{Name: "Recipe One"}},
			{Path: sourceFile, Recipe: &Recipe{Name: "Recipe Two"}},
			{Path: sourceFile, Recipe: &Recipe{Name: "Recipe Three"}},
		}

		success, failed := ConvertAll(recipes, outputDir, nil)

		if success != 3 {
			t.Errorf("ConvertAll() success = %d, want 3", success)
		}
		if failed != 0 {
			t.Errorf("ConvertAll() failed = %d, want 0", failed)
		}

		// Verify files exist
		files := []string{"recipe-one.md", "recipe-two.md", "recipe-three.md"}
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist", f)
			}
		}
	})

	t.Run("handles empty recipe list", func(t *testing.T) {
		tmpDir := t.TempDir()

		success, failed := ConvertAll(nil, tmpDir, nil)

		if success != 0 || failed != 0 {
			t.Errorf("ConvertAll() with empty list: success=%d, failed=%d, want 0,0", success, failed)
		}
	})
}

func TestFindYAMLDirectory(t *testing.T) {
	t.Run("uses custom path when specified", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create custom directory
		customDir := filepath.Join(tmpDir, "my-recipes")
		if err := os.MkdirAll(customDir, 0755); err != nil {
			t.Fatalf("Failed to create custom dir: %v", err)
		}

		result, err := FindYAMLDirectory(tmpDir, customDir)
		if err != nil {
			t.Fatalf("FindYAMLDirectory() error = %v", err)
		}

		if result != customDir {
			t.Errorf("FindYAMLDirectory() = %q, want %q", result, customDir)
		}
	})

	t.Run("returns error for non-existent custom path", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := FindYAMLDirectory(tmpDir, "/nonexistent/path")
		if err == nil {
			t.Error("FindYAMLDirectory() expected error for non-existent custom path")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	})

	t.Run("finds imports/.extracted directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create structure: tmpDir/parent/recipe-site (scriptDir)
		//                   tmpDir/parent/imports/.extracted/CookBook-Recipes-YAML/
		parent := filepath.Join(tmpDir, "parent")
		scriptDir := filepath.Join(parent, "recipe-site")
		extractedDir := filepath.Join(parent, "imports", ".extracted", "CookBook-Recipes-YAML")

		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			t.Fatalf("Failed to create script dir: %v", err)
		}
		if err := os.MkdirAll(extractedDir, 0755); err != nil {
			t.Fatalf("Failed to create extracted dir: %v", err)
		}

		result, err := FindYAMLDirectory(scriptDir, "")
		if err != nil {
			t.Fatalf("FindYAMLDirectory() error = %v", err)
		}

		if result != extractedDir {
			t.Errorf("FindYAMLDirectory() = %q, want %q", result, extractedDir)
		}
	})

	t.Run("finds CookBook directory in parent", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create structure: tmpDir/parent/recipe-site (scriptDir)
		//                   tmpDir/parent/CookBook-Recipes-YAML/
		parent := filepath.Join(tmpDir, "parent")
		scriptDir := filepath.Join(parent, "recipe-site")
		cookbookDir := filepath.Join(parent, "CookBook-Recipes-YAML")

		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			t.Fatalf("Failed to create script dir: %v", err)
		}
		if err := os.MkdirAll(cookbookDir, 0755); err != nil {
			t.Fatalf("Failed to create cookbook dir: %v", err)
		}

		result, err := FindYAMLDirectory(scriptDir, "")
		if err != nil {
			t.Fatalf("FindYAMLDirectory() error = %v", err)
		}

		if result != cookbookDir {
			t.Errorf("FindYAMLDirectory() = %q, want %q", result, cookbookDir)
		}
	})

	t.Run("falls back to data/recipes directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		parent := filepath.Join(tmpDir, "parent")
		scriptDir := filepath.Join(parent, "recipe-site")
		dataDir := filepath.Join(parent, "data", "recipes")

		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			t.Fatalf("Failed to create script dir: %v", err)
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("Failed to create data dir: %v", err)
		}

		result, err := FindYAMLDirectory(scriptDir, "")
		if err != nil {
			t.Fatalf("FindYAMLDirectory() error = %v", err)
		}

		if result != dataDir {
			t.Errorf("FindYAMLDirectory() = %q, want %q", result, dataDir)
		}
	})

	t.Run("returns error when no directory found", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create only the script dir, no YAML directories
		parent := filepath.Join(tmpDir, "parent")
		scriptDir := filepath.Join(parent, "recipe-site")
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			t.Fatalf("Failed to create script dir: %v", err)
		}

		_, err := FindYAMLDirectory(scriptDir, "")
		if err == nil {
			t.Error("FindYAMLDirectory() expected error when no directory found")
		}
	})

	t.Run("imported directory takes priority over parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		parent := filepath.Join(tmpDir, "parent")
		scriptDir := filepath.Join(parent, "recipe-site")
		extractedDir := filepath.Join(parent, "imports", ".extracted", "CookBook-Recipes-YAML")
		parentDir := filepath.Join(parent, "CookBook-Recipes-YAML")

		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			t.Fatalf("Failed to create script dir: %v", err)
		}
		if err := os.MkdirAll(extractedDir, 0755); err != nil {
			t.Fatalf("Failed to create extracted dir: %v", err)
		}
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("Failed to create parent dir: %v", err)
		}

		result, err := FindYAMLDirectory(scriptDir, "")
		if err != nil {
			t.Fatalf("FindYAMLDirectory() error = %v", err)
		}

		if result != extractedDir {
			t.Errorf("FindYAMLDirectory() = %q, want %q (imported should take priority)", result, extractedDir)
		}
	})
}
