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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type presignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

type presignResponse struct {
	PresignedURL string `json:"presignedUrl"`
	PublicURL    string `json:"publicUrl"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var (
	minioOnce    sync.Once
	minioClient  *minio.Client
	minioBucket  string
	minioBaseURL string
	minioInitErr error

	allowedTypes = map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/gif":  {},
	}
)

func GenerateFileUploadURL(c *gin.Context) {
	client, bucket, publicBase, err := getMinioClient()
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

	presignedURL, err := client.PresignedPutObject(context.Background(), bucket, key, 60*time.Second)
	if err != nil {
		log.Printf("presign error: %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not generate upload URL"})
		return
	}

	fileURL := buildPublicURL(publicBase, key)
	c.JSON(http.StatusOK, presignResponse{
		PresignedURL: presignedURL.String(),
		PublicURL:    fileURL,
	})
}

func UploadAvatarAndSave(c *gin.Context) {
	client, bucket, publicBase, err := getMinioClient()
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

	_, err = client.PutObject(context.Background(), bucket, key, file, fileHeader.Size, minio.PutObjectOptions{ContentType: contentType})
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

func getMinioClient() (*minio.Client, string, string, error) {
	minioOnce.Do(func() {
		endpoint, err := requiredEnv("MINIO_ENDPOINT")
		if err != nil {
			minioInitErr = err
			return
		}

		accessKey, err := requiredEnv("MINIO_ACCESS_KEY")
		if err != nil {
			minioInitErr = err
			return
		}

		secretKey, err := requiredEnv("MINIO_SECRET_KEY")
		if err != nil {
			minioInitErr = err
			return
		}

		bucket, err := requiredEnv("MINIO_BUCKET")
		if err != nil {
			minioInitErr = err
			return
		}

		publicBase, err := requiredEnv("MINIO_PUBLIC_BASE")
		if err != nil {
			minioInitErr = err
			return
		}

		useSSL := os.Getenv("MINIO_USE_SSL") != "false"
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			minioInitErr = fmt.Errorf("failed to init MinIO client: %w", err)
			return
		}

		minioClient = client
		minioBucket = bucket
		minioBaseURL = publicBase
	})

	return minioClient, minioBucket, minioBaseURL, minioInitErr
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

func buildPublicURL(publicBase string, key string) string {
	return strings.TrimRight(publicBase, "/") + "/" + key
}

func updateUserAvatarURL(userID uint64, avatarURL string) error {
	result := initializer.DB.Model(&models.UserDetail{}).
		Where("user_id = ?", userID).
		Update("avatar", avatarURL)
	return result.Error
}
