package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	Port         string    `json:"port"`
	APIKey       string    `json:"api_key"` // 网关自身的认证 Key
	DefaultModel string    `json:"default_model"`
	Accounts     []Account `json:"accounts"`
}

// Account 网页端账号配置
type Account struct {
	ID           string `json:"id"`
	ServiceToken string `json:"service_token"`
	UserID       string `json:"user_id"`
	Ph           string `json:"ph"`
	Active       bool   `json:"active"`
}

var (
	cfg  *Config
	mu   sync.RWMutex
	path string
)

func Load(p string) (*Config, error) {
	path = p
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &Config{Port: "8080", APIKey: "sk-mimo", DefaultModel: "mimo-v2.5-pro"}
			applyEnvOverrides(cfg)
			return cfg, Save()
		}
		return nil, err
	}
	cfg = &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	applyEnvOverrides(cfg)
	return cfg, nil
}

// applyEnvOverrides 用环境变量覆盖配置文件的值，便于容器化部署：
// 无需挂载 config.json 即可设置 API Key / 端口 / 默认模型。
// 仅当环境变量非空时覆盖。
func applyEnvOverrides(c *Config) {
	if v := os.Getenv("MIMO_PORT"); v != "" {
		c.Port = v
	}
	if v := os.Getenv("MIMO_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("MIMO_DEFAULT_MODEL"); v != "" {
		c.DefaultModel = v
	}
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return *cfg
}

func Save() error {
	mu.RLock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	mu.RUnlock()
	if err != nil {
		return err
	}
	// 0600：仅属主可读写，防止同机其他用户读取 Cookie 明文
	return os.WriteFile(path, data, 0600)
}

func Update(fn func(*Config)) {
	mu.Lock()
	fn(cfg)
	mu.Unlock()
}
