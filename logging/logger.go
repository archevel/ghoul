package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	e "github.com/archevel/ghoul/expressions"
)

// Custom log levels
const (
	LevelTrace = slog.Level(-8) // More verbose than DEBUG
)

// Custom handler that recognizes TRACE level
type customHandler struct {
	*slog.TextHandler
}

func (h customHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.TextHandler.Handle(ctx, r)
}

// Custom level replacer
func levelReplacer(groups []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		if level, ok := attr.Value.Any().(slog.Level); ok {
			if level == LevelTrace {
				return slog.Attr{Key: slog.LevelKey, Value: slog.StringValue("TRACE")}
			}
		}
	}
	return attr
}

var (
	NoLogger       = NewWithLevel(slog.LevelError + 1) // Above error = disabled
	StandardLogger = NewWithLevel(slog.LevelWarn)
	VerboseLogger  = NewWithLevel(LevelTrace) // Most verbose
)

type Logger interface {
	Trace(string, ...e.Expr)
	Debug(string, ...e.Expr)
	Warn(string, ...e.Expr)
}

type logger struct {
	slog *slog.Logger
}

// NewWithLevel creates a logger with specified minimum level
func NewWithLevel(level slog.Level) Logger {
	return NewWithWriter(os.Stderr, level)
}

// NewWithWriter creates a logger with custom writer and level
func NewWithWriter(writer io.Writer, level slog.Level) Logger {
	if writer == nil {
		// Return disabled logger for nil writer
		return &logger{nil}
	}
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: levelReplacer,
	}
	textHandler := slog.NewTextHandler(writer, opts)
	handler := customHandler{textHandler}
	return &logger{slog.New(handler)}
}

// New creates a logger for backward compatibility
// debug != nil enables debug level, warn != nil enables warn level
func New(debug io.Writer, warn io.Writer) Logger {
	if debug != nil {
		return NewWithWriter(debug, slog.LevelDebug)
	} else if warn != nil {
		return NewWithWriter(warn, slog.LevelWarn)
	} else {
		return &logger{nil} // Disabled
	}
}

func (l logger) Trace(msg string, args ...e.Expr) {
	if l.slog == nil {
		return // Disabled logger
	}
	if len(args) > 0 {
		// Convert args to strings for printf-style formatting (backward compatibility)
		strArgs := make([]any, len(args))
		for i, expr := range args {
			if expr == nil {
				strArgs[i] = "<nil>"
			} else {
				strArgs[i] = expr.Repr()
			}
		}
		l.slog.Log(nil, LevelTrace, msg, "args", strArgs)
	} else {
		l.slog.Log(nil, LevelTrace, msg)
	}
}

func (l logger) Debug(msg string, args ...e.Expr) {
	if l.slog == nil {
		return // Disabled logger
	}
	if len(args) > 0 {
		// Convert args to strings for printf-style formatting (backward compatibility)
		strArgs := make([]any, len(args))
		for i, expr := range args {
			if expr == nil {
				strArgs[i] = "<nil>"
			} else {
				strArgs[i] = expr.Repr()
			}
		}
		l.slog.Debug(msg, "args", strArgs)
	} else {
		l.slog.Debug(msg)
	}
}

func (l logger) Warn(msg string, args ...e.Expr) {
	if l.slog == nil {
		return // Disabled logger
	}
	if len(args) > 0 {
		// Convert args to strings for printf-style formatting (backward compatibility)
		strArgs := make([]any, len(args))
		for i, expr := range args {
			if expr == nil {
				strArgs[i] = "<nil>"
			} else {
				strArgs[i] = expr.Repr()
			}
		}
		l.slog.Warn(msg, "args", strArgs)
	} else {
		l.slog.Warn(msg)
	}
}

