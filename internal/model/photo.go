package model

import (
	"time"
)

type Photo struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	FilePath      string    `gorm:"type:varchar(512);uniqueIndex;not null" json:"file_path"`
	FileName      string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FolderPath    string    `gorm:"type:varchar(512);index;not null" json:"folder_path"`
	FileSize      int64     `gorm:"not null" json:"file_size"`
	Hash          string    `gorm:"type:varchar(64);index" json:"hash"`
	MediaType     string    `gorm:"type:varchar(32);default:'image'" json:"media_type"` // "image" or "video"
	MIMEType      string    `gorm:"type:varchar(64)" json:"mime_type"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	Duration      float64   `json:"duration,omitempty"` // Video duration in seconds
	TakenAt       time.Time `gorm:"index:idx_taken_at,sort:desc" json:"taken_at"`
	CameraMake    string    `gorm:"type:varchar(128)" json:"camera_make,omitempty"`
	CameraModel   string    `gorm:"type:varchar(128)" json:"camera_model,omitempty"`
	FNumber       string    `gorm:"type:varchar(32)" json:"f_number,omitempty"`
	ExposureTime  string    `gorm:"type:varchar(32)" json:"exposure_time,omitempty"`
	ISO           int       `json:"iso,omitempty"`
	FocalLength   string    `gorm:"type:varchar(32)" json:"focal_length,omitempty"`
	Latitude      *float64  `json:"latitude,omitempty"`
	Longitude     *float64  `json:"longitude,omitempty"`
	ThumbnailPath string    `gorm:"type:varchar(512)" json:"thumbnail_path"`
	ModTime       time.Time `json:"mod_time"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TimelineBucket struct {
	Year  int   `json:"year"`
	Month int   `json:"month"`
	Count int64 `json:"count"`
}
