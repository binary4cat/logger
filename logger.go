package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// A Level is a logging priority. Higher levels are more important.
type Level int8

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel
)

type Options struct {
	NotStdout  bool // 不写标准输出吗？默认输出到标准输出，不管有无写文件
	Level      Level
	Filename   string // 文件为空时，只输出到标准输出
	MaxSize    int    // megabytes
	MaxBackups int    // MaxBackups is the maximum number of old log files to retain
	MaxAge     int    // MaxAge is the maximum number of days to retain old log files based on the timestamp encoded in their filename
	Compress   bool
}

// 正在写入的日志信息
type LogInfo struct {
	Level      Level     // 日志级别
	Time       time.Time // 写日志的时间
	LoggerName string    // 写日志组件的名称
	Message    string    // 日志内容
	Stack      string    // 日志堆栈信息
	File       string    // 写日志操作的代码文件
	Line       int       // 写日志操作的代码行数
}

var (
	// 日志组件
	logger *zap.SugaredLogger
	// 写日志入文件组件
	fileWirtor *zapcore.WriteSyncer
	// 写日志到标准输出
	stdWirtor *zapcore.WriteSyncer
)

func init() {
	// 初始化一个默认只输出标准输出的日志对象
	InitLogger(&Options{
		NotStdout:  false,
		Level:      DebugLevel,
		Filename:   "",
		MaxSize:    0,
		MaxBackups: 0,
		MaxAge:     0,
		Compress:   false,
	})
}

func InitLogger(opt *Options, hooks ...func(LogInfo) error) {
	var treeCore zapcore.Core
	fileWirtor = getLogWriter(opt)
	encoder := getEncoder()

	if opt.NotStdout && opt.Filename != "" {
		treeCore = zapcore.NewCore(encoder, *fileWirtor, zapcore.Level(opt.Level))
	} else if opt.Filename == "" && !opt.NotStdout {
		sw := zapcore.AddSync(os.Stdout)
		stdWirtor = &sw
		treeCore = zapcore.NewCore(encoder, sw, zapcore.Level(opt.Level))
	} else {
		sw := zapcore.AddSync(os.Stdout)
		stdWirtor = &sw
		stdoutCore := zapcore.NewCore(encoder, sw, zapcore.Level(opt.Level))
		fileCore := zapcore.NewCore(encoder, *fileWirtor, zapcore.Level(opt.Level))
		treeCore = zapcore.NewTee(stdoutCore, fileCore)
	}

	// AddCallerSkip，因为封装调用了zap的logger的方法，所以runtime.Caller层级必须修正，否则无法获取真实的日志调用位置
	zl := zap.New(treeCore, zap.AddCaller()).WithOptions(zap.AddCallerSkip(1), zap.Hooks(hooksHandler(hooks...)...))
	logger = zl.Sugar()
}

// 返回一个日志writer，可自定义处理
func GetLogWirter() (io.Writer, error) {
	if logger == nil {
		return nil, errors.New("请初始化日志组件后调用")
	}
	return *fileWirtor, nil
}

// 对日志的其他处理，例如异步推送到Kafka或者ES数据库
func hooksHandler(hooks ...func(LogInfo) error) []func(zapcore.Entry) error {
	var resHooks []func(zapcore.Entry) error
	for _, hook := range hooks {
		resHook := func(entity zapcore.Entry) error {
			return hook(LogInfo{
				Level:      Level(entity.Level),
				Time:       entity.Time,
				LoggerName: entity.LoggerName,
				Message:    entity.Message,
				Stack:      entity.Stack,
				File:       entity.Caller.File,
				Line:       entity.Caller.Line,
			})
		}
		resHooks = append(resHooks, resHook)
	}
	return resHooks
}

// 获取除文件名外的默认配置：日志文件最大100M，备份最多10个，最多保存30天的数据，不压缩
func GetDefault(filename string) *Options {
	return &Options{
		Filename:   filename,
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   false,
	}
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(opt *Options) *zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   opt.Filename,
		MaxSize:    opt.MaxSize,
		MaxBackups: opt.MaxBackups,
		MaxAge:     opt.MaxAge,
		Compress:   opt.Compress,
	}
	ws := zapcore.AddSync(lumberJackLogger)
	return &ws
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	logger.Error(args...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	logger.DPanic(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	logger.Panic(args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	logger.DPanicf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	logger.Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//  s.With(keysAndValues).Debug(msg)
func Debugw(msg string, keysAndValues ...interface{}) {
	logger.Debugw(msg, keysAndValues...)
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Infow(msg string, keysAndValues ...interface{}) {
	logger.Infow(msg, keysAndValues...)
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Warnw(msg string, keysAndValues ...interface{}) {
	logger.Warnw(msg, keysAndValues...)
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Errorw(msg string, keysAndValues ...interface{}) {
	logger.Errorw(msg, keysAndValues...)
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func DPanicw(msg string, keysAndValues ...interface{}) {
	logger.DPanicw(msg, keysAndValues...)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func Panicw(msg string, keysAndValues ...interface{}) {
	logger.Panicw(msg, keysAndValues...)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func Fatalw(msg string, keysAndValues ...interface{}) {
	logger.Fatalw(msg, keysAndValues...)
}

// Output pure content without any additional information
func Pure(args ...interface{}) {
	if fileWirtor != nil {
		fw := *fileWirtor
		fw.Write([]byte(fmt.Sprintln(args...)))
	}
	if stdWirtor != nil {
		sw := *stdWirtor
		sw.Write([]byte(fmt.Sprintln(args...)))
	}
}

// Output purely formatted content without any additional information
func Puref(msg string, args ...interface{}) {
	if fileWirtor != nil {
		fw := *fileWirtor
		fw.Write([]byte(fmt.Sprintf(msg, args...) + "\n"))
	}
	if stdWirtor != nil {
		sw := *stdWirtor
		sw.Write([]byte(fmt.Sprintf(msg, args...) + "\n"))
	}
}
