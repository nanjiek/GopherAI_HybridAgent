package secret

import (
	"os"
	"strings"
)

// Provider 定义密钥加载接口，便于后续接入 Vault/KMS。
type Provider interface {
	Get(name string) string
}

// EnvProvider 从环境变量加载密钥。
type EnvProvider struct{}

// NewEnvProvider 构建环境变量密钥提供器。
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Get 返回密钥值，支持 *_FILE 文件引用形式。
func (p *EnvProvider) Get(name string) string {
	v := os.Getenv(name)
	if v != "" {
		return v
	}
	filePath := os.Getenv(name + "_FILE")
	if filePath == "" {
		return ""
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
