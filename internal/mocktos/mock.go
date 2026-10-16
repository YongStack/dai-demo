package mocktos

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MockTOS 用本地文件系统模拟对象存储操作
type MockTOS struct {
	baseDir string
}

// NewMockTOS 创建 MockTOS 实例
func NewMockTOS(baseDir string) (*MockTOS, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create mock tos dir: %w", err)
	}
	return &MockTOS{baseDir: baseDir}, nil
}

// CreateBucket 创建桶目录
func (m *MockTOS) CreateBucket(bucket string) error {
	dir := filepath.Join(m.baseDir, bucket)
	return os.MkdirAll(dir, 0755)
}

// DeleteBucket 删除桶目录
func (m *MockTOS) DeleteBucket(bucket string) error {
	dir := filepath.Join(m.baseDir, bucket)
	return os.RemoveAll(dir)
}

// PutObject 写入对象
func (m *MockTOS) PutObject(bucket, key string, reader io.Reader) error {
	dir := filepath.Join(m.baseDir, bucket)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, key)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

// DeleteObject 删除对象
func (m *MockTOS) DeleteObject(bucket, key string) error {
	path := filepath.Join(m.baseDir, bucket, key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("object not found: %s/%s", bucket, key)
	}
	return os.Remove(path)
}

// GetObject 读取对象内容
func (m *MockTOS) GetObject(bucket, key string) (io.ReadCloser, error) {
	path := filepath.Join(m.baseDir, bucket, key)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
		}
		return nil, err
	}
	return file, nil
}

// BucketExists 检查桶目录是否存在
func (m *MockTOS) BucketExists(bucket string) bool {
	dir := filepath.Join(m.baseDir, bucket)
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}
