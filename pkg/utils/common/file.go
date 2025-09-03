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

// IsYamlFile checks if the given filename has a YAML file extension.
// It returns true if the filename ends with either ".yaml" or ".yml".
//
// Parameters:
//
//	filename - the name of the file to check
//
// Returns:
//
//	bool - true if the file is a YAML file, false otherwise
func IsYamlFile(filename string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")
}
