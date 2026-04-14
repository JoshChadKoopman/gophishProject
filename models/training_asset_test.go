package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupAssetTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM training_assets")
	db.Exec("DELETE FROM training_presentations")
	return func() {
		db.Exec("DELETE FROM training_assets")
		db.Exec("DELETE FROM training_presentations")
	}
}

func makeTestPresentation(t *testing.T) TrainingPresentation {
	t.Helper()
	tp := &TrainingPresentation{
		OrgId:    1,
		Name:     "Custom Course",
		FileName: "course.pdf",
		FilePath: "/uploads/course.pdf",
		FileSize: 1024,
	}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("PostTrainingPresentation: %v", err)
	}
	return *tp
}

func TestClassifyAssetType(t *testing.T) {
	cases := map[string]string{
		"application/pdf":                 AssetTypePDF,
		"video/mp4":                       AssetTypeVideo,
		"video/webm":                      AssetTypeVideo,
		"image/jpeg":                      AssetTypeImage,
		"image/png":                       AssetTypeImage,
		"application/vnd.ms-powerpoint":   AssetTypePPTX,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": AssetTypePPTX,
		"application/octet-stream":        AssetTypeDocument,
		"text/plain":                      AssetTypeOther,
	}
	for ct, want := range cases {
		got := ClassifyAssetType(ct)
		if got != want {
			t.Errorf("ClassifyAssetType(%q) = %q, want %q", ct, got, want)
		}
	}
}

func TestPostAndGetTrainingAsset(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope := OrgScope{OrgId: 1}

	asset := &TrainingAsset{
		PresentationId: tp.Id,
		OrgId:          1,
		Title:          "Module 1: Intro",
		FileName:       "intro.pdf",
		FilePath:       "/uploads/intro.pdf",
		FileSize:       2048,
		ContentType:    "application/pdf",
		UploadedBy:     1,
	}
	if err := PostTrainingAsset(asset); err != nil {
		t.Fatalf("PostTrainingAsset: %v", err)
	}
	if asset.Id == 0 {
		t.Fatal("expected asset to have an ID")
	}
	if asset.AssetType != AssetTypePDF {
		t.Fatalf("expected asset_type %q, got %q", AssetTypePDF, asset.AssetType)
	}
	if asset.SortOrder != 1 {
		t.Fatalf("expected sort_order 1, got %d", asset.SortOrder)
	}

	// Fetch it back
	got, err := GetTrainingAsset(asset.Id, scope)
	if err != nil {
		t.Fatalf("GetTrainingAsset: %v", err)
	}
	if got.Title != "Module 1: Intro" {
		t.Fatalf("expected title %q, got %q", "Module 1: Intro", got.Title)
	}
}

func TestGetTrainingAssetsOrdering(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope := OrgScope{OrgId: 1}

	for i, title := range []string{"Module A", "Module B", "Module C"} {
		a := &TrainingAsset{
			PresentationId: tp.Id,
			OrgId:          1,
			Title:          title,
			FileName:       "file.pdf",
			FilePath:       "/uploads/file.pdf",
			FileSize:       100,
			ContentType:    "application/pdf",
			SortOrder:      i + 1,
			UploadedBy:     1,
		}
		if err := PostTrainingAsset(a); err != nil {
			t.Fatalf("PostTrainingAsset(%s): %v", title, err)
		}
	}

	assets, err := GetTrainingAssets(tp.Id, scope)
	if err != nil {
		t.Fatalf("GetTrainingAssets: %v", err)
	}
	if len(assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(assets))
	}
	if assets[0].Title != "Module A" || assets[2].Title != "Module C" {
		t.Fatalf("assets not in sort order: %v", assets)
	}
}

