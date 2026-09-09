package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
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
	DeleteThumbnail(sourcePath string) error
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

func (s *thumbnailService) DeleteThumbnail(sourcePath string) error {
	destPath := s.GetThumbnailPath(sourcePath)
	tempPath := strings.TrimSuffix(destPath, ".jpg") + ".tmp.jpg"
	_ = os.Remove(tempPath)
	if fileExists(destPath) {
		return os.Remove(destPath)
	}
	return nil
}

func (s *thumbnailService) GenerateThumbnail(sourcePath string) (string, error) {
	destPath := s.GetThumbnailPath(sourcePath)

	if isValidThumbnail(destPath) {
		return destPath, nil
	}

	srcImg, err := imaging.Open(sourcePath, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("failed to open source image for thumbnail: %w", err)
	}

	thumb := imaging.Fit(srcImg, s.cfg.ThumbWidth, s.cfg.ThumbHeight, imaging.Lanczos)

	tempPath := strings.TrimSuffix(destPath, ".jpg") + ".tmp.jpg"
	defer func() { _ = os.Remove(tempPath) }()

	if err := imaging.Save(thumb, tempPath, imaging.JPEGQuality(80)); err != nil {
		return "", fmt.Errorf("failed to save thumbnail temp file: %w", err)
	}

	if !isValidThumbnail(tempPath) {
		return "", fmt.Errorf("generated thumbnail failed image validation: %s", tempPath)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		return "", fmt.Errorf("failed to rename thumbnail temp file to destination: %w", err)
	}

	return destPath, nil
}

type ffprobeOutput struct {
	Streams []struct {
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Duration string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func (s *thumbnailService) extractVideoMetadata(sourcePath string) (float64, int, int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration:format=duration",
		"-of", "json",
		sourcePath,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	var data ffprobeOutput
	if err := json.Unmarshal(out, &data); err != nil {
		return 0, 0, 0, err
	}

	var duration float64
	var width, height int

	if len(data.Streams) > 0 {
		width = data.Streams[0].Width
		height = data.Streams[0].Height
		if durStr := data.Streams[0].Duration; durStr != "" {
			duration, _ = strconv.ParseFloat(durStr, 64)
		}
	}

	if duration == 0 && data.Format.Duration != "" {
		duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
	}

	return duration, width, height, nil
}

func (s *thumbnailService) GenerateVideoThumbnail(sourcePath string) (string, float64, int, int, error) {
	destPath := s.GetThumbnailPath(sourcePath)

	duration, origW, origH, _ := s.extractVideoMetadata(sourcePath)

	// If thumbnail already exists and is valid
	if isValidThumbnail(destPath) {
		return destPath, duration, origW, origH, nil
	}

	tempPath := strings.TrimSuffix(destPath, ".jpg") + ".tmp.jpg"
	defer func() { _ = os.Remove(tempPath) }()

	threads := strconv.Itoa(s.cfg.FFmpegThreads)

	// 1. Try extracting frame via ffmpeg at 1.0s to tempPath
	cmd := exec.Command("ffmpeg",
		"-y",
		"-threads", threads,
		"-ss", "00:00:01",
		"-i", sourcePath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", s.cfg.ThumbWidth),
		tempPath,
	)

	if err := cmd.Run(); err != nil || !isValidThumbnail(tempPath) {
		// Retry at 00:00:00
		cmdRetry := exec.Command("ffmpeg",
			"-y",
			"-threads", threads,
			"-ss", "00:00:00",
			"-i", sourcePath,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", s.cfg.ThumbWidth),
			tempPath,
		)
		_ = cmdRetry.Run()
	}

	// If ffmpeg succeeded, overlay video play icon
	if isValidThumbnail(tempPath) {
		if frameImg, err := imaging.Open(tempPath); err == nil {
			overlayImg := addPlayIconOverlay(frameImg)
			if err := imaging.Save(overlayImg, tempPath, imaging.JPEGQuality(85)); err == nil && isValidThumbnail(tempPath) {
				if err := os.Rename(tempPath, destPath); err == nil {
					return destPath, duration, origW, origH, nil
				}
			}
		}
	}

	// Fallback: Generate synthetic video placeholder thumbnail if ffmpeg fails
	log.Printf("[THUMBNAIL] ffmpeg frame extraction failed for %s, creating fallback thumbnail", sourcePath)
	placeholder := image.NewRGBA(image.Rect(0, 0, s.cfg.ThumbWidth, s.cfg.ThumbHeight))
	darkBlue := color.RGBA{R: 30, G: 45, B: 70, A: 255}
	draw.Draw(placeholder, placeholder.Bounds(), &image.Uniform{C: darkBlue}, image.Point{}, draw.Src)
	overlayImg := addPlayIconOverlay(placeholder)

	if err := imaging.Save(overlayImg, tempPath, imaging.JPEGQuality(85)); err != nil {
		return "", duration, origW, origH, err
	}

	if !isValidThumbnail(tempPath) {
		return "", duration, origW, origH, fmt.Errorf("fallback video thumbnail failed image validation: %s", tempPath)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		return "", duration, origW, origH, fmt.Errorf("failed to rename video thumbnail temp file: %w", err)
	}

	return destPath, duration, origW, origH, nil
}

func addPlayIconOverlay(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	w := bounds.Dx()
	h := bounds.Dy()
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0

	// Radius of play button circle
	r := float64(w)
	if float64(h) < r {
		r = float64(h)
	}
	r = r / 5.0
	if r < 18.0 {
		r = 18.0
	}
	r2 := r * r

	// Triangle vertices for play icon ▶
	x1 := cx - r*0.35
	y1 := cy - r*0.5
	x2 := cx - r*0.35
	y2 := cy + r*0.5
	x3 := cx + r*0.55
	y3 := cy

	minX := int(cx - r - 2)
	maxX := int(cx + r + 2)
	minY := int(cy - r - 2)
	maxY := int(cy + r + 2)

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			if x < 0 || x >= w || y < 0 || y >= h {
				continue
			}
			px := float64(x)
			py := float64(y)

			dx := px - cx
			dy := py - cy
			dist2 := dx*dx + dy*dy

			if dist2 <= r2 {
				if pointInTriangle(px, py, x1, y1, x2, y2, x3, y3) {
					// Solid white play arrow
					dst.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				} else {
					// Semi-transparent dark circle background
					orig := dst.At(x, y)
					or, og, ob, _ := orig.RGBA()
					nr := uint8((or >> 8) * 3 / 10)
					ng := uint8((og >> 8) * 3 / 10)
					nb := uint8((ob >> 8) * 3 / 10)
					dst.Set(x, y, color.RGBA{R: nr, G: ng, B: nb, A: 255})
				}
			}
		}
	}

	return dst
}

func pointInTriangle(px, py, x1, y1, x2, y2, x3, y3 float64) bool {
	d1 := sign(px, py, x1, y1, x2, y2)
	d2 := sign(px, py, x2, y2, x3, y3)
	d3 := sign(px, py, x3, y3, x1, y1)

	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)

	return !(hasNeg && hasPos)
}

func sign(pX, pY, x1, y1, x2, y2 float64) float64 {
	return (pX-x2)*(y1-y2) - (x1-x2)*(pY-y2)
}

func isValidThumbnail(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() == 0 {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, _, err = image.DecodeConfig(f)
	if err != nil {
		log.Printf("[THUMBNAIL] Corrupt thumbnail detected at %s (%v), removing file", path, err)
		_ = os.Remove(path)
		return false
	}

	return true
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}


