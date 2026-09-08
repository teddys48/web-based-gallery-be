package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gallery-be/config"
	"gallery-be/internal/model"
	"gallery-be/internal/repository"
)

type ScannerService interface {
	StartScan() (*model.ScanStatus, error)
	GetStatus() model.ScanStatus
}

type scannerService struct {
	cfg          *config.Config
	repo         repository.PhotoRepository
	exifSvc      EXIFService
	thumbSvc     ThumbnailService
	status       model.ScanStatus
	statusMu     sync.RWMutex
	imageExts    map[string]bool
	videoExts    map[string]bool
	supportedExt map[string]bool
}

func NewScannerService(
	cfg *config.Config,
	repo repository.PhotoRepository,
	exifSvc EXIFService,
	thumbSvc ThumbnailService,
) ScannerService {
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
		".heic": true,
	}

	videoExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".mov":  true,
		".avi":  true,
		".webm": true,
		".m4v":  true,
		".flv":  true,
		".3gp":  true,
		".ts":   true,
		".wmv":  true,
	}

	supportedExt := make(map[string]bool)
	for ext := range imageExts {
		supportedExt[ext] = true
	}
	for ext := range videoExts {
		supportedExt[ext] = true
	}

	return &scannerService{
		cfg:          cfg,
		repo:         repo,
		exifSvc:      exifSvc,
		thumbSvc:     thumbSvc,
		imageExts:    imageExts,
		videoExts:    videoExts,
		supportedExt: supportedExt,
		status:       model.ScanStatus{},
	}
}

func (s *scannerService) GetStatus() model.ScanStatus {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	return s.status
}

func (s *scannerService) StartScan() (*model.ScanStatus, error) {
	s.statusMu.Lock()
	if s.status.IsScanning {
		st := s.status
		s.statusMu.Unlock()
		return &st, fmt.Errorf("scan is already in progress")
	}

	now := time.Now()
	s.status = model.ScanStatus{
		IsScanning:   true,
		ScannedCount: 0,
		NewCount:     0,
		UpdatedCount: 0,
		ErrorCount:   0,
		TotalFound:   0,
		StartedAt:    &now,
	}
	currentStatus := s.status
	s.statusMu.Unlock()

	go s.runScan()

	return &currentStatus, nil
}

func (s *scannerService) runScan() {
	log.Println("[SCANNER] Starting media directory scan:", s.cfg.MediaDir)

	if err := os.MkdirAll(s.cfg.MediaDir, 0755); err != nil {
		s.finishScanWithError(fmt.Sprintf("failed to create media directory: %v", err))
		return
	}

	var filesToProcess []string
	err := filepath.WalkDir(s.cfg.MediaDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if s.supportedExt[ext] {
			filesToProcess = append(filesToProcess, path)
		}
		return nil
	})

	if err != nil {
		s.finishScanWithError(fmt.Sprintf("error walking directory: %v", err))
		return
	}

	s.statusMu.Lock()
	s.status.TotalFound = len(filesToProcess)
	s.statusMu.Unlock()

	dbPhotos, err := s.repo.GetAllFilePathsMap()
	if err != nil {
		log.Printf("[SCANNER] Warning: failed to fetch DB photos map: %v", err)
		dbPhotos = make(map[string]model.Photo)
	}

	var scannedPaths []string
	for _, path := range filesToProcess {
		s.statusMu.Lock()
		s.status.CurrentFile = path
		s.statusMu.Unlock()

		scannedPaths = append(scannedPaths, path)

		fi, err := os.Stat(path)
		if err != nil {
			s.incrementError()
			continue
		}

		if existing, ok := dbPhotos[path]; ok {
			if existing.FileSize == fi.Size() && existing.ModTime.Equal(fi.ModTime()) && existing.ThumbnailPath != "" {
				s.incrementScanned(false)
				continue
			}
		}

		photo, err := s.processFile(path, fi)
		if err != nil {
			log.Printf("[SCANNER] Error processing %s: %v", path, err)
			s.incrementError()
			continue
		}

		if err := s.repo.Upsert(photo); err != nil {
			log.Printf("[SCANNER] Error upserting DB for %s: %v", path, err)
			s.incrementError()
			continue
		}

		_, isExisting := dbPhotos[path]
		s.incrementScanned(!isExisting)
	}

	deletedCount, err := s.repo.DeleteMissingPaths(scannedPaths)
	if err != nil {
		log.Printf("[SCANNER] Warning: failed to delete missing records: %v", err)
	} else if deletedCount > 0 {
		log.Printf("[SCANNER] Cleaned up %d missing media items from database.", deletedCount)
	}

	now := time.Now()
	s.statusMu.Lock()
	s.status.IsScanning = false
	s.status.CurrentFile = ""
	s.status.FinishedAt = &now
	s.statusMu.Unlock()

	log.Printf("[SCANNER] Completed. Processed: %d, New: %d, Errors: %d",
		s.status.ScannedCount, s.status.NewCount, s.status.ErrorCount)
}

