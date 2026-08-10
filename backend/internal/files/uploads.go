package files

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"mime"
	"strings"
)

const (
	MaxImageUploadBytes int64 = 20 << 20  // 20 MiB
	MaxVideoUploadBytes int64 = 500 << 20 // 500 MiB
	maxImageDimension         = 16384
	maxImagePixels      int64 = 100_000_000
)

var (
	ErrUploadTooLarge      = errors.New("upload too large")
	ErrInvalidImageContent = errors.New("invalid image content")
)

// ReadValidatedImage reads an image upload within a bounded memory budget and
// verifies both its declared media type and its actual image signature.
func ReadValidatedImage(file io.Reader, declaredSize int64, contentType string) ([]byte, error) {
	if file == nil {
		return nil, ErrInvalidImageContent
	}
	if declaredSize > MaxImageUploadBytes {
		return nil, ErrUploadTooLarge
	}

	_, err := imageFormatForContentType(contentType)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxImageUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxImageUploadBytes {
		return nil, ErrUploadTooLarge
	}

	expectedFormat, _ := imageFormatForContentType(contentType)
	config, actualFormat, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || actualFormat != expectedFormat {
		return nil, ErrInvalidImageContent
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maxImageDimension || config.Height > maxImageDimension ||
		int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, ErrInvalidImageContent
	}

	return data, nil
}

// NormalizeImageToJPEG keeps the existing bg.jpg URL contract while allowing
// the settings UI to accept the same image formats as the other upload paths.
func NormalizeImageToJPEG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidImageContent
	}

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func imageFormatForContentType(contentType string) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = parsed
	}

	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpeg", nil
	case "image/png":
		return "png", nil
	case "image/webp":
		return "webp", nil
	case "image/gif":
		return "gif", nil
	default:
		return "", ErrInvalidImageType
	}
}
