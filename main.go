package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/YongStack/dai-demo/internal/handler"
	"github.com/YongStack/dai-demo/internal/mocktos"
	"github.com/YongStack/dai-demo/internal/service"
	"github.com/YongStack/dai-demo/internal/store"
)

//go:embed web
var webFS embed.FS

func main() {
	// 数据目录
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// 初始化 SQLite
	dbPath := filepath.Join(dataDir, "quota.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer st.Close()

	// 初始化 Mock TOS
	tosDir := filepath.Join(dataDir, "mock-tos")
	mockTOS, err := mocktos.NewMockTOS(tosDir)
	if err != nil {
		log.Fatalf("init mock tos: %v", err)
	}

	// 初始化 Service
	svc := service.NewService(st, mockTOS)

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 注册 API 路由
	h := handler.NewHandler(svc)
	h.RegisterRoutes(r)

	// 嵌入式前端静态资源
	webSub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("web fs: %v", err)
	}
	r.StaticFS("/static", http.FS(webSub))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Printf("Object Storage Quota Demo starting on http://localhost%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
