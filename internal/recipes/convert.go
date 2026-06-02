package recipes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	invalidCharsRegex = regexp.MustCompile(`[^\w\s-]`)
	spacesRegex       = regexp.MustCompile(`[-\s]+`)
)

// SanitizeFilename converts a recipe name to a valid filename.
func SanitizeFilename(name string) string {
	// Remove invalid characters
	name = invalidCharsRegex.ReplaceAllString(name, "")
	// Replace spaces with hyphens
	name = spacesRegex.ReplaceAllString(name, "-")
	// Convert to lowercase and trim
	return strings.ToLower(strings.Trim(name, "-"))
}

// LoadError represents an error that occurred while loading a recipe file.
type LoadError struct {
	File string
	Err  error
}

// LoadResult contains successfully loaded recipes and any errors encountered.
type LoadResult struct {
	Recipes  []RecipeFile
	Errors   []LoadError
	DirError error // Non-nil if the directory itself could not be scanned
}

// LoadRecipes loads all YAML recipe files from a directory.
// Returns successfully loaded recipes. Individual file errors are logged as warnings.
// Returns an error only if the directory cannot be scanned.
func LoadRecipes(dir string) ([]RecipeFile, error) {
	result := LoadRecipesWithErrors(dir)

	if result.DirError != nil {
		return nil, result.DirError
	}

	// Log individual file errors as warnings
	for _, loadErr := range result.Errors {
		Logger.Warn("Error loading recipe file", "file", loadErr.File, "error", loadErr.Err)
	}

	return result.Recipes, nil
}

// LoadRecipesWithErrors loads all YAML recipe files and returns detailed error information.
func LoadRecipesWithErrors(dir string) LoadResult {
	result := LoadResult{}

	pattern := filepath.Join(dir, "*.yml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		result.DirError = fmt.Errorf("failed to scan directory %s: %w", dir, err)
		return result
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			result.Errors = append(result.Errors, LoadError{
				File: filepath.Base(file),
				Err:  fmt.Errorf("failed to read file: %w", err),
			})
			continue
		}

		var recipe Recipe
		if err := yaml.Unmarshal(data, &recipe); err != nil {
			result.Errors = append(result.Errors, LoadError{
				File: filepath.Base(file),
				Err:  fmt.Errorf("YAML parse error: %w", err),
			})
			continue
		}

		if recipe.Name == "" {
			result.Errors = append(result.Errors, LoadError{
				File: filepath.Base(file),
				Err:  fmt.Errorf("missing required field: name"),
			})
			continue
		}

		result.Recipes = append(result.Recipes, RecipeFile{
			Path:   file,
			Recipe: &recipe,
		})
	}

	return result
}

