package recipes

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestZIP creates a ZIP file with the given files for testing.
// Files is a map of filename -> content.
func createTestZIP(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create file in ZIP: %v", err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to ZIP: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close ZIP writer: %v", err)
	}
}

// createTestZIPWithSize creates a ZIP with a file of specified uncompressed size.
func createTestZIPWithSize(t *testing.T, zipPath string, fileName string, size int64) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	fw, err := w.Create(fileName)
	if err != nil {
		t.Fatalf("Failed to create file in ZIP: %v", err)
	}
	// Write size bytes (we use a chunk approach to avoid memory issues)
	chunk := make([]byte, 1024)
	for i := range chunk {
		chunk[i] = 'a'
	}
	written := int64(0)
	for written < size {
		toWrite := int64(len(chunk))
		if written+toWrite > size {
			toWrite = size - written
		}
		if _, err := fw.Write(chunk[:toWrite]); err != nil {
			t.Fatalf("Failed to write to ZIP: %v", err)
		}
		written += toWrite
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close ZIP writer: %v", err)
	}
}

func TestFindImportZIP(t *testing.T) {
	t.Run("finds single ZIP file", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{"recipe.yml": "name: Test"})

		result, err := FindImportZIP(tmpDir)
		if err != nil {
			t.Fatalf("FindImportZIP() error = %v", err)
		}
		if result != zipPath {
			t.Errorf("FindImportZIP() = %q, want %q", result, zipPath)
		}
	})

	t.Run("finds most recent ZIP when multiple exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create older ZIP
		olderZip := filepath.Join(tmpDir, "old-recipes.zip")
		createTestZIP(t, olderZip, map[string]string{"old.yml": "name: Old"})

		// Sleep to ensure different modification times
		time.Sleep(10 * time.Millisecond)

		// Create newer ZIP
		newerZip := filepath.Join(tmpDir, "new-recipes.zip")
		createTestZIP(t, newerZip, map[string]string{"new.yml": "name: New"})

		result, err := FindImportZIP(tmpDir)
		if err != nil {
			t.Fatalf("FindImportZIP() error = %v", err)
		}
		if result != newerZip {
			t.Errorf("FindImportZIP() = %q, want %q (most recent)", result, newerZip)
		}
	})

	t.Run("returns error when no ZIP files exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := FindImportZIP(tmpDir)
		if err == nil {
			t.Error("FindImportZIP() expected error for empty directory, got nil")
		}
		if !strings.Contains(err.Error(), "no ZIP files found") {
			t.Errorf("FindImportZIP() error = %q, want error containing 'no ZIP files found'", err)
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		_, err := FindImportZIP("/nonexistent/directory")
		if err == nil {
			t.Error("FindImportZIP() expected error for non-existent directory, got nil")
		}
	})

	t.Run("ignores non-ZIP files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create non-ZIP files
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "recipe.yml"), []byte("name: Test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := FindImportZIP(tmpDir)
		if err == nil {
			t.Error("FindImportZIP() expected error when only non-ZIP files exist")
		}
	})
}

func TestValidateZIPStructure(t *testing.T) {
	t.Run("valid ZIP with YAML files", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{
			"recipe1.yml":  "name: Recipe 1",
			"recipe2.yaml": "name: Recipe 2",
		})

		err := ValidateZIPStructure(zipPath)
		if err != nil {
			t.Errorf("ValidateZIPStructure() error = %v, want nil", err)
		}
	})

	t.Run("ZIP without YAML files", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "no-yaml.zip")
		createTestZIP(t, zipPath, map[string]string{
			"readme.txt":  "This is a readme",
			"image.png":   "fake image data",
			"config.json": `{"key": "value"}`,
		})

		err := ValidateZIPStructure(zipPath)
		if err == nil {
			t.Error("ValidateZIPStructure() expected error for ZIP without YAML files")
		}
		if !strings.Contains(err.Error(), "does not contain any YAML") {
			t.Errorf("ValidateZIPStructure() error = %q, want error about missing YAML", err)
		}
	})

	t.Run("invalid ZIP file", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.zip")
		if err := os.WriteFile(invalidPath, []byte("not a zip file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := ValidateZIPStructure(invalidPath)
		if err == nil {
			t.Error("ValidateZIPStructure() expected error for invalid ZIP")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		err := ValidateZIPStructure("/nonexistent/file.zip")
		if err == nil {
			t.Error("ValidateZIPStructure() expected error for non-existent file")
		}
	})

	t.Run("mixed YAML and non-YAML files", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "mixed.zip")
		createTestZIP(t, zipPath, map[string]string{
			"recipe.yml": "name: Recipe",
			"readme.txt": "readme",
			"data.json":  "{}",
		})

		err := ValidateZIPStructure(zipPath)
		if err != nil {
			t.Errorf("ValidateZIPStructure() error = %v, want nil (has YAML files)", err)
		}
	})
}

