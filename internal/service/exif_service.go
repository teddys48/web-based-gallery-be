package service

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"

	_ "golang.org/x/image/webp"
)

type EXIFMetadata struct {
	Width        int
	Height       int
	TakenAt      time.Time
	CameraMake   string
	CameraModel  string
	FNumber      string
	ExposureTime string
	ISO          int
	FocalLength  string
	Latitude     *float64
	Longitude    *float64
}

type EXIFService interface {
	ExtractMetadata(filePath string) (*EXIFMetadata, error)
}

type exifService struct{}

func NewEXIFService() EXIFService {
	return &exifService{}
}

func (s *exifService) ExtractMetadata(filePath string) (*EXIFMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	modTime := fi.ModTime()

	meta := &EXIFMetadata{
		TakenAt: modTime, // Default fallback
	}

	// 1. Extract width & height
	cfg, _, err := image.DecodeConfig(file)
	if err == nil {
		meta.Width = cfg.Width
		meta.Height = cfg.Height
	}

	// Reset file pointer for EXIF reading
	if _, err := file.Seek(0, 0); err != nil {
		return meta, nil
	}

	// 2. Extract EXIF tags
	x, err := exif.Decode(file)
	if err != nil {
		// No EXIF or error decoding EXIF, return metadata with modTime & width/height
		return meta, nil
	}

	// Extract DateTimeOriginal
	if tm, err := x.DateTime(); err == nil {
		meta.TakenAt = tm
	}

	// Camera Make & Model
	if makeTag, err := x.Get(exif.Make); err == nil {
		if val, err := makeTag.StringVal(); err == nil {
			meta.CameraMake = val
		}
	}
	if modelTag, err := x.Get(exif.Model); err == nil {
		if val, err := modelTag.StringVal(); err == nil {
			meta.CameraModel = val
		}
	}

	// FNumber
	if fnTag, err := x.Get(exif.FNumber); err == nil {
		if num, den, err := fnTag.Rat2(0); err == nil && den != 0 {
			meta.FNumber = fmt.Sprintf("f/%.1f", float64(num)/float64(den))
		}
	}

	// ExposureTime
	if expTag, err := x.Get(exif.ExposureTime); err == nil {
		if num, den, err := expTag.Rat2(0); err == nil && den != 0 {
			if num == 1 {
				meta.ExposureTime = fmt.Sprintf("1/%d", den)
			} else {
				meta.ExposureTime = fmt.Sprintf("%.2fs", float64(num)/float64(den))
			}
		}
	}

	// ISO
	if isoTag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if val, err := isoTag.Int(0); err == nil {
			meta.ISO = val
		}
	}

	// FocalLength
	if flTag, err := x.Get(exif.FocalLength); err == nil {
		if num, den, err := flTag.Rat2(0); err == nil && den != 0 {
			meta.FocalLength = fmt.Sprintf("%.1fmm", float64(num)/float64(den))
		}
	}

	// GPS Coordinates
	lat, long, err := x.LatLong()
	if err == nil {
		meta.Latitude = &lat
		meta.Longitude = &long
	}

	return meta, nil
}
