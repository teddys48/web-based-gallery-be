package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gallery-be/config"
	"gallery-be/internal/service"
	"gallery-be/pkg/response"
	"gallery-be/pkg/util"

	"github.com/gofiber/fiber/v2"
)

type FolderHandler struct {
	photoSvc service.PhotoService
	cfg      *config.Config
}

func NewFolderHandler(photoSvc service.PhotoService, cfg *config.Config) *FolderHandler {
	return &FolderHandler{
		photoSvc: photoSvc,
		cfg:      cfg,
	}
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

// DownloadFolderZip GET /api/v1/folders/download?path=media/Vacation2025
// Packages a folder and all its subdirectories into a .zip streamed directly to the HTTP response
func (h *FolderHandler) DownloadFolderZip(c *fiber.Ctx) error {
	folderPath := c.Query("path", "")
	if folderPath == "" {
		folderPath = c.Query("folder_path", "")
	}

	// 1. Normalize input path
	normPath := util.NormalizeMediaPath(h.cfg.MediaDir, folderPath)
	physicalPath := util.ResolvePhysicalPath(h.cfg.MediaDir, normPath)

	// 2. Security Validation: Path Traversal & Symlink Escape Protection
	mediaDirAbs, err := filepath.Abs(h.cfg.MediaDir)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to resolve media directory path")
	}
	mediaDirCanonical, err := filepath.EvalSymlinks(mediaDirAbs)
	if err != nil {
		mediaDirCanonical = mediaDirAbs
	}

	targetAbs, err := filepath.Abs(physicalPath)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid folder path")
	}

	targetCanonical, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return response.Error(c, fiber.StatusNotFound, "Folder not found")
		}
		targetCanonical = targetAbs
	}

	// Verify target path remains inside MEDIA_DIR boundary
	mediaDirClean := filepath.Clean(mediaDirCanonical)
	targetClean := filepath.Clean(targetCanonical)

	if targetClean != mediaDirClean && !strings.HasPrefix(targetClean, mediaDirClean+string(filepath.Separator)) {
		return response.Error(c, fiber.StatusForbidden, "Path traversal detected: target path is outside media directory")
	}

	// 3. Verify target path is a valid directory
	fi, err := os.Stat(targetClean)
	if err != nil || !fi.IsDir() {
		return response.Error(c, fiber.StatusNotFound, "Folder not found")
	}

	// 4. Gather readable files recursively
	type zipEntryInfo struct {
		absPath      string
		relPathInZip string
		info         os.FileInfo
	}
	var filesToZip []zipEntryInfo

	err = filepath.WalkDir(targetClean, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		baseName := d.Name()
		if strings.HasPrefix(baseName, ".") || strings.HasSuffix(baseName, ".tmp") || strings.HasSuffix(baseName, ".tmp.jpg") {
			if d.IsDir() && baseName != "." {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() == 0 {
			return nil
		}

		// Calculate relative path inside the zip file
		rel, err := filepath.Rel(targetClean, path)
		if err != nil {
			return nil
		}

		relPathInZip := filepath.ToSlash(rel)
		filesToZip = append(filesToZip, zipEntryInfo{
			absPath:      path,
			relPathInZip: relPathInZip,
			info:         info,
		})

		return nil
	})

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to scan folder contents for compression")
	}

	// 5. Handle empty folder or folder with no readable files
	if len(filesToZip) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "Folder is empty or contains no readable files")
	}

	// 6. Format ZIP filename
	zipBaseName := filepath.Base(targetClean)
	if targetClean == mediaDirClean || zipBaseName == "." || zipBaseName == "/" {
		zipBaseName = "media"
	}
	zipFileName := fmt.Sprintf("%s.zip", zipBaseName)

	// 7. Set HTTP response headers
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipFileName))

	// 8. Stream ZIP file directly to Fiber HTTP response without temp file or RAM buffering
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		zipWriter := zip.NewWriter(pw)
		defer zipWriter.Close()

		for _, item := range filesToZip {
			header, err := zip.FileInfoHeader(item.info)
			if err != nil {
				continue
			}
			header.Name = item.relPathInZip
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				continue
			}

			srcFile, err := os.Open(item.absPath)
			if err != nil {
				log.Printf("[ZIP DOWNLOAD] Skipping unreadable file %s: %v", item.absPath, err)
				continue
			}

			_, _ = io.Copy(writer, srcFile)
			_ = srcFile.Close()
		}
	}()

	return c.SendStream(pr)
}
