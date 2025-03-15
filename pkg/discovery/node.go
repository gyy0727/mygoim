package discovery

import (
	"fmt"
	"strings"
)

type Node struct {
	Name string `json:"name"` //*名称
	Addr string `json:"addr"` //*地址
}

// *把服务名中的 . 转换为 /
func (s Node) transName() string {
	return strings.ReplaceAll(s.Name, ".", "/")
}

// *构建节点 key
func (s Node) buildKey() string {
	return fmt.Sprintf("/%s/%s", s.transName(), s.Addr)
}

// *构建节点前缀
func (s Node) buildPrefix() string {
	return fmt.Sprintf("/%s", s.transName())
}

func (s Node) SplitPath(input string) (string, string, error) {
	fmt.Println("要解析的路径：", input)

	// 提取路径部分
	start := strings.Index(input, `"`) + 1
	end := strings.LastIndex(input, `"`)
	if start < 0 || end < 0 || start >= end {
		return "", "", fmt.Errorf("无效的输入格式: %s", input)
	}
	path := input[start:end]
	fmt.Println("提取的路径：", path)

	// 去掉开头和结尾的斜杠
	path = strings.Trim(path, "/")
	fmt.Println("去掉斜杠后的路径：", path)

	// 按斜杠分割字符串
	parts := strings.Split(path, "/")
	fmt.Println("分割后的部分：", parts)

	// 检查分割后的部分是否符合预期
	if len(parts) != 2 {
		return "", "", fmt.Errorf("无效的路径格式: %s", path)
	}

	// 返回两个部分
	fmt.Println("第一部分:", parts[0], "第二部分:", parts[1])
	return parts[0], parts[1], nil
}

