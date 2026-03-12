package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 构建 Zap Logger。
func New(level string) (*zap.Logger, error) {
	parsed := zapcore.InfoLevel
	if err := parsed.Set(level); err != nil {
		parsed = zapcore.InfoLevel
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(parsed)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"
	return cfg.Build()
}
