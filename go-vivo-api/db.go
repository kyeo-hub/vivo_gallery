package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type Post struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserNick    string    `json:"user_nick"`
	Signature   string    `json:"signature"`
	ImageCount  int       `json:"image_count"`
	Images      []string  `json:"images,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Image struct {
	ID        int64     `json:"id"`
	PostID    string    `json:"post_id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func NewDB(path string) *DB {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	db := &DB{conn: conn}
	db.initTables()
	return db
}

func (d *DB) initTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS posts (
			post_id TEXT PRIMARY KEY,
			title TEXT,
			description TEXT,
			user_nick TEXT,
			signature TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id TEXT,
			url TEXT UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(post_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_images_post_id ON images(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at)`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			log.Fatalf("❌ 创建表失败: %v", err)
		}
	}

	log.Println("✅ 数据库初始化完成")
}

// 保存帖子（带去重）
func (d *DB) SavePost(post *Post, images []string) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 检查是否已存在
	var exists bool
	err = tx.QueryRow("SELECT 1 FROM posts WHERE post_id = ?", post.ID).Scan(&exists)
	if err == nil {
		log.Printf("⏭️ 帖子 %s 已存在，跳过", post.ID)
		return nil
	}

	// 插入帖子
	_, err = tx.Exec(
		`INSERT INTO posts (post_id, title, description, user_nick, signature, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		post.ID, post.Title, post.Description, post.UserNick, post.Signature, time.Now(),
	)
	if err != nil {
		return err
	}

	// 插入图片（URL 唯一，自动去重）
	for _, url := range images {
		_, err = tx.Exec(
			"INSERT OR IGNORE INTO images (post_id, url) VALUES (?, ?)",
			post.ID, url,
		)
		if err != nil {
			log.Printf("⚠️ 插入图片失败 %s: %v", url, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("✅ 保存帖子: %s, 图片: %d张", post.ID, len(images))
	return nil
}

// 获取帖子列表
func (d *DB) GetPosts(page, pageSize int) ([]Post, int, error) {
	offset := (page - 1) * pageSize

	// 获取总数
	var total int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM posts").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	rows, err := d.conn.Query(
		`SELECT p.post_id, p.title, p.description, p.user_nick, p.signature, 
		        COUNT(i.id) as image_count, p.created_at
		 FROM posts p
		 LEFT JOIN images i ON p.post_id = i.post_id
		 GROUP BY p.post_id
		 ORDER BY p.created_at DESC
		 LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.UserNick, 
			&p.Signature, &p.ImageCount, &p.CreatedAt)
		if err != nil {
			continue
		}
		posts = append(posts, p)
	}

	return posts, total, nil
}

// 获取帖子详情（含图片）
func (d *DB) GetPostWithImages(postID string) (*Post, error) {
	var p Post
	err := d.conn.QueryRow(
		`SELECT post_id, title, description, user_nick, signature, created_at 
		 FROM posts WHERE post_id = ?`, postID,
	).Scan(&p.ID, &p.Title, &p.Description, &p.UserNick, &p.Signature, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	rows, err := d.conn.Query("SELECT url FROM images WHERE post_id = ? ORDER BY id", postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err == nil {
			p.Images = append(p.Images, url)
		}
	}
	p.ImageCount = len(p.Images)

	return &p, nil
}

// 获取所有已存在的 PostID（用于增量同步）
func (d *DB) GetExistingPostIDs() (map[string]bool, error) {
	rows, err := d.conn.Query("SELECT post_id FROM posts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			existing[id] = true
		}
	}
	return existing, nil
}

func (d *DB) Close() {
	d.conn.Close()
}
