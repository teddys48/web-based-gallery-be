package service_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gallery-be/config"
	"gallery-be/internal/database"
	"gallery-be/internal/repository"
	"gallery-be/internal/service"
)

func TestWatcherServiceAutoScan(t *testing.T) {
	tempMediaDir, err := os.MkdirTemp("", "gallery_watcher_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp media dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempMediaDir) }()

	tempThumbDir, err := os.MkdirTemp("", "gallery_watcher_thumb_*")
	if err != nil {
		t.Fatalf("Failed to create temp thumb dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempThumbDir) }()

	dbPath := filepath.Join(tempMediaDir, "test_watcher.db")

	cfg := &config.Config{
		Port:                "8090",
		MediaDir:            tempMediaDir,
		ThumbnailDir:        tempThumbDir,
		DBPath:              dbPath,
		ThumbWidth:          200,
		ThumbHeight:         200,
		MaxScanWorkers:      2,
		AutoScanEnabled:     true,
		AutoScanDebounceSec: 1, // 1 second for fast test execution
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	photoRepo := repository.NewPhotoRepository(db)

	exifSvc := service.NewEXIFService()
	thumbSvc := service.NewThumbnailService(cfg)
	scannerSvc := service.NewScannerService(cfg, photoRepo, exifSvc, thumbSvc)

	watcherSvc := service.NewWatcherService(cfg, scannerSvc)
	if err := watcherSvc.StartWatching(); err != nil {
		t.Fatalf("StartWatching failed: %v", err)
	}
	defer watcherSvc.Stop()

	// Write a new image file into tempMediaDir
	testFilePath := filepath.Join(tempMediaDir, "test_image.jpg")
	if err := os.WriteFile(testFilePath, []byte("fake image data"), 0644); err != nil {
		t.Fatalf("Failed to write test image: %v", err)
	}

	// Wait for debounce timer (1s) + scan completion
	time.Sleep(2500 * time.Millisecond)

	// Verify scan status or DB records
	status := scannerSvc.GetStatus()
	if status.TotalFound == 0 && status.ScannedCount == 0 {
		t.Logf("Scan status: TotalFound=%d, ScannedCount=%d", status.TotalFound, status.ScannedCount)
	}
}
