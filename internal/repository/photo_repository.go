package repository

import (
	"time"

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
	FindByDatePaginated(filter model.DateFilter) ([]model.Photo, int64, error)
	GetPhotosByDateGrouped(filter model.DateFilter) ([]model.PhotosByDateGroup, error)
	GetDistinctFolderPaths() ([]string, error)
	DeleteMissingPaths(existingPaths []string) (int64, error)
	GetAllFilePathsMap() (map[string]model.Photo, error)
	UpdateThumbnailPathByID(id uint, thumbPath string) error
}

type photoRepository struct {
	db *gorm.DB
}

func NewPhotoRepository(db *gorm.DB) PhotoRepository {
	return &photoRepository{db: db}
}

func (r *photoRepository) Upsert(photo *model.Photo) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "file_path"}},
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

func buildDateQuery(db *gorm.DB, filter model.DateFilter) *gorm.DB {
	query := db.Model(&model.Photo{})

	if filter.MediaType != "" {
		query = query.Where("media_type = ?", filter.MediaType)
	}

	// 1. Specific Date (YYYY-MM-DD or YYYY-MM)
	if filter.Date != "" {
		if len(filter.Date) == 10 { // YYYY-MM-DD
			if startTime, err := time.Parse("2006-01-02", filter.Date); err == nil {
				endTime := startTime.Add(24*time.Hour - time.Nanosecond)
				return query.Where("taken_at >= ? AND taken_at <= ?", startTime, endTime)
			}
		} else if len(filter.Date) == 7 { // YYYY-MM
			if startTime, err := time.Parse("2006-01", filter.Date); err == nil {
				endTime := startTime.AddDate(0, 1, 0).Add(-time.Nanosecond)
				return query.Where("taken_at >= ? AND taken_at <= ?", startTime, endTime)
			}
		}
	}

	// 2. Date Range (start_date & end_date)
	if filter.StartDate != "" {
		if startTime, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			query = query.Where("taken_at >= ?", startTime)
		}
	}
	if filter.EndDate != "" {
		if endTime, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			endTime = endTime.Add(24*time.Hour - time.Nanosecond)
			query = query.Where("taken_at <= ?", endTime)
		}
	}

	// 3. Year & Month parameters
	if filter.Year > 0 {
		if filter.Month >= 1 && filter.Month <= 12 {
			startTime := time.Date(filter.Year, time.Month(filter.Month), 1, 0, 0, 0, 0, time.UTC)
			endTime := startTime.AddDate(0, 1, 0).Add(-time.Nanosecond)
			query = query.Where("taken_at >= ? AND taken_at <= ?", startTime, endTime)
		} else {
			startTime := time.Date(filter.Year, 1, 1, 0, 0, 0, 0, time.UTC)
			endTime := startTime.AddDate(1, 0, 0).Add(-time.Nanosecond)
			query = query.Where("taken_at >= ? AND taken_at <= ?", startTime, endTime)
		}
	}

	return query
}

func (r *photoRepository) FindByDatePaginated(filter model.DateFilter) ([]model.Photo, int64, error) {
	var photos []model.Photo
	var total int64

	query := buildDateQuery(r.db, filter)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 50
	}

	offset := (filter.Page - 1) * filter.Limit
	err := query.Order("taken_at DESC, id DESC").Offset(offset).Limit(filter.Limit).Find(&photos).Error
	return photos, total, err
}

func (r *photoRepository) GetPhotosByDateGrouped(filter model.DateFilter) ([]model.PhotosByDateGroup, error) {
	var photos []model.Photo
	query := buildDateQuery(r.db, filter)
	if err := query.Order("taken_at DESC, id DESC").Find(&photos).Error; err != nil {
		return nil, err
	}

	groupMap := make(map[string][]model.Photo)
	var dateOrder []string

	for _, p := range photos {
		dateStr := p.TakenAt.Format("2006-01-02")
		if _, exists := groupMap[dateStr]; !exists {
			dateOrder = append(dateOrder, dateStr)
		}
		groupMap[dateStr] = append(groupMap[dateStr], p)
	}

	result := make([]model.PhotosByDateGroup, 0, len(dateOrder))
	for _, d := range dateOrder {
		groupPhotos := groupMap[d]
		result = append(result, model.PhotosByDateGroup{
			Date:   d,
			Count:  int64(len(groupPhotos)),
			Photos: groupPhotos,
		})
	}

	return result, nil
}

func (r *photoRepository) GetDistinctFolderPaths() ([]string, error) {
	var paths []string
	err := r.db.Model(&model.Photo{}).Distinct().Pluck("folder_path", &paths).Error
	return paths, err
}

func (r *photoRepository) GetAllFilePathsMap() (map[string]model.Photo, error) {
	var photos []model.Photo
	if err := r.db.Select("id, file_path, mod_time, file_size, hash, thumbnail_path").Find(&photos).Error; err != nil {
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

func (r *photoRepository) UpdateThumbnailPathByID(id uint, thumbPath string) error {
	return r.db.Model(&model.Photo{}).Where("id = ?", id).Update("thumbnail_path", thumbPath).Error
}
