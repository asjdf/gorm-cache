package cache

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 联合主键测试模型
type UserRole struct {
	UserID int64  `gorm:"primaryKey;column:user_id"`
	RoleID int64  `gorm:"primaryKey;column:role_id"`
	Name   string `gorm:"column:name"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// Unique键测试模型
type User struct {
	ID       uint   `gorm:"primaryKey;column:id"`
	Email    string `gorm:"uniqueIndex:idx_email;column:email;size:255"`
	Username string `gorm:"uniqueIndex:idx_username;column:username;size:100"`
	Name     string `gorm:"column:name;size:255"`
}

func (User) TableName() string {
	return "users"
}

// 联合Unique键测试模型
type UserSession struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    int64     `gorm:"uniqueIndex:idx_user_token;column:user_id"`
	Token     string    `gorm:"uniqueIndex:idx_user_token;column:token;size:255"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}

var (
	mysqlPool        *dockertest.Pool
	mysqlResource    *dockertest.Resource
	mysqlDB          *gorm.DB
	mysqlDSN         string
	setupMySQLOnce   sync.Once
	cleanupMySQLOnce sync.Once

	pgPool        *dockertest.Pool
	pgResource    *dockertest.Resource
	pgDB          *gorm.DB
	pgDSN         string
	setupPGOnce   sync.Once
	cleanupPGOnce sync.Once
)

