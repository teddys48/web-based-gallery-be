package handler

import (
	"fmt"
	"os"
	"strconv"

	"gallery-be/internal/middleware"
	"gallery-be/internal/service"
	"gallery-be/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type PhotoHandler struct {
	photoSvc service.PhotoService
}

func NewPhotoHandler(photoSvc service.PhotoService) *PhotoHandler {
	return &PhotoHandler{photoSvc: photoSvc}
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

// ServeThumbnail GET /api/v1/photos/:id/thumbnail
func (h *PhotoHandler) ServeThumbnail(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid photo ID")
	}

	photo, err := h.photoSvc.GetPhotoByID(uint(id))
	if err != nil || photo.ThumbnailPath == "" {
		return response.Error(c, fiber.StatusNotFound, "Thumbnail not found")
	}

	// Check if thumbnail file exists
	fi, err := os.Stat(photo.ThumbnailPath)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Thumbnail file missing from disk")
	}

	// Set browser caching headers
	etag := middleware.GenerateETag(fmt.Sprintf("%s-%d-%d", photo.ThumbnailPath, fi.Size(), fi.ModTime().Unix()))
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")

	if middleware.CheckETag(c, etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	return c.SendFile(photo.ThumbnailPath)
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

	fi, err := os.Stat(photo.FilePath)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Photo file missing from disk")
	}

	// Set browser caching headers
	etag := middleware.GenerateETag(fmt.Sprintf("%s-%d-%d", photo.FilePath, fi.Size(), fi.ModTime().Unix()))
	c.Set("ETag", etag)
	c.Set("Cache-Control", "public, max-age=86400")

	if middleware.CheckETag(c, etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	c.Set("Content-Type", photo.MIMEType)
	return c.SendFile(photo.FilePath)
}
