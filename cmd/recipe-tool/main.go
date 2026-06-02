package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bclews/cookbook-site/internal/recipes"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		validateCmd(os.Args[2:])
	case "convert":
		convertCmd(os.Args[2:])
	case "import":
		importCmd(os.Args[2:])
	case "cleanup":
		cleanupCmd(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// siteRootDir returns the Hugo site root: the nearest ancestor of the current
// working directory that contains hugo.toml. This makes the tool work whether
// it is run via "go run" or as a built binary, and from any directory inside
// the site. It falls back to the executable's directory (handy when the binary
// sits in the site root) and finally to the working directory.
func siteRootDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determining working directory: %w", err)
	}
	if dir, ok := findHugoRoot(cwd); ok {
		return dir, nil
	}
	if exe, err := os.Executable(); err == nil {
		if dir, ok := findHugoRoot(filepath.Dir(exe)); ok {
			return dir, nil
		}
	}
	return cwd, nil
}

// findHugoRoot walks up from start looking for a directory containing hugo.toml.
func findHugoRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "hugo.toml")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func printUsage() {
	fmt.Println("Usage: recipe-tool <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  validate    Validate YAML recipe files")
	fmt.Println("  convert     Convert YAML recipes to Hugo markdown")
	fmt.Println("  import      Extract ZIP file from imports directory")
	fmt.Println("  cleanup     Remove extracted files and ZIPs from imports")
	fmt.Println("  help        Show this help message")
	fmt.Println()
	fmt.Println("Run 'recipe-tool <command> -h' for command-specific options")
}

// exitOnError prints an error message and exits with code 1.
func exitOnError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// ============================================================================
// Validate Command
// ============================================================================

// validateOptions holds parsed flags for the validate command.
type validateOptions struct {
	yamlDir string
	strict  bool
	verbose bool
}

// parseValidateFlags parses command-line flags for the validate command.
func parseValidateFlags(args []string) validateOptions {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	yamlDir := fs.String("yaml-dir", "", "Path to YAML recipes directory (auto-discovered if not specified)")
	strict := fs.Bool("strict", false, "Treat warnings as errors")
	verbose := fs.Bool("verbose", false, "Show all warnings, not just errors")
	_ = fs.Parse(args)

	return validateOptions{
		yamlDir: *yamlDir,
		strict:  *strict,
		verbose: *verbose,
	}
}

// printValidationResults prints individual file errors and warnings.
func printValidationResults(results []recipes.ValidationResult, verbose bool) {
	for _, result := range results {
		if len(result.Errors) > 0 {
			fmt.Printf("Error: %s\n", result.File)
			for _, e := range result.Errors {
				fmt.Printf("   ERROR: %s\n", e)
			}
		} else if verbose && len(result.Warnings) > 0 {
			fmt.Printf("Warning: %s\n", result.File)
			for _, w := range result.Warnings {
				fmt.Printf("   WARNING: %s\n", w)
			}
		}
	}
}

// printValidationSummary prints the validation summary.
func printValidationSummary(summary recipes.ValidationSummary) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("Validation Summary")
	fmt.Println("============================================================")
	fmt.Printf("Total files:          %d\n", summary.TotalFiles)
	fmt.Printf("Files with errors:    %d\n", summary.FilesWithErrors)
	fmt.Printf("Files with warnings:  %d\n", summary.FilesWithWarnings)
	fmt.Printf("Total errors:         %d\n", summary.TotalErrors)
	fmt.Printf("Total warnings:       %d\n", summary.TotalWarnings)
	fmt.Println("============================================================")
	fmt.Println()
}

// handleValidationExit prints the final status message and exits if needed.
func handleValidationExit(summary recipes.ValidationSummary, opts validateOptions) {
	if summary.TotalErrors > 0 {
		fmt.Println("Validation failed with errors")
		os.Exit(1)
	} else if opts.strict && summary.TotalWarnings > 0 {
		fmt.Println("Validation failed (strict mode: warnings treated as errors)")
		os.Exit(1)
	} else if summary.TotalWarnings > 0 {
		fmt.Println("Validation passed with warnings")
		if !opts.verbose {
			fmt.Println("   (use --verbose to see all warnings)")
		}
	} else {
		fmt.Println("All recipes validated successfully!")
	}
}

