package repository

import (
	"gallery-be/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PhotoRepository interface {
	Upsert(photo *model.Photo) error
	FindByID(id uint) (*model.Photo, error)
	FindByFilePath(path string) (*model.Photo, error)
	FindTimelinePaginated(page, limit int) ([]model.Photo, int64, error)
	GetTimelineBuckets() ([]model.TimelineBucket, error)
	FindByFolderPaginated(folderPath string, page, limit int) ([]model.Photo, int64, error)
	GetDistinctFolderPaths() ([]string, error)
	DeleteMissingPaths(existingPaths []string) (int64, error)
	GetAllFilePathsMap() (map[string]model.Photo, error)
}

type photoRepository struct {
	db *gorm.DB
}

func NewPhotoRepository(db *gorm.DB) PhotoRepository {
	return &photoRepository{db: db}
}

func (r *photoRepository) Upsert(photo *model.Photo) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "file_path"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"file_name", "folder_path", "file_size", "hash", "media_type", "mime_type",
			"width", "height", "duration", "taken_at", "camera_make", "camera_model",
			"f_number", "exposure_time", "iso", "focal_length", "latitude",
			"longitude", "thumbnail_path", "mod_time", "updated_at",
		}),
	}).Create(photo).Error
}

func (r *photoRepository) FindByID(id uint) (*model.Photo, error) {
	var photo model.Photo
	err := r.db.First(&photo, id).Error
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *photoRepository) FindByFilePath(path string) (*model.Photo, error) {
	var photo model.Photo
	err := r.db.Where("file_path = ?", path).First(&photo).Error
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *photoRepository) FindTimelinePaginated(page, limit int) ([]model.Photo, int64, error) {
	var photos []model.Photo
	var total int64

	if err := r.db.Model(&model.Photo{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := r.db.Order("taken_at DESC, id DESC").Offset(offset).Limit(limit).Find(&photos).Error
	return photos, total, err
}

func (r *photoRepository) GetTimelineBuckets() ([]model.TimelineBucket, error) {
	var buckets []model.TimelineBucket
	// SQLite strftime for Year and Month
	query := `
		SELECT 
			CAST(strftime('%Y', taken_at) AS INTEGER) as year,
			CAST(strftime('%m', taken_at) AS INTEGER) as month,
			COUNT(*) as count
		FROM photos
		WHERE taken_at IS NOT NULL AND strftime('%Y', taken_at) IS NOT NULL
		GROUP BY year, month
		ORDER BY year DESC, month DESC
	`
	err := r.db.Raw(query).Scan(&buckets).Error
	return buckets, err
}

func (r *photoRepository) FindByFolderPaginated(folderPath string, page, limit int) ([]model.Photo, int64, error) {
	var photos []model.Photo
	var total int64

	query := r.db.Model(&model.Photo{}).Where("folder_path = ?", folderPath)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Order("file_name ASC").Offset(offset).Limit(limit).Find(&photos).Error
	return photos, total, err
}

func (r *photoRepository) GetDistinctFolderPaths() ([]string, error) {
	var paths []string
	err := r.db.Model(&model.Photo{}).Distinct().Pluck("folder_path", &paths).Error
	return paths, err
}

func (r *photoRepository) GetAllFilePathsMap() (map[string]model.Photo, error) {
	var photos []model.Photo
	if err := r.db.Select("id, file_path, mod_time, file_size, hash").Find(&photos).Error; err != nil {
		return nil, err
	}

	result := make(map[string]model.Photo, len(photos))
	for _, p := range photos {
		result[p.FilePath] = p
	}
	return result, nil
}

func (r *photoRepository) DeleteMissingPaths(existingPaths []string) (int64, error) {
	if len(existingPaths) == 0 {
		res := r.db.Where("1 = 1").Delete(&model.Photo{})
		return res.RowsAffected, res.Error
	}

	res := r.db.Where("file_path NOT IN ?", existingPaths).Delete(&model.Photo{})
	return res.RowsAffected, res.Error
}
