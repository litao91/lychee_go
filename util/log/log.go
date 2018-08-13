package log

import (
	"fmt"

	"github.com/astaxie/beego/logs"
)

// Log the logger
var Log *logs.BeeLogger

func init() {
	Log = logs.NewLogger(200)
	Log.EnableFuncCallDepth(true)
	Log.SetLogFuncCallDepth(Log.GetLogFuncCallDepth() + 1)
}

// InitLog initialize the logger
func InitLog(logWay string, logFile string, logLevel string, maxdays int64) {
	SetLogFile(logWay, logFile, maxdays)
	SetLogLevel(logLevel)
}

// SetLogFile logWay: file or console
func SetLogFile(logWay string, logFile string, maxdays int64) {
	if logWay == "console" {
		Log.SetLogger("console", "")
	} else {
		params := fmt.Sprintf(`{"filename": "%s", "maxdays": %d}`, logFile, maxdays)
		Log.SetLogger("file", params)
	}
}

// SetLogLevel value: error, warning, info, debug, trace
func SetLogLevel(logLevel string) {
	level := 4 // warning
	switch logLevel {
	case "error":
		level = 3
	case "warn":
		level = 4
	case "info":
		level = 6
	case "debug":
		level = 7
	case "trace":
		level = 8
	default:
		level = 4
	}
	Log.SetLevel(level)
}

// Error wrap log
func Error(format string, v ...interface{}) {
	Log.Error(format, v...)
}

// Warn Wrap log
func Warn(format string, v ...interface{}) {
	Log.Warn(format, v...)
}

// Info Wrap log
func Info(format string, v ...interface{}) {
	Log.Info(format, v...)
}

// Debug Wrap log
func Debug(format string, v ...interface{}) {
	Log.Debug(format, v...)
}

// Trace Wrap log
func Trace(format string, v ...interface{}) {
	Log.Trace(format, v...)
}

// Logger the loger interface
type Logger interface {
	AddLogPrefix(string)
	GetAllPrefix() []string
	ClearLogPrefix()
	Error(string, ...interface{})
	Warn(string, ...interface{})
	Info(string, ...interface{})
	Debug(string, ...interface{})
	Trace(string, ...interface{})
}

// PrefixLogger add prefix to the logger
type PrefixLogger struct {
	prefix    string
	allPrefix []string
}

// NewPrefixLogger Create new PrefixLogger
func NewPrefixLogger(prefix string) *PrefixLogger {
	logger := &PrefixLogger{
		allPrefix: make([]string, 0),
	}
	logger.AddLogPrefix(prefix)
	return logger
}

// AddLogPrefix add prefix to a logger
func (pl *PrefixLogger) AddLogPrefix(prefix string) {
	if len(prefix) == 0 {
		return
	}

	pl.prefix += "[" + prefix + "] "
	pl.allPrefix = append(pl.allPrefix, prefix)
}

// GetAllPrefix get all prefix of a logger
func (pl *PrefixLogger) GetAllPrefix() []string {
	return pl.allPrefix
}

// ClearLogPrefix clear the prefix
func (pl *PrefixLogger) ClearLogPrefix() {
	pl.prefix = ""
	pl.allPrefix = make([]string, 0)
}

// Error log error
func (pl *PrefixLogger) Error(format string, v ...interface{}) {
	Log.Error(pl.prefix+format, v...)
}

// Warn log warning
func (pl *PrefixLogger) Warn(format string, v ...interface{}) {
	Log.Warn(pl.prefix+format, v...)
}

// Info log information
func (pl *PrefixLogger) Info(format string, v ...interface{}) {
	Log.Info(pl.prefix+format, v...)
}

// Debug log debugging info
func (pl *PrefixLogger) Debug(format string, v ...interface{}) {
	Log.Debug(pl.prefix+format, v...)
}

// Trace log trace
func (pl *PrefixLogger) Trace(format string, v ...interface{}) {
	Log.Trace(pl.prefix+format, v...)
}
