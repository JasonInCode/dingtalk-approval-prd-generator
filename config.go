package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 配置文件结构
type Config struct {
	Cookie         string `yaml:"cookie"`          // 请求 Cookie
	CsrfToken      string `yaml:"csrf_token"`      // CSRF Token
	CorpId         string `yaml:"corp_id"`         // 企业 ID
	OutputDir      string `yaml:"output_dir"`      // 输出目录
	OutputFilename string `yaml:"output_filename"` // 输出文件名
	Workers        int    `yaml:"workers"`         // 并发线程数
	RetryCount     int    `yaml:"retry_count"`     // 失败重试次数
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	OutputDir:      "D:\\Download",
	OutputFilename: "approval_forms.csv",
	Workers:        5,
	RetryCount:     3,
}

// LoadConfig 从 YAML 文件加载配置
func LoadConfig(filename string) (*Config, error) {
	// 📁 读取配置文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 📝 解析 YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 🔧 应用默认值
	if config.OutputDir == "" {
		config.OutputDir = DefaultConfig.OutputDir
	}
	if config.OutputFilename == "" {
		config.OutputFilename = DefaultConfig.OutputFilename
	}
	if config.Workers <= 0 {
		config.Workers = DefaultConfig.Workers
	}
	if config.RetryCount <= 0 {
		config.RetryCount = DefaultConfig.RetryCount
	}

	return &config, nil
}