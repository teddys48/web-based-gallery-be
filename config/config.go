package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MediaDir       string
	ThumbnailDir   string
	DBPath         string
	ThumbWidth     int
	ThumbHeight    int
	MaxScanWorkers int
}

func LoadConfig() *Config {
	// Automatically load .env file if available
	if err := godotenv.Load(); err == nil {
		log.Println("[CONFIG] Successfully loaded environment variables from .env")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		MediaDir:       getEnv("MEDIA_DIR", "./media"),
		ThumbnailDir:   getEnv("THUMBNAIL_DIR", "./.thumbnails"),
		DBPath:         getEnv("DB_PATH", "./gallery.db"),
		ThumbWidth:     getEnvAsInt("THUMB_WIDTH", 400),
		ThumbHeight:    getEnvAsInt("THUMB_HEIGHT", 400),
		MaxScanWorkers: getEnvAsInt("MAX_SCAN_WORKERS", 4),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return fallback
}
