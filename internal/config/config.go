package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ConfigPath string
	TokenPath  string
	ClientID   string
	UserAgent  string
	Scopes     []string
}

func Default() Config {
	base := defaultConfigDir()
	userAgent := strings.TrimSpace(os.Getenv("RDLXD_USER_AGENT"))
	if userAgent == "" {
		userAgent = strings.TrimSpace(os.Getenv("REDDITDOWNLOADER_USER_AGENT"))
	}
	if userAgent == "" {
		userAgent = strings.TrimSpace(os.Getenv("REDDIT_USER_AGENT"))
	}
	if userAgent == "" {
		userAgent = "rdlxd/0.1"
	}
	clientID := strings.TrimSpace(os.Getenv("RDLXD_CLIENT_ID"))
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("REDDITDOWNLOADER_CLIENT_ID"))
	}
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("REDDIT_CLIENT_ID"))
	}
	return Config{
		ConfigPath: filepath.Join(base, "config.json"),
		TokenPath:  filepath.Join(base, "token.json"),
		ClientID:   clientID,
		UserAgent:  userAgent,
		Scopes:     []string{"read", "identity"},
	}
}

func (c Config) ScopesString() string {
	return strings.Join(c.Scopes, " ")
}

func defaultConfigDir() string {
	if dir := os.Getenv("RDLXD_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("REDDITDOWNLOADER_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "rdlxd")
	}
	return ".rdlxd"
}
