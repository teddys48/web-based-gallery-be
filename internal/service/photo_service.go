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
	GetFolderTree() ([]*model.FolderNode, error)
	GetFolderContents(targetPath string, page, limit int) (*model.FolderContent, int64, error)
}

type photoService struct {
	photoRepo  repository.PhotoRepository
	folderRepo repository.FolderRepository
}

func NewPhotoService(photoRepo repository.PhotoRepository, folderRepo repository.FolderRepository) PhotoService {
	return &photoService{
		photoRepo:  photoRepo,
		folderRepo: folderRepo,
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
