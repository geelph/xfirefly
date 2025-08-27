package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"xfirefly/pkg/utils/proto"

	"github.com/donnie4w/go-logger/logger"
)

// InitOutput 初始化输出文件，写入表头
func InitOutput(outputPath, format string) error {
	if outputPath == "" {
		return nil
	}
	return openOutputFile(outputPath, format)
}

// WriteHeader 写入输出文件的表头
func WriteHeader(format string) error {
	if headerWritten || outputFile == nil {
		return nil
	}

	if format == "csv" {
		if csvWriter == nil {
			csvWriter = csv.NewWriter(outputFile)
		}

		// 写入扩展的CSV表头
		if err := csvWriter.Write([]string{
			"URL", "状态码", "标题", "服务器信息",
			"Web服务器", "JS框架", "JS库", "Web框架", "编程语言",
			"指纹ID", "指纹名称", "响应头", "匹配结果", "备注",
		}); err != nil {
			return fmt.Errorf("写入CSV表头失败: %v", err)
		}
		csvWriter.Flush()
	} else if format == "json" {
		// JSON格式不需要写表头
	} else {
		// 文本格式表头
		header := fmt.Sprintf("%-40s%-10s%-30s%-20s%-20s%-20s%-20s%-20s%-20s%-30s%-30s%-50s%-15s%-20s\n",
			"URL", "状态码", "标题", "服务器信息",
			"Web服务器", "JS框架", "JS库", "Web框架", "编程语言",
			"指纹ID", "指纹名称", "响应头", "匹配结果", "备注")

		// 写入表头和分隔线
		if _, err := outputFile.WriteString(header); err != nil {
			return fmt.Errorf("写入表头失败: %v", err)
		}

		if _, err := outputFile.WriteString(strings.Repeat("-", 300) + "\n"); err != nil {
			return fmt.Errorf("写入分隔线失败: %v", err)
		}
	}

	headerWritten = true
	return nil
}

// openOutputFile 打开或创建输出文件的通用函数
func openOutputFile(output, format string) error {
	// 如果文件已经正确打开，直接返回
	if outputFile != nil && outputFile.Name() == output {
		return nil
	}

	// 关闭现有的文件
	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		_ = outputFile.Close()
		outputFile = nil
		csvWriter = nil
	}

	// 确保输出目录存在
	dir := filepath.Dir(output)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 检查文件是否存在
	fileExists := false
	if _, err := os.Stat(output); err == nil {
		fileExists = true
	}

	// 创建模式：新文件或覆盖已有文件
	var file *os.File
	var err error

	if format == "csv" && !fileExists {
		// 对于新的CSV文件，先创建文件并写入UTF-8 BOM
		file, err = os.Create(output)
		if err != nil {
			return fmt.Errorf("创建输出文件失败: %v", err)
		}

		// 写入UTF-8 BOM标识 (EF BB BF)
		if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			file.Close()
			return fmt.Errorf("写入UTF-8 BOM失败: %v", err)
		}
	} else {
		// 非CSV文件或已存在的CSV文件，使用追加模式打开
		file, err = os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("打开输出文件失败: %v", err)
		}
	}

	outputFile = file
	headerWritten = fileExists

	// 初始化CSV写入器
	if format == "csv" {
		csvWriter = csv.NewWriter(file)
	}

	// 如果是新文件，写入表头
	if !fileExists {
		if err := WriteHeader(format); err != nil {
			return err
		}
	}

	return nil
}

