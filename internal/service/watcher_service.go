package service

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gallery-be/config"

	"github.com/fsnotify/fsnotify"
)

type WatcherService interface {
	StartWatching() error
	Stop()
}

type watcherService struct {
	cfg        *config.Config
	scannerSvc ScannerService
	watcher    *fsnotify.Watcher
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func NewWatcherService(cfg *config.Config, scannerSvc ScannerService) WatcherService {
	return &watcherService{
		cfg:        cfg,
		scannerSvc: scannerSvc,
		stopChan:   make(chan struct{}),
	}
}

func (w *watcherService) StartWatching() error {
	if !w.cfg.AutoScanEnabled {
		log.Println("[WATCHER] Auto-scan is disabled via configuration (AUTO_SCAN_ENABLED=false)")
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = watcher

	// Recursively add all existing directories under MediaDir
	if err := w.addRecursiveWatch(w.cfg.MediaDir); err != nil {
		log.Printf("[WATCHER] Warning: failed to watch media root %s: %v", w.cfg.MediaDir, err)
	}

	w.wg.Add(1)
	go w.eventLoop()

	log.Printf("[WATCHER] Real-time file watcher started on %s (Debounce: %ds)", w.cfg.MediaDir, w.cfg.AutoScanDebounceSec)
	return nil
}

func (w *watcherService) Stop() {
	if w.watcher == nil {
		return
	}
	close(w.stopChan)
	_ = w.watcher.Close()
	w.wg.Wait()
	log.Println("[WATCHER] Real-time file watcher stopped")
}

func (w *watcherService) addRecursiveWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			if err := w.watcher.Add(path); err != nil {
				log.Printf("[WATCHER] Failed to add watch for directory %s: %v", path, err)
			}
		}
		return nil
	})
}

func (w *watcherService) eventLoop() {
	defer w.wg.Done()

	debounceDuration := time.Duration(w.cfg.AutoScanDebounceSec) * time.Second
	if debounceDuration <= 0 {
		debounceDuration = 3 * time.Second
	}

	var timer *time.Timer
	var timerChan <-chan time.Time

	for {
		select {
		case <-w.stopChan:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Ignore temporary/hidden files, dotfiles, database files, and thumbnails
			baseName := filepath.Base(event.Name)
			ext := strings.ToLower(filepath.Ext(baseName))
			if strings.HasPrefix(baseName, ".") ||
				strings.HasSuffix(baseName, ".tmp.jpg") ||
				strings.HasSuffix(baseName, ".tmp") ||
				ext == ".db" || ext == ".db-wal" || ext == ".db-shm" || ext == ".db-journal" {
				continue
			}

			// If a new directory was created, dynamically add it to fsnotify watcher
			if event.Has(fsnotify.Create) {
				if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
					log.Printf("[WATCHER] New directory detected: %s, adding to watcher", event.Name)
					_ = w.addRecursiveWatch(event.Name)
				}
			}

			// Filter meaningful operations
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				log.Printf("[WATCHER] Detected file change (%s): %s. Debouncing scan...", event.Op.String(), event.Name)

				if timer != nil {
					timer.Stop()
				}
				timer = time.NewTimer(debounceDuration)
				timerChan = timer.C
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[WATCHER] File watcher error: %v", err)

		case <-timerChan:
			timerChan = nil
			timer = nil
			log.Println("[WATCHER] Debounce timer expired. Triggering automatic media scan...")

			// Trigger scan safely in background
			go func() {
				if _, err := w.scannerSvc.StartScan(); err != nil {
					log.Printf("[WATCHER] Auto-scan trigger info: %v", err)
				}
			}()
		}
	}
}
