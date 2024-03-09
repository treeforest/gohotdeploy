package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// ServerConfig 包含服务器配置信息
type ServerConfig struct {
	Port         int                         `yaml:"port"`         // 服务器端口号
	Repositories map[string]RepositoryConfig `yaml:"repositories"` // 仓库配置信息
}

// RepositoryConfig 包含仓库配置信息
type RepositoryConfig struct {
	RelativeBuildDir string `yaml:"relative_build_dir"` // 相对仓库的构建目录路径
	RunArgs          string `yaml:"run_args"`           // 运行参数
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config *ServerConfig, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config = new(ServerConfig)
	err = yaml.Unmarshal(data, config)

	return config, err
}
