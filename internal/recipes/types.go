package recipes

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrSlice handles YAML fields that can be either a string or []string.
type StringOrSlice []string

// UnmarshalYAML implements custom unmarshaling for StringOrSlice.
func (s *StringOrSlice) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string value
		val := strings.TrimSpace(node.Value)
		if val != "" {
			*s = []string{val}
		}
		return nil
	case yaml.SequenceNode:
		// Array of strings
		var arr []string
		if err := node.Decode(&arr); err != nil {
			return err
		}
		// Filter empty strings
		var result []string
		for _, v := range arr {
			if strings.TrimSpace(v) != "" {
				result = append(result, v)
			}
		}
		*s = result
		return nil
	default:
		*s = nil
		return nil
	}
}

// Recipe represents a recipe from CookBook Manager YAML format.
type Recipe struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Servings    string        `yaml:"servings,omitempty"`
	PrepTime    string        `yaml:"prep_time,omitempty"`
	CookTime    string        `yaml:"cook_time,omitempty"`
	Source      string        `yaml:"source,omitempty"`
	Image       string        `yaml:"image,omitempty"`
	Tags        StringOrSlice `yaml:"tags,omitempty"`
	Keywords    StringOrSlice `yaml:"keywords,omitempty"`
	Ingredients []string      `yaml:"ingredients,omitempty"`
	Directions  []string      `yaml:"directions,omitempty"`
	Nutrition   string        `yaml:"nutrition,omitempty"`
	Notes       string        `yaml:"notes,omitempty"`
	Favorite    bool          `yaml:"favorite,omitempty"`
	OnFavorites string        `yaml:"on_favorites,omitempty"`
	CookCount   int           `yaml:"cook_count,omitempty"`
	Images      []string      `yaml:"images,omitempty"`
	ExportedBy  string        `yaml:"exportedBy,omitempty"`
}

// RecipeFile holds a recipe and its source file path.
type RecipeFile struct {
	Path   string
	Recipe *Recipe
}

// ValidationResult holds the result of validating a single file.
type ValidationResult struct {
	File     string
	Errors   []string
	Warnings []string
}

// ValidationSummary holds aggregate validation statistics.
type ValidationSummary struct {
	TotalFiles        int
	FilesWithErrors   int
	FilesWithWarnings int
	TotalErrors       int
	TotalWarnings     int
}

// DownloadStats tracks image download statistics.
type DownloadStats struct {
	Downloaded int
	Skipped    int
	Failed     int
}
