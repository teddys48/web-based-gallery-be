package util_test

import (
	"path/filepath"
	"testing"

	"gallery-be/pkg/util"
)

func TestPathNormalizationAndResolution(t *testing.T) {
	testCases := []struct {
		name               string
		mediaDir           string
		inputPath          string
		expectedNormalized string
		expectedPhysical   string
	}{
		{
			name:               "Docker container absolute path",
			mediaDir:           "/app/media",
			inputPath:          "/app/media/Vacation2025/beach.jpg",
			expectedNormalized: "media/Vacation2025/beach.jpg",
			expectedPhysical:   "/app/media/Vacation2025/beach.jpg",
		},
		{
			name:               "Local relative path",
			mediaDir:           "./media",
			inputPath:          "media/Vacation2025/beach.jpg",
			expectedNormalized: "media/Vacation2025/beach.jpg",
			expectedPhysical:   "media/Vacation2025/beach.jpg",
		},
		{
			name:               "Local host absolute path",
			mediaDir:           "/home/teddy/code/gallery-be/media",
			inputPath:          "/home/teddy/code/gallery-be/media/Vacation2025/beach.jpg",
			expectedNormalized: "media/Vacation2025/beach.jpg",
			expectedPhysical:   "/home/teddy/code/gallery-be/media/Vacation2025/beach.jpg",
		},
		{
			name:               "Root media file in Docker",
			mediaDir:           "/app/media",
			inputPath:          "/app/media/photo.jpg",
			expectedNormalized: "media/photo.jpg",
			expectedPhysical:   "/app/media/photo.jpg",
		},
		{
			name:               "API path query without media prefix",
			mediaDir:           "/app/media",
			inputPath:          "Vacation2025/beach.jpg",
			expectedNormalized: "media/Vacation2025/beach.jpg",
			expectedPhysical:   "/app/media/Vacation2025/beach.jpg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			norm := util.NormalizeMediaPath(tc.mediaDir, tc.inputPath)
			if norm != tc.expectedNormalized {
				t.Errorf("NormalizeMediaPath(%s, %s) = %s, expected %s", tc.mediaDir, tc.inputPath, norm, tc.expectedNormalized)
			}

			phys := util.ResolvePhysicalPath(tc.mediaDir, norm)
			expectedPhysClean := filepath.ToSlash(filepath.Clean(tc.expectedPhysical))
			if phys != expectedPhysClean {
				t.Errorf("ResolvePhysicalPath(%s, %s) = %s, expected %s", tc.mediaDir, norm, phys, expectedPhysClean)
			}
		})
	}
}
