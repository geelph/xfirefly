/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: embed.go
    @Date: 2025/3/10 下午4:37*
*/
package utils

import (
	"embed"
	"fmt"
	"io/fs"
	finger2 "xfirefly/pkg/finger"
	"xfirefly/pkg/utils/common"
)

//go:embed finger/*
var EmbeddedFingerFS embed.FS
var hasEmbeddedFingers bool

func init() {
	files, err := EmbeddedFingerFS.ReadDir("finger")
	if err != nil || len(files) == 0 {
		hasEmbeddedFingers = false
		fmt.Printf("提示: 未嵌入指纹库，将使用文件系统中的指纹库。错误信息：%v\n", err)
		if len(files) == 0 {
			fmt.Println("提示：指纹目录为空。")
		}
	} else {
		hasEmbeddedFingers = true
	}
}

func GetFingerPath() string {
	if hasEmbeddedFingers {
		fmt.Println("使用嵌入的指纹库路径")
		return "embedded://finger/"
	}
	fmt.Println("使用文件系统中的指纹库路径")
	return "finger/"
}

// GetFingerYaml 获取指纹yaml文件
func GetFingerYaml() ([]*finger2.Finger, error) {
	var allFinger []*finger2.Finger

	// 递归遍历所有目录查找yaml文件
	err := fs.WalkDir(EmbeddedFingerFS, "finger", func(path string, d fs.DirEntry, err error) error {
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
