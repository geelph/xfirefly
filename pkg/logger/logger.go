package logger

import (
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
)

var h = handler.NewConsoleHandler(slog.DangerLevels)

// h.Formatter().(*slog.TextFormatter).SetTemplate(slog.NamedTemplate)
var Log = slog.NewWithHandlers(h)

//logger := slog.NewConsoleLogger()
