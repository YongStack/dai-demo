package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/YongStack/dai-demo/internal/model"
	"github.com/YongStack/dai-demo/internal/service"
)

// Handler 封装 HTTP 路由处理
type Handler struct {
	svc *service.Service
}

// NewHandler 创建 Handler 实例
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册所有路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/buckets", h.ListBuckets)
		api.POST("/buckets", h.CreateBucket)
		api.GET("/buckets/:bucket/quota", h.GetQuota)
		api.DELETE("/buckets/:bucket", h.DeleteBucket)
		api.POST("/buckets/:bucket/objects", h.UploadObject)
		api.GET("/buckets/:bucket/objects", h.ListObjects)
		api.DELETE("/buckets/:bucket/objects/:key", h.DeleteObject)
	}
}

// CreateBucket POST /api/buckets
func (h *Handler) CreateBucket(c *gin.Context) {
	var req model.CreateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body"})
		return
	}

	bucket, err := h.svc.CreateBucket(&req)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, bucket)
}

// ListBuckets GET /api/buckets
func (h *Handler) ListBuckets(c *gin.Context) {
	buckets, err := h.svc.ListBuckets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:  "internal server error",
			Detail: err.Error(),
		})
		return
	}
	if buckets == nil {
		buckets = []*model.Bucket{}
	}
	c.JSON(http.StatusOK, buckets)
}

// GetQuota GET /api/buckets/:bucket/quota
func (h *Handler) GetQuota(c *gin.Context) {
	bucket := c.Param("bucket")
	q, err := h.svc.GetQuota(bucket)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, q)
}

// DeleteBucket DELETE /api/buckets/:bucket
func (h *Handler) DeleteBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	if err := h.svc.DeleteBucket(bucket); err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UploadObject POST /api/buckets/:bucket/objects
func (h *Handler) UploadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "file field is required"})
		return
	}

	data := make([]byte, file.Size)
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:  "internal server error",
			Detail: err.Error(),
		})
		return
	}
	defer f.Close()

	if _, err := f.Read(data); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:  "internal server error",
			Detail: fmt.Sprintf("read file: %v", err),
		})
		return
	}

	resp, err := h.svc.UploadObject(bucket, file.Filename, file.Size, data)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListObjects GET /api/buckets/:bucket/objects
func (h *Handler) ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")
	objects, err := h.svc.ListObjects(bucket)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	if objects == nil {
		objects = []model.ObjectMeta{}
	}
	c.JSON(http.StatusOK, model.ListObjectsResponse{
		Bucket:  bucket,
		Objects: objects,
	})
}

// DeleteObject DELETE /api/buckets/:bucket/objects/:key
func (h *Handler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := c.Param("key")
	if err := h.svc.DeleteObject(bucket, key); err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleServiceError 统一处理 service 层错误并返回对应 HTTP 状态码
func (h *Handler) handleServiceError(c *gin.Context, err error) {
	var quotaErr *service.QuotaExceededError
	if errors.As(err, &quotaErr) {
		c.JSON(http.StatusRequestEntityTooLarge, model.ErrorResponse{Error: err.Error()})
		return
	}

	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "invalid bucket name"):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: msg})
	case strings.HasPrefix(msg, "invalid quota"):
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: msg})
	case strings.HasPrefix(msg, "bucket not found"):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: msg})
	case strings.HasPrefix(msg, "bucket already exists"):
		c.JSON(http.StatusConflict, model.ErrorResponse{Error: msg})
	case strings.HasPrefix(msg, "object not found"):
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: msg})
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:  "internal server error",
			Detail: msg,
		})
	}
}
