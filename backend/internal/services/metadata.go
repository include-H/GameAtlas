package services

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type MetadataService struct {
	repo      *repositories.MetadataRepository
	assetsDir string
	dataDir   string
}

type MetadataResource struct {
	Table        string
	ResourceName string
}

func NewMetadataService(cfg config.Config, repo *repositories.MetadataRepository) *MetadataService {
	return &MetadataService{
		repo:      repo,
		assetsDir: cfg.AssetsDir,
		dataDir:   filepath.Dir(cfg.AssetsDir),
	}
}

func (s *MetadataService) List(resource MetadataResource) ([]domain.MetadataItem, error) {
	items, err := s.repo.List(resource.Table)
	if err != nil {
		return nil, err
	}
	if resource.Table == "series" {
		for index := range items {
			s.enrichSeriesItem(&items[index])
		}
	}
	if items == nil {
		return []domain.MetadataItem{}, nil
	}
	return items, nil
}

func (s *MetadataService) Create(resource MetadataResource, input domain.MetadataWriteInput) (*domain.MetadataItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrValidation
	}

	slug := trimStringPtr(input.Slug)
	slugValue := ""
	if slug != nil {
		slugValue = slugify(*slug)
	}
	if slugValue == "" {
		slugValue = slugify(name)
	}
	if slugValue == "" {
		return nil, ErrValidation
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	cleanInput := domain.MetadataWriteInput{
		Name:      name,
		Slug:      &slugValue,
		SortOrder: &sortOrder,
	}

	switch resource.Table {
	case "series":
		return s.repo.CreateSeries(cleanInput, slugValue, sortOrder)
	case "platforms", "developers", "publishers":
		return s.repo.CreateSimple(resource.Table, cleanInput, slugValue, sortOrder)
	default:
		return nil, fmt.Errorf("unsupported metadata resource: %s", resource.Table)
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(builder.String(), "-")
	return result
}

func (s *MetadataService) enrichSeriesItem(item *domain.MetadataItem) {
	games, err := s.repo.ListSeriesGames(item.ID)
	if err != nil {
		return
	}

	item.GameCount = len(games)
	if len(games) == 0 {
		return
	}

	item.LatestUpdatedAt = &games[0].UpdatedAt
	coverCandidates := make([]string, 0, len(games))
	for _, game := range games {
		if path := pickSeriesCoverSource(game); path != "" {
			coverCandidates = append(coverCandidates, path)
		}
		if len(coverCandidates) >= 4 {
			break
		}
	}

	if len(coverCandidates) >= 4 {
		if generated, err := s.ensureSeriesCompositeCover(item.ID, coverCandidates[:4]); err == nil && strings.TrimSpace(generated) != "" {
			item.CoverImage = &generated
			return
		}
	}

	fallback := pickSeriesCoverSource(games[0])
	if fallback != "" {
		item.CoverImage = &fallback
	}
}

func pickSeriesCoverSource(game domain.Game) string {
	if game.CoverImage != nil && strings.TrimSpace(*game.CoverImage) != "" {
		return strings.TrimSpace(*game.CoverImage)
	}
	if game.BannerImage != nil && strings.TrimSpace(*game.BannerImage) != "" {
		return strings.TrimSpace(*game.BannerImage)
	}
	if game.PrimaryScreenshot != nil && strings.TrimSpace(*game.PrimaryScreenshot) != "" {
		return strings.TrimSpace(*game.PrimaryScreenshot)
	}
	return ""
}

func (s *MetadataService) ensureSeriesCompositeCover(seriesID int64, assetPaths []string) (string, error) {
	if len(assetPaths) < 4 {
		return "", nil
	}

	filename := seriesCompositeFilename(seriesID, assetPaths)
	targetDir := filepath.Join(s.dataDir, "series")
	targetPath := filepath.Join(targetDir, filename)
	publicPath := "/data/series/" + filename

	if _, err := os.Stat(targetPath); err == nil {
		return publicPath, nil
	}

	images := make([]image.Image, 0, 4)
	for _, assetPath := range assetPaths {
		img, err := s.loadLocalAssetImage(assetPath)
		if err != nil {
			continue
		}
		images = append(images, img)
		if len(images) == 4 {
			break
		}
	}
	if len(images) < 4 {
		return "", nil
	}

	canvas := image.NewRGBA(image.Rect(0, 0, 1200, 1800))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)
	cellWidth := canvas.Bounds().Dx() / 2
	cellHeight := canvas.Bounds().Dy() / 2

	for index, src := range images[:4] {
		x := (index % 2) * cellWidth
		y := (index / 2) * cellHeight
		targetRect := image.Rect(x, y, x+cellWidth, y+cellHeight)
		draw.Draw(canvas, targetRect, cropToFill(src, cellWidth, cellHeight), image.Point{}, draw.Src)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := jpeg.Encode(file, canvas, &jpeg.Options{Quality: 88}); err != nil {
		return "", err
	}

	return publicPath, nil
}

func seriesCompositeFilename(seriesID int64, assetPaths []string) string {
	key := fmt.Sprintf("%d|%s", seriesID, strings.Join(assetPaths, "|"))
	sum := sha1.Sum([]byte(key))
	shortID := hex.EncodeToString(sum[:3])
	return shortID + ".jpg"
}

func (s *MetadataService) loadLocalAssetImage(publicPath string) (image.Image, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(publicPath), "/")
	if !strings.HasPrefix(trimmed, "assets/") {
		return nil, fmt.Errorf("unsupported asset path")
	}
	relative := strings.TrimPrefix(trimmed, "assets/")
	fullPath := filepath.Join(s.assetsDir, filepath.FromSlash(relative))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func cropToFill(src image.Image, width int, height int) image.Image {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW <= 0 || srcH <= 0 || width <= 0 || height <= 0 {
		return src
	}

	targetRatio := float64(width) / float64(height)
	srcRatio := float64(srcW) / float64(srcH)

	crop := srcBounds
	if srcRatio > targetRatio {
		newW := int(float64(srcH) * targetRatio)
		offsetX := (srcW - newW) / 2
		crop = image.Rect(srcBounds.Min.X+offsetX, srcBounds.Min.Y, srcBounds.Min.X+offsetX+newW, srcBounds.Max.Y)
	} else {
		newH := int(float64(srcW) / targetRatio)
		offsetY := (srcH - newH) / 2
		crop = image.Rect(srcBounds.Min.X, srcBounds.Min.Y+offsetY, srcBounds.Max.X, srcBounds.Min.Y+offsetY+newH)
	}

	target := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := crop.Min.Y + (y*crop.Dy())/height
		for x := 0; x < width; x++ {
			srcX := crop.Min.X + (x*crop.Dx())/width
			target.Set(x, y, src.At(srcX, srcY))
		}
	}
	return target
}
