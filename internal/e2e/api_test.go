package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/YongStack/dai-demo/internal/handler"
	"github.com/YongStack/dai-demo/internal/mocktos"
	"github.com/YongStack/dai-demo/internal/model"
	"github.com/YongStack/dai-demo/internal/service"
	"github.com/YongStack/dai-demo/internal/store"
)

// setupTestServer 搭建完整的 E2E 测试服务器（handler → service → store → mocktos）
// 返回 httptest.Server 和清理函数
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
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

	svc := service.NewService(st, mockTOS)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewHandler(svc)
	h.RegisterRoutes(r)

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
		st.Close()
		os.RemoveAll(dir)
	}

	return server, cleanup
}

// ==================== 辅助函数 ====================

func doJSON(t *testing.T, server *httptest.Server, method, path string, body interface{}) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func uploadFile(t *testing.T, server *httptest.Server, bucket, filename string, content []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	fw.Write(content)
	w.Close()

	req, err := http.NewRequest("POST", server.URL+"/api/buckets/"+bucket+"/objects", &buf)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return data
}

func parseJSON(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	data := readBody(t, resp)
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("unmarshal response %q: %v", string(data), err)
	}
}

func createBucket(t *testing.T, server *httptest.Server, name string, quotaCapacity int64) *model.Bucket {
	t.Helper()
	req := &model.CreateBucketRequest{
		Bucket:         name,
		QuotaCapacity:  quotaCapacity,
		QuotaReadBPS:   10485760,
		QuotaWriteBPS:  10485760,
		QuotaReadIOPS:  1000,
		QuotaWriteIOPS: 1000,
	}
	resp := doJSON(t, server, "POST", "/api/buckets", req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(readBody(t, resp)))
	}
	var b model.Bucket
	parseJSON(t, resp, &b)
	return &b
}

// ==================== E2E: 创建存储桶 ====================

func TestE2E_CreateBucket_Success(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := &model.CreateBucketRequest{
		Bucket:         "my-bucket",
		QuotaCapacity:  1073741824,
		QuotaReadBPS:   10485760,
		QuotaWriteBPS:  10485760,
		QuotaReadIOPS:  1000,
		QuotaWriteIOPS: 1000,
	}
	resp := doJSON(t, server, "POST", "/api/buckets", req)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var b model.Bucket
	parseJSON(t, resp, &b)

	if b.Name != "my-bucket" {
		t.Errorf("expected bucket name 'my-bucket', got %q", b.Name)
	}
	if b.QuotaCapacity != 1073741824 {
		t.Errorf("expected quota_capacity 1073741824, got %d", b.QuotaCapacity)
	}
	if b.QuotaReadBPS != 10485760 {
		t.Errorf("expected quota_read_bps 10485760, got %d", b.QuotaReadBPS)
	}
	if b.QuotaWriteBPS != 10485760 {
		t.Errorf("expected quota_write_bps 10485760, got %d", b.QuotaWriteBPS)
	}
	if b.QuotaReadIOPS != 1000 {
		t.Errorf("expected quota_read_iops 1000, got %d", b.QuotaReadIOPS)
	}
	if b.QuotaWriteIOPS != 1000 {
		t.Errorf("expected quota_write_iops 1000, got %d", b.QuotaWriteIOPS)
	}
	if b.UsedCapacity != 0 {
		t.Errorf("expected used_capacity 0, got %d", b.UsedCapacity)
	}
	if b.ObjectCount != 0 {
		t.Errorf("expected object_count 0, got %d", b.ObjectCount)
	}
}

func TestE2E_CreateBucket_InvalidName(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []string{
		"INVALID_NAME",
		"ab",
		"my_bucket",
		"-bucket",
		"bucket-",
	}

	for _, name := range tests {
		req := &model.CreateBucketRequest{
			Bucket:        name,
			QuotaCapacity: 1073741824,
		}
		resp := doJSON(t, server, "POST", "/api/buckets", req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bucket name %q, got %d", name, resp.StatusCode)
		}
		var errResp model.ErrorResponse
		parseJSON(t, resp, &errResp)
		if errResp.Error != "invalid bucket name" {
			t.Errorf("expected 'invalid bucket name', got %q", errResp.Error)
		}
	}
}

