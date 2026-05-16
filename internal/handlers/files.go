package handlers

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mm/service/internal/models"
	"mm/service/pkg/initializer"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TODO: Rewrite this file to use the S3 client registered once on init rather than by endpoint.
// Move structs to a new folder/file?
// Util functions can stay outside of loadEnv; want to use in other files.

type presignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Bucket      string `json:"bucket"`
}

type presignResponse struct {
	PresignedURL string `json:"presignedUrl"`
	PublicURL    string `json:"publicUrl"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var (
	garageOnce    sync.Once
	garageClient  *s3.Client
	garageBaseURL string
	garageInitErr error

	defaultUploadBucket = "matchmaking"
	avatarUploadBucket  = "matchmaking-avatar"

	allowedTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/gif":  {},
	}
)

func GenerateFileUploadURL(c *gin.Context) {
	client, publicBase, err := getGarageClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var req presignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Filename == "" || req.ContentType == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "filename and contentType are required"})
		return
	}

	bucket := strings.TrimSpace(req.Bucket)
	if bucket == "" {
		bucket = defaultUploadBucket
	}

	user, err := getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	key, err := makeObjectKey("uploads", user.ID, req.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	presignClient := s3.NewPresignClient(client)
	presignResult, err := presignClient.PresignPutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(60*time.Second),
	)
	if err != nil {
		log.Printf("presign error: %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not generate upload URL"})
		return
	}

	fileURL := buildPublicURL(publicBase, key)
	c.JSON(http.StatusOK, presignResponse{
		PresignedURL: presignResult.URL,
		PublicURL:    fileURL,
	})
}

func UploadAvatarAndSave(c *gin.Context) {
	client, publicBase, err := getGarageClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	user, err := getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "file is required (multipart field 'file')"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "filename must include an extension"})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}
	if _, ok := allowedTypes[contentType]; !ok {
		c.JSON(http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("unsupported content type: %s", contentType)})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not read uploaded file"})
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("close uploaded file error: %v", closeErr)
		}
	}()

	key, err := makeObjectKey("avatars", user.ID, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(avatarUploadBucket),
		Key:           aws.String(key),
		Body:          file,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(fileHeader.Size),
	})
	if err != nil {
		log.Printf("upload error: %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not upload avatar"})
		return
	}

	avatarURL := buildPublicURL(publicBase, key)
	if err := updateUserAvatarURL(user.ID, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "avatar uploaded but failed to save profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"publicUrl": avatarURL})
}

func SetAvatarURL(c *gin.Context) {
	user, err := getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	var body struct {
		AvatarURL string `json:"avatarUrl"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.AvatarURL == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "avatarUrl is required"})
		return
	}

	if err := updateUserAvatarURL(user.ID, body.AvatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not save avatar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"avatarUrl": body.AvatarURL})
}

func getCurrentUser(c *gin.Context) (models.User, error) {
	userValue, exists := c.Get("currentUser")
	if !exists {
		return models.User{}, fmt.Errorf("user not found in context")
	}

	user, ok := userValue.(models.User)
	if !ok {
		return models.User{}, fmt.Errorf("invalid user type")
	}

	return user, nil
}

// getGarageClient returns a singleton S3 client pointed at a Garage instance.
// Garage is S3-compatible but requires path-style addressing and ignores the
// region value; we pass "garage" as a harmless placeholder.
func getGarageClient() (*s3.Client, string, error) {
	garageOnce.Do(func() {
		endpoint, err := requiredEnv("GARAGE_ENDPOINT")
		if err != nil {
			garageInitErr = err
			return
		}

		accessKey, err := requiredEnv("GARAGE_ACCESS_KEY")
		if err != nil {
			garageInitErr = err
			return
		}

		secretKey, err := requiredEnv("GARAGE_SECRET_KEY")
		if err != nil {
			garageInitErr = err
			return
		}

		publicBase, err := requiredEnv("GARAGE_PUBLIC_BASE")
		if err != nil {
			garageInitErr = err
			return
		}

		// Build a fully-qualified endpoint URL. GARAGE_USE_SSL defaults to true;
		// set it to "false" explicitly to disable (e.g. in local dev).
		useSSL := os.Getenv("GARAGE_USE_SSL") != "false"
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = scheme + "://" + endpoint
		}

		cfg, err := awsconfig.LoadDefaultConfig(
			context.Background(),
			// Garage ignores the region but the SDK requires a non-empty value.
			awsconfig.WithRegion("garage"),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		)
		if err != nil {
			garageInitErr = fmt.Errorf("failed to load S3 config: %w", err)
			return
		}

		garageClient = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			// Garage does not support virtual-hosted–style bucket addressing;
			// path-style (e.g. http://host/bucket/key) must be used.
			o.UsePathStyle = true
		})
		garageBaseURL = publicBase
	})

	return garageClient, garageBaseURL, garageInitErr
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing required env var: %s", key)
	}
	return value, nil
}

func makeObjectKey(prefix string, userID uint64, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "", fmt.Errorf("filename must include an extension")
	}
	return fmt.Sprintf("%s/%d/%s%s", prefix, userID, uuid.NewString(), ext), nil
}

func buildPublicURL(publicBase, key string) string {
	return strings.TrimRight(publicBase, "/") + "/" + key
}

func updateUserAvatarURL(userID uint64, avatarURL string) error {
	result := initializer.DB.Model(&models.UserDetail{}).
		Where("user_id = ?", userID).
		Update("avatar", avatarURL)
	return result.Error
}