func (s *scannerService) processFile(path string, fi os.FileInfo) (*model.Photo, error) {
	relFolder := filepath.Dir(path)
	ext := strings.ToLower(filepath.Ext(path))

	hashStr := calculateFileHash(path)

	if s.videoExts[ext] {
		// Process Video File
		thumbPath, duration, width, height, err := s.thumbSvc.GenerateVideoThumbnail(path)
		if err != nil {
			log.Printf("[SCANNER] Failed video thumbnail generation for %s: %v", path, err)
		}

		mimeType := getMIMEType(ext, "video/mp4")

		return &model.Photo{
			FilePath:      path,
			FileName:      fi.Name(),
			FolderPath:    relFolder,
			FileSize:      fi.Size(),
			Hash:          hashStr,
			MediaType:     "video",
			MIMEType:      mimeType,
			Width:         width,
			Height:        height,
			Duration:      duration,
			TakenAt:       fi.ModTime(),
			ThumbnailPath: thumbPath,
			ModTime:       fi.ModTime(),
		}, nil
	}

	// Process Image File
	meta, err := s.exifSvc.ExtractMetadata(path)
	if err != nil {
		meta = &EXIFMetadata{
			TakenAt: fi.ModTime(),
		}
	}

	thumbPath, err := s.thumbSvc.GenerateThumbnail(path)
	if err != nil {
		log.Printf("[SCANNER] Failed thumbnail generation for %s: %v", path, err)
		thumbPath = ""
	}

	mimeType := getMIMEType(ext, "image/jpeg")

	photo := &model.Photo{
		FilePath:      path,
		FileName:      fi.Name(),
		FolderPath:    relFolder,
		FileSize:      fi.Size(),
		Hash:          hashStr,
		MediaType:     "image",
		MIMEType:      mimeType,
		Width:         meta.Width,
		Height:        meta.Height,
		TakenAt:       meta.TakenAt,
		CameraMake:    meta.CameraMake,
		CameraModel:   meta.CameraModel,
		FNumber:       meta.FNumber,
		ExposureTime:  meta.ExposureTime,
		ISO:           meta.ISO,
		FocalLength:   meta.FocalLength,
		Latitude:      meta.Latitude,
		Longitude:     meta.Longitude,
		ThumbnailPath: thumbPath,
		ModTime:       fi.ModTime(),
	}

	return photo, nil
}

func getMIMEType(ext, fallback string) string {
	t := mime.TypeByExtension(ext)
	if t != "" {
		return t
	}
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".avi":
		return "video/x-msvideo"
	default:
		return fallback
	}
}

func calculateFileHash(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, 1024*1024)
	n, _ := io.ReadFull(f, buf)
	h.Write(buf[:n])

	return hex.EncodeToString(h.Sum(nil))
}

func (s *scannerService) incrementScanned(isNew bool) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status.ScannedCount++
	if isNew {
		s.status.NewCount++
	} else {
		s.status.UpdatedCount++
	}
}

func (s *scannerService) incrementError() {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status.ErrorCount++
}

func (s *scannerService) finishScanWithError(errMsg string) {
	now := time.Now()
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status.IsScanning = false
	s.status.FinishedAt = &now
	s.status.LastError = errMsg
	log.Printf("[SCANNER] Scan failed: %s", errMsg)
}
