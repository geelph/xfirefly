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

type PrettyHandlerOptions struct {
	slog.HandlerOptions
}

type PrettyHandler struct {
	l      *log.Logger
	opts   PrettyHandlerOptions
	attrs  []slog.Attr
	groups []string
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	level := "[" + r.Level.String() + "]"
	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	// 提取 attrs
	attrs := make(map[string]interface{})
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	var fieldsStr string
	if len(attrs) > 0 {
		if b, err := json.Marshal(attrs); err == nil {
			fieldsStr = " " + string(b)
		}
	}

	timeStr := color.CyanString("[" + r.Time.Format("15:04:05") + "]")
	msg := r.Message

	// 处理 source
	var source string
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := fs.Next()
		file := path.Base(frame.File)
		source = color.CyanString(fmt.Sprintf(" %s:%d", file, frame.Line))
	}

	h.l.Printf("%s %s%s %s%s\n", timeStr, level, source, msg, fieldsStr)
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &PrettyHandler{
		l:      h.l,
		opts:   h.opts,
		attrs:  append(h.attrs[:len(h.attrs):len(h.attrs)], attrs...), // 安全拷贝
		groups: h.groups,
	}
	return newHandler
}

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

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func NewPrettyHandler(w io.Writer, opts PrettyHandlerOptions) *PrettyHandler {
	return &PrettyHandler{
		l:    log.New(w, "", 0),
		opts: opts,
	}
}

var Logger = slog.New(NewPrettyHandler(os.Stdout, PrettyHandlerOptions{
	HandlerOptions: slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	},
}))