func validateCmd(args []string) {
	opts := parseValidateFlags(args)

	siteRoot, err := siteRootDir()
	if err != nil {
		exitOnError("%v", err)
	}

	dir, err := recipes.FindYAMLDirectory(siteRoot, opts.yamlDir)
	if err != nil {
		exitOnError("%v", err)
	}

	fmt.Printf("Validating recipe files from %s\n\n", filepath.Base(dir))

	results, summary, err := recipes.ValidateDirectoryWithError(dir)
	if err != nil {
		exitOnError("scanning directory: %v", err)
	}

	printValidationResults(results, opts.verbose)
	printValidationSummary(summary)
	handleValidationExit(summary, opts)
}

// ============================================================================
// Convert Command
// ============================================================================

// convertOptions holds parsed flags for the convert command.
type convertOptions struct {
	yamlDir    string
	parallel   int
	skipImages bool
}

// parseConvertFlags parses command-line flags for the convert command.
func parseConvertFlags(args []string) convertOptions {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	yamlDir := fs.String("yaml-dir", "", "Path to YAML recipes directory (auto-discovered if not specified)")
	parallel := fs.Int("parallel", 10, "Number of parallel image downloads")
	skipImages := fs.Bool("skip-images", false, "Skip image downloading")
	_ = fs.Parse(args)

	return convertOptions{
		yamlDir:    *yamlDir,
		parallel:   *parallel,
		skipImages: *skipImages,
	}
}

// convertDirs holds the directories used during conversion.
type convertDirs struct {
	source string
	output string
	images string
}

// setupConvertDirs discovers and creates necessary directories for conversion.
func setupConvertDirs(opts convertOptions) (convertDirs, error) {
	siteRoot, err := siteRootDir()
	if err != nil {
		return convertDirs{}, err
	}

	sourceDir, err := recipes.FindYAMLDirectory(siteRoot, opts.yamlDir)
	if err != nil {
		return convertDirs{}, err
	}

	dirs := convertDirs{
		source: sourceDir,
		output: filepath.Join(siteRoot, "content", "recipes"),
		images: filepath.Join(siteRoot, "static", "images", "recipes"),
	}

	if err := os.MkdirAll(dirs.output, 0755); err != nil {
		return convertDirs{}, fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.MkdirAll(dirs.images, 0755); err != nil {
		return convertDirs{}, fmt.Errorf("creating images directory: %w", err)
	}

	return dirs, nil
}

// printConvertHeader prints the conversion header with directory info.
func printConvertHeader(dirs convertDirs) {
	fmt.Println()
	fmt.Println("Converting recipes from CookBook Manager to Hugo format")
	fmt.Printf("   Source: %s\n", dirs.source)
	fmt.Printf("   Output: %s\n", dirs.output)
	fmt.Printf("   Images: %s\n", dirs.images)
	fmt.Println()
}

// downloadImages downloads recipe images and returns the URL-to-path mapping.
func downloadImages(recipeFiles []recipes.RecipeFile, imagesDir string, parallel int) map[string]string {
	downloader := recipes.NewImageDownloader(imagesDir, parallel)
	imageMap := downloader.DownloadImages(recipeFiles)
	stats := downloader.GetStats()

	fmt.Println("Image download complete:")
	fmt.Printf("   Downloaded: %d\n", stats.Downloaded)
	fmt.Printf("   Skipped (cached): %d\n", stats.Skipped)
	fmt.Printf("   Failed: %d\n", stats.Failed)
	fmt.Println()

	return imageMap
}

// printConvertSummary prints the conversion summary.
func printConvertSummary(successCount, totalCount, imageCount int, outputDir string) {
	fmt.Println()
	fmt.Println("Conversion complete!")
	fmt.Printf("   Successfully converted: %d/%d recipes\n", successCount, totalCount)
	fmt.Printf("   Total images: %d\n", imageCount)
	fmt.Printf("   Output directory: %s\n", outputDir)
}

func convertCmd(args []string) {
	opts := parseConvertFlags(args)

	dirs, err := setupConvertDirs(opts)
	if err != nil {
		exitOnError("%v", err)
	}

	printConvertHeader(dirs)

	// Load recipes
	fmt.Println("Loading recipe files...")
	recipeFiles, err := recipes.LoadRecipes(dirs.source)
	if err != nil {
		exitOnError("loading recipes: %v", err)
	}
	fmt.Printf("   Found %d valid recipe files\n\n", len(recipeFiles))

	if len(recipeFiles) == 0 {
		exitOnError("no recipes found to convert")
	}

	// Download images (unless skipped)
	imageMap := make(map[string]string)
	if !opts.skipImages {
		imageMap = downloadImages(recipeFiles, dirs.images, opts.parallel)
	}

	// Convert recipes
	fmt.Printf("Converting %d recipes to markdown...\n", len(recipeFiles))
	successCount, failedCount := recipes.ConvertAll(recipeFiles, dirs.output, imageMap)

	// Count images
	imageCount := 0
	if entries, err := os.ReadDir(dirs.images); err == nil {
		imageCount = len(entries)
	}

	printConvertSummary(successCount, len(recipeFiles), imageCount, dirs.output)

	if failedCount > 0 {
		os.Exit(1)
	}
}

// ============================================================================
// Import Command
// ============================================================================

// importOptions holds parsed flags for the import command.
type importOptions struct {
	importsDir string
}

// parseImportFlags parses command-line flags for the import command.
func parseImportFlags(args []string) importOptions {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	importsDir := fs.String("imports-dir", "", "Path to imports directory (default: ../imports)")
	_ = fs.Parse(args)

	return importOptions{
		importsDir: *importsDir,
	}
}

// resolveImportsDir resolves the imports directory path.
func resolveImportsDir(customDir string) (string, error) {
	if customDir != "" {
		return customDir, nil
	}

	siteRoot, err := siteRootDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(siteRoot), "imports"), nil
}