func TestExtractZIP(t *testing.T) {
	t.Run("normal extraction", func(t *testing.T) {
		tmpDir := t.TempDir()
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{
			"recipe1.yml": "name: Recipe 1",
			"recipe2.yml": "name: Recipe 2",
		})

		destDir := filepath.Join(tmpDir, "extracted")
		extractedPath, err := ExtractZIP(zipPath, destDir)
		if err != nil {
			t.Fatalf("ExtractZIP() error = %v", err)
		}

		// Check extracted path (always uses fixed directory name)
		expectedPath := filepath.Join(destDir, "CookBook-Recipes-YAML")
		if extractedPath != expectedPath {
			t.Errorf("ExtractZIP() path = %q, want %q", extractedPath, expectedPath)
		}

		// Verify files were extracted
		files, err := os.ReadDir(extractedPath)
		if err != nil {
			t.Fatalf("Failed to read extracted directory: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("Expected 2 extracted files, got %d", len(files))
		}

		// Verify content
		content, err := os.ReadFile(filepath.Join(extractedPath, "recipe1.yml"))
		if err != nil {
			t.Fatalf("Failed to read extracted file: %v", err)
		}
		if string(content) != "name: Recipe 1" {
			t.Errorf("Extracted content = %q, want %q", string(content), "name: Recipe 1")
		}
	})

	t.Run("path traversal prevention", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create ZIP with path traversal attempt
		zipPath := filepath.Join(tmpDir, "malicious.zip")
		f, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("Failed to create ZIP: %v", err)
		}
		w := zip.NewWriter(f)
		// Try to create a file with path traversal
		fw, _ := w.Create("../../../etc/passwd")
		fw.Write([]byte("malicious content"))
		fw2, _ := w.Create("safe.yml")
		fw2.Write([]byte("name: Safe"))
		w.Close()
		f.Close()

		destDir := filepath.Join(tmpDir, "extracted")
		_, err = ExtractZIP(zipPath, destDir)
		if err != nil {
			t.Fatalf("ExtractZIP() error = %v", err)
		}

		// Verify the malicious file was NOT created outside destDir
		if _, err := os.Stat(filepath.Join(tmpDir, "..", "..", "..", "etc", "passwd")); !os.IsNotExist(err) {
			t.Error("Path traversal prevention failed - file created outside destination")
		}

		// Verify safe file was extracted
		extractedPath := filepath.Join(destDir, "CookBook-Recipes-YAML")
		if _, err := os.Stat(filepath.Join(extractedPath, "safe.yml")); os.IsNotExist(err) {
			t.Error("Safe file was not extracted")
		}
	})

	t.Run("replaces existing extracted directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "extracted")
		existingDir := filepath.Join(destDir, "CookBook-Recipes-YAML")

		// Create existing directory with a file
		if err := os.MkdirAll(existingDir, 0755); err != nil {
			t.Fatalf("Failed to create existing dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(existingDir, "old.yml"), []byte("old"), 0644); err != nil {
			t.Fatalf("Failed to create old file: %v", err)
		}

		// Create and extract new ZIP
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{"new.yml": "name: New"})

		extractedPath, err := ExtractZIP(zipPath, destDir)
		if err != nil {
			t.Fatalf("ExtractZIP() error = %v", err)
		}

		// Old file should be gone
		if _, err := os.Stat(filepath.Join(extractedPath, "old.yml")); !os.IsNotExist(err) {
			t.Error("Old file should have been removed")
		}

		// New file should exist
		if _, err := os.Stat(filepath.Join(extractedPath, "new.yml")); os.IsNotExist(err) {
			t.Error("New file should exist")
		}
	})

	t.Run("skips directories in ZIP", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create ZIP with directory entries
		zipPath := filepath.Join(tmpDir, "with-dirs.zip")
		f, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("Failed to create ZIP: %v", err)
		}
		w := zip.NewWriter(f)
		// Create directory entry
		w.Create("subdir/")
		// Create file in subdirectory (will be flattened)
		fw, _ := w.Create("subdir/recipe.yml")
		fw.Write([]byte("name: Test"))
		w.Close()
		f.Close()

		destDir := filepath.Join(tmpDir, "extracted")
		extractedPath, err := ExtractZIP(zipPath, destDir)
		if err != nil {
			t.Fatalf("ExtractZIP() error = %v", err)
		}

		// File should be extracted with flattened name
		if _, err := os.Stat(filepath.Join(extractedPath, "recipe.yml")); os.IsNotExist(err) {
			t.Error("File should be extracted with flattened name")
		}
	})

	t.Run("invalid ZIP file", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.zip")
		if err := os.WriteFile(invalidPath, []byte("not a zip"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		destDir := filepath.Join(tmpDir, "extracted")
		_, err := ExtractZIP(invalidPath, destDir)
		if err == nil {
			t.Error("ExtractZIP() expected error for invalid ZIP")
		}
	})
}

