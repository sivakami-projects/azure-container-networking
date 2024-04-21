package logger

import (
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initZapLogger(loggingLevel zapcore.Level) (*zap.Logger, error) {
	platformCore, err := GetPlatformCores(loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Platform cores")
	}
	// Create a new logger with the platform core.
	return zap.New(platformCore, zap.AddCaller()).With(zap.Int("pid", os.Getpid())), nil
}
