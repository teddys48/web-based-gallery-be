package repository

import (
	"fmt"
	"gallery-be/internal/model"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type FolderRepository interface {
	GetFolderTree() ([]*model.FolderNode, error)
	GetFolderContents(targetPath string, page, limit int) (*model.FolderContent, int64, error)
}

type folderRepository struct {
	db *gorm.DB
}

func NewFolderRepository(db *gorm.DB) FolderRepository {
	return &folderRepository{db: db}
}

func (r *folderRepository) GetFolderTree() ([]*model.FolderNode, error) {
	var stats []model.FolderStat
	query := `
		SELECT folder_path, COUNT(*) as count
		FROM photos
		GROUP BY folder_path
	`
	if err := r.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, err
	}

	if len(stats) == 0 {
		return []*model.FolderNode{}, nil
	}

	coverMap := r.getFolderCoversMap()
	baseRoot := findCommonRoot(stats)

	// Collect all unique folder paths and build nodes map
	nodeMap := make(map[string]*model.FolderNode)
	for _, s := range stats {
		clean := filepath.Clean(s.FolderPath)
		thumb, coverID, thumbURL := getCoverForFolder(clean, coverMap)

		if node, exists := nodeMap[clean]; exists {
			node.PhotoCount += s.Count
			if node.ThumbnailPath == "" {
				node.ThumbnailPath = thumb
				node.CoverPhotoID = coverID
				node.ThumbnailURL = thumbURL
			}
		} else {
			nodeMap[clean] = &model.FolderNode{
				Name:          filepath.Base(clean),
				Path:          clean,
				PhotoCount:    s.Count,
				ThumbnailPath: thumb,
				ThumbnailURL:  thumbURL,
				CoverPhotoID:  coverID,
				SubFolders:    []*model.FolderNode{},
			}
		}

		// Ensure intermediate parent folders exist up to baseRoot
		curr := clean
		for curr != baseRoot && curr != "." && curr != "/" {
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			if _, exists := nodeMap[parent]; !exists {
				pThumb, pCoverID, pThumbURL := getCoverForFolder(parent, coverMap)
				nodeMap[parent] = &model.FolderNode{
					Name:          filepath.Base(parent),
					Path:          parent,
					PhotoCount:    0,
					ThumbnailPath: pThumb,
					ThumbnailURL:  pThumbURL,
					CoverPhotoID:  pCoverID,
					SubFolders:    []*model.FolderNode{},
				}
			}
			curr = parent
		}
	}

	// Link parent-child relationships cleanly
	var rootNodes []*model.FolderNode
	seenRoots := make(map[string]bool)

	for path, node := range nodeMap {
		if path == baseRoot {
			if !seenRoots[path] {
				rootNodes = append(rootNodes, node)
				seenRoots[path] = true
			}
			continue
		}

		parentPath := filepath.Dir(path)
		if parentNode, exists := nodeMap[parentPath]; exists && parentPath != path {
			parentNode.SubFolders = append(parentNode.SubFolders, node)
		} else {
			if !seenRoots[path] {
				rootNodes = append(rootNodes, node)
				seenRoots[path] = true
			}
		}
	}

	// Deduplicate subfolders in each node and sort
	for _, node := range nodeMap {
		node.SubFolders = deduplicateAndSortNodes(node.SubFolders)
	}

	rootNodes = deduplicateAndSortNodes(rootNodes)
	return rootNodes, nil
}

