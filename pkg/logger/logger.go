package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

// Define log levels
var LevelMap = map[string]string{
	"error": "[ERR]",
	"debug": "[DBG]",
	"warn":  "[WRN]",
	"info":  "[INF]",
}

const (
	ResetColor = "\033[0m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Gray       = "\033[90m"
	Red        = "\033[31m"
)

type CustomLogger struct {
	levels              []string
	logBackTraceEnabled bool
}

type Options struct {
	LogBackTraceEnabled bool
}

func NewDefaultLogger() *CustomLogger {
	return &CustomLogger{
		levels:              []string{"error", "debug", "warn", "info"},
		logBackTraceEnabled: true,
	}
}

// TODO
func NewLoggerWithOptions(levels []string, options *Options) *CustomLogger {
	if len(levels) == 1 && levels[0] == "all" {
		levels = []string{"error", "debug", "warn", "info"}
	}

	return &CustomLogger{
		levels:              levels,
		logBackTraceEnabled: options.LogBackTraceEnabled,
	}
}

func (cl *CustomLogger) Info(msg string) {
	if cl.shouldLog("info") {
		logMessage("info", Green, msg)
	}
}

func (cl *CustomLogger) Warn(msg string) {
	if cl.shouldLog("warn") {
		logMessage("warn", Yellow, msg)
	}
}

func (cl *CustomLogger) Debug(msg string) {
	if cl.shouldLog("debug") {
		logMessage("debug", Gray, msg)
	}
}

func (cl *CustomLogger) Error(msg string) {
	if cl.shouldLog("error") {
		if cl.logBackTraceEnabled {
			logMessageWithBacktrace("error", Red, msg)
		} else {
			_, file, line, _ := runtime.Caller(1)
			logMessage("error", Red, fmt.Sprintf("Error at line %s:%d : %s", file, line, msg))
		}
		os.Exit(1)
	}
}

func logMessage(level, color, msg string) {
	log.SetFlags(0)
	log.SetPrefix(fmt.Sprintf("%s %s %s: ", time.Now().Format("2006-01-02  15:04:05.000"), LevelMap[level], color))
	log.Println(msg, ResetColor)
}

// logMessageWithBacktrace logs a message with a backtrace
func logMessageWithBacktrace(level, color, msg string) {
	logMessage(level, color, msg)
	stack := make([]byte, 1<<16)
	length := runtime.Stack(stack, false)
	log.Printf("Stack trace:\n%s\n", stack[:length])
}

func (cl *CustomLogger) shouldLog(level string) bool {
	for _, l := range cl.levels {
		if l == level {
			return true
		}
	}
	return false
}
