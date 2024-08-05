package minio

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go"
	"go.uber.org/zap"

	"github.com/instill-ai/model-backend/config"

	log "github.com/instill-ai/model-backend/pkg/logger"
)

type MinioI interface {
	GetClient() *minio.Client
	UploadBase64File(ctx context.Context, filePath string, base64Content string, fileMimeType string) (err error)
	DeleteFile(ctx context.Context, filePath string) (err error)
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	GetFilesByPaths(ctx context.Context, filePaths []string) ([]FileContent, error)
}

const Location = "us-east-1"

type Minio struct {
	client *minio.Client
	bucket string
}

func NewMinioClientAndInitBucket(cfg *config.MinioConfig) (*Minio, error) {
	logger, err := log.GetZapLogger(context.Background())
	if err != nil {
		return nil, err
	}
	logger.Info("Initializing Minio client and bucket...")

	client, err := minio.New(cfg.Host+":"+cfg.Port, cfg.RootUser, cfg.RootPwd, false)
	if err != nil {
		logger.Error("cannot connect to minio",
			zap.String("host:port", cfg.Host+":"+cfg.Port),
			zap.String("user", cfg.RootUser),
			zap.String("pwd", cfg.RootPwd), zap.Error(err))
		return nil, err
	}

	exists, err := client.BucketExists(cfg.BucketName)
	if err != nil {
		logger.Error("failed in checking BucketExists", zap.Error(err))
		return nil, err
	}
	if exists {
		logger.Info("Bucket already exists", zap.String("bucket", cfg.BucketName))
		return &Minio{client: client, bucket: cfg.BucketName}, nil
	}

	if err = client.MakeBucket(cfg.BucketName, Location); err != nil {
		logger.Error("creating Bucket failed", zap.Error(err))
		return nil, err
	}
	logger.Info("Successfully created bucket", zap.String("bucket", cfg.BucketName))

	return &Minio{client: client, bucket: cfg.BucketName}, nil
}

func (m *Minio) GetClient() *minio.Client {
	return m.client
}

func (m *Minio) UploadBase64File(ctx context.Context, filePathName string, base64Content string, fileMimeType string) (err error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return err
	}
	// Decode the base64 content
	decodedContent, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return err
	}
	// Convert the decoded content to an io.Reader
	contentReader := strings.NewReader(string(decodedContent))
	// Upload the content to MinIO
	size := int64(len(decodedContent))
	// Create the file path with folder structure
	_, err = m.client.PutObjectWithContext(ctx, m.bucket, filePathName, contentReader, size, minio.PutObjectOptions{ContentType: fileMimeType})
	if err != nil {
		logger.Error("Failed to upload file to MinIO", zap.Error(err))
		return err
	}
	return nil
}

// DeleteFile delete the file from minio
func (m *Minio) DeleteFile(ctx context.Context, filePathName string) (err error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return err
	}
	// Delete the file from MinIO
	err = m.client.RemoveObject(m.bucket, filePathName)
	if err != nil {
		logger.Error("Failed to delete file from MinIO", zap.Error(err))
		return err
	}
	return nil
}

func (m *Minio) GetFile(ctx context.Context, filePathName string) ([]byte, error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return nil, err
	}

	// Get the object using the client
	object, err := m.client.GetObject(m.bucket, filePathName, minio.GetObjectOptions{})
	if err != nil {
		logger.Error("Failed to get file from MinIO", zap.Error(err))
		return nil, err
	}
	defer object.Close()

	// Read the object's content
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		logger.Error("Failed to read file from MinIO", zap.Error(err))
		return nil, err
	}

	return buf.Bytes(), nil
}

// FileContent represents a file and its content
type FileContent struct {
	Name    string
	Content []byte
}

// GetFilesByPaths GetFiles retrieves the contents of specified files from MinIO
func (m *Minio) GetFilesByPaths(ctx context.Context, filePaths []string) ([]FileContent, error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	fileCount := len(filePaths)

	errCh := make(chan error, fileCount)
	resultCh := make(chan FileContent, fileCount)

	for _, path := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			obj, err := m.client.GetObject(m.bucket, filePath, minio.GetObjectOptions{})
			if err != nil {
				logger.Error("Failed to get object from MinIO", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}
			defer obj.Close()

			var buffer bytes.Buffer
			_, err = io.Copy(&buffer, obj)
			if err != nil {
				logger.Error("Failed to read object content", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}

			fileContent := FileContent{
				Name:    filepath.Base(filePath),
				Content: buffer.Bytes(),
			}
			resultCh <- fileContent
		}(path)
	}

	wg.Wait()
	close(errCh)
	close(resultCh)

	var errs []error
	for err = range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	files := make([]FileContent, fileCount)
	for fileContent := range resultCh {
		files = append(files, fileContent)
	}

	return files, nil
}
