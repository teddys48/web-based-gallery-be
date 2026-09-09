package util

import (
	"path/filepath"
	"strings"
)

// NormalizeMediaPath converts any path into a clean, uniform media path (e.g. "media/Vacation/beach.jpg")
func NormalizeMediaPath(mediaDir, path string) string {
	if path == "" {
		return "media"
	}
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	cleanMediaDir := filepath.ToSlash(filepath.Clean(mediaDir))

	rel, err := filepath.Rel(cleanMediaDir, cleanPath)
	if err == nil && !strings.HasPrefix(rel, "..") {
		cleanPath = rel
	} else {
		cleanPath = strings.TrimPrefix(cleanPath, cleanMediaDir+"/")
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}

	cleanPath = filepath.ToSlash(filepath.Clean(cleanPath))
	if !strings.HasPrefix(cleanPath, "media/") && cleanPath != "media" {
		cleanPath = "media/" + cleanPath
	}
	return cleanPath
}

// ResolvePhysicalPath maps a normalized DB file path (e.g. "media/Vacation/beach.jpg") to an actual file on disk relative to mediaDir.
func ResolvePhysicalPath(mediaDir, filePath string) string {
	cleanPath := filepath.ToSlash(filepath.Clean(filePath))

	// If filePath is already an absolute path that exists on disk, return it directly
	if filepath.IsAbs(cleanPath) {
		return cleanPath
	}

	relPath := strings.TrimPrefix(cleanPath, "media/")
	relPath = strings.TrimPrefix(relPath, "/")

	return filepath.ToSlash(filepath.Clean(filepath.Join(mediaDir, relPath)))
}
