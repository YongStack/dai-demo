package model

import "time"

// Bucket 代表一个存储桶及其全部配额信息
type Bucket struct {
	Name               string    `json:"bucket"`
	QuotaCapacity      int64     `json:"quota_capacity"`       // 容量配额上限（字节）
	QuotaReadBPS       int64     `json:"quota_read_bps"`        // 读带宽配额上限（字节/秒），0 表示不限
	QuotaWriteBPS      int64     `json:"quota_write_bps"`       // 写带宽配额上限（字节/秒），0 表示不限
	QuotaReadIOPS      int64     `json:"quota_read_iops"`       // 读 IOPS 配额上限（次/秒），0 表示不限
	QuotaWriteIOPS     int64     `json:"quota_write_iops"`      // 写 IOPS 配额上限（次/秒），0 表示不限
	UsedCapacity       int64     `json:"used_capacity"`        // 已用容量（字节）
	UsedReadBPS        int64     `json:"used_read_bps"`        // 当前读带宽使用量（字节/秒）
	UsedWriteBPS       int64     `json:"used_write_bps"`       // 当前写带宽使用量（字节/秒）
	UsedReadIOPS       int64     `json:"used_read_iops"`       // 当前读 IOPS 使用量（次/秒）
	UsedWriteIOPS      int64     `json:"used_write_iops"`      // 当前写 IOPS 使用量（次/秒）
	ObjectCount        int       `json:"object_count"`         // 对象数量
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateBucketRequest 创建存储桶请求
type CreateBucketRequest struct {
	Bucket          string `json:"bucket"          binding:"required"`
	QuotaCapacity   int64  `json:"quota_capacity"   binding:"required"`
	QuotaReadBPS    int64  `json:"quota_read_bps"`
	QuotaWriteBPS   int64  `json:"quota_write_bps"`
	QuotaReadIOPS   int64  `json:"quota_read_iops"`
	QuotaWriteIOPS  int64  `json:"quota_write_iops"`
}

// QuotaResponse 配额查询响应
type QuotaResponse struct {
	Bucket            string  `json:"bucket"`
	QuotaCapacity     int64   `json:"quota_capacity"`
	UsedCapacity      int64   `json:"used_capacity"`
	RemainingCapacity int64   `json:"remaining_capacity"`
	UsagePercent      float64 `json:"usage_percent"`
	QuotaReadBPS      int64   `json:"quota_read_bps"`
	UsedReadBPS       int64   `json:"used_read_bps"`
	QuotaWriteBPS     int64   `json:"quota_write_bps"`
	UsedWriteBPS      int64   `json:"used_write_bps"`
	QuotaReadIOPS     int64   `json:"quota_read_iops"`
	UsedReadIOPS      int64   `json:"used_read_iops"`
	QuotaWriteIOPS    int64   `json:"quota_write_iops"`
	UsedWriteIOPS     int64   `json:"used_write_iops"`
	ObjectCount       int     `json:"object_count"`
}

// ObjectMeta 对象元数据
type ObjectMeta struct {
	Key       string    `json:"key"`
	Size      int64     `json:"size"`
	Bucket    string    `json:"bucket"`
	CreatedAt time.Time `json:"created_at"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	File     string `json:"file"`
	Size     int64  `json:"size"`
	UsedBytes int64 `json:"used_bytes"`
}

// ListObjectsResponse 列出对象响应
type ListObjectsResponse struct {
	Bucket  string       `json:"bucket"`
	Objects []ObjectMeta `json:"objects"`
}

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}
