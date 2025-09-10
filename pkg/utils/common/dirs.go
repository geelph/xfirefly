package common

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/donnie4w/go-logger/logger"
)

// DirIsExist 判断指定目录是否存在
func DirIsExist(path string) bool {
	// 无效路径
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		// 如果是路径不存在的错误，返回 false
		if os.IsNotExist(err) {
			return false
		}
		// 其他错误（如权限问题），根据需求处理，这里也返回 false
		return false
	}
	// 确保是目录而不是文件
	return info.IsDir()
}

// ExistYamlFile
//
//	@Description: 判断目录及其子目录是否存在yaml文件
//	@param path 目录路径
//	@return bool 是否存在yaml文件
func ExistYamlFile(path string) bool {
	// 遍历目录及其子目录
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Errorf("遍历目录时出现错误：%v", err)
			return err
		}
		// 检查文件是否是 YAML 文件
		if !d.IsDir() && IsYamlFile(path) {
			return filepath.SkipDir // 找到文件后停止遍历
		}
		return nil
	}) == nil
}
