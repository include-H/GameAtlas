package services

import (
	"errors"
	"mime/multipart"
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestAssetsServiceUploadRejectsOversizedImage(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := newServicesAssetsService(db, t.TempDir())
	header := &multipart.FileHeader{Filename: "big.png", Size: maxImageUploadBytes + 1}

	_, err := service.Upload(1, "cover", header)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestAssetsServiceUploadRejectsOversizedVideo(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := newServicesAssetsService(db, t.TempDir())
	header := &multipart.FileHeader{Filename: "big.mp4", Size: maxVideoUploadBytes + 1}

	_, err := service.Upload(1, "video", header)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestAssetsServiceUploadAllowsLargeVideoWithinLimit(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := newServicesAssetsService(db, t.TempDir())
	header := &multipart.FileHeader{Filename: "movie.mp4", Size: 100 << 20}

	// 100 MiB passes the size check; the next failure should come from a missing game.
	_, err := service.Upload(1, "video", header)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound (size check should pass)", err)
	}
}
