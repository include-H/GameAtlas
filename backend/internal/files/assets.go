package files

import (
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register webp decoder for image.Decode

	_ "image/gif"  // register gif decoder for image.Decode
	_ "image/jpeg" // register jpeg decoder for image.Decode
	_ "image/png"  // register png decoder for image.Decode
)

// AllowedVariantWidths 是允许生成的缩放变体宽度白名单，
// 防止任意 ?w= 参数触发无限生成撑爆磁盘。
var AllowedVariantWidths = map[int]bool{320: true, 480: true, 640: true, 960: true, 1280: true, 1920: true}

// IsAllowedVariantWidth reports whether the requested variant width is allowed.
func IsAllowedVariantWidth(width int) bool {
	return AllowedVariantWidths[width]
}

var allowedImageContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

var allowedVideoContentTypes = map[string]string{
	"video/mp4":  ".mp4",
	"video/webm": ".webm",
}

var ErrInvalidImageType = errors.New("invalid image type")
var ErrInvalidAssetName = errors.New("invalid asset name")
var ErrInvalidRemoteURL = errors.New("invalid remote image url")

var uuidAssetNamePattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)

// AssetStore 只负责本地素材的 staging/permanent 两阶段存取。
// 远程素材下载已收敛到 /api/steam/proxy + 前端重新上传的单一路径，
// 这里不再持有出站 HTTP 客户端，避免维护一条没有生产调用的 SSRF 边界。
type AssetStore struct {
	baseDir string
}

func NewAssetStore(baseDir string) *AssetStore {
	return &AssetStore{
		baseDir: strings.TrimSpace(baseDir),
	}
}

// SaveToStaging writes an uploaded file to the staging directory and returns
// the permanent-path format (/assets/{gamePublicID}/{uuid}.{ext}). The actual
// file lives under {baseDir}/_staging/{uuid}.{ext} until MoveToPermanent is called.
func (s *AssetStore) SaveToStaging(
	gamePublicID string,
	assetType string,
	assetName string,
	file io.Reader,
	contentType string,
) (string, error) {
	extension, err := validateAssetContentType(assetType, contentType)
	if err != nil {
		return "", err
	}
	if !uuidAssetNamePattern.MatchString(strings.ToLower(strings.TrimSpace(assetName))) {
		return "", ErrInvalidAssetName
	}

	stagingDir := filepath.Join(s.baseDir, "_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return "", fmt.Errorf("create staging directory: %w", err)
	}

	filename := strings.ToLower(strings.TrimSpace(assetName)) + extension
	targetPath := filepath.Join(stagingDir, filename)
	output, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("create staging file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, file); err != nil {
		return "", fmt.Errorf("write staging file: %w", err)
	}

	dir := strings.ToLower(strings.TrimSpace(gamePublicID))
	if dir == "" {
		dir = "unknown-game"
	}
	return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
}

// MoveToPermanent moves a file from the staging directory to the permanent
// game-specific directory. permanentPath is the /assets/{gamePublicID}/{filename} format.
func (s *AssetStore) MoveToPermanent(permanentPath string, gamePublicID string) (string, error) {
	cleaned := strings.TrimSpace(permanentPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || !strings.HasPrefix(cleaned, "assets/") {
		return "", fmt.Errorf("invalid permanent path: %s", permanentPath)
	}

	filename := filepath.Base(cleaned)
	stagingFile := filepath.Join(s.baseDir, "_staging", filename)

	dir := strings.ToLower(strings.TrimSpace(gamePublicID))
	if dir == "" {
		dir = "unknown-game"
	}
	permDir := filepath.Join(s.baseDir, dir)
	if err := os.MkdirAll(permDir, 0o755); err != nil {
		return "", fmt.Errorf("create permanent directory: %w", err)
	}

	permFile := filepath.Join(permDir, filename)

	// If already in permanent location, nothing to do.
	if _, err := os.Stat(permFile); err == nil {
		// Clean up staging file if it exists.
		_ = os.Remove(stagingFile)
		return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
	}

	// Move from staging to permanent.
	if err := os.Rename(stagingFile, permFile); err != nil {
		return "", fmt.Errorf("move staging to permanent: %w", err)
	}

	return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
}

// BaseDir returns the root directory for asset storage.
func (s *AssetStore) BaseDir() string {
	return s.baseDir
}

// CleanStaging deletes files in the staging directory that are older than maxAge.
func (s *AssetStore) CleanStaging(maxAge time.Duration) (int, error) {
	stagingDir := filepath.Join(s.baseDir, "_staging")
	if _, err := os.Stat(stagingDir); os.IsNotExist(err) {
		return 0, nil
	}

	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return 0, fmt.Errorf("read staging directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(stagingDir, entry.Name())); err == nil {
				deleted++
			}
		}
	}
	return deleted, nil
}

