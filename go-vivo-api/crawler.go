package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type VivoCrawler struct {
	userID string
	db     *DB
	client *http.Client
}

type VivoPost struct {
	PostID    string   `json:"postId"`
	Title     string   `json:"postTitle"`
	Desc      string   `json:"postDesc"`
	UserNick  string   `json:"userNick"`
	Signature string   `json:"signature"`
	Images    []string `json:"images"`
}

func NewVivoCrawler(userID string, db *DB) *VivoCrawler {
	return &VivoCrawler{
		userID: userID,
		db:     db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 主同步逻辑
func (c *VivoCrawler) Sync() error {
	log.Println("🔄 开始同步 vivo 相册...")
	start := time.Now()

	// 获取已存在的帖子ID
	existing, err := c.db.GetExistingPostIDs()
	if err != nil {
		return fmt.Errorf("获取已有数据失败: %w", err)
	}

	// 获取所有帖子ID
	postIDs, err := c.fetchPostIDs()
	if err != nil {
		return fmt.Errorf("获取帖子列表失败: %w", err)
	}

	log.Printf("📊 发现 %d 个帖子，已有 %d 个", len(postIDs), len(existing))

	// 过滤出新帖子
	var newIDs []string
	for _, id := range postIDs {
		if !existing[id] {
			newIDs = append(newIDs, id)
		}
	}

	if len(newIDs) == 0 {
		log.Println("✅ 没有新数据需要同步")
		return nil
	}

	log.Printf("🆕 需要同步 %d 个新帖子", len(newIDs))

	// 获取详情并保存
	success := 0
	for i, id := range newIDs {
		post, err := c.fetchPostDetail(id)
		if err != nil {
			log.Printf("❌ 获取帖子 %s 失败: %v", id, err)
			continue
		}

		// 转换并保存
		dbPost := &Post{
			ID:          post.PostID,
			Title:       post.Title,
			Description: post.Desc,
			UserNick:    post.UserNick,
			Signature:   post.Signature,
		}

		if err := c.db.SavePost(dbPost, post.Images); err != nil {
			log.Printf("❌ 保存帖子 %s 失败: %v", id, err)
			continue
		}

		success++
		time.Sleep(500 * time.Millisecond) // 限速，避免请求过快

		if (i+1)%10 == 0 {
			log.Printf("📈 进度: %d/%d", i+1, len(newIDs))
		}
	}

	elapsed := time.Since(start)
	log.Printf("✅ 同步完成: 成功 %d/%d, 耗时 %v", success, len(newIDs), elapsed)
	return nil
}

// 获取帖子列表（分页获取所有）
func (c *VivoCrawler) fetchPostIDs() ([]string, error) {
	var allIDs []string
	pageNo := 1

	for {
		ids, hasMore, err := c.fetchPage(pageNo)
		if err != nil {
			return nil, err
		}

		allIDs = append(allIDs, ids...)
		log.Printf("📄 第 %d 页: %d 个帖子", pageNo, len(ids))

		if !hasMore || len(ids) == 0 {
			break
		}

		pageNo++
		time.Sleep(200 * time.Millisecond)
	}

	return allIDs, nil
}

// 获取单页帖子
func (c *VivoCrawler) fetchPage(pageNo int) ([]string, bool, error) {
	url := fmt.Sprintf("https://gallery.vivo.com.cn/gallery/wap/share/user/post/list/%s.do", c.userID)
	
	timestamp := time.Now().UnixMilli()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, err
	}

	q := req.URL.Query()
	q.Add("dataFrom", "1")
	q.Add("pageNo", fmt.Sprintf("%d", pageNo))
	q.Add("requestTime", fmt.Sprintf("%d", timestamp))
	q.Add("searchType", "4")
	q.Add("t", fmt.Sprintf("%d", timestamp))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}

	var result struct {
		Data struct {
			Posts []struct {
				PostID string `json:"postId"`
			} `json:"posts"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false, fmt.Errorf("解析失败: %w", err)
	}

	var ids []string
	for _, p := range result.Data.Posts {
		ids = append(ids, p.PostID)
	}

	hasMore := len(ids) > 0
	return ids, hasMore, nil
}

// 获取帖子详情
func (c *VivoCrawler) fetchPostDetail(postID string) (*VivoPost, error) {
	url := "https://gallery.vivo.com.cn/gallery/wap/H5/post/getPostDetailById.do"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("postId", postID)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Post VivoPost `json:"post"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析详情失败: %w", err)
	}

	return &result.Data.Post, nil
}
