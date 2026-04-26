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
	".rmvb": true,
	".rm":   true,
	".ts":   true,
	".m2ts": true,
	".3gp":  true,
	".mpg":  true,
	".mpeg": true,
	".vob":  true,
}

var directPlayExts = map[string]bool{
	".mp4":  true,
	".webm": true,
	".m4v":  true,
	".ogg":  true,
	".ogv":  true,
}

// 字幕扩展名
var subtitleExts = map[string]bool{
	".srt": true,
	".vtt": true,
}

type MaterialHandler struct {
	materialRepo *repository.MaterialRepository
	storageSvc service.StorageService
	minioSvc  *service.MinIOService // 可选，用于同步功能
	jwtCfg    *config.JWTConfig
	sourceCfg *config.VideoSourceConfig
}

func NewMaterialHandler(materialRepo *repository.MaterialRepository, storageSvc service.StorageService, jwtCfg *config.JWTConfig) *MaterialHandler {
	return &MaterialHandler{
		materialRepo: materialRepo,
		storageSvc:   storageSvc,
		jwtCfg:       jwtCfg,
	}
}

// SetVideoSource 设置视频源配置
func (h *MaterialHandler) SetVideoSource(sourceCfg *config.VideoSourceConfig) {
	h.sourceCfg = sourceCfg
}

// SetMinIOService 设置 MinIO 服务（用于同步功能）
func (h *MaterialHandler) SetMinIOService(minioSvc *service.MinIOService) {
	h.minioSvc = minioSvc
}

