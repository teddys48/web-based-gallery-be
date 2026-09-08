package handler

import (
	"gallery-be/internal/service"
	"gallery-be/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type ScanHandler struct {
	scannerSvc service.ScannerService
}

func NewScanHandler(scannerSvc service.ScannerService) *ScanHandler {
	return &ScanHandler{scannerSvc: scannerSvc}
}

// TriggerScan POST /api/v1/scan/start
func (h *ScanHandler) TriggerScan(c *fiber.Ctx) error {
	status, err := h.scannerSvc.StartScan()
	if err != nil {
		return response.Error(c, fiber.StatusConflict, err.Error())
	}

	return response.Success(c, fiber.StatusAccepted, "Scan initiated in background", status)
}

// GetScanStatus GET /api/v1/scan/status
func (h *ScanHandler) GetScanStatus(c *fiber.Ctx) error {
	status := h.scannerSvc.GetStatus()
	return response.Success(c, fiber.StatusOK, "Scan status retrieved successfully", status)
}
