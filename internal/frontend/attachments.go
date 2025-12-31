package frontend

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/services"
)

// AttachmentHandler handles attachment-related requests
type AttachmentHandler struct {
	taskService    *services.TaskService
	attachmentPath string
}

func NewAttachmentHandler(taskService *services.TaskService, attachmentPath string) *AttachmentHandler {
	return &AttachmentHandler{
		taskService:    taskService,
		attachmentPath: attachmentPath,
	}
}

// ServeAttachment serves attachment files
func (h *AttachmentHandler) ServeAttachment(c *gin.Context) {
	attachmentIDStr := c.Param("id")
	attachmentID, err := strconv.ParseUint(attachmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
		return
	}

	// Get attachment from service
	attachment, err := h.taskService.GetAttachment(uint(attachmentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}

	// Construct file path
	filePath := filepath.Join(h.attachmentPath, attachment.FilePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on disk"})
		return
	}

	// Set content type and filename
	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, attachment.OriginalName))

	// Serve the file
	c.File(filePath)
}