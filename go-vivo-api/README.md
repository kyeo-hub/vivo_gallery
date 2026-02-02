# Vivo Gallery API

vivo 相册自动同步服务，定时爬取数据保存到 SQLite，提供 REST API 查询。

## 特性

- ✅ 定时自动增量同步（每30分钟）
- ✅ 重复数据自动去重（基于 post_id 和 image url）
- ✅ 内存缓存，减少数据库查询
- ✅ RESTful API 接口
- ✅ 轻量级 Docker 部署
- ✅ 兼容已有 Caddy 反向代理

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填写 VIVO_USER_ID
```

### 2. 启动服务

```bash
docker-compose up -d
```

### 3. 查看日志

```bash
docker-compose logs -f
```

## API 接口文档

### 健康检查
```
GET /health

Response:
{
  "status": "ok"
}
```

### 获取帖子列表
```
GET /api/v1/posts?page=1&pageSize=20

Response:
{
  "data": [
    {
      "id": "123456",
      "title": "相册标题",
      "description": "相册描述",
      "user_nick": "用户名",
      "signature": "签名",
      "image_count": 5,
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "pageSize": 20
}
```

### 获取帖子详情
```
GET /api/v1/posts/:id

Response:
{
  "data": {
    "id": "123456",
    "title": "相册标题",
    "description": "相册描述",
    "user_nick": "用户名",
    "signature": "签名",
    "image_count": 5,
    "images": [
      "https://example.com/image1.jpg",
      "https://example.com/image2.jpg"
    ],
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 手动触发同步
```
GET /api/v1/sync

Response:
{
  "message": "sync started"
}
```

## 与已有 Caddy 集成

假设你的 Caddy 已经在运行，加入以下配置：

```caddyfile
vivo.yourdomain.com {
    reverse_proxy vivo-api:8080
}
```

确保 vivo-api 和 Caddy 在同一个 Docker 网络：

```bash
# 1. 创建共享网络（如果还没有）
docker network create vivo-network

# 2. 将 Caddy 加入网络
docker network connect vivo-network your-caddy-container

# 3. 重启 vivo-api
docker-compose up -d
```

## 数据备份

数据库文件挂载在 `./data/vivo.db`，直接备份该文件即可：

```bash
cp data/vivo.db backup/vivo-$(date +%Y%m%d).db
```

## 目录结构

```
.
├── main.go           # 服务入口
├── crawler.go        # vivo 爬虫逻辑
├── db.go             # SQLite 数据库操作
├── cache.go          # 内存缓存实现
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .dockerignore
├── Caddyfile.example
└── README.md
```

## 技术栈

- Go 1.21 + Gin
- SQLite3 (单文件数据库)
- Cron (定时任务)
- Docker / Docker Compose
