package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gallery-be/config"

	"github.com/disintegration/imaging"
)

type ThumbnailService interface {
	GenerateThumbnail(sourcePath string) (string, error)
	GenerateVideoThumbnail(sourcePath string) (string, float64, int, int, error)
	GetThumbnailPath(sourcePath string) string
}

type thumbnailService struct {
	cfg *config.Config
}

func NewThumbnailService(cfg *config.Config) ThumbnailService {
	_ = os.MkdirAll(cfg.ThumbnailDir, 0755)
	return &thumbnailService{cfg: cfg}
}

func (s *thumbnailService) GetThumbnailPath(sourcePath string) string {
	hash := sha256.Sum256([]byte(sourcePath))
	hashStr := hex.EncodeToString(hash[:16])
	return filepath.Join(s.cfg.ThumbnailDir, fmt.Sprintf("thumb_%s.jpg", hashStr))
}

func (s *thumbnailService) GenerateThumbnail(sourcePath string) (string, error) {
	destPath := s.GetThumbnailPath(sourcePath)

	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	srcImg, err := imaging.Open(sourcePath, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("failed to open source image for thumbnail: %w", err)
	}

	thumb := imaging.Fit(srcImg, s.cfg.ThumbWidth, s.cfg.ThumbHeight, imaging.Lanczos)

	if err := imaging.Save(thumb, destPath, imaging.JPEGQuality(80)); err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return destPath, nil
}

func (s *thumbnailService) GenerateVideoThumbnail(sourcePath string) (string, float64, int, int, error) {
	destPath := s.GetThumbnailPath(sourcePath)

	duration := s.extractVideoDuration(sourcePath)

	// If thumbnail already exists
	if _, err := os.Stat(destPath); err == nil {
		w, h := getThumbnailDimensions(destPath)
		return destPath, duration, w, h, nil
	}

	// 1. Try extracting frame via ffmpeg at 1.0s (or 0.0s)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", "00:00:01",
		"-i", sourcePath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", s.cfg.ThumbWidth, s.cfg.ThumbHeight),
		destPath,
	)

	if err := cmd.Run(); err != nil || !fileExists(destPath) {
		// Retry at 00:00:00
		cmdRetry := exec.Command("ffmpeg",
			"-y",
			"-ss", "00:00:00",
			"-i", sourcePath,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", s.cfg.ThumbWidth, s.cfg.ThumbHeight),
			destPath,
		)
		_ = cmdRetry.Run()
	}

	// If ffmpeg succeeded
	if fileExists(destPath) {
		w, h := getThumbnailDimensions(destPath)
		return destPath, duration, w, h, nil
	}

	// Fallback: Generate synthetic video placeholder thumbnail if ffmpeg fails
	log.Printf("[THUMBNAIL] ffmpeg frame extraction failed for %s, creating fallback thumbnail", sourcePath)
	placeholder := image.NewRGBA(image.Rect(0, 0, s.cfg.ThumbWidth, s.cfg.ThumbHeight))
	blue := color.RGBA{R: 40, G: 60, B: 90, A: 255}
	draw.Draw(placeholder, placeholder.Bounds(), &image.Uniform{C: blue}, image.Point{}, draw.Src)

	if err := imaging.Save(placeholder, destPath, imaging.JPEGQuality(80)); err != nil {
		return "", duration, 0, 0, err
	}

	return destPath, duration, s.cfg.ThumbWidth, s.cfg.ThumbHeight, nil
}

func (s *thumbnailService) extractVideoDuration(sourcePath string) float64 {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		sourcePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	durStr := strings.TrimSpace(string(out))
	if dur, err := strconv.ParseFloat(durStr, 64); err == nil {
		return dur
	}

	return 0
}

func getThumbnailDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}
