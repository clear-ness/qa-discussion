package mlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Very verbose messages for debugging specific issues
	LevelDebug = "debug"
	// Default log level, informational
	LevelInfo = "info"
	// Warnings are messages about possible issues
	LevelWarn = "warn"
	// Errors are messages about things we know are problems
	LevelError = "error"
)

type Field = zapcore.Field

var Int64 = zap.Int64
var Int32 = zap.Int32
var Int = zap.Int
var Uint32 = zap.Uint32
var String = zap.String
var Any = zap.Any
var Err = zap.Error
var NamedErr = zap.NamedError
var Bool = zap.Bool
var Duration = zap.Duration

type Logger struct {
	zap          *zap.Logger
	consoleLevel zap.AtomicLevel
	fileLevel    zap.AtomicLevel
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func NewLogger() *Logger {
	cores := []zapcore.Core{}
	combinedCore := zapcore.NewTee(cores...)

	logger := &Logger{
		consoleLevel: zap.NewAtomicLevelAt(getZapLevel("DEBUG")),
		fileLevel:    zap.NewAtomicLevelAt(getZapLevel("INFO")),
	}

	logger.zap = zap.New(combinedCore,
		zap.AddCaller(),
	)
	return logger
}

func (l *Logger) Debug(message string, fields ...Field) {
	l.zap.Debug(message, fields...)
}

func (l *Logger) Info(message string, fields ...Field) {
	l.zap.Info(message, fields...)
}

func (l *Logger) Warn(message string, fields ...Field) {
	l.zap.Warn(message, fields...)
}

func (l *Logger) Error(message string, fields ...Field) {
	l.zap.Error(message, fields...)
}

func (l *Logger) Critical(message string, fields ...Field) {
	l.zap.Error(message, fields...)
}

func (l *Logger) With(fields ...Field) *Logger {
	newlogger := *l
	newlogger.zap = newlogger.zap.With(fields...)
	return &newlogger
}
