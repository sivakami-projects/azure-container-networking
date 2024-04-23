package logger

import (
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initZapLogger(loggingLevel zapcore.Level, encoder zapcore.Encoder) (*zap.Logger, error) {
	platformCore, err := GetPlatformCores(loggingLevel, encoder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Platform cores")
	}
	// Create a new logger with the platform core.
	return zap.New(platformCore, zap.AddCaller()).With(zap.Int("pid", os.Getpid())), nil
}

func getJsonEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}