// WriteFingerprints 使用结构体选项写入指纹组合结果
func WriteFingerprints(opts *WriteOptions) error {
	// 检查参数有效性
	if opts.Output == "" {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	// 确保文件已打开
	if err := openOutputFile(opts.Output, opts.Format); err != nil {
		return err
	}

	// 收集指纹信息并格式化
	fingersCount := len(opts.Fingers)
	fingerIDs := make([]string, 0, fingersCount)
	fingerNames := make([]string, 0, fingersCount)

	for _, f := range opts.Fingers {
		fingerIDs = append(fingerIDs, f.Id)
		fingerNames = append(fingerNames, f.Info.Name)
	}

	fingerIDStr := fmt.Sprintf("[%s]", strings.Join(fingerIDs, "，"))
	fingerNameStr := fmt.Sprintf("[%s]", strings.Join(fingerNames, "，"))

	// 使用传入的备注或生成默认备注
	remark := opts.Remark
	if remark == "" {
		remark = fmt.Sprintf("发现%d个指纹", fingersCount)
	}

	// 处理服务器信息
	serverInfoStr := ""
	if opts.ServerInfo != nil {
		serverInfoStr = opts.ServerInfo.ServerType
	}

	// 格式化响应头为HTTP标准格式
	headersStr := ""
	if opts.Response != nil && opts.Response.RawHeader != nil {
		headersStr = string(opts.Response.RawHeader)
	} else if opts.RespHeaders != "" {
		headersStr = opts.RespHeaders
	}

	// 提取Wappalyzer信息
	webServers := "-"
	jsFrameworks := "-"
	jsLibraries := "-"
	webFrameworks := "-"
	programmingLangs := "-"

	if opts.Wappalyzer != nil {
		webServers = formatStringArray(opts.Wappalyzer.WebServers)
		jsFrameworks = formatStringArray(opts.Wappalyzer.JavaScriptFrameworks)
		jsLibraries = formatStringArray(opts.Wappalyzer.JavaScriptLibraries)
		webFrameworks = formatStringArray(opts.Wappalyzer.WebFrameworks)
		programmingLangs = formatStringArray(opts.Wappalyzer.ProgrammingLanguages)
	}

	// 构建技术栈信息
	var techStackParts []string
	if webServers != "-" {
		techStackParts = append(techStackParts, fmt.Sprintf("Web服务器：%s", webServers))
	}
	if jsFrameworks != "-" {
		techStackParts = append(techStackParts, fmt.Sprintf("JS框架：%s", jsFrameworks))
	}
	if jsLibraries != "-" {
		techStackParts = append(techStackParts, fmt.Sprintf("JS库：%s", jsLibraries))
	}
	if webFrameworks != "-" {
		techStackParts = append(techStackParts, fmt.Sprintf("Web框架：%s", webFrameworks))
	}
	if programmingLangs != "-" {
		techStackParts = append(techStackParts, fmt.Sprintf("编程语言：%s", programmingLangs))
	}

	techStackStr := "-"
	if len(techStackParts) > 0 {
		techStackStr = strings.Join(techStackParts, " | ")
	}

	// 根据不同格式写入结果
	if opts.Format == "json" {
		// 构建JSON对象
		jsonOutput := &JSONOutput{
			URL:         opts.Target,
			StatusCode:  opts.StatusCode,
			Title:       opts.Title,
			Server:      serverInfoStr,
			FingerIDs:   fingerIDs,
			FingerNames: fingerNames,
			Headers:     headersStr,
			Wappalyzer:  opts.Wappalyzer,
			MatchResult: opts.FinalResult,
			Remark:      remark,
		}

		// 序列化为JSON
		jsonData, err := json.MarshalIndent(jsonOutput, "", "")
		if err != nil {
			return fmt.Errorf("JSON序列化失败: %v", err)
		}

		// 写入JSON数据和换行符
		if _, err := outputFile.Write(jsonData); err != nil {
			return fmt.Errorf("写入JSON数据失败: %v", err)
		}
		if _, err := outputFile.Write([]byte("\n")); err != nil {
			return fmt.Errorf("写入换行符失败: %v", err)
		}

	} else if opts.Format == "csv" {
		if err := csvWriter.Write([]string{
			opts.Target,
			fmt.Sprintf("%d", opts.StatusCode),
			opts.Title,
			serverInfoStr,
			webServers,
			jsFrameworks,
			jsLibraries,
			webFrameworks,
			programmingLangs,
			fingerIDStr,
			fingerNameStr,
			strings.ReplaceAll(headersStr, "\n", "\\n"), // CSV中换行符需要转义
			fmt.Sprintf("%v", opts.FinalResult),
			remark,
		}); err != nil {
			return fmt.Errorf("写入CSV记录失败: %v", err)
		}
		csvWriter.Flush()
	} else {
		// 使用strings.Builder提高字符串拼接效率
		var sb strings.Builder
		// 预分配合理的缓冲区大小
		sb.Grow(512 + len(headersStr))

		sb.WriteString("URL: ")
		sb.WriteString(opts.Target)
		sb.WriteString("\n状态码: ")
		sb.WriteString(fmt.Sprintf("%d", opts.StatusCode))
		sb.WriteString("\n标题: ")
		sb.WriteString(opts.Title)
		sb.WriteString("\n服务器: ")
		sb.WriteString(serverInfoStr)

		// 技术栈信息单行显示
		sb.WriteString("\n技术栈: ")
		sb.WriteString(techStackStr)

		sb.WriteString("\n指纹ID: ")
		sb.WriteString(fingerIDStr)
		sb.WriteString("\n指纹名称: ")
		sb.WriteString(fingerNameStr)
		sb.WriteString("\n匹配结果: ")
		sb.WriteString(fmt.Sprintf("%v", opts.FinalResult))
		sb.WriteString("\n备注: ")
		sb.WriteString(remark)
		sb.WriteString("\n响应头:\n")
		sb.WriteString(headersStr)
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("-", 100))
		sb.WriteString("\n")

		if _, err := outputFile.WriteString(sb.String()); err != nil {
			return fmt.Errorf("写入结果失败: %v", err)
		}
	}

	return nil
}

// WriteResultToFile 将结果写入文件
func WriteResultToFile(targetResult *TargetResult, outputs, format string, lastResponse *proto.Response) {
	writeOpts := CreateWriteOptions(targetResult, outputs, format, lastResponse)

	// 写入结果
	if err := WriteFingerprints(writeOpts); err != nil {
		logger.Error(fmt.Sprintf("写入结果文件失败: %v", err))
	}
}

// CloseFileOutput 关闭仅文件输出资源
func CloseFileOutput() error {
	mu.Lock()
	defer mu.Unlock()

	// 关闭常规输出文件
	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		err := outputFile.Close()
		outputFile = nil
		csvWriter = nil
		headerWritten = false
		return err
	}

	return nil
}
