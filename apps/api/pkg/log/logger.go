/**
 * @Author: DPY
 * @Description:
 * @File:  log
 * @Version: 1.0.0
 * @Date: 2021/11/2 16:33
 */

package log

/*
- 日志的输出可以配置：1-stdout、2-本地文件、3-syslog、4-es、5-clickhouse
- 日志输出的格式为：
	1- stdout -> console
	2- 本地文件 -> console (可以设置日志滚动)
	3- syslog -> syslog
	4- other -> json
- 可以自定义日志字段
*/

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DebugLevel int8 = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
)

var (
	// Engine 日志结构，默认没有任何输出
	Log    *zap.Logger
	Logger *zap.SugaredLogger

	// Config 保存当前日志配置的全局变量
	Config = &Conf{
		Level:      0,
		Format:     "console",
		StdOut:     true,
		FileOut:    false,
		FileRotate: lumberjack.Logger{},
		ShowCaller: false,
		CallerSkip: 0,
		ShowStack:  false,
	}

	// Encoding 日志输出格式
	Encoding = zapcore.EncoderConfig{
		MessageKey:       "msg",                          // 消息key名称
		LevelKey:         "level",                        // 日志级别key名称
		TimeKey:          "time",                         // 日志时间key名称
		NameKey:          "logger",                       // 日志信息key名称
		CallerKey:        "caller",                       // 调用方key名称
		FunctionKey:      "func",                         // 函数key名称
		StacktraceKey:    "stack",                        // 调用栈key名
		LineEnding:       zapcore.DefaultLineEnding,      //每行分隔符,默认\n
		EncodeLevel:      zapcore.CapitalLevelEncoder,    // level值的封装,配置为序列化为全大写
		EncodeTime:       timeFormatter,                  // 时间格式,配置为[2006-01-02 15:04:05]
		EncodeDuration:   zapcore.SecondsDurationEncoder, // 执行消耗时间格式,配置为浮点秒
		EncodeCaller:     zapcore.ShortCallerEncoder,     // 调用者格式,配置为包/文件:行号
		EncodeName:       zapcore.FullNameEncoder,        // 日志信息名处理,默认无处理
		ConsoleSeparator: " ",
	}
)

func init() {
	Log = Config.New()
	Logger = Log.Sugar()
}

// Conf 日志配置结构，不对外开放，用户需要通过 LogConfig 变量操作日志配置
type Conf struct {
	Format     string            `json:"format,omitempty" yaml:"format,omitempty"`         // 日志格式，取值json,console 默认json
	Level      int8              `json:"level,omitempty" yaml:"level,omitempty"`           // 日志级别
	StdOut     bool              `json:"stdout,omitempty" yaml:"stdout,omitempty"`         // 是否输出到StdOut
	FileOut    bool              `json:"fileout,omitempty" yaml:"fileout,omitempty"`       // 是否输出到本地文件
	FileRotate lumberjack.Logger `json:"filerotate,omitempty" yaml:"filerotate,omitempty"` // 本地文件的滚动配置
	//Syslog     bool              `json:"syslog"`      // 是否输出到syslog
	// 以下参数不会在WEB日志中体现
	ShowCaller bool `json:"showcaller,omitempty" yaml:"showcaller,omitempty"` // 是否打印调用方法
	CallerSkip int  `json:"callerskip,omitempty" yaml:"callerskip,omitempty"` // 调用方法的越过层数，通常用于跳过包装的方法，返回真实的调用方(例如gin中通常加1)
	ShowStack  bool `json:"showstack,omitempty" yaml:"showstack,omitempty"`   // 是否打印调用栈，一般不建议开启
}

// Reset 将现有日志对象按option参数修改
func (s *Conf) Reset() {
	Log = s.New()
	Logger = Log.Sugar()
}

// New 按 logConfig 中的参数计算日志选项
func (s *Conf) New() *zap.Logger {
	var (
		encoder = zapcore.NewJSONEncoder(Encoding)
		outList []zapcore.WriteSyncer
		options []zap.Option
	)
	// 构建zapcore的Encoder
	if s.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(Encoding)
	}
	if s.StdOut {
		outList = append(outList, os.Stdout)
	}
	if s.FileOut {
		outList = append(outList, zapcore.AddSync(&s.FileRotate))
	}
	writer := zapcore.NewMultiWriteSyncer(outList...)
	core := zapcore.NewCore(encoder, writer, zapcore.Level(s.Level))
	// 构建option
	if s.ShowCaller {
		options = append(options, zap.WithCaller(true))
	}
	if s.ShowStack {
		options = append(options, zap.AddStacktrace(zapcore.Level(s.Level)))
	}
	if s.CallerSkip > 0 {
		options = append(options, zap.AddCallerSkip(s.CallerSkip))
	}
	return zap.New(core, options...)
}

// timeFormatter 自定义日志时间格式
func timeFormatter(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}
