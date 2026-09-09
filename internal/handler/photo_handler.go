package handler

import (
	"fmt"
	"os"
	"strconv"

	"gallery-be/config"
	"gallery-be/internal/middleware"
	"gallery-be/internal/service"
	"gallery-be/pkg/response"
	"gallery-be/pkg/util"

	"github.com/gofiber/fiber/v2"
)

type PhotoHandler struct {
	photoSvc service.PhotoService
	cfg      *config.Config
}

func NewPhotoHandler(photoSvc service.PhotoService, cfg *config.Config) *PhotoHandler {
	return &PhotoHandler{
		photoSvc: photoSvc,
		cfg:      cfg,
	}
}

// GetPhotoDetail GET /api/v1/photos/:id
func (h *PhotoHandler) GetPhotoDetail(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid photo ID")
	}

	photo, err := h.photoSvc.GetPhotoByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Photo not found")
	}

	return response.Success(c, fiber.StatusOK, "Photo retrieved successfully", photo)
}

// ServeThumbnail GET /api/v1/photos/:id/thumbnail or GET /api/v1/thumbnails/:id
func (h *PhotoHandler) ServeThumbnail(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid photo ID")
	}

	thumbPath, _, err := h.photoSvc.GetPhotoThumbnailByID(uint(id))
	if err != nil || thumbPath == "" {
		return response.Error(c, fiber.StatusNotFound, "Thumbnail not found")
	}

	// Check if thumbnail file exists
	fi, err := os.Stat(thumbPath)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Thumbnail file missing from disk")
	}

	// Set browser caching headers
	etag := middleware.GenerateETag(fmt.Sprintf("%s-%d-%d", thumbPath, fi.Size(), fi.ModTime().Unix()))
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")

	if middleware.CheckETag(c, etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return c.SendFile(thumbPath)
}

// ServeThumbnailByPath GET /api/v1/thumbnails?path=media/Vacation2025/beach.jpg
func (h *PhotoHandler) ServeThumbnailByPath(c *fiber.Ctx) error {
	filePath := c.Query("path", "")
	if filePath == "" {
		return response.Error(c, fiber.StatusBadRequest, "Path query parameter is required")
	}

	physicalPath := util.ResolvePhysicalPath(h.cfg.MediaDir, filePath)
	if _, err := os.Stat(physicalPath); err != nil {
		return response.Error(c, fiber.StatusNotFound, "Source file missing from disk")
	}

	thumbPath, err := h.photoSvc.GetPhotoThumbnailByPath(filePath)
	if err != nil || thumbPath == "" {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate or retrieve thumbnail")
	}

	fi, err := os.Stat(thumbPath)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Thumbnail file missing from disk")
	}

	etag := middleware.GenerateETag(fmt.Sprintf("%s-%d-%d", thumbPath, fi.Size(), fi.ModTime().Unix()))
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")

	if middleware.CheckETag(c, etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return c.SendFile(thumbPath)
}

// ServeRawPhoto GET /api/v1/photos/:id/raw
func (h *PhotoHandler) ServeRawPhoto(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid photo ID")
	}

	photo, err := h.photoSvc.GetPhotoByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Photo not found")
	}

	physicalPath := util.ResolvePhysicalPath(h.cfg.MediaDir, photo.FilePath)
	fi, err := os.Stat(physicalPath)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Photo file missing from disk")
	}

	// Set browser caching headers
	etag := middleware.GenerateETag(fmt.Sprintf("%s-%d-%d", physicalPath, fi.Size(), fi.ModTime().Unix()))
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, max-age=86400")

	if middleware.CheckETag(c, etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	c.Set("Content-Type", photo.MIMEType)
	return c.SendFile(physicalPath)
}
