package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"all2wei/internal/config"
)

type MinIOService struct {
	client *minio.Client
	bucket string
}

func NewMinIOService(cfg *config.MinIOConfig) (*MinIOService, error) {
	// 创建自定义 HTTP 客户端，跳过 SSL 证书验证
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// 设置 region，默认为 us-east-1
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:    cfg.UseSSL,
		Transport: httpClient.Transport,
		Region:    region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	svc := &MinIOService{
		client: client,
		bucket: cfg.BucketName,
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return svc, nil
}

func (s *MinIOService) UploadFile(ctx context.Context, objectKey string, filePath string, contentType string) error {
	_, err := s.client.FPutObject(ctx, s.bucket, objectKey, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOService) UploadBytes(ctx context.Context, objectKey string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, 
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOService) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if objectKey == "" {
		return "", nil
	}
	reqParams := make(map[string][]string)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (s *MinIOService) DeleteObject(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *MinIOService) GetLocalPath(key string) string {
	return ""
}

func (s *MinIOService) GetBucketName() string {
	return s.bucket
}

// ListObjects 列出桶中所有对象
func (s *MinIOService) ListObjects(ctx context.Context, prefix string) ([]minio.ObjectInfo, error) {
	var objects []minio.ObjectInfo
	
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object)
	}
	
	return objects, nil
}
