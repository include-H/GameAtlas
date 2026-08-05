package files

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

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

const thumbnailMaxWidth = 480

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
		s.moveThumbnailToPermanent(permDir, permanentPath)
		return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
	}

	// Move from staging to permanent.
	if err := os.Rename(stagingFile, permFile); err != nil {
		return "", fmt.Errorf("move staging to permanent: %w", err)
	}
	s.moveThumbnailToPermanent(permDir, permanentPath)

	return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
}

// moveThumbnailToPermanent moves the deterministic thumbnail file from staging
// into the permanent game directory when one was generated.
func (s *AssetStore) moveThumbnailToPermanent(permDir string, permanentPath string) {
	thumbURL := ThumbURLFor(permanentPath)
	thumbFilename := filepath.Base(thumbURL)
	thumbStaging := filepath.Join(s.baseDir, "_staging", thumbFilename)
	thumbPerm := filepath.Join(permDir, thumbFilename)

	if _, err := os.Stat(thumbPerm); err == nil {
		_ = os.Remove(thumbStaging)
		return
	}
	if _, err := os.Stat(thumbStaging); err == nil {
		_ = os.Rename(thumbStaging, thumbPerm)
	}
}

// BaseDir returns the root directory for asset storage.
func (s *AssetStore) BaseDir() string {
	return s.baseDir
}

// StagingDir returns the path to the staging directory.
func (s *AssetStore) StagingDir() string {
	return filepath.Join(s.baseDir, "_staging")
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
	if err := os.Remove(targetPath); err != nil {
		return err
	}
	if thumbPath, thumbErr := s.resolveAssetPath(ThumbURLFor(assetPath)); thumbErr == nil {
		_ = os.Remove(thumbPath)
	}
	return nil
}

// ThumbURLFor returns the deterministic thumbnail URL for an asset path.
// Thumbnails are only generated for images; video paths fall back to themselves.
func ThumbURLFor(assetPath string) string {
	ext := filepath.Ext(assetPath)
	if ext == "" {
		return assetPath
	}
	return strings.TrimSuffix(assetPath, ext) + ".thumb.jpg"
}

// WriteThumbnail generates a small JPEG thumbnail next to the original asset.
// It is a no-op for videos and for images already at or below the max width.
func (s *AssetStore) WriteThumbnail(assetPath string) error {
	srcPath, err := s.resolveAssetPath(assetPath)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(srcPath); os.IsNotExist(statErr) {
		srcPath = filepath.Join(s.baseDir, "_staging", filepath.Base(srcPath))
	}
	thumbPath, err := s.resolveAssetPath(ThumbURLFor(assetPath))
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(thumbPath); os.IsNotExist(statErr) {
		thumbPath = filepath.Join(s.baseDir, "_staging", filepath.Base(thumbPath))
	}
	return generateThumbnail(srcPath, thumbPath)
}

func generateThumbnail(srcPath string, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	if bounds.Dx() <= thumbnailMaxWidth {
		return nil
	}

	width := thumbnailMaxWidth
	height := bounds.Dy() * width / bounds.Dx()
	if height < 1 {
		height = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, dst, &jpeg.Options{Quality: 80})
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