func (r *folderRepository) GetFolderContents(targetPath string, page, limit int) (*model.FolderContent, int64, error) {
	var stats []model.FolderStat
	query := `
		SELECT folder_path, COUNT(*) as count
		FROM photos
		GROUP BY folder_path
	`
	if err := r.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, 0, err
	}

	coverMap := r.getFolderCoversMap()
	allPaths := make([]string, len(stats))
	for i, s := range stats {
		allPaths[i] = filepath.Clean(s.FolderPath)
	}

	baseRoot := findCommonRoot(stats)
	cleanTarget := resolveFolderPath(targetPath, baseRoot, allPaths)

	// Calculate parent folder path
	parentFolder := ""
	if cleanTarget != baseRoot && cleanTarget != "." && cleanTarget != "/" {
		parent := filepath.Dir(cleanTarget)
		if parent != cleanTarget {
			parentFolder = parent
		}
	}

	// Extract direct subfolders, thumbnail covers, and counts
	subFolderMap := make(map[string]*model.FolderNode)

	for _, s := range stats {
		fp := filepath.Clean(s.FolderPath)
		if fp == cleanTarget {
			continue
		}

		rel, err := filepath.Rel(cleanTarget, fp)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			continue
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) > 0 && parts[0] != "" {
			childName := parts[0]
			childPath := filepath.Join(cleanTarget, childName)

			childThumb, childCoverID, childThumbURL := getCoverForFolder(childPath, coverMap)

			if existing, ok := subFolderMap[childPath]; ok {
				existing.PhotoCount += s.Count
				if existing.ThumbnailPath == "" {
					existing.ThumbnailPath = childThumb
					existing.CoverPhotoID = childCoverID
					existing.ThumbnailURL = childThumbURL
				}
			} else {
				subFolderMap[childPath] = &model.FolderNode{
					Name:          childName,
					Path:          childPath,
					PhotoCount:    s.Count,
					ThumbnailPath: childThumb,
					ThumbnailURL:  childThumbURL,
					CoverPhotoID:  childCoverID,
				}
			}
		}
	}

	subFolders := make([]model.FolderNode, 0, len(subFolderMap))
	for _, node := range subFolderMap {
		subFolders = append(subFolders, *node)
	}

	sort.Slice(subFolders, func(i, j int) bool {
		return strings.ToLower(subFolders[i].Name) < strings.ToLower(subFolders[j].Name)
	})

	// Fetch direct photos inside cleanTarget ordered by date taken (paginated)
	var photos []model.Photo
	var totalPhotos int64

	photoQuery := r.db.Model(&model.Photo{}).Where("folder_path = ?", cleanTarget)
	if err := photoQuery.Count(&totalPhotos).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := photoQuery.Order("taken_at DESC, file_name ASC").Offset(offset).Limit(limit).Find(&photos).Error; err != nil {
		return nil, 0, err
	}

	// Group photos by date
	dateGroups := groupPhotosByDate(photos)

	content := &model.FolderContent{
		CurrentFolder: cleanTarget,
		ParentFolder:  parentFolder,
		SubFolders:    subFolders,
		DateGroups:    dateGroups,
		Photos:        photos,
	}

	return content, totalPhotos, nil
}

func groupPhotosByDate(photos []model.Photo) []model.DateGroup {
	if len(photos) == 0 {
		return []model.DateGroup{}
	}

	var groups []model.DateGroup
	groupMap := make(map[string]*model.DateGroup)
	var dateOrder []string

	for _, p := range photos {
		dateStr := "Unknown"
		if !p.TakenAt.IsZero() {
			dateStr = p.TakenAt.Format("2006-01-02")
		} else if !p.ModTime.IsZero() {
			dateStr = p.ModTime.Format("2006-01-02")
		}

		if existing, ok := groupMap[dateStr]; ok {
			existing.Photos = append(existing.Photos, p)
			existing.Count++
		} else {
			dg := &model.DateGroup{
				Date:   dateStr,
				Count:  1,
				Photos: []model.Photo{p},
			}
			groupMap[dateStr] = dg
			dateOrder = append(dateOrder, dateStr)
		}
	}

	for _, d := range dateOrder {
		groups = append(groups, *groupMap[d])
	}

	return groups
}