func TestReorderTrainingAssets(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope := OrgScope{OrgId: 1}

	ids := make([]int64, 3)
	for i := range ids {
		a := &TrainingAsset{
			PresentationId: tp.Id,
			OrgId:          1,
			Title:          string(rune('A' + i)),
			FileName:       "f.pdf",
			FilePath:       "/uploads/f.pdf",
			FileSize:       100,
			ContentType:    "application/pdf",
			SortOrder:      i + 1,
			UploadedBy:     1,
		}
		if err := PostTrainingAsset(a); err != nil {
			t.Fatalf("PostTrainingAsset: %v", err)
		}
		ids[i] = a.Id
	}

	// Reverse the order: C, B, A
	reversed := []int64{ids[2], ids[1], ids[0]}
	if err := ReorderTrainingAssets(tp.Id, reversed, scope); err != nil {
		t.Fatalf("ReorderTrainingAssets: %v", err)
	}

	assets, _ := GetTrainingAssets(tp.Id, scope)
	if assets[0].Id != ids[2] {
		t.Fatalf("expected first asset to be id %d, got %d", ids[2], assets[0].Id)
	}
	if assets[2].Id != ids[0] {
		t.Fatalf("expected last asset to be id %d, got %d", ids[0], assets[2].Id)
	}
}

func TestDeleteTrainingAsset(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope := OrgScope{OrgId: 1}

	asset := &TrainingAsset{
		PresentationId: tp.Id,
		OrgId:          1,
		Title:          "To Delete",
		FileName:       "delete.pdf",
		FilePath:       "/uploads/delete.pdf",
		FileSize:       100,
		ContentType:    "application/pdf",
		UploadedBy:     1,
	}
	PostTrainingAsset(asset)

	if err := DeleteTrainingAsset(asset.Id, scope); err != nil {
		t.Fatalf("DeleteTrainingAsset: %v", err)
	}
	if _, err := GetTrainingAsset(asset.Id, scope); err == nil {
		t.Fatal("expected asset to be gone after delete")
	}
}

func TestDeleteTrainingAssetsByPresentation(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope := OrgScope{OrgId: 1}

	for i := 0; i < 3; i++ {
		a := &TrainingAsset{
			PresentationId: tp.Id,
			OrgId:          1,
			Title:          "Bulk",
			FileName:       "b.pdf",
			FilePath:       "/uploads/b.pdf",
			FileSize:       100,
			ContentType:    "application/pdf",
			UploadedBy:     1,
		}
		PostTrainingAsset(a)
	}

	deleted, err := DeleteTrainingAssetsByPresentation(tp.Id)
	if err != nil {
		t.Fatalf("DeleteTrainingAssetsByPresentation: %v", err)
	}
	if len(deleted) != 3 {
		t.Fatalf("expected 3 deleted assets, got %d", len(deleted))
	}

	remaining, _ := GetTrainingAssets(tp.Id, scope)
	if len(remaining) != 0 {
		t.Fatalf("expected 0 assets remaining, got %d", len(remaining))
	}
}

func TestAutoSortOrderIncrement(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)

	for i := 0; i < 3; i++ {
		a := &TrainingAsset{
			PresentationId: tp.Id,
			OrgId:          1,
			Title:          "Auto",
			FileName:       "a.pdf",
			FilePath:       "/uploads/a.pdf",
			FileSize:       100,
			ContentType:    "application/pdf",
			UploadedBy:     1,
			// SortOrder left at 0 — should auto-increment
		}
		if err := PostTrainingAsset(a); err != nil {
			t.Fatalf("PostTrainingAsset: %v", err)
		}
		if a.SortOrder != i+1 {
			t.Fatalf("expected auto sort_order %d, got %d", i+1, a.SortOrder)
		}
	}
}

func TestOrgScopeIsolation(t *testing.T) {
	teardown := setupAssetTest(t)
	defer teardown()

	tp := makeTestPresentation(t)
	scope2 := OrgScope{OrgId: 99}

	asset := &TrainingAsset{
		PresentationId: tp.Id,
		OrgId:          1,
		Title:          "Org 1 only",
		FileName:       "o.pdf",
		FilePath:       "/uploads/o.pdf",
		FileSize:       100,
		ContentType:    "application/pdf",
		UploadedBy:     1,
	}
	PostTrainingAsset(asset)

	// Org 99 should not see this asset.
	assets, _ := GetTrainingAssets(tp.Id, scope2)
	if len(assets) != 0 {
		t.Fatalf("org 99 should not see org 1's assets, got %d", len(assets))
	}
	if _, err := GetTrainingAsset(asset.Id, scope2); err == nil {
		t.Fatal("org 99 should not be able to fetch org 1's asset")
	}
}