func TestE2E_CreateBucket_InvalidQuota(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// quota_capacity = -1: 通过 Gin binding 校验后由 service 层拒绝
	req := &model.CreateBucketRequest{
		Bucket:        "valid-bucket",
		QuotaCapacity: -1,
	}
	resp := doJSON(t, server, "POST", "/api/buckets", req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for quota -1, got %d", resp.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp, &errResp)
	if errResp.Error != "invalid quota limit" {
		t.Errorf("expected 'invalid quota limit', got %q", errResp.Error)
	}

	// quota_capacity = 0: Gin binding:"required" 将零值视为缺失，返回 "invalid request body"
	req2 := &model.CreateBucketRequest{
		Bucket:        "valid-bucket-2",
		QuotaCapacity: 0,
	}
	resp2 := doJSON(t, server, "POST", "/api/buckets", req2)
	if resp2.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for quota 0, got %d", resp2.StatusCode)
	}
	var errResp2 model.ErrorResponse
	parseJSON(t, resp2, &errResp2)
	if errResp2.Error != "invalid request body" {
		t.Errorf("expected 'invalid request body' for quota 0, got %q", errResp2.Error)
	}
}

func TestE2E_CreateBucket_Duplicate(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "dup-bucket", 1073741824)

	req := &model.CreateBucketRequest{
		Bucket:        "dup-bucket",
		QuotaCapacity: 1073741824,
	}
	resp := doJSON(t, server, "POST", "/api/buckets", req)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp, &errResp)
	if errResp.Error != "bucket already exists: dup-bucket" {
		t.Errorf("expected 'bucket already exists: dup-bucket', got %q", errResp.Error)
	}
}

// ==================== E2E: 查询配额 ====================

func TestE2E_GetQuota_Success(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "quota-bucket", 1073741824)

	// 上传 100MB 文件
	data := make([]byte, 104857600)
	resp := uploadFile(t, server, "quota-bucket", "data.bin", data)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload failed: %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 查询配额
	qResp := doJSON(t, server, "GET", "/api/buckets/quota-bucket/quota", nil)
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", qResp.StatusCode)
	}
	var q model.QuotaResponse
	parseJSON(t, qResp, &q)

	if q.Bucket != "quota-bucket" {
		t.Errorf("expected bucket 'quota-bucket', got %q", q.Bucket)
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
	// 验证带宽和 IOPS 配额也被正确持久化和返回
	if q.QuotaReadBPS != 10485760 {
		t.Errorf("expected read bps 10485760, got %d", q.QuotaReadBPS)
	}
	if q.QuotaWriteBPS != 10485760 {
		t.Errorf("expected write bps 10485760, got %d", q.QuotaWriteBPS)
	}
	if q.QuotaReadIOPS != 1000 {
		t.Errorf("expected read iops 1000, got %d", q.QuotaReadIOPS)
	}
	if q.QuotaWriteIOPS != 1000 {
		t.Errorf("expected write iops 1000, got %d", q.QuotaWriteIOPS)
	}
	if q.ObjectCount != 1 {
		t.Errorf("expected object_count 1, got %d", q.ObjectCount)
	}
}

func TestE2E_GetQuota_NotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp := doJSON(t, server, "GET", "/api/buckets/nonexistent/quota", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp, &errResp)
	if errResp.Error != "bucket not found: nonexistent" {
		t.Errorf("expected 'bucket not found: nonexistent', got %q", errResp.Error)
	}
}

// ==================== E2E: 文件上传与配额校验 ====================

