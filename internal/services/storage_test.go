package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageService_SaveAttachment(t *testing.T) {
	// Create temporary directory for tests
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	testData := []byte("test file content")
	filename := "test.txt"
	contentType := "text/plain"

	attachment, err := service.SaveAttachment(filename, contentType, testData)
	if err != nil {
		t.Errorf("Failed to save attachment: %v", err)
	}

	if attachment == nil {
		t.Fatal("Expected attachment to be returned")
	}

	if attachment.OriginalName != filename {
		t.Errorf("Expected original name %s, got %s", filename, attachment.OriginalName)
	}

	if attachment.ContentType != contentType {
		t.Errorf("Expected content type %s, got %s", contentType, attachment.ContentType)
	}

	// Verify file was actually saved
	fullPath := service.GetFullPath(attachment.FilePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("Expected file to be saved to disk")
	}

	// Verify file content
	savedContent, err := os.ReadFile(fullPath)
	if err != nil {
		t.Errorf("Failed to read saved file: %v", err)
	}

	if string(savedContent) != string(testData) {
		t.Errorf("Expected content %s, got %s", string(testData), string(savedContent))
	}
}

func TestStorageService_GetAttachment(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	testData := []byte("test file content")
	filename := "test.txt"
	contentType := "text/plain"

	// Save attachment first
	attachment, err := service.SaveAttachment(filename, contentType, testData)
	if err != nil {
		t.Fatalf("Failed to save attachment: %v", err)
	}

	// Retrieve attachment
	retrievedData, err := service.GetAttachment(attachment.FilePath)
	if err != nil {
		t.Errorf("Failed to get attachment: %v", err)
	}

	if string(retrievedData) != string(testData) {
		t.Errorf("Expected content %s, got %s", string(testData), string(retrievedData))
	}
}

func TestStorageService_DeleteAttachment(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	testData := []byte("test file content")
	filename := "test.txt"
	contentType := "text/plain"

	// Save attachment first
	attachment, err := service.SaveAttachment(filename, contentType, testData)
	if err != nil {
		t.Fatalf("Failed to save attachment: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(service.GetFullPath(attachment.FilePath)); os.IsNotExist(err) {
		t.Fatal("Expected file to exist before deletion")
	}

	// Delete attachment
	err = service.DeleteAttachment(attachment.FilePath)
	if err != nil {
		t.Errorf("Failed to delete attachment: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(service.GetFullPath(attachment.FilePath)); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted")
	}
}

func TestStorageService_GetExtensionFromContentType(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{"JPEG image", "image/jpeg", ".jpg"},
		{"PNG image", "image/png", ".png"},
		{"PDF document", "application/pdf", ".pdf"},
		{"JSON data", "application/json", ".json"},
		{"XML data", "application/xml", ".xml"},
		{"CSV data", "text/csv", ".csv"},
		{"ZIP archive", "application/zip", ".zip"},
		{"JavaScript", "application/javascript", ".js"},
		{"Unknown type", "application/unknown", ".bin"},
		{"Empty type", "", ".bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected extension %s for content type %s, got %s", tt.expected, tt.contentType, result)
			}
		})
	}
}

func TestStorageService_UniqueFilenames(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	testData := []byte("test file content")
	filename := "test.txt"
	contentType := "text/plain"

	// Save two attachments with same original name
	attachment1, err := service.SaveAttachment(filename, contentType, testData)
	if err != nil {
		t.Fatalf("Failed to save first attachment: %v", err)
	}

	attachment2, err := service.SaveAttachment(filename, contentType, testData)
	if err != nil {
		t.Fatalf("Failed to save second attachment: %v", err)
	}

	// Verify they have different generated filenames
	if attachment1.FileName == attachment2.FileName {
		t.Error("Expected different generated filenames for duplicate uploads")
	}

	// Verify both original names are preserved
	if attachment1.OriginalName != filename || attachment2.OriginalName != filename {
		t.Error("Expected original names to be preserved")
	}

	// Verify both files exist
	if _, err := os.Stat(service.GetFullPath(attachment1.FilePath)); os.IsNotExist(err) {
		t.Error("Expected first file to exist")
	}
	if _, err := os.Stat(service.GetFullPath(attachment2.FilePath)); os.IsNotExist(err) {
		t.Error("Expected second file to exist")
	}
}

func TestStorageService_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "subdir", "nested")

	// This should create the directory structure
	service := NewStorageService(nonExistentDir)

	// Verify directory was created
	if _, err := os.Stat(nonExistentDir); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}

	// Verify we can save files to it
	testData := []byte("test")
	_, err := service.SaveAttachment("test.txt", "text/plain", testData)
	if err != nil {
		t.Errorf("Failed to save attachment to created directory: %v", err)
	}
}

func TestStorageService_EmptyFilename(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	testData := []byte("test content")
	contentType := "application/pdf"

	// Test with empty filename - should use content type for extension
	attachment, err := service.SaveAttachment("", contentType, testData)
	if err != nil {
		t.Errorf("Failed to save attachment with empty filename: %v", err)
	}

	// Should have PDF extension from content type
	if filepath.Ext(attachment.FileName) != ".pdf" {
		t.Errorf("Expected .pdf extension, got %s", filepath.Ext(attachment.FileName))
	}
}

func TestStorageService_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	service := NewStorageService(tempDir)

	// Create 1MB test data
	testData := make([]byte, 1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	attachment, err := service.SaveAttachment("large.bin", "application/octet-stream", testData)
	if err != nil {
		t.Errorf("Failed to save large attachment: %v", err)
	}

	// Verify file was saved correctly
	savedData, err := service.GetAttachment(attachment.FilePath)
	if err != nil {
		t.Errorf("Failed to retrieve large attachment: %v", err)
	}

	if len(savedData) != len(testData) {
		t.Errorf("Expected %d bytes, got %d bytes", len(testData), len(savedData))
	}

	// Verify content integrity
	for i, b := range savedData {
		if b != testData[i] {
			t.Errorf("Data mismatch at byte %d: expected %d, got %d", i, testData[i], b)
			break
		}
	}
}
