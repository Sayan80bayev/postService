package s3

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"mime/multipart"
	"postService/internal/config"
	"strings"
)

// Init initializes a MinIO client
func Init(cfg *config.Config) *minio.Client {
	endpoint := "localhost:9000"
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to MinIO")
	return minioClient
}

// UploadFile uploads a file to MinIO without using gin.Context
func UploadFile(file multipart.File, header *multipart.FileHeader, cfg *config.Config, minioClient *minio.Client) (string, error) {
	if file == nil || header == nil {
		return "", errors.New("invalid file")
	}
	defer file.Close()

	objectName := header.Filename
	contentType := header.Header.Get("Content-Type")
	bucketName := cfg.MinioBucket

	_, err := minioClient.PutObject(
		context.Background(),
		bucketName,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", errors.New("failed to upload file")
	}

	// Generate file URL
	fileURL := fmt.Sprintf("http://localhost:9000/%s/%s", bucketName, objectName)

	return fileURL, nil
}

// DeleteFileByURL deletes a file from MinIO by its URL
// DeleteFileByURL deletes a file from MinIO by its URL
func DeleteFileByURL(fileURL string, minioClient *minio.Client) error {
	if fileURL == "" {
		return errors.New("missing file_url parameter")
	}

	// Убираем префикс "http://localhost:9000/"
	prefix := "http://localhost:9000/"
	if !strings.HasPrefix(fileURL, prefix) {
		return errors.New("invalid file_url format")
	}
	path := strings.TrimPrefix(fileURL, prefix)

	// Разбиваем на части (bucketName и objectName)
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return errors.New("invalid file_url format")
	}

	bucketName := parts[0] // crudexample
	objectName := parts[1] // 1a7eb51265f3bc780eeb0e7b1cb165d0.jpg

	// Удаление файла из MinIO
	err := minioClient.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return errors.New("failed to delete file")
	}

	return nil
}
