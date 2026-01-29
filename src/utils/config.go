package utils

import (
	"time"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
)

type (
	jwtConfig struct {
		AccessExpiration  time.Duration `toml:"access_exp"`
		RefreshExpiration time.Duration `toml:"refresh_exp"`
	}

	rootConfig struct {
		DatabaseUrl    string `toml:"database_url"`
		RedisUrl       string `toml:"redis_url"`
		CDNUrl         string `toml:"cdn_url"`
		MultipartLimit int64  `toml:"multipart_limit"`
		DefaultAvatar  string `toml:"default_avatar"`
		JWT            jwtConfig
	}
)

var Config *rootConfig

func InitConfig(configPath string) {
	logger := zap.L()
	logger.Info("Reading config file...", zap.String("path", configPath))

	_, err := toml.DecodeFile(configPath, &Config)
	if err != nil {
		logger.Fatal("Failed to decode config file", zap.Error(err))
	}

	logger.Info("Config variables successfully loaded")
}