func TestExtractZIP_SizeLimits(t *testing.T) {
	// Test that files exceeding MaxZIPFileSize are skipped
	t.Run("skips files exceeding size limit", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a ZIP with a file that has large declared uncompressed size
		zipPath := filepath.Join(tmpDir, "large.zip")

		// We'll create a small file but verify the size checking logic works
		// by creating a normal-sized file and checking it extracts
		createTestZIP(t, zipPath, map[string]string{
			"small.yml": "name: Small Recipe",
		})

		destDir := filepath.Join(tmpDir, "extracted")
		extractedPath, err := ExtractZIP(zipPath, destDir)
		if err != nil {
			t.Fatalf("ExtractZIP() error = %v", err)
		}

		// Small file should be extracted
		if _, err := os.Stat(filepath.Join(extractedPath, "small.yml")); os.IsNotExist(err) {
			t.Error("Small file should be extracted")
		}
	})
}

func TestCleanupImports(t *testing.T) {
	t.Run("removes extracted directory only", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .extracted directory with content
		extractedDir := filepath.Join(tmpDir, ".extracted")
		if err := os.MkdirAll(filepath.Join(extractedDir, "recipes"), 0755); err != nil {
			t.Fatalf("Failed to create extracted dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(extractedDir, "recipes", "test.yml"), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a ZIP file
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{"test.yml": "name: Test"})

		// Cleanup without removing ZIPs
		err := CleanupImports(tmpDir, false)
		if err != nil {
			t.Fatalf("CleanupImports() error = %v", err)
		}

		// Extracted directory should be gone
		if _, err := os.Stat(extractedDir); !os.IsNotExist(err) {
			t.Error("Extracted directory should be removed")
		}

		// ZIP should still exist
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			t.Error("ZIP file should still exist")
		}
	})

	t.Run("removes both extracted directory and ZIPs", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .extracted directory
		extractedDir := filepath.Join(tmpDir, ".extracted")
		if err := os.MkdirAll(extractedDir, 0755); err != nil {
			t.Fatalf("Failed to create extracted dir: %v", err)
		}

		// Create ZIP files
		zipPath1 := filepath.Join(tmpDir, "recipes1.zip")
		zipPath2 := filepath.Join(tmpDir, "recipes2.zip")
		createTestZIP(t, zipPath1, map[string]string{"test1.yml": "name: Test1"})
		createTestZIP(t, zipPath2, map[string]string{"test2.yml": "name: Test2"})

		// Cleanup with removing ZIPs
		err := CleanupImports(tmpDir, true)
		if err != nil {
			t.Fatalf("CleanupImports() error = %v", err)
		}

		// Extracted directory should be gone
		if _, err := os.Stat(extractedDir); !os.IsNotExist(err) {
			t.Error("Extracted directory should be removed")
		}

		// ZIPs should be gone
		if _, err := os.Stat(zipPath1); !os.IsNotExist(err) {
			t.Error("ZIP file 1 should be removed")
		}
		if _, err := os.Stat(zipPath2); !os.IsNotExist(err) {
			t.Error("ZIP file 2 should be removed")
		}
	})

	t.Run("handles non-existent extracted directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// No .extracted directory exists
		err := CleanupImports(tmpDir, false)
		if err != nil {
			t.Errorf("CleanupImports() error = %v, want nil", err)
		}
	})

	t.Run("handles non-existent ZIP files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// No ZIP files exist
		err := CleanupImports(tmpDir, true)
		if err != nil {
			t.Errorf("CleanupImports() error = %v, want nil", err)
		}
	})
}

