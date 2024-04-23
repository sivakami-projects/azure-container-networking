package logger

import (
	"go.uber.org/zap/zapcore"
)

const (
	// LogPath is the path where log files are stored.
	LogPath = "/var/log/"
)

func GetPlatformCores(zapcore.Level, zapcore.Encoder) (zapcore.Core, error) {
	return nil, nil
}
