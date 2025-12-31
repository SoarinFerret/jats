package services

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/soarinferret/jats/internal/models"
)

type StorageService struct {
	baseDir string
}

func NewStorageService(baseDir string) *StorageService {
	// Create base directory if it doesn't exist
	os.MkdirAll(baseDir, 0755)
	return &StorageService{baseDir: baseDir}
}

func (s *StorageService) SaveAttachment(filename string, contentType string, data []byte) (*models.Attachment, error) {
	// Generate unique filename to avoid conflicts
	hash := sha256.Sum256(data)
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	nanoseconds := now.Nanosecond()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = s.getExtensionFromContentType(contentType)
	}

	uniqueFilename := fmt.Sprintf("%s-%09d-%x%s", timestamp, nanoseconds, hash[:8], ext)
	filePath := filepath.Join(s.baseDir, uniqueFilename)

	// Save file to disk
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	attachment := &models.Attachment{
		FileName:     uniqueFilename,
		OriginalName: filename,
		ContentType:  contentType,
		FilePath:     uniqueFilename, // Store relative path, not absolute
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return attachment, nil
}

func (s *StorageService) GetAttachment(filePath string) ([]byte, error) {
	fullPath := filepath.Join(s.baseDir, filePath)
	return os.ReadFile(fullPath)
}

func (s *StorageService) DeleteAttachment(filePath string) error {
	fullPath := filepath.Join(s.baseDir, filePath)
	return os.Remove(fullPath)
}

func (s *StorageService) GetFullPath(fileName string) string {
	return filepath.Join(s.baseDir, fileName)
}

func (s *StorageService) getExtensionFromContentType(contentType string) string {
	switch contentType {
	// Images
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"

	// Documents
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"

	// Text/Data
	case "text/plain":
		return ".txt"
	case "text/csv":
		return ".csv"
	case "application/json":
		return ".json"
	case "text/xml", "application/xml":
		return ".xml"
	case "text/html":
		return ".html"
	case "text/css":
		return ".css"
	case "text/javascript", "application/javascript":
		return ".js"
	case "application/yaml", "text/yaml":
		return ".yml"

	// Archives
	case "application/zip":
		return ".zip"
	case "application/x-tar":
		return ".tar"
	case "application/gzip":
		return ".gz"
	case "application/x-7z-compressed":
		return ".7z"
	case "application/x-rar-compressed":
		return ".rar"

	// Other
	case "application/octet-stream":
		return ".bin"

	default:
		return ".bin"
	}
}
