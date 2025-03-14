package comet

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	_ "github.com/gyy0727/mygoim/internal/comet/conf"
)

var logger *zap.Logger

func init() {
	//*Zap 基础配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	//*配置 lumberjack 日志切割
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "/cloudide/workspace/mygoim/log/app.log", //*日志文件名
		MaxSize:    10,       //*单个文件最大大小 (MB)
		MaxBackups: 30,         //*保留旧文件的最大数量
		MaxAge:     70,         //*保留旧文件的最大天数 (days)
		Compress:   true,      //*是否压缩旧文件
	}

	//*创建 Zap 的 WriteSyncer（同时输出到控制台和文件）
	consoleSyncer := zapcore.AddSync(os.Stdout)
	fileSyncer := zapcore.AddSync(lumberjackLogger)
	combinedSyncer := zapcore.NewMultiWriteSyncer(consoleSyncer, fileSyncer)

	//*构建 Zap Core
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), //*编码器（保持与之前一致）
		combinedSyncer,                           //*输出位置
		zap.DebugLevel,                           //*日志级别
	)

	//*创建 Logger
	logger = zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger) //*替换全局 Logger
}
