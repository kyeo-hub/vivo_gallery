import mariadb
import logging

class MySQLDatabase:
    def __init__(self, host, port, database, user, password,pool_size=5):
        self.host = host
        self.port = port
        self.database = database
        self.user = user
        self.password = password
        self.pool_size = pool_size
        self.pool = None
        logging.info(f"🔔初始化 MySQLDatabase: host={host}, port={port}, database={database}, user={user}")

    def connect(self):
       
        try:
             # 先尝试连接到 MySQL 服务器，不指定数据库
            self.pool = mariadb.ConnectionPool(
                pool_name="db_pool",
                pool_size=self.pool_size,
                host=self.host,
                port=self.port,
                user=self.user,
                password=self.password
            )
            logging.info("✅ MySQL 数据库连接池创建成功")

            # 检查并创建数据库
            self.create_database_if_not_exists()

            # 重新连接到指定的数据库
            self.pool = mariadb.ConnectionPool(
                pool_name="vivo_pool",
                pool_size=self.pool_size,
                host=self.host,
                port=self.port,
                database=self.database,
                user=self.user,
                password=self.password
            )
            logging.info(f"✅ 连接到数据库 {self.database} 成功")

        except mariadb.Error as e:
            logging.error(f"❌ 连接失败: {e}")
    
    def create_database_if_not_exists(self):
        try:
            # 获取连接并创建数据库
            conn = self.pool.get_connection()
            cursor = conn.cursor()
            cursor.execute(f"CREATE DATABASE IF NOT EXISTS {self.database}")
            logging.info(f"✅ 数据库 {self.database} 已创建或已存在。")
            cursor.close()
            conn.close()
        except mariadb.Error as err:
            logging.error(f"❌ 创建数据库失败: {err}")

    def disconnect(self):
        if self.pool:
            self.pool.close()
            logging.info("✅MySQL 数据库连接池已关闭")
    def get_connection(self):
        try:
            return self.pool.get_connection()
        except mariadb.Error as e:
            logging.error(f"获取连接失败: {e}")
            return None

    def execute_query(self, query, params=None):
        cursor = None
        try:
            cursor = self.connection.cursor()
            cursor.execute(query, params)
            self.connection.commit()
            print("查询执行成功")
        except mariadb.Error as e:
            print(f"执行查询失败: {e}")
        finally:
            if cursor:
                cursor.close()

    def fetch_all(self, query, params=None):
        cursor = None
        try:
            cursor = self.connection.cursor(dictionary=True)
            cursor.execute(query, params)
            result = cursor.fetchall()
            return result
        except mariadb.Error as e:
            print(f"获取数据失败: {e}")
            return None
        finally:
            if cursor:
                cursor.close()

    def fetch_one(self, query, params=None):
        cursor = None
        try:
            cursor = self.connection.cursor(dictionary=True)
            cursor.execute(query, params)
            result = cursor.fetchone()
            return result
        except mariadb.Error as e:
            print(f"获取单条数据失败: {e}")
            return None
        finally:
            if cursor:
                cursor.close()

