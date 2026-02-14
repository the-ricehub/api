package utils

import (
	"time"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
)

type (
	rootConfig struct {
		DatabaseUrl       string `toml:"database_url"`
		RedisUrl          string `toml:"redis_url"`
		CDNUrl            string `toml:"cdn_url"`
		DefaultAvatar     string `toml:"default_avatar"`
		CorsOrigin        string `toml:"cors_origin"`
		DisableRateLimits bool   `toml:"disable_rate_limits"`
		JWT               jwtConfig
		Limits            limitsConfig
		Blacklist         blacklistConfig
	}

	jwtConfig struct {
		AccessExpiration  time.Duration `toml:"access_exp"`
		RefreshExpiration time.Duration `toml:"refresh_exp"`
	}

	limitsConfig struct {
		MaxPreviewsPerRice  int   `toml:"max_previews_per_rice"`
		UserAvatarSizeLimit int64 `toml:"user_avatar_size_limit"`
		DotfilesSizeLimit   int64 `toml:"dotfiles_size_limit"`
		PreviewSizeLimit    int64 `toml:"preview_size_limit"`
	}

	blacklistConfig struct {
		Words     []string
		Usernames []string
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
