package models

import (
	"errors"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Asset type constants. Authors upload arbitrary PDF/PPTX/video/image files
// to build a multi-module custom training course.
const (
	AssetTypePDF      = "pdf"
	AssetTypePPTX     = "pptx"
	AssetTypeVideo    = "video"
	AssetTypeImage    = "image"
	AssetTypeDocument = "document"
	AssetTypeOther    = "other"
)

// TrainingAsset represents one module (file) inside a custom training
// presentation. A TrainingPresentation can have many assets; the set of
// assets forms the ordered modules of a "custom training builder" course.
type TrainingAsset struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	PresentationId int64     `json:"presentation_id" gorm:"column:presentation_id"`
	OrgId          int64     `json:"-" gorm:"column:org_id"`
	Title          string    `json:"title" gorm:"column:title"`
	Description    string    `json:"description" gorm:"column:description;type:text"`
	FileName       string    `json:"file_name" gorm:"column:file_name"`
	FilePath       string    `json:"-" gorm:"column:file_path"`
	FileSize       int64     `json:"file_size" gorm:"column:file_size"`
	ContentType    string    `json:"content_type" gorm:"column:content_type"`
	AssetType      string    `json:"asset_type" gorm:"column:asset_type"`
	SortOrder      int       `json:"sort_order" gorm:"column:sort_order"`
	UploadedBy     int64     `json:"uploaded_by" gorm:"column:uploaded_by"`
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName ensures GORM uses the correct table name.
func (TrainingAsset) TableName() string { return "training_assets" }

// ErrTrainingAssetNotFound is returned when an asset lookup misses.
var ErrTrainingAssetNotFound = errors.New("Training asset not found")

// ClassifyAssetType maps a file's MIME type to a coarse asset_type category
// so the frontend can render an appropriate preview (PDF viewer, video
// player, etc.) without re-sniffing the blob.
func ClassifyAssetType(contentType string) string {
	ct := strings.ToLower(contentType)
	switch {
	case strings.HasPrefix(ct, "video/"):
		return AssetTypeVideo
	case strings.HasPrefix(ct, "image/"):
		return AssetTypeImage
	case ct == "application/pdf":
		return AssetTypePDF
	case strings.Contains(ct, "presentation") || strings.Contains(ct, "powerpoint"):
		return AssetTypePPTX
	case strings.HasPrefix(ct, "application/"):
		return AssetTypeDocument
	default:
		return AssetTypeOther
	}
}

// GetTrainingAssets returns all assets for a presentation, ordered by sort_order.
func GetTrainingAssets(presentationId int64, scope OrgScope) ([]TrainingAsset, error) {
	assets := []TrainingAsset{}
	query := scopeQuery(db.Where(queryWherePresentationID, presentationId), scope)
	err := query.Order("sort_order asc, id asc").Find(&assets).Error
	return assets, err
}

// GetTrainingAsset returns a single asset by id, scoped to the caller's org.
func GetTrainingAsset(id int64, scope OrgScope) (TrainingAsset, error) {
	a := TrainingAsset{}
	err := scopeQuery(db.Where("id=?", id), scope).First(&a).Error
	if err != nil {
		return a, ErrTrainingAssetNotFound
	}
	return a, nil
}

// PostTrainingAsset creates a new asset record. The caller is responsible
// for having already written the file to disk and populating FilePath.
func PostTrainingAsset(a *TrainingAsset) error {
	a.CreatedDate = time.Now().UTC()
	if a.AssetType == "" {
		a.AssetType = ClassifyAssetType(a.ContentType)
	}
	if a.SortOrder == 0 {
		// Default new assets to the end of the current module list.
		var max struct {
			Max int
		}
		db.Table("training_assets").
			Select("COALESCE(MAX(sort_order), 0) as max").
			Where(queryWherePresentationID, a.PresentationId).
			Scan(&max)
		a.SortOrder = max.Max + 1
	}
	err := db.Save(a).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutTrainingAsset updates metadata on an existing asset (title/description/order).
// The underlying file blob is immutable — callers who need to change the
// file should delete + re-upload to keep file path handling simple.
func PutTrainingAsset(a *TrainingAsset) error {
	return db.Save(a).Error
}

// DeleteTrainingAsset removes an asset row by id. The on-disk file is
// removed by the controller layer since the model package has no
// knowledge of the upload directory.
func DeleteTrainingAsset(id int64, scope OrgScope) error {
	a, err := GetTrainingAsset(id, scope)
	if err != nil {
		return err
	}
	return db.Where("id=?", a.Id).Delete(&TrainingAsset{}).Error
}

// DeleteTrainingAssetsByPresentation removes all assets attached to a
// presentation. Used by DeleteTrainingPresentation's cascade.
func DeleteTrainingAssetsByPresentation(presentationId int64) ([]TrainingAsset, error) {
	assets := []TrainingAsset{}
	if err := db.Where(queryWherePresentationID, presentationId).Find(&assets).Error; err != nil {
		return nil, err
	}
	if err := db.Where(queryWherePresentationID, presentationId).Delete(&TrainingAsset{}).Error; err != nil {
		return assets, err
	}
	return assets, nil
}

// ReorderTrainingAssets updates the sort_order of the given asset ids in the
// order they appear in the slice. Used by the drag-and-drop module reorder UI.
func ReorderTrainingAssets(presentationId int64, ids []int64, scope OrgScope) error {
	for i, id := range ids {
		a, err := GetTrainingAsset(id, scope)
		if err != nil {
			return err
		}
		if a.PresentationId != presentationId {
			return ErrTrainingAssetNotFound
		}
		a.SortOrder = i + 1
		if err := db.Save(&a).Error; err != nil {
			return err
		}
	}
	return nil
}
