package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"all2wei/internal/config"
	"all2wei/internal/model"
	"all2wei/internal/repository"
	"all2wei/internal/service"
	"all2wei/internal/utils"
)

// 视频扩展名
var videoExts = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mkv":  true,
	".mov":  true,
	".webm": true,
	".flv":  true,
	".wmv":  true,
	".m4v":  true,
}

// 字幕扩展名
var subtitleExts = map[string]bool{
	".srt": true,
	".vtt": true,
}

type MaterialHandler struct {
	materialRepo *repository.MaterialRepository
	storageSvc   service.StorageService
	minioSvc     *service.MinIOService // 可选，用于同步功能
	jwtCfg       *config.JWTConfig
}

func NewMaterialHandler(materialRepo *repository.MaterialRepository, storageSvc service.StorageService, jwtCfg *config.JWTConfig) *MaterialHandler {
	return &MaterialHandler{
		materialRepo: materialRepo,
		storageSvc:   storageSvc,
		jwtCfg:       jwtCfg,
	}
}

// SetMinIOService 设置 MinIO 服务（用于同步功能）
func (h *MaterialHandler) SetMinIOService(minioSvc *service.MinIOService) {
	h.minioSvc = minioSvc
}

// Upload 上传视频和字幕
func (h *MaterialHandler) Upload(c *gin.Context) {
	userID := c.GetUint("userID")

	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	description := c.PostForm("description")

	// 获取视频文件
	videoFile, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file is required"})
		return
	}

	// 保存视频到临时文件
	videoTempPath := filepath.Join("uploads", fmt.Sprintf("video_%d_%d%s", userID, time.Now().Unix(), filepath.Ext(videoFile.Filename)))
	if err := c.SaveUploadedFile(videoFile, videoTempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save video"})
		return
	}

	// 上传视频到存储
	videoKey := fmt.Sprintf("users/%d/videos/%d_%s", userID, time.Now().Unix(), videoFile.Filename)
	ctx := c.Request.Context()
	if err := h.storageSvc.UploadFile(ctx, videoKey, videoTempPath, videoFile.Header.Get("Content-Type")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload video"})
		return
	}

	// 处理字幕文件（可选）
	var subtitleKey string
	subtitleFile, err := c.FormFile("subtitle")
	if err == nil && subtitleFile != nil {
		subtitleTempPath := filepath.Join("uploads", fmt.Sprintf("subtitle_%d_%d%s", userID, time.Now().Unix(), filepath.Ext(subtitleFile.Filename)))
		if err := c.SaveUploadedFile(subtitleFile, subtitleTempPath); err == nil {
			subtitleKey = fmt.Sprintf("users/%d/subtitles/%d_%s", userID, time.Now().Unix(), subtitleFile.Filename)
			h.storageSvc.UploadFile(ctx, subtitleKey, subtitleTempPath, subtitleFile.Header.Get("Content-Type"))
		}
	}

	// 创建数据库记录
	material := &model.Material{
		UserID:      userID,
		Title:       title,
		Description: description,
		VideoKey:    videoKey,
		SubtitleKey: subtitleKey,
	}

	if err := h.materialRepo.Create(material); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create material record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": material.ID, "message": "uploaded successfully"})
}