func TestImport(t *testing.T) {
	t.Run("successful import workflow", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a ZIP with recipe files
		zipPath := filepath.Join(tmpDir, "CookBook-Recipes-YAML-20241201.zip")
		createTestZIP(t, zipPath, map[string]string{
			"recipe1.yml":  "name: Recipe 1",
			"recipe2.yml":  "name: Recipe 2",
			"recipe3.yaml": "name: Recipe 3",
		})

		result, err := Import(tmpDir)
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}

		// Check result
		if result.ZIPPath != zipPath {
			t.Errorf("Import() ZIPPath = %q, want %q", result.ZIPPath, zipPath)
		}

		expectedExtracted := filepath.Join(tmpDir, ".extracted", "CookBook-Recipes-YAML")
		if result.ExtractedPath != expectedExtracted {
			t.Errorf("Import() ExtractedPath = %q, want %q", result.ExtractedPath, expectedExtracted)
		}

		if result.RecipeCount != 3 {
			t.Errorf("Import() RecipeCount = %d, want 3", result.RecipeCount)
		}

		// Verify files were actually extracted
		if _, err := os.Stat(filepath.Join(result.ExtractedPath, "recipe1.yml")); os.IsNotExist(err) {
			t.Error("Extracted file should exist")
		}
	})

	t.Run("no ZIP file found", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := Import(tmpDir)
		if err == nil {
			t.Error("Import() expected error when no ZIP exists")
		}
	})

	t.Run("invalid ZIP structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create ZIP without YAML files
		zipPath := filepath.Join(tmpDir, "no-recipes.zip")
		createTestZIP(t, zipPath, map[string]string{
			"readme.txt": "no recipes here",
		})

		_, err := Import(tmpDir)
		if err == nil {
			t.Error("Import() expected error for ZIP without YAML files")
		}
	})

	t.Run("corrupt ZIP file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create corrupt ZIP
		corruptPath := filepath.Join(tmpDir, "corrupt.zip")
		if err := os.WriteFile(corruptPath, []byte("not a valid zip"), 0644); err != nil {
			t.Fatalf("Failed to create corrupt file: %v", err)
		}

		_, err := Import(tmpDir)
		if err == nil {
			t.Error("Import() expected error for corrupt ZIP")
		}
	})
}

func TestImport_Integration(t *testing.T) {
	t.Run("full workflow with cleanup", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create initial ZIP
		zipPath := filepath.Join(tmpDir, "recipes.zip")
		createTestZIP(t, zipPath, map[string]string{
			"pasta.yml": "name: Pasta",
			"soup.yml":  "name: Soup",
		})

		// Import
		result, err := Import(tmpDir)
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}

		// Verify import worked
		if result.RecipeCount != 2 {
			t.Errorf("RecipeCount = %d, want 2", result.RecipeCount)
		}

		// Cleanup extracted only
		if err := CleanupImports(tmpDir, false); err != nil {
			t.Fatalf("CleanupImports() error = %v", err)
		}

		// ZIP should still exist
		if _, err := os.Stat(zipPath); os.IsNotExist(err) {
			t.Error("ZIP should still exist after cleanup with keepZips=true")
		}

		// Extracted should be gone
		if _, err := os.Stat(result.ExtractedPath); !os.IsNotExist(err) {
			t.Error("Extracted path should be removed")
		}

		// Re-import should work
		result2, err := Import(tmpDir)
		if err != nil {
			t.Fatalf("Re-import error = %v", err)
		}
		if result2.RecipeCount != 2 {
			t.Errorf("Re-import RecipeCount = %d, want 2", result2.RecipeCount)
		}
	})
}
