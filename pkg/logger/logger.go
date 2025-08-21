// Package logger 提供了一个美化格式的日志记录器，支持不同级别的日志和彩色输出
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"

	"log/slog"

	"github.com/fatih/color"
)

// PrettyHandlerOptions 定义了美化处理器的配置选项
type PrettyHandlerOptions struct {
	slog.HandlerOptions // 嵌入标准的 slog 处理器选项
}

// PrettyHandler 是自定义的 slog 处理器，用于美化日志输出格式
type PrettyHandler struct {
	l      *log.Logger          // 底层日志记录器
	opts   PrettyHandlerOptions // 处理器选项
	attrs  []slog.Attr          // 属性列表
	groups []string             // 分组名称列表
}

// Handle 实现 slog.Handler 接口，处理单条日志记录
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	// 根据日志级别设置不同的颜色
	level := "[" + r.Level.String() + "]"
	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level) // 调试级别使用洋红色
	case slog.LevelInfo:
		level = color.BlueString(level) // 信息级别使用蓝色
	case slog.LevelWarn:
		level = color.YellowString(level) // 警告级别使用黄色
	case slog.LevelError:
		level = color.RedString(level) // 错误级别使用红色
	}

	// 提取所有属性到 map 中
	attrs := make(map[string]interface{})
	// 添加处理器级别的属性
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	// 添加记录级别的属性
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	// 将属性转换为 JSON 字符串
	var fieldsStr string
	if len(attrs) > 0 {
		if b, err := json.Marshal(attrs); err == nil {
			fieldsStr = " " + string(b)
		}
	}

	// 格式化时间戳（青色显示）
	timeStr := color.CyanString("[" + r.Time.Format("15:04:05") + "]")
	msg := r.Message // 日志消息内容

	// 处理源代码位置信息
	var source string
	if h.opts.AddSource && r.PC != 0 {
		// 获取调用栈帧信息
		fs := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := fs.Next()
		// 只显示文件名而不是完整路径
		file := path.Base(frame.File)
		source = color.CyanString(fmt.Sprintf(" %s:%d", file, frame.Line))
	}

	// 输出格式化的日志行
	h.l.Printf("%s %s%s %s%s\n", timeStr, level, source, msg, fieldsStr)
	return nil
}

// WithAttrs 实现 slog.Handler 接口，返回带有额外属性的新处理器
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &PrettyHandler{
		l:      h.l,
		opts:   h.opts,
		attrs:  append(h.attrs[:len(h.attrs):len(h.attrs)], attrs...), // 安全拷贝原有属性并追加新属性
		groups: h.groups,
	}
	return newHandler
}

// WithGroup 实现 slog.Handler 接口，返回带有指定分组的新处理器
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	// 简单实现：只记录 group 名（实际可嵌套处理）
	newGroups := append(h.groups, name)
	return &PrettyHandler{
		l:      h.l,
		opts:   h.opts,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// Enabled 实现 slog.Handler 接口，判断指定级别的日志是否应该被处理
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// NewPrettyHandler 创建一个新的美化格式处理器
func NewPrettyHandler(w io.Writer, opts PrettyHandlerOptions) *PrettyHandler {
	return &PrettyHandler{
		l:    log.New(w, "", 0), // 创建不带前缀和标志的标准日志记录器
		opts: opts,
	}
}

// Logger 是全局的日志记录器实例，配置为调试级别并显示源代码位置
var Logger = slog.New(NewPrettyHandler(os.Stdout, PrettyHandlerOptions{
	HandlerOptions: slog.HandlerOptions{
		Level:     slog.LevelDebug, // 设置最低日志级别为 Debug
		AddSource: true,            // 启用源代码位置信息
	},
}))
