package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// StorageService 存储服务接口
type StorageService interface {
	UploadFile(ctx context.Context, key string, filePath string, contentType string) error
	UploadBytes(ctx context.Context, key string, data []byte, contentType string) error
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	DeleteObject(ctx context.Context, key string) error
	GetLocalPath(key string) string
}

// LocalStorage 本地文件存储
type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &LocalStorage{baseDir: baseDir, baseURL: baseURL}, nil
}

func (s *LocalStorage) objectPath(key string) string {
	return filepath.Join(s.baseDir, key)
}

func (s *LocalStorage) UploadFile(ctx context.Context, key string, filePath string, contentType string) error {
	dst := s.objectPath(key)
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (s *LocalStorage) UploadBytes(ctx context.Context, key string, data []byte, contentType string) error {
	dst := s.objectPath(key)
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func (s *LocalStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if key == "" {
		return "", nil
	}
	// 本地存储直接返回相对路径
	return fmt.Sprintf("%s/%s", s.baseURL, key), nil
}

func (s *LocalStorage) DeleteObject(ctx context.Context, key string) error {
	return os.Remove(s.objectPath(key))
}

func (s *LocalStorage) GetLocalPath(key string) string {
	if key == "" {
		return ""
	}
	path := s.objectPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	return path
}
