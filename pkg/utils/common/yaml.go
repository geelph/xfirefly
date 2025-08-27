package common

import "strings"

// IsYamlFile 判断文件是否为YAML格式
func IsYamlFile(filename string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")
}