func TestE2E_UploadObject_Success(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "upload-bucket", 1073741824)

	data := make([]byte, 1048576) // 1MB
	resp := uploadFile(t, server, "upload-bucket", "test.txt", data)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(readBody(t, resp)))
	}

	var ur model.UploadResponse
	parseJSON(t, resp, &ur)

	if ur.File != "test.txt" {
		t.Errorf("expected file 'test.txt', got %q", ur.File)
	}
	if ur.Size != 1048576 {
		t.Errorf("expected size 1048576, got %d", ur.Size)
	}
	if ur.UsedBytes != 1048576 {
		t.Errorf("expected used_bytes 1048576, got %d", ur.UsedBytes)
	}
}

func TestE2E_UploadObject_QuotaExceeded(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "quota-exceeded-bucket", 1048576) // 1MB quota

	// 上传占满配额
	data := make([]byte, 1048576)
	resp := uploadFile(t, server, "quota-exceeded-bucket", "fill.txt", data)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first upload failed: %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 再上传应超限
	data2 := make([]byte, 1024)
	resp2 := uploadFile(t, server, "quota-exceeded-bucket", "overflow.txt", data2)
	if resp2.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp2.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp2, &errResp)
	expectedMsg := "quota exceeded: file size 1024 bytes would exceed remaining quota 0 bytes"
	if errResp.Error != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, errResp.Error)
	}
}

func TestE2E_UploadObject_Boundary(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "boundary-bucket", 1048576) // 1MB quota

	// 上传恰好等于配额大小 — 应成功
	data := make([]byte, 1048576)
	resp := uploadFile(t, server, "boundary-bucket", "exact.txt", data)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("boundary upload should succeed: %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 再上传 1 字节 — 应失败
	resp2 := uploadFile(t, server, "boundary-bucket", "over.txt", []byte("x"))
	if resp2.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for 1 byte over, got %d", resp2.StatusCode)
	}
	readBody(t, resp2)
}

func TestE2E_UploadObject_BucketNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp := uploadFile(t, server, "nonexistent", "test.txt", []byte("data"))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp, &errResp)
	if errResp.Error != "bucket not found: nonexistent" {
		t.Errorf("expected 'bucket not found: nonexistent', got %q", errResp.Error)
	}
}

func TestE2E_UploadObject_MultipleAccumulate(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "multi-bucket", 3145728) // 3MB quota

	// 上传 3 个 1MB 文件 — 应成功
	for i := 0; i < 3; i++ {
		data := make([]byte, 1048576)
		name := string(rune('a'+i)) + ".txt"
		resp := uploadFile(t, server, "multi-bucket", name, data)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("upload %s failed: %d", name, resp.StatusCode)
		}
		readBody(t, resp)
	}

	// 第 4 个文件 — 应超限
	resp := uploadFile(t, server, "multi-bucket", "d.txt", []byte("x"))
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 on 4th upload, got %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 查询配额确认累计值
	qResp := doJSON(t, server, "GET", "/api/buckets/multi-bucket/quota", nil)
	var q model.QuotaResponse
	parseJSON(t, qResp, &q)
	if q.UsedCapacity != 3145728 {
		t.Errorf("expected used 3145728, got %d", q.UsedCapacity)
	}
	if q.ObjectCount != 3 {
		t.Errorf("expected 3 objects, got %d", q.ObjectCount)
	}
}

// ==================== E2E: 删除存储桶 ====================

func TestE2E_DeleteBucket_Success(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "del-bucket", 1073741824)

	resp := doJSON(t, server, "DELETE", "/api/buckets/del-bucket", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// 确认已删除 — 查询应返回 404
	qResp := doJSON(t, server, "GET", "/api/buckets/del-bucket/quota", nil)
	if qResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", qResp.StatusCode)
	}
}

func TestE2E_DeleteBucket_NotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp := doJSON(t, server, "DELETE", "/api/buckets/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	var errResp model.ErrorResponse
	parseJSON(t, resp, &errResp)
	if errResp.Error != "bucket not found: nonexistent" {
		t.Errorf("expected 'bucket not found: nonexistent', got %q", errResp.Error)
	}
}

// ==================== E2E: 列出所有桶 ====================