// FrontMatter represents Hugo front matter for a recipe.
type FrontMatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Draft       bool     `yaml:"draft"`
	Description string   `yaml:"description,omitempty"`
	Servings    string   `yaml:"servings,omitempty"`
	PrepTime    string   `yaml:"prep_time,omitempty"`
	CookTime    string   `yaml:"cook_time,omitempty"`
	Source      string   `yaml:"source,omitempty"`
	Image       string   `yaml:"image,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
	Ingredients []string `yaml:"ingredients,omitempty"`
	Directions  []string `yaml:"directions,omitempty"`
	Nutrition   string   `yaml:"nutrition,omitempty"`
	Notes       string   `yaml:"notes,omitempty"`
	Favorite    bool     `yaml:"favorite,omitempty"`
	CookCount   int      `yaml:"cook_count,omitempty"`
}

// ConvertRecipe converts a single recipe to Hugo markdown format.
//
// It creates a markdown file with YAML front matter in outputDir,
// using the recipe name as the filename (sanitized via SanitizeFilename).
//
// The imageMap parameter maps original image URLs to local paths.
// If a recipe's image URL is found in imageMap, the local path is used
// in the output; otherwise, the original URL is preserved. Pass nil
// if no image mapping is needed.
//
// The source file's modification date is used as the recipe date.
// Empty directions are automatically filtered out.
//
// Returns an error if front matter marshaling or file writing fails.
func ConvertRecipe(rf RecipeFile, outputDir string, imageMap map[string]string) error {
	recipe := rf.Recipe

	// Get date from source file modification time, fallback to current time
	date := time.Now().Format("2006-01-02")
	if info, err := os.Stat(rf.Path); err == nil {
		date = info.ModTime().Format("2006-01-02")
	}

	// Build front matter — omitempty tags handle zero-value suppression
	fm := FrontMatter{
		Title:       recipe.Name,
		Date:        date,
		Draft:       false,
		Description: recipe.Description,
		Servings:    recipe.Servings,
		PrepTime:    recipe.PrepTime,
		CookTime:    recipe.CookTime,
		Source:      recipe.Source,
		Tags:        []string(recipe.Tags),
		Keywords:    []string(recipe.Keywords),
		Ingredients: recipe.Ingredients,
		Nutrition:   recipe.Nutrition,
		Notes:       recipe.Notes,
		Favorite:    recipe.Favorite,
		CookCount:   recipe.CookCount,
	}

	// Handle image path - prefer local cached version if available
	if recipe.Image != "" {
		if localPath, ok := imageMap[recipe.Image]; ok {
			fm.Image = localPath
		} else {
			fm.Image = recipe.Image
		}
	}

	// Filter out empty directions (common issue from CookBook Manager export)
	for _, d := range recipe.Directions {
		if strings.TrimSpace(d) != "" {
			fm.Directions = append(fm.Directions, d)
		}
	}

	// Generate YAML front matter
	fmData, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("failed to marshal front matter: %w", err)
	}

	// Create output file
	filename := SanitizeFilename(recipe.Name) + ".md"
	outputPath := filepath.Join(outputDir, filename)

	content := fmt.Sprintf("---\n%s---\n\n", string(fmData))

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ConvertAll converts all recipes to Hugo markdown format.
//
// It processes each recipe using ConvertRecipe and tracks success/failure counts.
// Progress is logged every ConversionProgressInterval recipes (default: 50).
// Individual failures are logged but do not stop the overall conversion.
//
// Parameters:
//   - recipes: slice of RecipeFile structs to convert
//   - outputDir: directory where markdown files will be created
//   - imageMap: URL-to-local-path mapping for images (can be nil)
//
// Returns:
//   - successCount: number of successfully converted recipes
//   - failedCount: number of recipes that failed to convert
func ConvertAll(recipes []RecipeFile, outputDir string, imageMap map[string]string) (int, int) {
	successCount := 0
	failedCount := 0

	for i, rf := range recipes {
		if err := ConvertRecipe(rf, outputDir, imageMap); err != nil {
			Logger.Error("Error converting recipe", "name", rf.Recipe.Name, "error", err)
			failedCount++
		} else {
			successCount++
		}

		if (i+1)%ConversionProgressInterval == 0 {
			Logger.Info("Conversion progress", "processed", i+1, "total", len(recipes))
		}
	}

	return successCount, failedCount
}

// FindYAMLDirectory locates the directory of YAML recipe files, given the Hugo
// site root. It checks, in order:
//
//  1. customPath, if given via --yaml-dir
//  2. imports/.extracted/CookBook-Recipes-YAML (from a ZIP import)
//  3. CookBook-Recipes-YAML (your exported recipes)
//  4. data/recipes
//  5. examples/recipes (sample recipes shipped with this repo)
//
// Paths 2–5 are resolved relative to the site root's parent directory.
func FindYAMLDirectory(siteRoot, customPath string) (string, error) {
	if customPath != "" {
		if info, err := os.Stat(customPath); err == nil && info.IsDir() {
			return customPath, nil
		}
		return "", fmt.Errorf("specified YAML directory not found: %s", customPath)
	}

	parent := filepath.Dir(siteRoot)

	candidates := []struct {
		path string
		desc string
	}{
		{filepath.Join(parent, "imports", ".extracted", "CookBook-Recipes-YAML"), "imported"},
		{filepath.Join(parent, "CookBook-Recipes-YAML"), "exported"},
		{filepath.Join(parent, "data", "recipes"), "data"},
		{filepath.Join(parent, "examples", "recipes"), "examples"},
	}

	for _, c := range candidates {
		if info, err := os.Stat(c.path); err == nil && info.IsDir() {
			Logger.Info("Using YAML directory", "directory", c.path, "source", c.desc)
			return c.path, nil
		}
	}

	return "", fmt.Errorf("no YAML recipe directory found under %s", parent)
}
