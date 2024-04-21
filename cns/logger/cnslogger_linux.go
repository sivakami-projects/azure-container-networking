package logger

import (
	"go.uber.org/zap/zapcore"
)

const (
	// LogPath is the path where log files are stored.
	LogPath = "/var/log/"
)

func GetPlatformCores(_ zapcore.Level) (zapcore.Core, error) {
	return nil, nil
}