func TestE2E_ListBuckets_Empty(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp := doJSON(t, server, "GET", "/api/buckets", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var buckets []*model.Bucket
	parseJSON(t, resp, &buckets)
	if len(buckets) != 0 {
		t.Errorf("expected empty list, got %d buckets", len(buckets))
	}
}

func TestE2E_ListBuckets_WithBuckets(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "bucket-a", 1073741824)
	createBucket(t, server, "bucket-b", 2147483648)

	resp := doJSON(t, server, "GET", "/api/buckets", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var buckets []*model.Bucket
	parseJSON(t, resp, &buckets)
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}

	names := map[string]bool{}
	for _, b := range buckets {
		names[b.Name] = true
	}
	if !names["bucket-a"] || !names["bucket-b"] {
		t.Errorf("expected bucket-a and bucket-b in list, got %v", names)
	}
}

// ==================== E2E: 对象管理 ====================

func TestE2E_ListObjects(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "list-obj-bucket", 1073741824)

	// 上传两个文件
	uploadFile(t, server, "list-obj-bucket", "file1.txt", []byte("hello"))
	uploadFile(t, server, "list-obj-bucket", "file2.txt", []byte("world"))

	resp := doJSON(t, server, "GET", "/api/buckets/list-obj-bucket/objects", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var lr model.ListObjectsResponse
	parseJSON(t, resp, &lr)

	if lr.Bucket != "list-obj-bucket" {
		t.Errorf("expected bucket 'list-obj-bucket', got %q", lr.Bucket)
	}
	if len(lr.Objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(lr.Objects))
	}

	keys := map[string]bool{}
	for _, o := range lr.Objects {
		keys[o.Key] = true
	}
	if !keys["file1.txt"] || !keys["file2.txt"] {
		t.Errorf("expected file1.txt and file2.txt, got %v", keys)
	}
}