// extractFolder 从 videoKey 中提取文件夹名
func extractFolder(videoKey string) string {
	parts := strings.Split(videoKey, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
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
		Folder:      extractFolder(videoKey),
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
	folder := c.Query("folder")

	var materials []model.Material
	var err error

	if folder != "" {
		materials, err = h.materialRepo.GetByUserIDAndFolder(userID, folder)
	} else {
		materials, err = h.materialRepo.GetByUserID(userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get materials"})
		return
	}

	ctx := c.Request.Context()
	var responses []gin.H
	for _, m := range materials {
		videoURL, _ := h.storageSvc.GetPresignedURL(ctx, m.VideoKey, 24*time.Hour)
		if videoURL == "" {
			log.Printf("[List] Failed to generate video URL for material %d, videoKey=%s", m.ID, m.VideoKey)
		}
		responses = append(responses, gin.H{
			"id":           m.ID,
			"title":        m.Title,
			"description":  m.Description,
			"folder":       m.Folder,
			"video_url":    videoURL,
			"has_subtitle": m.SubtitleKey != "",
			"created_at":   m.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// Folders 获取用户的文件夹列表
func (h *MaterialHandler) Folders(c *gin.Context) {
	userID := c.GetUint("userID")

	folders, err := h.materialRepo.GetFolders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get folders"})
		return
	}

	type FolderInfo struct {
		Name      string `json:"name"`
		Count     int    `json:"count"`
	}

	counts := make(map[string]int)
	allMaterials, _ := h.materialRepo.GetByUserID(userID)
	for _, m := range allMaterials {
		if m.Folder != "" {
			counts[m.Folder]++
		}
	}

	var result []FolderInfo
	for _, f := range folders {
		result = append(result, FolderInfo{
			Name:  f,
			Count: counts[f],
		})
	}

	if result == nil {
		result = []FolderInfo{}
	}

	c.JSON(http.StatusOK, result)
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

	ext := strings.ToLower(filepath.Ext(material.VideoKey))
	forceTranscode := c.Query("transcode") == "1"
	directPlay := !forceTranscode && directPlayExts[ext]

	localPath := h.storageSvc.GetLocalPath(material.VideoKey)

	if !directPlay {
		if localPath != "" {
			h.streamTranscodedLocalVideo(c, localPath)
		} else {
			videoURL := h.resolveVideoURL(c, material.VideoKey)
			if videoURL == "" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get video URL"})
				return
			}
			h.streamTranscodedVideo(c, videoURL)
		}
		return
	}

	if localPath != "" {
		c.File(localPath)
		return
	}

	videoURL := h.resolveVideoURL(c, material.VideoKey)
	if videoURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get video URL"})
		return
	}
	h.streamDirectVideo(c, c.Request.Context(), videoURL, ext)
}

// resolveVideoURL 获取视频的完整访问 URL
func (h *MaterialHandler) resolveVideoURL(c *gin.Context, videoKey string) string {
	ctx := c.Request.Context()
	var videoURL string

	videoURL, _ = h.storageSvc.GetPresignedURL(ctx, videoKey, 1*time.Hour)
	if videoURL == "" && h.minioSvc != nil {
		videoURL, _ = h.minioSvc.GetPresignedURL(ctx, videoKey, 1*time.Hour)
	}

	if videoURL != "" && !strings.HasPrefix(videoURL, "http") {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		videoURL = fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, videoURL)
	}

	return videoURL
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

	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[FFmpeg] Failed to create stdout pipe: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to setup transcoding"})
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[FFmpeg] Failed to start: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ffmpeg not available, transcoding failed"})
		return
	}

	c.Header("Content-Type", "video/mp4")
	c.Status(http.StatusOK)

	_, err = io.Copy(c.Writer, stdout)
	if err != nil {
		log.Printf("[FFmpeg] Stream error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("[FFmpeg] Process error: %v", err)
	}
}

// streamTranscodedLocalVideo 使用 FFmpeg 实时转码本地文件为 MP4
func (h *MaterialHandler) streamTranscodedLocalVideo(c *gin.Context, localPath string) {
	cmd := exec.Command("ffmpeg",
		"-i", localPath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		"-",
	)

	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[FFmpeg] Failed to create stdout pipe: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to setup transcoding"})
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[FFmpeg] Failed to start: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ffmpeg not available, transcoding failed"})
		return
	}

	c.Header("Content-Type", "video/mp4")
	c.Status(http.StatusOK)

	_, err = io.Copy(c.Writer, stdout)
	if err != nil {
		log.Printf("[FFmpeg] Stream error: %v", err)
	}

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
		subtitlePath := h.storageSvc.GetLocalPath(material.SubtitleKey)
		if subtitlePath == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open subtitle file"})
			return
		}
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
			log.Printf("[Sync] Found subtitle: %s", obj.Key)
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
			Folder:      extractFolder(videoKey),
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

// ScanSource 扫描视频源文件夹并导入
func (h *MaterialHandler) ScanSource(c *gin.Context) {
	userID := c.GetUint("userID")
	sourcePath := c.Query("path")
	
	// 如果没有提供 path，返回当前配置的路径
	if sourcePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query required"})
		return
	}

	// 检查路径是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "source path not found"})
		return
	}

	// 遍历文件夹查找视频文件
	var videoFiles []string
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if videoExts[ext] {
			videoFiles = append(videoFiles, path)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan: " + err.Error()})
		return
	}

	log.Printf("[ScanSource] Found %d video files in %s", len(videoFiles), sourcePath)

	// 获取已存在的记录
	existingMaterials, _ := h.materialRepo.GetByUserID(userID)
	existingKeys := make(map[string]bool)
	for _, m := range existingMaterials {
		existingKeys[m.VideoKey] = true
	}

	// 上传并创建记录
	var imported []string
	var skipped []string
	ctx := c.Request.Context()

	for _, videoPath := range videoFiles {
		filename := filepath.Base(videoPath)
		relPath, err := filepath.Rel(sourcePath, videoPath)
		if err != nil {
			continue
		}
		videoKey := fmt.Sprintf("sources/%d/%s", userID, relPath)
		videoKey = strings.ReplaceAll(videoKey, "\\", "/")

		// 检查是否已存在
		if existingKeys[videoKey] {
			skipped = append(skipped, filename)
			continue
		}

		// 上传到存储
		if err := h.storageSvc.UploadFile(ctx, videoKey, videoPath, "video/mp4"); err != nil {
			skipped = append(skipped, filename+" (upload error)")
			continue
		}

		// 查找同名的字幕文件
		var subtitleKey string
		baseName := filename[:len(filename)-len(filepath.Ext(filename))]
		for _, ext := range []string{".srt", ".vtt"} {
			subtitlePath := filepath.Join(filepath.Dir(videoPath), baseName+ext)
			if _, err := os.Stat(subtitlePath); err == nil {
				subtitleKey = strings.ReplaceAll(filepath.Join(filepath.Dir(videoKey), baseName+ext), "\\", "/")
				h.storageSvc.UploadFile(ctx, subtitleKey, subtitlePath, "text/vtt")
				break
			}
		}

		// 提取标题
		title := baseName
		if relPath != filename {
			// 保留相对路径作为标题
			dir := filepath.Dir(relPath)
			if dir != "." {
				title = filepath.Join(dir, baseName)
			}
		}

		// 创建记录
		material := &model.Material{
			UserID:      userID,
			Title:       title,
			Folder:      extractFolder(videoKey),
			VideoKey:    videoKey,
			SubtitleKey: subtitleKey,
		}

		if err := h.materialRepo.Create(material); err != nil {
			skipped = append(skipped, filename+" (db error)")
			continue
		}

		imported = append(imported, filename)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("扫描完成，导入 %d 个，跳过 %d 个", len(imported), len(skipped)),
		"imported": imported,
		"skipped":  skipped,
	})
}

