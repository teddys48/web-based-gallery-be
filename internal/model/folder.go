package model

type DateGroup struct {
	Date   string  `json:"date"`
	Count  int     `json:"count"`
	Photos []Photo `json:"photos"`
}

type FolderNode struct {
	Name          string        `json:"name"`
	Path          string        `json:"path"`
	PhotoCount    int64         `json:"photo_count"`
	ThumbnailURL  string        `json:"thumbnail_url,omitempty"`
	ThumbnailPath string        `json:"thumbnail_path,omitempty"`
	CoverPhotoID  uint          `json:"cover_photo_id,omitempty"`
	SubFolders    []*FolderNode `json:"sub_folders,omitempty"`
}

type FolderStat struct {
	FolderPath string `json:"folder_path"`
	Count      int64  `json:"count"`
}

type FolderCover struct {
	FolderPath    string `json:"folder_path"`
	ThumbnailPath string `json:"thumbnail_path"`
	CoverPhotoID  uint   `json:"cover_photo_id"`
}

type FolderContent struct {
	CurrentFolder string       `json:"current_folder"`
	ParentFolder  string       `json:"parent_folder,omitempty"`
	SubFolders    []FolderNode `json:"sub_folders"`
	DateGroups    []DateGroup  `json:"date_groups"`
	Photos        []Photo      `json:"photos"`
}
