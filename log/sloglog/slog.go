package sloglog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/voidluo/trojan-go/log"
)

type Logger struct {
	inner *slog.Logger
	level *slog.LevelVar
}

func NewLogger(w io.Writer) *Logger {
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	return &Logger{
		inner: slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})),
		level: level,
	}
}

func (l *Logger) SetLogLevel(level log.LogLevel) {
	switch level {
	case log.AllLevel:
		l.level.Set(slog.LevelDebug - 1)
	case log.InfoLevel:
		l.level.Set(slog.LevelInfo)
	case log.WarnLevel:
		l.level.Set(slog.LevelWarn)
	case log.ErrorLevel:
		l.level.Set(slog.LevelError)
	case log.FatalLevel:
		l.level.Set(slog.LevelError + 4)
	case log.OffLevel:
		l.level.Set(slog.LevelError + 10)
	}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.inner = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: l.level}))
}

func (l *Logger) log(level slog.Level, msg string, v ...any) {
	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	l.inner.Log(context.Background(), level, msg)
}

func (l *Logger) Trace(v ...any)                 { l.log(slog.LevelDebug-1, fmt.Sprint(v...)) }
func (l *Logger) Tracef(format string, v ...any) { l.log(slog.LevelDebug-1, format, v...) }
func (l *Logger) Debug(v ...any)                 { l.log(slog.LevelDebug, fmt.Sprint(v...)) }
func (l *Logger) Debugf(format string, v ...any) { l.log(slog.LevelDebug, format, v...) }
func (l *Logger) Info(v ...any)                  { l.log(slog.LevelInfo, fmt.Sprint(v...)) }
func (l *Logger) Infof(format string, v ...any)  { l.log(slog.LevelInfo, format, v...) }
func (l *Logger) Warn(v ...any)                  { l.log(slog.LevelWarn, fmt.Sprint(v...)) }
func (l *Logger) Warnf(format string, v ...any)  { l.log(slog.LevelWarn, format, v...) }
func (l *Logger) Error(v ...any)                 { l.log(slog.LevelError, fmt.Sprint(v...)) }
func (l *Logger) Errorf(format string, v ...any) { l.log(slog.LevelError, format, v...) }

func (l *Logger) Fatal(v ...any) {
	l.log(slog.LevelError+4, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.log(slog.LevelError+4, format, v...)
	os.Exit(1)
}