func setupMySQL(t *testing.T) *gorm.DB {
	setupMySQLOnce.Do(func() {
		var err error
		mysqlPool, err = dockertest.NewPool("")
		if err != nil {
			t.Fatalf("Could not connect to docker: %s", err)
		}

		mysqlResource, err = mysqlPool.RunWithOptions(&dockertest.RunOptions{
			Repository: "mysql",
			Tag:        "8.0",
			Env: []string{
				"MYSQL_ROOT_PASSWORD=testpass",
				"MYSQL_DATABASE=testdb",
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3306/tcp": {{HostPort: "0"}},
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
		if err != nil {
			t.Fatalf("Could not start MySQL resource: %s", err)
		}

		host := mysqlResource.GetHostPort("3306/tcp")
		mysqlDSN = fmt.Sprintf("root:testpass@tcp(%s)/testdb?charset=utf8mb4&parseTime=True&loc=Local", host)

		// Wait for MySQL to be ready
		mysqlPool.MaxWait = 120 * time.Second
		if err := mysqlPool.Retry(func() error {
			var err error
			mysqlDB, err = gorm.Open(mysql.Open(mysqlDSN), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				return err
			}
			sqlDB, err := mysqlDB.DB()
			if err != nil {
				return err
			}
			// 设置连接池参数
			sqlDB.SetMaxOpenConns(10)
			sqlDB.SetMaxIdleConns(5)
			sqlDB.SetConnMaxLifetime(time.Hour)
			return sqlDB.Ping()
		}); err != nil {
			t.Fatalf("Could not connect to MySQL: %s", err)
		}
	})

	// Clean tables before each test
	mysqlDB.Exec("DROP TABLE IF EXISTS user_roles, users, user_sessions")

	// Auto migrate
	err := mysqlDB.AutoMigrate(&UserRole{}, &User{}, &UserSession{})
	if err != nil {
		t.Fatalf("Auto migrate error: %v", err)
	}

	return mysqlDB
}

func setupPostgreSQL(t *testing.T) *gorm.DB {
	setupPGOnce.Do(func() {
		var err error
		pgPool, err = dockertest.NewPool("")
		if err != nil {
			t.Fatalf("Could not connect to docker: %s", err)
		}

		pgResource, err = pgPool.RunWithOptions(&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "15-alpine",
			Env: []string{
				"POSTGRES_PASSWORD=testpass",
				"POSTGRES_DB=testdb",
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"5432/tcp": {{HostPort: "0"}},
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
		if err != nil {
			t.Fatalf("Could not start PostgreSQL resource: %s", err)
		}

		hostPort := pgResource.GetHostPort("5432/tcp")
		// hostPort 格式是 "host:port"，需要解析出端口
		host, port, err := net.SplitHostPort(hostPort)
		if err != nil {
			t.Fatalf("Could not parse host:port: %s", err)
		}
		// 如果 host 是 IPv6 地址，需要用方括号
		if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
			host = "[" + host + "]"
		}
		pgDSN = fmt.Sprintf("host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=disable", host, port)

		// Wait for PostgreSQL to be ready
		pgPool.MaxWait = 120 * time.Second
		if err := pgPool.Retry(func() error {
			var err error
			pgDB, err = gorm.Open(postgres.Open(pgDSN), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				return err
			}
			sqlDB, err := pgDB.DB()
			if err != nil {
				return err
			}
			// 设置连接池参数
			sqlDB.SetMaxOpenConns(10)
			sqlDB.SetMaxIdleConns(5)
			sqlDB.SetConnMaxLifetime(time.Hour)
			return sqlDB.Ping()
		}); err != nil {
			t.Fatalf("Could not connect to PostgreSQL: %s", err)
		}
	})

	// Clean tables before each test
	pgDB.Exec("DROP TABLE IF EXISTS user_roles, users, user_sessions CASCADE")

	// Auto migrate
	err := pgDB.AutoMigrate(&UserRole{}, &User{}, &UserSession{})
	if err != nil {
		t.Fatalf("Auto migrate error: %v", err)
	}

	return pgDB
}

func TestMain(m *testing.M) {
	code := m.Run()

	// Cleanup
	cleanupMySQLOnce.Do(func() {
		if mysqlDB != nil {
			sqlDB, _ := mysqlDB.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
		if mysqlResource != nil && mysqlPool != nil {
			_ = mysqlPool.Purge(mysqlResource)
		}
	})

	cleanupPGOnce.Do(func() {
		if pgDB != nil {
			sqlDB, _ := pgDB.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
		if pgResource != nil && pgPool != nil {
			_ = pgPool.Purge(pgResource)
		}
	})

	os.Exit(code)
}

// 辅助函数：创建标准缓存配置
func createTestCache(tables []string) (Cache, error) {
	return NewGorm2Cache(&config.CacheConfig{
		CacheLevel:           config.CacheLevelAll,
		CacheStorage:         storage.NewMem(storage.DefaultMemStoreConfig),
		InvalidateWhenUpdate: true,
		CacheTTL:             5000,
		CacheMaxItemCnt:      100,
		DebugMode:            false,
		Tables:               tables,
	})
}

// 测试联合主键 - MySQL
func TestCompositePrimaryKey_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"user_roles"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	userRoles := []UserRole{
		{UserID: 1, RoleID: 1, Name: "Admin"},
		{UserID: 1, RoleID: 2, Name: "User"},
		{UserID: 2, RoleID: 1, Name: "Admin"},
	}
	if err := db.Create(&userRoles).Error; err != nil {
		t.Fatalf("Failed to create user roles: %v", err)
	}

	// 第一次查询 - 应该从数据库读取并缓存
	var result1 []UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Find(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result1) != 1 || result1[0].Name != "Admin" {
		t.Errorf("Expected 1 result with name 'Admin', got %d results", len(result1))
	}

	// 第二次查询 - 应该从缓存读取
	var result2 []UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Find(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result2) != 1 || result2[0].Name != "Admin" {
		t.Errorf("Expected 1 result with name 'Admin', got %d results", len(result2))
	}

	// 测试IN查询
	var result3 []UserRole
	if err := db.Where("user_id = ? AND role_id IN (?)", 1, []int64{1, 2}).Find(&result3).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result3) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result3))
	}

	// 更新数据 - 应该失效缓存
	if err := db.Model(&UserRole{}).Where("user_id = ? AND role_id = ?", 1, 1).Update("name", "SuperAdmin").Error; err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// 再次查询 - 应该从数据库读取新数据
	var result4 []UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Find(&result4).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result4) != 1 || result4[0].Name != "SuperAdmin" {
		t.Errorf("Expected name 'SuperAdmin', got %s", result4[0].Name)
	}
}

