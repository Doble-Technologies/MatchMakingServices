package handlers

import (
	"fmt"
	"log"
	"mime/multipart"
	"mm/service/internal/models"
	"mm/service/pkg/initializer"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

// AI GENERATED + TUTORIAL, NEEDS REVIEW FOR FINAL
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

// ImageUploadResponse is the successful response body for image upload and URL retrieval.
type ImageUploadResponse struct {
	Key string `json:"key" example:"images/1234567890_photo.jpg"`
	URL string `json:"url" example:"https://s3.example.com/matchmaking/images/1234567890_photo.jpg?X-Amz-Signature=..."`
}

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
	Error string `json:"error" example:"missing 'image' field"`
}

// UnsupportedMediaTypeResponse is returned when the uploaded file's MIME type is not allowed.
type UnsupportedMediaTypeResponse struct {
	Error   string   `json:"error"   example:"unsupported image type"`
	Allowed []string `json:"allowed" example:"image/jpeg,image/png,image/webp,image/gif"`
}

type ImageUploadHandler struct {
	s3 *s3.Client
}

func NewImageUploadHandler(s3Client *s3.Client) *ImageUploadHandler {
	return &ImageUploadHandler{s3: s3Client}
}

// UploadImage godoc
//
//	@Summary		Upload an image
//	@Description	Accepts a multipart/form-data request containing a single image file,
//	@Description	validates its MIME type, stores it in the "matchmaking" S3 bucket under
//	@Description	the images/ prefix, and returns a 7-day presigned GET URL.
//	@Tags			images
//	@Accept			mpfd
//	@Produce		json
//	@Param			image	formData	file					true	"Image file to upload (JPEG, PNG, WebP, or GIF; max 10 MB)"
//	@Success		201		{object}	ImageUploadResponse		"Upload successful — presigned URL valid for 7 days"
//	@Failure		400		{object}	ErrorResponse			"Request too large, not multipart, or missing the 'image' field"
//	@Failure		415		{object}	UnsupportedMediaTypeResponse	"MIME type not allowed"
//	@Failure		500		{object}	ErrorResponse			"S3 upload failed or presign failed"
//	@Router			/upload/image [post]
func (h *ImageUploadHandler) UploadImage(c *gin.Context) {
	//Valid Token
	userInterface, exists := c.Get("currentUser")
	if !exists {
		c.JSON(401, gin.H{"error": "user not found in context"})
		return
	}
	user, _ := userInterface.(models.User)

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
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}(file)

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
	//Update database
	initializer.DB.Table("user_details").
		Where("? = user_id", user.ID).
		Update("avatar", aws.String(key))

	// Generate presigned URL valid for 7 days
	presignClient := s3.NewPresignClient(h.s3)
	presigned, err := presignClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(7*24*time.Hour))
	if err != nil {
		log.Printf("%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload succeeded but failed to generate URL"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key": key,
		"url": presigned.URL,
	})
}

// GetImageURL godoc
//
//	@Summary		Get a presigned image URL
//	@Description	Given an existing S3 object key, generates and returns a 7-day presigned
//	@Description	GET URL for the corresponding object in the "matchmaking" bucket.
//	@Tags			images
//	@Produce		json
//	@Param			key	query		string				true	"S3 object key"	example(images/1234567890_photo.jpg)
//	@Success		201	{object}	ImageUploadResponse	"Presigned URL generated successfully"
//	@Failure		400	{object}	ErrorResponse		"Missing key query parameter"
//	@Failure		500	{object}	ErrorResponse		"Failed to generate presigned URL"
//	@Router			/images/url [get]
func (h *ImageUploadHandler) GetImageURL(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing key"})
		return
	}
	//Passing in the handles and contex
	var url = GenerateUrl(key, c, h)
	if url == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate URL"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"key": key,
		"url": url,
	})
}

func GenerateUrl(key string, c *gin.Context, h *ImageUploadHandler) string {
	presignClient := s3.NewPresignClient(h.s3)
	presigned, err := presignClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(7*24*time.Hour))
	if err != nil {
		log.Printf("Failed to generate URL: %v", err)
		return ""
	}
	return presigned.URL
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
