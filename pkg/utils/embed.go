package utils

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	finger2 "xfirefly/pkg/finger"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
)

//go:embed fingerprint/*
var EmbeddedFingerFS embed.FS
var hasEmbeddedFingers bool

// init 函数在包初始化时执行，用于检测和初始化嵌入的指纹库
// 该函数会读取嵌入的指纹文件系统中的fingerprint目录，判断是否存在嵌入的指纹数据
// 并设置全局变量hasEmbeddedFingers来标识是否使用嵌入指纹库
func init() {
	// 读取嵌入的指纹文件系统中fingerprint目录下的所有文件
	files, err := EmbeddedFingerFS.ReadDir("fingerprint")

	// 检查是否存在错误或者指纹目录为空的情况
	if err != nil || len(files) == 0 {
		hasEmbeddedFingers = false
		logger.Errorf("未嵌入指纹库，将使用文件系统中的指纹库。错误信息：%v", err)
		// 单独处理指纹目录为空的情况
		if len(files) == 0 {
			logger.Warn("指纹目录为空")
		}
	} else {
		// 成功读取到嵌入的指纹文件，设置标识为true
		hasEmbeddedFingers = true
	}
}

// GetFingerPath 获取指纹库路径
// 返回值: string - 指纹库路径，如果使用嵌入的指纹库则返回"embedded://fingerprint/"，否则返回"fingerprint/"
func GetFingerPath() string {
	// 根据是否使用嵌入的指纹库来决定返回的路径
	if hasEmbeddedFingers {
		logger.Info("使用嵌入的指纹库路径")
		return "embedded://fingerprint/"
	}
	logger.Info("使用文件系统中的指纹库路径")
	return "fingerprint/"
}

// GetFingerYaml 获取指纹yaml文件
func GetFingerYaml() ([]*finger2.Finger, error) {
	var allFinger []*finger2.Finger

	// 递归遍历所有目录查找yaml文件
	err := fs.WalkDir(EmbeddedFingerFS, "fingerprint", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理yaml文件
		if !d.IsDir() && common.IsYamlFile(d.Name()) {
			poc, err := finger2.Load(path, EmbeddedFingerFS)
			if err != nil {
				return fmt.Errorf("加载文件 %s 出错: %v", path, err)
			}
			if poc != nil {
				allFinger = append(allFinger, poc)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历指纹目录出错: %v", err)
	}

	return allFinger, nil
}

// GetCustomFingerYaml 获取指定目录及其子目录下所有指纹文件并返回
func GetCustomFingerYaml(path string) ([]*finger2.Finger, error) {
	// 临时存储所有指纹文件
	var fingerYamls []*finger2.Finger
	// 读取所有目录下的指纹文件
	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && common.IsYamlFile(path) {
			if poc, err := finger2.Read(path); err == nil && poc != nil {
				// 添加到临时存储
				fingerYamls = append(fingerYamls, poc)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 返回所有指纹文件
	return fingerYamls, nil
}
