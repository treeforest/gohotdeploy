package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// ServerConfig 包含服务器配置信息
type ServerConfig struct {
	Port         int                          `yaml:"port"`         // 服务器端口号
	Repositories map[string]*RepositoryConfig `yaml:"repositories"` // 仓库配置信息
}

// RepositoryConfig 包含仓库配置信息
type RepositoryConfig struct {
	BuildRelativeDir string `yaml:"build_relative_dir"` // 相对仓库的构建目录路径
	BuildArgsBin     string `yaml:"build_args_bin"`     // 执行二进制时的运行参数
}

func (c *ServerConfig) setDefault() {
	for repoName := range c.Repositories {
		repo := c.Repositories[repoName]
		if repo.BuildRelativeDir == "" {
			repo.BuildRelativeDir = "."
		}
	}
}

func (c *RepositoryConfig) BuildCmd() string {
	return fmt.Sprintf("go build -o ./tmp/main %s", c.BuildRelativeDir)
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (conf *ServerConfig, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf = new(ServerConfig)
	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}

	conf.setDefault()

	return conf, err
}
