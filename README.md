# 对象存储 Quota 管控 Demo

一个在线网站，用于对象存储资源的 quota 管控，支持**容量**、**读写带宽**、**读写 IOPS** 三类配额维度的展示、配置和管控。

## 技术栈

- **后端**: Go 1.26+ / Gin / SQLite (modernc.org/sqlite 纯 Go 驱动)
- **前端**: HTML / CSS / JavaScript (嵌入式静态资源)
- **对象存储**: Mock 模式（本地文件系统模拟 TOS 操作，无需真实 TOS 凭据）

## 功能特性

### 配额维度
- **容量配额**: 上传前预校验 `已用量 + 文件大小 ≤ 容量上限`，超限拒绝上传
- **读写带宽配额**: 读带宽 (Read BPS) / 写带宽 (Write BPS) 配额设置与展示
- **读写 IOPS 配额**: 读 IOPS / 写 IOPS 配额设置与展示

### Web UI 功能
- 存储桶列表展示与创建（支持设置三类配额维度）
- 配额使用详情面板（容量进度条、带宽/IOPS 展示）
- 文件上传与对象管理
- 存储桶删除

### REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/buckets` | 创建存储桶并设置配额 |
| GET | `/api/buckets` | 列出所有桶 |
| GET | `/api/buckets/:bucket/quota` | 查询配额使用情况 |
| DELETE | `/api/buckets/:bucket` | 删除存储桶 |
| POST | `/api/buckets/:bucket/objects` | 上传文件（容量配额校验） |
| GET | `/api/buckets/:bucket/objects` | 列出桶内对象 |
| DELETE | `/api/buckets/:bucket/objects/:key` | 删除单个对象 |

## 快速开始

```bash
# 启动服务
go run main.go

# 访问 Web UI
open http://localhost:8080

# 或通过 REST API 操作
curl -X POST http://localhost:8080/api/buckets \
  -H "Content-Type: application/json" \
  -d '{"bucket":"my-bucket","quota_capacity":1073741824,"quota_read_bps":10485760,"quota_write_bps":10485760,"quota_read_iops":1000,"quota_write_iops":1000}'

curl http://localhost:8080/api/buckets/my-bucket/quota
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `ADDR` | `:8080` | 监听地址 |
| `DATA_DIR` | `./data` | 数据目录（SQLite + Mock TOS 文件） |

## 运行测试

```bash
go test ./... -v
```

## 项目结构

```
├── main.go                      # 入口
├── internal/
│   ├── model/model.go           # 数据模型
│   ├── store/sqlite.go          # SQLite 持久化层
│   ├── mocktos/mock.go          # Mock 对象存储
│   ├── service/service.go       # 业务逻辑层
│   ├── service/service_test.go  # 单元测试
│   └── handler/handler.go       # HTTP handler
├── web/
│   ├── index.html               # 前端页面
│   ├── style.css                # 样式
│   └── app.js                   # 前端逻辑
├── go.mod
└── go.sum
```
