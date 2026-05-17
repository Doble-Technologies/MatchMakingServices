package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

const (
	bucket       = "matchmaking"
	maxImageSize = 10 << 20 // 10 MB
)

// allowedMIMETypes maps permitted content types to their canonical extension.
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type ImageUploadHandler struct {
	s3 *s3.Client
}

func NewImageUploadHandler(s3Client *s3.Client) *ImageUploadHandler {
	return &ImageUploadHandler{s3: s3Client}
}

// UploadImage handles multipart image uploads to the Garage "matchmaking" bucket.
// POST /upload/image
// Form field: "image" (file)
func (h *ImageUploadHandler) UploadImage(c *gin.Context) {
	// 1. Parse and size-limit the multipart form.
	if err := c.Request.ParseMultipartForm(maxImageSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request too large or not multipart"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'image' field"})
		return
	}
	defer file.Close()

	// 2. Validate MIME type from the Content-Type header on the part.
	contentType := header.Header.Get("Content-Type")
	if _, ok := allowedMIMETypes[contentType]; !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error":   "unsupported image type",
			"allowed": keys(allowedMIMETypes),
		})
		return
	}

	// 3. Build a collision-resistant object key.
	ext := allowedMIMETypes[contentType]
	originalName := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	key := fmt.Sprintf("images/%d_%s%s", time.Now().UnixNano(), sanitise(originalName), ext)

	// 4. Stream directly to Garage — no local temp file needed.
	_, err = h.s3.PutObject(c.Request.Context(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(header.Size),
	})
	if err != nil {
		log.Printf("%s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":  key,
		"size": header.Size,
	})
}

// sanitise strips characters that are awkward in S3 object keys.
func sanitise(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