// printImportSuccess prints the import success message.
func printImportSuccess(result *recipes.ImportResult) {
	fmt.Println("Import successful!")
	fmt.Printf("   ZIP file: %s\n", filepath.Base(result.ZIPPath))
	fmt.Printf("   Extracted to: %s\n", result.ExtractedPath)
	fmt.Printf("   Recipe files: %d\n", result.RecipeCount)
	fmt.Println()
	fmt.Println("Run 'recipe-tool convert' to convert the imported recipes")
}

func importCmd(args []string) {
	opts := parseImportFlags(args)

	importPath, err := resolveImportsDir(opts.importsDir)
	if err != nil {
		exitOnError("%v", err)
	}

	fmt.Println()
	fmt.Println("Importing recipe ZIP file")
	fmt.Printf("   Imports directory: %s\n", importPath)
	fmt.Println()

	if _, err := os.Stat(importPath); os.IsNotExist(err) {
		fmt.Printf("Imports directory does not exist: %s\n", importPath)
		fmt.Println("Create the directory and place a ZIP file in it, or use --imports-dir to specify a different location")
		os.Exit(1)
	}

	result, err := recipes.Import(importPath)
	if err != nil {
		exitOnError("import failed: %v", err)
	}

	printImportSuccess(result)
}

// ============================================================================
// Cleanup Command
// ============================================================================

// cleanupOptions holds parsed flags for the cleanup command.
type cleanupOptions struct {
	importsDir string
	keepZips   bool
}

// parseCleanupFlags parses command-line flags for the cleanup command.
func parseCleanupFlags(args []string) cleanupOptions {
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	importsDir := fs.String("imports-dir", "", "Path to imports directory (default: ../imports)")
	keepZips := fs.Bool("keep-zips", false, "Keep ZIP files, only remove extracted content")
	_ = fs.Parse(args)

	return cleanupOptions{
		importsDir: *importsDir,
		keepZips:   *keepZips,
	}
}

// printCleanupHeader prints the cleanup header with mode info.
func printCleanupHeader(importPath string, keepZips bool) {
	fmt.Println()
	fmt.Println("Cleaning imports directory")
	fmt.Printf("   Imports directory: %s\n", importPath)
	if keepZips {
		fmt.Println("   Mode: Keep ZIP files, remove extracted content only")
	} else {
		fmt.Println("   Mode: Remove both ZIP files and extracted content")
	}
	fmt.Println()
}

func cleanupCmd(args []string) {
	opts := parseCleanupFlags(args)

	importPath, err := resolveImportsDir(opts.importsDir)
	if err != nil {
		exitOnError("%v", err)
	}

	printCleanupHeader(importPath, opts.keepZips)

	if _, err := os.Stat(importPath); os.IsNotExist(err) {
		fmt.Println("Imports directory does not exist, nothing to clean")
		return
	}

	if err := recipes.CleanupImports(importPath, !opts.keepZips); err != nil {
		exitOnError("cleanup failed: %v", err)
	}

	fmt.Println("Cleanup complete!")
}
