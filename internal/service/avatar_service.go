package service

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	identicon "github.com/dgryski/go-identicon"
)

const (
	avatarSize   = 256
	avatarDir    = "uploads/avatars"
	avatarPrefix = "/uploads/avatars"
)

// AvatarService manages avatar file storage, generation, and processing.
type AvatarService struct {
	uploadDir string
	baseURL   string
}

// NewAvatarService creates a new AvatarService and ensures the upload directory exists.
func NewAvatarService(uploadDir string) *AvatarService {
	if uploadDir == "" {
		uploadDir = avatarDir
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("[AvatarService] Failed to create upload directory %s: %v", uploadDir, err)
	}
	return &AvatarService{
		uploadDir: uploadDir,
		baseURL:   avatarPrefix,
	}
}

// GenerateDefaultAvatar generates a GitHub-style identicon avatar from the user's UUID.
// Returns the relative URL path to the generated avatar.
func (as *AvatarService) GenerateDefaultAvatar(userUUID string) (string, error) {
	// Archive any existing avatar (custom or previous default) before regenerating
	as.archiveIfExists(userUUID)

	filename := as.filename(userUUID)
	filePath := filepath.Join(as.uploadDir, filename)

	// Generate identicon from UUID hash
	// Use UUID hash as both the key and the data for the identicon
	hash := md5.Sum([]byte(userUUID))
	renderer := identicon.New7x7(hash[:])
	pngBytes := renderer.Render(hash[:])

	// Decode the identicon PNG
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode identicon: %w", err)
	}

	// Resize to standard avatar size
	resized := imaging.Resize(img, avatarSize, avatarSize, imaging.Lanczos)

	// Save
	if err := imaging.Save(resized, filePath, imaging.PNGCompressionLevel(png.BestCompression)); err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	log.Printf("[AvatarService] Generated default avatar for user %s", userUUID)
	return as.baseURL + "/" + filename, nil
}

// SaveAvatar reads an image from the reader, resizes it to 256x256, and saves as PNG.
// Returns the relative URL path to the saved avatar.
func (as *AvatarService) SaveAvatar(userUUID string, reader io.Reader) (string, error) {
	// Archive the old avatar before writing the new one
	as.archiveIfExists(userUUID)

	// Decode the uploaded image (supports JPEG, PNG, GIF, BMP, etc.)
	img, _, err := image.Decode(reader)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to square avatar (crop to center square first, then resize)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w != h {
		// Crop to center square
		size := w
		if h < w {
			size = h
		}
		img = imaging.CropCenter(img, size, size)
	}
	resized := imaging.Resize(img, avatarSize, avatarSize, imaging.Lanczos)

	// Save as PNG
	filename := as.filename(userUUID)
	savePath := filepath.Join(as.uploadDir, filename)
	if err := imaging.Save(resized, savePath, imaging.PNGCompressionLevel(png.BestCompression)); err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	log.Printf("[AvatarService] Saved avatar for user %s", userUUID)
	return as.baseURL + "/" + filename, nil
}

// archiveIfExists renames the current avatar file to a timestamped archive name.
// This preserves old avatars while freeing the canonical filename for the new upload.
func (as *AvatarService) archiveIfExists(userUUID string) {
	canonicalPath := filepath.Join(as.uploadDir, as.filename(userUUID))
	if _, err := os.Stat(canonicalPath); err == nil {
		archiveName := fmt.Sprintf("%s_%d.png", userUUID, time.Now().UnixNano())
		archivePath := filepath.Join(as.uploadDir, archiveName)
		if err := os.Rename(canonicalPath, archivePath); err != nil {
			log.Printf("[AvatarService] Failed to archive old avatar for %s: %v", userUUID, err)
		} else {
			log.Printf("[AvatarService] Archived old avatar for %s as %s", userUUID, archiveName)
		}
	}
}

// AvatarExists checks if an avatar file exists for the user.
func (as *AvatarService) AvatarExists(userUUID string) bool {
	path := filepath.Join(as.uploadDir, as.filename(userUUID))
	_, err := os.Stat(path)
	return err == nil
}

// DeleteAvatar removes a user's avatar file.
func (as *AvatarService) DeleteAvatar(userUUID string) error {
	path := filepath.Join(as.uploadDir, as.filename(userUUID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete avatar: %w", err)
	}
	return nil
}

func (as *AvatarService) filename(userUUID string) string {
	return userUUID + ".png"
}
