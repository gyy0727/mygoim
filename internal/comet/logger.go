package comet

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	//*自定义配置
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel), //*日志级别
		Development: true,                                     //*开发模式
		Encoding:    "console",                                //*日志格式（json 或 console）
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder, //*带颜色的日志级别
			EncodeTime:     zapcore.ISO8601TimeEncoder,       //*时间格式
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder, //*调用者信息
		},
		OutputPaths:      []string{"stdout", "app.log"}, //*输出位置
		ErrorOutputPaths: []string{"stderr"},            //*错误输出位置
	}

	//*创建 Logger
	logger, _ := config.Build()
	defer logger.Sync()
}