func (r *folderRepository) getFolderCoversMap() map[string]model.FolderCover {
	var covers []model.FolderCover
	query := `
		SELECT folder_path, thumbnail_path, id as cover_photo_id
		FROM (
			SELECT folder_path, thumbnail_path, id,
				   ROW_NUMBER() OVER (PARTITION BY folder_path ORDER BY file_name ASC, id ASC) as rn
			FROM photos
			WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''
		)
		WHERE rn = 1
	`
	_ = r.db.Raw(query).Scan(&covers).Error

	coverMap := make(map[string]model.FolderCover)
	for _, c := range covers {
		coverMap[filepath.Clean(c.FolderPath)] = c
	}
	return coverMap
}

func getCoverForFolder(folderPath string, coverMap map[string]model.FolderCover) (string, uint, string) {
	clean := filepath.Clean(folderPath)
	if c, ok := coverMap[clean]; ok && c.CoverPhotoID > 0 {
		url := fmt.Sprintf("/api/v1/photos/%d/thumbnail", c.CoverPhotoID)
		return c.ThumbnailPath, c.CoverPhotoID, url
	}

	prefix := clean + string(filepath.Separator)
	var subFolderKeys []string
	for p := range coverMap {
		if strings.HasPrefix(p, prefix) {
			subFolderKeys = append(subFolderKeys, p)
		}
	}
	sort.Strings(subFolderKeys)
	for _, p := range subFolderKeys {
		c := coverMap[p]
		if c.CoverPhotoID > 0 {
			url := fmt.Sprintf("/api/v1/photos/%d/thumbnail", c.CoverPhotoID)
			return c.ThumbnailPath, c.CoverPhotoID, url
		}
	}

	return "", 0, ""
}

func resolveFolderPath(targetPath, baseRoot string, knownPaths []string) string {
	if targetPath == "" || targetPath == "." {
		return baseRoot
	}

	cleanTarget := filepath.Clean(targetPath)

	for _, p := range knownPaths {
		if p == cleanTarget {
			return p
		}
	}

	for _, p := range knownPaths {
		if strings.HasSuffix(p, cleanTarget) || strings.HasSuffix(cleanTarget, p) {
			return p
		}
	}

	if !strings.HasPrefix(cleanTarget, baseRoot) && !filepath.IsAbs(cleanTarget) {
		joined := filepath.Join(baseRoot, cleanTarget)
		for _, p := range knownPaths {
			if p == joined || strings.HasPrefix(p, joined) {
				return joined
			}
		}
		return joined
	}

	return cleanTarget
}

func findCommonRoot(stats []model.FolderStat) string {
	if len(stats) == 0 {
		return "media"
	}

	paths := make([]string, len(stats))
	for i, s := range stats {
		paths[i] = filepath.Clean(s.FolderPath)
	}

	common := strings.Split(paths[0], string(filepath.Separator))

	for _, p := range paths[1:] {
		parts := strings.Split(p, string(filepath.Separator))
		i := 0
		for i < len(common) && i < len(parts) && common[i] == parts[i] {
			i++
		}
		common = common[:i]
	}

	result := strings.Join(common, string(filepath.Separator))
	if result == "" {
		if filepath.IsAbs(paths[0]) {
			return string(filepath.Separator)
		}
		return "."
	}
	return result
}

func deduplicateAndSortNodes(nodes []*model.FolderNode) []*model.FolderNode {
	seen := make(map[string]*model.FolderNode)
	for _, n := range nodes {
		if existing, ok := seen[n.Path]; ok {
			existing.PhotoCount += n.PhotoCount
			if existing.ThumbnailPath == "" {
				existing.ThumbnailPath = n.ThumbnailPath
				existing.CoverPhotoID = n.CoverPhotoID
				existing.ThumbnailURL = n.ThumbnailURL
			}
			if len(n.SubFolders) > 0 {
				existing.SubFolders = append(existing.SubFolders, n.SubFolders...)
			}
		} else {
			seen[n.Path] = n
		}
	}

	result := make([]*model.FolderNode, 0, len(seen))
	for _, n := range seen {
		n.SubFolders = deduplicateAndSortNodes(n.SubFolders)
		result = append(result, n)
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}
