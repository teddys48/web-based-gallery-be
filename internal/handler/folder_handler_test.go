package handler_test

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gallery-be/config"
	"gallery-be/internal/database"
	"gallery-be/internal/handler"
	"gallery-be/internal/repository"
	"gallery-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp(t *testing.T) (*fiber.App, string) {
	tempMediaDir, err := os.MkdirTemp("", "gallery_test_media_*")
	if err != nil {
		t.Fatalf("Failed to create temp media dir: %v", err)
	}

	tempThumbDir, err := os.MkdirTemp("", "gallery_test_thumb_*")
	if err != nil {
		t.Fatalf("Failed to create temp thumb dir: %v", err)
	}

	dbPath := filepath.Join(tempMediaDir, "test.db")

	cfg := &config.Config{
		MediaDir:            tempMediaDir,
		ThumbnailDir:        tempThumbDir,
		DBPath:              dbPath,
		AutoScanEnabled:     false,
		AutoScanDebounceSec: 1,
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	photoRepo := repository.NewPhotoRepository(db)
	folderRepo := repository.NewFolderRepository(db)
	thumbSvc := service.NewThumbnailService(cfg)
	photoSvc := service.NewPhotoService(photoRepo, folderRepo, thumbSvc)
	folderHdl := handler.NewFolderHandler(photoSvc, cfg)

	app := fiber.New()
	app.Get("/api/v1/folders/download", folderHdl.DownloadFolderZip)

	return app, tempMediaDir
}

func TestDownloadFolderZipSuccess(t *testing.T) {
	app, tempMediaDir := setupTestApp(t)
	defer func() { _ = os.RemoveAll(tempMediaDir) }()

	// Create folder structure: media/Vacation/Day1/photo1.jpg and media/Vacation/photo2.jpg
	vacationDir := filepath.Join(tempMediaDir, "Vacation")
	day1Dir := filepath.Join(vacationDir, "Day1")
	if err := os.MkdirAll(day1Dir, 0755); err != nil {
		t.Fatalf("Failed to create test dirs: %v", err)
	}

	if err := os.WriteFile(filepath.Join(vacationDir, "photo2.jpg"), []byte("photo2 content"), 0644); err != nil {
		t.Fatalf("Failed to write photo2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(day1Dir, "photo1.jpg"), []byte("photo1 content"), 0644); err != nil {
		t.Fatalf("Failed to write photo1: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/folders/download?path=Vacation", nil)
	resp, err := app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/zip" {
		t.Errorf("Expected Content-Type application/zip, got %s", contentType)
	}

	contentDisposition := resp.Header.Get("Content-Disposition")
	expectedDisposition := `attachment; filename="Vacation.zip"`
	if contentDisposition != expectedDisposition {
		t.Errorf("Expected Content-Disposition %s, got %s", expectedDisposition, contentDisposition)
	}

	// Read and verify zip contents
	zipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("Failed to parse zip archive: %v", err)
	}

	foundFiles := make(map[string]string)
	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("Failed to open file in zip %s: %v", file.Name, err)
		}
		content, _ := io.ReadAll(rc)
		_ = rc.Close()
		foundFiles[file.Name] = string(content)
	}

	if foundFiles["photo2.jpg"] != "photo2 content" {
		t.Errorf("Expected photo2.jpg in zip, got map: %v", foundFiles)
	}
	if foundFiles["Day1/photo1.jpg"] != "photo1 content" {
		t.Errorf("Expected Day1/photo1.jpg in zip, got map: %v", foundFiles)
	}
}

func TestDownloadFolderZipEmptyFolder(t *testing.T) {
	app, tempMediaDir := setupTestApp(t)
	defer func() { _ = os.RemoveAll(tempMediaDir) }()

	// Create empty folder
	emptyDir := filepath.Join(tempMediaDir, "EmptyDir")
	_ = os.MkdirAll(emptyDir, 0755)

	req := httptest.NewRequest("GET", "/api/v1/folders/download?path=EmptyDir", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 Bad Request for empty folder, got %d", resp.StatusCode)
	}
}

func TestDownloadFolderZipPathTraversal(t *testing.T) {
	app, tempMediaDir := setupTestApp(t)
	defer func() { _ = os.RemoveAll(tempMediaDir) }()

	// Attempt path traversal escaping MEDIA_DIR
	req := httptest.NewRequest("GET", "/api/v1/folders/download?path=../../etc", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected error status for path traversal attempt, got %d", resp.StatusCode)
	}
}
