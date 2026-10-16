package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YongStack/dai-demo/internal/mocktos"
	"github.com/YongStack/dai-demo/internal/model"
	"github.com/YongStack/dai-demo/internal/store"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	tosDir := filepath.Join(dir, "mock-tos")
	mockTOS, err := mocktos.NewMockTOS(tosDir)
	if err != nil {
		t.Fatalf("create mock tos: %v", err)
	}
	return NewService(st, mockTOS)
}

// ==================== 桶名校验测试 ====================

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"my-bucket", true},
		{"bucket-123", true},
		{"ab", false},        // 太短
		{"AB", false},         // 大写
		{"my_bucket", false},  // 下划线
		{"my.bucket", false},  // 点
		{"-bucket", false},    // 开头连字符
		{"bucket-", false},    // 结尾连字符
		{string(make([]byte, 64)), false}, // 太长（全零字节）
		{"a1b", true},         // 3 字符
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBucketName(tt.name)
			if got != tt.want {
				t.Errorf("ValidateBucketName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// ==================== 创建桶测试 ====================

func TestCreateBucket_Success(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:         "test-bucket",
		QuotaCapacity:  1073741824,
		QuotaReadBPS:   1048576,
		QuotaWriteBPS:  1048576,
		QuotaReadIOPS:  1000,
		QuotaWriteIOPS: 1000,
	}
	bucket, err := svc.CreateBucket(req)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if bucket.Name != "test-bucket" {
		t.Errorf("expected name 'test-bucket', got %q", bucket.Name)
	}
	if bucket.QuotaCapacity != 1073741824 {
		t.Errorf("expected quota 1073741824, got %d", bucket.QuotaCapacity)
	}
	if bucket.QuotaReadBPS != 1048576 {
		t.Errorf("expected read bps 1048576, got %d", bucket.QuotaReadBPS)
	}
	if bucket.QuotaReadIOPS != 1000 {
		t.Errorf("expected read iops 1000, got %d", bucket.QuotaReadIOPS)
	}
	if time.Since(bucket.CreatedAt) > 5*time.Second {
		t.Error("created_at not recent")
	}
}

func TestCreateBucket_InvalidName(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "INVALID_NAME",
		QuotaCapacity: 1073741824,
	}
	_, err := svc.CreateBucket(req)
	if err == nil || err.Error() != "invalid bucket name" {
		t.Errorf("expected invalid bucket name error, got %v", err)
	}
}

func TestCreateBucket_InvalidQuota(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "valid-bucket",
		QuotaCapacity: -1,
	}
	_, err := svc.CreateBucket(req)
	if err == nil || err.Error() != "invalid quota limit" {
		t.Errorf("expected invalid quota limit error, got %v", err)
	}
}

func TestCreateBucket_Duplicate(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "dup-bucket",
		QuotaCapacity: 1073741824,
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreateBucket(req)
	if err == nil || err.Error() != "bucket already exists: dup-bucket" {
		t.Errorf("expected duplicate error, got %v", err)
	}
}

// ==================== 删除桶测试 ====================