// 测试联合主键 - PostgreSQL
func TestCompositePrimaryKey_PostgreSQL(t *testing.T) {
	db := setupPostgreSQL(t)

	cache, err := createTestCache([]string{"user_roles"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	userRoles := []UserRole{
		{UserID: 1, RoleID: 1, Name: "Admin"},
		{UserID: 1, RoleID: 2, Name: "User"},
		{UserID: 2, RoleID: 1, Name: "Admin"},
	}
	if err := db.Create(&userRoles).Error; err != nil {
		t.Fatalf("Failed to create user roles: %v", err)
	}

	// 第一次查询
	var result1 []UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Find(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result1) != 1 || result1[0].Name != "Admin" {
		t.Errorf("Expected 1 result with name 'Admin', got %d results", len(result1))
	}

	// 第二次查询 - 应该从缓存读取
	var result2 []UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Find(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if len(result2) != 1 || result2[0].Name != "Admin" {
		t.Errorf("Expected 1 result with name 'Admin', got %d results", len(result2))
	}
}

// 测试Unique键 - MySQL
func TestUniqueKey_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"users"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	users := []User{
		{Email: "user1@example.com", Username: "user1", Name: "User 1"},
		{Email: "user2@example.com", Username: "user2", Name: "User 2"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("Failed to create users: %v", err)
	}

	// 第一次查询 - 通过email unique键查询
	var result1 User
	if err := db.Where("email = ?", "user1@example.com").First(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result1.Name != "User 1" {
		t.Errorf("Expected name 'User 1', got %s", result1.Name)
	}

	// 第二次查询 - 应该从缓存读取
	var result2 User
	if err := db.Where("email = ?", "user1@example.com").First(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result2.Name != "User 1" {
		t.Errorf("Expected name 'User 1', got %s", result2.Name)
	}

	// 通过username unique键查询
	var result3 User
	if err := db.Where("username = ?", "user2").First(&result3).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result3.Name != "User 2" {
		t.Errorf("Expected name 'User 2', got %s", result3.Name)
	}
}

// 测试联合Unique键 - MySQL
func TestCompositeUniqueKey_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := NewGorm2Cache(&config.CacheConfig{
		CacheLevel:           config.CacheLevelAll,
		CacheStorage:         storage.NewMem(storage.DefaultMemStoreConfig),
		InvalidateWhenUpdate: true,
		CacheTTL:             5000,
		CacheMaxItemCnt:      100,
		DebugMode:            false,
		Tables:               []string{"user_sessions"},
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	sessions := []UserSession{
		{UserID: 1, Token: "token1", ExpiresAt: time.Now().Add(24 * time.Hour)},
		{UserID: 2, Token: "token2", ExpiresAt: time.Now().Add(24 * time.Hour)},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("Failed to create sessions: %v", err)
	}

	// 第一次查询 - 通过联合unique键查询
	var result1 UserSession
	if err := db.Where("user_id = ? AND token = ?", 1, "token1").First(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result1.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", result1.UserID)
	}

	// 第二次查询 - 应该从缓存读取
	var result2 UserSession
	if err := db.Where("user_id = ? AND token = ?", 1, "token1").First(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result2.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", result2.UserID)
	}
}

// 测试Unique键 - PostgreSQL
func TestUniqueKey_PostgreSQL(t *testing.T) {
	db := setupPostgreSQL(t)

	cache, err := NewGorm2Cache(&config.CacheConfig{
		CacheLevel:           config.CacheLevelAll,
		CacheStorage:         storage.NewMem(storage.DefaultMemStoreConfig),
		InvalidateWhenUpdate: true,
		CacheTTL:             5000,
		CacheMaxItemCnt:      100,
		DebugMode:            false,
		Tables:               []string{"users"},
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	users := []User{
		{Email: "user1@example.com", Username: "user1", Name: "User 1"},
		{Email: "user2@example.com", Username: "user2", Name: "User 2"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("Failed to create users: %v", err)
	}

	// 第一次查询
	var result1 User
	if err := db.Where("email = ?", "user1@example.com").First(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result1.Name != "User 1" {
		t.Errorf("Expected name 'User 1', got %s", result1.Name)
	}

	// 第二次查询 - 应该从缓存读取
	var result2 User
	if err := db.Where("email = ?", "user1@example.com").First(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result2.Name != "User 1" {
		t.Errorf("Expected name 'User 1', got %s", result2.Name)
	}
}

// 测试缓存失效 - MySQL
func TestCacheInvalidation_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"user_roles", "users"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	userRole := UserRole{UserID: 1, RoleID: 1, Name: "Admin"}
	if err := db.Create(&userRole).Error; err != nil {
		t.Fatalf("Failed to create user role: %v", err)
	}

	// 第一次查询 - 缓存
	var result1 UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).First(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// 更新数据
	if err := db.Model(&UserRole{}).Where("user_id = ? AND role_id = ?", 1, 1).Update("name", "SuperAdmin").Error; err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// 再次查询 - 应该获取新数据
	var result2 UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).First(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if result2.Name != "SuperAdmin" {
		t.Errorf("Expected name 'SuperAdmin', got %s", result2.Name)
	}

	// 删除数据
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).Delete(&UserRole{}).Error; err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// 查询已删除的数据 - 应该返回错误
	var result3 UserRole
	if err := db.Where("user_id = ? AND role_id = ?", 1, 1).First(&result3).Error; err == nil {
		t.Error("Expected error for deleted record, got nil")
	}
}

// 测试缓存统计 - MySQL
func TestCacheStats_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"users"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	// 创建测试数据
	user := User{Email: "test@example.com", Username: "test", Name: "Test User"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 第一次查询 - 应该从数据库读取
	var result1 User
	if err := db.Where("id = ?", user.ID).First(&result1).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// 第二次查询 - 应该从缓存读取
	var result2 User
	if err := db.Where("id = ?", user.ID).First(&result2).Error; err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// 验证查询结果正确
	if result1.ID != user.ID || result2.ID != user.ID {
		t.Errorf("Expected user ID %d, got result1=%d, result2=%d", user.ID, result1.ID, result2.ID)
	}
	if result1.Name != "Test User" || result2.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got result1=%s, result2=%s", result1.Name, result2.Name)
	}

	// 检查统计 - 至少应该有查询发生
	lookupCount := cache.LookupCount()
	if lookupCount == 0 {
		t.Errorf("Expected at least 1 lookup, got 0")
	}
}

// 测试缓存一致性 - 综合测试（创建、更新、删除、联合主键、Unique键）
func TestCacheConsistency_Comprehensive_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"user_roles", "users"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	t.Run("CreateAndQuery", func(t *testing.T) {
		// 创建数据后立即查询
		userRole := UserRole{UserID: 10, RoleID: 20, Name: "NewRole"}
		if err := db.Create(&userRole).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		var result UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 10, 20).First(&result).Error; err != nil {
			t.Fatalf("Failed to query after create: %v", err)
		}
		if result.Name != "NewRole" || result.UserID != 10 || result.RoleID != 20 {
			t.Errorf("Expected NewRole(10,20), got %s(%d,%d)", result.Name, result.UserID, result.RoleID)
		}
	})

	t.Run("UpdateCompositeKey", func(t *testing.T) {
		// 联合主键更新
		userRole := UserRole{UserID: 100, RoleID: 200, Name: "Original"}
		if err := db.Create(&userRole).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		// 查询缓存
		db.Where("user_id = ? AND role_id = ?", 100, 200).First(&UserRole{})

		// 更新
		if err := db.Model(&UserRole{}).Where("user_id = ? AND role_id = ?", 100, 200).Update("name", "Updated").Error; err != nil {
			t.Fatalf("Failed to update: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// 验证一致性
		var result UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 100, 200).First(&result).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if result.Name != "Updated" {
			t.Errorf("Expected 'Updated', got '%s'", result.Name)
		}

		// 验证数据库
		sqlDB, _ := db.DB()
		var dbName string
		row := sqlDB.QueryRow("SELECT name FROM user_roles WHERE user_id = ? AND role_id = ?", 100, 200)
		if err := row.Scan(&dbName); err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}
		if result.Name != dbName {
			t.Errorf("Cache inconsistency: cache='%s', db='%s'", result.Name, dbName)
		}
	})

	t.Run("UpdateUniqueKey", func(t *testing.T) {
		// Unique键更新
		user := User{Email: "unique@test.com", Username: "unique", Name: "Original"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		// 通过unique键查询缓存
		db.Where("email = ?", "unique@test.com").First(&User{})

		// 更新
		if err := db.Model(&User{}).Where("id = ?", user.ID).Update("name", "Updated").Error; err != nil {
			t.Fatalf("Failed to update: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// 验证
		var result User
		if err := db.Where("email = ?", "unique@test.com").First(&result).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if result.Name != "Updated" {
			t.Errorf("Expected 'Updated', got '%s'", result.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// 删除测试
		userRole := UserRole{UserID: 200, RoleID: 300, Name: "ToDelete"}
		if err := db.Create(&userRole).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		// 查询缓存
		db.Where("user_id = ? AND role_id = ?", 200, 300).First(&UserRole{})

		// 删除
		if err := db.Where("user_id = ? AND role_id = ?", 200, 300).Delete(&UserRole{}).Error; err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// 验证已删除
		var result UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 200, 300).First(&result).Error; err == nil {
			t.Error("Expected error for deleted record")
		} else if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("MultipleUpdates", func(t *testing.T) {
		// 多次更新
		userRole := UserRole{UserID: 300, RoleID: 400, Name: "V1"}
		if err := db.Create(&userRole).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		updates := []string{"V2", "V3", "V4"}
		for _, newName := range updates {
			if err := db.Model(&UserRole{}).Where("user_id = ? AND role_id = ?", 300, 400).Update("name", newName).Error; err != nil {
				t.Fatalf("Failed to update: %v", err)
			}
			time.Sleep(50 * time.Millisecond)

			var result UserRole
			if err := db.Where("user_id = ? AND role_id = ?", 300, 400).First(&result).Error; err != nil {
				t.Fatalf("Failed to query: %v", err)
			}
			if result.Name != newName {
				t.Errorf("Expected '%s', got '%s'", newName, result.Name)
			}
		}
	})
}

// 测试缓存一致性 - 高级场景（批量更新、并发、删除重建、Unique键交叉查询）
func TestCacheConsistency_Advanced_MySQL(t *testing.T) {
	db := setupMySQL(t)

	cache, err := createTestCache([]string{"user_roles", "users"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	db.Use(cache)

	t.Run("BatchUpdate", func(t *testing.T) {
		// 批量更新
		userRoles := []UserRole{
			{UserID: 700, RoleID: 701, Name: "Role1"},
			{UserID: 700, RoleID: 702, Name: "Role2"},
		}
		if err := db.Create(&userRoles).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		db.Where("user_id = ?", 700).Find(&[]UserRole{})

		if err := db.Model(&UserRole{}).Where("user_id = ?", 700).Update("name", "BatchUpdated").Error; err != nil {
			t.Fatalf("Failed to batch update: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var results []UserRole
		if err := db.Where("user_id = ?", 700).Find(&results).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		for _, result := range results {
			if result.Name != "BatchUpdated" {
				t.Errorf("Expected 'BatchUpdated', got '%s'", result.Name)
			}
		}
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		// 并发读写
		userRole := UserRole{UserID: 1000, RoleID: 2000, Name: "Initial"}
		if err := db.Create(&userRole).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		const numGoroutines = 5
		const numUpdates = 3
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < numUpdates; j++ {
					newName := fmt.Sprintf("Update-%d-%d", id, j)
					db.Model(&UserRole{}).Where("user_id = ? AND role_id = ?", 1000, 2000).Update("name", newName)
					time.Sleep(10 * time.Millisecond)
				}
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		time.Sleep(200 * time.Millisecond)

		var result UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 1000, 2000).First(&result).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		sqlDB, _ := db.DB()
		var dbName string
		row := sqlDB.QueryRow("SELECT name FROM user_roles WHERE user_id = ? AND role_id = ?", 1000, 2000)
		if err := row.Scan(&dbName); err != nil {
			t.Fatalf("Failed to query database: %v", err)
		}
		if result.Name != dbName {
			t.Errorf("Cache inconsistency: cache='%s', db='%s'", result.Name, dbName)
		}
	})

	t.Run("DeleteAndRecreate", func(t *testing.T) {
		// 删除后重新创建
		userRole1 := UserRole{UserID: 2000, RoleID: 3000, Name: "First"}
		if err := db.Create(&userRole1).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		db.Where("user_id = ? AND role_id = ?", 2000, 3000).First(&UserRole{})

		if err := db.Where("user_id = ? AND role_id = ?", 2000, 3000).Delete(&UserRole{}).Error; err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var result2 UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 2000, 3000).First(&result2).Error; err == nil {
			t.Error("Expected error for deleted record")
		}

		userRole2 := UserRole{UserID: 2000, RoleID: 3000, Name: "Second"}
		if err := db.Create(&userRole2).Error; err != nil {
			t.Fatalf("Failed to recreate: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var result3 UserRole
		if err := db.Where("user_id = ? AND role_id = ?", 2000, 3000).First(&result3).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if result3.Name != "Second" {
			t.Errorf("Expected 'Second', got '%s'", result3.Name)
		}
	})

	t.Run("UniqueKeyCrossQuery", func(t *testing.T) {
		// Unique键交叉查询
		user := User{Email: "cross@test.com", Username: "crossuser", Name: "Original"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("Failed to create: %v", err)
		}

		db.Where("email = ?", "cross@test.com").First(&User{})
		db.Where("username = ?", "crossuser").First(&User{})

		if err := db.Model(&User{}).Where("id = ?", user.ID).Update("name", "Updated").Error; err != nil {
			t.Fatalf("Failed to update: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		var result3, result4 User
		if err := db.Where("email = ?", "cross@test.com").First(&result3).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if err := db.Where("username = ?", "crossuser").First(&result4).Error; err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		if result3.Name != "Updated" || result4.Name != "Updated" {
			t.Errorf("Expected 'Updated', got email='%s', username='%s'", result3.Name, result4.Name)
		}
		if result3.Name != result4.Name {
			t.Errorf("Inconsistent results: email='%s', username='%s'", result3.Name, result4.Name)
		}
	})
}
