package service

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/YongStack/dai-demo/internal/mocktos"
	"github.com/YongStack/dai-demo/internal/model"
	"github.com/YongStack/dai-demo/internal/store"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

// Service 封装核心业务逻辑
type Service struct {
	store   *store.Store
	mockTOS *mocktos.MockTOS
}

// NewService 创建 Service 实例
func NewService(s *store.Store, t *mocktos.MockTOS) *Service {
	return &Service{store: s, mockTOS: t}
}

// ValidateBucketName 校验桶名是否合法
func ValidateBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	return bucketNameRegex.MatchString(name)
}

// CreateBucket 创建存储桶并设置配额
func (s *Service) CreateBucket(req *model.CreateBucketRequest) (*model.Bucket, error) {
	if !ValidateBucketName(req.Bucket) {
		return nil, fmt.Errorf("invalid bucket name")
	}
	if req.QuotaCapacity <= 0 {
		return nil, fmt.Errorf("invalid quota limit")
	}

	// 检查是否已存在
	existing, err := s.store.GetBucket(req.Bucket)
	if err != nil {
		return nil, fmt.Errorf("query bucket: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("bucket already exists: %s", req.Bucket)
	}

	// Mock TOS 创建桶
	if err := s.mockTOS.CreateBucket(req.Bucket); err != nil {
		return nil, fmt.Errorf("tos create bucket: %w", err)
	}

	now := time.Now()
	bucket := &model.Bucket{
		Name:           req.Bucket,
		QuotaCapacity:  req.QuotaCapacity,
		QuotaReadBPS:   req.QuotaReadBPS,
		QuotaWriteBPS:  req.QuotaWriteBPS,
		QuotaReadIOPS:  req.QuotaReadIOPS,
		QuotaWriteIOPS: req.QuotaWriteIOPS,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.store.CreateBucket(bucket); err != nil {
		// 回滚 Mock TOS
		s.mockTOS.DeleteBucket(req.Bucket)
		return nil, fmt.Errorf("persist bucket: %w", err)
	}

	return bucket, nil
}

// GetBucket 获取桶信息
func (s *Service) GetBucket(name string) (*model.Bucket, error) {
	b, err := s.store.GetBucket(name)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("bucket not found: %s", name)
	}
	return b, nil
}

// ListBuckets 列出所有桶
func (s *Service) ListBuckets() ([]*model.Bucket, error) {
	return s.store.ListBuckets()
}

// DeleteBucket 删除存储桶
func (s *Service) DeleteBucket(name string) error {
	b, err := s.store.GetBucket(name)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("bucket not found: %s", name)
	}

	// 先删除 Mock TOS 桶
	if err := s.mockTOS.DeleteBucket(name); err != nil {
		return fmt.Errorf("tos delete bucket: %w", err)
	}

	// 删除 SQLite 记录
	if err := s.store.DeleteBucket(name); err != nil {
		return fmt.Errorf("delete bucket record: %w", err)
	}

	return nil
}

// UploadObject 上传对象并校验容量配额
func (s *Service) UploadObject(bucket, key string, size int64, data []byte) (*model.UploadResponse, error) {
	b, err := s.store.GetBucket(bucket)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("bucket not found: %s", bucket)
	}

	// 容量配额校验
	remaining := b.QuotaCapacity - b.UsedCapacity
	if size > remaining {
		return nil, &QuotaExceededError{
			Size:      size,
			Remaining: remaining,
		}
	}

	// 写入 Mock TOS
	if err := s.mockTOS.PutObject(bucket, key, bytesReader(data)); err != nil {
		return nil, fmt.Errorf("tos upload: %w", err)
	}

	// 更新 SQLite 已用容量
	if err := s.store.AddObject(bucket, key, size); err != nil {
		// 回滚 Mock TOS
		s.mockTOS.DeleteObject(bucket, key)
		return nil, fmt.Errorf("update used capacity: %w", err)
	}

	updated := b.UsedCapacity + size
	return &model.UploadResponse{
		File:      key,
		Size:      size,
		UsedBytes: updated,
	}, nil
}

// GetQuota 查询配额使用情况
func (s *Service) GetQuota(name string) (*model.QuotaResponse, error) {
	b, err := s.store.GetBucket(name)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("bucket not found: %s", name)
	}

	remaining := b.QuotaCapacity - b.UsedCapacity
	usagePercent := 0.0
	if b.QuotaCapacity > 0 {
		usagePercent = float64(b.UsedCapacity) / float64(b.QuotaCapacity) * 100
	}

	return &model.QuotaResponse{
		Bucket:            b.Name,
		QuotaCapacity:     b.QuotaCapacity,
		UsedCapacity:      b.UsedCapacity,
		RemainingCapacity: remaining,
		UsagePercent:      usagePercent,
		QuotaReadBPS:      b.QuotaReadBPS,
		UsedReadBPS:       b.UsedReadBPS,
		QuotaWriteBPS:     b.QuotaWriteBPS,
		UsedWriteBPS:      b.UsedWriteBPS,
		QuotaReadIOPS:     b.QuotaReadIOPS,
		UsedReadIOPS:      b.UsedReadIOPS,
		QuotaWriteIOPS:    b.QuotaWriteIOPS,
		UsedWriteIOPS:     b.UsedWriteIOPS,
		ObjectCount:       b.ObjectCount,
	}, nil
}

// ListObjects 列出桶内对象
func (s *Service) ListObjects(bucket string) ([]model.ObjectMeta, error) {
	b, err := s.store.GetBucket(bucket)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("bucket not found: %s", bucket)
	}
	return s.store.ListObjects(bucket)
}

// DeleteObject 删除单个对象
func (s *Service) DeleteObject(bucket, key string) error {
	b, err := s.store.GetBucket(bucket)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("bucket not found: %s", bucket)
	}
	if err := s.mockTOS.DeleteObject(bucket, key); err != nil {
		return fmt.Errorf("tos delete object: %w", err)
	}
	if err := s.store.DeleteObject(bucket, key); err != nil {
		return fmt.Errorf("delete object record: %w", err)
	}
	return nil
}

// QuotaExceededError 容量配额超限错误
type QuotaExceededError struct {
	Size      int64
	Remaining int64
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("quota exceeded: file size %d bytes would exceed remaining quota %d bytes", e.Size, e.Remaining)
}

// bytesReader 将 []byte 转为 io.Reader
func bytesReader(data []byte) io.Reader {
	return &byteReader{data: data}
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
