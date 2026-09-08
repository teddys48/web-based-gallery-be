package handler

import (
	"gallery-be/internal/service"
	"gallery-be/pkg/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type TimelineHandler struct {
	photoSvc service.PhotoService
}

func NewTimelineHandler(photoSvc service.PhotoService) *TimelineHandler {
	return &TimelineHandler{photoSvc: photoSvc}
}

// GetTimeline GET /api/v1/photos/timeline?page=1&limit=50
func (h *TimelineHandler) GetTimeline(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	photos, total, err := h.photoSvc.GetTimeline(page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve timeline photos")
	}

	return response.SuccessWithPagination(c, photos, page, limit, total)
}

// GetTimelineBuckets GET /api/v1/photos/timeline/buckets
func (h *TimelineHandler) GetTimelineBuckets(c *fiber.Ctx) error {
	buckets, err := h.photoSvc.GetTimelineBuckets()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve timeline buckets")
	}

	return response.Success(c, fiber.StatusOK, "Timeline buckets retrieved successfully", buckets)
}
