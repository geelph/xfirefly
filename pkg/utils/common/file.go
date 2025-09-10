package common

import (
	"os"
	"strings"
)

// Exists 判断文件是否存在（仅当可访问且存在时返回 true）
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		// 文件不存在，或权限不足，或I/O错误，都视为“不存在”
		return false
	}
	return true
}

// IsYamlFile
//
//	@Description: 依据扩展名判断文件是否为 YAML 文件
//	@param filename 文件名
//	@return bool 是否为 YAML 文件
func IsYamlFile(filename string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")
}
