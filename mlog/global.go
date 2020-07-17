package mlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *Logger

type LogFunc func(string, ...Field)

var Debug LogFunc = defaultDebugLog
var Info LogFunc = defaultInfoLog
var Warn LogFunc = defaultWarnLog
var Error LogFunc = defaultErrorLog
var Critical LogFunc = defaultCriticalLog

func InitGlobalLogger(logger *Logger) {
	glob := *logger
	glob.zap = glob.zap.WithOptions(zap.AddCallerSkip(1))
	globalLogger = &glob
	Debug = globalLogger.Debug
	Info = globalLogger.Info
	Warn = globalLogger.Warn
	Error = globalLogger.Error
	Critical = globalLogger.Critical
}

func RedirectStdLog(logger *Logger) {
	zap.RedirectStdLogAt(logger.zap.With(zap.String("source", "stdlog")).WithOptions(zap.AddCallerSkip(-2)), zapcore.ErrorLevel)
}