// List 获取用户的学习资料列表
func (h *MaterialHandler) List(c *gin.Context) {
	userID := c.GetUint("userID")

	materials, err := h.materialRepo.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get materials"})
		return
	}

	// 生成预签名 URL（优先使用 MinIO）
	ctx := c.Request.Context()
	var responses []gin.H
	for _, m := range materials {
		var videoURL string
		var err error
		if h.minioSvc != nil {
			videoURL, err = h.minioSvc.GetPresignedURL(ctx, m.VideoKey, 24*time.Hour)
			if err != nil {
				log.Printf("[List] MinIO URL generation failed for video %s: %v", m.VideoKey, err)
			}
		}
		if videoURL == "" {
			videoURL, _ = h.storageSvc.GetPresignedURL(ctx, m.VideoKey, 24*time.Hour)
			if videoURL == "" {
				log.Printf("[List] Failed to generate video URL for material %d, videoKey=%s", m.ID, m.VideoKey)
			}
		}
		responses = append(responses, gin.H{
			"id":           m.ID,
			"title":        m.Title,
			"description":  m.Description,
			"video_url":    videoURL,
			"has_subtitle": m.SubtitleKey != "",
			"created_at":   m.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// Get 获取单个学习资料详情
func (h *MaterialHandler) Get(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	material, err := h.materialRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
		return
	}

	// 检查权限
	if material.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 生成预签名 URL（优先使用 MinIO）
	ctx := c.Request.Context()
	var videoURL, subtitleURL string
	
	log.Printf("[Get] Generating URLs for material %d, videoKey=%s, subtitleKey=%s, minioSvc=%v", material.ID, material.VideoKey, material.SubtitleKey, h.minioSvc != nil)
	
	if h.minioSvc != nil {
		var err error
		videoURL, err = h.minioSvc.GetPresignedURL(ctx, material.VideoKey, 24*time.Hour)
		if err != nil {
			log.Printf("[Get] MinIO video URL generation failed: %v", err)
		} else {
			log.Printf("[Get] MinIO video URL: %s", videoURL)
		}
		if material.SubtitleKey != "" {
			subtitleURL, err = h.minioSvc.GetPresignedURL(ctx, material.SubtitleKey, 24*time.Hour)
			if err != nil {
				log.Printf("[Get] MinIO subtitle URL generation failed: %v", err)
			}
		}
	}
	if videoURL == "" {
		videoURL, _ = h.storageSvc.GetPresignedURL(ctx, material.VideoKey, 24*time.Hour)
		log.Printf("[Get] Using local storage for video URL: %s", videoURL)
	}
	if subtitleURL == "" && material.SubtitleKey != "" {
		subtitleURL, _ = h.storageSvc.GetPresignedURL(ctx, material.SubtitleKey, 24*time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           material.ID,
		"title":        material.Title,
		"description":  material.Description,
		"video_url":    videoURL,
		"subtitle_url": subtitleURL,
		"created_at":   material.CreatedAt,
	})
}

// Delete 删除学习资料
func (h *MaterialHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	material, err := h.materialRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
		return
	}

	if material.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.materialRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete material"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// StreamVideo 代理视频流（解决 MinIO 跨域/端口问题，支持 FFmpeg 转码）
func (h *MaterialHandler) StreamVideo(c *gin.Context) {
	// 从 URL 参数获取 token（视频标签无法携带 Authorization Header）
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	claims, err := utils.ParseToken(token, h.jwtCfg)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID := claims.UserID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	material, err := h.materialRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
		return
	}

	if material.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 检查文件格式
	ext := strings.ToLower(filepath.Ext(material.VideoKey))
	needsTranscode := ext == ".avi" || ext == ".mkv" || ext == ".mov" || ext == ".wmv" || ext == ".flv"

	// 获取 MinIO 预签名 URL
	ctx := c.Request.Context()
	var videoURL string
	if h.minioSvc != nil {
		videoURL, err = h.minioSvc.GetPresignedURL(ctx, material.VideoKey, 1*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate video URL"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not available"})
		return
	}

	// 如果需要转码，使用 FFmpeg
	if needsTranscode {
		h.streamTranscodedVideo(c, videoURL)
		return
	}

	// 直接代理（MP4/WebM 等浏览器支持格式）
	h.streamDirectVideo(c, ctx, videoURL, ext)
}

// streamDirectVideo 直接代理视频流
func (h *MaterialHandler) streamDirectVideo(c *gin.Context, ctx context.Context, videoURL string, ext string) {
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// 转发 Range 头（支持视频分段加载）
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	client := &http.Client{
		Timeout: 0,
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch video"})
		return
	}
	defer resp.Body.Close()

	mimeTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".ogv":  "video/ogg",
	}
	contentType := mimeTypes[ext]
	if contentType == "" {
		contentType = resp.Header.Get("Content-Type")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	
	// 关键：始终设置 Accept-Ranges: bytes，让浏览器知道支持 Range 请求
	// 如果 MinIO 没有返回这个头，我们手动设置
	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges == "" {
		acceptRanges = "bytes"
	}
	c.Header("Accept-Ranges", acceptRanges)
	
	// 正确处理 Content-Length
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Header("Content-Length", contentLength)
	}
	
	// 正确转发 Content-Range（部分内容响应）
	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		c.Header("Content-Range", contentRange)
	}
	
	// 设置正确的状态码：200 OK 或 206 Partial Content
	c.Status(resp.StatusCode)

	io.Copy(c.Writer, resp.Body)
}

// streamTranscodedVideo 使用 FFmpeg 实时转码为 MP4
func (h *MaterialHandler) streamTranscodedVideo(c *gin.Context, videoURL string) {
	// 设置响应头（转码后为 MP4）
	c.Header("Content-Type", "video/mp4")
	c.Status(http.StatusOK)

	// FFmpeg 命令：从 URL 读取，转码为 MP4 流
	// -i input: 输入 URL
	// -c:v libx264: 视频编码为 H.264
	// -preset fast: 快速编码
	// -crf 23: 质量参数
	// -c:a aac: 音频编码为 AAC
	// -movflags frag_keyframe+empty_moov: 支持流式传输
	// -f mp4: 输出格式 MP4
	// -: 输出到 stdout
	cmd := exec.Command("ffmpeg",
		"-i", videoURL,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		"-",
	)

	// 设置 stderr 丢弃（避免阻塞）
	cmd.Stderr = os.Stderr

	// 获取 stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[FFmpeg] Failed to create stdout pipe: %v", err)
		return
	}

	// 启动 FFmpeg
	if err := cmd.Start(); err != nil {
		log.Printf("[FFmpeg] Failed to start: %v", err)
		return
	}

	// 流式传输到客户端
	_, err = io.Copy(c.Writer, stdout)
	if err != nil {
		log.Printf("[FFmpeg] Stream error: %v", err)
	}

	// 等待 FFmpeg 结束
	if err := cmd.Wait(); err != nil {
		log.Printf("[FFmpeg] Process error: %v", err)
	}
}