func (s *AssetStore) DeleteAsset(assetPath string) error {
	targetPath, err := s.resolveAssetPath(assetPath)
	if err != nil {
		return err
	}
	return os.Remove(targetPath)
}

// AssetExists treats invalid or escaped asset paths as missing so callers can safely prune
// stale database references without leaking filesystem traversal semantics back up the stack.
// It checks both the permanent location and the staging directory.
func (s *AssetStore) AssetExists(assetPath string) bool {
	targetPath, err := s.resolveAssetPath(assetPath)
	if err != nil {
		return false
	}
	if _, err := os.Stat(targetPath); err == nil {
		return true
	}
	// Fallback: check staging directory.
	filename := filepath.Base(targetPath)
	stagingPath := filepath.Join(s.baseDir, "_staging", filename)
	if _, err := os.Stat(stagingPath); err == nil {
		return true
	}
	return false
}

// EnsureVariant returns the disk path to serve for assetPath at the given width.
// It lazily generates and permanently stores a WebP variant named
// <original-name>.w<width>.webp next to the original, and serves the original
// unchanged when the width is not allowed, the source is an animated GIF,
// decoding/resizing fails, or the original is already narrower than the target.
func (s *AssetStore) EnsureVariant(assetPath string, width int) (string, error) {
	originalPath, err := s.resolveAssetPath(assetPath)
	if err != nil {
		return "", err
	}
	if width <= 0 || !IsAllowedVariantWidth(width) || strings.EqualFold(filepath.Ext(originalPath), ".gif") {
		return originalPath, nil
	}

	variantPath := variantFilePath(originalPath, width)
	if _, err := os.Stat(variantPath); err == nil {
		return variantPath, nil
	}

	source, err := openImage(originalPath)
	if err != nil {
		return originalPath, nil
	}
	defer source.Close()

	img, _, err := image.Decode(source)
	if err != nil {
		return originalPath, nil
	}
	bounds := img.Bounds()
	if bounds.Dx() <= width {
		return originalPath, nil
	}

	resized := resizeToWidth(img, width)
	if err := writeWebPFile(variantPath, resized); err != nil {
		return originalPath, nil
	}
	return variantPath, nil
}

func variantFilePath(originalPath string, width int) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return fmt.Sprintf("%s.w%d.webp", base, width)
}

func openImage(path string) (*os.File, error) {
	return os.Open(path)
}

func resizeToWidth(src image.Image, width int) image.Image {
	srcBounds := src.Bounds()
	height := int(math.Round(float64(srcBounds.Dy()) * float64(width) / float64(srcBounds.Dx())))
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)
	return dst
}

func writeWebPFile(path string, img image.Image) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".variant-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if err := webp.Encode(tmp, img, &webp.Options{Quality: 80}); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (s *AssetStore) resolveAssetPath(assetPath string) (string, error) {
	cleaned := strings.TrimSpace(assetPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || !strings.HasPrefix(cleaned, "assets/") {
		return "", os.ErrNotExist
	}

	relativeAssetPath := strings.TrimPrefix(cleaned, "assets/")
	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(relativeAssetPath))
	relative, err := filepath.Rel(s.baseDir, targetPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrInvalidRemoteURL
	}
	return targetPath, nil
}

func validateAssetContentType(assetType string, contentType string) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	allowed := allowedImageContentTypes
	if assetType == "video" {
		allowed = allowedVideoContentTypes
	}
	if extension, ok := allowed[contentType]; ok {
		return extension, nil
	}

	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		if extension, ok := allowed[parsed]; ok {
			return extension, nil
		}
	}

	return "", ErrInvalidImageType
}
