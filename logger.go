package rmqrpc

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/fatih/color"
)

type LogEntry struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type CustomLog struct {
	logger    *slog.Logger
	namespace string
}

func NewCustomLog(namespace string) *CustomLog {
	opts := slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	prettyHandler := NewPrettyHandler(os.Stdout, PrettyHandlerOptions{opts})
	logger := slog.New(prettyHandler)
	return &CustomLog{
		logger:    logger,
		namespace: namespace,
	}
}

func (cl *CustomLog) Info(msg string, data any) {
	ctx := context.Background()
	cl.logger.Log(ctx, slog.LevelInfo, cl.namespace, slog.String("message", msg), slog.Any("data", data))
}

func (cl *CustomLog) Debug(msg string, data any) {
	ctx := context.Background()
	cl.logger.Log(ctx, slog.LevelDebug, cl.namespace, slog.String("message", msg), slog.Any("data", data))
}

func (cl *CustomLog) Error(msg string, data any) {
	ctx := context.Background()
	cl.logger.Log(ctx, slog.LevelError, cl.namespace, slog.String("message", msg), slog.Any("data", data))
}

type PrettyHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type PrettyHandler struct {
	slog.Handler
	l *log.Logger
}

func NewPrettyHandler(
	out io.Writer,
	opts PrettyHandlerOptions,
) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

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

	var entry LogEntry

	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "message" {
			entry.Message = a.Value.String()
		}
		if a.Key == "data" {
			entry.Data = a.Value.Any()
		}
		return true
	})

	b, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	timeStr := r.Time.Format(time.RFC3339)
	msg := color.CyanString(r.Message)

	h.l.Println(timeStr, level, msg, color.WhiteString(string(b)))

	return nil
}
