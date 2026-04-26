package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Storage    StorageConfig    `mapstructure:"storage"`
	VideoSource VideoSourceConfig `mapstructure:"video_source"`
	MinIO      MinIOConfig      `mapstructure:"minio"`
	JWT        JWTConfig        `mapstructure:"jwt"`
}

type VideoSourceConfig struct {
	Path    string `mapstructure:"path"`    // 视频源文件夹路径
	Enabled bool   `mapstructure:"enabled"` // 是否启用
	UserID  uint   `mapstructure:"user_id"` // 导入到哪个用户
}

type StorageConfig struct {
	Type string `mapstructure:"type"` // "minio" 或 "local"
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"` // hours
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 默认值
	viper.SetDefault("server.port", ":8080")
	viper.SetDefault("database.path", "all2wei.db")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("video_source.enabled", false)
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.bucket_name", "all2wei")
	viper.SetDefault("minio.region", "us-east-1")
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.expire", 168) // 7 days

	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在也没关系，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
