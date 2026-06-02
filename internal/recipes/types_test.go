package recipes

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStringOrSlice_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StringOrSlice
		wantErr bool
	}{
		{
			name:    "single string",
			input:   `"hello"`,
			want:    StringOrSlice{"hello"},
			wantErr: false,
		},
		{
			name:    "single string with whitespace",
			input:   `"  hello  "`,
			want:    StringOrSlice{"hello"},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   `""`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "whitespace only string",
			input:   `"   "`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "array of strings",
			input:   `["foo", "bar", "baz"]`,
			want:    StringOrSlice{"foo", "bar", "baz"},
			wantErr: false,
		},
		{
			name:    "array with empty strings filtered",
			input:   `["foo", "", "bar", "  ", "baz"]`,
			want:    StringOrSlice{"foo", "bar", "baz"},
			wantErr: false,
		},
		{
			name:    "empty array",
			input:   `[]`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "array with only empty strings",
			input:   `["", "  ", ""]`,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "null value",
			input:   `null`,
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got StringOrSlice
			err := yaml.Unmarshal([]byte(tt.input), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("StringOrSlice.UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringOrSlice.UnmarshalYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringOrSlice_InRecipeContext(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StringOrSlice
		wantErr bool
	}{
		{
			name: "tags as single string",
			input: `
name: Test Recipe
tags: Dinner
`,
			want:    StringOrSlice{"Dinner"},
			wantErr: false,
		},
		{
			name: "tags as array",
			input: `
name: Test Recipe
tags:
  - Dinner
  - Vegetarian
  - Quick
`,
			want:    StringOrSlice{"Dinner", "Vegetarian", "Quick"},
			wantErr: false,
		},
		{
			name: "no tags",
			input: `
name: Test Recipe
`,
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recipe Recipe
			err := yaml.Unmarshal([]byte(tt.input), &recipe)

			if (err != nil) != tt.wantErr {
				t.Errorf("Recipe unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(recipe.Tags, tt.want) {
				t.Errorf("Recipe.Tags = %v, want %v", recipe.Tags, tt.want)
			}
		})
	}
}
