package bot

import (
	"os"
	"path"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GetModuleLogger - 提供一个为 Module 使用的 logrus.Entry
// 包含 logrus.Fields
func GetModuleLogger(name string) *zap.Logger {
	return zap.L().With(zap.String("module", name))
}

// WriteLogToPath 将日志转储至文件
// 请务必在 init() 阶段调用此函数
// 否则会出现日志缺失
// 日志存储位置 p
func writeLogToPath(p string) error {
	writer, err := rotatelogs.New(
		path.Join(p, "%Y-%m-%d.log"),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return err
	}

	// 将日志文件写入文件和终端
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(writer),
			zapcore.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		),
	)

	zap.ReplaceGlobals(zap.New(core))

	return nil
}
