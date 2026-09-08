package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// ThumbnailCacheMiddleware sets aggressive caching headers for static thumbnails
func ThumbnailCacheMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.Next()
	}
}

// MediaCacheMiddleware handles ETag and conditional requests for raw media files
func MediaCacheMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "public, max-age=86400")
		return c.Next()
	}
}

// GenerateETag returns weak or strong ETag for a given byte slice or string identifier
func GenerateETag(identifier string) string {
	hash := md5.Sum([]byte(identifier))
	return fmt.Sprintf(`W/"%s"`, hex.EncodeToString(hash[:]))
}

// CheckETag checks If-None-Match header and returns true if cache is valid (304)
func CheckETag(c *fiber.Ctx, etag string) bool {
	clientETag := c.Get("If-None-Match")
	if clientETag != "" && (clientETag == etag || clientETag == "*") {
		return true
	}
	return false
}