func TestDeleteBucket_Success(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "del-bucket",
		QuotaCapacity: 1073741824,
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.DeleteBucket("del-bucket"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// 确认已删除
	b, err := svc.GetBucket("del-bucket")
	if err == nil || b != nil {
		t.Error("expected bucket not found after deletion")
	}
}

func TestDeleteBucket_NotFound(t *testing.T) {
	svc := setupTestService(t)
	err := svc.DeleteBucket("nonexistent")
	if err == nil || err.Error() != "bucket not found: nonexistent" {
		t.Errorf("expected not found error, got %v", err)
	}
}

// ==================== 配额校验测试 ====================

func TestUploadObject_Success(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "upload-bucket",
		QuotaCapacity: 1073741824, // 1 GB
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	data := make([]byte, 1048576) // 1 MB
	resp, err := svc.UploadObject("upload-bucket", "test.txt", int64(len(data)), data)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if resp.Size != 1048576 {
		t.Errorf("expected size 1048576, got %d", resp.Size)
	}
	if resp.UsedBytes != 1048576 {
		t.Errorf("expected used 1048576, got %d", resp.UsedBytes)
	}
}

func TestUploadObject_QuotaExceeded(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "quota-bucket",
		QuotaCapacity: 1048576, // 1 MB
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 先上传占满配额
	data := make([]byte, 1048576)
	if _, err := svc.UploadObject("quota-bucket", "fill.txt", int64(len(data)), data); err != nil {
		t.Fatalf("first upload: %v", err)
	}

	// 再上传应超限
	data2 := make([]byte, 1024)
	_, err := svc.UploadObject("quota-bucket", "overflow.txt", int64(len(data2)), data2)
	if err == nil {
		t.Fatal("expected quota exceeded error")
	}
	qe, ok := err.(*QuotaExceededError)
	if !ok {
		t.Fatalf("expected *QuotaExceededError, got %T: %v", err, err)
	}
	if qe.Remaining != 0 {
		t.Errorf("expected remaining 0, got %d", qe.Remaining)
	}
}

func TestUploadObject_Boundary(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "boundary-bucket",
		QuotaCapacity: 1048576, // 1 MB
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 上传恰好等于配额大小 — 应成功
	data := make([]byte, 1048576)
	resp, err := svc.UploadObject("boundary-bucket", "exact.txt", int64(len(data)), data)
	if err != nil {
		t.Fatalf("boundary upload should succeed: %v", err)
	}
	if resp.UsedBytes != 1048576 {
		t.Errorf("expected used 1048576, got %d", resp.UsedBytes)
	}

	// 再上传 1 字节 — 应失败
	_, err = svc.UploadObject("boundary-bucket", "over.txt", 1, []byte("x"))
	if err == nil {
		t.Fatal("expected quota exceeded for 1 byte over")
	}
}

func TestUploadObject_BucketNotFound(t *testing.T) {
	svc := setupTestService(t)
	_, err := svc.UploadObject("nonexistent", "test.txt", 100, []byte("data"))
	if err == nil || err.Error() != "bucket not found: nonexistent" {
		t.Errorf("expected bucket not found, got %v", err)
	}
}

// ==================== 配额查询测试 ====================

func TestGetQuota_Success(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "quota-query-bucket",
		QuotaCapacity: 1073741824,
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	data := make([]byte, 104857600) // 100 MB
	if _, err := svc.UploadObject("quota-query-bucket", "data.bin", int64(len(data)), data); err != nil {
		t.Fatalf("upload: %v", err)
	}

	q, err := svc.GetQuota("quota-query-bucket")
	if err != nil {
		t.Fatalf("get quota: %v", err)
	}
	if q.QuotaCapacity != 1073741824 {
		t.Errorf("expected quota 1073741824, got %d", q.QuotaCapacity)
	}
	if q.UsedCapacity != 104857600 {
		t.Errorf("expected used 104857600, got %d", q.UsedCapacity)
	}
	if q.RemainingCapacity != 968884224 {
		t.Errorf("expected remaining 968884224, got %d", q.RemainingCapacity)
	}
	expectedPercent := 9.77
	if q.UsagePercent < expectedPercent-0.01 || q.UsagePercent > expectedPercent+0.01 {
		t.Errorf("expected ~%.2f%%, got %.2f%%", expectedPercent, q.UsagePercent)
	}
}

func TestGetQuota_NotFound(t *testing.T) {
	svc := setupTestService(t)
	_, err := svc.GetQuota("nonexistent")
	if err == nil || err.Error() != "bucket not found: nonexistent" {
		t.Errorf("expected not found, got %v", err)
	}
}

// ==================== 列出桶测试 ====================

func TestListBuckets(t *testing.T) {
	svc := setupTestService(t)

	// 空列表
	buckets, err := svc.ListBuckets()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("expected empty list, got %d", len(buckets))
	}

	// 创建两个桶
	for _, name := range []string{"bucket-a", "bucket-b"} {
		req := &model.CreateBucketRequest{
			Bucket:        name,
			QuotaCapacity: 1073741824,
		}
		if _, err := svc.CreateBucket(req); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	buckets, err = svc.ListBuckets()
	if err != nil {
		t.Fatalf("list after create: %v", err)
	}
	if len(buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(buckets))
	}
}

// ==================== 对象删除测试 ====================

func TestDeleteObject_Success(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "del-obj-bucket",
		QuotaCapacity: 1073741824,
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	data := make([]byte, 1048576)
	if _, err := svc.UploadObject("del-obj-bucket", "test.txt", int64(len(data)), data); err != nil {
		t.Fatalf("upload: %v", err)
	}

	if err := svc.DeleteObject("del-obj-bucket", "test.txt"); err != nil {
		t.Fatalf("delete object: %v", err)
	}

	// 容量应恢复
	q, err := svc.GetQuota("del-obj-bucket")
	if err != nil {
		t.Fatalf("quota: %v", err)
	}
	if q.UsedCapacity != 0 {
		t.Errorf("expected used 0 after delete, got %d", q.UsedCapacity)
	}
	if q.ObjectCount != 0 {
		t.Errorf("expected 0 objects, got %d", q.ObjectCount)
	}
}

// ==================== 多对象上传与配额累计测试 ====================

func TestMultipleUploads_AccumulateUsage(t *testing.T) {
	svc := setupTestService(t)
	req := &model.CreateBucketRequest{
		Bucket:        "multi-bucket",
		QuotaCapacity: 3145728, // 3 MB
	}
	if _, err := svc.CreateBucket(req); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 上传 3 个 1MB 文件 — 应成功
	for i := 0; i < 3; i++ {
		data := make([]byte, 1048576)
		_, err := svc.UploadObject("multi-bucket", string(rune('a'+i))+".txt", 1048576, data)
		if err != nil {
			t.Fatalf("upload %d: %v", i, err)
		}
	}

	// 第 4 个文件 — 应超限
	data := make([]byte, 1)
	_, err := svc.UploadObject("multi-bucket", "d.txt", 1, data)
	if err == nil {
		t.Fatal("expected quota exceeded on 4th upload")
	}

	q, _ := svc.GetQuota("multi-bucket")
	if q.UsedCapacity != 3145728 {
		t.Errorf("expected used 3145728, got %d", q.UsedCapacity)
	}
	if q.ObjectCount != 3 {
		t.Errorf("expected 3 objects, got %d", q.ObjectCount)
	}
}

// 确保编译时引用 os
var _ = os.Getenv
