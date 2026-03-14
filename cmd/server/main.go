package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"all2wei/internal/config"
	"all2wei/internal/handler"
	"all2wei/internal/middleware"
	"all2wei/internal/model"
	"all2wei/internal/repository"
	"all2wei/internal/service"
	"all2wei/internal/utils"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := repository.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化存储服务
	var storageSvc service.StorageService
	var minioSvc *service.MinIOService // 保存 MinIO 服务引用用于同步

	if cfg.Storage.Type == "minio" {
		minioSvc, err = service.NewMinIOService(&cfg.MinIO)
		if err != nil {
			log.Printf("Warning: Failed to connect MinIO: %v, falling back to local storage", err)
			localSvc, err := service.NewLocalStorage("uploads", "/files")
			if err != nil {
				log.Fatalf("Failed to initialize local storage: %v", err)
			}
			storageSvc = localSvc
			// 尝试重新连接 MinIO（用于同步功能）
			minioSvc, _ = service.NewMinIOService(&cfg.MinIO)
		} else {
			storageSvc = minioSvc
			log.Println("Using MinIO storage")
		}
	} else {
		localSvc, err := service.NewLocalStorage("uploads", "/files")
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
		storageSvc = localSvc
		log.Println("Using local storage (uploads/)")
		// 即使使用本地存储，也尝试连接 MinIO 用于同步
		minioSvc, _ = service.NewMinIOService(&cfg.MinIO)
	}

	// 初始化 repositories
	userRepo := repository.NewUserRepository(db)
	materialRepo := repository.NewMaterialRepository(db)

	// 初始化默认账号
	if err := initDefaultUser(userRepo); err != nil {
		log.Printf("Warning: Failed to init default user: %v", err)
	}

	// 初始化 handlers
	userHandler := handler.NewUserHandler(userRepo, &cfg.JWT)
	materialHandler := handler.NewMaterialHandler(materialRepo, storageSvc, &cfg.JWT)
	materialHandler.SetMinIOService(minioSvc) // 设置 MinIO 服务用于同步

	// 设置路由
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// 本地文件服务
	r.Static("/files", "./uploads")

	// 公开路由
	api := r.Group("/api")
	{
		api.POST("/login", userHandler.Login)
	}

	// 公开 API（视频流，自己处理 token）
	api.GET("/materials/:id/stream", materialHandler.StreamVideo)

	// 需要认证的路由
	auth := api.Group("")
	auth.Use(middleware.AuthMiddleware(&cfg.JWT))
	{
		auth.GET("/profile", userHandler.GetProfile)

		// 学习资料相关
		auth.POST("/materials", materialHandler.Upload)
		auth.POST("/materials/sync", materialHandler.Sync) // 同步 MinIO
		auth.GET("/materials", materialHandler.List)
		auth.GET("/materials/:id", materialHandler.Get)
		auth.DELETE("/materials/:id", materialHandler.Delete)
		auth.GET("/materials/:id/subtitle", materialHandler.GetSubtitle)
	}

	// 静态文件服务（前端）
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.svg", "./web/dist/favicon.svg")
	r.StaticFile("/icons.svg", "./web/dist/icons.svg")
	
	// 首页
	r.GET("/", func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
	
	// SPA 路由 - 所有非 API 路由返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 如果是 API 请求但没找到路由，返回 404
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.File("./web/dist/index.html")
	})

	// 启动服务器
	log.Printf("Server starting on %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDefaultUser 初始化默认账号
func initDefaultUser(userRepo *repository.UserRepository) error {
	// 检查默认账号是否已存在
	if userRepo.Exists("all2wei") {
		return nil
	}

	// 创建默认账号
	hashedPassword, err := utils.HashPassword("all2wei")
	if err != nil {
		return err
	}

	user := &model.User{
		Username: "all2wei",
		Password: hashedPassword,
	}

	if err := userRepo.Create(user); err != nil {
		return err
	}

	log.Println("Default user created: all2wei / all2wei")
	return nil
}