// ScanSourceFolder 启动时自动扫描视频源文件夹
func (h *MaterialHandler) ScanSourceFolder(userID uint) error {
	if h.sourceCfg == nil || !h.sourceCfg.Enabled || h.sourceCfg.Path == "" {
		log.Println("[ScanSourceFolder] Video source not enabled or path not set")
		return nil
	}

	sourcePath := h.sourceCfg.Path

	log.Printf("[ScanSourceFolder] Starting scan: path=%s", sourcePath)

	// 检查路径是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		log.Printf("[ScanSourceFolder] Path not found: %s", sourcePath)
		return fmt.Errorf("source path not found: %s", sourcePath)
	}

	// 遍历文件夹查找视频文件
	var videoFiles []string
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if videoExts[ext] {
			videoFiles = append(videoFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan: %v", err)
	}

	log.Printf("[ScanSourceFolder] Found %d video files in %s", len(videoFiles), sourcePath)

	// 获取已存在的记录
	existingMaterials, _ := h.materialRepo.GetByUserID(userID)
	existingKeys := make(map[string]bool)
	for _, m := range existingMaterials {
		existingKeys[m.VideoKey] = true
	}

	// 上传并创建记录
	var imported []string
	var skipped []string
	ctx := context.Background()

	for _, videoPath := range videoFiles {
		filename := filepath.Base(videoPath)
		relPath, err := filepath.Rel(sourcePath, videoPath)
		if err != nil {
			continue
		}
		videoKey := fmt.Sprintf("sources/%d/%s", userID, relPath)
		videoKey = strings.ReplaceAll(videoKey, "\\", "/")

		// 检查是否已存在
		if existingKeys[videoKey] {
			skipped = append(skipped, filename)
			continue
		}

		// 上传到存储
		if err := h.storageSvc.UploadFile(ctx, videoKey, videoPath, "video/mp4"); err != nil {
			skipped = append(skipped, filename+" (upload error)")
			continue
		}

		// 查找同名的字幕文件
		var subtitleKey string
		baseName := filename[:len(filename)-len(filepath.Ext(filename))]
		for _, ext := range []string{".srt", ".vtt"} {
			subtitlePath := filepath.Join(filepath.Dir(videoPath), baseName+ext)
			if _, err := os.Stat(subtitlePath); err == nil {
				subtitleKey = strings.ReplaceAll(filepath.Join(filepath.Dir(videoKey), baseName+ext), "\\", "/")
				h.storageSvc.UploadFile(ctx, subtitleKey, subtitlePath, "text/vtt")
				break
			}
		}

		// 提取标题
		title := baseName
		if relPath != filename {
			dir := filepath.Dir(relPath)
			if dir != "." {
				title = filepath.Join(dir, baseName)
			}
		}

		// 创建记录
		material := &model.Material{
			UserID:      userID,
			Title:       title,
			Folder:      extractFolder(videoKey),
			VideoKey:    videoKey,
			SubtitleKey: subtitleKey,
		}

		if err := h.materialRepo.Create(material); err != nil {
			skipped = append(skipped, filename+" (db error)")
			continue
		}

		imported = append(imported, filename)
	}

	log.Printf("[ScanSourceFolder] Scan completed: imported %d, skipped %d", len(imported), len(skipped))
	return nil
}