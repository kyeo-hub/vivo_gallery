import os
from dotenv import load_dotenv
import requests
from sql import MySQLDatabase
import time
import logging

# 加载 .env 文件
load_dotenv()
class VivoGalleryDB:
    def __init__(self):
        self.userId = os.getenv('VIVO_USER_ID')
        self.host = os.getenv('DB_HOST')
        self.port = int(os.getenv('DB_PORT'))   
        self.database = os.getenv('DB_NAME')
        self.user = os.getenv('DB_USER')
        self.password =  os.getenv('DB_PASSWORD')
    
    def db_connect(self):
        try:
            self.db = MySQLDatabase(host=self.host, port=self.port, database=self.database, user=self.user, password=self.password)
            self.db.connect()
            self.conn = self.db.get_connection()
            self.cursor = self.conn.cursor()
            self.create_tables_if_not_exists()
        except Exception as e:
            logging.error(f"❌ 无法连接到数据库: {e}")
            return None
    def create_tables_if_not_exists(self):
        try:
            self.cursor.execute("""
                CREATE TABLE IF NOT EXISTS posts (
                    post_id VARCHAR(20) PRIMARY KEY,
                    title VARCHAR(255),
                    description TEXT,
                    user_nick VARCHAR(45),
                    signature VARCHAR(255),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            self.cursor.execute("""
                CREATE TABLE IF NOT EXISTS images (
                    image_id BIGINT(20) AUTO_INCREMENT PRIMARY KEY,
                    post_id VARCHAR(20),
                    url VARCHAR(2083),
                    FOREIGN KEY (post_id) REFERENCES posts(post_id),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            self.conn.commit()
            logging.info("✅ 表已创建或已存在。")
        except Exception as err:
            logging.error(f"❌ 创建表失败: {err}")
        
    def fetch_posts(self):
        pageNo = 1
        all_data = []

        while True:
            data = self.fetch_data(pageNo)
            if not data:
                break
            all_data.extend(data) 
            pageNo += 1
        if all_data:
            post_ids = [item['postId'] for item in all_data]  # 提取每个元素的 postId
            logging.info(f'✅找到 {len(post_ids)} 个相册。')
            return post_ids
        else:
            logging.info("❌ 没有发现相册.")
            return
        
    def fetch_data(self,pageNo):
        base_url = f'https://gallery.vivo.com.cn/gallery/wap/share/user/post/list/{self.userId}.do'
        current_time = int(time.time() * 1000)  # 获取当前时间的时间戳（毫秒级）
        params = {
            "dataFrom": 1,
            "pageNo": pageNo,
            "requestTime": current_time,
            "searchType": 4,
            "t": current_time
        }
        headers = {}
        try:
            with requests.Session() as session:
                response = session.get(base_url, headers=headers, params=params)
                response.raise_for_status()  # 检查请求是否成功
                rq = response.json()
                if "data" in rq:
                    data = rq["data"]['posts']
                    return data  # 假设返回的数据是 JSON 格式
                else:
                    return None
        except requests.exceptions.RequestException as e:
            logging.error(f"❌ 请求失败: {e}")
            return None
        
    def save_albums(self,post_ids):
        # 批量查询是否存在
        placeholders = ', '.join(['%s'] * len(post_ids))
        query = f"SELECT post_id FROM posts WHERE post_id IN ({placeholders})"
        self.cursor.execute(query, post_ids)
        existing_post_ids = [row[0] for row in self.cursor.fetchall()]
        
        url = "https://gallery.vivo.com.cn/gallery/wap/H5/post/getPostDetailById.do"
        headers = {
        'Content-Type': 'application/x-www-form-urlencoded'
        }
        
        for post_id in post_ids:
            if str(post_id) not in existing_post_ids:
                params = {
                    "postId": post_id
                }
                try:
                    with requests.Session() as session:
                        response = session.post(url, headers=headers, params=params)
                        response.raise_for_status()  # 检查请求是否成功
                        rq = response.json()
                        try:
                            post = rq["data"]['post']
                            post_id = post['postId']
                            title = post.get('postTitle',None)
                            description = post.get('postDesc',None)
                            user_nick = post.get('userNick',None)
                            signature = post.get('signature',None)
                            urls = post.get('images',[])
                            #数据库操作
                            self.cursor.execute("INSERT INTO posts (post_id, title, description,user_nick,signature) VALUES (%s, %s, %s,%s,%s)", (post_id, title, description,user_nick,signature))
                            for image_url in urls:
                                self.cursor.execute("INSERT INTO images (post_id, url) VALUES (%s, %s)", (post_id, image_url))
                            logging.info(f"✅ 新增相册: {post_id} ✅ 相册: {title} ✅ 相册描述: {description}🎉 ✅ 照片: {len(urls)}张🎉")
                            self.conn.commit()
                            
                        except Exception as e:
                            logging.error(f"❌ 处理数据失败: {e}")
                except requests.exceptions.RequestException as e:
                    logging.error(f"❌ 请求失败: {e}")
            else:
                logging.info(f"❗️ 相册 {post_id} 已存在，跳过。")

def main():
    vivo = VivoGalleryDB()
    vivo.db_connect()
    post_ids = vivo.fetch_posts()
    vivo.save_albums(post_ids)
    logging.info(f"📢 影相册已保存到数据库。")
    vivo.cursor.close()
    vivo.conn.close()
    vivo.db.disconnect()  
    logging.info(f"📢 数据库已关闭。") 
        
       

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
    main()
        