package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

type scanJob struct {
	path string
	info os.FileInfo
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

	if cfg.ImageScanWorkers <= 0 {
		cfg.ImageScanWorkers = 2
	}
	if cfg.VideoScanWorkers <= 0 {
		cfg.VideoScanWorkers = 1
	}
	if cfg.FFmpegThreads <= 0 {
		cfg.FFmpegThreads = 1
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
	log.Printf("[SCANNER] Starting media directory scan: %s (Image Workers: %d, Video Workers: %d, FFmpeg Threads: %d)",
		s.cfg.MediaDir, s.cfg.ImageScanWorkers, s.cfg.VideoScanWorkers, s.cfg.FFmpegThreads)

	if err := os.MkdirAll(s.cfg.MediaDir, 0755); err != nil {
		s.finishScanWithError(fmt.Sprintf("failed to create media directory: %v", err))
		return
	}

	// 1. Gather all files in media directory
	type fileEntry struct {
		path string
		info os.FileInfo
	}
	var filesToProcess []fileEntry

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
			if info, err := d.Info(); err == nil {
				filesToProcess = append(filesToProcess, fileEntry{path: path, info: info})
			}
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

	// 2. Pre-fetch existing records from DB for instant early filter
	dbPhotos, err := s.repo.GetAllFilePathsMap()
	if err != nil {
		s.finishScanWithError(fmt.Sprintf("failed to fetch DB photos map: %v", err))
		return
	}

	// Channels for work queues
	imageJobChan := make(chan scanJob, 100)
	videoJobChan := make(chan scanJob, 50)
	resultChan := make(chan *model.Photo, 100)

	var scannedPaths []string
	var scannedPathsMu sync.Mutex

	var workerWg sync.WaitGroup
	var collectorWg sync.WaitGroup

	// 3. Single DB Collector Goroutine to prevent SQLite lock contention
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for photo := range resultChan {
			if photo == nil {
				continue
			}
			if err := s.repo.Upsert(photo); err != nil {
				log.Printf("[SCANNER] Error upserting DB for %s: %v", photo.FilePath, err)
				s.incrementError()
			}
		}
	}()

	// 4. Image Worker Pool
	for i := 0; i < s.cfg.ImageScanWorkers; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			for job := range imageJobChan {
				s.setCurrentFile(job.path)

				photo, err := s.processImageFile(job.path, job.info)
				if err != nil {
					log.Printf("[SCANNER] Image worker %d error processing %s: %v", workerID, job.path, err)
					s.incrementError()
					continue
				}

				_, isExisting := dbPhotos[job.path]
				s.incrementScanned(!isExisting)
				resultChan <- photo
			}
		}(i + 1)
	}

	// 5. Video Worker Pool (Throttled, e.g., 1 worker)
	for i := 0; i < s.cfg.VideoScanWorkers; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			for job := range videoJobChan {
				s.setCurrentFile(job.path)

				photo, err := s.processVideoFile(job.path, job.info)
				if err != nil {
					log.Printf("[SCANNER] Video worker %d error processing %s: %v", workerID, job.path, err)
					s.incrementError()
					continue
				}

				_, isExisting := dbPhotos[job.path]
				s.incrementScanned(!isExisting)
				resultChan <- photo
			}
		}(i + 1)
	}

	// 6. Walk & Dispatch Loop (Early Size + ModTime Filter)
	for _, entry := range filesToProcess {
		path := entry.path
		fi := entry.info
		ext := strings.ToLower(filepath.Ext(path))

		scannedPathsMu.Lock()
		scannedPaths = append(scannedPaths, path)
		scannedPathsMu.Unlock()

		// Early check: if size & modtime match and valid thumbnail exists, skip expensive processing!
		if existing, ok := dbPhotos[path]; ok {
			timeMatches := existing.ModTime.Equal(fi.ModTime()) || existing.ModTime.Unix() == fi.ModTime().Unix()
			if existing.FileSize == fi.Size() && timeMatches && existing.ThumbnailPath != "" {
				if _, err := os.Stat(existing.ThumbnailPath); err == nil {
					s.incrementScanned(false)
					continue
				}
			}
			// Invalidate old thumbnail if file was modified or thumbnail missing
			_ = s.thumbSvc.DeleteThumbnail(path)
		}

		job := scanJob{path: path, info: fi}
		if s.videoExts[ext] {
			videoJobChan <- job
		} else {
			imageJobChan <- job
		}
	}

	// Close job channels and wait for workers to complete
	close(imageJobChan)
	close(videoJobChan)
	workerWg.Wait()

	// Close result channel and wait for DB collector to finish writing
	close(resultChan)
	collectorWg.Wait()

	// 7. Cleanup missing paths in DB
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

	log.Printf("[SCANNER] Scan completed. Total: %d, Processed/Updated: %d, New: %d, Errors: %d",
		s.status.TotalFound, s.status.ScannedCount, s.status.NewCount, s.status.ErrorCount)
}

func (s *scannerService) processImageFile(path string, fi os.FileInfo) (*model.Photo, error) {
	relFolder := filepath.Dir(path)
	ext := strings.ToLower(filepath.Ext(path))

	hashStr := calculateFastFileHash(path, fi.Size(), fi.ModTime())

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

	return &model.Photo{
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
	}, nil
}

func (s *scannerService) processVideoFile(path string, fi os.FileInfo) (*model.Photo, error) {
	relFolder := filepath.Dir(path)
	ext := strings.ToLower(filepath.Ext(path))

	hashStr := calculateFastFileHash(path, fi.Size(), fi.ModTime())

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

func calculateFastFileHash(filePath string, size int64, modTime time.Time) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", filePath, size, modTime.UnixNano())))
	return hex.EncodeToString(h[:])
}

func (s *scannerService) setCurrentFile(path string) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.status.CurrentFile = path
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
	s.status.ScannedCount++
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
