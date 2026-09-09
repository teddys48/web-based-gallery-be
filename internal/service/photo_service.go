package service

import (
	"gallery-be/internal/model"
	"gallery-be/internal/repository"
)

type PhotoService interface {
	GetPhotoByID(id uint) (*model.Photo, error)
	GetTimeline(page, limit int) ([]model.Photo, int64, error)
	GetTimelineBuckets() ([]model.TimelineBucket, error)
	GetPhotosByFolder(folderPath string, page, limit int) ([]model.Photo, int64, error)
	GetPhotosByDate(filter model.DateFilter) ([]model.Photo, int64, error)
	GetPhotosByDateGrouped(filter model.DateFilter) ([]model.PhotosByDateGroup, error)
	GetFolderTree() ([]*model.FolderNode, error)
	GetFolderContents(targetPath string, page, limit int) (*model.FolderContent, int64, error)
	GetPhotoThumbnailByID(id uint) (string, *model.Photo, error)
	GetPhotoThumbnailByPath(filePath string) (string, error)
}

type photoService struct {
	photoRepo  repository.PhotoRepository
	folderRepo repository.FolderRepository
	thumbSvc   ThumbnailService
}

func NewPhotoService(photoRepo repository.PhotoRepository, folderRepo repository.FolderRepository, thumbSvc ThumbnailService) PhotoService {
	return &photoService{
		photoRepo:  photoRepo,
		folderRepo: folderRepo,
		thumbSvc:   thumbSvc,
	}
}

func (s *photoService) GetPhotoByID(id uint) (*model.Photo, error) {
	return s.photoRepo.FindByID(id)
}

func (s *photoService) GetTimeline(page, limit int) ([]model.Photo, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	return s.photoRepo.FindTimelinePaginated(page, limit)
}

func (s *photoService) GetTimelineBuckets() ([]model.TimelineBucket, error) {
	return s.photoRepo.GetTimelineBuckets()
}

func (s *photoService) GetPhotosByFolder(folderPath string, page, limit int) ([]model.Photo, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	return s.photoRepo.FindByFolderPaginated(folderPath, page, limit)
}

func (s *photoService) GetPhotosByDate(filter model.DateFilter) ([]model.Photo, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return s.photoRepo.FindByDatePaginated(filter)
}

func (s *photoService) GetPhotosByDateGrouped(filter model.DateFilter) ([]model.PhotosByDateGroup, error) {
	return s.photoRepo.GetPhotosByDateGrouped(filter)
}

func (s *photoService) GetFolderTree() ([]*model.FolderNode, error) {
	return s.folderRepo.GetFolderTree()
}

func (s *photoService) GetFolderContents(targetPath string, page, limit int) (*model.FolderContent, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	return s.folderRepo.GetFolderContents(targetPath, page, limit)
}

func (s *photoService) GetPhotoThumbnailByID(id uint) (string, *model.Photo, error) {
	photo, err := s.photoRepo.FindByID(id)
	if err != nil {
		return "", nil, err
	}

	// 1. If stored thumbnail is valid and not a fallback placeholder, return it directly
	if photo.ThumbnailPath != "" && isValidThumbnail(photo.ThumbnailPath) && !isFallbackPlaceholder(photo.ThumbnailPath) {
		return photo.ThumbnailPath, photo, nil
	}

	// Check if expected thumbnail path in current ThumbnailDir exists
	expectedThumbPath := s.thumbSvc.GetThumbnailPath(photo.FilePath)
	if isValidThumbnail(expectedThumbPath) && !isFallbackPlaceholder(expectedThumbPath) {
		if photo.ThumbnailPath != expectedThumbPath {
			photo.ThumbnailPath = expectedThumbPath
			_ = s.photoRepo.UpdateThumbnailPathByID(id, expectedThumbPath)
		}
		return expectedThumbPath, photo, nil
	}

	// 2. Generate on-demand if missing or placeholder
	thumbPath, err := s.thumbSvc.GetOrGenerateThumbnail(photo.FilePath, photo.MediaType)
	if err != nil {
		return "", photo, err
	}

	// 3. Update ONLY the thumbnail_path column for this photo ID
	if photo.ThumbnailPath != thumbPath {
		photo.ThumbnailPath = thumbPath
		_ = s.photoRepo.UpdateThumbnailPathByID(id, thumbPath)
	}

	return thumbPath, photo, nil
}

func (s *photoService) GetPhotoThumbnailByPath(filePath string) (string, error) {
	return s.thumbSvc.GetOrGenerateThumbnail(filePath, "")
}