func TestE2E_ListObjects_BucketNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	resp := doJSON(t, server, "GET", "/api/buckets/nonexistent/objects", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestE2E_DeleteObject_Success(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	createBucket(t, server, "del-obj-bucket", 1073741824)

	data := make([]byte, 1048576)
	uploadFile(t, server, "del-obj-bucket", "test.txt", data)

	// 删除对象
	resp := doJSON(t, server, "DELETE", "/api/buckets/del-obj-bucket/objects/test.txt", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// 验证容量恢复
	qResp := doJSON(t, server, "GET", "/api/buckets/del-obj-bucket/quota", nil)
	var q model.QuotaResponse
	parseJSON(t, qResp, &q)
	if q.UsedCapacity != 0 {
		t.Errorf("expected used 0 after delete, got %d", q.UsedCapacity)
	}
	if q.ObjectCount != 0 {
		t.Errorf("expected 0 objects after delete, got %d", q.ObjectCount)
	}
}

// ==================== E2E: 完整主流程 ====================

func TestE2E_FullWorkflow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. 创建桶
	createBucket(t, server, "workflow-bucket", 10485760) // 10MB

	// 2. 上传文件 — 应成功
	data1 := make([]byte, 4194304) // 4MB
	resp := uploadFile(t, server, "workflow-bucket", "data1.bin", data1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload data1 failed: %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 3. 查询配额 — 验证已用量
	qResp := doJSON(t, server, "GET", "/api/buckets/workflow-bucket/quota", nil)
	var q model.QuotaResponse
	parseJSON(t, qResp, &q)
	if q.UsedCapacity != 4194304 {
		t.Errorf("expected used 4194304, got %d", q.UsedCapacity)
	}
	if q.RemainingCapacity != 6291456 {
		t.Errorf("expected remaining 6291456, got %d", q.RemainingCapacity)
	}

	// 4. 上传第二个文件 — 6MB，应成功（总计 10MB，恰好）
	data2 := make([]byte, 6291456) // 6MB
	resp2 := uploadFile(t, server, "workflow-bucket", "data2.bin", data2)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("upload data2 failed: %d", resp2.StatusCode)
	}
	readBody(t, resp2)

	// 5. 上传第三个文件 — 应超限
	data3 := make([]byte, 1)
	resp3 := uploadFile(t, server, "workflow-bucket", "data3.bin", data3)
	if resp3.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for third upload, got %d", resp3.StatusCode)
	}
	readBody(t, resp3)

	// 6. 删除第二个文件 — 容量恢复
	resp4 := doJSON(t, server, "DELETE", "/api/buckets/workflow-bucket/objects/data2.bin", nil)
	if resp4.StatusCode != http.StatusNoContent {
		t.Fatalf("delete data2 failed: %d", resp4.StatusCode)
	}

	// 7. 再次查询配额 — 容量应恢复到 4MB
	qResp2 := doJSON(t, server, "GET", "/api/buckets/workflow-bucket/quota", nil)
	var q2 model.QuotaResponse
	parseJSON(t, qResp2, &q2)
	if q2.UsedCapacity != 4194304 {
		t.Errorf("expected used 4194304 after delete, got %d", q2.UsedCapacity)
	}
	if q2.ObjectCount != 1 {
		t.Errorf("expected 1 object after delete, got %d", q2.ObjectCount)
	}

	// 8. 现在可以再上传 — 应成功
	resp5 := uploadFile(t, server, "workflow-bucket", "data3.bin", data3)
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("upload data3 after delete failed: %d", resp5.StatusCode)
	}
	readBody(t, resp5)

	// 9. 列出所有桶
	listResp := doJSON(t, server, "GET", "/api/buckets", nil)
	var buckets []*model.Bucket
	parseJSON(t, listResp, &buckets)
	if len(buckets) != 1 {
		t.Errorf("expected 1 bucket in list, got %d", len(buckets))
	}

	// 10. 删除桶
	delResp := doJSON(t, server, "DELETE", "/api/buckets/workflow-bucket", nil)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete bucket failed: %d", delResp.StatusCode)
	}

	// 11. 确认桶已删除
	listResp2 := doJSON(t, server, "GET", "/api/buckets", nil)
	var buckets2 []*model.Bucket
	parseJSON(t, listResp2, &buckets2)
	if len(buckets2) != 0 {
		t.Errorf("expected 0 buckets after delete, got %d", len(buckets2))
	}
}

// ==================== E2E: 带宽和 IOPS 配额展示 ====================

func TestE2E_BandwidthIOPS_QuotaDisplay(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// 创建桶时设置带宽和 IOPS 配额
	req := &model.CreateBucketRequest{
		Bucket:         "bw-iops-bucket",
		QuotaCapacity:  1073741824,
		QuotaReadBPS:   52428800,  // 50 MB/s
		QuotaWriteBPS:  26214400,  // 25 MB/s
		QuotaReadIOPS:  5000,
		QuotaWriteIOPS: 3000,
	}
	resp := doJSON(t, server, "POST", "/api/buckets", req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	readBody(t, resp)

	// 查询配额 — 验证带宽和 IOPS 配额正确返回
	qResp := doJSON(t, server, "GET", "/api/buckets/bw-iops-bucket/quota", nil)
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", qResp.StatusCode)
	}
	var q model.QuotaResponse
	parseJSON(t, qResp, &q)

	if q.QuotaReadBPS != 52428800 {
		t.Errorf("expected read bps 52428800, got %d", q.QuotaReadBPS)
	}
	if q.QuotaWriteBPS != 26214400 {
		t.Errorf("expected write bps 26214400, got %d", q.QuotaWriteBPS)
	}
	if q.QuotaReadIOPS != 5000 {
		t.Errorf("expected read iops 5000, got %d", q.QuotaReadIOPS)
	}
	if q.QuotaWriteIOPS != 3000 {
		t.Errorf("expected write iops 3000, got %d", q.QuotaWriteIOPS)
	}
}

// ==================== E2E: 无效请求体 ====================

func TestE2E_CreateBucket_InvalidJSON(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req, _ := http.NewRequest("POST", server.URL+"/api/buckets", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	readBody(t, resp)
}
