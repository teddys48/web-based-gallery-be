package service_test

import (
	"testing"
	"time"

	"gallery-be/config"
	"gallery-be/internal/database"
	"gallery-be/internal/model"
	"gallery-be/internal/repository"
	"gallery-be/internal/service"
)

func TestPhotoServiceAndScanner(t *testing.T) {
	cfg := &config.Config{
		Port:           "8089",
		MediaDir:       "../../media",
		ThumbnailDir:   "../../.thumbnails_test",
		DBPath:         "../../gallery_test.db",
		ThumbWidth:     200,
		ThumbHeight:    200,
		MaxScanWorkers: 2,
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	photoRepo := repository.NewPhotoRepository(db)
	folderRepo := repository.NewFolderRepository(db)

	exifSvc := service.NewEXIFService()
	thumbSvc := service.NewThumbnailService(cfg)
	photoSvc := service.NewPhotoService(photoRepo, folderRepo, thumbSvc)
	scannerSvc := service.NewScannerService(cfg, photoRepo, exifSvc, thumbSvc)

	// Test Manual Scan Trigger
	status, err := scannerSvc.StartScan()
	if err != nil {
		t.Fatalf("StartScan failed: %v", err)
	}
	if !status.IsScanning && status.TotalFound == 0 {
		t.Errorf("Expected scan to start")
	}

	// Wait briefly for scan to complete
	time.Sleep(1 * time.Second)

	// Test Timeline View
	photos, total, err := photoSvc.GetTimeline(1, 10)
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}
	if total == 0 {
		t.Errorf("Expected photos in database, got 0")
	}
	t.Logf("Found %d total photos in timeline", total)

	// Test Timeline Buckets
	buckets, err := photoSvc.GetTimelineBuckets()
	if err != nil {
		t.Fatalf("GetTimelineBuckets failed: %v", err)
	}
	t.Logf("Found %d timeline buckets", len(buckets))

	// Test Folder Tree
	tree, err := photoSvc.GetFolderTree()
	if err != nil {
		t.Fatalf("GetFolderTree failed: %v", err)
	}
	if len(tree) == 0 {
		t.Errorf("Expected folder tree nodes, got 0")
	}
	t.Logf("Folder tree top-level count: %d", len(tree))

	// Test Photo Detail
	if len(photos) > 0 {
		first := photos[0]
		p, err := photoSvc.GetPhotoByID(first.ID)
		if err != nil {
			t.Fatalf("GetPhotoByID failed: %v", err)
		}
		if p.FileName != first.FileName {
			t.Errorf("Expected filename %s, got %s", first.FileName, p.FileName)
		}

		// Test GetPhotoThumbnailByID
		thumbPath, _, err := photoSvc.GetPhotoThumbnailByID(first.ID)
		if err != nil || thumbPath == "" {
			t.Fatalf("GetPhotoThumbnailByID failed: %v", err)
		}
		t.Logf("Thumbnail path by ID: %s", thumbPath)

		// Test GetPhotoThumbnailByPath
		thumbPathByPath, err := photoSvc.GetPhotoThumbnailByPath(first.FilePath)
		if err != nil || thumbPathByPath == "" {
			t.Fatalf("GetPhotoThumbnailByPath failed: %v", err)
		}
		t.Logf("Thumbnail path by Path: %s", thumbPathByPath)

		// Test GetPhotosByDate
		targetDate := first.TakenAt.Format("2006-01-02")
		datePhotos, dateTotal, err := photoSvc.GetPhotosByDate(model.DateFilter{
			Date:  targetDate,
			Page:  1,
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("GetPhotosByDate failed: %v", err)
		}
		if dateTotal == 0 || len(datePhotos) == 0 {
			t.Errorf("Expected photos on date %s, got total %d, items %d", targetDate, dateTotal, len(datePhotos))
		}
		t.Logf("Found %d photos on date %s", dateTotal, targetDate)

		// Test GetPhotosByDateGrouped
		groups, err := photoSvc.GetPhotosByDateGrouped(model.DateFilter{
			Date: targetDate,
		})
		if err != nil {
			t.Fatalf("GetPhotosByDateGrouped failed: %v", err)
		}
		if len(groups) == 0 {
			t.Errorf("Expected date groups for %s, got 0", targetDate)
		}
		t.Logf("Found %d date groups for date %s", len(groups), targetDate)
	}
}
