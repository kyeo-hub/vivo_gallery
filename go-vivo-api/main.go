package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

var (
	crawler  *VivoCrawler
	cache    *Cache
	db       *DB
)

func main() {
	log.Println("🚀 Starting vivo gallery service...")

	// 初始化数据库
	db = NewDB("vivo.db")
	defer db.Close()

	// 初始化缓存
	cache = NewCache(5 * time.Minute)

	// 初始化爬虫
	userID := os.Getenv("VIVO_USER_ID")
	if userID == "" {
		log.Fatal("❌ VIVO_USER_ID not set")
	}
	crawler = NewVivoCrawler(userID, db)

	// 启动定时任务
	startCron()

	// 启动 API 服务
	startServer()
}

func startCron() {
	c := cron.New(cron.WithSeconds())
	
	// 每30分钟执行一次增量同步
	c.AddFunc("0 */30 * * * *", func() {
		log.Println("⏰ 开始定时同步...")
		if err := crawler.Sync(); err != nil {
			log.Printf("❌ 同步失败: %v", err)
		}
	})

	c.Start()
	log.Println("✅ 定时任务已启动 (每30分钟)")

	// 首次立即执行一次
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("🔄 首次同步...")
		if err := crawler.Sync(); err != nil {
			log.Printf("❌ 首次同步失败: %v", err)
		}
	}()
}

func startServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由组
	api := r.Group("/api/v1")
	{
		api.GET("/posts", getPosts)
		api.GET("/posts/:id", getPostDetail)
		api.GET("/sync", manualSync)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	log.Printf("✅ API Server running on :%s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// 获取帖子列表（带缓存）
func getPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	cacheKey := "posts:" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize)
	
	if data, ok := cache.Get(cacheKey); ok {
		c.JSON(200, gin.H{
			"data":     data,
			"cached":   true,
			"page":     page,
			"pageSize": pageSize,
		})
		return
	}

	posts, total, err := db.GetPosts(page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	result := gin.H{
		"data":     posts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}

	cache.Set(cacheKey, result, 5*time.Minute)
	c.JSON(200, result)
}

// 获取帖子详情（带缓存）
func getPostDetail(c *gin.Context) {
	postID := c.Param("id")
	cacheKey := "post:" + postID

	if data, ok := cache.Get(cacheKey); ok {
		c.JSON(200, gin.H{"data": data, "cached": true})
		return
	}

	post, err := db.GetPostWithImages(postID)
	if err != nil {
		c.JSON(404, gin.H{"error": "post not found"})
		return
	}

	cache.Set(cacheKey, post, 10*time.Minute)
	c.JSON(200, gin.H{"data": post})
}

// 手动触发同步
func manualSync(c *gin.Context) {
	go func() {
		if err := crawler.Sync(); err != nil {
			log.Printf("❌ 手动同步失败: %v", err)
		}
	}()
	c.JSON(202, gin.H{"message": "sync started"})
}
