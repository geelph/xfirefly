package output

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"os"
	"path/filepath"
	"strings"
	"xfirefly/pkg/finger"
	"xfirefly/pkg/utils/proto"
)

// CreateProgressBar 创建进度条
func CreateProgressBar(total int) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		int64(total),
		progressbar.OptionSetWidth(50),
		progressbar.OptionEnableColorCodes(false),
		progressbar.OptionShowBytes(false),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetDescription("指纹识别"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionClearOnFinish(),
	)
}

// GetOutputFormat 确定输出格式
func GetOutputFormat(jsonOutput bool, outputPath string) string {
	// 优先判断是否启用JSON输出
	if jsonOutput {
		return "json"
	}

	if outputPath == "" {
		return "txt" // 默认为txt格式
	}

	ext := strings.ToLower(filepath.Ext(outputPath))
	if ext == ".csv" {
		return "csv"
	}
	return "txt"
}

// PrintSummary 打印汇总信息
func PrintSummary(targets []string, results map[string]*TargetResult) {
	matchCount := 0
	noMatchCount := 0

	// 统计匹配成功和失败的数量
	for _, targetResult := range results {
		if len(targetResult.Matches) > 0 {
			matchCount++
		} else {
			noMatchCount++
		}
	}

	// 输出统计信息
	fmt.Println(color.CyanString("─────────────────────────────────────────────────────"))
	fmt.Printf("扫描统计: 目标总数 %d, 匹配成功 %d, 匹配失败 %d\n",
		len(targets), matchCount, noMatchCount)
}

// HandleMatchResults 处理匹配结果并输出到控制台
func HandleMatchResults(targetResult *TargetResult, output string, sockOutput string, printResult func(string), outputFormat string, lastResponse *proto.Response) {
	// 构建基础信息
	statusCodeStr := ""
	if targetResult.StatusCode > 0 {
		statusCodeStr = fmt.Sprintf("（%d）", targetResult.StatusCode)
	}

	serverInfo := ""
	if targetResult.ServerInfo != nil {
		serverInfo = targetResult.ServerInfo.ServerType
	}

	// 构建输出信息
	baseInfoStr := fmt.Sprintf("URL：%s %s  标题：%s  Server：%s",
		targetResult.URL, statusCodeStr, targetResult.Title, serverInfo)

	// 构建技术栈信息（合并为一行）
	var techInfoStr string
	if targetResult.Wappalyzer != nil {
		var techParts []string

		// Web服务器
		if len(targetResult.Wappalyzer.WebServers) > 0 {
			techParts = append(techParts, fmt.Sprintf("Web服务器：[%s]", strings.Join(targetResult.Wappalyzer.WebServers, ", ")))
		}

		// 编程语言
		if len(targetResult.Wappalyzer.ProgrammingLanguages) > 0 {
			techParts = append(techParts, fmt.Sprintf("编程语言：[%s]", strings.Join(targetResult.Wappalyzer.ProgrammingLanguages, ", ")))
		}

		// Web框架
		if len(targetResult.Wappalyzer.WebFrameworks) > 0 {
			techParts = append(techParts, fmt.Sprintf("Web框架：[%s]", strings.Join(targetResult.Wappalyzer.WebFrameworks, ", ")))
		}

		// JS框架和库 (合并展示，减少输出宽度)
		jsComponents := append([]string{}, targetResult.Wappalyzer.JavaScriptFrameworks...)
		jsComponents = append(jsComponents, targetResult.Wappalyzer.JavaScriptLibraries...)
		if len(jsComponents) > 0 {
			techParts = append(techParts, fmt.Sprintf("JS组件：[%s]", strings.Join(jsComponents, ", ")))
		}

		techInfoStr = strings.Join(techParts, "")
	}

	// 根据匹配结果构建完整输出信息
	var outputMsg string
	var matchResultStr string
	var successColor = "\033[32m" // 绿色
	var failColor = "\033[31m"    // 红色
	var resetColor = "\033[0m"    // 重置颜色

	if len(targetResult.Matches) > 0 {
		// 收集所有匹配的指纹名称
		fingerNames := make([]string, 0, len(targetResult.Matches))
		for _, match := range targetResult.Matches {
			fingerNames = append(fingerNames, match.Finger.Info.Name)
		}
		matchResultStr = fmt.Sprintf("  指纹：[%s]  匹配结果：%s%s%s",
			strings.Join(fingerNames, "，"), successColor, "成功", resetColor)
	} else {
		matchResultStr = fmt.Sprintf("  匹配结果：%s%s%s", failColor, "未匹配", resetColor)
	}

	// 组合最终输出信息，技术栈在一行，匹配结果放在末尾
	if techInfoStr != "" {
		outputMsg = fmt.Sprintf("%s%s%s", baseInfoStr, techInfoStr, matchResultStr)
	} else {
		outputMsg = fmt.Sprintf("%s%s", baseInfoStr, matchResultStr)
	}

	// 输出结果
	printResult(outputMsg)

	// 写入输出文件
	if output != "" {
		WriteResultToFile(targetResult, output, outputFormat, lastResponse)
	}

	// 写入socket文件
	if sockOutput != "" {
		WriteResultToSock(targetResult, lastResponse)
	}
}

// CreateWriteOptions 创建通用的写入选项结构体
func CreateWriteOptions(targetResult *TargetResult, outputPath string, format string, lastResponse *proto.Response) *WriteOptions {
	// 收集指纹信息
	var IsMatch bool
	fingerList := make([]*finger.Finger, 0, len(targetResult.Matches))
	for _, match := range targetResult.Matches {
		fingerList = append(fingerList, match.Finger)
	}
	if len(targetResult.Matches) > 0 {
		IsMatch = true
	}

	// 创建写入选项结构体
	writeOpts := &WriteOptions{
		Output:      outputPath,
		Format:      format,
		Target:      targetResult.URL,
		Fingers:     fingerList,
		StatusCode:  targetResult.StatusCode,
		Title:       targetResult.Title,
		ServerInfo:  targetResult.ServerInfo,
		Wappalyzer:  targetResult.Wappalyzer,
		FinalResult: IsMatch,
	}

	// 检查并设置响应头信息
	if lastResponse != nil {
		writeOpts.RespHeaders = string(lastResponse.RawHeader)
		writeOpts.Response = lastResponse
	}

	return writeOpts
}
