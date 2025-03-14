package conf

import (
	"testing"
)

func TestConfig(t *testing.T) {
	// 设置测试用的配置文件路径
	confPath = "conf.toml"

	// 初始化配置
	err := Init()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	// 输出解析后的配置
	t.Log("Parsed Config:")
	t.Log(Conf.String())
}
