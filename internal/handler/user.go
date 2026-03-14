package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"all2wei/internal/config"
	"all2wei/internal/model"
	"all2wei/internal/repository"
	"all2wei/internal/utils"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	jwtCfg   *config.JWTConfig
}

func NewUserHandler(userRepo *repository.UserRepository, jwtCfg *config.JWTConfig) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req model.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	if h.userRepo.Exists(req.Username) {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	// 密码加密
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: hashedPassword,
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// 生成 token
	token, err := utils.GenerateToken(user.ID, user.Username, h.jwtCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, model.UserLoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req model.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找用户
	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 验证密码
	if !utils.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 生成 token
	token, err := utils.GenerateToken(user.ID, user.Username, h.jwtCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, model.UserLoginResponse{
		Token: token,
		User:  *user,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