// GetSubtitle 获取字幕内容（解析后的 JSON）
func (h *MaterialHandler) GetSubtitle(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	material, err := h.materialRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "material not found"})
		return
	}

	if material.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if material.SubtitleKey == "" {
		// Return empty array instead of 404 - materials may not have subtitles
		c.JSON(http.StatusOK, []model.SubtitleEntry{})
		return
	}

	var buf []byte

	// 优先从 MinIO 读取字幕
	if h.minioSvc != nil {
		subtitleURL, err := h.minioSvc.GetPresignedURL(ctx, material.SubtitleKey, 1*time.Hour)
		if err == nil && subtitleURL != "" {
			// 下载字幕内容
			resp, err := http.Get(subtitleURL)
			if err == nil {
				defer resp.Body.Close()
				buf, _ = io.ReadAll(resp.Body)
			}
		}
	}

	// 如果 MinIO 读取失败，尝试本地文件
	if len(buf) == 0 {
		subtitlePath := filepath.Join("uploads", filepath.Base(material.SubtitleKey))
		file, err := os.Open(subtitlePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open subtitle file"})
			return
		}
		defer file.Close()

		buf, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read subtitle"})
			return
		}
	}

	// 解析字幕
	entries, err := service.ParseSubtitle(buf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse subtitle"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// Sync 同步 MinIO 中的文件到数据库
func (h *MaterialHandler) Sync(c *gin.Context) {
	userID := c.GetUint("userID")

	if h.minioSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MinIO not configured"})
		return
	}

	ctx := c.Request.Context()
	
	// 列出 bucket 中所有对象（前缀为空，因为文件直接存放在 bucket 根目录下）
	objects, err := h.minioSvc.ListObjects(ctx, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list objects: " + err.Error()})
		return
	}

	log.Printf("[Sync] Found %d objects in MinIO", len(objects))
	for _, obj := range objects {
		log.Printf("[Sync] Object: %s", obj.Key)
	}

	// 构建文件映射：路径 -> 对象信息
	// 分别存储视频和字幕的 key，避免覆盖
	fileMap := make(map[string]struct {
		videoKey    string
		subtitleKey string
	})
	
	for _, obj := range objects {
		ext := strings.ToLower(filepath.Ext(obj.Key))
		baseKey := obj.Key[:len(obj.Key)-len(ext)]
		info := fileMap[baseKey]
		
		if videoExts[ext] {
			info.videoKey = obj.Key
			log.Printf("[Sync] Found video: %s (baseKey: %s)", obj.Key, baseKey)
		} else if subtitleExts[ext] {
			info.subtitleKey = obj.Key
			log.Printf("[Sync] Found subtitle: %s (baseKey: %s)", obj.Key, baseKey)
		} else {
			log.Printf("[Sync] Skipped (not video/subtitle): %s (ext: %s)", obj.Key, ext)
		}
		fileMap[baseKey] = info
	}

	log.Printf("[Sync] fileMap has %d entries", len(fileMap))

	// 获取已存在的记录
	existingMaterials, _ := h.materialRepo.GetByUserID(userID)
	existingKeys := make(map[string]bool)
	for _, m := range existingMaterials {
		existingKeys[m.VideoKey] = true
	}

	// 创建新的资料记录
	var imported []string
	var skipped []string

	for _, info := range fileMap {
		if info.videoKey == "" {
			continue // 没有视频文件
		}

		videoKey := info.videoKey
		
		// 检查是否已存在
		if existingKeys[videoKey] {
			skipped = append(skipped, videoKey)
			continue
		}

		// 字幕 key 已在构建 fileMap 时设置
		subtitleKey := info.subtitleKey

		// 从文件路径提取标题（保留目录结构）
		// all2wei/cad/01.xxx.avi -> cad/01.xxx
		relativePath := videoKey
		if strings.HasPrefix(relativePath, "all2wei/") {
			relativePath = relativePath[8:] // 去掉 "all2wei/"
		}
		title := relativePath[:len(relativePath)-len(filepath.Ext(relativePath))]

		// 创建记录
		material := &model.Material{
			UserID:      userID,
			Title:       title,
			VideoKey:    videoKey,
			SubtitleKey: subtitleKey,
		}

		if err := h.materialRepo.Create(material); err != nil {
			skipped = append(skipped, videoKey+" (error)")
			continue
		}

		imported = append(imported, videoKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("同步完成，导入 %d 个，跳过 %d 个", len(imported), len(skipped)),
		"imported": imported,
		"skipped":  skipped,
	})
}