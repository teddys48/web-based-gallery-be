package repository_test

import (
	"testing"
	"time"

	"gallery-be/config"
	"gallery-be/internal/database"
	"gallery-be/internal/model"
	"gallery-be/internal/repository"
)

func TestFolderRepositoryDeduplicationAndPathResolution(t *testing.T) {
	cfg := &config.Config{
		DBPath: "../../gallery_folder_test.db",
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	repo := repository.NewPhotoRepository(db)
	folderRepo := repository.NewFolderRepository(db)

	// Insert photos in /home/teddy/Pictures and /home/teddy/Pictures/test
	photos := []model.Photo{
		{
			FilePath:   "/home/teddy/Pictures/pic1.png",
			FileName:   "pic1.png",
			FolderPath: "/home/teddy/Pictures",
			FileSize:   100,
			TakenAt:    time.Now(),
		},
		{
			FilePath:   "/home/teddy/Pictures/pic2.png",
			FileName:   "pic2.png",
			FolderPath: "/home/teddy/Pictures",
			FileSize:   100,
			TakenAt:    time.Now(),
		},
		{
			FilePath:   "/home/teddy/Pictures/test/test1.png",
			FileName:   "test1.png",
			FolderPath: "/home/teddy/Pictures/test",
			FileSize:   100,
			TakenAt:    time.Now(),
		},
	}

	for i := range photos {
		if err := repo.Upsert(&photos[i]); err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	}

	// 1. Test GetFolderTree deduplication
	tree, err := folderRepo.GetFolderTree()
	if err != nil {
		t.Fatalf("GetFolderTree failed: %v", err)
	}
	if len(tree) != 1 {
		t.Errorf("Expected 1 root node (Pictures), got %d", len(tree))
	}
	rootNode := tree[0]
	if rootNode.Name != "Pictures" {
		t.Errorf("Expected root node name Pictures, got %s", rootNode.Name)
	}
	if len(rootNode.SubFolders) != 1 {
		t.Errorf("Expected 1 subfolder (test), got %d", len(rootNode.SubFolders))
	}
	if rootNode.SubFolders[0].Name != "test" {
		t.Errorf("Expected subfolder name test, got %s", rootNode.SubFolders[0].Name)
	}

	// 2. Test GetFolderContents for root Pictures (empty string)
	contents, total, err := folderRepo.GetFolderContents("", 1, 50)
	if err != nil {
		t.Fatalf("GetFolderContents empty failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 photos in Pictures, got %d", total)
	}
	if len(contents.Photos) != 2 {
		t.Errorf("Expected 2 photos returned in Pictures, got %d", len(contents.Photos))
	}
	if len(contents.SubFolders) != 1 || contents.SubFolders[0].Name != "test" {
		t.Errorf("Expected 1 subfolder (test) in Pictures, got %v", contents.SubFolders)
	}

	// 3. Test GetFolderContents for subfolder "test" or "Pictures/test"
	contentsTest, totalTest, err := folderRepo.GetFolderContents("test", 1, 50)
	if err != nil {
		t.Fatalf("GetFolderContents test failed: %v", err)
	}
	if totalTest != 1 {
		t.Errorf("Expected 1 photo in test folder, got %d", totalTest)
	}
	if len(contentsTest.Photos) != 1 {
		t.Errorf("Expected 1 photo returned in test folder, got %d", len(contentsTest.Photos))
	}
}
