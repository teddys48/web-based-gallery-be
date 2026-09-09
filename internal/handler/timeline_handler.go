package handler

import (
	"gallery-be/internal/model"
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

// GetPhotosByDate GET /api/v1/photos/by-date
func (h *TimelineHandler) GetPhotosByDate(c *fiber.Ctx) error {
	var filter model.DateFilter
	filter.Date = c.Query("date", "")
	filter.StartDate = c.Query("start_date", "")
	filter.EndDate = c.Query("end_date", "")
	filter.MediaType = c.Query("media_type", "")
	filter.Year, _ = strconv.Atoi(c.Query("year", "0"))
	filter.Month, _ = strconv.Atoi(c.Query("month", "0"))
	filter.Page, _ = strconv.Atoi(c.Query("page", "1"))
	filter.Limit, _ = strconv.Atoi(c.Query("limit", "50"))

	photos, total, err := h.photoSvc.GetPhotosByDate(filter)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve photos by date")
	}

	return response.SuccessWithPagination(c, photos, filter.Page, filter.Limit, total)
}

// GetPhotosByDateGrouped GET /api/v1/photos/by-date/grouped
func (h *TimelineHandler) GetPhotosByDateGrouped(c *fiber.Ctx) error {
	var filter model.DateFilter
	filter.Date = c.Query("date", "")
	filter.StartDate = c.Query("start_date", "")
	filter.EndDate = c.Query("end_date", "")
	filter.MediaType = c.Query("media_type", "")
	filter.Year, _ = strconv.Atoi(c.Query("year", "0"))
	filter.Month, _ = strconv.Atoi(c.Query("month", "0"))

	groups, err := h.photoSvc.GetPhotosByDateGrouped(filter)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve grouped photos by date")
	}

	return response.Success(c, fiber.StatusOK, "Grouped photos retrieved successfully", groups)
}
