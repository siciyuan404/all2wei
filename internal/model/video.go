package model

import (
	"time"

	"gorm.io/gorm"
)

// Material 学习资料（视频）
type Material struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	Title       string         `gorm:"not null" json:"title"`
	Description string         `json:"description"`
	VideoKey    string         `gorm:"not null" json:"video_key"`       // MinIO 中的对象键
	VideoURL    string         `gorm:"-" json:"video_url"`              // 临时预签名 URL
	SubtitleKey string         `json:"subtitle_key"`                    // 字幕文件 Key
	Duration    int            `json:"duration"`                        // 视频时长（秒）
	Status      string         `gorm:"default:'active'" json:"status"` // active, deleted
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// SubtitleEntry 字幕条目
type SubtitleEntry struct {
	Index     int     `json:"index"`      // 序号
	StartTime float64 `json:"start_time"` // 开始时间（秒）
	EndTime   float64 `json:"end_time"`   // 结束时间（秒）
	Text      string  `json:"text"`       // 字幕文本
}

// CreateMaterialRequest 创建资料请求
type CreateMaterialRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

// MaterialResponse 资料响应
type MaterialResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	VideoURL    string    `json:"video_url"`
	Duration    int       `json:"duration"`
	CreatedAt   time.Time `json:"created_at"`
}
