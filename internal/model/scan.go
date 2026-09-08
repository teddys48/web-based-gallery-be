package model

import "time"

type ScanStatus struct {
	IsScanning   bool       `json:"is_scanning"`
	ScannedCount int        `json:"scanned_count"`
	NewCount     int        `json:"new_count"`
	UpdatedCount int        `json:"updated_count"`
	ErrorCount   int        `json:"error_count"`
	TotalFound   int        `json:"total_found"`
	CurrentFile  string     `json:"current_file,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
}
