package zapetw

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getETWencoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getETWCore(eventName string, loggingLevel zapcore.Level) (zapcore.Core, error) {
	etwSyncer, err := NewETWWriteSyncer(eventName, loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize ETW logger core")
	}
	etwcore := zapcore.NewCore(getETWencoder(), zapcore.AddSync(etwSyncer), loggingLevel)
	return etwcore, nil
}

func InitETWLogger(eventName string, loggingLevel zapcore.Level) (*zap.Logger, error) {
	etwcore, err := getETWCore(eventName, loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize ETW logger")
	}
	return zap.New(etwcore, zap.AddCaller()), nil
}

func AttachETWLogger(baseLogger *zap.Logger, eventName string, loggingLevel zapcore.Level) (*zap.Logger, error) {
	etwcore, err := getETWCore(eventName, loggingLevel)
	if err != nil {
		return baseLogger, errors.Wrap(err, "Failed to attach ETW logger")
	}
	teecore := zapcore.NewTee(baseLogger.Core(), etwcore)
	return zap.New(teecore, zap.AddCaller()), nil
}