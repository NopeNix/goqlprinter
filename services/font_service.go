package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"goqlprinter/config"
)

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// findFontFiles recursively searches for font files in directory
func findFontFiles(dir string) ([]string, error) {
	var fontFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			name := info.Name()
			lowerName := strings.ToLower(name)
			if strings.HasSuffix(lowerName, ".ttf") || strings.HasSuffix(lowerName, ".otf") {
				fontFiles = append(fontFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return fontFiles, nil
}


// ListFonts scans the font directories and returns a list of available fonts.
// Paths are now relative to the project root, not the working directory.
func ListFonts() ([]string, error) {

	fontDirs := config.Cfg.App.FontDirs

	slog.Debug("Searching for fonts in configured directories", "dirs", fontDirs)

	var fonts []string
	for _, dir := range fontDirs {
		// Expand ~ to home directory
		expandedDir := expandTilde(dir)
		fontFiles, err := findFontFiles(expandedDir)
		if err != nil {
			if os.IsNotExist(err) {
				slog.Warn("Font directory does not exist", "dir", expandedDir)
				continue
			}
			return nil, fmt.Errorf("error reading directory %s: %w", expandedDir, err)
		}

		for _, fontPath := range fontFiles {
			fontName := strings.TrimSuffix(filepath.Base(fontPath), filepath.Ext(fontPath))
			fonts = append(fonts, fontName)
			slog.Debug("Found font", "name", fontName, "path", fontPath)
		}
	}

	if len(fonts) == 0 {
		slog.Warn("No fonts found in any of the searched directories")
	} else {
		slog.Info("Found fonts total", "count", len(fonts))
	}

	return fonts, nil
}

// GetFontPath returns the absolute path to a font file given its family name.
// Searches recursively in configured font directories.
func GetFontPath(fontFamily string) (string, error) {
	fontDirs := config.Cfg.App.FontDirs

	slog.Debug("Searching for font in configured directories", "font", fontFamily, "dirs", fontDirs)

	// Search recursively in each font directory
	for _, dir := range fontDirs {
		// Expand ~ to home directory
		expandedDir := expandTilde(dir)
		var foundPath string
		err := filepath.Walk(expandedDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue searching
			}
			if info.IsDir() {
				return nil
			}

			// Check if filename matches (without extension)
			baseName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			if baseName == fontFamily {
				ext := strings.ToLower(filepath.Ext(info.Name()))
				if ext == ".ttf" || ext == ".otf" {
					foundPath = path
					return filepath.SkipAll // Found it, stop walking
				}
			}
			return nil
		})

		if err == nil && foundPath != "" {
			slog.Debug("Found font", "path", foundPath)
			return foundPath, nil
		}
	}

	slog.Error("font not found in any searched directories", "font", fontFamily)
	return "", fmt.Errorf("font %q not found in any searched directories", fontFamily)
}
