package handler

import (
	"strconv"

	"gallery-be/internal/service"
	"gallery-be/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type FolderHandler struct {
	photoSvc service.PhotoService
}

func NewFolderHandler(photoSvc service.PhotoService) *FolderHandler {
	return &FolderHandler{photoSvc: photoSvc}
}

// GetFolderContents GET /api/v1/folders/contents?folder_path=media/Events&page=1&limit=50
// File-manager style folder view: returns current path, parent path, immediate subfolders, and direct photos
func (h *FolderHandler) GetFolderContents(c *fiber.Ctx) error {
	folderPath := c.Query("folder_path", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	contents, totalPhotos, err := h.photoSvc.GetFolderContents(folderPath, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve folder contents")
	}

	return response.SuccessWithPagination(c, contents, page, limit, totalPhotos)
}

// GetFolderTree GET /api/v1/folders/tree
func (h *FolderHandler) GetFolderTree(c *fiber.Ctx) error {
	tree, err := h.photoSvc.GetFolderTree()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve folder tree")
	}

	return response.Success(c, fiber.StatusOK, "Folder tree retrieved successfully", tree)
}

// GetFolderPhotos GET /api/v1/photos?folder_path=media/Events/Party&page=1&limit=50
func (h *FolderHandler) GetFolderPhotos(c *fiber.Ctx) error {
	folderPath := c.Query("folder_path", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	if folderPath == "" {
		return h.photoSvcGetTimelineFallback(c, page, limit)
	}

	photos, total, err := h.photoSvc.GetPhotosByFolder(folderPath, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve folder photos")
	}

	return response.SuccessWithPagination(c, photos, page, limit, total)
}

func (h *FolderHandler) photoSvcGetTimelineFallback(c *fiber.Ctx, page, limit int) error {
	photos, total, err := h.photoSvc.GetTimeline(page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve photos")
	}
	return response.SuccessWithPagination(c, photos, page, limit, total)
}
